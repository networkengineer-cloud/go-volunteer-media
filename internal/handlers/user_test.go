package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
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

	group1 := models.Group{
		Name:        "Test Group 1",
		Description: "Test group 1",
	}
	group2 := models.Group{
		Name:        "Test Group 2",
		Description: "Test group 2",
	}
	db.Create(&group1)
	db.Create(&group2)

	// Add user to group1
	db.Model(&user).Association("Groups").Append(&group1)

	return db
}

func TestGetAllUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of all users",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUserTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/users", nil)

			// Execute
			handler := GetAllUsers(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "testuser")
		})
	}
}

func TestSetDefaultGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful default group setting",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
			},
			requestBody: SetDefaultGroupRequest{
				GroupID: 1,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when user doesn't have access to group",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
			},
			requestBody: SetDefaultGroupRequest{
				GroupID: 2,
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "You do not have access to this group",
		},
		{
			name: "admin can set any group as default",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", true)
			},
			requestBody: SetDefaultGroupRequest{
				GroupID: 2,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "bad request when group_id is missing",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
			},
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUserTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/users/default-group", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			tt.setupContext(c)

			// Execute
			handler := SetDefaultGroup(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestGetDefaultGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupData      func(*gorm.DB)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful retrieval when default group is set",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupData: func(db *gorm.DB) {
				var user models.User
				db.First(&user, 1)
				groupID := uint(1)
				user.DefaultGroupID = &groupID
				db.Save(&user)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Test Group 1",
		},
		{
			name: "returns null when no default group set",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupData:      func(db *gorm.DB) {},
			expectedStatus: http.StatusOK,
			expectedBody:   "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUserTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			tt.setupData(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/users/default-group", nil)
			tt.setupContext(c)

			// Execute
			handler := GetDefaultGroup(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
