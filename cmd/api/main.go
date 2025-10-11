package main

import (
	"log"
	"os"

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

	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
