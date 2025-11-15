package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// NullableTime is a custom type that handles empty strings from JSON
// Empty strings are treated as nil, while valid timestamps are parsed normally
type NullableTime struct {
	Time  *time.Time
	Valid bool
}

// UnmarshalJSON implements custom unmarshaling for NullableTime
// It handles empty strings ("") by treating them as null values
func (nt *NullableTime) UnmarshalJSON(data []byte) error {
	// Handle null explicitly
	if string(data) == "null" {
		nt.Time = nil
		nt.Valid = false
		return nil
	}

	// Handle empty string by treating it as null
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	
	if s == "" {
		nt.Time = nil
		nt.Valid = false
		return nil
	}

	// Parse the time string
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	nt.Time = &t
	nt.Valid = true
	return nil
}

// MarshalJSON implements custom marshaling for NullableTime
func (nt NullableTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid || nt.Time == nil {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

// AnimalRequest represents the request structure for creating/updating animals
type AnimalRequest struct {
	Name                string       `json:"name" binding:"required"`
	Species             string       `json:"species"`
	Breed               string       `json:"breed"`
	Age                 int          `json:"age"`
	Description         string       `json:"description"`
	ImageURL            string       `json:"image_url,omitempty"`
	Status              string       `json:"status"`
	GroupID             uint         `json:"group_id,omitempty"`
	QuarantineStartDate NullableTime `json:"quarantine_start_date,omitempty"`
	IsReturned          *bool        `json:"is_returned,omitempty"` // Pointer to distinguish null from false
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
