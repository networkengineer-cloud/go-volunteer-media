package handlers

import (
	"time"

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
