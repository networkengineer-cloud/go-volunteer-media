package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupGroupTestDB creates an in-memory SQLite database for group testing
func setupGroupTestDB(t *testing.T) *gorm.DB {
	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createGroupTestUser creates a user for testing
func createGroupTestUser(t *testing.T, db *gorm.DB, username, email string, isAdmin bool) *models.User {
	hashedPassword, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		IsAdmin:  isAdmin,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	return user
}

// createTestGroup creates a group in the database
func createTestGroup(t *testing.T, db *gorm.DB, name, description string) *models.Group {
	group := &models.Group{
		Name:         name,
		Description:  description,
		ImageURL:     "/test-image.jpg",
		HeroImageURL: "/test-hero.jpg",
		HasProtocols: false,
	}

	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	return group
}

// setupGroupTestContext creates a Gin context with authenticated user
func setupGroupTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	return c, w
}

// TestGetGroups tests retrieving groups with different user permissions
func TestGetGroups(t *testing.T) {
	tests := []struct {
		name           string
		isAdmin        bool
		setupFunc      func(*gorm.DB, *models.User) []*models.Group
		expectedStatus int
		expectedCount  int
	}{
		{
			name:    "admin sees all groups",
			isAdmin: true,
			setupFunc: func(db *gorm.DB, user *models.User) []*models.Group {
				group1 := createTestGroup(t, db, "Group 1", "Description 1")
				group2 := createTestGroup(t, db, "Group 2", "Description 2")
				group3 := createTestGroup(t, db, "Group 3", "Description 3")
				return []*models.Group{group1, group2, group3}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:    "regular user sees only their groups",
			isAdmin: false,
			setupFunc: func(db *gorm.DB, user *models.User) []*models.Group {
				group1 := createTestGroup(t, db, "User Group", "User's group")
				group2 := createTestGroup(t, db, "Other Group", "Other group")

				// Associate user with only group1
				db.Model(user).Association("Groups").Append(group1)

				return []*models.Group{group1, group2}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1, // Only sees their own group
		},
		{
			name:    "regular user with no groups",
			isAdmin: false,
			setupFunc: func(db *gorm.DB, user *models.User) []*models.Group {
				createTestGroup(t, db, "Some Group", "Description")
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			user := createGroupTestUser(t, db, "testuser", "test@example.com", tt.isAdmin)

			if tt.setupFunc != nil {
				tt.setupFunc(db, user)
			}

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Request = httptest.NewRequest("GET", "/api/v1/groups", nil)

			handler := GetGroups(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var groups []models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &groups); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(groups) != tt.expectedCount {
					t.Errorf("Expected %d groups, got %d", tt.expectedCount, len(groups))
				}
			}
		})
	}
}

// TestGetGroupsErrorPaths tests error handling in GetGroups
func TestGetGroupsErrorPaths(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing user_id context",
			setupContext: func(c *gin.Context) {
				// Don't set user_id
				c.Set("is_admin", false)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "User context not found",
		},
		{
			name: "missing is_admin context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				// Don't set is_admin
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Admin context not found",
		},
		{
			name: "invalid admin flag type",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", "not_a_bool") // Invalid type
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Invalid admin flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/groups", nil)

			tt.setupContext(c)

			handler := GetGroups(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if errorMsg, ok := response["error"].(string); ok {
				if errorMsg != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorMsg)
				}
			} else {
				t.Error("Expected error in response")
			}
		})
	}
}

