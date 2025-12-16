package handlers

import (
	"os"
	"testing"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database for testing with all models migrated
func SetupTestDB(t *testing.T) *gorm.DB {
	// Set JWT_SECRET for testing - must be random and secure for validation to pass
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")

	// IMPORTANT: SQLite in-memory databases are per-connection.
	// GORM's connection pool may open multiple connections, which can lead to
	// flaky "no such table" errors if migrations run on one connection and
	// queries execute on another.
	// We force a single connection for deterministic tests.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Run migrations for all models
	err = db.AutoMigrate(
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
		&models.AnimalNameHistory{},
	)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// CreateTestUser creates a user in the test database
func CreateTestUser(t *testing.T, db *gorm.DB, username, email, password string, isAdmin bool) *models.User {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		IsAdmin:  isAdmin,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateTestGroup creates a group in the test database
func CreateTestGroup(t *testing.T, db *gorm.DB, name, description string) *models.Group {
	group := &models.Group{
		Name:        name,
		Description: description,
	}

	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create test group: %v", err)
	}

	return group
}

// CreateTestAnimal creates an animal in the test database
func CreateTestAnimal(t *testing.T, db *gorm.DB, groupID uint, name, species string) *models.Animal {
	animal := &models.Animal{
		GroupID: groupID,
		Name:    name,
		Species: species,
		Status:  "available",
	}

	if err := db.Create(animal).Error; err != nil {
		t.Fatalf("Failed to create test animal: %v", err)
	}

	return animal
}

// AddUserToGroupWithAdmin adds a user to a group and optionally makes them a group admin
func AddUserToGroupWithAdmin(t *testing.T, db *gorm.DB, userID, groupID uint, isGroupAdmin bool) {
	userGroup := &models.UserGroup{
		UserID:       userID,
		GroupID:      groupID,
		IsGroupAdmin: isGroupAdmin,
	}

	if err := db.Create(userGroup).Error; err != nil {
		t.Fatalf("Failed to add user to group: %v", err)
	}
}
