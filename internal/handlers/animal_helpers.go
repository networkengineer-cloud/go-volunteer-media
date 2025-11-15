package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// AnimalRequest represents the request structure for creating/updating animals
type AnimalRequest struct {
	Name                string     `json:"name" binding:"required"`
	Species             string     `json:"species"`
	Breed               string     `json:"breed"`
	Age                 int        `json:"age"`
	Description         string     `json:"description"`
	ImageURL            string     `json:"image_url,omitempty"`
	Status              string     `json:"status"`
	GroupID             uint       `json:"group_id,omitempty"`
	QuarantineStartDate *time.Time `json:"quarantine_start_date,omitempty"`
}

// DuplicateNameInfo represents information about animals with duplicate names
type DuplicateNameInfo struct {
	Name          string          `json:"name"`
	Count         int             `json:"count"`
	Animals       []models.Animal `json:"animals"`
	HasDuplicates bool            `json:"has_duplicates"`
}

// checkGroupAccess verifies if the user has access to a specific group
func checkGroupAccess(db *gorm.DB, userID interface{}, isAdmin interface{}, groupID string) bool {
	if isAdmin.(bool) {
		return true
	}

	var user models.User
	if err := db.Preload("Groups", "id = ?", groupID).First(&user, userID).Error; err != nil {
		return false
	}
	return len(user.Groups) > 0
}

// CheckDuplicateNames checks if any animals in a group have duplicate names
func CheckDuplicateNames(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		name := c.Query("name")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name parameter is required"})
			return
		}

		var animals []models.Animal
		// Case-insensitive search for animals with the same name in the group
		// Includes all statuses (available, foster, quarantine, archived) to properly detect duplicates
		query := db.Where("group_id = ? AND LOWER(name) = ?", groupID, strings.ToLower(name))
		if err := query.Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for duplicates"})
			return
		}

		result := DuplicateNameInfo{
			Name:          name,
			Count:         len(animals),
			Animals:       animals,
			HasDuplicates: len(animals) > 1,
		}

		c.JSON(http.StatusOK, result)
	}
}
