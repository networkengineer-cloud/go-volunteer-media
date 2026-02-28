package main

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/networkengineer-cloud/go-volunteer-media/frontend"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/handlers"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
)

func main() {
	// Initialize logging from environment variables
	logging.InitFromEnv()
	logger := logging.GetDefaultLogger()

	// Register custom validators and use JSON tag names in validation error messages
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		_ = v.RegisterValidation("usernamechars", func(fl validator.FieldLevel) bool {
			username := fl.Field().String()
			// Allow letters, numbers, underscore, dot, and dash
			matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, username)
			return matched
		})
		// Use the json tag name so validation errors report field names as they appear in the API
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using system environment variables")
	}

	// Initialize database
	db, err := database.Initialize()
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}

	// Get underlying SQL database for proper connection management
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get database instance", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Error("Error closing database", err)
		}
		logger.Info("Database connection closed")
	}()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations", err)
	}

	// Initialize storage provider
	storageConfig := storage.LoadConfig()
	storageProvider, err := storage.NewProvider(storageConfig, db)
	if err != nil {
		logger.Fatal("Failed to initialize storage provider", err)
	}
	logger.WithFields(map[string]interface{}{
		"provider": storageConfig.Provider,
	}).Info("Storage provider initialized")

	// Initialize email service
	emailService := email.NewService(db)
	if emailService.IsConfigured() {
		logger.Info("Email service configured and ready")
	} else {
		logger.Info("Email service not configured - password reset and email notifications will be disabled")
	}

	// Initialize GroupMe service
	groupMeService := groupme.NewService()
	logger.Info("GroupMe service initialized and ready")

	// Load embedded frontend assets at startup
	distFS, err := fs.Sub(frontend.DistFS, "dist")
	if err != nil {
		logger.Fatal("Failed to create dist sub-filesystem", err)
	}
	indexBytes, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		logger.Fatal("Failed to read embedded index.html", err)
	}
	assetsSubFS, err := fs.Sub(frontend.DistFS, "dist/assets")
	if err != nil {
		logger.Fatal("Failed to create assets sub-filesystem", err)
	}

	// Set up Gin router
	// Disable Gin's default logger since we're using our own
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Security headers middleware (add before CORS)
	router.Use(middleware.SecurityHeaders())

	// Request ID middleware for tracing
	router.Use(middleware.RequestID())

	// Structured logging middleware
	router.Use(middleware.LoggingMiddleware())

	// Max request body size middleware (10MB default, prevents DOS attacks)
	router.Use(middleware.MaxRequestBodySize(10 * 1024 * 1024))

	// CORS middleware
	router.Use(middleware.CORS())

	// Health check endpoints (public, no auth required)
	router.GET("/health", handlers.HealthCheck())
	router.GET("/healthz", handlers.HealthCheck())
	router.GET("/ready", handlers.ReadinessCheck(db))

	// Serve uploaded images from database (public, cached)
	// Legacy: also serve from filesystem for backwards compatibility
	router.Static("/uploads", "./public/uploads")
	router.StaticFile("/default-hero.svg", "./public/default-hero.svg")

	// Serve security.txt for responsible vulnerability disclosure
	router.StaticFile("/.well-known/security.txt", "./public/.well-known/security.txt")

	// API routes
	api := router.Group("/api")

	// Serve images from database (public endpoint, no auth required)
	api.GET("/images/:uuid", handlers.ServeImage(db, storageProvider))

	// Public routes (with rate limiting for auth endpoints)
	authRateLimit := 5
	if v := os.Getenv("AUTH_RATE_LIMIT_PER_MINUTE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			authRateLimit = parsed
		}
	}
	authLimiter := middleware.RateLimit(authRateLimit, 1*time.Minute)
	api.POST("/login", authLimiter, handlers.Login(db))
	// Registration disabled - invite-only system. Admins can create users via /api/admin/users
	// api.POST("/register", authLimiter, handlers.Register(db))
	api.POST("/request-password-reset", authLimiter, handlers.RequestPasswordReset(db, emailService))
	api.POST("/reset-password", authLimiter, handlers.ResetPassword(db))
	api.POST("/setup-password", authLimiter, handlers.SetupPassword(db)) // New user password setup (invite flow)

	// Site settings (public read)
	api.GET("/settings", handlers.GetSiteSettings(db))

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		// Environment info (authenticated users can check environment)
		protected.GET("/environment", handlers.GetEnvironment())

		// User routes
		protected.GET("/me", handlers.GetCurrentUser(db))
		protected.GET("/users/:id/profile", handlers.GetUserProfile(db))
		protected.PUT("/me/profile", handlers.UpdateCurrentUserProfile(db))
		protected.GET("/email-preferences", handlers.GetEmailPreferences(db))
		protected.PUT("/email-preferences", handlers.UpdateEmailPreferences(db))
		protected.PUT("/default-group", handlers.SetDefaultGroup(db))
		protected.GET("/default-group", handlers.GetDefaultGroup(db))

		// Announcement routes (all authenticated users can view)
		protected.GET("/announcements", handlers.GetAnnouncements(db))

		// Group routes
		protected.GET("/groups", handlers.GetGroups(db))

		// Image upload (authenticated users only) - stores in database
		protected.POST("/animals/upload-image", handlers.UploadAnimalImageSimple(db, storageProvider))

		// Document serving route (PROTECTED): requires authentication and group membership
		protected.GET("/documents/:uuid", handlers.ServeAnimalProtocolDocument(db, storageProvider))

		// Statistics routes (accessible by authenticated users, filtered by permissions)
		protected.GET("/statistics/comment-tags", handlers.GetCommentTagStatistics(db))

		// Group admin management (accessible by site admins and group admins)
		// Authorization is checked within the handlers
		protected.POST("/groups/:id/admins/:userId", handlers.PromoteGroupAdmin(db))
		protected.DELETE("/groups/:id/admins/:userId", handlers.DemoteGroupAdmin(db))

		// User management (accessible by site admins and group admins for users in their groups)
		// Authorization is checked within the handlers
		protected.POST("/users", handlers.GroupAdminCreateUser(db, emailService))
		protected.PUT("/users/:userId", handlers.GroupAdminUpdateUser(db)) // Handles both site admins and group admins
		protected.POST("/users/:userId/reset-password", handlers.AdminResetUserPassword(db))
		protected.POST("/users/:userId/resend-invitation", handlers.ResendInvitation(db, emailService))
		protected.POST("/users/:userId/unlock", handlers.UnlockUserAccount(db)) // Site admins and group admins

		// Admin only routes
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", handlers.GetAllUsers(db))
			admin.POST("/users", handlers.AdminCreateUser(db, emailService))
			admin.PUT("/users/:userId", handlers.AdminUpdateUser(db)) // Admin-specific endpoint (preferred path for admins)
			admin.DELETE("/users/:userId", handlers.AdminDeleteUser(db))
			admin.GET("/users/deleted", handlers.GetDeletedUsers(db))
			admin.POST("/users/:userId/restore", handlers.RestoreUser(db))
			admin.POST("/users/:userId/promote", handlers.PromoteUser(db))
			admin.POST("/users/:userId/demote", handlers.DemoteUser(db))

			// Group management (admin only)
			admin.POST("/groups", handlers.CreateGroup(db))
			admin.PUT("/groups/:id", handlers.UpdateGroup(db))
			admin.DELETE("/groups/:id", handlers.DeleteGroup(db))
			admin.POST("/groups/upload-image", handlers.UploadGroupImage(storageProvider))
			admin.POST("/users/:userId/groups/:groupId", handlers.AddUserToGroup(db))
			admin.DELETE("/users/:userId/groups/:groupId", handlers.RemoveUserFromGroup(db))

			// Announcement routes (admin only)
			admin.POST("/announcements", handlers.CreateAnnouncement(db, emailService, groupMeService))
			admin.DELETE("/announcements/:id", handlers.DeleteAnnouncement(db))

			// Site settings management (admin only)
			admin.PUT("/settings/:key", handlers.UpdateSiteSetting(db))
			admin.POST("/settings/upload-hero-image", handlers.UploadHeroImage(storageProvider))

			// Bulk animal management (admin only)
			admin.GET("/animals", handlers.GetAllAnimals(db))
			admin.POST("/animals/bulk-update", handlers.BulkUpdateAnimals(db))
			admin.POST("/animals/import-csv", handlers.ImportAnimalsCSV(db))
			admin.POST("/animals/export-csv", handlers.ExportAnimalsCSV(db))
			admin.GET("/animals/export-comments-csv", handlers.ExportAnimalCommentsCSV(db))
			admin.PUT("/animals/:animalId", handlers.UpdateAnimalAdmin(db))

			// Animal image management (admin only)
			admin.PUT("/animals/:animalId/images/:imageId/set-profile", handlers.SetAnimalProfilePicture(db))

			// Database seeding (admin only, dangerous operation)
			admin.POST("/seed-database", handlers.SeedDatabase(db))

			// Statistics routes (admin only)
			admin.GET("/statistics/groups", handlers.GetGroupStatistics(db))
			admin.GET("/statistics/users", handlers.GetUserStatistics(db))

			// Admin dashboard
			admin.GET("/dashboard/stats", handlers.GetAdminDashboardStats(db))

			// Admin content moderation - view deleted content
			admin.GET("/groups/:id/deleted-comments", handlers.GetDeletedComments(db))
			admin.GET("/groups/:id/deleted-images", handlers.GetDeletedImages(db))
		}

		// Group-specific routes
		group := protected.Group("/groups/:id")
		{
			group.GET("", handlers.GetGroup(db))
			group.GET("/membership", handlers.GetGroupMembership(db))

			// Animal routes - viewing accessible to all group members
			group.GET("/animals", handlers.GetAnimals(db))
			group.GET("/animals/:animalId", handlers.GetAnimal(db))
			group.GET("/animals/check-duplicates", handlers.CheckDuplicateNames(db))

			// Animal images - all group members can view, upload, and set profile pictures
			group.GET("/animals/:animalId/images", handlers.GetAnimalImages(db))
			group.POST("/animals/:animalId/images", handlers.UploadAnimalImageToGallery(db, storageProvider))
			group.DELETE("/animals/:animalId/images/:imageId", handlers.DeleteAnimalImage(db, storageProvider))
			// Profile picture selection - available to all group members to help curate animal photos
			group.PUT("/animals/:animalId/images/:imageId/set-profile", handlers.SetAnimalProfilePictureGroupScoped(db))

			// Animal comments - all group members can view, add, and edit own comments
			group.GET("/animals/:animalId/comments", handlers.GetAnimalComments(db))
			group.POST("/animals/:animalId/comments", handlers.CreateAnimalComment(db))
			group.PUT("/animals/:animalId/comments/:commentId", handlers.UpdateAnimalComment(db))
			group.DELETE("/animals/:animalId/comments/:commentId", handlers.DeleteAnimalComment(db))
			group.GET("/animals/:animalId/comments/:commentId/history", handlers.GetCommentHistory(db))

			// Latest comments across the group
			group.GET("/latest-comments", handlers.GetGroupLatestComments(db))

			// Activity feed - unified view of announcements and comments
			group.GET("/activity-feed", handlers.GetGroupActivityFeed(db))

			// Updates routes
			group.GET("/updates", handlers.GetUpdates(db))
			group.POST("/updates", handlers.CreateUpdate(db, emailService, groupMeService))

			// Protocol routes - all group members can view
			group.GET("/protocols", handlers.GetProtocols(db))
			group.GET("/protocols/:protocolId", handlers.GetProtocol(db))

			// Group-specific tag routes - viewing accessible to all group members
			// Tag management (create/update/delete) requires group admin or site admin
			group.GET("/animal-tags", handlers.GetAnimalTags(db))
			group.POST("/animal-tags", handlers.CreateAnimalTag(db))
			group.PUT("/animal-tags/:tagId", handlers.UpdateAnimalTag(db))
			group.DELETE("/animal-tags/:tagId", handlers.DeleteAnimalTag(db))

			group.GET("/comment-tags", handlers.GetCommentTags(db))
			group.POST("/comment-tags", handlers.CreateCommentTag(db))
			group.DELETE("/comment-tags/:tagId", handlers.DeleteCommentTag(db))

			// Group settings - group admin or site admin can update
			group.PUT("/settings", handlers.UpdateGroupSettings(db))

			// Member management - group admin or site admin (checks access within handlers)
			group.GET("/members", handlers.GetGroupMembers(db))
			group.POST("/members/:userId", handlers.AddMemberToGroup(db))
			group.DELETE("/members/:userId", handlers.RemoveMemberFromGroup(db))
			group.POST("/members/:userId/promote", handlers.PromoteMemberToGroupAdmin(db))
			group.POST("/members/:userId/demote", handlers.DemoteMemberFromGroupAdmin(db))

			// Content moderation - group admin or site admin can view deleted content
			group.GET("/deleted-comments", handlers.GetDeletedComments(db))
			group.GET("/deleted-images", handlers.GetDeletedImages(db))

			// Group announcements - group admin or site admin can create announcements for their group
			group.POST("/announcements", handlers.CreateGroupAnnouncement(db, emailService, groupMeService))
		}

		// Group admin or site admin animal management routes
		// These routes check for site admin OR group admin access within the handlers
		groupAdminAnimals := protected.Group("/groups/:id/animals")
		{
			groupAdminAnimals.POST("", handlers.CreateAnimal(db))
			groupAdminAnimals.PUT("/:animalId", handlers.UpdateAnimal(db))
			groupAdminAnimals.DELETE("/:animalId", handlers.DeleteAnimal(db))
			// Tag assignment for animals
			groupAdminAnimals.POST("/:animalId/tags", handlers.AssignTagsToAnimal(db))
			// Protocol document management
			groupAdminAnimals.POST("/:animalId/protocol-document", handlers.UploadAnimalProtocolDocument(db, storageProvider))
			groupAdminAnimals.DELETE("/:animalId/protocol-document", handlers.DeleteAnimalProtocolDocument(db, storageProvider))
		}

		// Group admin or site admin protocol management routes
		// These routes check for site admin OR group admin access within the handlers
		groupAdminProtocols := protected.Group("/groups/:id/protocols")
		{
			groupAdminProtocols.POST("/upload-image", handlers.UploadProtocolImage(db, storageProvider))
			groupAdminProtocols.POST("", handlers.CreateProtocol(db))
			groupAdminProtocols.PUT("/:protocolId", handlers.UpdateProtocol(db))
			groupAdminProtocols.DELETE("/:protocolId", handlers.DeleteProtocol(db))
		}

		// Bulk animal management routes accessible to group admins and site admins
		// Authorization is checked within the handlers
		protected.GET("/bulk-animals", handlers.GetAllAnimals(db))
		protected.POST("/bulk-animals/bulk-update", handlers.BulkUpdateAnimals(db))
	}

	// Serve frontend from embedded filesystem (guarantees correct MIME types)
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
	})
	router.GET("/favicon.svg", func(c *gin.Context) {
		c.FileFromFS("favicon.svg", http.FS(distFS))
	})
	router.GET("/vite.svg", func(c *gin.Context) {
		c.FileFromFS("vite.svg", http.FS(distFS))
	})
	router.StaticFS("/assets", http.FS(assetsSubFS))

	// Serve index.html for all non-API routes (SPA routing)
	router.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithFields(map[string]interface{}{
			"port": port,
			"env":  os.Getenv("ENV"),
		}).Info("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", err)
	}

	logger.Info("Server exited gracefully")
}
