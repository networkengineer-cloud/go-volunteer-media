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

	// CRITICAL: Drop legacy single-column unique indexes BEFORE AutoMigrate
	// These old indexes conflict with the new composite indexes (group_id, name)
	// GORM AutoMigrate won't remove old indexes when index names change
	if err := dropLegacyIndexes(db); err != nil {
		logging.WithField("error", err.Error()).Warn("Failed to drop legacy indexes (may not exist)")
	}

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

	// CRITICAL: Fix NULL group_ids and add NOT NULL constraint AFTER AutoMigrate
	// AutoMigrate allows NULL values, so we fix them here, then add the constraint
	if err := fixAndEnforceGroupIDConstraints(db); err != nil {
		logging.WithField("error", err.Error()).Warn("Failed to fix group_id constraints (may be first run)")
	}

	// Create default groups if they don't exist
	if err := createDefaultGroups(db); err != nil {
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

// fixAndEnforceGroupIDConstraints fixes NULL group_ids in tag tables and adds NOT NULL constraint
// This runs AFTER AutoMigrate to ensure the tables exist first
func fixAndEnforceGroupIDConstraints(db *gorm.DB) error {
	// Get or create a default group
	var groupID uint
	if err := db.Raw("SELECT id FROM groups ORDER BY id LIMIT 1").Scan(&groupID).Error; err != nil {
		logging.WithField("error", err.Error()).Warn("Failed to query for first group")
		return nil
	}

	// If no groups exist, create the default one
	if groupID == 0 {
		logging.Info("No groups exist, creating default group to assign to tag records")
		if err := db.Exec(`
			INSERT INTO groups (name, description, has_protocols, created_at, updated_at)
			VALUES ('modsquad', 'Behavior modification volunteers group', true, NOW(), NOW())
			ON CONFLICT (name) DO NOTHING
		`).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to create default group")
			return nil
		}

		// Get the group ID again
		if err := db.Raw("SELECT id FROM groups WHERE name = 'modsquad' LIMIT 1").Scan(&groupID).Error; err != nil || groupID == 0 {
			logging.Warn("Still no group available after creation attempt")
			return nil
		}
	}

	// Fix animal_tags: set NULL values to the group ID, then add NOT NULL constraint
	if err := fixAndEnforceTableConstraint(db, "animal_tags", groupID); err != nil {
		logging.WithField("error", err.Error()).Warn("Failed to fix and enforce animal_tags constraint")
	}

	// Fix comment_tags: set NULL values to the group ID, then add NOT NULL constraint
	if err := fixAndEnforceTableConstraint(db, "comment_tags", groupID); err != nil {
		logging.WithField("error", err.Error()).Warn("Failed to fix and enforce comment_tags constraint")
	}

	return nil
}

// fixAndEnforceTableConstraint fixes NULL group_ids in a specific table and adds NOT NULL constraint
func fixAndEnforceTableConstraint(db *gorm.DB, tableName string, groupID uint) error {
	// Check if table and column exist
	var columnExists bool
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_name = '%s' AND column_name = 'group_id'
		)
	`, tableName)
	if err := db.Raw(query).Scan(&columnExists).Error; err != nil || !columnExists {
		return nil // Column doesn't exist yet
	}

	// Count NULL values
	var nullCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE group_id IS NULL", tableName)
	if err := db.Raw(countQuery).Scan(&nullCount).Error; err != nil {
		return nil
	}

	// Fix NULL values
	if nullCount > 0 {
		logging.WithFields(map[string]interface{}{
			"table": tableName,
			"count": nullCount,
		}).Info("Fixing NULL group_ids in table...")

		updateQuery := fmt.Sprintf("UPDATE %s SET group_id = ? WHERE group_id IS NULL", tableName)
		if result := db.Exec(updateQuery, groupID); result.Error != nil {
			return result.Error
		}
		logging.WithFields(map[string]interface{}{
			"table":        tableName,
			"rows_updated": nullCount,
			"group_id":     groupID,
		}).Info("Fixed NULL group_ids in table")
	}

	// Check if NOT NULL constraint already exists
	var constraintExists bool
	constraintQuery := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_name = '%s' AND column_name = 'group_id' AND is_nullable = 'NO'
		)
	`, tableName)

	if err := db.Raw(constraintQuery).Scan(&constraintExists).Error; err == nil && constraintExists {
		logging.WithField("table", tableName).Debug("NOT NULL constraint already exists")
		return nil
	}

	// Add NOT NULL constraint using ALTER TABLE
	alterQuery := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN group_id SET NOT NULL", tableName)
	if result := db.Exec(alterQuery); result.Error != nil {
		// Ignore errors if constraint already exists
		logging.WithFields(map[string]interface{}{
			"table": tableName,
			"error": result.Error.Error(),
		}).Debug("Error adding NOT NULL constraint (may already exist)")
		return nil
	}

	logging.WithField("table", tableName).Info("Added NOT NULL constraint to group_id")
	return nil
}

