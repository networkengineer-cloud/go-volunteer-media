package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type AnimalCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ImageURL string `json:"image_url"`
	TagIDs   []uint `json:"tag_ids"` // Array of tag IDs to attach
}

// GetAnimalComments returns all comments for an animal
func GetAnimalComments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify animal exists and belongs to group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Get filter parameter (comma-separated tag names)
		tagFilter := c.Query("tags")

		query := db.Preload("User").Preload("Tags").Where("animal_id = ?", animalID)

		// Apply tag filter if provided
		if tagFilter != "" {
			tagNames := strings.Split(tagFilter, ",")
			query = query.Joins("JOIN comment_tags ON comment_tags.animal_comment_id = animal_comments.id").
				Joins("JOIN comment_tags AS ct ON ct.id = comment_tags.comment_tag_id").
				Where("ct.name IN ?", tagNames).
				Group("animal_comments.id")
		}

		var comments []models.AnimalComment
		if err := query.Order("created_at DESC").Find(&comments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
			return
		}

		c.JSON(http.StatusOK, comments)
	}
}

// CreateAnimalComment creates a new comment on an animal
func CreateAnimalComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify animal exists and belongs to group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		var req AnimalCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		aid, err := strconv.ParseUint(animalID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid animal ID"})
			return
		}

		comment := models.AnimalComment{
			AnimalID: uint(aid),
			UserID:   userID.(uint),
			Content:  req.Content,
			ImageURL: req.ImageURL,
		}

		if err := db.Create(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
			return
		}

		// Attach tags if provided
		if len(req.TagIDs) > 0 {
			var tags []models.CommentTag
			if err := db.Where("id IN ?", req.TagIDs).Find(&tags).Error; err == nil {
				if err := db.Model(&comment).Association("Tags").Append(&tags); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to attach tags"})
					return
				}
			}
		}

		// Reload with user info and tags
		if err := db.Preload("User").Preload("Tags").First(&comment, comment.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load comment"})
			return
		}

		c.JSON(http.StatusCreated, comment)
	}
}