// TestGetGroup tests retrieving a single group with access control
func TestGetGroup(t *testing.T) {
	tests := []struct {
		name           string
		isAdmin        bool
		setupFunc      func(*gorm.DB, *models.User) (uint, bool)
		expectedStatus int
	}{
		{
			name:    "admin can access any group",
			isAdmin: true,
			setupFunc: func(db *gorm.DB, user *models.User) (uint, bool) {
				group := createTestGroup(t, db, "Admin Test Group", "Description")
				return group.ID, true
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "regular user can access their group",
			isAdmin: false,
			setupFunc: func(db *gorm.DB, user *models.User) (uint, bool) {
				group := createTestGroup(t, db, "User Group", "User's group")
				db.Model(user).Association("Groups").Append(group)
				return group.ID, true
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "regular user cannot access other groups",
			isAdmin: false,
			setupFunc: func(db *gorm.DB, user *models.User) (uint, bool) {
				group := createTestGroup(t, db, "Other Group", "Not user's group")
				// Don't associate with user
				return group.ID, true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:    "non-existent group returns 404",
			isAdmin: true,
			setupFunc: func(db *gorm.DB, user *models.User) (uint, bool) {
				return 99999, false
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			user := createGroupTestUser(t, db, "testuser", "test@example.com", tt.isAdmin)

			groupID, _ := tt.setupFunc(db, user)

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", groupID)}}
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d", groupID), nil)

			handler := GetGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var group models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if group.ID != groupID {
					t.Errorf("Expected group ID %d, got %d", groupID, group.ID)
				}
			}
		})
	}
}

// TestCreateGroup tests creating new groups (admin only)
func TestCreateGroup(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, *httptest.ResponseRecorder)
	}{
		{
			name: "create group with all fields",
			payload: map[string]interface{}{
				"name":           "New Group",
				"description":    "Test description",
				"image_url":      "/images/test.jpg",
				"hero_image_url": "/images/hero.jpg",
				"has_protocols":  true,
			},
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var group models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if group.Name != "New Group" {
					t.Errorf("Expected name 'New Group', got '%s'", group.Name)
				}
				if group.Description != "Test description" {
					t.Errorf("Expected description 'Test description', got '%s'", group.Description)
				}
				if !group.HasProtocols {
					t.Error("Expected has_protocols to be true")
				}

				// Verify it was saved to database
				var dbGroup models.Group
				if err := db.First(&dbGroup, group.ID).Error; err != nil {
					t.Errorf("Group not found in database: %v", err)
				}
			},
		},
		{
			name: "create group with minimal fields (defaults applied)",
			payload: map[string]interface{}{
				"name": "Minimal Group",
			},
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var group models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if group.HeroImageURL != "/default-hero.svg" {
					t.Errorf("Expected default hero image, got '%s'", group.HeroImageURL)
				}
			},
		},
		{
			name: "validation error - missing required name",
			payload: map[string]interface{}{
				"description": "Missing name",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - name too short",
			payload: map[string]interface{}{
				"name": "A", // Less than 2 chars
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - name too long",
			payload: map[string]interface{}{
				"name": string(make([]byte, 101)), // More than 100 chars
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - description too long",
			payload: map[string]interface{}{
				"name":        "Valid Name",
				"description": string(make([]byte, 501)), // More than 500 chars
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - invalid GroupMe bot id (too short)",
			payload: map[string]interface{}{
				"name":           "GroupMe Invalid",
				"groupme_bot_id": "1234abcd",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["error"] == nil || !strings.Contains(resp["error"].(string), "Invalid GroupMe bot ID") {
					t.Errorf("Expected GroupMe bot ID validation error, got: %v", resp["error"])
				}
			},
		},
		{
			name: "validation error - invalid GroupMe bot id (non-hex)",
			payload: map[string]interface{}{
				"name":           "GroupMe Invalid",
				"groupme_bot_id": "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["error"] == nil || !strings.Contains(resp["error"].(string), "Invalid GroupMe bot ID") {
					t.Errorf("Expected GroupMe bot ID validation error, got: %v", resp["error"])
				}
			},
		},
		{
			name: "accepts valid GroupMe bot id",
			payload: map[string]interface{}{
				"name":           "GroupMe Valid",
				"groupme_bot_id": "abcdef0123456789abcdef0123",
			},
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var group models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if group.GroupMeBotID != "abcdef0123456789abcdef0123" {
					t.Errorf("Expected GroupMeBotID to be set, got '%s'", group.GroupMeBotID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			user := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			c, w := setupGroupTestContext(user.ID, true)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/groups", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := CreateGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, w)
			}
		})
	}
}

// TestUpdateGroup tests updating existing groups (admin only)
func TestUpdateGroup(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		payload        map[string]interface{}
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "update all fields",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "Original Name", "Original Description")
				return group.ID
			},
			payload: map[string]interface{}{
				"name":           "Updated Name",
				"description":    "Updated Description",
				"image_url":      "/updated-image.jpg",
				"hero_image_url": "/updated-hero.jpg",
				"has_protocols":  true,
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, groupID uint) {
				var group models.Group
				if err := db.First(&group, groupID).Error; err != nil {
					t.Fatalf("Failed to find updated group: %v", err)
				}

				if group.Name != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got '%s'", group.Name)
				}
				if group.Description != "Updated Description" {
					t.Errorf("Expected description 'Updated Description', got '%s'", group.Description)
				}
				if !group.HasProtocols {
					t.Error("Expected has_protocols to be true")
				}
			},
		},
		{
			name: "update non-existent group",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			payload: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name: "validation error on update",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "Test Group", "Description")
				return group.ID
			},
			payload: map[string]interface{}{
				"name": "A", // Too short
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - invalid GroupMe bot id (too short)",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "GroupMe Update", "desc")
				return group.ID
			},
			payload: map[string]interface{}{
				"name":           "GroupMe Update",
				"groupme_bot_id": "1234abcd",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - invalid GroupMe bot id (non-hex)",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "GroupMe Update", "desc")
				return group.ID
			},
			payload: map[string]interface{}{
				"name":           "GroupMe Update",
				"groupme_bot_id": "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "accepts valid GroupMe bot id",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "GroupMe Update", "desc")
				return group.ID
			},
			payload: map[string]interface{}{
				"name":           "GroupMe Update",
				"groupme_bot_id": "abcdef0123456789abcdef0123",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, groupID uint) {
				var group models.Group
				if err := db.First(&group, groupID).Error; err != nil {
					t.Fatalf("Failed to find updated group: %v", err)
				}
				if group.GroupMeBotID != "abcdef0123456789abcdef0123" {
					t.Errorf("Expected GroupMeBotID to be set, got '%s'", group.GroupMeBotID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			user := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, true)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", groupID)}}

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d", groupID), bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, groupID)
			}
		})
	}
}

