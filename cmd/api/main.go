package main

import (
	"context"
	"log"
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
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
)

func main() {
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
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database
	db, err := database.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Get underlying SQL database for proper connection management
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		log.Println("Database connection closed")
	}()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize email service
	emailService := email.NewService()
	if emailService.IsConfigured() {
		log.Println("Email service configured and ready")
	} else {
		log.Println("Email service not configured - password reset and email notifications will be disabled")
	}

	// Set up Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// Serve uploaded images
	router.Static("/uploads", "./public/uploads")

	// Public routes
	api := router.Group("/api")
	{
		api.POST("/login", handlers.Login(db))
		api.POST("/register", handlers.Register(db))
		api.POST("/request-password-reset", handlers.RequestPasswordReset(db, emailService))
		api.POST("/reset-password", handlers.ResetPassword(db))
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		// User routes
		protected.GET("/me", handlers.GetCurrentUser(db))
		protected.GET("/email-preferences", handlers.GetEmailPreferences(db))
		protected.PUT("/email-preferences", handlers.UpdateEmailPreferences(db))

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
		router.Static("/", "./frontend/dist")
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
		log.Printf("Server starting on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}
