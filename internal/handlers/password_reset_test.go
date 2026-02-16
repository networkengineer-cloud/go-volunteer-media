package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// createTestEmailService creates a mock email service for testing
func createTestEmailService(configured bool, db *gorm.DB) *email.Service {
	if configured {
		// Create a service with a mock SMTP provider
		provider := &email.SMTPProvider{
			Host:      "smtp.example.com",
			Port:      "587",
			Username:  "user",
			Password:  "pass",
			FromEmail: "noreply@example.com",
		}
		return email.NewServiceWithProvider(provider, db)
	}
	// Return unconfigured service
	return email.NewServiceWithProvider(nil, db)
}

func TestRequestPasswordReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupDB        func(*gorm.DB)
		emailService   *email.Service
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{}, *gorm.DB)
	}{
		{
			name: "successful password reset request",
			payload: map[string]interface{}{
				"email": "test@example.com",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "testuser", "test@example.com", "password123", false)
			},
			emailService: createTestEmailService(true, nil),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}, db *gorm.DB) {
				// Check that reset token was set in database
				var user models.User
				db.Where("email = ?", "test@example.com").First(&user)
				if user.ResetToken == "" {
					t.Error("Expected reset token to be set")
				}
				if user.ResetTokenExpiry == nil {
					t.Error("Expected reset token expiry to be set")
				}
				if user.ResetTokenExpiry.Before(time.Now()) {
					t.Error("Expected reset token expiry to be in the future")
				}
			},
		},
		{
			name: "email not found - returns success anyway (prevent enumeration)",
			payload: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			emailService: createTestEmailService(true, nil),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}, db *gorm.DB) {
				if msg, ok := resp["message"].(string); !ok || msg == "" {
					t.Error("Expected generic success message")
				}
			},
		},
		{
			name: "email service not configured - returns success anyway",
			payload: map[string]interface{}{
				"email": "test@example.com",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "testuser", "test@example.com", "password123", false)
			},
			emailService:   createTestEmailService(false, nil), // Not configured
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid email format",
			payload: map[string]interface{}{
				"email": "not-an-email",
			},
			emailService:   createTestEmailService(false, nil),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "missing email",
			payload: map[string]interface{}{
				// No email field
			},
			emailService:   createTestEmailService(false, nil),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			if tt.setupDB != nil {
				tt.setupDB(db)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/auth/request-password-reset", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := RequestPasswordReset(db, tt.emailService)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response, db)
			}
		})
	}
}

func TestResetPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupDB        func(*gorm.DB) string // returns the unhashed token
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, map[string]interface{}, *gorm.DB)
	}{
		{
			name: "successful password reset",
			setupDB: func(db *gorm.DB) string {
				user := createTestUser(t, db, "testuser", "test@example.com", "oldpassword", false)
				token := "valid-reset-token-abc123"
				hashedToken, _ := auth.HashPassword(token)
				expiry := time.Now().Add(1 * time.Hour)
				db.Model(&user).Updates(map[string]interface{}{
					"reset_token":        hashedToken,
					"reset_token_expiry": expiry,
				})
				return token
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}, db *gorm.DB) {
				// Verify password was changed and token cleared
				var user models.User
				db.Where("email = ?", "test@example.com").First(&user)
				if user.ResetToken != "" {
					t.Error("Expected reset token to be cleared")
				}
				if user.ResetTokenExpiry != nil {
					t.Error("Expected reset token expiry to be cleared")
				}
				// Verify new password works
				if err := auth.CheckPassword(user.Password, "NewSecurePass123!"); err != nil {
					t.Error("Expected new password to be set correctly")
				}
			},
		},
		{
			name: "invalid reset token",
			payload: map[string]interface{}{
				"token":        "invalid-token",
				"new_password": "NewSecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid or expired reset token",
		},
		{
			name: "expired reset token",
			setupDB: func(db *gorm.DB) string {
				user := createTestUser(t, db, "testuser", "test@example.com", "oldpassword", false)
				token := "expired-reset-token"
				hashedToken, _ := auth.HashPassword(token)
				expiry := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
				db.Model(&user).Updates(map[string]interface{}{
					"reset_token":        hashedToken,
					"reset_token_expiry": expiry,
				})
				return token
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reset token has expired. Please request a new one.",
		},
		{
			name: "password reset clears account lock",
			setupDB: func(db *gorm.DB) string {
				user := createTestUser(t, db, "testuser", "test@example.com", "oldpassword", false)
				token := "reset-token-for-locked"
				hashedToken, _ := auth.HashPassword(token)
				expiry := time.Now().Add(1 * time.Hour)
				lockUntil := time.Now().Add(30 * time.Minute)
				db.Model(&user).Updates(map[string]interface{}{
					"reset_token":           hashedToken,
					"reset_token_expiry":    expiry,
					"failed_login_attempts": 5,
					"locked_until":          lockUntil,
				})
				return token
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}, db *gorm.DB) {
				var user models.User
				db.Where("email = ?", "test@example.com").First(&user)
				if user.FailedLoginAttempts != 0 {
					t.Error("Expected failed login attempts to be reset")
				}
				if user.LockedUntil != nil {
					t.Error("Expected account lock to be cleared")
				}
			},
		},
		{
			name: "password too short",
			setupDB: func(db *gorm.DB) string {
				user := createTestUser(t, db, "testuser", "test@example.com", "oldpassword", false)
				token := "valid-token"
				hashedToken, _ := auth.HashPassword(token)
				expiry := time.Now().Add(1 * time.Hour)
				db.Model(&user).Updates(map[string]interface{}{
					"reset_token":        hashedToken,
					"reset_token_expiry": expiry,
				})
				return token
			},
			payload: map[string]interface{}{
				"new_password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing token",
			payload: map[string]interface{}{
				"new_password": "NewSecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing new password",
			payload: map[string]interface{}{
				"token": "some-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			var token string
			if tt.setupDB != nil {
				token = tt.setupDB(db)
			}

			// Build payload
			payload := tt.payload
			if payload == nil {
				payload = make(map[string]interface{})
			}
			// Add token if we got one from setupDB and it's not already in payload
			if token != "" && payload["token"] == nil {
				payload["token"] = token
			}
			// Add default new password if not specified and token is present
			if payload["token"] != nil && payload["new_password"] == nil && tt.name != "missing new password" {
				payload["new_password"] = "NewSecurePass123!"
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := ResetPassword(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedError != "" {
				if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.expectedError {
					t.Errorf("Expected error '%s', got '%v'", tt.expectedError, response["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response, db)
			}
		})
	}
}

func TestUpdateEmailPreferences(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupContext   func(*gin.Context)
		setupDB        func(*gorm.DB) uint
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "enable email notifications",
			payload: map[string]interface{}{
				"email_notifications_enabled": true,
			},
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupDB: func(db *gorm.DB) uint {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if enabled, ok := resp["email_notifications_enabled"].(bool); !ok || !enabled {
					t.Error("Expected email_notifications_enabled to be true")
				}
			},
		},
		{
			name: "disable email notifications",
			payload: map[string]interface{}{
				"email_notifications_enabled": false,
			},
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupDB: func(db *gorm.DB) uint {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if enabled, ok := resp["email_notifications_enabled"].(bool); !ok || enabled {
					t.Error("Expected email_notifications_enabled to be false")
				}
			},
		},
		{
			name:           "missing user context",
			payload:        map[string]interface{}{"email_notifications_enabled": true},
			setupContext:   func(c *gin.Context) {},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "User context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			if tt.setupDB != nil {
				tt.setupDB(db)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("PUT", "/api/v1/auth/email-preferences", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			if tt.setupContext != nil {
				tt.setupContext(c)
			}

			handler := UpdateEmailPreferences(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedError != "" {
				if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.expectedError {
					t.Errorf("Expected error '%s', got '%v'", tt.expectedError, response["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestGetEmailPreferences(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupDB        func(*gorm.DB) uint
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "get email preferences",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupDB: func(db *gorm.DB) uint {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				db.Model(&user).Update("email_notifications_enabled", true)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if enabled, ok := resp["email_notifications_enabled"].(bool); !ok || !enabled {
					t.Error("Expected email_notifications_enabled to be true")
				}
			},
		},
		{
			name:           "missing user context",
			setupContext:   func(c *gin.Context) {},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "User context not found",
		},
		{
			name: "user not found",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			if tt.setupDB != nil {
				tt.setupDB(db)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/auth/email-preferences", nil)

			if tt.setupContext != nil {
				tt.setupContext(c)
			}

			handler := GetEmailPreferences(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedError != "" {
				if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.expectedError {
					t.Errorf("Expected error '%s', got '%v'", tt.expectedError, response["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}
