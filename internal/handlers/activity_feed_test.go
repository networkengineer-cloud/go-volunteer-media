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

func setupActivityFeedTestDB(t *testing.T) *gorm.DB {
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
		&models.Update{},
		&models.CommentTag{},
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

	// Add user to group
	db.Model(&user).Association("Groups").Append(&group)

	animal := models.Animal{
		Name:        "Test Animal",
		Species:     "Dog",
		GroupID:     group.ID,
		Status:      "available",
		Description: "Test animal",
	}
	db.Create(&animal)

	comment := models.AnimalComment{
		AnimalID: animal.ID,
		UserID:   user.ID,
		Content:  "Test comment",
	}
	db.Create(&comment)

	update := models.Update{
		GroupID: group.ID,
		UserID:  user.ID,
		Title:   "Test Update",
		Content: "Test update content",
	}
	db.Create(&update)

	return db
}

func TestGetGroupActivityFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		queryString    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of activity feed",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "successful with limit parameter",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?limit=5",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with offset parameter",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?offset=0&limit=10",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with type filter for comments",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?type=comments",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with type filter for announcements",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?type=announcements",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupActivityFeedTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/1/activity"+tt.queryString, nil)
			tt.setupContext(c)

			// Execute
			handler := GetGroupActivityFeed(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