// TestDeleteGroup tests deleting groups (admin only)
func TestDeleteGroup(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		expectedStatus int
		shouldExist    bool
	}{
		{
			name: "delete existing group",
			setupFunc: func(db *gorm.DB) uint {
				group := createTestGroup(t, db, "Group to Delete", "Will be deleted")
				return group.ID
			},
			expectedStatus: http.StatusOK,
			shouldExist:    false,
		},
		{
			name: "delete non-existent group (idempotent)",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			expectedStatus: http.StatusOK, // GORM Delete is idempotent
			shouldExist:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			user := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, true)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", groupID)}}
			c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/groups/%d", groupID), nil)

			handler := DeleteGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			// Verify deletion
			var group models.Group
			err := db.First(&group, groupID).Error
			if tt.shouldExist && err != nil {
				t.Error("Expected group to exist but it doesn't")
			}
			if !tt.shouldExist && err == nil {
				t.Error("Expected group to be deleted but it still exists")
			}
		})
	}
}

// TestAddUserToGroup tests adding users to groups (admin only)
func TestAddUserToGroup(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (uint, uint) // Returns userID, groupID
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint, uint)
	}{
		{
			name: "successfully add user to group",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "regularuser", "user@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				return user.ID, group.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID, groupID uint) {
				var user models.User
				if err := db.Preload("Groups").First(&user, userID).Error; err != nil {
					t.Fatalf("Failed to load user: %v", err)
				}

				found := false
				for _, group := range user.Groups {
					if group.ID == groupID {
						found = true
						break
					}
				}
				if !found {
					t.Error("User was not added to group")
				}
			},
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				group := createTestGroup(t, db, "Test Group", "Description")
				return 99999, group.ID
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name: "group not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "regularuser", "user@example.com", false)
				return user.ID, 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name: "invalid user ID",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				return 0, 0 // Will cause parsing error
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			admin := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			userID, groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(admin.ID, true)

			// Handle invalid ID test case specially
			if tt.name == "invalid user ID" {
				c.Params = gin.Params{
					{Key: "userId", Value: "invalid"},
					{Key: "groupId", Value: "1"},
				}
			} else {
				c.Params = gin.Params{
					{Key: "userId", Value: fmt.Sprintf("%d", userID)},
					{Key: "groupId", Value: fmt.Sprintf("%d", groupID)},
				}
			}

			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/groups/%d", userID, groupID), nil)

			handler := AddUserToGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID, groupID)
			}
		})
	}
}

