package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// escapeSQLWildcards escapes SQL wildcard characters (%, _) in user input
// to prevent unintended pattern matching in LIKE queries
func escapeSQLWildcards(input string) string {
	// Escape backslash first to prevent double-escaping
	result := strings.ReplaceAll(input, "\\", "\\\\")
	// Escape SQL wildcard characters
	result = strings.ReplaceAll(result, "%", "\\%")
	result = strings.ReplaceAll(result, "_", "\\_")
	return result
}

// GetAnimals returns all animals in a group with optional filtering
func GetAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Build query with filters
		query := db.Where("group_id = ?", groupID)

		// Status filter (default to "available" and "bite_quarantine" if not specified)
		status := c.Query("status")
		if status == "" {
			// Default: show available and bite_quarantine animals
			query = query.Where("status IN ?", []string{"available", "bite_quarantine"})
		} else if status != "all" {
			// Support comma-separated statuses for multiple filters
			if strings.Contains(status, ",") {
				statuses := strings.Split(status, ",")
				query = query.Where("status IN ?", statuses)
			} else {
				query = query.Where("status = ?", status)
			}
		}

		// Name search filter
		nameSearch := c.Query("name")
		if nameSearch != "" {
			// Escape SQL wildcards to prevent unintended pattern matching
			escaped := escapeSQLWildcards(strings.ToLower(nameSearch))
			query = query.Where("LOWER(name) LIKE ?", "%"+escaped+"%")
		}

		var animals []models.Animal
		if err := query.Preload("Tags").Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		c.JSON(http.StatusOK, animals)
	}
}

// GetAnimal returns a specific animal by ID
func GetAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var animal models.Animal
		if err := db.Preload("Tags").Preload("NameHistory").Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		c.JSON(http.StatusOK, animal)
	}
}

// CreateAnimal creates a new animal in a group
func CreateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		now := time.Now()
		animal := models.Animal{
			GroupID:          uint(gid),
			Name:             req.Name,
			Species:          req.Species,
			Breed:            req.Breed,
			Age:              req.Age,
			Description:      req.Description,
			ImageURL:         req.ImageURL,
			Status:           req.Status,
			ArrivalDate:      &now,
			LastStatusChange: &now,
		}

		if animal.Status == "" {
			animal.Status = "available"
		}

		// Set status-specific dates based on initial status
		switch animal.Status {
		case "foster":
			animal.FosterStartDate = &now
		case "bite_quarantine":
			// Use provided quarantine start date if available, otherwise use current time
			if req.QuarantineStartDate.Valid && req.QuarantineStartDate.Time != nil {
				animal.QuarantineStartDate = req.QuarantineStartDate.Time
			} else {
				animal.QuarantineStartDate = &now
			}
		case "archived":
			animal.ArchivedDate = &now
			// Set is_returned based on request, default to false
			if req.IsReturned != nil {
				animal.IsReturned = *req.IsReturned
			} else {
				animal.IsReturned = false
			}
		}

		if err := db.Create(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal"})
			return
		}

		c.JSON(http.StatusCreated, animal)
	}
}

// UpdateAnimal updates an existing animal
func UpdateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var animal models.Animal
		if err := db.Preload("Tags").Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Track name changes
		oldName := animal.Name
		if req.Name != oldName {
			// Create name history record
			nameHistory := models.AnimalNameHistory{
				AnimalID:  animal.ID,
				OldName:   oldName,
				NewName:   req.Name,
				ChangedBy: userID.(uint),
			}
			if err := db.Create(&nameHistory).Error; err != nil {
				// Log error but don't fail the update
				c.Error(err)
			}
		}

		// Track status changes
		oldStatus := animal.Status
		newStatus := req.Status
		if newStatus != "" && newStatus != oldStatus {
			now := time.Now()
			animal.LastStatusChange = &now

			// Update status-specific dates
			switch newStatus {
			case "available":
				// When moving back to available from archived, increment return count
				if oldStatus == "archived" {
					animal.ReturnCount++
				}
				// Clear specific status dates
				animal.FosterStartDate = nil
				animal.QuarantineStartDate = nil
				animal.ArchivedDate = nil
				animal.IsReturned = false // Clear is_returned flag when no longer archived
			case "foster":
				animal.FosterStartDate = &now
				animal.QuarantineStartDate = nil
				animal.ArchivedDate = nil
			case "bite_quarantine":
				// Use provided quarantine start date if available, otherwise use current time
				if req.QuarantineStartDate.Valid && req.QuarantineStartDate.Time != nil {
					animal.QuarantineStartDate = req.QuarantineStartDate.Time
				} else {
					animal.QuarantineStartDate = &now
				}
				animal.FosterStartDate = nil
				animal.ArchivedDate = nil
			case "archived":
				animal.ArchivedDate = &now
				// Set is_returned based on request, default to false
				if req.IsReturned != nil {
					animal.IsReturned = *req.IsReturned
				} else {
					animal.IsReturned = false
				}
			}
			animal.Status = newStatus
		} else if req.QuarantineStartDate.Valid && req.QuarantineStartDate.Time != nil && animal.Status == "bite_quarantine" {
			// Update quarantine start date if provided and animal is already in quarantine status
			// This handles the case where only the date is being updated without status change
			animal.QuarantineStartDate = req.QuarantineStartDate.Time
		} else if animal.Status == "archived" && req.IsReturned != nil {
			// Update is_returned flag when editing an archived animal
			animal.IsReturned = *req.IsReturned
		}

		// Update other fields
		animal.Name = req.Name
		animal.Species = req.Species
		animal.Breed = req.Breed
		animal.Age = req.Age
		animal.Description = req.Description
		animal.ImageURL = req.ImageURL

		if err := db.Save(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal"})
			return
		}

		c.JSON(http.StatusOK, animal)
	}
}

// DeleteAnimal deletes an animal
func DeleteAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).Delete(&models.Animal{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete animal"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Animal deleted successfully"})
	}
}
