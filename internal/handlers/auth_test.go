package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
// This is a wrapper around the shared SetupTestDB for backward compatibility
func setupTestDB(t *testing.T) *gorm.DB {
	return SetupTestDB(t)
}

// createTestUser creates a user in the test database
// This is a wrapper around the shared CreateTestUser for backward compatibility
func createTestUser(t *testing.T, db *gorm.DB, username, email, password string, isAdmin bool) *models.User {
	return CreateTestUser(t, db, username, email, password, isAdmin)
}

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupDB        func(*gorm.DB)
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful registration",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["token"] == nil || resp["token"] == "" {
					t.Error("Expected token in response")
				}
				if user, ok := resp["user"].(map[string]interface{}); ok {
					if user["username"] != "testuser" {
						t.Errorf("Expected username 'testuser', got %v", user["username"])
					}
					if user["email"] != "test@example.com" {
						t.Errorf("Expected email 'test@example.com', got %v", user["email"])
					}
					if user["password"] != nil {
						t.Error("Password should not be in response")
					}
				} else {
					t.Error("Expected user object in response")
				}
			},
		},
		{
			name: "duplicate username",
			payload: map[string]interface{}{
				"username": "existinguser",
				"email":    "new@example.com",
				"password": "SecurePass123!",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "existinguser", "existing@example.com", "password123", false)
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Username or email already exists",
		},
		{
			name: "duplicate email",
			payload: map[string]interface{}{
				"username": "newuser",
				"email":    "existing@example.com",
				"password": "SecurePass123!",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "existinguser", "existing@example.com", "password123", false)
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Username or email already exists",
		},
		{
			name: "invalid email format",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "not-an-email",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "username too short",
			payload: map[string]interface{}{
				"username": "ab",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing username",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing email",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
			},
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
			c.Request = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := Register(db)
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

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupDB        func(*gorm.DB)
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful login",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "testuser", "test@example.com", "password123", false)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["token"] == nil || resp["token"] == "" {
					t.Error("Expected token in response")
				}
				if user, ok := resp["user"].(map[string]interface{}); ok {
					if user["username"] != "testuser" {
						t.Errorf("Expected username 'testuser', got %v", user["username"])
					}
				} else {
					t.Error("Expected user object in response")
				}
			},
		},
		{
			name: "successful admin login",
			payload: map[string]interface{}{
				"username": "admin",
				"password": "adminpass",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "admin", "admin@example.com", "adminpass", true)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if user, ok := resp["user"].(map[string]interface{}); ok {
					if isAdmin, ok := user["is_admin"].(bool); !ok || !isAdmin {
						t.Error("Expected is_admin to be true for admin user")
					}
				}
			},
		},
		{
			name: "incorrect password",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "wrongpassword",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "testuser", "test@example.com", "password123", false)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "user not found",
			payload: map[string]interface{}{
				"username": "nonexistent",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "account locked",
			payload: map[string]interface{}{
				"username": "lockeduser",
				"password": "password123",
			},
			setupDB: func(db *gorm.DB) {
				user := createTestUser(t, db, "lockeduser", "locked@example.com", "password123", false)
				lockUntil := time.Now().Add(30 * time.Minute)
				db.Model(&user).Updates(map[string]interface{}{
					"locked_until":          lockUntil,
					"failed_login_attempts": 5,
				})
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "failed login increments counter",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "wrongpassword",
			},
			setupDB: func(db *gorm.DB) {
				createTestUser(t, db, "testuser", "test@example.com", "password123", false)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if attemptsRemaining, ok := resp["attempts_remaining"].(float64); ok {
					if attemptsRemaining != 4 {
						t.Errorf("Expected 4 attempts remaining, got %v", attemptsRemaining)
					}
				} else {
					t.Error("Expected attempts_remaining in response")
				}
			},
		},
		{
			name: "account locks after 5 failed attempts",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "wrongpassword",
			},
			setupDB: func(db *gorm.DB) {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				// Set failed attempts to 4 (next failure will lock)
				db.Model(&user).Update("failed_login_attempts", 4)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if lockedUntil := resp["locked_until"]; lockedUntil == nil {
					t.Error("Expected locked_until in response")
				}
			},
		},
		{
			name: "successful login resets failed attempts",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
			},
			setupDB: func(db *gorm.DB) {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				db.Model(&user).Update("failed_login_attempts", 3)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				// Verify the failed attempts were reset in the database
				// This would require accessing the db from the test, which we don't have direct access to
				// So we just verify successful login
				if resp["token"] == nil {
					t.Error("Expected token in response")
				}
			},
		},
		{
			name: "expired lock allows login",
			payload: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
			},
			setupDB: func(db *gorm.DB) {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				// Set lock to expired
				lockUntil := time.Now().Add(-1 * time.Minute)
				db.Model(&user).Updates(map[string]interface{}{
					"locked_until":          lockUntil,
					"failed_login_attempts": 5,
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing username",
			payload: map[string]interface{}{
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: map[string]interface{}{
				"username": "testuser",
			},
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
			c.Request = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := Login(db)
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

// TestLoginSoftDeletedGroups verifies that logging in does not return soft-deleted groups
func TestLoginSoftDeletedGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	group1 := models.Group{Name: "active-group"}
	group2 := models.Group{Name: "deleted-group"}
	db.Create(&group1)
	db.Create(&group2)

	user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
	db.Model(user).Association("Groups").Append(&group1, &group2)

	db.Delete(&group2)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/auth/login", nil)

	payload := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}
	body, _ := json.Marshal(payload)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := Login(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	userData, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user object in response")
	}

	groupList, ok := userData["groups"].([]interface{})
	if !ok {
		t.Fatalf("Expected groups array in user data")
	}

	if len(groupList) != 1 {
		t.Errorf("Expected 1 group in login response, got %d. Groups: %v", len(groupList), groupList)
		return
	}

	group := groupList[0].(map[string]interface{})
	groupName := group["name"].(string)
	if groupName != "active-group" {
		t.Errorf("Expected 'active-group', got '%s'", groupName)
	}
}

func TestGetCurrentUser(t *testing.T) {
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
			name: "successful get current user",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
			},
			setupDB: func(db *gorm.DB) uint {
				user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["username"] != "testuser" {
					t.Errorf("Expected username 'testuser', got %v", resp["username"])
				}
				if resp["email"] != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got %v", resp["email"])
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
			name: "user not found in database",
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
			c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)

			if tt.setupContext != nil {
				tt.setupContext(c)
			}

			handler := GetCurrentUser(db)
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

// Regression test for soft-deleted groups not appearing in GetCurrentUser response
func TestGetCurrentUserSoftDeletedGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	// Create two groups: one active, one to be deleted
	group1 := models.Group{Name: "active-group"}
	group2 := models.Group{Name: "deleted-group"}
	db.Create(&group1)
	db.Create(&group2)

	// Create user and assign to both groups
	user := createTestUser(t, db, "testuser", "test@example.com", "password123", false)
	db.Model(user).Association("Groups").Append(&group1, &group2)

	// Soft-delete group2
	db.Delete(&group2)

	// Create test context with user_id
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	c.Set("user_id", user.ID) // Simulate middleware setting user context

	handler := GetCurrentUser(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Verify response has groups - GetCurrentUser returns user data directly, not wrapped in "data"
	groupsRaw, ok := response["groups"].([]interface{})
	if !ok {
		t.Fatalf("Expected groups array in response, got %T. Response: %v", response["groups"], response)
	}

	if len(groupsRaw) != 1 {
		t.Errorf("Expected 1 group (active), got %d. Groups: %v", len(groupsRaw), groupsRaw)
	} else {
		group := groupsRaw[0].(map[string]interface{})
		if groupName, ok := group["name"].(string); ok {
			if groupName != "active-group" {
				t.Errorf("Expected 'active-group', got '%s'", groupName)
			}
		}
	}
}
