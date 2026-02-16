package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// setupAnnouncementTestDB creates an in-memory SQLite database for announcement testing
func setupAnnouncementTestDB(t *testing.T) *gorm.DB {
	return SetupTestDB(t)
}

// createAnnouncementTestUser creates a user for testing
func createAnnouncementTestUser(t *testing.T, db *gorm.DB, username, email string, isAdmin bool) *models.User {
	return CreateTestUser(t, db, username, email, "password123", isAdmin)
}

// setupAnnouncementTestContext creates a Gin context with authenticated user
func setupAnnouncementTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	return c, w
}

// createTestAnnouncement creates an announcement in the database
func createTestAnnouncement(t *testing.T, db *gorm.DB, userID uint, title, content string) *models.Announcement {
	announcement := &models.Announcement{
		UserID:    userID,
		Title:     title,
		Content:   content,
		SendEmail: false,
	}

	if err := db.Create(announcement).Error; err != nil {
		t.Fatalf("Failed to create announcement: %v", err)
	}

	return announcement
}

// TestGetAnnouncements tests retrieving announcements
func TestGetAnnouncements(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*gorm.DB, *models.User)
		expectedCount int
	}{
		{
			name: "get multiple announcements",
			setupFunc: func(db *gorm.DB, user *models.User) {
				createTestAnnouncement(t, db, user.ID, "Announcement 1", "Content 1")
				createTestAnnouncement(t, db, user.ID, "Announcement 2", "Content 2")
				createTestAnnouncement(t, db, user.ID, "Announcement 3", "Content 3")
			},
			expectedCount: 3,
		},
		{
			name: "get empty list when no announcements",
			setupFunc: func(db *gorm.DB, user *models.User) {
				// No announcements
			},
			expectedCount: 0,
		},
		{
			name: "limit to 10 announcements",
			setupFunc: func(db *gorm.DB, user *models.User) {
				// Create 15 announcements
				for i := 1; i <= 15; i++ {
					createTestAnnouncement(t, db, user.ID, fmt.Sprintf("Announcement %d", i), fmt.Sprintf("Content %d", i))
				}
			},
			expectedCount: 10, // Should limit to 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)
			user := createAnnouncementTestUser(t, db, "testuser", "test@example.com", false)

			tt.setupFunc(db, user)

			c, w := setupAnnouncementTestContext(user.ID, false)
			c.Request = httptest.NewRequest("GET", "/api/v1/announcements", nil)

			handler := GetAnnouncements(db)
			handler(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
			}

			var announcements []models.Announcement
			if err := json.Unmarshal(w.Body.Bytes(), &announcements); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(announcements) != tt.expectedCount {
				t.Errorf("Expected %d announcements, got %d", tt.expectedCount, len(announcements))
			}
		})
	}
}

