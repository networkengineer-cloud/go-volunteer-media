package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	// Register custom username validator for testing
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("usernamechars", func(fl validator.FieldLevel) bool {
			username := fl.Field().String()
			matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, username)
			return matched
		})
	}
}

// setupUserAdminTestDB creates an in-memory SQLite database for user admin testing
func setupUserAdminTestDB(t *testing.T) *gorm.DB {
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

// createUserAdminTestUser creates a user for testing
func createUserAdminTestUser(t *testing.T, db *gorm.DB, username, email string, isAdmin bool) *models.User {
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

// setupUserAdminTestContext creates a Gin context with authenticated admin user
func setupUserAdminTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	return c, w
}

// TestPromoteUser tests promoting a regular user to admin
func TestPromoteUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "successfully promote regular user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "regularuser", "user@example.com", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID uint) {
				var user models.User
				if err := db.First(&user, userID).Error; err != nil {
					t.Fatalf("Failed to find user: %v", err)
				}
				if !user.IsAdmin {
					t.Error("User should be admin after promotion")
				}
			},
		},
		{
			name: "cannot promote already admin user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "adminuser", "admin@example.com", true)
				return user.ID
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			userID := tt.setupFunc(db)

			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", userID)}}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/users/%d/promote", userID), nil)

			handler := PromoteUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID)
			}
		})
	}
}

// TestDemoteUser tests demoting an admin user to regular user
func TestDemoteUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "successfully demote admin user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "adminuser", "admin@example.com", true)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID uint) {
				var user models.User
				if err := db.First(&user, userID).Error; err != nil {
					t.Fatalf("Failed to find user: %v", err)
				}
				if user.IsAdmin {
					t.Error("User should not be admin after demotion")
				}
			},
		},
		{
			name: "cannot demote regular user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "regularuser", "user@example.com", false)
				return user.ID
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			userID := tt.setupFunc(db)

			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", userID)}}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/users/%d/demote", userID), nil)

			handler := DemoteUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID)
			}
		})
	}
}

// TestAdminCreateUser tests admin creating new users
func TestAdminCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		setupFunc      func(*gorm.DB)
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, *httptest.ResponseRecorder)
	}{
		{
			name: "successfully create regular user",
			payload: map[string]interface{}{
				"username": "newuser",
				"email":    "newuser@example.com",
				"password": "password123",
				"is_admin": false,
			},
			setupFunc:      nil,
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var user models.User
				if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if user.Username != "newuser" {
					t.Errorf("Expected username 'newuser', got '%s'", user.Username)
				}
				if user.Email != "newuser@example.com" {
					t.Errorf("Expected email 'newuser@example.com', got '%s'", user.Email)
				}
				if user.IsAdmin {
					t.Error("User should not be admin")
				}

				// Verify password was hashed
				var dbUser models.User
				if err := db.First(&dbUser, user.ID).Error; err != nil {
					t.Fatalf("Failed to find user in database: %v", err)
				}
				if dbUser.Password == "password123" {
					t.Error("Password should be hashed")
				}
			},
		},
		{
			name: "successfully create admin user",
			payload: map[string]interface{}{
				"username": "newadmin",
				"email":    "newadmin@example.com",
				"password": "password123",
				"is_admin": true,
			},
			setupFunc:      nil,
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var user models.User
				if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if !user.IsAdmin {
					t.Error("User should be admin")
				}
			},
		},
		{
			name: "create user with groups",
			payload: map[string]interface{}{
				"username":  "groupuser",
				"email":     "groupuser@example.com",
				"password":  "password123",
				"is_admin":  false,
				"group_ids": []uint{1, 2},
			},
			setupFunc: func(db *gorm.DB) {
				// Create test groups
				db.Create(&models.Group{Name: "Group 1", Description: "Test group 1"})
				db.Create(&models.Group{Name: "Group 2", Description: "Test group 2"})
			},
			expectedStatus: http.StatusCreated,
			checkFunc: func(t *testing.T, db *gorm.DB, w *httptest.ResponseRecorder) {
				var user models.User
				if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(user.Groups) != 2 {
					t.Errorf("Expected 2 groups, got %d", len(user.Groups))
				}
			},
		},
		{
			name: "duplicate username",
			payload: map[string]interface{}{
				"username": "existinguser",
				"email":    "newemail@example.com",
				"password": "password123",
			},
			setupFunc: func(db *gorm.DB) {
				createUserAdminTestUser(t, db, "existinguser", "existing@example.com", false)
			},
			expectedStatus: http.StatusConflict,
			checkFunc:      nil,
		},
		{
			name: "duplicate email",
			payload: map[string]interface{}{
				"username": "newusername",
				"email":    "existing@example.com",
				"password": "password123",
			},
			setupFunc: func(db *gorm.DB) {
				createUserAdminTestUser(t, db, "existinguser", "existing@example.com", false)
			},
			expectedStatus: http.StatusConflict,
			checkFunc:      nil,
		},
		{
			name: "validation error - missing username",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			setupFunc:      nil,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - invalid email",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "invalid-email",
				"password": "password123",
			},
			setupFunc:      nil,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - password too short",
			payload: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "short",
			},
			setupFunc:      nil,
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			if tt.setupFunc != nil {
				tt.setupFunc(db)
			}

			c, w := setupUserAdminTestContext(admin.ID, true)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/api/v1/admin/users", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// Create a nil email service for test (email functionality will be skipped)
			handler := AdminCreateUser(db, nil)
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

