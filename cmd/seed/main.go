package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

func main() {
	// Initialize logging
	logging.InitFromEnv()
	logger := logging.GetDefaultLogger()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using system environment variables")
	}

	logger.Info("Starting database seed process...")

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
	}()

	// Run migrations first to ensure all tables exist
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations", err)
	}

	// Check if force flag is provided
	force := false
	if len(os.Args) > 1 && os.Args[1] == "--force" {
		force = true
		logger.Info("Force flag detected - will seed data even if users exist")
	}

	// Seed data
	if err := database.SeedData(db, force); err != nil {
		logger.Fatal("Failed to seed database", err)
	}

	fmt.Println("\n✅ Database seeding completed successfully!")
	// Output demo credentials
	fmt.Println("\n=================================")
	fmt.Println("Database seeded successfully!")
	fmt.Println("=================================")
	fmt.Println("\nDemo Accounts:")
	fmt.Println("  Site Admin (password: demo1234):              admin")
	fmt.Println("  Group Admin (password: demo1234):             merry")
	fmt.Println("  Group Admin (password: demo1234):             sophia")
	fmt.Println("  Volunteer (password: volunteer2026!):         terry")
	fmt.Println("  Volunteer (password: volunteer2026!):         alex")
	fmt.Println("  Volunteer (password: volunteer2026!):         jordan")
	fmt.Println("  Volunteer (password: volunteer2026!):         casey")
	fmt.Println("  Volunteer (password: volunteer2026!):         taylor")
	fmt.Println("\nAll users have access to the ModSquad group.")
	fmt.Println("merry and sophia are group admins for ModSquad.")
	fmt.Println("ModSquad has 10 dogs with Unsplash images!")
	fmt.Println("Email notifications are disabled by default (opt-in).")
	fmt.Println("=================================")
}
