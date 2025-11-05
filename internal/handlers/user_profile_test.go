package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserProfileTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Animal{},
		&models.AnimalComment{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test data
	user := models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	db.Create(&user)

	group := models.Group{
		Name:        "Test Group",
		Description: "Test group description",
	}
	db.Create(&group)

	db.Model(&user).Association("Groups").Append(&group)

	return db
}

func TestGetUserProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		userID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of own profile",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			userID:         "1",
			expectedStatus: http.StatusOK,
		},
		{
			name: "admin can view any profile",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("is_admin", true)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			userID:         "1",
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when trying to view another user's profile",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			userID:         "1",
			expectedStatus: http.StatusForbidden,
			expectedError:  "You can only view your own profile",
		},
		{
			name: "not found when user doesn't exist",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "999"}}
			},
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name: "bad request when user ID is invalid",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "invalid"}}
			},
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUserProfileTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/users/"+tt.userID+"/profile", nil)
			tt.setupContext(c)

			// Execute
			handler := GetUserProfile(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
