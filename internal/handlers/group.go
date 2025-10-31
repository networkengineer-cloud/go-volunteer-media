package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type GroupRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=100"`
	Description  string `json:"description" binding:"max=500"`
	ImageURL     string `json:"image_url,omitempty"`
	HeroImageURL string `json:"hero_image_url,omitempty"`
}

// UploadGroupImage handles secure group image uploads (admin only)
func UploadGroupImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("image")
		if err != nil {
			log.Printf("Failed to get form file: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded: " + err.Error()})
			return
		}

		// Only allow jpg, jpeg, png, gif
		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
		if !allowed[ext] {
			log.Printf("Invalid file type: %s", ext)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
			return
		}

		// Generate unique filename
		fname := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), ext)
		uploadPath := filepath.Join("public", "uploads", fname)

		log.Printf("Attempting to save file to: %s", uploadPath)

		// Save file
		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			log.Printf("Failed to save file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
			return
		}

		// Return public URL
		url := "/uploads/" + fname
		log.Printf("File uploaded successfully: %s", url)
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

// GetGroups returns all groups the user has access to
func GetGroups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		isAdmin, exists := c.Get("is_admin")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin context not found"})
			return
		}

		var groups []models.Group

		adminFlag, ok := isAdmin.(bool)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid admin flag"})
			return
		}
		if adminFlag {
			// Admins can see all groups
			if err := db.WithContext(ctx).Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
		} else {
			// Regular users see only their groups
			var user models.User
			if err := db.WithContext(ctx).Preload("Groups").First(&user, userID).Error; err != nil {
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
			Name:         req.Name,
			Description:  req.Description,
			ImageURL:     req.ImageURL,
			HeroImageURL: req.HeroImageURL,
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
		group.ImageURL = req.ImageURL
		group.HeroImageURL = req.HeroImageURL

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
