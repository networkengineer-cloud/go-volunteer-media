package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type UpdateEmailPreferencesRequest struct {
	EmailNotificationsEnabled bool `json:"email_notifications_enabled"`
	ShowLengthOfStay          bool `json:"show_length_of_stay"`
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RequestPasswordReset sends a password reset email
func RequestPasswordReset(db *gorm.DB, emailService *email.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req RequestPasswordResetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if email service is configured
		if !emailService.IsConfigured() {
			logger := middleware.GetLogger(c)
			logger.Warn("Password reset requested but email service is not configured")
			// Return success anyway to prevent email enumeration
			c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link will be sent"})
			return
		}

		// Find user by email
		var user models.User
		if err := db.WithContext(ctx).Where("email = ?", req.Email).First(&user).Error; err != nil {
			// Don't reveal if email exists - return success anyway
			c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link will be sent"})
			return
		}

		// Generate secure reset token
		token, err := generateSecureToken()
		if err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to generate reset token", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
			return
		}

		// Hash the token before storing
		hashedToken, err := auth.HashPassword(token)
		if err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to hash reset token", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process reset token"})
			return
		}

		// Set token expiry to 1 hour from now
		expiry := time.Now().Add(1 * time.Hour)

		// Update user with reset token and expiry
		if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
			"reset_token":        hashedToken,
			"reset_token_expiry": expiry,
		}).Error; err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to update user with reset token", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process reset request"})
			return
		}

		// Send password reset email (use unhashed token in email)
		if err := emailService.SendPasswordResetEmail(user.Email, user.Username, token); err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to send password reset email", err)
			// Still return success to prevent email enumeration
		} else {
			// Log successful password reset request (audit log already exists via LogPasswordResetRequest)
			logging.LogPasswordResetRequest(ctx, req.Email, c.ClientIP())
		}

		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link will be sent"})
	}
}

// ResetPassword resets the user's password using the reset token
func ResetPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find all users with a reset token (we'll check which one matches)
		var users []models.User
		if err := db.WithContext(ctx).Where("reset_token IS NOT NULL AND reset_token != ''").Find(&users).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}

		// Find the user whose hashed token matches
		var targetUser *models.User
		for i := range users {
			if err := auth.CheckPassword(users[i].ResetToken, req.Token); err == nil {
				targetUser = &users[i]
				break
			}
		}

		if targetUser == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}

		// Check if token has expired
		if targetUser.ResetTokenExpiry == nil || targetUser.ResetTokenExpiry.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reset token has expired. Please request a new one."})
			return
		}

		// Hash new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Update password and clear reset token
		if err := db.WithContext(ctx).Model(targetUser).Updates(map[string]interface{}{
			"password":              hashedPassword,
			"reset_token":           "",
			"reset_token_expiry":    nil,
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
	}
}

// SetupPassword allows a new user to set their password using a setup token (invite flow)
// This is separate from ResetPassword to prevent token confusion and add proper validation
func SetupPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req ResetPasswordRequest // Reuse same request structure
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find all users with a setup token (we'll check which one matches)
		var users []models.User
		if err := db.WithContext(ctx).Where("setup_token IS NOT NULL AND setup_token != ''").Find(&users).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired setup token"})
			return
		}

		// Find the user whose hashed setup token matches
		var targetUser *models.User
		for i := range users {
			if err := auth.CheckPassword(users[i].SetupToken, req.Token); err == nil {
				targetUser = &users[i]
				break
			}
		}

		if targetUser == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired setup token. Please contact your administrator for a new invitation."})
			return
		}

		// Check if token has expired (7-day expiry for setup)
		if targetUser.SetupTokenExpiry == nil || targetUser.SetupTokenExpiry.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Setup token has expired. Please contact your administrator for a new invitation."})
			return
		}

		// Verify this is actually a new account requiring setup
		if !targetUser.RequiresPasswordSetup {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This account has already been set up. Please use the password reset flow instead."})
			return
		}

		// Hash new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Update password, clear setup token, and mark account as fully set up
		if err := db.WithContext(ctx).Model(targetUser).Updates(map[string]interface{}{
			"password":                  hashedPassword,
			"setup_token":               "",
			"setup_token_expiry":        nil,
			"requires_password_setup":   false, // Allow login now
			"failed_login_attempts":     0,
			"locked_until":              nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set up password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password has been set successfully! You can now log in."})
	}
}

// UpdateEmailPreferences updates the user's email notification preferences
func UpdateEmailPreferences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var req UpdateEmailPreferencesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update preferences
		updates := map[string]interface{}{
			"email_notifications_enabled": req.EmailNotificationsEnabled,
			"show_length_of_stay":         req.ShowLengthOfStay,
		}
		if err := db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":                     "Preferences updated successfully",
			"email_notifications_enabled": req.EmailNotificationsEnabled,
			"show_length_of_stay":         req.ShowLengthOfStay,
		})
	}
}

// GetEmailPreferences returns the user's email notification preferences
func GetEmailPreferences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).Select("email_notifications_enabled, show_length_of_stay").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"email_notifications_enabled": user.EmailNotificationsEnabled,
			"show_length_of_stay":         user.ShowLengthOfStay,
		})
	}
}