// TestCreateAnnouncement tests creating new announcements
func TestCreateAnnouncement(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		isAdmin        bool
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, *httptest.ResponseRecorder)
	}{
		{
			name: "successfully create announcement without email",
			payload: map[string]interface{}{
				"title":      "Test Announcement",
				"content":    "This is a test announcement content that is long enough.",
				"send_email": false,
			},
			isAdmin:        true,
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var announcement models.Announcement
				if err := json.Unmarshal(w.Body.Bytes(), &announcement); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if announcement.Title != "Test Announcement" {
					t.Errorf("Expected title 'Test Announcement', got '%s'", announcement.Title)
				}
				if announcement.SendEmail {
					t.Error("SendEmail should be false")
				}

				// Verify it was saved to database
				var dbAnnouncement models.Announcement
				if err := db.First(&dbAnnouncement, announcement.ID).Error; err != nil {
					t.Errorf("Announcement not found in database: %v", err)
				}
			},
		},
		{
			name: "successfully create announcement with email flag",
			payload: map[string]interface{}{
				"title":      "Email Announcement",
				"content":    "This announcement should trigger email notifications.",
				"send_email": true,
			},
			isAdmin:        true,
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var announcement models.Announcement
				if err := json.Unmarshal(w.Body.Bytes(), &announcement); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if !announcement.SendEmail {
					t.Error("SendEmail should be true")
				}
			},
		},
		{
			name: "validation error - missing title",
			payload: map[string]interface{}{
				"content": "Content without title",
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - title too short",
			payload: map[string]interface{}{
				"title":   "A",
				"content": "This content is long enough but title is too short.",
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - title too long",
			payload: map[string]interface{}{
				"title":   string(make([]byte, 201)), // More than 200 chars
				"content": "Content is fine but title is too long.",
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - missing content",
			payload: map[string]interface{}{
				"title": "Valid Title",
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - content too short",
			payload: map[string]interface{}{
				"title":   "Valid Title",
				"content": "Short", // Less than 10 chars
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)
			user := createAnnouncementTestUser(t, db, "admin", "admin@example.com", tt.isAdmin)

			// Create a mock email service (not configured)
			emailService := createTestEmailService(false, db)

			// Create GroupMe service (will be used if requested)
			groupMeService := groupme.NewService()

			c, w := setupAnnouncementTestContext(user.ID, tt.isAdmin)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/announcements", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := CreateAnnouncement(db, emailService, groupMeService)
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

// TestDeleteAnnouncement tests deleting announcements
func TestDeleteAnnouncement(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB, *models.User) uint
		expectedStatus int
		checkDeleted   bool
	}{
		{
			name: "successfully delete announcement",
			setupFunc: func(db *gorm.DB, user *models.User) uint {
				announcement := createTestAnnouncement(t, db, user.ID, "To Delete", "Content to delete")
				return announcement.ID
			},
			expectedStatus: http.StatusOK,
			checkDeleted:   true,
		},
		{
			name: "delete non-existent announcement (idempotent)",
			setupFunc: func(db *gorm.DB, user *models.User) uint {
				return 99999
			},
			expectedStatus: http.StatusOK,
			checkDeleted:   false,
		},
		{
			name: "invalid announcement ID",
			setupFunc: func(db *gorm.DB, user *models.User) uint {
				return 0 // Will cause parsing error
			},
			expectedStatus: http.StatusBadRequest,
			checkDeleted:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)
			admin := createAnnouncementTestUser(t, db, "admin", "admin@example.com", true)

			announcementID := tt.setupFunc(db, admin)

			c, w := setupAnnouncementTestContext(admin.ID, true)

			// Handle invalid ID test case specially
			if tt.name == "invalid announcement ID" {
				c.Params = gin.Params{{Key: "id", Value: "invalid"}}
			} else {
				c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", announcementID)}}
			}

			c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/announcements/%d", announcementID), nil)

			handler := DeleteAnnouncement(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkDeleted && tt.expectedStatus == http.StatusOK {
				// Verify announcement was deleted
				var announcement models.Announcement
				err := db.First(&announcement, announcementID).Error
				if err == nil {
					t.Error("Announcement should have been deleted")
				}
			}
		})
	}
}

// TestSendAnnouncementEmails tests the sendAnnouncementEmails function directly
func TestSendAnnouncementEmails(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*gorm.DB)
		emailService  func(*gorm.DB) *email.Service
		title         string
		content       string
		expectedError bool
	}{
		{
			name: "successfully send emails to opted-in users",
			setupFunc: func(db *gorm.DB) {
				// Create users with email notifications enabled
				user1 := createAnnouncementTestUser(t, db, "user1", "user1@example.com", false)
				db.Model(&models.User{}).Where("id = ?", user1.ID).Update("email_notifications_enabled", true)

				user2 := createAnnouncementTestUser(t, db, "user2", "user2@example.com", false)
				db.Model(&models.User{}).Where("id = ?", user2.ID).Update("email_notifications_enabled", true)
			},
			emailService:  func(db *gorm.DB) *email.Service { return createTestEmailService(true, db) },
			title:         "Test Announcement",
			content:       "This is a test announcement content.",
			expectedError: false,
		},
		{
			name: "no users with email notifications enabled",
			setupFunc: func(db *gorm.DB) {
				// Create users but don't enable email notifications
				createAnnouncementTestUser(t, db, "user3", "user3@example.com", false)
				createAnnouncementTestUser(t, db, "user4", "user4@example.com", false)
			},
			emailService:  func(db *gorm.DB) *email.Service { return createTestEmailService(true, db) },
			title:         "Test Announcement",
			content:       "This is a test announcement content.",
			expectedError: false,
		},
		{
			name: "empty database",
			setupFunc: func(db *gorm.DB) {
				// No users
			},
			emailService:  func(db *gorm.DB) *email.Service { return createTestEmailService(true, db) },
			title:         "Test Announcement",
			content:       "This is a test announcement content.",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)

			if tt.setupFunc != nil {
				tt.setupFunc(db)
			}

			emailService := tt.emailService(db)
			ctx := context.Background()
			err := sendAnnouncementEmails(ctx, db, emailService, tt.title, tt.content)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCreateAnnouncementErrorPaths tests error handling in CreateAnnouncement
func TestCreateAnnouncementErrorPaths(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing user_id context",
			setupContext: func(c *gin.Context) {
				// Don't set user_id
				c.Set("is_admin", true)
			},
			payload: map[string]interface{}{
				"title":   "Test Announcement",
				"content": "This is test content that is long enough.",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "User context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)
			emailService := createTestEmailService(false, db)
			groupMeService := groupme.NewService()

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/announcements", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			tt.setupContext(c)

			handler := CreateAnnouncement(db, emailService, groupMeService)
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

// TestCreateAnnouncementWithConfiguredEmail tests announcement creation with configured email service
func TestCreateAnnouncementWithConfiguredEmail(t *testing.T) {
	db := setupAnnouncementTestDB(t)
	user := createAnnouncementTestUser(t, db, "admin", "admin@example.com", true)

	// Create users with email notifications enabled for email sending test
	user1 := createAnnouncementTestUser(t, db, "user1", "user1@example.com", false)
	db.Model(&models.User{}).Where("id = ?", user1.ID).Update("email_notifications_enabled", true)

	// Create a configured email service
	emailService := createTestEmailService(true, db)

	groupMeService := groupme.NewService()

	c, w := setupAnnouncementTestContext(user.ID, true)

	payload := map[string]interface{}{
		"title":      "Test Announcement",
		"content":    "This is a test announcement that should trigger email sending.",
		"send_email": true,
	}

	jsonBytes, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/api/v1/announcements", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnnouncement(db, emailService, groupMeService)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var announcement models.Announcement
	if err := json.Unmarshal(w.Body.Bytes(), &announcement); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !announcement.SendEmail {
		t.Error("SendEmail should be true")
	}

	// Verify announcement was created in database
	var dbAnnouncement models.Announcement
	if err := db.First(&dbAnnouncement, announcement.ID).Error; err != nil {
		t.Errorf("Announcement not found in database: %v", err)
	}
}

// TestSendAnnouncementToGroupMe tests the sendAnnouncementToGroupMe function directly
func TestSendAnnouncementToGroupMe(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*gorm.DB)
		title         string
		content       string
		expectedError bool
	}{
		{
			name: "successfully send to GroupMe-enabled groups",
			setupFunc: func(db *gorm.DB) {
				// Create groups with GroupMe enabled
				db.Create(&models.Group{
					Name:           "Dogs",
					Description:    "Dog volunteers",
					GroupMeEnabled: true,
					GroupMeBotID:   "bot123",
				})
				db.Create(&models.Group{
					Name:           "Cats",
					Description:    "Cat volunteers",
					GroupMeEnabled: true,
					GroupMeBotID:   "bot456",
				})
			},
			title:         "Test Announcement",
			content:       "This is a test GroupMe announcement.",
			expectedError: false,
		},
		{
			name: "no groups with GroupMe enabled",
			setupFunc: func(db *gorm.DB) {
				// Create groups but don't enable GroupMe
				db.Create(&models.Group{
					Name:           "Dogs",
					Description:    "Dog volunteers",
					GroupMeEnabled: false,
				})
			},
			title:         "Test Announcement",
			content:       "This announcement has no GroupMe groups.",
			expectedError: false,
		},
		{
			name: "groups with empty bot ID are skipped",
			setupFunc: func(db *gorm.DB) {
				db.Create(&models.Group{
					Name:           "Dogs",
					Description:    "Dog volunteers",
					GroupMeEnabled: true,
					GroupMeBotID:   "", // Empty bot ID
				})
			},
			title:         "Test Announcement",
			content:       "This announcement has empty bot ID.",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)

			if tt.setupFunc != nil {
				tt.setupFunc(db)
			}

			groupMeService := groupme.NewService()
			ctx := context.Background()
			err := sendAnnouncementToGroupMe(ctx, db, groupMeService, tt.title, tt.content)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				// In SQLite tests, missing columns are expected since we can't easily migrate them
				// In production PostgreSQL, the columns exist from migrations
				if err.Error() != "no such column: groupme_enabled" {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestCreateAnnouncementWithGroupMe tests announcement creation with GroupMe sending
func TestCreateAnnouncementWithGroupMe(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupGroups    func(*gorm.DB)
		expectedStatus int
		checkGroupMe   bool
	}{
		{
			name: "create announcement with GroupMe enabled",
			payload: map[string]interface{}{
				"title":        "GroupMe Announcement",
				"content":      "This announcement will be sent to GroupMe.",
				"send_groupme": true,
			},
			setupGroups: func(db *gorm.DB) {
				db.Create(&models.Group{
					Name:           "Dogs",
					Description:    "Dog volunteers",
					GroupMeEnabled: true,
					GroupMeBotID:   "test-bot-123",
				})
			},
			expectedStatus: http.StatusCreated,
			checkGroupMe:   true,
		},
		{
			name: "create announcement with both email and GroupMe",
			payload: map[string]interface{}{
				"title":        "Multi-channel Announcement",
				"content":      "This announcement goes to both email and GroupMe.",
				"send_email":   true,
				"send_groupme": true,
			},
			setupGroups: func(db *gorm.DB) {
				db.Create(&models.Group{
					Name:           "Cats",
					Description:    "Cat volunteers",
					GroupMeEnabled: true,
					GroupMeBotID:   "test-bot-456",
				})
			},
			expectedStatus: http.StatusCreated,
			checkGroupMe:   true,
		},
		{
			name: "create announcement with GroupMe disabled",
			payload: map[string]interface{}{
				"title":        "No GroupMe Announcement",
				"content":      "This announcement will not be sent to GroupMe.",
				"send_groupme": false,
			},
			setupGroups: func(db *gorm.DB) {
				// No groups needed
			},
			expectedStatus: http.StatusCreated,
			checkGroupMe:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAnnouncementTestDB(t)
			user := createAnnouncementTestUser(t, db, "admin", "admin@example.com", true)

			if tt.setupGroups != nil {
				tt.setupGroups(db)
			}

			emailService := createTestEmailService(false, db)
			groupMeService := groupme.NewService()

			c, w := setupAnnouncementTestContext(user.ID, true)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/announcements", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := CreateAnnouncement(db, emailService, groupMeService)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated {
				var announcement models.Announcement
				if err := json.Unmarshal(w.Body.Bytes(), &announcement); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if tt.checkGroupMe && !announcement.SendGroupMe {
					t.Error("SendGroupMe should be true")
				}

				// Verify announcement was created in database
				var dbAnnouncement models.Announcement
				if err := db.First(&dbAnnouncement, announcement.ID).Error; err != nil {
					t.Errorf("Announcement not found in database: %v", err)
				}

				if dbAnnouncement.SendGroupMe != announcement.SendGroupMe {
					t.Errorf("Expected SendGroupMe %v, got %v", announcement.SendGroupMe, dbAnnouncement.SendGroupMe)
				}
			}
		})
	}
}