// TestAdminResetUserPassword tests admin resetting user passwords
func TestAdminResetUserPassword(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		payload        map[string]interface{}
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "successfully reset password",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "targetuser", "target@example.com", false)
				return user.ID
			},
			payload: map[string]interface{}{
				"new_password": "newpassword123",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID uint) {
				var user models.User
				if err := db.First(&user, userID).Error; err != nil {
					t.Fatalf("Failed to find user: %v", err)
				}

				// Verify new password works
				if err := auth.CheckPassword(user.Password, "newpassword123"); err != nil {
					t.Error("New password should be valid")
				}

				// Verify old password doesn't work
				if err := auth.CheckPassword(user.Password, "password123"); err == nil {
					t.Error("Old password should not work")
				}

				// Verify lockout was cleared
				if user.FailedLoginAttempts != 0 {
					t.Error("Failed login attempts should be cleared")
				}
			},
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			payload: map[string]interface{}{
				"new_password": "newpassword123",
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name: "validation error - password too short",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "targetuser", "target@example.com", false)
				return user.ID
			},
			payload: map[string]interface{}{
				"new_password": "short",
			},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name: "validation error - missing password",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "targetuser", "target@example.com", false)
				return user.ID
			},
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			userID := tt.setupFunc(db)

			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", userID)}}

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/users/%d/reset-password", userID), bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := AdminResetUserPassword(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID)
			}
		})
	}
}

// TestSelfServicePasswordChange tests the self-service password change flow
func TestSelfServicePasswordChange(t *testing.T) {
	t.Run("valid self-reset with correct current_password", func(t *testing.T) {
		db := setupUserAdminTestDB(t)
		user := createUserAdminTestUser(t, db, "selfuser", "self@test.com", false)

		c, w := setupUserAdminTestContext(user.ID, false)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", user.ID)}}

		payload := map[string]interface{}{
			"current_password": "password123",
			"new_password":     "newpassword456",
		}
		jsonBytes, _ := json.Marshal(payload)
		c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/reset-password", user.ID), bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler := AdminResetUserPassword(db)
		handler(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify new password works
		var updated models.User
		db.First(&updated, user.ID)
		if err := auth.CheckPassword(updated.Password, "newpassword456"); err != nil {
			t.Error("New password should be valid after self-reset")
		}
	})

	t.Run("self-reset rejected when current_password is wrong", func(t *testing.T) {
		db := setupUserAdminTestDB(t)
		user := createUserAdminTestUser(t, db, "selfuser", "self@test.com", false)

		c, w := setupUserAdminTestContext(user.ID, false)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", user.ID)}}

		payload := map[string]interface{}{
			"current_password": "wrongpassword",
			"new_password":     "newpassword456",
		}
		jsonBytes, _ := json.Marshal(payload)
		c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/reset-password", user.ID), bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler := AdminResetUserPassword(db)
		handler(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, w.Code, w.Body.String())
		}
	})

	t.Run("self-reset rejected when current_password is missing", func(t *testing.T) {
		db := setupUserAdminTestDB(t)
		user := createUserAdminTestUser(t, db, "selfuser", "self@test.com", false)

		c, w := setupUserAdminTestContext(user.ID, false)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", user.ID)}}

		payload := map[string]interface{}{
			"new_password": "newpassword456",
		}
		jsonBytes, _ := json.Marshal(payload)
		c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/reset-password", user.ID), bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler := AdminResetUserPassword(db)
		handler(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("regular user cannot reset another user's password", func(t *testing.T) {
		db := setupUserAdminTestDB(t)
		attacker := createUserAdminTestUser(t, db, "attacker", "attacker@test.com", false)
		target := createUserAdminTestUser(t, db, "target", "target@test.com", false)

		c, w := setupUserAdminTestContext(attacker.ID, false)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}

		payload := map[string]interface{}{
			"new_password": "newpassword456",
		}
		jsonBytes, _ := json.Marshal(payload)
		c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/reset-password", target.ID), bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler := AdminResetUserPassword(db)
		handler(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, w.Code, w.Body.String())
		}
	})

	t.Run("admin reset does not require current_password", func(t *testing.T) {
		db := setupUserAdminTestDB(t)
		admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)
		target := createUserAdminTestUser(t, db, "target", "target@test.com", false)

		c, w := setupUserAdminTestContext(admin.ID, true)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}

		payload := map[string]interface{}{
			"new_password": "newpassword456",
		}
		jsonBytes, _ := json.Marshal(payload)
		c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/reset-password", target.ID), bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler := AdminResetUserPassword(db)
		handler(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})
}

