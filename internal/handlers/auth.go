package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50,usernamechars"`
	FirstName string `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName  string `json:"last_name" binding:"omitempty,min=1,max=100"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=72"` // bcrypt limit is 72
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string      `json:"token"`
	User      models.User `json:"user"`
	LastLogin *time.Time  `json:"last_login,omitempty"`
}

// Register creates a new user account
func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Normalize username to lowercase
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))

		// Check if username or email already exists (case-insensitive username check)
		var existing models.User
		if err := db.WithContext(ctx).Where("LOWER(username) = ? OR email = ?", req.Username, req.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Create user
		user := models.User{
			Username:  req.Username,
			FirstName: strings.TrimSpace(req.FirstName),
			LastName:  strings.TrimSpace(req.LastName),
			Email:     req.Email,
			Password:  hashedPassword,
			IsAdmin:   false,
		}

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Audit log: user registration
		logging.LogRegistration(ctx, user.ID, user.Username, user.Email, c.ClientIP())

		// Generate token
		token, err := auth.GenerateToken(user.ID, user.IsAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusCreated, AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

// Login authenticates a user and returns a token
func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Find user (case-insensitive username match)
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups", func(db *gorm.DB) *gorm.DB {
			return db.Where("groups.deleted_at IS NULL")
		}).Where("LOWER(username) = ?", strings.ToLower(req.Username)).First(&user).Error; err != nil {
			// Audit log: failed login attempt (user not found)
			logging.LogAuthFailure(ctx, req.Username, c.ClientIP(), "user_not_found")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Check if account is locked
		if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
			remainingTime := time.Until(*user.LockedUntil)
			minutes := int(remainingTime.Minutes())
			// Audit log: attempt to access locked account
			logging.LogAuthFailure(ctx, req.Username, c.ClientIP(), "account_locked")
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Account is temporarily locked due to too many failed login attempts",
				"locked_until":  user.LockedUntil,
				"retry_in_mins": minutes + 1,
			})
			return
		}

		// Check if account requires password setup (new invite-only user)
		if user.RequiresPasswordSetup {
			logging.LogAuthFailure(ctx, req.Username, c.ClientIP(), "password_setup_required")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your account requires password setup. Please check your email for the setup link, or contact an administrator for a new invitation.",
			})
			return
		}

		// If lock period has expired, reset failed attempts
		if user.LockedUntil != nil && user.LockedUntil.Before(time.Now()) {
			user.FailedLoginAttempts = 0
			user.LockedUntil = nil
			if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
				"failed_login_attempts": 0,
				"locked_until":          nil,
			}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}
		}

		// Check password
		if err := auth.CheckPassword(user.Password, req.Password); err != nil {
			// Increment failed login attempts
			user.FailedLoginAttempts++

			// Lock account if max failed attempts reached
			if user.FailedLoginAttempts >= MaxFailedLoginAttempts {
				lockUntil := time.Now().Add(AccountLockoutDuration)
				user.LockedUntil = &lockUntil

				if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
					"failed_login_attempts": user.FailedLoginAttempts,
					"locked_until":          lockUntil,
				}).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
					return
				}

				// Audit log: account locked
				logging.LogAccountLocked(ctx, user.ID, user.Username, c.ClientIP(), user.FailedLoginAttempts)

				c.JSON(http.StatusForbidden, gin.H{
					"error":         "Account has been locked due to too many failed login attempts. Please try again in 30 minutes or reset your password.",
					"locked_until":  lockUntil,
					"retry_in_mins": int(AccountLockoutDuration.Minutes()),
				})
				return
			}

			// Update failed attempts count
			if err := db.WithContext(ctx).Model(&user).Update("failed_login_attempts", user.FailedLoginAttempts).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}

			// Audit log: failed login attempt
			logging.LogAuthFailure(ctx, req.Username, c.ClientIP(), "invalid_password")

			attemptsRemaining := MaxFailedLoginAttempts - user.FailedLoginAttempts
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":              "Invalid credentials",
				"attempts_remaining": attemptsRemaining,
			})
			return
		}

		// Successful login - record last login timestamp and reset failed attempts if needed
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"last_login": now,
		}
		if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
			updates["failed_login_attempts"] = 0
			updates["locked_until"] = nil
			user.FailedLoginAttempts = 0
			user.LockedUntil = nil
		}
		if err := db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
		user.LastLogin = &now

		// Audit log: successful login
		logging.LogAuthSuccess(ctx, user.ID, user.Username, c.ClientIP())

		// Generate token
		token, err := auth.GenerateToken(user.ID, user.IsAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			Token:     token,
			User:      user,
			LastLogin: user.LastLogin,
		})
	}
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).Preload("Groups", func(db *gorm.DB) *gorm.DB {
			return db.Where("groups.deleted_at IS NULL")
		}).First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Check if user is a group admin of any group
		var userGroups []models.UserGroup
		db.WithContext(ctx).Where("user_id = ? AND is_group_admin = ?", userID, true).Find(&userGroups)

		// Add is_group_admin flag to response
		response := map[string]interface{}{
			"id":                          user.ID,
			"username":                    user.Username,
			"first_name":                  user.FirstName,
			"last_name":                   user.LastName,
			"email":                       user.Email,
			"phone_number":                user.PhoneNumber,
			"hide_email":                  user.HideEmail,
			"hide_phone_number":           user.HidePhoneNumber,
			"is_admin":                    user.IsAdmin,
			"default_group_id":            user.DefaultGroupID,
			"groups":                      user.Groups,
			"email_notifications_enabled": user.EmailNotificationsEnabled,
			"is_group_admin":              len(userGroups) > 0,
			"created_at":                  user.CreatedAt,
			"updated_at":                  user.UpdatedAt,
			"last_login":                  user.LastLogin,
		}

		c.JSON(http.StatusOK, response)
	}
}
