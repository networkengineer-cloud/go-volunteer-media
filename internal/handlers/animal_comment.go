package handlers

import (
	"errors"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type AnimalCommentRequest struct {
	Content  string                  `json:"content" binding:"required"`
	ImageURL string                  `json:"image_url"`
	TagIDs   []uint                  `json:"tag_ids"`  // Array of tag IDs to attach
	Metadata *models.SessionMetadata `json:"metadata"` // Optional structured session data
}

// validateSessionMetadata validates the structured session metadata field lengths
func validateSessionMetadata(metadata *models.SessionMetadata) error {
	if metadata == nil {
		return nil
	}

	if len(metadata.SessionGoal) > 200 {
		return errors.New("session goal exceeds 200 character limit")
	}
	if len(metadata.SessionOutcome) > 2000 {
		return errors.New("session outcome exceeds 2000 character limit")
	}
	if len(metadata.BehaviorNotes) > 1000 {
		return errors.New("behavior notes exceed 1000 character limit")
	}
	if len(metadata.MedicalNotes) > 1000 {
		return errors.New("medical notes exceed 1000 character limit")
	}
	if len(metadata.OtherNotes) > 1000 {
		return errors.New("other notes exceed 1000 character limit")
	}
	if metadata.SessionRating < 0 || metadata.SessionRating > 5 {
		return errors.New("session rating must be between 1 and 5 (or 0 for not set)")
	}

	return nil
}

// sanitizeSessionMetadata sanitizes all text fields in metadata to prevent XSS attacks
func sanitizeSessionMetadata(metadata *models.SessionMetadata) {
	if metadata == nil {
		return
	}

	// HTML escape all text fields to prevent XSS
	metadata.SessionGoal = html.EscapeString(metadata.SessionGoal)
	metadata.SessionOutcome = html.EscapeString(metadata.SessionOutcome)
	metadata.BehaviorNotes = html.EscapeString(metadata.BehaviorNotes)
	metadata.MedicalNotes = html.EscapeString(metadata.MedicalNotes)
	metadata.OtherNotes = html.EscapeString(metadata.OtherNotes)
}

// GetAnimalComments returns comments for an animal with pagination support
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

		// Get pagination parameters
		limit := 10 // Default limit
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

		// Get sort order (default: DESC for newest first)
		sortOrder := "DESC"
		if order := c.Query("order"); order == "asc" || order == "ASC" {
			sortOrder = "ASC"
		}

		// Get filter parameter (comma-separated tag names)
		tagFilter := c.Query("tags")

		query := db.Preload("User").Preload("Tags").Where("animal_id = ?", animalID)

		// Apply tag filter if provided (multiple tags = OR logic)
		if tagFilter != "" {
			tagNames := strings.Split(tagFilter, ",")
			// Trim whitespace from tag names
			for i, name := range tagNames {
				tagNames[i] = strings.TrimSpace(name)
			}

			query = query.Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
				Joins("JOIN comment_tags ON comment_tags.id = animal_comment_tags.comment_tag_id").
				Where("comment_tags.name IN ?", tagNames).
				Group("animal_comments.id")
		}

		// Get total count
		var total int64
		countQuery := db.Model(&models.AnimalComment{}).Where("animal_id = ?", animalID)
		if tagFilter != "" {
			tagNames := strings.Split(tagFilter, ",")
			for i, name := range tagNames {
				tagNames[i] = strings.TrimSpace(name)
			}
			countQuery = countQuery.Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
				Joins("JOIN comment_tags ON comment_tags.id = animal_comment_tags.comment_tag_id").
				Where("comment_tags.name IN ?", tagNames).
				Group("animal_comments.id")
		}
		if err := countQuery.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count comments"})
			return
		}

		var comments []models.AnimalComment
		if err := query.Order("created_at " + sortOrder).Limit(limit).Offset(offset).Find(&comments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
			return
		}

		// Return paginated response
		c.JSON(http.StatusOK, gin.H{
			"comments": comments,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"hasMore":  offset+len(comments) < int(total),
		})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Validate metadata if provided
		if err := validateSessionMetadata(req.Metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Sanitize metadata to prevent XSS
		sanitizeSessionMetadata(req.Metadata)

		aid, err := strconv.ParseUint(animalID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid animal ID"})
			return
		}

		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		comment := models.AnimalComment{
			AnimalID: uint(aid),
			UserID:   userIDUint,
			Content:  req.Content,
			ImageURL: req.ImageURL,
			Metadata: req.Metadata,
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

// UpdateAnimalComment updates a comment on an animal
// Users can only edit their own comments
func UpdateAnimalComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		commentID := c.Param("commentId")
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

		// Get the comment
		var comment models.AnimalComment
		if err := db.Where("id = ? AND animal_id = ?", commentID, animalID).First(&comment).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}

		// Users can only edit their own comments
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		if comment.UserID != userIDUint {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own comments"})
			return
		}

		var req AnimalCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Validate metadata if provided
		if err := validateSessionMetadata(req.Metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Sanitize metadata to prevent XSS
		sanitizeSessionMetadata(req.Metadata)

		// Save current version to history before updating
		// EditedBy records who authored this version (which is now being replaced)
		// On first edit: comment.UserID (original author)
		// On subsequent edits: comment.UserID (previous editor who is now in history)
		history := models.CommentHistory{
			CommentID: comment.ID,
			Content:   comment.Content,
			ImageURL:  comment.ImageURL,
			Metadata:  comment.Metadata,
			EditedBy:  comment.UserID,
		}
		if err := db.Create(&history).Error; err != nil {
			// Log error but don't fail the update
			log.Printf("Failed to save comment history: %v", err)
		}

		// Update comment fields
		comment.Content = req.Content
		comment.ImageURL = req.ImageURL
		comment.Metadata = req.Metadata

		if err := db.Save(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
			return
		}

		// Update tags if provided
		if len(req.TagIDs) > 0 {
			var tags []models.CommentTag
			if err := db.Where("id IN ?", req.TagIDs).Find(&tags).Error; err == nil {
				if err := db.Model(&comment).Association("Tags").Replace(&tags); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tags"})
					return
				}
			}
		}

		// Reload with user info and tags
		if err := db.Preload("User").Preload("Tags").First(&comment, comment.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load comment"})
			return
		}

		c.JSON(http.StatusOK, comment)
	}
}

