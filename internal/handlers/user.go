package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// ErrEmailInUse is returned when an email is already in use by another user.
var ErrEmailInUse = errors.New("email address is already in use")

// validateEmailUniqueness checks if an email is already in use by another user
func validateEmailUniqueness(ctx context.Context, db *gorm.DB, email string, currentUserID uint) error {
	var existingUser models.User
	err := db.WithContext(ctx).Where("email = ? AND id != ?", email, currentUserID).First(&existingUser).Error
	if err == nil {
		return ErrEmailInUse
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("database error checking email uniqueness: %w", err)
	}
	return nil
}

// GetAllUsers returns all users with pagination support (admin only)
func GetAllUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get pagination parameters
		limit := 20 // Default limit for users (consistent with statistics endpoints)
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100 // Max 100 per page
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get total count
		var total int64
		if err := db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
			return
		}

		// Get users with pagination
		var users []models.User
		if err := db.WithContext(ctx).
			Preload("Groups").
			Limit(limit).
			Offset(offset).
			Order("created_at DESC").
			Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    users,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": offset+len(users) < int(total),
		})
	}
}

type SetDefaultGroupRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
}

// SetDefaultGroup sets the user's default group
func SetDefaultGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var req SetDefaultGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify user has access to the group
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Check if user belongs to the group (unless admin)
		isAdmin, _ := c.Get("is_admin")
		hasAccess := isAdmin.(bool)
		if !hasAccess {
			for _, group := range user.Groups {
				if group.ID == req.GroupID {
					hasAccess = true
					break
				}
			}
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this group"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, req.GroupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Update user's default group
		if err := db.WithContext(ctx).Model(&user).Update("default_group_id", req.GroupID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update default group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Default group updated successfully", "default_group_id": req.GroupID})
	}
}

// GetDefaultGroup returns the user's default group details
func GetDefaultGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		if user.DefaultGroupID == nil {
			c.JSON(http.StatusOK, gin.H{"default_group_id": nil})
			return
		}

		var group models.Group
		if err := db.WithContext(ctx).First(&group, *user.DefaultGroupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Default group not found"})
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

type UpdateProfileRequest struct {
	FirstName       string `json:"first_name" binding:"omitempty,max=100"`
	LastName        string `json:"last_name" binding:"omitempty,max=100"`
	Email           string `json:"email" binding:"required,email"`
	PhoneNumber     string `json:"phone_number"`
	HideEmail       bool   `json:"hide_email"`
	HidePhoneNumber bool   `json:"hide_phone_number"`
}

// UpdateCurrentUserProfile allows users to update their own profile information
func UpdateCurrentUserProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		var req UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Fetch current user to check if email is being changed
		var user models.User
		if err := db.WithContext(ctx).First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Check if email is being changed to an already-taken email
		if req.Email != user.Email {
			if err := validateEmailUniqueness(ctx, db, req.Email, userID.(uint)); err != nil {
				if errors.Is(err, ErrEmailInUse) {
					c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate email"})
				}
				return
			}
		}

		// Update user profile (first name, last name, email, phone, and privacy settings)
		updates := map[string]interface{}{
			"first_name":        req.FirstName,
			"last_name":         req.LastName,
			"email":             req.Email,
			"phone_number":      req.PhoneNumber,
			"hide_email":        req.HideEmail,
			"hide_phone_number": req.HidePhoneNumber,
		}
		if err := db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":           "Profile updated successfully",
			"id":                user.ID,
			"first_name":        req.FirstName,
			"last_name":         req.LastName,
			"email":             req.Email,
			"phone_number":      req.PhoneNumber,
			"hide_email":        req.HideEmail,
			"hide_phone_number": req.HidePhoneNumber,
		})
	}
}

// GetPrivacyPreferences returns the current user's privacy settings
func GetPrivacyPreferences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"hide_email":        user.HideEmail,
			"hide_phone_number": user.HidePhoneNumber,
		})
	}
}
