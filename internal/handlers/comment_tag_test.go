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

func setupCommentTagTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(&models.CommentTag{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test tags
	tag1 := models.CommentTag{Name: "urgent", Color: "#FF0000", IsSystem: false}
	tag2 := models.CommentTag{Name: "medical", Color: "#00FF00", IsSystem: true}
	db.Create(&tag1)
	db.Create(&tag2)

	return db
}

func TestGetCommentTags(t *testing.T) {
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
			db := setupCommentTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/comment-tags", nil)

			// Execute
			handler := GetCommentTags(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var tags []models.CommentTag
			json.Unmarshal(w.Body.Bytes(), &tags)
			assert.Equal(t, tt.expectedCount, len(tags))
		})
	}
}

func TestCreateCommentTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful tag creation with color",
			requestBody: CommentTagRequest{
				Name:  "important",
				Color: "#FFFF00",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "successful tag creation without color (uses default)",
			requestBody: CommentTagRequest{
				Name: "note",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "bad request when name is missing",
			requestBody: CommentTagRequest{
				Color: "#FFFF00",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupCommentTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/comment-tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// Execute
			handler := CreateCommentTag(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
			
			// Verify default color is applied
			if tt.name == "successful tag creation without color (uses default)" && w.Code == http.StatusCreated {
				var tag models.CommentTag
				json.Unmarshal(w.Body.Bytes(), &tag)
				assert.Equal(t, "#6b7280", tag.Color)
				assert.False(t, tag.IsSystem)
			}
		})
	}
}

func TestDeleteCommentTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		tagID          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful tag deletion",
			tagID:          "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "forbidden when trying to delete system tag",
			tagID:          "2",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Cannot delete system tags",
		},
		{
			name:           "not found when tag doesn't exist",
			tagID:          "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Tag not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupCommentTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/comment-tags/"+tt.tagID, nil)
			c.Params = gin.Params{{Key: "tagId", Value: tt.tagID}}

			// Execute
			handler := DeleteCommentTag(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
