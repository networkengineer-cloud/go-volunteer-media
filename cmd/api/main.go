package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin/binding"

	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/handlers"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
)

func main() {
	// Initialize logging from environment variables
	logging.InitFromEnv()
	logger := logging.GetDefaultLogger()

	// Register custom username validator
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		_ = v.RegisterValidation("usernamechars", func(fl validator.FieldLevel) bool {
			username := fl.Field().String()
			// Allow letters, numbers, underscore, dot, and dash
			matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, username)
			return matched
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

	// Initialize email service
	emailService := email.NewService()
	if emailService.IsConfigured() {
		logger.Info("Email service configured and ready")
	} else {
		logger.Info("Email service not configured - password reset and email notifications will be disabled")
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

	// CORS middleware
	router.Use(middleware.CORS())

	// Health check endpoints (public, no auth required)
	router.GET("/health", handlers.HealthCheck())
	router.GET("/healthz", handlers.HealthCheck())
	router.GET("/ready", handlers.ReadinessCheck(db))

	// Serve uploaded images and default assets
	router.Static("/uploads", "./public/uploads")
	router.StaticFile("/default-hero.svg", "./public/default-hero.svg")

	// API routes
	api := router.Group("/api")

	// Public routes (with rate limiting for auth endpoints)
	authLimiter := middleware.RateLimit(5, 1*time.Minute) // 5 requests per minute
	api.POST("/login", authLimiter, handlers.Login(db))
	api.POST("/register", authLimiter, handlers.Register(db))
	api.POST("/request-password-reset", authLimiter, handlers.RequestPasswordReset(db, emailService))
	api.POST("/reset-password", authLimiter, handlers.ResetPassword(db))

	// Site settings (public read)
	api.GET("/settings", handlers.GetSiteSettings(db))

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		// User routes
		protected.GET("/me", handlers.GetCurrentUser(db))
		protected.GET("/email-preferences", handlers.GetEmailPreferences(db))
		protected.PUT("/email-preferences", handlers.UpdateEmailPreferences(db))
		protected.PUT("/default-group", handlers.SetDefaultGroup(db))
		protected.GET("/default-group", handlers.GetDefaultGroup(db))

		// Announcement routes (all authenticated users can view)
		protected.GET("/announcements", handlers.GetAnnouncements(db))

		// Group routes
		protected.GET("/groups", handlers.GetGroups(db))

		// Comment tags (all authenticated users can view)
		protected.GET("/comment-tags", handlers.GetCommentTags(db))

		// Image upload (authenticated users only)
		protected.POST("/animals/upload-image", handlers.UploadAnimalImage())

		// Admin only routes
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", handlers.GetAllUsers(db))
			admin.POST("/users", handlers.AdminCreateUser(db))
			admin.DELETE("/users/:userId", handlers.AdminDeleteUser(db))
			admin.GET("/users/deleted", handlers.GetDeletedUsers(db))
			admin.POST("/users/:userId/restore", handlers.RestoreUser(db))
			admin.POST("/users/:userId/promote", handlers.PromoteUser(db))
			admin.POST("/users/:userId/demote", handlers.DemoteUser(db))

			// Group management (admin only)
			admin.POST("/groups", handlers.CreateGroup(db))
			admin.PUT("/groups/:id", handlers.UpdateGroup(db))
			admin.DELETE("/groups/:id", handlers.DeleteGroup(db))
			admin.POST("/groups/upload-image", handlers.UploadGroupImage())
			admin.POST("/users/:userId/groups/:groupId", handlers.AddUserToGroup(db))
			admin.DELETE("/users/:userId/groups/:groupId", handlers.RemoveUserFromGroup(db))

			// Announcement routes (admin only)
			admin.POST("/announcements", handlers.CreateAnnouncement(db, emailService))
			admin.DELETE("/announcements/:id", handlers.DeleteAnnouncement(db))

			// Comment tag management (admin only)
			admin.POST("/comment-tags", handlers.CreateCommentTag(db))
			admin.DELETE("/comment-tags/:tagId", handlers.DeleteCommentTag(db))

			// Site settings management (admin only)
			admin.PUT("/settings/:key", handlers.UpdateSiteSetting(db))
			admin.POST("/settings/upload-hero-image", handlers.UploadHeroImage())

			// Bulk animal management (admin only)
			admin.GET("/animals", handlers.GetAllAnimals(db))
			admin.POST("/animals/bulk-update", handlers.BulkUpdateAnimals(db))
			admin.POST("/animals/import-csv", handlers.ImportAnimalsCSV(db))
			admin.GET("/animals/export-csv", handlers.ExportAnimalsCSV(db))
			admin.GET("/animals/export-comments-csv", handlers.ExportAnimalCommentsCSV(db))
			admin.PUT("/animals/:animalId", handlers.UpdateAnimalAdmin(db))
		}

		// Group-specific routes
		group := protected.Group("/groups/:id")
		{
			group.GET("", handlers.GetGroup(db))

			// Animal routes - viewing accessible to all group members
			group.GET("/animals", handlers.GetAnimals(db))
			group.GET("/animals/:animalId", handlers.GetAnimal(db))

			// Animal comments - all group members can view and add comments
			group.GET("/animals/:animalId/comments", handlers.GetAnimalComments(db))
			group.POST("/animals/:animalId/comments", handlers.CreateAnimalComment(db))
			
			// Latest comments across the group
			group.GET("/latest-comments", handlers.GetGroupLatestComments(db))

			// Activity feed - unified view of announcements and comments
			group.GET("/activity-feed", handlers.GetGroupActivityFeed(db))

			// Updates routes
			group.GET("/updates", handlers.GetUpdates(db))
			group.POST("/updates", handlers.CreateUpdate(db))
		}

		// Admin-only animal management routes
		adminAnimals := protected.Group("/groups/:id/animals")
		adminAnimals.Use(middleware.AdminRequired())
		{
			adminAnimals.POST("", handlers.CreateAnimal(db))
			adminAnimals.PUT("/:animalId", handlers.UpdateAnimal(db))
			adminAnimals.DELETE("/:animalId", handlers.DeleteAnimal(db))
		}
	}

	// Serve frontend in production
	if os.Getenv("ENV") == "production" {
		// Serve static files
		router.StaticFile("/", "./frontend/dist/index.html")
		router.Static("/assets", "./frontend/dist/assets")

		// Serve index.html for all non-API routes (SPA routing)
		router.NoRoute(func(c *gin.Context) {
			c.File("./frontend/dist/index.html")
		})
	}

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
