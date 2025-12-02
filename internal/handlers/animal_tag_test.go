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
	err = db.AutoMigrate(&models.AnimalTag{}, &models.Animal{}, &models.Group{}, &models.User{}, &models.UserGroup{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test group
	group := models.Group{Name: "Test Group", Description: "Test Description"}
	db.Create(&group)

	// Create test user (site admin)
	user := models.User{Username: "admin", Email: "admin@test.com", Password: "test", IsAdmin: true}
	db.Create(&user)

	// Create test group admin user
	groupAdmin := models.User{Username: "groupadmin", Email: "groupadmin@test.com", Password: "test", IsAdmin: false}
	db.Create(&groupAdmin)

	// Create user group relationship for group admin
	userGroup := models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true}
	db.Create(&userGroup)

	// Create test tags for the group
	tag1 := models.AnimalTag{GroupID: group.ID, Name: "friendly", Category: "behavior", Color: "#00FF00"}
	tag2 := models.AnimalTag{GroupID: group.ID, Name: "needs-walker", Category: "walker_status", Color: "#FF0000"}
	db.Create(&tag1)
	db.Create(&tag2)

	return db
}

func TestGetAnimalTags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		groupID        string
		userID         uint
		isAdmin        bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "successful retrieval of group tags by admin",
			groupID:        "1",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "successful retrieval of group tags by group admin",
			groupID:        "1",
			userID:         2,
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "access denied when user is not a member",
			groupID:        "1",
			userID:         999,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedCount:  0,
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
			c.Request = httptest.NewRequest("GET", "/groups/"+tt.groupID+"/animal-tags", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.groupID}}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

			// Execute
			handler := GetAnimalTags(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var tags []models.AnimalTag
				json.Unmarshal(w.Body.Bytes(), &tags)
				assert.Equal(t, tt.expectedCount, len(tags))
			}
		})
	}
}

func TestCreateAnimalTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up logger for tests
	logging.SetLevel(logging.ERROR)

	tests := []struct {
		name           string
		groupID        string
		userID         uint
		isAdmin        bool
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "successful tag creation by site admin",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "energetic",
				Category: "behavior",
				Color:    "#FFFF00",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "successful tag creation by group admin",
			groupID: "1",
			userID:  2,
			isAdmin: false,
			requestBody: AnimalTagRequest{
				Name:     "calm",
				Category: "behavior",
				Color:    "#00FFFF",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "forbidden when regular user tries to create tag",
			groupID: "1",
			userID:  999,
			isAdmin: false,
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
				Color:    "#FFFFFF",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only group admins can create tags",
		},
		{
			name:    "bad request when name is missing",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Category: "behavior",
				Color:    "#FFFF00",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "bad request when category is invalid",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "invalid_category",
				Color:    "#FFFF00",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "bad request when color is missing",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
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
			c.Request = httptest.NewRequest("POST", "/groups/"+tt.groupID+"/animal-tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.groupID}}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

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
		groupID        string
		tagID          string
		userID         uint
		isAdmin        bool
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "successful tag update by site admin",
			groupID: "1",
			tagID:   "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "very-friendly",
				Category: "behavior",
				Color:    "#00FFFF",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "successful tag update by group admin",
			groupID: "1",
			tagID:   "1",
			userID:  2,
			isAdmin: false,
			requestBody: AnimalTagRequest{
				Name:     "super-friendly",
				Category: "behavior",
				Color:    "#00FFFF",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "not found when tag doesn't exist in group",
			groupID: "1",
			tagID:   "999",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
				Color:    "#FFFFFF",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal tag not found in this group",
		},
		{
			name:    "forbidden when regular user tries to update",
			groupID: "1",
			tagID:   "1",
			userID:  999,
			isAdmin: false,
			requestBody: AnimalTagRequest{
				Name:     "test",
				Category: "behavior",
				Color:    "#FFFFFF",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only group admins can update tags",
		},
		{
			name:    "bad request when data is invalid",
			groupID: "1",
			tagID:   "1",
			userID:  1,
			isAdmin: true,
			requestBody: AnimalTagRequest{
				Name:     "",
				Category: "behavior",
				Color:    "#FFFFFF",
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
			c.Request = httptest.NewRequest("PUT", "/groups/"+tt.groupID+"/animal-tags/"+tt.tagID, bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{
				{Key: "id", Value: tt.groupID},
				{Key: "tagId", Value: tt.tagID},
			}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)
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
		groupID        string
		tagID          string
		userID         uint
		isAdmin        bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful tag deletion by site admin",
			groupID:        "1",
			tagID:          "1",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   "deleted successfully",
		},
		{
			name:           "successful tag deletion by group admin",
			groupID:        "1",
			tagID:          "1",
			userID:         2,
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			expectedBody:   "deleted successfully",
		},
		{
			name:           "not found when tag doesn't exist in group",
			groupID:        "1",
			tagID:          "999",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
		{
			name:           "forbidden when regular user tries to delete",
			groupID:        "1",
			tagID:          "1",
			userID:         999,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Only group admins can delete tags",
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
			c.Request = httptest.NewRequest("DELETE", "/groups/"+tt.groupID+"/animal-tags/"+tt.tagID, nil)
			c.Params = gin.Params{
				{Key: "id", Value: tt.groupID},
				{Key: "tagId", Value: tt.tagID},
			}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)
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
		groupID        string
		animalID       string
		userID         uint
		isAdmin        bool
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful tag assignment by site admin",
			setupData: func(db *gorm.DB) {
				animal := models.Animal{
					Name:    "Test Dog",
					Species: "Dog",
					Status:  "available",
					GroupID: 1,
				}
				db.Create(&animal)
			},
			groupID:  "1",
			animalID: "1",
			userID:   1,
			isAdmin:  true,
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1, 2},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful tag assignment by group admin",
			setupData: func(db *gorm.DB) {
				animal := models.Animal{
					Name:    "Test Dog",
					Species: "Dog",
					Status:  "available",
					GroupID: 1,
				}
				db.Create(&animal)
			},
			groupID:  "1",
			animalID: "1",
			userID:   2,
			isAdmin:  false,
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "not found when animal doesn't exist in group",
			setupData: func(db *gorm.DB) {},
			groupID:   "1",
			animalID:  "999",
			userID:    1,
			isAdmin:   true,
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1},
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal not found",
		},
		{
			name: "forbidden when regular user tries to assign",
			setupData: func(db *gorm.DB) {
				animal := models.Animal{
					Name:    "Test Dog",
					Species: "Dog",
					Status:  "available",
					GroupID: 1,
				}
				db.Create(&animal)
			},
			groupID:  "1",
			animalID: "1",
			userID:   999,
			isAdmin:  false,
			requestBody: map[string]interface{}{
				"tag_ids": []uint{1},
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only group admins can assign tags",
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
			groupID:        "1",
			animalID:       "1",
			userID:         1,
			isAdmin:        true,
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

			tt.setupData(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/groups/"+tt.groupID+"/animals/"+tt.animalID+"/tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{
				{Key: "id", Value: tt.groupID},
				{Key: "animalId", Value: tt.animalID},
			}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)
			c.Set("logger", logging.GetDefaultLogger())

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
