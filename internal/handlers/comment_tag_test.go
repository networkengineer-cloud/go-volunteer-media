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
	err = db.AutoMigrate(&models.CommentTag{}, &models.Group{}, &models.User{}, &models.UserGroup{})
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
	tag1 := models.CommentTag{GroupID: group.ID, Name: "urgent", Color: "#FF0000", IsSystem: false}
	tag2 := models.CommentTag{GroupID: group.ID, Name: "medical", Color: "#00FF00", IsSystem: true}
	db.Create(&tag1)
	db.Create(&tag2)

	return db
}

func TestGetCommentTags(t *testing.T) {
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
			db := setupCommentTagTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/"+tt.groupID+"/comment-tags", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.groupID}}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

			// Execute
			handler := GetCommentTags(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var tags []models.CommentTag
				json.Unmarshal(w.Body.Bytes(), &tags)
				assert.Equal(t, tt.expectedCount, len(tags))
			}
		})
	}
}

func TestCreateCommentTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
			name:    "successful tag creation with color by site admin",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: CommentTagRequest{
				Name:  "important",
				Color: "#FFFF00",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "successful tag creation by group admin",
			groupID: "1",
			userID:  2,
			isAdmin: false,
			requestBody: CommentTagRequest{
				Name:  "note",
				Color: "#00FFFF",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "successful tag creation without color (uses default)",
			groupID: "1",
			userID:  1,
			isAdmin: true,
			requestBody: CommentTagRequest{
				Name: "reminder",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "forbidden when regular user tries to create tag",
			groupID: "1",
			userID:  999,
			isAdmin: false,
			requestBody: CommentTagRequest{
				Name:  "test",
				Color: "#FFFFFF",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only group admins can create tags",
		},
		{
			name:    "bad request when name is missing",
			groupID: "1",
			userID:  1,
			isAdmin: true,
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
			c.Request = httptest.NewRequest("POST", "/groups/"+tt.groupID+"/comment-tags", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.groupID}}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

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
		groupID        string
		tagID          string
		userID         uint
		isAdmin        bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful tag deletion by site admin",
			groupID:        "1",
			tagID:          "1",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "successful tag deletion by group admin",
			groupID:        "1",
			tagID:          "1",
			userID:         2,
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "forbidden when trying to delete system tag",
			groupID:        "1",
			tagID:          "2",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Cannot delete system tags",
		},
		{
			name:           "not found when tag doesn't exist in group",
			groupID:        "1",
			tagID:          "999",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Tag not found",
		},
		{
			name:           "forbidden when regular user tries to delete",
			groupID:        "1",
			tagID:          "1",
			userID:         999,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only group admins can delete tags",
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
			c.Request = httptest.NewRequest("DELETE", "/groups/"+tt.groupID+"/comment-tags/"+tt.tagID, nil)
			c.Params = gin.Params{
				{Key: "id", Value: tt.groupID},
				{Key: "tagId", Value: tt.tagID},
			}
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

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
