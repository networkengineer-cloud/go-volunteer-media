package database

import (
	"fmt"
	"os"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initialize creates and returns a database connection
func Initialize() (*gorm.DB, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Default values for development
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "volunteer_media_dev"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	// Validate SSL mode to prevent injection
	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[dbSSLMode] {
		return nil, fmt.Errorf("invalid SSL mode: %s (must be one of: disable, require, verify-ca, verify-full)", dbSSLMode)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logging.Info("Database connection established")
	return db, nil
}

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	logging.Info("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Animal{},
		&models.Update{},
		&models.Announcement{},
		&models.CommentTag{},
		&models.AnimalComment{},
		&models.SiteSetting{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logging.Info("Migrations completed successfully")

	// Create default groups if they don't exist
	if err := createDefaultGroups(db); err != nil {
		return err
	}

	// Create default comment tags if they don't exist
	if err := createDefaultCommentTags(db); err != nil {
		return err
	}

	// Create default site settings if they don't exist
	if err := createDefaultSiteSettings(db); err != nil {
		return err
	}

	return nil
}

// createDefaultGroups creates the default groups if they don't exist
func createDefaultGroups(db *gorm.DB) error {
	defaultGroups := []models.Group{
		{Name: "dogs", Description: "Dog volunteers group"},
		{Name: "cats", Description: "Cat volunteers group"},
		{Name: "modsquad", Description: "Mod Squad volunteers group"},
	}

	for _, group := range defaultGroups {
		var existing models.Group
		result := db.Where("name = ?", group.Name).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&group).Error; err != nil {
				return fmt.Errorf("failed to create default group %s: %w", group.Name, err)
			}
			logging.WithField("group_name", group.Name).Info("Created default group")
		}
	}

	return nil
}

// createDefaultCommentTags creates the default comment tags if they don't exist
func createDefaultCommentTags(db *gorm.DB) error {
	defaultTags := []models.CommentTag{
		{Name: "behavior", Color: "#3b82f6", IsSystem: true},
		{Name: "medical", Color: "#ef4444", IsSystem: true},
	}

	for _, tag := range defaultTags {
		var existing models.CommentTag
		result := db.Where("name = ?", tag.Name).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&tag).Error; err != nil {
				return fmt.Errorf("failed to create default tag %s: %w", tag.Name, err)
			}
			logging.WithField("tag_name", tag.Name).Info("Created default comment tag")
		}
	}

	return nil
}

// createDefaultSiteSettings creates the default site settings if they don't exist
func createDefaultSiteSettings(db *gorm.DB) error {
	defaultSettings := []models.SiteSetting{
		{
			Key:   "hero_image_url",
			Value: "", // Empty by default - admin should upload an image
		},
	}

	for _, setting := range defaultSettings {
		var existing models.SiteSetting
		result := db.Where("key = ?", setting.Key).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to create default setting %s: %w", setting.Key, err)
			}
			logging.WithField("setting_key", setting.Key).Info("Created default site setting")
		}
	}

	return nil
}