// TestAdminDeleteUser tests soft-deleting users
func TestAdminDeleteUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "successfully soft delete user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "deleteuser", "delete@example.com", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID uint) {
				// Should not be found in normal queries
				var user models.User
				err := db.First(&user, userID).Error
				if err == nil {
					t.Error("User should be soft-deleted and not found in normal queries")
				}

				// Should be found with Unscoped
				err = db.Unscoped().First(&user, userID).Error
				if err != nil {
					t.Error("User should still exist with Unscoped query")
				}
				if !user.DeletedAt.Valid {
					t.Error("User should have deleted_at timestamp")
				}
			},
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			userID := tt.setupFunc(db)

			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", userID)}}
			c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/users/%d", userID), nil)

			handler := AdminDeleteUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, userID)
			}
		})
	}
}

// TestGetDeletedUsers tests retrieving soft-deleted users
func TestGetDeletedUsers(t *testing.T) {
	db := setupUserAdminTestDB(t)
	admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

	// Create and delete some users
	user1 := createUserAdminTestUser(t, db, "deleted1", "deleted1@example.com", false)
	user2 := createUserAdminTestUser(t, db, "deleted2", "deleted2@example.com", false)
	createUserAdminTestUser(t, db, "active", "active@example.com", false)

	// Soft delete two users
	db.Delete(&user1)
	db.Delete(&user2)

	c, w := setupUserAdminTestContext(admin.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/users/deleted", nil)

	handler := GetDeletedUsers(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var users []models.User
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 deleted users, got %d", len(users))
	}
}

// TestRestoreUser tests restoring soft-deleted users
func TestRestoreUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) uint
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name: "successfully restore deleted user",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "deleteduser", "deleted@example.com", false)
				db.Delete(&user)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, userID uint) {
				var user models.User
				err := db.First(&user, userID).Error
				if err != nil {
					t.Error("User should be restored and found in normal queries")
				}
				if user.DeletedAt.Valid {
					t.Error("User should not have deleted_at timestamp after restore")
				}
			},
		},
		{
			name: "restore already active user (idempotent)",
			setupFunc: func(db *gorm.DB) uint {
				user := createUserAdminTestUser(t, db, "activeuser", "active@example.com", false)
				return user.ID
			},
			expectedStatus: http.StatusOK,
			checkFunc:      nil,
		},
		{
			name: "user not found",
			setupFunc: func(db *gorm.DB) uint {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@test.com", true)

			userID := tt.setupFunc(db)

			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", userID)}}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/users/%d/restore", userID), nil)

			handler := RestoreUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				// Need to wait a bit for the database to commit
				time.Sleep(10 * time.Millisecond)
				tt.checkFunc(t, db, userID)
			}
		})
	}
}

// assignUserToGroup assigns a user to a group, optionally as group admin
func assignUserToGroup(t *testing.T, db *gorm.DB, userID, groupID uint, isGroupAdmin bool) {
	t.Helper()
	ug := &models.UserGroup{UserID: userID, GroupID: groupID, IsGroupAdmin: isGroupAdmin}
	if err := db.Create(ug).Error; err != nil {
		t.Fatalf("Failed to assign user to group: %v", err)
	}
}

func TestAdminUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*testing.T, *gorm.DB) string // returns userId
		body           UpdateUserRequest
		expectedStatus int
		checkFunc      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful update",
			setupFunc: func(t *testing.T, db *gorm.DB) string {
				target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
				return fmt.Sprintf("%d", target.ID)
			},
			body: UpdateUserRequest{
				FirstName: "Updated",
				LastName:  "Name",
				Email:     "updated@example.com",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp models.User
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if resp.FirstName != "Updated" || resp.LastName != "Name" {
					t.Errorf("Expected name Updated Name, got %s %s", resp.FirstName, resp.LastName)
				}
			},
		},
		{
			name: "invalid user ID",
			setupFunc: func(t *testing.T, db *gorm.DB) string {
				return "abc"
			},
			body: UpdateUserRequest{
				Email: "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "nonexistent user",
			setupFunc: func(t *testing.T, db *gorm.DB) string {
				return "99999"
			},
			body: UpdateUserRequest{
				Email: "test@example.com",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "email conflict",
			setupFunc: func(t *testing.T, db *gorm.DB) string {
				createUserAdminTestUser(t, db, "existing", "existing@example.com", false)
				target := createUserAdminTestUser(t, db, "target2", "target2@example.com", false)
				return fmt.Sprintf("%d", target.ID)
			},
			body: UpdateUserRequest{
				Email: "existing@example.com",
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			admin := createUserAdminTestUser(t, db, "admin", "admin@example.com", true)
			userId := tt.setupFunc(t, db)

			body, _ := json.Marshal(tt.body)
			c, w := setupUserAdminTestContext(admin.ID, true)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/users/"+userId, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "userId", Value: userId}}

			handler := AdminUpdateUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil && w.Code == tt.expectedStatus {
				tt.checkFunc(t, w)
			}
		})
	}
}

func TestGroupAdminUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*testing.T, *gorm.DB) (adminID uint, targetUserID string)
		body           UpdateUserRequest
		expectedStatus int
	}{
		{
			name: "group admin can update user in their group",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
				group := createTestGroup(t, db, "TestGroup", "Test group")
				assignUserToGroup(t, db, groupAdmin.ID, group.ID, true)
				assignUserToGroup(t, db, target.ID, group.ID, false)
				return groupAdmin.ID, fmt.Sprintf("%d", target.ID)
			},
			body:           UpdateUserRequest{FirstName: "New", LastName: "Name", Email: "target@example.com"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "group admin rejected for user not in their group",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
				group1 := createTestGroup(t, db, "Group1", "Group 1")
				group2 := createTestGroup(t, db, "Group2", "Group 2")
				assignUserToGroup(t, db, groupAdmin.ID, group1.ID, true)
				assignUserToGroup(t, db, target.ID, group2.ID, false)
				return groupAdmin.ID, fmt.Sprintf("%d", target.ID)
			},
			body:           UpdateUserRequest{Email: "target@example.com"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group admin rejected for user with no groups",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
				group := createTestGroup(t, db, "TestGroup", "Test group")
				assignUserToGroup(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin.ID, fmt.Sprintf("%d", target.ID)
			},
			body:           UpdateUserRequest{Email: "target@example.com"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "invalid user ID returns 400",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				return groupAdmin.ID, "abc"
			},
			body:           UpdateUserRequest{Email: "test@example.com"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "nonexistent user returns 404",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				return groupAdmin.ID, "99999"
			},
			body:           UpdateUserRequest{Email: "test@example.com"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "email conflict returns 409",
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, string) {
				groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
				createUserAdminTestUser(t, db, "other", "taken@example.com", false)
				target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
				group := createTestGroup(t, db, "TestGroup", "Test group")
				assignUserToGroup(t, db, groupAdmin.ID, group.ID, true)
				assignUserToGroup(t, db, target.ID, group.ID, false)
				return groupAdmin.ID, fmt.Sprintf("%d", target.ID)
			},
			body:           UpdateUserRequest{Email: "taken@example.com"},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)
			adminID, targetUserID := tt.setupFunc(t, db)

			body, _ := json.Marshal(tt.body)
			c, w := setupUserAdminTestContext(adminID, false)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/group-admin/users/"+targetUserID, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "userId", Value: targetUserID}}

			handler := GroupAdminUpdateUser(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGroupAdminUpdateUser_SiteAdminBypass(t *testing.T) {
	// #10: site admin calling GroupAdminUpdateUser on user with no groups
	db := setupUserAdminTestDB(t)
	admin := createUserAdminTestUser(t, db, "admin", "admin@example.com", true)
	target := createUserAdminTestUser(t, db, "target", "target@example.com", false)

	body, _ := json.Marshal(UpdateUserRequest{
		FirstName: "Updated",
		Email:     "target@example.com",
	})
	c, w := setupUserAdminTestContext(admin.ID, true)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/users/"+fmt.Sprintf("%d", target.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}

	handler := GroupAdminUpdateUser(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGroupAdminUpdateUser_RejectsAdminTarget(t *testing.T) {
	// #4: group admin cannot update a site admin
	db := setupUserAdminTestDB(t)
	groupAdmin := createUserAdminTestUser(t, db, "gadmin", "gadmin@example.com", false)
	siteAdmin := createUserAdminTestUser(t, db, "siteadmin", "sa@example.com", true)
	group := createTestGroup(t, db, "SharedGroup", "Shared")
	assignUserToGroup(t, db, groupAdmin.ID, group.ID, true)
	assignUserToGroup(t, db, siteAdmin.ID, group.ID, false)

	body, _ := json.Marshal(UpdateUserRequest{Email: "sa@example.com", FirstName: "Hacked"})
	c, w := setupUserAdminTestContext(groupAdmin.ID, false)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/users/"+fmt.Sprintf("%d", siteAdmin.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", siteAdmin.ID)}}

	handler := GroupAdminUpdateUser(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGroupAdminUpdateUser_RejectsGroupAdminTarget(t *testing.T) {
	// #4: group admin cannot update another group admin
	db := setupUserAdminTestDB(t)
	groupAdmin1 := createUserAdminTestUser(t, db, "gadmin1", "gadmin1@example.com", false)
	groupAdmin2 := createUserAdminTestUser(t, db, "gadmin2", "gadmin2@example.com", false)
	group := createTestGroup(t, db, "SharedGroup", "Shared")
	assignUserToGroup(t, db, groupAdmin1.ID, group.ID, true)
	assignUserToGroup(t, db, groupAdmin2.ID, group.ID, true)

	body, _ := json.Marshal(UpdateUserRequest{Email: "gadmin2@example.com", FirstName: "Hacked"})
	c, w := setupUserAdminTestContext(groupAdmin1.ID, false)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/users/"+fmt.Sprintf("%d", groupAdmin2.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", groupAdmin2.ID)}}

	handler := GroupAdminUpdateUser(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestAdminUpdateUser_ClearNameFields(t *testing.T) {
	// #11: clearing first/last name with empty strings
	db := setupUserAdminTestDB(t)
	admin := createUserAdminTestUser(t, db, "admin", "admin@example.com", true)
	target := createUserAdminTestUser(t, db, "target", "target@example.com", false)
	db.Model(target).Updates(map[string]interface{}{"first_name": "Original", "last_name": "Name"})

	body, _ := json.Marshal(UpdateUserRequest{
		FirstName: "",
		LastName:  "",
		Email:     "target@example.com",
	})
	c, w := setupUserAdminTestContext(admin.ID, true)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/users/"+fmt.Sprintf("%d", target.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}

	handler := AdminUpdateUser(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var updated models.User
	db.First(&updated, target.ID)
	if updated.FirstName != "" || updated.LastName != "" {
		t.Errorf("Expected empty names, got first=%q last=%q", updated.FirstName, updated.LastName)
	}
}

func TestAdminUpdateUser_TrimWhitespaceNames(t *testing.T) {
	// #6: names should be trimmed
	db := setupUserAdminTestDB(t)
	admin := createUserAdminTestUser(t, db, "admin", "admin@example.com", true)
	target := createUserAdminTestUser(t, db, "target", "target@example.com", false)

	body, _ := json.Marshal(UpdateUserRequest{
		FirstName: "  Alice  ",
		LastName:  "  Smith  ",
		Email:     "target@example.com",
	})
	c, w := setupUserAdminTestContext(admin.ID, true)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/users/"+fmt.Sprintf("%d", target.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}

	handler := AdminUpdateUser(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp models.User
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.FirstName != "Alice" || resp.LastName != "Smith" {
		t.Errorf("Expected trimmed names, got first=%q last=%q", resp.FirstName, resp.LastName)
	}
}

func TestUpdateCurrentUserProfile_NameFields(t *testing.T) {
	// #12: test UpdateCurrentUserProfile with name fields and phone max length
	db := setupUserAdminTestDB(t)
	user := createUserAdminTestUser(t, db, "user1", "user1@example.com", false)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		checkFunc      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "update first and last name",
			body:           `{"first_name":"Jane","last_name":"Doe","email":"user1@example.com"}`,
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["first_name"] != "Jane" || resp["last_name"] != "Doe" {
					t.Errorf("Expected Jane Doe, got %v %v", resp["first_name"], resp["last_name"])
				}
			},
		},
		{
			name:           "trimmed whitespace names",
			body:           `{"first_name":"  Bob  ","last_name":"  Jones  ","email":"user1@example.com"}`,
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["first_name"] != "Bob" || resp["last_name"] != "Jones" {
					t.Errorf("Expected trimmed names, got first=%v last=%v", resp["first_name"], resp["last_name"])
				}
			},
		},
		{
			name:           "phone number exceeding max length rejected",
			body:           `{"email":"user1@example.com","phone_number":"123456789012345678901"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid phone number accepted",
			body:           `{"email":"user1@example.com","phone_number":"555-123-4567"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupUserAdminTestContext(user.ID, false)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/me/profile", bytes.NewReader([]byte(tt.body)))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateCurrentUserProfile(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.checkFunc != nil && w.Code == tt.expectedStatus {
				tt.checkFunc(t, w)
			}
		})
	}
}

