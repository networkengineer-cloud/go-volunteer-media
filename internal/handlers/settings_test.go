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

func setupSettingsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(&models.SiteSetting{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test settings
	setting1 := models.SiteSetting{Key: "site_name", Value: "Test Site"}
	setting2 := models.SiteSetting{Key: "hero_image_url", Value: "http://example.com/hero.jpg"}
	db.Create(&setting1)
	db.Create(&setting2)

	return db
}

func TestGetSiteSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "successful retrieval of site settings",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupSettingsTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/settings", nil)

			// Execute
			handler := GetSiteSettings(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "site_name")
			assert.Contains(t, w.Body.String(), "Test Site")
		})
	}
}

func TestUpdateSiteSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		key            string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful update of existing setting",
			key:  "site_name",
			requestBody: map[string]interface{}{
				"value": "Updated Site Name",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful creation of new setting",
			key:  "new_setting",
			requestBody: map[string]interface{}{
				"value": "New Value",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "bad request when value is missing",
			key:  "site_name",
			requestBody: map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupSettingsTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("PUT", "/settings/"+tt.key, bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "key", Value: tt.key}}

			// Execute
			handler := UpdateSiteSetting(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