// TestRemoveUserFromGroup tests removing users from groups (admin only)
func TestRemoveUserFromGroup(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (uint, uint) // Returns userID, groupID
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint, uint)
	}{
		{
			name: "successfully remove user from group",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "regularuser", "user@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")

				// Add user to group first
				db.Model(&user).Association("Groups").Append(group)

				return user.ID, group.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID, groupID uint) {
				var user models.User
				if err := db.Preload("Groups").First(&user, userID).Error; err != nil {
					t.Fatalf("Failed to load user: %v", err)
				}

				for _, group := range user.Groups {
					if group.ID == groupID {
						t.Error("User was not removed from group")
						break
					}
				}
			},
		},
		{
			name: "remove user from group they're not in (idempotent)",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "regularuser", "user@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Don't add user to group
				return user.ID, group.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc:      nil,
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				group := createTestGroup(t, db, "Test Group", "Description")
				return 99999, group.ID
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name: "group not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "regularuser", "user@example.com", false)
				return user.ID, 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			admin := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			userID, groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(admin.ID, true)
			c.Params = gin.Params{
				{Key: "userId", Value: fmt.Sprintf("%d", userID)},
				{Key: "groupId", Value: fmt.Sprintf("%d", groupID)},
			}
			c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d/groups/%d", userID, groupID), nil)

			handler := RemoveUserFromGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID, groupID)
			}
		})
	}
}