// TestResendInvitation tests the ResendInvitation handler
func TestResendInvitation(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		callerID       uint
		callerIsAdmin  bool
		setupUser      func(t *testing.T, db *gorm.DB, callerID uint) string // returns target user ID
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid user ID",
			userID:         "abc",
			callerID:       1,
			callerIsAdmin:  true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid user ID",
		},
		{
			name:           "user not found",
			userID:         "999",
			callerID:       1,
			callerIsAdmin:  true,
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:          "user already completed setup",
			callerIsAdmin: true,
			setupUser: func(t *testing.T, db *gorm.DB, callerID uint) string {
				user := &models.User{
					Username:              "setupdone",
					Email:                 "done@test.com",
					Password:              "hashed",
					RequiresPasswordSetup: false,
				}
				if err := db.Create(user).Error; err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				return fmt.Sprintf("%d", user.ID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "User has already set up their account",
		},
		{
			name:          "non-admin non-group-admin is forbidden",
			callerIsAdmin: false,
			setupUser: func(t *testing.T, db *gorm.DB, callerID uint) string {
				group := &models.Group{Name: "TestGroup"}
				db.Create(group)
				user := &models.User{
					Username:              "pending",
					Email:                 "pending@test.com",
					Password:              "hashed",
					RequiresPasswordSetup: true,
					Groups:                []models.Group{*group},
				}
				if err := db.Create(user).Error; err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				return fmt.Sprintf("%d", user.ID)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "You must be a site admin or group admin",
		},
		{
			name:          "group admin forbidden for user in different group",
			callerIsAdmin: false,
			setupUser: func(t *testing.T, db *gorm.DB, callerID uint) string {
				callerGroup := &models.Group{Name: "CallerGroup"}
				db.Create(callerGroup)
				db.Create(&models.UserGroup{UserID: callerID, GroupID: callerGroup.ID, IsGroupAdmin: true})
				targetGroup := &models.Group{Name: "TargetGroup"}
				db.Create(targetGroup)
				user := &models.User{
					Username:              "othergroup",
					Email:                 "other@test.com",
					Password:              "hashed",
					RequiresPasswordSetup: true,
					Groups:                []models.Group{*targetGroup},
				}
				if err := db.Create(user).Error; err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				return fmt.Sprintf("%d", user.ID)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "You must be a site admin or group admin",
		},
		{
			name:          "email not configured returns error after auth passes",
			callerIsAdmin: true,
			setupUser: func(t *testing.T, db *gorm.DB, callerID uint) string {
				user := &models.User{
					Username:              "noemail",
					Email:                 "noemail@test.com",
					Password:              "hashed",
					RequiresPasswordSetup: true,
				}
				if err := db.Create(user).Error; err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				return fmt.Sprintf("%d", user.ID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Email service is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)

			callerUser := &models.User{
				Username: "admin",
				Email:    "admin@test.com",
				Password: "hashed",
				IsAdmin:  tt.callerIsAdmin,
			}
			db.Create(callerUser)

			targetUserID := tt.userID
			if tt.setupUser != nil {
				targetUserID = tt.setupUser(t, db, callerUser.ID)
			}

			c, w := setupUserAdminTestContext(callerUser.ID, tt.callerIsAdmin)
			c.Params = gin.Params{{Key: "userId", Value: targetUserID}}
			c.Request = httptest.NewRequest(http.MethodPost, "/api/users/"+targetUserID+"/resend-invitation", nil)

			emailSvc := email.NewService(nil)
			handler := ResendInvitation(db, emailSvc)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedError != "" {
				var resp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
					if errMsg := resp["error"]; !strings.Contains(errMsg, tt.expectedError) {
						t.Errorf("Expected error containing %q, got %q", tt.expectedError, errMsg)
					}
				}
			}
		})
	}
}

// mockEmailProvider implements email.Provider for testing
type mockEmailProvider struct{}

func (m *mockEmailProvider) SendEmail(_ context.Context, _, _, _ string) error { return nil }
func (m *mockEmailProvider) IsConfigured() bool                                { return true }
func (m *mockEmailProvider) GetProviderName() string                           { return "mock" }

func TestResendInvitation_SiteAdminSuccess(t *testing.T) {
	db := setupUserAdminTestDB(t)

	admin := &models.User{Username: "admin", Email: "admin@test.com", Password: "hashed", IsAdmin: true}
	db.Create(admin)

	target := &models.User{
		Username:              "pending",
		Email:                 "pending@test.com",
		Password:              "hashed",
		RequiresPasswordSetup: true,
	}
	db.Create(target)

	c, w := setupUserAdminTestContext(admin.ID, true)
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}
	c.Request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/users/%d/resend-invitation", target.ID), nil)

	emailSvc := email.NewServiceWithProvider(&mockEmailProvider{}, db)
	handler := ResendInvitation(db, emailSvc)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify token was updated in DB
	var updated models.User
	db.First(&updated, target.ID)
	if updated.SetupToken == "" {
		t.Error("Expected setup token to be set after resend")
	}
	if updated.SetupTokenExpiry == nil || updated.SetupTokenExpiry.Before(time.Now()) {
		t.Error("Expected setup token expiry to be in the future")
	}
}

