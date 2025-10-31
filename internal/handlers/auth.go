package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"` // bcrypt limit is 72
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Register creates a new user account
func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if username or email already exists
		var existing models.User
		if err := db.WithContext(ctx).Where("username = ? OR email = ?", req.Username, req.Email).First(&existing).Error; err == nil {
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
			Username: req.Username,
			Email:    req.Email,
			Password: hashedPassword,
			IsAdmin:  false,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find user
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").Where("username = ?", req.Username).First(&user).Error; err != nil {
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

			// Lock account if 5 or more failed attempts
			if user.FailedLoginAttempts >= 5 {
				lockUntil := time.Now().Add(30 * time.Minute)
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
					"retry_in_mins": 30,
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

			attemptsRemaining := 5 - user.FailedLoginAttempts
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":              "Invalid credentials",
				"attempts_remaining": attemptsRemaining,
			})
			return
		}

		// Successful login - reset failed attempts
		if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
			if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
				"failed_login_attempts": 0,
				"locked_until":          nil,
			}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}
		}

		// Audit log: successful login
		logging.LogAuthSuccess(ctx, user.ID, user.Username, c.ClientIP())

		// Generate token
		token, err := auth.GenerateToken(user.ID, user.IsAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
