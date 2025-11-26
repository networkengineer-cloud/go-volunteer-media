package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnimalTagTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(&models.AnimalTag{}, &models.Animal{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test tags
	tag1 := models.AnimalTag{Name: "friendly", Category: "behavior", Color: "#00FF00", Icon: "😊"}
	tag2 := models.AnimalTag{Name: "needs-walker", Category: "walker_status", Color: "#FF0000", Icon: "🚶"}
	db.Create(&tag1)
	db.Create(&tag2)

	return db
}

func TestGetAnimalTags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "successful retrieval of all tags",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/tags", nil)

			// Execute
			handler := GetAnimalTags(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var tags []models.AnimalTag
			json.Unmarshal(w.Body.Bytes(), &tags)
			assert.Equal(t, tt.expectedCount, len(tags))
		})
	}
}

func TestCreateAnimalTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up logger for tests
	logging.SetLevel(logging.ERROR)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful tag creation",
			requestBody: AnimalTagRequest{
				Name:     "energetic",
				Category: "behavior",
				Color:    "#FFFF00",
				Icon:     "⚡",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "bad request when name is missing",
			requestBody: AnimalTagRequest{
				Category: "behavior",
				Color:    "#FFFF00",
				Icon:     "⚡",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request when category is invalid",
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "invalid_category",
				Color:    "#FFFF00",
				Icon:     "⚡",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request when color is missing",
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
				Icon:     "⚡",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// Add logger to context
			c.Set("logger", logging.GetDefaultLogger())

			// Execute
			handler := CreateAnimalTag(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestUpdateAnimalTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logging.SetLevel(logging.ERROR)

	tests := []struct {
		name           string
		tagID          string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:  "successful tag update",
			tagID: "1",
			requestBody: AnimalTagRequest{
				Name:     "very-friendly",
				Category: "behavior",
				Color:    "#00FFFF",
				Icon:     "😄",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "not found when tag doesn't exist",
			tagID: "999",
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
				Color:    "#FFFFFF",
				Icon:     "❓",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal tag not found",
		},
		{
			name:  "bad request when data is invalid",
			tagID: "1",
			requestBody: AnimalTagRequest{
				Name:     "",
				Category: "behavior",
				Color:    "#FFFFFF",
				Icon:     "❓",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("PUT", "/tags/"+tt.tagID, bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "tagId", Value: tt.tagID}}
			c.Set("logger", logging.GetDefaultLogger())

			// Execute
			handler := UpdateAnimalTag(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestDeleteAnimalTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logging.SetLevel(logging.ERROR)

	tests := []struct {
		name           string
		tagID          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful tag deletion",
			tagID:          "1",
			expectedStatus: http.StatusOK,
			expectedBody:   "deleted successfully",
		},
		{
			name:           "deletion with non-existent tag still succeeds",
			tagID:          "999",
			expectedStatus: http.StatusOK,
			expectedBody:   "deleted successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/tags/"+tt.tagID, nil)
			c.Params = gin.Params{{Key: "tagId", Value: tt.tagID}}
			c.Set("logger", logging.GetDefaultLogger())

			// Execute
			handler := DeleteAnimalTag(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestAssignTagsToAnimal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(*gorm.DB)
		animalID       string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful tag assignment",
			setupData: func(db *gorm.DB) {
				animal := models.Animal{
					Name:    "Test Dog",
					Species: "Dog",
					Status:  "available",
					GroupID: 1,
				}
				db.Create(&animal)
			},
			animalID: "1",
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1, 2},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "not found when animal doesn't exist",
			setupData: func(db *gorm.DB) {},
			animalID:  "999",
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1},
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal not found",
		},
		{
			name: "bad request when tag_ids is missing",
			setupData: func(db *gorm.DB) {
				animal := models.Animal{
					Name:    "Test Dog",
					Species: "Dog",
					Status:  "available",
					GroupID: 1,
				}
				db.Create(&animal)
			},
			animalID:       "1",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			// Migrate Animal and Group models
			db.AutoMigrate(&models.Group{})
			group := models.Group{Name: "Test Group", Description: "Test"}
			db.Create(&group)

			tt.setupData(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/animals/"+tt.animalID+"/tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "animalId", Value: tt.animalID}}

			// Execute
			handler := AssignTagsToAnimal(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
