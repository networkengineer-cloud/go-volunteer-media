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
			name:           "bad request when value is missing",
			key:            "site_name",
			requestBody:    map[string]interface{}{},
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

// TestUpdateSiteSetting_Validation tests comprehensive validation rules
func TestUpdateSiteSetting_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		key            string
		value          string
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		// site_name validation (required, max 100 chars)
		{
			name:           "site_name: reject empty string",
			key:            "site_name",
			value:          "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "site_name is required",
		},
		{
			name:           "site_name: reject whitespace only",
			key:            "site_name",
			value:          "   ",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "site_name is required",
		},
		{
			name:           "site_name: accept valid short name",
			key:            "site_name",
			value:          "Pet Shelter",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_name: accept max length (100 chars)",
			key:            "site_name",
			value:          "A" + string(make([]byte, 99)), // 100 'A's
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_name: reject over max length (101 chars)",
			key:            "site_name",
			value:          "A" + string(make([]byte, 100)), // 101 'A's
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "must be 100 characters or less",
		},

		// site_short_name validation (required, max 50 chars)
		{
			name:           "site_short_name: reject empty string",
			key:            "site_short_name",
			value:          "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "site_short_name is required",
		},
		{
			name:           "site_short_name: reject whitespace only",
			key:            "site_short_name",
			value:          "  \t  ",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "site_short_name is required",
		},
		{
			name:           "site_short_name: accept valid name",
			key:            "site_short_name",
			value:          "PetApp",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_short_name: accept max length (50 chars)",
			key:            "site_short_name",
			value:          "B" + string(make([]byte, 49)), // 50 'B's
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_short_name: reject over max length (51 chars)",
			key:            "site_short_name",
			value:          "B" + string(make([]byte, 50)), // 51 'B's
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "must be 50 characters or less",
		},

		// site_description validation (optional, max 500 chars)
		{
			name:           "site_description: accept empty string (optional)",
			key:            "site_description",
			value:          "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_description: accept valid description",
			key:            "site_description",
			value:          "A volunteer management system for animal shelters.",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_description: accept max length (500 chars)",
			key:            "site_description",
			value:          "C" + string(make([]byte, 499)), // 500 'C's
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "site_description: reject over max length (501 chars)",
			key:            "site_description",
			value:          "C" + string(make([]byte, 500)), // 501 'C's
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "must be 500 characters or less",
		},

		// hero_image_url validation (optional, max 500 chars)
		{
			name:           "hero_image_url: accept empty string",
			key:            "hero_image_url",
			value:          "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "hero_image_url: accept valid URL",
			key:            "hero_image_url",
			value:          "https://example.com/images/hero.jpg",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "hero_image_url: reject over max length (501 chars)",
			key:            "hero_image_url",
			value:          "https://example.com/" + string(make([]byte, 481)), // 501 total
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "must be 500 characters or less",
		},

		// Unknown keys (should be accepted - no validation rules)
		{
			name:           "unknown_key: accept without validation",
			key:            "custom_setting",
			value:          "any value",
			expectedStatus: http.StatusOK,
			expectError:    false,
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

			requestBody := map[string]interface{}{"value": tt.value}
			bodyBytes, _ := json.Marshal(requestBody)
			c.Request = httptest.NewRequest("PUT", "/settings/"+tt.key, bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "key", Value: tt.key}}

			// Execute
			handler := UpdateSiteSetting(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch for test: %s", tt.name)

			if tt.expectError {
				assert.Contains(t, w.Body.String(), tt.errorContains, "Error message mismatch for test: %s", tt.name)
			} else {
				// Verify successful update persisted to database
				var updatedSetting models.SiteSetting
				err := db.Where("key = ?", tt.key).First(&updatedSetting).Error
				assert.NoError(t, err, "Failed to retrieve updated setting from database")
				assert.Equal(t, tt.value, updatedSetting.Value, "Database value mismatch for test: %s", tt.name)
			}
		})
	}
}

// TestUpdateSiteSetting_UpsertBehavior tests that settings are created if they don't exist
func TestUpdateSiteSetting_UpsertBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	db := setupSettingsTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Verify 'new_setting_key' does not exist
	var existingCount int64
	db.Model(&models.SiteSetting{}).Where("key = ?", "new_setting_key").Count(&existingCount)
	assert.Equal(t, int64(0), existingCount, "Setting should not exist initially")

	// Create new setting via UpdateSiteSetting
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	requestBody := map[string]interface{}{"value": "New Setting Value"}
	bodyBytes, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest("PUT", "/settings/new_setting_key", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "new_setting_key"}}

	handler := UpdateSiteSetting(db)
	handler(c)

	// Assert success
	assert.Equal(t, http.StatusOK, w.Code, "Expected successful creation")

	// Verify setting was created in database
	var newSetting models.SiteSetting
	err := db.Where("key = ?", "new_setting_key").First(&newSetting).Error
	assert.NoError(t, err, "Setting should exist after upsert")
	assert.Equal(t, "New Setting Value", newSetting.Value, "Value should match")

	// Update the same setting
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)

	requestBody2 := map[string]interface{}{"value": "Updated Setting Value"}
	bodyBytes2, _ := json.Marshal(requestBody2)
	c2.Request = httptest.NewRequest("PUT", "/settings/new_setting_key", bytes.NewBuffer(bodyBytes2))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Params = gin.Params{{Key: "key", Value: "new_setting_key"}}

	handler2 := UpdateSiteSetting(db)
	handler2(c2)

	// Assert success
	assert.Equal(t, http.StatusOK, w2.Code, "Expected successful update")

	// Verify setting was updated (not duplicated)
	var updatedSetting models.SiteSetting
	err = db.Where("key = ?", "new_setting_key").First(&updatedSetting).Error
	assert.NoError(t, err, "Setting should still exist after update")
	assert.Equal(t, "Updated Setting Value", updatedSetting.Value, "Value should be updated")

	// Verify only one record exists
	var finalCount int64
	db.Model(&models.SiteSetting{}).Where("key = ?", "new_setting_key").Count(&finalCount)
	assert.Equal(t, int64(1), finalCount, "Should only have one setting record (no duplicates)")
}