func TestResendInvitation_GroupAdminSuccess(t *testing.T) {
	db := setupUserAdminTestDB(t)

	groupAdmin := &models.User{Username: "gadmin", Email: "gadmin@test.com", Password: "hashed", IsAdmin: false}
	db.Create(groupAdmin)

	sharedGroup := &models.Group{Name: "SharedGroup"}
	db.Create(sharedGroup)
	db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: sharedGroup.ID, IsGroupAdmin: true})

	target := &models.User{
		Username:              "pending",
		Email:                 "pending@test.com",
		Password:              "hashed",
		RequiresPasswordSetup: true,
		Groups:                []models.Group{*sharedGroup},
	}
	db.Create(target)

	c, w := setupUserAdminTestContext(groupAdmin.ID, false)
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", target.ID)}}
	c.Request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/users/%d/resend-invitation", target.ID), nil)

	emailSvc := email.NewServiceWithProvider(&mockEmailProvider{}, db)
	handler := ResendInvitation(db, emailSvc)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestToAdminUserResponse — lockout field shadowing
// ---------------------------------------------------------------------------

// TestToAdminUserResponse verifies that LockedUntil and FailedLoginAttempts
// are present in the admin response and are not leaked through non-admin
// serialization (which uses models.User directly with json:"-" tags).
func TestToAdminUserResponse(t *testing.T) {
	now := time.Now().Add(30 * time.Minute)
	user := models.User{
		Username:            "testuser",
		Email:               "test@example.com",
		LockedUntil:         &now,
		FailedLoginAttempts: 5,
	}

	resp := toAdminUserResponse(user)

	// Lockout fields should be promoted to the outer struct
	if resp.LockedUntil == nil || !resp.LockedUntil.Equal(now) {
		t.Errorf("Expected LockedUntil to be %v, got %v", now, resp.LockedUntil)
	}
	if resp.FailedLoginAttempts != 5 {
		t.Errorf("Expected FailedLoginAttempts to be 5, got %d", resp.FailedLoginAttempts)
	}

	// Verify lockout fields render in JSON (admin response)
	adminJSON, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal admin response: %v", err)
	}
	adminStr := string(adminJSON)
	if !strings.Contains(adminStr, "locked_until") {
		t.Error("Expected 'locked_until' to appear in admin JSON response")
	}
	if !strings.Contains(adminStr, "failed_login_attempts") {
		t.Error("Expected 'failed_login_attempts' to appear in admin JSON response")
	}

	// Verify the base User model still hides lockout fields (json:"-")
	baseJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal base user: %v", err)
	}
	baseStr := string(baseJSON)
	if strings.Contains(baseStr, "locked_until") {
		t.Error("'locked_until' should NOT appear in base User JSON (non-admin response)")
	}
	if strings.Contains(baseStr, "failed_login_attempts") {
		t.Error("'failed_login_attempts' should NOT appear in base User JSON (non-admin response)")
	}
}

// ---------------------------------------------------------------------------
// TestUnlockUserAccount — handler authorization paths
// ---------------------------------------------------------------------------

