package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// GetSiteSettings returns all site settings (public endpoint)
func GetSiteSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings []models.SiteSetting
		if err := db.Find(&settings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch site settings"})
			return
		}

		// Convert to map for easier frontend consumption
		settingsMap := make(map[string]string)
		for _, setting := range settings {
			settingsMap[setting.Key] = setting.Value
		}

		c.JSON(http.StatusOK, settingsMap)
	}
}

// UpdateSiteSetting updates a specific site setting (admin only)
func UpdateSiteSetting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")

		var req struct {
			Value string `json:"value" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var setting models.SiteSetting
		result := db.Where("key = ?", key).First(&setting)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new setting if it doesn't exist
			setting = models.SiteSetting{
				Key:   key,
				Value: req.Value,
			}
			if err := db.Create(&setting).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create setting"})
				return
			}
		} else if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch setting"})
			return
		} else {
			// Update existing setting
			setting.Value = req.Value
			if err := db.Save(&setting).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
				return
			}
		}

		c.JSON(http.StatusOK, setting)
	}
}
