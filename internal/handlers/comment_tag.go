package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type CommentTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// GetCommentTags returns all comment tags
func GetCommentTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tags []models.CommentTag
		if err := db.Order("is_system DESC, name ASC").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
			return
		}
		c.JSON(http.StatusOK, tags)
	}
}

// CreateCommentTag creates a new comment tag (admin only)
func CreateCommentTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CommentTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tag := models.CommentTag{
			Name:     req.Name,
			Color:    req.Color,
			IsSystem: false, // Custom tags are never system tags
		}

		if tag.Color == "" {
			tag.Color = "#6b7280" // Default gray color
		}

		if err := db.Create(&tag).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
			return
		}

		c.JSON(http.StatusCreated, tag)
	}
}

// DeleteCommentTag deletes a comment tag (admin only, cannot delete system tags)
func DeleteCommentTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagID := c.Param("tagId")

		var tag models.CommentTag
		if err := db.First(&tag, tagID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
			return
		}

		if tag.IsSystem {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system tags"})
			return
		}

		if err := db.Delete(&tag).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tag"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
	}
}