// Unit tests for isValidGroupMeBotID
func TestIsValidGroupMeBotID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"empty is valid", "", true},
		{"valid lowercase hex", "0123456789abcdef0123456789", true},
		{"valid uppercase hex", "0123456789ABCDEF0123456789", true},
		{"valid mixed case", "0123456789aBcDeF0123456789", true},
		{"too short", "0123456789abcdef", false},
		{"too long", "0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"non-hex char", "0123456789abcdef012345678g", false},
		{"special chars", "0123456789abcdef012345678!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidGroupMeBotID(tt.id); got != tt.want {
				t.Errorf("isValidGroupMeBotID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// TestPromoteGroupAdmin tests promoting a user to group admin
func TestPromoteGroupAdmin(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (uint, uint) // returns userID, groupID
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successfully promote user to group admin",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "member", "member@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Add user as regular member (not group admin)
				userGroup := &models.UserGroup{
					UserID:       user.ID,
					GroupID:      group.ID,
					IsGroupAdmin: false,
				}
				db.Create(userGroup)
				return user.ID, group.ID
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "promoted to group admin",
		},
		{
			name: "user not a member of group",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "nonmember", "nonmember@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Don't add user to group
				return user.ID, group.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "not a member",
		},
		{
			name: "user already a group admin",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "admin_member", "admin_member@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				userGroup := &models.UserGroup{
					UserID:       user.ID,
					GroupID:      group.ID,
					IsGroupAdmin: true,
				}
				db.Create(userGroup)
				return user.ID, group.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "already a group admin",
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				group := createTestGroup(t, db, "Test Group", "Description")
				return 99999, group.ID
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name: "group not found",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "member", "member@example.com", false)
				return user.ID, 99999
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			admin := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			userID, groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(admin.ID, true)
			c.Params = gin.Params{
				{Key: "userId", Value: fmt.Sprintf("%d", userID)},
				{Key: "id", Value: fmt.Sprintf("%d", groupID)},
			}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/admins/%d", groupID, userID), nil)

			handler := PromoteGroupAdmin(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestDemoteGroupAdmin tests demoting a user from group admin
func TestDemoteGroupAdmin(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (uint, uint) // returns userID, groupID
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successfully demote group admin",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "groupadmin", "groupadmin@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				userGroup := &models.UserGroup{
					UserID:       user.ID,
					GroupID:      group.ID,
					IsGroupAdmin: true,
				}
				db.Create(userGroup)
				return user.ID, group.ID
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "demoted from group admin",
		},
		{
			name: "user not a group admin",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "member", "member@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				userGroup := &models.UserGroup{
					UserID:       user.ID,
					GroupID:      group.ID,
					IsGroupAdmin: false,
				}
				db.Create(userGroup)
				return user.ID, group.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "not a group admin",
		},
		{
			name: "user not a member of group",
			setupFunc: func(db *gorm.DB) (uint, uint) {
				user := createGroupTestUser(t, db, "nonmember", "nonmember@example.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				return user.ID, group.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "not a member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			admin := createGroupTestUser(t, db, "admin", "admin@example.com", true)

			userID, groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(admin.ID, true)
			c.Params = gin.Params{
				{Key: "userId", Value: fmt.Sprintf("%d", userID)},
				{Key: "id", Value: fmt.Sprintf("%d", groupID)},
			}
			c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/groups/%d/admins/%d", groupID, userID), nil)

			handler := DemoteGroupAdmin(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestGetGroupMembers tests retrieving group members with admin status
func TestGetGroupMembers(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, uint) // returns requesting user and groupID
		expectedStatus int
		checkFunc      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "site admin can view members",
			setupFunc: func(db *gorm.DB) (*models.User, uint) {
				admin := createGroupTestUser(t, db, "admin", "admin@example.com", true)
				group := createTestGroup(t, db, "Test Group", "Description")
				member := createGroupTestUser(t, db, "member", "member@example.com", false)
				userGroup := &models.UserGroup{
					UserID:       member.ID,
					GroupID:      group.ID,
					IsGroupAdmin: false,
				}
				db.Create(userGroup)
				return admin, group.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var members []map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &members); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(members) != 1 {
					t.Errorf("Expected 1 member, got %d", len(members))
				}
			},
		},
		{
			name: "group member can view members",
			setupFunc: func(db *gorm.DB) (*models.User, uint) {
				group := createTestGroup(t, db, "Test Group", "Description")
				member := createGroupTestUser(t, db, "member", "member@example.com", false)
				userGroup := &models.UserGroup{
					UserID:       member.ID,
					GroupID:      group.ID,
					IsGroupAdmin: false,
				}
				db.Create(userGroup)
				return member, group.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc:      nil,
		},
		{
			name: "non-member cannot view members",
			setupFunc: func(db *gorm.DB) (*models.User, uint) {
				group := createTestGroup(t, db, "Test Group", "Description")
				nonmember := createGroupTestUser(t, db, "nonmember", "nonmember@example.com", false)
				return nonmember, group.ID
			},
			expectedStatus: http.StatusForbidden,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)

			user, groupID := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, user.IsAdmin)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", groupID)},
			}
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/groups/%d/members", groupID), nil)

			handler := GetGroupMembers(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, w)
			}
		})
	}
}

// TestIsGroupAdmin tests the IsGroupAdmin helper function
func TestIsGroupAdmin(t *testing.T) {
	db := setupGroupTestDB(t)
	
	// Create a user and group
	user := createGroupTestUser(t, db, "member", "member@example.com", false)
	group := createTestGroup(t, db, "Test Group", "Description")
	
	// Initially user is not a group admin
	if IsGroupAdmin(db, user.ID, group.ID) {
		t.Error("Expected user to not be a group admin initially")
	}
	
	// Add user as regular member
	userGroup := &models.UserGroup{
		UserID:       user.ID,
		GroupID:      group.ID,
		IsGroupAdmin: false,
	}
	db.Create(userGroup)
	
	// Still not a group admin
	if IsGroupAdmin(db, user.ID, group.ID) {
		t.Error("Expected user to not be a group admin")
	}
	
	// Promote to group admin
	db.Model(userGroup).Update("is_group_admin", true)
	
	// Now should be a group admin
	if !IsGroupAdmin(db, user.ID, group.ID) {
		t.Error("Expected user to be a group admin")
	}
}

