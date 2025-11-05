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

func setupAdminDashboardTestDB(t *testing.T) *gorm.DB {
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

	return db
}

func TestGetAdminDashboardStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of admin dashboard stats",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAdminDashboardTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/admin/dashboard/stats", nil)

			// Execute
			handler := GetAdminDashboardStats(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "total_users")
			assert.Contains(t, w.Body.String(), "total_groups")
			assert.Contains(t, w.Body.String(), "total_animals")
			assert.Contains(t, w.Body.String(), "total_comments")
		})
	}
}
