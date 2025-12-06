package database

import (
	"fmt"
	"os"
	"time"

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

	// Get underlying SQL database for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool for security and performance
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database
	// This prevents resource exhaustion attacks
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	// This helps with database connection rotation and security
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Add statement timeout for query security (prevent long-running queries)
	// This is a PostgreSQL-specific setting that prevents queries from running indefinitely
	db.Exec("SET statement_timeout = '30s'")

	logging.Info("Database connection established")
	return db, nil
}

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	logging.Info("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Animal{},
		&models.Update{},
		&models.Announcement{},
		&models.CommentTag{},
		&models.AnimalComment{},
		&models.SiteSetting{},
		&models.Protocol{},
		&models.AnimalTag{},
		&models.AnimalImage{},
		&models.AnimalNameHistory{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logging.Info("Migrations completed successfully")

	// Create default groups if they don't exist
	if err := createDefaultGroups(db); err != nil {
		return err
	}

	// Handle existing null group_ids in animal_tags before creating new tags
	// This is needed for databases upgraded from the non-group-specific tags version
	if err := fixAnimalTagsGroupID(db); err != nil {
		return err
	}

	// Create default animal tags if they don't exist
	if err := createDefaultAnimalTags(db); err != nil {
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

// fixAnimalTagsGroupID migrates existing animal_tags with NULL group_id to use the first group
// This handles databases that were upgraded from the version where tags weren't group-specific
func fixAnimalTagsGroupID(db *gorm.DB) error {
	// Check if the animal_tags table exists and has columns with NULL group_id
	var nullCount int64
	result := db.Raw("SELECT COUNT(*) FROM animal_tags WHERE group_id IS NULL").Scan(&nullCount)
	if result.Error != nil {
		// Table might not exist yet, that's fine
		return nil
	}

	if nullCount == 0 {
		// No NULL values, nothing to fix
		return nil
	}

	logging.WithField("count", nullCount).Info("Found animal_tags with NULL group_id, assigning to first group...")

	// Get the first group (should be modsquad)
	var group models.Group
	if err := db.First(&group).Error; err != nil {
		return fmt.Errorf("failed to find default group for animal_tags migration: %w", err)
	}

	// First, drop the NOT NULL constraint if it exists
	// Use a transaction to ensure atomicity
	tx := db.Begin()

	// Temporarily drop the NOT NULL constraint
	if err := tx.Exec("ALTER TABLE animal_tags DROP CONSTRAINT IF EXISTS animal_tags_group_id_check").Error; err != nil {
		// Constraint might not exist with that name, try another approach
		// This is just a cleanup step, so we can ignore the error
	}

	// Update all NULL group_ids to the first group
	if err := tx.Model(&models.AnimalTag{}).Where("group_id IS NULL").Update("group_id", group.ID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fix animal_tags group_id: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit animal_tags migration: %w", err)
	}

	logging.WithFields(map[string]interface{}{
		"count":      nullCount,
		"group_id":   group.ID,
		"group_name": group.Name,
	}).Info("Successfully migrated animal_tags to group")

	return nil
}

// createDefaultGroups creates the default groups if they don't exist
func createDefaultGroups(db *gorm.DB) error {
	defaultGroups := []models.Group{
		{Name: "modsquad", Description: "Behavior modification volunteers group", HasProtocols: true},
	}

	for _, group := range defaultGroups {
		var existing models.Group
		result := db.Where("name = ?", group.Name).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&group).Error; err != nil {
				return fmt.Errorf("failed to create default group %s: %w", group.Name, err)
			}
			logging.WithField("group_name", group.Name).Info("Created default group")
		} else if result.Error == nil {
			// Update existing modsquad group to enable protocols
			if group.Name == "modsquad" && !existing.HasProtocols {
				existing.HasProtocols = true
				if err := db.Save(&existing).Error; err != nil {
					logging.WithField("group_name", group.Name).Error("Failed to enable protocols for existing group", err)
				} else {
					logging.WithField("group_name", group.Name).Info("Enabled protocols for existing group")
				}
			}
		}
	}

	return nil
}

// createDefaultCommentTags creates the default comment tags for each group if they don't exist
func createDefaultCommentTags(db *gorm.DB) error {
	// Get all groups
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to fetch groups: %w", err)
	}

	defaultTagTemplates := []struct {
		Name     string
		Color    string
		IsSystem bool
	}{
		{Name: "behavior", Color: "#3b82f6", IsSystem: true},
		{Name: "medical", Color: "#ef4444", IsSystem: true},
	}

	for _, group := range groups {
		for _, template := range defaultTagTemplates {
			var existing models.CommentTag
			result := db.Where("name = ? AND group_id = ?", template.Name, group.ID).First(&existing)
			if result.Error == gorm.ErrRecordNotFound {
				tag := models.CommentTag{
					GroupID:  group.ID,
					Name:     template.Name,
					Color:    template.Color,
					IsSystem: template.IsSystem,
				}
				if err := db.Create(&tag).Error; err != nil {
					return fmt.Errorf("failed to create default tag %s for group %s: %w", template.Name, group.Name, err)
				}
				logging.WithFields(map[string]interface{}{
					"tag_name":   template.Name,
					"group_name": group.Name,
				}).Info("Created default comment tag for group")
			}
		}
	}

	return nil
}

// createDefaultAnimalTags creates the default animal tags for each group if they don't exist
func createDefaultAnimalTags(db *gorm.DB) error {
	// Get all groups
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to fetch groups: %w", err)
	}

	defaultTagTemplates := []struct {
		Name     string
		Category string
		Color    string
	}{
		// Behavior tags
		{Name: "resource guarding", Category: "behavior", Color: "#ef4444"},
		{Name: "shy", Category: "behavior", Color: "#a855f7"},
		{Name: "reactive", Category: "behavior", Color: "#f97316"},
		{Name: "friendly", Category: "behavior", Color: "#22c55e"},
		// Walker status tags (only these 3)
		{Name: "iso", Category: "walker_status", Color: "#ef4444"},
		{Name: "experienced only", Category: "walker_status", Color: "#8b5cf6"},
		{Name: "dual walker", Category: "walker_status", Color: "#06b6d4"},
	}

	for _, group := range groups {
		for _, template := range defaultTagTemplates {
			var existing models.AnimalTag
			result := db.Where("name = ? AND group_id = ?", template.Name, group.ID).First(&existing)
			if result.Error == gorm.ErrRecordNotFound {
				tag := models.AnimalTag{
					GroupID:  group.ID,
					Name:     template.Name,
					Category: template.Category,
					Color:    template.Color,
				}
				if err := db.Create(&tag).Error; err != nil {
					return fmt.Errorf("failed to create default animal tag %s for group %s: %w", template.Name, group.Name, err)
				}
				logging.WithFields(map[string]interface{}{
					"tag_name":   template.Name,
					"group_name": group.Name,
				}).Info("Created default animal tag for group")
			}
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
