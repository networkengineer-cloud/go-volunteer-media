package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // Register GIF format
	"image/jpeg"
	_ "image/png" // Register PNG format
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

// UploadAnimalImage handles secure animal image uploads with optimization
// Images are stored in the database for persistence across container restarts
func UploadAnimalImage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		// Get animal ID from URL parameter
		animalIDStr := c.Param("animalId")
		animalID, err := strconv.ParseUint(animalIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid animal ID"})
			return
		}

		// Get user ID from context
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		file, err := c.FormFile("image")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Validate file upload (size, type, content)
		if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil {
			logger.Error("File validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		defer src.Close()

		// Decode the image
		img, format, err := image.Decode(src)
		if err != nil {
			logger.Error("Failed to decode image", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file"})
			return
		}
		logger.WithFields(map[string]interface{}{
			"format": format,
			"width":  img.Bounds().Dx(),
			"height": img.Bounds().Dy(),
		}).Debug("Received image for upload")

		// Resize image if it's larger than 1200px on the longest side
		maxDimension := uint(1200)
		var resizedImg image.Image

		bounds := img.Bounds()
		width := uint(bounds.Dx())
		height := uint(bounds.Dy())

		if width > maxDimension || height > maxDimension {
			if width > height {
				resizedImg = resize.Resize(maxDimension, 0, img, resize.Lanczos3)
			} else {
				resizedImg = resize.Resize(0, maxDimension, img, resize.Lanczos3)
			}
			logger.WithFields(map[string]interface{}{
				"new_width":  resizedImg.Bounds().Dx(),
				"new_height": resizedImg.Bounds().Dy(),
			}).Debug("Image resized")
		} else {
			resizedImg = img
			logger.Debug("Image dimensions acceptable, no resizing needed")
		}

		// Encode image to JPEG bytes
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85}); err != nil {
			logger.Error("Failed to encode image", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image"})
			return
		}

		imageData := buf.Bytes()
		finalBounds := resizedImg.Bounds()

		// Generate unique image identifier
		imageUUID := uuid.New().String()
		imageURL := fmt.Sprintf("/api/images/%s", imageUUID)

		// Create image record in database
		animalIDUint := uint(animalID)
		animalImage := models.AnimalImage{
			AnimalID:  &animalIDUint,
			UserID:    userID.(uint),
			ImageURL:  imageURL,
			ImageData: imageData,
			MimeType:  "image/jpeg",
			Width:     finalBounds.Dx(),
			Height:    finalBounds.Dy(),
			FileSize:  len(imageData),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := db.Create(&animalImage).Error; err != nil {
			logger.Error("Failed to save image to database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":  animalImage.ID,
			"animal_id": animalID,
			"url":       imageURL,
			"size":      len(imageData),
		}).Info("Image uploaded and stored in database")

		c.JSON(http.StatusOK, gin.H{
			"url":      imageURL,
			"image_id": animalImage.ID,
			"width":    finalBounds.Dx(),
			"height":   finalBounds.Dy(),
		})
	}
}

// ServeImage serves an image using the configured storage provider
func ServeImage(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		imageUUID := c.Param("uuid")
		imageURL := fmt.Sprintf("/api/images/%s", imageUUID)

		// First, get the image metadata from database
		var animalImage models.AnimalImage
		if err := db.Where("image_url = ?", imageURL).First(&animalImage).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
			return
		}

		// Check which storage provider was used for this image
		if animalImage.StorageProvider == "azure" && animalImage.BlobIdentifier != "" {
			// Retrieve from Azure Blob Storage
			data, mimeType, err := storageProvider.GetImage(ctx, animalImage.BlobIdentifier)
			if err != nil {
				if err == storage.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Image not found in storage"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve image"})
				}
				return
			}

			// Set caching headers (images don't change)
			c.Header("Cache-Control", "public, max-age=31536000") // 1 year
			c.Header("Content-Type", mimeType)
			c.Header("Content-Length", strconv.Itoa(len(data)))
			c.Data(http.StatusOK, mimeType, data)
		} else {
			// Legacy: retrieve from PostgreSQL database
			if len(animalImage.ImageData) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Image data not available"})
				return
			}

			// Set caching headers (images don't change)
			c.Header("Cache-Control", "public, max-age=31536000") // 1 year
			c.Header("Content-Type", animalImage.MimeType)
			c.Header("Content-Length", strconv.Itoa(len(animalImage.ImageData)))
			c.Data(http.StatusOK, animalImage.MimeType, animalImage.ImageData)
		}
	}
}