// dropLegacyIndexes drops old single-column unique indexes that conflict with new composite indexes
// These legacy indexes were created before the (group_id, name) composite unique constraint was added
// GORM AutoMigrate doesn't remove old indexes when index names change, so we must do it manually
func dropLegacyIndexes(db *gorm.DB) error {
	// These are hardcoded index names from our own schema history - not user input
	legacyIndexNames := []string{
		// Old index on animal_tags.name (should be replaced by idx_animal_tag_group_name)
		"idx_animal_tags_name",
		// Old index on comment_tags.name (should be replaced by idx_comment_tag_group_name)
		"idx_comment_tags_name",
	}

	for _, indexName := range legacyIndexNames {
		// Use PostgreSQL's quote_ident function for safe identifier quoting
		// This prevents SQL injection even though we control the index names
		dropQuery := "DROP INDEX IF EXISTS " + quoteIdentifier(indexName)
		if err := db.Exec(dropQuery).Error; err != nil {
			logging.WithFields(map[string]interface{}{
				"index": indexName,
				"error": err.Error(),
			}).Warn("Failed to drop legacy index (may not exist)")
		} else {
			logging.WithField("index", indexName).Debug("Attempted to drop legacy index (if existed)")
		}
	}

	return nil
}

// quoteIdentifier safely quotes a PostgreSQL identifier (table name, column name, index name)
// to prevent SQL injection. This follows PostgreSQL's identifier quoting rules.
func quoteIdentifier(name string) string {
	// PostgreSQL identifiers are quoted by doubling internal quotes and wrapping in quotes
	// Since our index names are hardcoded and don't contain quotes, this is straightforward
	return `"` + name + `"`
}

// createDefaultGroups creates the default groups if they don't exist
func createDefaultGroups(db *gorm.DB) error {
	defaultGroups := []models.Group{
		{Name: "modsquad", Description: "Behavior modification volunteers group", HasProtocols: true},
		{Name: "activity-sandbox", Description: "Empty group reserved for automated tests", HasProtocols: false},
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
			var existingTag models.CommentTag
			// Check if tag already exists for this group
			checkResult := db.Where("group_id = ? AND name = ?", group.ID, template.Name).First(&existingTag)
			
			if checkResult.Error == gorm.ErrRecordNotFound {
				// Tag doesn't exist, create it
				tag := models.CommentTag{
					GroupID:  group.ID,
					Name:     template.Name,
					Color:    template.Color,
					IsSystem: template.IsSystem,
				}
				if err := db.Create(&tag).Error; err != nil {
					return fmt.Errorf("failed to create default tag %s for group %s: %w", template.Name, group.Name, err)
				}
			} else if checkResult.Error != nil {
				// Some other database error occurred
				return fmt.Errorf("failed to check for existing comment tag %s in group %s: %w", template.Name, group.Name, checkResult.Error)
			}
			// else: tag already exists, skip it
			
			logging.WithFields(map[string]interface{}{
				"tag_name":   template.Name,
				"group_name": group.Name,
			}).Debug("Ensured default comment tag exists for group")
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
			var existingTag models.AnimalTag
			// Check if tag already exists for this group
			checkResult := db.Where("group_id = ? AND name = ?", group.ID, template.Name).First(&existingTag)
			
			if checkResult.Error == gorm.ErrRecordNotFound {
				// Tag doesn't exist, create it
				tag := models.AnimalTag{
					GroupID:  group.ID,
					Name:     template.Name,
					Category: template.Category,
					Color:    template.Color,
				}
				if err := db.Create(&tag).Error; err != nil {
					return fmt.Errorf("failed to create default animal tag %s for group %s: %w", template.Name, group.Name, err)
				}
			} else if checkResult.Error != nil {
				// Some other database error occurred
				return fmt.Errorf("failed to check for existing animal tag %s in group %s: %w", template.Name, group.Name, checkResult.Error)
			}
			// else: tag already exists, skip it
			
			logging.WithFields(map[string]interface{}{
				"tag_name":   template.Name,
				"group_name": group.Name,
			}).Debug("Ensured default animal tag exists for group")
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
