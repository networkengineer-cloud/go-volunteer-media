package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// PromoteUser sets is_admin to true for a user
func PromoteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if user.IsAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is already admin"})
			return
		}
		if err := db.WithContext(ctx).Model(&user).Update("is_admin", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User promoted to admin"})
	}
}

// DemoteUser sets is_admin to false for a user
func DemoteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if !user.IsAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not admin"})
			return
		}
		if err := db.WithContext(ctx).Model(&user).Update("is_admin", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User demoted from admin"})
	}
}

// GetDeletedUsers returns all soft-deleted users (admin only)
func GetDeletedUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var users []models.User
		if err := db.WithContext(ctx).Unscoped().Preload("Groups").Where("deleted_at IS NOT NULL").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch deleted users"})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

// RestoreUser restores a soft-deleted user (admin only)
func RestoreUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).Unscoped().First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if user.DeletedAt.Valid {
			if err := db.WithContext(ctx).Unscoped().Model(&user).Update("deleted_at", nil).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore user"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "User restored"})
	}
}

// AdminDeleteUser soft-deletes (deactivates) a user (marks as deleted, disables login)
func AdminDeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if err := db.WithContext(ctx).Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
	}
}

// (removed duplicate import block)

type AdminCreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,usernamechars"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	IsAdmin  bool   `json:"is_admin"`
	GroupIDs []uint `json:"group_ids"`
}

type AdminResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// AdminCreateUser allows an admin to create a new user
func AdminCreateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req AdminCreateUserRequest
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

		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		user := models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: hashedPassword,
			IsAdmin:  req.IsAdmin,
		}

		// If group IDs are provided, fetch and associate groups
		if len(req.GroupIDs) > 0 {
			var groups []models.Group
			if err := db.WithContext(ctx).Where("id IN ?", req.GroupIDs).Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
			user.Groups = groups
		}

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Preload groups for response
		if err := db.WithContext(ctx).Preload("Groups").First(&user, user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user groups"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

// AdminResetUserPassword allows an admin to reset a user's password
func AdminResetUserPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		
		var req AdminResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find the user
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Hash the new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Update password and clear any lockouts or reset tokens
		if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
			"password":              hashedPassword,
			"reset_token":           "",
			"reset_token_expiry":    nil,
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
	}
}
