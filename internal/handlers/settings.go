package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

// settingValidationRules defines validation rules for specific setting keys
var settingValidationRules = map[string]struct {
	required bool
	maxLen   int
}{
	"site_name":        {required: true, maxLen: 100},
	"site_short_name":  {required: true, maxLen: 50},
	"site_description": {required: false, maxLen: 500},
	"hero_image_url":   {required: false, maxLen: 500},
}

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
			Value string `json:"value"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Validate setting value if validation rules exist for this key
		if rules, ok := settingValidationRules[key]; ok {
			trimmedValue := strings.TrimSpace(req.Value)

			if rules.required && trimmedValue == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s is required", key)})
				return
			}

			if len(req.Value) > rules.maxLen {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s must be %d characters or less", key, rules.maxLen)})
				return
			}
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

// UploadHeroImage handles hero image upload (admin only)
func UploadHeroImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
			return
		}

		// Validate file upload (size, type, content) - use smaller limit for hero images
		if err := upload.ValidateImageUpload(file, upload.MaxHeroImageSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Get validated extension
		ext := strings.ToLower(filepath.Ext(file.Filename))

		// Generate unique filename
		filename := fmt.Sprintf("hero-%s%s", uuid.New().String(), ext)
		filepath := fmt.Sprintf("./public/uploads/%s", filename)

		// Save file
		if err := c.SaveUploadedFile(file, filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		// Return the URL path
		imageURL := fmt.Sprintf("/uploads/%s", filename)
		c.JSON(http.StatusOK, gin.H{"url": imageURL})
	}
}
