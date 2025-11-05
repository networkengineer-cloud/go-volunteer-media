package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// UpdateAnimalAdmin updates an existing animal by ID (admin only, no group check needed)
func UpdateAnimalAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		animalID := c.Param("animalId")

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var animal models.Animal
		if err := db.Preload("Tags").First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Build update map with only provided fields
		updates := make(map[string]interface{})
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Species != "" {
			updates["species"] = req.Species
		}
		if req.Breed != "" {
			updates["breed"] = req.Breed
		}
		if req.Age > 0 {
			updates["age"] = req.Age
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.ImageURL != "" {
			updates["image_url"] = req.ImageURL
		}
		if req.Status != "" && req.Status != animal.Status {
			// Track status change
			now := time.Now()
			updates["status"] = req.Status
			updates["last_status_change"] = now

			// Update status-specific dates
			switch req.Status {
			case "available":
				updates["foster_start_date"] = nil
				updates["quarantine_start_date"] = nil
				updates["archived_date"] = nil
			case "foster":
				updates["foster_start_date"] = now
				updates["quarantine_start_date"] = nil
				updates["archived_date"] = nil
			case "bite_quarantine":
				// Use provided quarantine start date if available, otherwise use current time
				if req.QuarantineStartDate != nil {
					updates["quarantine_start_date"] = *req.QuarantineStartDate
				} else {
					updates["quarantine_start_date"] = now
				}
				updates["foster_start_date"] = nil
				updates["archived_date"] = nil
			case "archived":
				updates["archived_date"] = now
			}
		}
		if req.GroupID != 0 {
			updates["group_id"] = req.GroupID
		}
		// Update quarantine start date if provided and status is quarantine
		if req.QuarantineStartDate != nil && (req.Status == "bite_quarantine" || animal.Status == "bite_quarantine") {
			updates["quarantine_start_date"] = *req.QuarantineStartDate
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
			return
		}

		if err := db.Model(&animal).Updates(updates).Error; err != nil {
			logger.Error("Failed to update animal", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal"})
			return
		}

		// Reload animal to get updated data
		if err := db.Preload("Tags").First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload animal"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"animal_id": animal.ID,
			"updates":   updates,
		}).Info("Updated animal")

		c.JSON(http.StatusOK, animal)
	}
}

// BulkUpdateAnimalsRequest represents the bulk update request
type BulkUpdateAnimalsRequest struct {
	AnimalIDs []uint  `json:"animal_ids" binding:"required"`
	GroupID   *uint   `json:"group_id,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// BulkUpdateAnimals updates multiple animals at once (admin only)
func BulkUpdateAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		var req BulkUpdateAnimalsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.AnimalIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No animal IDs provided"})
			return
		}

		// Build update map
		updates := make(map[string]interface{})
		if req.GroupID != nil {
			updates["group_id"] = *req.GroupID
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
			return
		}

		// Perform bulk update
		if err := db.Model(&models.Animal{}).Where("id IN ?", req.AnimalIDs).Updates(updates).Error; err != nil {
			logger.Error("Failed to bulk update animals", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animals"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"count":    len(req.AnimalIDs),
			"group_id": req.GroupID,
			"status":   req.Status,
		}).Info("Bulk updated animals")

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Successfully updated %d animals", len(req.AnimalIDs)),
			"count":   len(req.AnimalIDs),
		})
	}
}

// GetAllAnimals returns all animals (admin only, for bulk edit page)
func GetAllAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build query with filters
		query := db.Model(&models.Animal{})

		// Status filter
		status := c.Query("status")
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}

		// Group filter
		groupID := c.Query("group_id")
		if groupID != "" {
			query = query.Where("group_id = ?", groupID)
		}

		// Name search filter
		nameSearch := c.Query("name")
		if nameSearch != "" {
			query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(nameSearch)+"%")
		}

		var animals []models.Animal
		if err := query.Preload("Tags").Order("group_id, name").Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		c.JSON(http.StatusOK, animals)
	}
}
