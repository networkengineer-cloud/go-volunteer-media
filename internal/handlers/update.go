package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type UpdateRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ImageURL string `json:"image_url"`
}

// GetUpdates returns all updates for a group
func GetUpdates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var updates []models.Update
		if err := db.Preload("User").Where("group_id = ?", groupID).Order("created_at DESC").Find(&updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updates"})
			return
		}

		c.JSON(http.StatusOK, updates)
	}
}

// CreateUpdate creates a new update/post in a group
func CreateUpdate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var req UpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		update := models.Update{
			GroupID:  uint(gid),
			UserID:   userID.(uint),
			Title:    req.Title,
			Content:  req.Content,
			ImageURL: req.ImageURL,
		}

		if err := db.Create(&update).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create update"})
			return
		}

		// Reload with user info
		if err := db.Preload("User").First(&update, update.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load update"})
			return
		}

		c.JSON(http.StatusCreated, update)
	}
}
