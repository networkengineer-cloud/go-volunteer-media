package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
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

// animalWithCounts extends Animal with photo/video counts for the list endpoint.
type animalWithCounts struct {
	models.Animal
	ImageCount int `json:"image_count"`
	VideoCount int `json:"video_count"`
}

// GetAnimals returns all animals in a group with optional filtering
func GetAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Build query with filters
		query := db.WithContext(ctx).Where("group_id = ?", groupID)

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

		var baseAnimals []models.Animal
		if err := query.Preload("Tags").Find(&baseAnimals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		// Collect IDs for count subquery
		type countRow struct {
			AnimalID   uint `gorm:"column:animal_id"`
			ImageCount int  `gorm:"column:image_count"`
			VideoCount int  `gorm:"column:video_count"`
		}
		ids := make([]uint, len(baseAnimals))
		for i, a := range baseAnimals {
			ids[i] = a.ID
		}
		var counts []countRow
		if len(ids) > 0 {
			// Best-effort: counts remain zero on error so the list still renders.
			if result := db.WithContext(ctx).Raw(`
				SELECT a.id AS animal_id,
					COUNT(DISTINCT ai.id) AS image_count,
					COUNT(DISTINCT av.id) AS video_count
				FROM animals a
				LEFT JOIN animal_images ai ON ai.animal_id = a.id
				LEFT JOIN animal_videos av ON av.animal_id = a.id
				WHERE a.id IN ?
				GROUP BY a.id`, ids).Scan(&counts); result.Error != nil {
				log.Printf("GetAnimals: failed to fetch media counts: %v", result.Error)
			}
		}
		countMap := make(map[uint]countRow, len(counts))
		for _, cr := range counts {
			countMap[cr.AnimalID] = cr
		}

		animals := make([]animalWithCounts, len(baseAnimals))
		for i, a := range baseAnimals {
			animals[i] = animalWithCounts{
				Animal:     a,
				ImageCount: countMap[a.ID].ImageCount,
				VideoCount: countMap[a.ID].VideoCount,
			}
		}

		c.JSON(http.StatusOK, animals)
	}
}

// GetAnimal returns a specific animal by ID
func GetAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
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
		if err := db.WithContext(ctx).Preload("Tags").Preload("NameHistory").Preload("Scripts").Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		c.JSON(http.StatusOK, animal)
	}
}

// CreateAnimal creates a new animal in a group
func CreateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
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
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}
		if !isValidApprovalStatus(req.QuarantineApprovalStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quarantine_approval_status: must be '', 'requested', or 'granted'"})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		now := time.Now()

		// Use provided arrival_date if available, otherwise use current time
		arrivalDate := &now
		if req.ArrivalDate.Valid && req.ArrivalDate.Time != nil {
			arrivalDate = req.ArrivalDate.Time
		}

		animal := models.Animal{
			GroupID:          uint(gid),
			Name:             req.Name,
			Species:          req.Species,
			Breed:            req.Breed,
			Age:              req.Age,
			Description:      req.Description,
			TrainerNotes:     req.TrainerNotes,
			ImageURL:         req.ImageURL,
			Status:           req.Status,
			ArrivalDate:      arrivalDate,
			LastStatusChange: &now,
		}

		// Set estimated birth date if provided
		if req.EstimatedBirthDate.Valid && req.EstimatedBirthDate.Time != nil {
			animal.EstimatedBirthDate = req.EstimatedBirthDate.Time
			// Auto-compute Age (whole years) from birth date for backward compatibility
			animal.Age = animal.AgeYearsFromBirthDate()
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
			// Set third-party approval status if provided
			if req.QuarantineApprovalStatus != nil && *req.QuarantineApprovalStatus != "" {
				animal.QuarantineApprovalStatus = *req.QuarantineApprovalStatus
				animal.QuarantineApprovalDate = &now
			}
		case "archived":
			animal.ArchivedDate = &now
		}

		if req.IsReturned != nil {
			animal.IsReturned = *req.IsReturned
		}

		if err := db.WithContext(ctx).Create(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal"})
			return
		}

		// If an image_url was provided, link any unlinked images with this URL to this animal
		// Only link images uploaded by the current user to prevent race conditions
		if req.ImageURL != "" {
			if userIDUint, ok := userID.(uint); ok {
				if err := db.WithContext(ctx).Model(&models.AnimalImage{}).
					Where("image_url = ? AND animal_id IS NULL AND user_id = ?", req.ImageURL, userIDUint).
					Update("animal_id", animal.ID).Error; err != nil {
					// Log error with context but don't fail the creation
					logger := middleware.GetLogger(c)
					logger.WithFields(map[string]interface{}{
						"animal_id": animal.ID,
						"image_url": req.ImageURL,
						"user_id":   userIDUint,
					}).Error("Failed to link uploaded images to animal", err)
				}
			}
		}

		c.JSON(http.StatusCreated, animal)
	}
}