// GetCommentHistory returns the edit history for a comment (admin only)
func GetCommentHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		commentID := c.Param("commentId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Verify comment belongs to the specified animal and group
		var comment models.AnimalComment
		err := db.Joins("JOIN animals ON animals.id = animal_comments.animal_id").
			Where("animal_comments.id = ? AND animal_comments.animal_id = ? AND animals.group_id = ?", commentID, animalID, groupID).
			First(&comment).Error
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}

		// Get history entries
		var history []models.CommentHistory
		err = db.Where("comment_id = ?", commentID).
			Preload("User").
			Order("created_at DESC").
			Find(&history).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}

		c.JSON(http.StatusOK, history)
	}
}

// GetGroupLatestComments returns the latest comments across all animals in a group
func GetGroupLatestComments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Get limit parameter (default 20, max 100)
		limit := 20
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100
				}
			}
		}

		// Get animals in this group first
		var animals []models.Animal
		if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		// Get animal IDs
		var animalIDs []uint
		animalMap := make(map[uint]models.Animal)
		for _, animal := range animals {
			animalIDs = append(animalIDs, animal.ID)
			animalMap[animal.ID] = animal
		}

		if len(animalIDs) == 0 {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}

		// Get latest comments from these animals
		var comments []models.AnimalComment
		err := db.WithContext(ctx).
			Where("animal_id IN ?", animalIDs).
			Preload("User").
			Preload("Tags").
			Order("created_at DESC").
			Limit(limit).
			Find(&comments).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
			return
		}

		// Build response with animal information
		type CommentWithAnimal struct {
			models.AnimalComment
			Animal models.Animal `json:"animal"`
		}

		var results []CommentWithAnimal
		for _, comment := range comments {
			if animal, ok := animalMap[comment.AnimalID]; ok {
				results = append(results, CommentWithAnimal{
					AnimalComment: comment,
					Animal:        animal,
				})
			}
		}

		c.JSON(http.StatusOK, results)
	}
}

// DeleteAnimalComment deletes a comment (soft delete)
// Users can delete their own comments, admins can delete any comment
func DeleteAnimalComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		commentID := c.Param("commentId")
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

		// Get the comment
		var comment models.AnimalComment
		if err := db.Where("id = ? AND animal_id = ?", commentID, animalID).First(&comment).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}

		// Check if user owns the comment, is group admin, or is site admin
		isGroupAdmin := checkGroupAdminAccess(db, userID, isAdmin, groupID)
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		if comment.UserID != userIDUint && !isGroupAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
			return
		}

		// Soft delete the comment
		if err := db.Delete(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
	}
}

// GetDeletedComments returns all soft-deleted comments (group admin or site admin)
func GetDeletedComments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Get animals in this group
		var animals []models.Animal
		if err := db.Where("group_id = ?", groupID).Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		var animalIDs []uint
		animalMap := make(map[uint]models.Animal)
		for _, animal := range animals {
			animalIDs = append(animalIDs, animal.ID)
			animalMap[animal.ID] = animal
		}

		if len(animalIDs) == 0 {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}

		// Get deleted comments (unscoped to include soft-deleted)
		var comments []models.AnimalComment
		err := db.Unscoped().
			Where("animal_id IN ? AND deleted_at IS NOT NULL", animalIDs).
			Preload("User").
			Preload("Tags").
			Order("deleted_at DESC").
			Find(&comments).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch deleted comments"})
			return
		}

		// Build response with animal information
		type DeletedCommentWithAnimal struct {
			models.AnimalComment
			Animal models.Animal `json:"animal"`
		}

		var results []DeletedCommentWithAnimal
		for _, comment := range comments {
			if animal, ok := animalMap[comment.AnimalID]; ok {
				results = append(results, DeletedCommentWithAnimal{
					AnimalComment: comment,
					Animal:        animal,
				})
			}
		}

		c.JSON(http.StatusOK, results)
	}
}
