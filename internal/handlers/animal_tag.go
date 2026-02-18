package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// AnimalTagRequest represents a request to create or update an animal tag
type AnimalTagRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=50"`
	Category string `json:"category" binding:"required,oneof=behavior walker_status"`
	Color    string `json:"color" binding:"required"`
}

// GetAnimalTags returns all animal tags for a specific group
// Route: GET /api/groups/:id/animal-tags
func GetAnimalTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access - user must be member of the group
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var tags []models.AnimalTag
		if err := db.Where("group_id = ?", groupID).Order("category, name").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animal tags"})
			return
		}

		c.JSON(http.StatusOK, tags)
	}
}

// CreateAnimalTag creates a new animal tag for a specific group (group admin or site admin only)
// Route: POST /api/groups/:id/animal-tags
func CreateAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can create tags"})
			return
		}

		var req AnimalTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		groupIDUint, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		tag := models.AnimalTag{
			GroupID:  uint(groupIDUint),
			Name:     req.Name,
			Category: req.Category,
			Color:    req.Color,
		}

		if err := db.Create(&tag).Error; err != nil {
			logger.Error("Failed to create animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal tag"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"tag_id":   tag.ID,
			"tag_name": tag.Name,
			"group_id": groupID,
		}).Info("Created animal tag for group")

		c.JSON(http.StatusCreated, tag)
	}
}

// UpdateAnimalTag updates an existing animal tag (group admin or site admin only)
// Route: PUT /api/groups/:id/animal-tags/:tagId
func UpdateAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		tagID := c.Param("tagId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can update tags"})
			return
		}

		var req AnimalTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		var tag models.AnimalTag
		// Ensure the tag belongs to this group
		if err := db.Where("id = ? AND group_id = ?", tagID, groupID).First(&tag).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal tag not found in this group"})
			return
		}

		tag.Name = req.Name
		tag.Category = req.Category
		tag.Color = req.Color

		if err := db.Save(&tag).Error; err != nil {
			logger.Error("Failed to update animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal tag"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"tag_id":   tag.ID,
			"tag_name": tag.Name,
			"group_id": groupID,
		}).Info("Updated animal tag")

		c.JSON(http.StatusOK, tag)
	}
}

// DeleteAnimalTag deletes an animal tag (group admin or site admin only)
// Route: DELETE /api/groups/:id/animal-tags/:tagId
func DeleteAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		tagID := c.Param("tagId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can delete tags"})
			return
		}

		// Ensure the tag belongs to this group before deleting
		var tag models.AnimalTag
		if err := db.Where("id = ? AND group_id = ?", tagID, groupID).First(&tag).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal tag not found in this group"})
			return
		}

		if err := db.Delete(&tag).Error; err != nil {
			logger.Error("Failed to delete animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete animal tag"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"tag_id":   tagID,
			"group_id": groupID,
		}).Info("Deleted animal tag")
		c.JSON(http.StatusOK, gin.H{"message": "Animal tag deleted successfully"})
	}
}

// AssignTagsToAnimal assigns tags to an animal (group admin or site admin only)
// Route: POST /api/groups/:id/animals/:animalId/tags
func AssignTagsToAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can assign tags"})
			return
		}

		var req struct {
			TagIDs []uint `json:"tag_ids" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Get the animal and verify it belongs to this group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found in this group"})
			return
		}

		// Get the tags (ensure they belong to this group)
		var tags []models.AnimalTag
		if len(req.TagIDs) > 0 {
			if err := db.Where("id IN ? AND group_id = ?", req.TagIDs, groupID).Find(&tags).Error; err != nil {
				logger.Error("Failed to fetch tags", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
				return
			}
		}

		// Replace all tags with the new set
		if err := db.Model(&animal).Association("Tags").Replace(tags); err != nil {
			logger.Error("Failed to assign tags to animal", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign tags"})
			return
		}

		// Reload animal with tags
		if err := db.Preload("Tags").First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload animal"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"animal_id": animal.ID,
			"tag_count": len(tags),
			"group_id":  groupID,
		}).Info("Assigned tags to animal")

		c.JSON(http.StatusOK, animal)
	}
}