func TestUnlockUserAccount(t *testing.T) {
	futureTime := time.Now().Add(15 * time.Minute)

	tests := []struct {
		name           string
		contextUserID  func(adminID, targetID uint) uint
		isAdmin        bool
		setupFunc      func(*testing.T, *gorm.DB) (actorID, targetID uint)
		userIDParam    func(targetID uint) string
		expectedStatus int
		checkFunc      func(*testing.T, *gorm.DB, uint)
	}{
		{
			name:    "site admin can unlock a locked user",
			isAdmin: true,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				actor := createUserAdminTestUser(t, db, "siteadmin", "sa@test.com", true)
				target := createUserAdminTestUser(t, db, "lockeduser", "locked@test.com", false)
				db.Model(target).Updates(map[string]interface{}{
					"locked_until":          &futureTime,
					"failed_login_attempts": 5,
				})
				return actor.ID, target.ID
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, targetID uint) {
				var u models.User
				db.First(&u, targetID)
				if u.LockedUntil != nil {
					t.Error("Expected LockedUntil to be nil after unlock")
				}
				if u.FailedLoginAttempts != 0 {
					t.Error("Expected FailedLoginAttempts to be 0 after unlock")
				}
			},
		},
		{
			name:    "site admin cannot unlock own account",
			isAdmin: true,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				actor := createUserAdminTestUser(t, db, "siteadmin", "sa@test.com", true)
				return actor.ID, actor.ID // target = self
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusForbidden,
			checkFunc:      nil,
		},
		{
			name:    "user not found returns 404",
			isAdmin: true,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				actor := createUserAdminTestUser(t, db, "siteadmin", "sa@test.com", true)
				return actor.ID, 99999
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusNotFound,
			checkFunc:      nil,
		},
		{
			name:    "soft-deleted user returns 400 with descriptive error",
			isAdmin: true,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				actor := createUserAdminTestUser(t, db, "siteadmin", "sa@test.com", true)
				target := createUserAdminTestUser(t, db, "deleteduser", "deleted@test.com", false)
				db.Delete(target) // soft delete
				return actor.ID, target.ID
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusBadRequest,
			checkFunc: func(t *testing.T, db *gorm.DB, _ uint) {
				// Verify the error body is descriptive
			},
		},
		{
			name:    "invalid user ID returns 400",
			isAdmin: true,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				actor := createUserAdminTestUser(t, db, "siteadmin", "sa@test.com", true)
				return actor.ID, 0
			},
			userIDParam:    func(_ uint) string { return "not-a-number" },
			expectedStatus: http.StatusBadRequest,
			checkFunc:      nil,
		},
		{
			name:    "group admin can unlock volunteer in shared group",
			isAdmin: false,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				groupAdmin := createUserAdminTestUser(t, db, "groupadmin", "ga@test.com", false)
				target := createUserAdminTestUser(t, db, "volunteer", "vol@test.com", false)

				group := &models.Group{Name: "TestGroup"}
				db.Create(group)
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				db.Create(&models.UserGroup{UserID: target.ID, GroupID: group.ID, IsGroupAdmin: false})

				// Assign groups via association so Preload picks them up
				db.Model(target).Association("Groups").Append(group)

				db.Model(target).Updates(map[string]interface{}{
					"locked_until":          &futureTime,
					"failed_login_attempts": 3,
				})
				return groupAdmin.ID, target.ID
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, db *gorm.DB, targetID uint) {
				var u models.User
				db.First(&u, targetID)
				if u.LockedUntil != nil {
					t.Error("Expected LockedUntil to be nil after group-admin unlock")
				}
			},
		},
		{
			name:    "group admin cannot unlock a site admin",
			isAdmin: false,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				groupAdmin := createUserAdminTestUser(t, db, "groupadmin", "ga@test.com", false)
				target := createUserAdminTestUser(t, db, "siteadmin2", "sa2@test.com", true)

				group := &models.Group{Name: "SharedGroup"}
				db.Create(group)
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				db.Model(target).Association("Groups").Append(group)

				return groupAdmin.ID, target.ID
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusForbidden,
			checkFunc:      nil,
		},
		{
			name:    "group admin cannot unlock user with no shared group",
			isAdmin: false,
			setupFunc: func(t *testing.T, db *gorm.DB) (uint, uint) {
				groupAdmin := createUserAdminTestUser(t, db, "groupadmin", "ga@test.com", false)
				target := createUserAdminTestUser(t, db, "unrelated", "unrelated@test.com", false)

				group := &models.Group{Name: "GroupAdminGroup"}
				db.Create(group)
				db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true})
				// target is not in any group with groupAdmin

				return groupAdmin.ID, target.ID
			},
			userIDParam:    func(targetID uint) string { return fmt.Sprintf("%d", targetID) },
			expectedStatus: http.StatusForbidden,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserAdminTestDB(t)

			actorID, targetID := tt.setupFunc(t, db)
			userIDParam := tt.userIDParam(targetID)

			c, w := setupUserAdminTestContext(actorID, tt.isAdmin)
			c.Params = gin.Params{{Key: "userId", Value: userIDParam}}
			c.Request = httptest.NewRequest(http.MethodPost, "/api/users/"+userIDParam+"/unlock", nil)

			handler := UnlockUserAccount(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, db, targetID)
			}
		})
	}
}
