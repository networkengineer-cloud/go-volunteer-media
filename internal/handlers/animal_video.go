package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type mediaResponse struct {
	Images []models.AnimalImage `json:"images"`
	Videos []models.AnimalVideo `json:"videos"`
}

// GetAnimalMedia returns all images and videos for an animal.
// GET /api/groups/:id/animals/:animalId/media
func GetAnimalMedia(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userIDUint, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		var images []models.AnimalImage
		if err := db.
			Select("id, created_at, updated_at, animal_id, user_id, image_url, caption, is_profile_picture, width, height, file_size").
			Where("animal_id = ?", animalID).
			Order("is_profile_picture DESC, created_at DESC").
			Find(&images).Error; err != nil {
			logger.Error("Failed to fetch animal images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}

		var videos []models.AnimalVideo
		if err := db.Preload("User").
			Where("animal_id = ?", animalID).
			Order("created_at DESC").
			Find(&videos).Error; err != nil {
			logger.Error("Failed to fetch animal videos", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}

		c.JSON(http.StatusOK, mediaResponse{Images: images, Videos: videos})
	}
}
