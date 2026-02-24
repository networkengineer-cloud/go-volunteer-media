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
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{}, &models.AnimalTag{}, &models.AnimalNameHistory{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createAnimalTestUser creates a user with a group for testing
// The user is automatically made a group admin for the created group (for backward compatibility with existing tests)
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

	// Create UserGroup entry with group admin privileges
	// This maintains backward compatibility with existing tests that assume
	// group members can perform admin operations on their groups
	userGroup := &models.UserGroup{
		UserID:       user.ID,
		GroupID:      group.ID,
		IsGroupAdmin: true, // Make user a group admin by default for test backward compatibility
	}
	if err := db.Create(userGroup).Error; err != nil {
		t.Fatalf("Failed to create user group association: %v", err)
	}

	return user, group
}

// createTestAnimal creates an animal in the database for testing
func createTestAnimal(t *testing.T, db *gorm.DB, groupID uint, name, species string) *models.Animal {
	now := time.Now()
	birthDate := now.AddDate(-2, -3, 0) // 2 years 3 months ago
	animal := &models.Animal{
		GroupID:            groupID,
		Name:               name,
		Species:            species,
		Breed:              "Test Breed",
		Age:                2,
		EstimatedBirthDate: &birthDate,
		Description:        "Test animal",
		Status:             "available",
		ArrivalDate:        &now,
		LastStatusChange:   &now,
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

// TestNullableTime_UnmarshalJSON tests the NullableTime unmarshaling
func TestNullableTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectNil bool
		expectErr bool
		checkDate bool // Only check date part, not time
	}{
		{
			name:      "null value",
			input:     `null`,
			expectNil: true,
			expectErr: false,
		},
		{
			name:      "empty string",
			input:     `""`,
			expectNil: true,
			expectErr: false,
		},
		{
			name:      "date only format (YYYY-MM-DD)",
			input:     `"2025-11-24"`,
			expectNil: false,
			expectErr: false,
			checkDate: true,
		},
		{
			name:      "RFC3339 timestamp",
			input:     `"2025-11-24T10:30:00Z"`,
			expectNil: false,
			expectErr: false,
		},
		{
			name:      "RFC3339 timestamp with timezone",
			input:     `"2025-11-24T10:30:00-05:00"`,
			expectNil: false,
			expectErr: false,
		},
		{
			name:      "invalid format",
			input:     `"not a date"`,
			expectNil: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nt NullableTime
			err := nt.UnmarshalJSON([]byte(tt.input))

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectNil {
				if nt.Valid {
					t.Error("Expected Valid to be false for nil value")
				}
				if nt.Time != nil {
					t.Error("Expected Time to be nil")
				}
			} else {
				if !nt.Valid {
					t.Error("Expected Valid to be true")
				}
				if nt.Time == nil {
					t.Error("Expected Time to be set")
					return
				}

				// For date-only inputs, verify the date part matches
				if tt.checkDate {
					expected := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
					if nt.Time.Year() != expected.Year() ||
						nt.Time.Month() != expected.Month() ||
						nt.Time.Day() != expected.Day() {
						t.Errorf("Date mismatch: got %v, expected date to be 2025-11-24", nt.Time)
					}
				}
			}
		})
	}
}
