package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/handlers"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
)

func main() {
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

	// Set up Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// Public routes
	api := router.Group("/api")
	{
		api.POST("/login", handlers.Login(db))
		api.POST("/register", handlers.Register(db))
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		// User routes
		protected.GET("/me", handlers.GetCurrentUser(db))

		// Group routes
		protected.GET("/groups", handlers.GetGroups(db))
		
		// Admin only routes
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.POST("/groups", handlers.CreateGroup(db))
			admin.PUT("/groups/:id", handlers.UpdateGroup(db))
			admin.DELETE("/groups/:id", handlers.DeleteGroup(db))
			admin.POST("/users/:userId/groups/:groupId", handlers.AddUserToGroup(db))
			admin.DELETE("/users/:userId/groups/:groupId", handlers.RemoveUserFromGroup(db))
		}

		// Group-specific routes
		group := protected.Group("/groups/:id")
		{
			group.GET("", handlers.GetGroup(db))
			
			// Animal routes (group members can access)
			group.GET("/animals", handlers.GetAnimals(db))
			group.GET("/animals/:animalId", handlers.GetAnimal(db))
			group.POST("/animals", handlers.CreateAnimal(db))
			group.PUT("/animals/:animalId", handlers.UpdateAnimal(db))
			group.DELETE("/animals/:animalId", handlers.DeleteAnimal(db))

			// Updates routes
			group.GET("/updates", handlers.GetUpdates(db))
			group.POST("/updates", handlers.CreateUpdate(db))
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
