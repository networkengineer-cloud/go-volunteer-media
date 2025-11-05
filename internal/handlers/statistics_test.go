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

func setupStatisticsTestDB(t *testing.T) *gorm.DB {
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

	tag := models.CommentTag{
		Name:  "urgent",
		Color: "#FF0000",
	}
	db.Create(&tag)
	db.Model(&comment).Association("Tags").Append(&tag)

	return db
}

func TestGetGroupStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of group statistics",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupStatisticsTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/statistics/groups", nil)

			// Execute
			handler := GetGroupStatistics(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "group_id")
			assert.Contains(t, w.Body.String(), "user_count")
			assert.Contains(t, w.Body.String(), "animal_count")
		})
	}
}

func TestGetUserStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of user statistics",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupStatisticsTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/statistics/users", nil)

			// Execute
			handler := GetUserStatistics(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "user_id")
			assert.Contains(t, w.Body.String(), "comment_count")
		})
	}
}

func TestGetCommentTagStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of comment tag statistics",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupStatisticsTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/statistics/comment-tags", nil)

			// Execute
			handler := GetCommentTagStatistics(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "tag_id")
			assert.Contains(t, w.Body.String(), "usage_count")
		})
	}
}
