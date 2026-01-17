package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
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
	err = db.AutoMigrate(&models.User{}, &models.Group{})
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