// UpdateAnimal updates an existing animal
func UpdateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
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
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}
		if !isValidApprovalStatus(req.QuarantineApprovalStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quarantine_approval_status: must be '', 'requested', or 'granted'"})
			return
		}

		var animal models.Animal
		if err := db.WithContext(ctx).Preload("Tags").Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Track name changes
		oldName := animal.Name
		if req.Name != oldName {
			// Create name history record
			changedByID, ok := middleware.GetUserID(c)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
				return
			}
			nameHistory := models.AnimalNameHistory{
				AnimalID:  animal.ID,
				OldName:   oldName,
				NewName:   req.Name,
				ChangedBy: changedByID,
			}
			if err := db.WithContext(ctx).Create(&nameHistory).Error; err != nil {
				// Log error but don't fail the update
				c.Error(err)
			}
		}

		// Track status changes
		oldStatus := animal.Status
		newStatus := req.Status
		now := time.Now()
		if newStatus != "" && newStatus != oldStatus {
			animal.LastStatusChange = &now

			// Update status-specific dates
			switch newStatus {
			case "available":
				// When moving back to available from archived, reset arrival date
				if oldStatus == "archived" {
					animal.ArrivalDate = &now
				}
				// Clear specific status dates and approval when leaving quarantine
				animal.FosterStartDate = nil
				animal.QuarantineStartDate = nil
				animal.QuarantineApprovalStatus = ""
				animal.QuarantineApprovalDate = nil
				animal.ArchivedDate = nil
			case "foster":
				animal.FosterStartDate = &now
				animal.QuarantineStartDate = nil
				animal.QuarantineApprovalStatus = ""
				animal.QuarantineApprovalDate = nil
				animal.ArchivedDate = nil
			case "bite_quarantine":
				// Use provided quarantine start date if available, otherwise use current time
				if req.QuarantineStartDate.Valid && req.QuarantineStartDate.Time != nil {
					animal.QuarantineStartDate = req.QuarantineStartDate.Time
				} else {
					animal.QuarantineStartDate = &now
				}
				// Always start clean, then apply provided value if any
				animal.QuarantineApprovalStatus = ""
				animal.QuarantineApprovalDate = nil
				if req.QuarantineApprovalStatus != nil && *req.QuarantineApprovalStatus != "" {
					animal.QuarantineApprovalStatus = *req.QuarantineApprovalStatus
					animal.QuarantineApprovalDate = &now
				}
				animal.FosterStartDate = nil
				animal.ArchivedDate = nil
			case "archived":
				// Always clear approval fields on archive (defensive: approval is only meaningful during quarantine)
				animal.QuarantineApprovalStatus = ""
				animal.QuarantineApprovalDate = nil
				animal.ArchivedDate = &now
			}
			animal.Status = newStatus
		} else if animal.Status == "bite_quarantine" {
			// Update approval status only when explicitly provided (nil = not sent = no change)
			if req.QuarantineApprovalStatus != nil && *req.QuarantineApprovalStatus != animal.QuarantineApprovalStatus {
				if *req.QuarantineApprovalStatus == "" {
					animal.QuarantineApprovalStatus = ""
					animal.QuarantineApprovalDate = nil
				} else {
					animal.QuarantineApprovalStatus = *req.QuarantineApprovalStatus
					animal.QuarantineApprovalDate = &now
				}
			}
			// Update quarantine start date independently — both fields can change in one request
			if req.QuarantineStartDate.Valid && req.QuarantineStartDate.Time != nil {
				animal.QuarantineStartDate = req.QuarantineStartDate.Time
			}
		}

		if req.IsReturned != nil {
			animal.IsReturned = *req.IsReturned
		}

		// Update arrival_date if provided
		if req.ArrivalDate.Valid && req.ArrivalDate.Time != nil {
			animal.ArrivalDate = req.ArrivalDate.Time
		}

		// Update estimated birth date if provided
		if req.EstimatedBirthDate.Valid && req.EstimatedBirthDate.Time != nil {
			animal.EstimatedBirthDate = req.EstimatedBirthDate.Time
		}

		// Update other fields
		animal.Name = req.Name
		animal.Species = req.Species
		animal.Breed = req.Breed
		animal.Age = req.Age
		animal.Description = req.Description
		animal.TrainerNotes = req.TrainerNotes
		animal.ImageURL = req.ImageURL

		// Auto-compute Age from birth date if set
		if animal.EstimatedBirthDate != nil {
			animal.Age = animal.AgeYearsFromBirthDate()
		}

		if err := db.WithContext(ctx).Save(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal"})
			return
		}

		// If an image_url was provided, link any unlinked images with this URL to this animal
		// Only link images uploaded by the current user to prevent race conditions
		if req.ImageURL != "" {
			if userIDUint, ok := userID.(uint); ok {
				if err := db.WithContext(ctx).Model(&models.AnimalImage{}).
					Where("image_url = ? AND animal_id IS NULL AND user_id = ?", req.ImageURL, userIDUint).
					Update("animal_id", animal.ID).Error; err != nil {
					// Log error with context but don't fail the update
					logger := middleware.GetLogger(c)
					logger.WithFields(map[string]interface{}{
						"animal_id": animal.ID,
						"image_url": req.ImageURL,
						"user_id":   userIDUint,
					}).Error("Failed to link uploaded images to animal", err)
				}
			}
		}

		c.JSON(http.StatusOK, animal)
	}
}

// DeleteAnimal deletes an animal
func DeleteAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", animalID, groupID).Delete(&models.Animal{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete animal"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Animal deleted successfully"})
	}
}
