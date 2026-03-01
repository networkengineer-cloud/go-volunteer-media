package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
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

// UploadHeroImage handles hero image upload (admin only).
// The image is persisted to durable storage (postgres bytea or Azure Blob) via
// an AnimalImage record so that ServeImage can resolve it on subsequent requests.
// The caller must persist the returned URL separately via PUT /api/admin/settings/hero_image_url.
func UploadHeroImage(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		userID := c.GetUint("user_id")

		file, err := c.FormFile("image")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
			return
		}

		// Validate file upload (size, type, content) - use smaller limit for hero images
		if err := upload.ValidateImageUpload(file, upload.MaxHeroImageSize); err != nil {
			logger.Error("File validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Open and read file bytes
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
			return
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			logger.Error("Failed to read file bytes", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
			return
		}

		// Detect MIME type from file content; fall back to extension-based lookup
		// for formats like HEIC/HEIF that http.DetectContentType does not recognise.
		mimeType := http.DetectContentType(data)
		if mimeType == "application/octet-stream" {
			ext := strings.ToLower(filepath.Ext(file.Filename))
			if types, ok := upload.AllowedImageTypes[ext]; ok {
				mimeType = types[0]
			}
		}

		// Upload to storage provider (generates URL and, for Azure, persists the blob)
		storageURL, blobUUID, blobExt, err := storageProvider.UploadImage(ctx, data, mimeType, nil)
		if err != nil {
			logger.Error("Failed to upload image to storage", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}

		// Persist image data so ServeImage can resolve the /api/images/:uuid URL.
		// For postgres the raw bytes are stored; for Azure only the blob identifier.
		var imageDataForDB []byte
		var storageProviderName string
		var blobIdentifier string
		if storageProvider.Name() == "azure" {
			storageProviderName = "azure"
			blobIdentifier = blobUUID + blobExt
		} else {
			storageProviderName = "postgres"
			imageDataForDB = data
		}

		record := models.AnimalImage{
			AnimalID:        nil, // Not linked to any animal — hero image
			UserID:          userID,
			ImageURL:        storageURL,
			ImageData:       imageDataForDB,
			MimeType:        mimeType,
			FileSize:        len(data),
			StorageProvider: storageProviderName,
			BlobIdentifier:  blobIdentifier,
			BlobExtension:   blobExt,
		}
		if err := db.WithContext(ctx).Create(&record).Error; err != nil {
			logger.Error("Failed to persist hero image record", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		logger.WithField("url", storageURL).Info("Hero image uploaded successfully")
		c.JSON(http.StatusOK, gin.H{"url": storageURL})
	}
}
