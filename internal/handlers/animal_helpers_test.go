package handlers

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAnimalTestDB creates an in-memory SQLite database for animal testing
func setupAnimalTestDB(t *testing.T) *gorm.DB {
	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.Animal{}, &models.AnimalTag{}, &models.AnimalNameHistory{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createAnimalTestUser creates a user with a group for testing
func createAnimalTestUser(t *testing.T, db *gorm.DB, username, email string, isAdmin bool) (*models.User, *models.Group) {
	hashedPassword, err := auth.HashPassword("password123")
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
		t.Fatalf("Failed to create user: %v", err)
	}

	group := &models.Group{
		Name:        fmt.Sprintf("%s's Group", username),
		Description: "Test group",
	}

	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Associate user with group
	if err := db.Model(user).Association("Groups").Append(group); err != nil {
		t.Fatalf("Failed to associate user with group: %v", err)
	}

	return user, group
}

// createTestAnimal creates an animal in the database for testing
func createTestAnimal(t *testing.T, db *gorm.DB, groupID uint, name, species string) *models.Animal {
	now := time.Now()
	animal := &models.Animal{
		GroupID:          groupID,
		Name:             name,
		Species:          species,
		Breed:            "Test Breed",
		Age:              2,
		Description:      "Test animal",
		Status:           "available",
		ArrivalDate:      &now,
		LastStatusChange: &now,
	}

	if err := db.Create(animal).Error; err != nil {
		t.Fatalf("Failed to create animal: %v", err)
	}

	return animal
}

// setupAnimalTestContext creates a Gin context with authenticated user
func setupAnimalTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set authentication context
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)

	return c, w
}
