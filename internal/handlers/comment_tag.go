package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type CommentTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// GetCommentTags returns all comment tags for a specific group
// Route: GET /api/groups/:id/comment-tags
func GetCommentTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access - user must be member of the group
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var tags []models.CommentTag
		if err := db.Where("group_id = ?", groupID).Order("is_system DESC, name ASC").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
			return
		}
		c.JSON(http.StatusOK, tags)
	}
}

// CreateCommentTag creates a new comment tag for a specific group (group admin or site admin only)
// Route: POST /api/groups/:id/comment-tags
func CreateCommentTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can create tags"})
			return
		}

		var req CommentTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		groupIDUint, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		tag := models.CommentTag{
			GroupID:  uint(groupIDUint),
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

// DeleteCommentTag deletes a comment tag (group admin or site admin only, cannot delete system tags)
// Route: DELETE /api/groups/:id/comment-tags/:tagId
func DeleteCommentTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		tagID := c.Param("tagId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can delete tags"})
			return
		}

		// Ensure the tag belongs to this group
		var tag models.CommentTag
		if err := db.Where("id = ? AND group_id = ?", tagID, groupID).First(&tag).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found in this group"})
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