// UploadAnimalImageSimple handles simple image upload without animal context
// Used for profile picture uploads before animal is fully created
func UploadAnimalImageSimple(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		// Get user ID from context
		userIDVal, exists := c.Get("user_id")
		if !exists {
			logger.Error("User ID not found in context", nil)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		userID, ok := userIDVal.(uint)
		if !ok {
			logger.Error("User ID is not a uint", nil)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user session"})
			return
		}

		file, err := c.FormFile("image")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Validate file upload (size, type, content)
		if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil {
			logger.Error("File validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		defer src.Close()

		// Decode the image
		img, format, err := image.Decode(src)
		if err != nil {
			logger.Error("Failed to decode image", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file"})
			return
		}
		logger.WithFields(map[string]interface{}{
			"format": format,
			"width":  img.Bounds().Dx(),
			"height": img.Bounds().Dy(),
		}).Debug("Received image for upload")

		// Resize image if it's larger than 1200px on the longest side
		maxDimension := uint(1200)
		var resizedImg image.Image

		bounds := img.Bounds()
		width := uint(bounds.Dx())
		height := uint(bounds.Dy())

		if width > maxDimension || height > maxDimension {
			if width > height {
				resizedImg = resize.Resize(maxDimension, 0, img, resize.Lanczos3)
			} else {
				resizedImg = resize.Resize(0, maxDimension, img, resize.Lanczos3)
			}
		} else {
			resizedImg = img
		}

		// Encode image to JPEG bytes
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85}); err != nil {
			logger.Error("Failed to encode image", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image"})
			return
		}

		imageData := buf.Bytes()
		finalBounds := resizedImg.Bounds()

		// Generate unique image identifier
		imageUUID := uuid.New().String()

		// Upload to storage provider
		metadata := map[string]string{
			"width":  strconv.Itoa(finalBounds.Dx()),
			"height": strconv.Itoa(finalBounds.Dy()),
		}

		storageURL, blobIdentifier, err := storageProvider.UploadImage(ctx, imageData, "image/jpeg", metadata)
		var imageURL string
		var imageDataForDB []byte
		var storageProviderName string

		if err != nil {
			// If Azure upload fails, fall back to PostgreSQL
			logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Failed to upload to storage provider, falling back to PostgreSQL")

			imageURL = fmt.Sprintf("/api/images/%s", imageUUID)
			imageDataForDB = imageData
			storageProviderName = "postgres"
			blobIdentifier = ""
		} else {
			// Successfully uploaded to storage provider
			imageURL = storageURL
			imageDataForDB = nil // Don't store in DB when using external storage
			storageProviderName = "azure"
		}

		// Create image record in database with AnimalID = nil (will be linked later)
		animalImage := models.AnimalImage{
			AnimalID:        nil, // Will be linked when animal is created/updated
			UserID:          userID,
			ImageURL:        imageURL,
			ImageData:       imageDataForDB,
			MimeType:        "image/jpeg",
			Width:           finalBounds.Dx(),
			Height:          finalBounds.Dy(),
			FileSize:        len(imageData),
			StorageProvider: storageProviderName,
			BlobIdentifier:  blobIdentifier,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := db.Create(&animalImage).Error; err != nil {
			logger.Error("Failed to save image to database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":         animalImage.ID,
			"url":              imageURL,
			"size":             len(imageData),
			"storage_provider": storageProviderName,
		}).Info("Image uploaded and stored (unlinked)")

		c.JSON(http.StatusOK, gin.H{"url": imageURL})
	}
}
