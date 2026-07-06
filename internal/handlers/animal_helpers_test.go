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

	// SQLite in-memory databases are per-connection; pin to one to share state.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Run migrations
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Animal{},
		&models.AnimalTag{},
		&models.AnimalNameHistory{},
		&models.AnimalImage{},
		&models.AnimalVideo{},
	)
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

// TestResolveQuarantineEndDate covers the shared "explicit override, validated
// against start; else computed default" rule used by CreateAnimal, UpdateAnimal,
// and UpdateAnimalAdmin.
func TestResolveQuarantineEndDate(t *testing.T) {
	t.Run("no explicit end date returns the computed default", func(t *testing.T) {
		start := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC) // Monday
		result, err := resolveQuarantineEndDate(&start, NullableTime{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC) // Thursday, 10 days later
		if result == nil || !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("explicit end date after start is honored verbatim", func(t *testing.T) {
		start := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
		result, err := resolveQuarantineEndDate(&start, NullableTime{Time: &end, Valid: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil || !result.Equal(end) {
			t.Errorf("expected %v, got %v", end, result)
		}
	})

	t.Run("explicit end date before start's calendar day is rejected", func(t *testing.T) {
		start := time.Date(2025, 11, 10, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC)
		_, err := resolveQuarantineEndDate(&start, NullableTime{Time: &end, Valid: true})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if err.Error() != "quarantine end date cannot be before start date" {
			t.Errorf("unexpected error message: %q", err.Error())
		}
	})

	t.Run("explicit end date same calendar day as start is accepted even with a later time-of-day start", func(t *testing.T) {
		// Regression: start defaults to time.Now() (a real timestamp), while a
		// date-only end date parses to UTC midnight. Comparing exact instants
		// would wrongly reject a same-day end date; comparison must be by
		// calendar day.
		start := time.Date(2026, 7, 2, 14, 30, 0, 0, time.UTC) // 2:30pm
		end := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)     // midnight, same day
		result, err := resolveQuarantineEndDate(&start, NullableTime{Time: &end, Valid: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil || !result.Equal(end) {
			t.Errorf("expected %v, got %v", end, result)
		}
	})

	t.Run("explicit end date on the calendar day before start, even with an earlier time-of-day, is rejected", func(t *testing.T) {
		start := time.Date(2026, 7, 2, 23, 0, 0, 0, time.UTC)  // 11pm
		end := time.Date(2026, 7, 1, 1, 0, 0, 0, time.UTC)     // 1am, previous calendar day
		_, err := resolveQuarantineEndDate(&start, NullableTime{Time: &end, Valid: true})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("nil start with no explicit end date returns nil", func(t *testing.T) {
		result, err := resolveQuarantineEndDate(nil, NullableTime{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("explicit end date with nil start is rejected", func(t *testing.T) {
		// Reachable for an animal whose status is bite_quarantine but whose
		// QuarantineStartDate is nil (e.g. set via CSV import, which validates the
		// status value but never touches quarantine dates) — an explicit end date
		// can't be validated against a start that doesn't exist, so it must be
		// rejected rather than silently stored.
		end := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
		_, err := resolveQuarantineEndDate(nil, NullableTime{Time: &end, Valid: true})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if err.Error() != "quarantine end date cannot be set without a quarantine start date" {
			t.Errorf("unexpected error message: %q", err.Error())
		}
	})
}