// TestAddMemberToGroup tests the AddMemberToGroup handler
func TestAddMemberToGroup(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, *models.User, *models.Group)
		contextUserID  uint
		isAdmin        bool
		groupID        string
		targetUserID   string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "site admin can add user to group",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				admin := createGroupTestUser(t, db, "admin", "admin@test.com", true)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				return admin, user, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "group admin can add user to their group",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := createGroupTestUser(t, db, "groupadmin", "groupadmin@test.com", false)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Make groupAdmin a group admin
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				return groupAdmin, user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular user cannot add members",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				regular := createGroupTestUser(t, db, "regular", "regular@test.com", false)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Regular user is a member but not an admin
				db.Create(&models.UserGroup{UserID: regular.ID, GroupID: group.ID, IsGroupAdmin: false})
				return regular, user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name: "cannot add user already in group",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				admin := createGroupTestUser(t, db, "admin", "admin@test.com", true)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// User already in group
				db.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID})
				return admin, user, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "already a member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			contextUser, targetUser, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/groups/%d/members/%d", group.ID, targetUser.ID), nil)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "userId", Value: fmt.Sprintf("%d", targetUser.ID)},
			}

			handler := AddMemberToGroup(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
		})
	}
}

// TestPromoteMemberToGroupAdmin tests the PromoteMemberToGroupAdmin handler
func TestPromoteMemberToGroupAdmin(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, *models.User, *models.Group)
		contextUserID  uint
		isAdmin        bool
		expectedStatus int
		expectedError  string
	}{
		{
			name: "site admin can promote member",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				admin := createGroupTestUser(t, db, "admin", "admin@test.com", true)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// User is a regular member
				db.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID, IsGroupAdmin: false})
				return admin, user, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "group admin can promote member in their group",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := createGroupTestUser(t, db, "groupadmin", "groupadmin@test.com", false)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Make groupAdmin a group admin
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				// User is a regular member
				db.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID, IsGroupAdmin: false})
				return groupAdmin, user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular member cannot promote",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				regular := createGroupTestUser(t, db, "regular", "regular@test.com", false)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// Regular user is a member but not an admin
				db.Create(&models.UserGroup{UserID: regular.ID, GroupID: group.ID, IsGroupAdmin: false})
				db.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID, IsGroupAdmin: false})
				return regular, user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name: "cannot promote non-member",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				admin := createGroupTestUser(t, db, "admin", "admin@test.com", true)
				user := createGroupTestUser(t, db, "user", "user@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				// User is NOT a member
				return admin, user, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "not a member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			contextUser, targetUser, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/groups/%d/members/%d/promote", group.ID, targetUser.ID), nil)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "userId", Value: fmt.Sprintf("%d", targetUser.ID)},
			}

			handler := PromoteMemberToGroupAdmin(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
		})
	}
}

// TestUpdateGroupSettings tests the UpdateGroupSettings handler
func TestUpdateGroupSettings(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, *models.Group)
		isAdmin        bool
		expectedStatus int
		expectedError  string
	}{
		{
			name: "site admin can update any group settings",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				admin := createGroupTestUser(t, db, "admin", "admin@test.com", true)
				group := createTestGroup(t, db, "Test Group", "Description")
				return admin, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "group admin can update their group settings",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := createGroupTestUser(t, db, "groupadmin", "groupadmin@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				return groupAdmin, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular member cannot update settings",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				regular := createGroupTestUser(t, db, "regular", "regular@test.com", false)
				group := createTestGroup(t, db, "Test Group", "Description")
				db.Create(&models.UserGroup{UserID: regular.ID, GroupID: group.ID, IsGroupAdmin: false})
				return regular, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupTestDB(t)
			contextUser, group := tt.setupFunc(db)

			reqBody := GroupRequest{
				Name:         "Updated Name",
				Description:  "Updated Description",
				HasProtocols: true,
			}
			jsonBody, _ := json.Marshal(reqBody)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/settings", group.ID), bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
			}

			handler := UpdateGroupSettings(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
		})
	}
}
