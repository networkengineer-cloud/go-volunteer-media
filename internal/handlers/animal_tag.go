package handlers

import (
	"net/http"

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
	Icon     string `json:"icon" binding:"required,max=10"` // Unicode emoji or icon identifier
}

// GetAnimalTags returns all animal tags
func GetAnimalTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tags []models.AnimalTag
		if err := db.Order("category, name").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animal tags"})
			return
		}

		c.JSON(http.StatusOK, tags)
	}
}

// CreateAnimalTag creates a new animal tag (admin only)
func CreateAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		
		var req AnimalTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tag := models.AnimalTag{
			Name:     req.Name,
			Category: req.Category,
			Color:    req.Color,
			Icon:     req.Icon,
		}

		if err := db.Create(&tag).Error; err != nil {
			logger.Error("Failed to create animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal tag"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"tag_id":   tag.ID,
			"tag_name": tag.Name,
		}).Info("Created animal tag")

		c.JSON(http.StatusCreated, tag)
	}
}

// UpdateAnimalTag updates an existing animal tag (admin only)
func UpdateAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		tagID := c.Param("tagId")

		var req AnimalTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var tag models.AnimalTag
		if err := db.First(&tag, tagID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal tag not found"})
			return
		}

		tag.Name = req.Name
		tag.Category = req.Category
		tag.Color = req.Color
		tag.Icon = req.Icon

		if err := db.Save(&tag).Error; err != nil {
			logger.Error("Failed to update animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal tag"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"tag_id":   tag.ID,
			"tag_name": tag.Name,
		}).Info("Updated animal tag")

		c.JSON(http.StatusOK, tag)
	}
}

// DeleteAnimalTag deletes an animal tag (admin only)
func DeleteAnimalTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		tagID := c.Param("tagId")

		if err := db.Delete(&models.AnimalTag{}, tagID).Error; err != nil {
			logger.Error("Failed to delete animal tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete animal tag"})
			return
		}

		logger.WithField("tag_id", tagID).Info("Deleted animal tag")
		c.JSON(http.StatusOK, gin.H{"message": "Animal tag deleted successfully"})
	}
}

// AssignTagsToAnimal assigns tags to an animal (admin only)
func AssignTagsToAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		animalID := c.Param("animalId")

		var req struct {
			TagIDs []uint `json:"tag_ids" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the animal
		var animal models.Animal
		if err := db.First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Get the tags
		var tags []models.AnimalTag
		if len(req.TagIDs) > 0 {
			if err := db.Where("id IN ?", req.TagIDs).Find(&tags).Error; err != nil {
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
		}).Info("Assigned tags to animal")

		c.JSON(http.StatusOK, animal)
	}
}
