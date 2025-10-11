package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type GroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// GetGroups returns all groups the user has access to
func GetGroups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		var groups []models.Group
		
		if isAdmin.(bool) {
			// Admins can see all groups
			if err := db.Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
		} else {
			// Regular users see only their groups
			var user models.User
			if err := db.Preload("Groups").First(&user, userID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user groups"})
				return
			}
			groups = user.Groups
		}

		c.JSON(http.StatusOK, groups)
	}
}

// GetGroup returns a specific group by ID
func GetGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user has access to this group
		if !isAdmin.(bool) {
			var user models.User
			if err := db.Preload("Groups", "id = ?", groupID).First(&user, userID).Error; err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
			if len(user.Groups) == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
		}

		c.JSON(http.StatusOK, group)
	}
}

// CreateGroup creates a new group (admin only)
func CreateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		group := models.Group{
			Name:        req.Name,
			Description: req.Description,
		}

		if err := db.Create(&group).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
			return
		}

		c.JSON(http.StatusCreated, group)
	}
}

// UpdateGroup updates an existing group (admin only)
func UpdateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		var req GroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		group.Name = req.Name
		group.Description = req.Description

		if err := db.Save(&group).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

// DeleteGroup deletes a group (admin only)
func DeleteGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")

		if err := db.Delete(&models.Group{}, groupID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
	}
}

// AddUserToGroup adds a user to a group (admin only)
func AddUserToGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var user models.User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		var group models.Group
		if err := db.First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if err := db.Model(&user).Association("Groups").Append(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User added to group successfully"})
	}
}

// RemoveUserFromGroup removes a user from a group (admin only)
func RemoveUserFromGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var user models.User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		var group models.Group
		if err := db.First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if err := db.Model(&user).Association("Groups").Delete(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User removed from group successfully"})
	}
}
