package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
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

// GetAnimalImages returns all images for an animal (authenticated users)
// GET /api/groups/:id/animals/:animalId/images
func GetAnimalImages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify animal exists and belongs to group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Get all images for this animal (exclude the binary data for listing)
		var images []models.AnimalImage
		if err := db.Preload("User").
			Select("id, created_at, updated_at, animal_id, user_id, image_url, caption, is_profile_picture, width, height, file_size").
			Where("animal_id = ?", animalID).
			Order("is_profile_picture DESC, created_at DESC").
			Find(&images).Error; err != nil {
			logger.Error("Failed to fetch animal images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch images"})
			return
		}

		c.JSON(http.StatusOK, images)
	}
}

// UploadAnimalImageToGallery handles image uploads to animal gallery (authenticated users)
// POST /api/groups/:id/animals/:animalId/images
// Images are stored using the configured storage provider
func UploadAnimalImageToGallery(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify animal exists and belongs to group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Get uploaded file
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

		bounds := img.Bounds()
		originalWidth := bounds.Dx()
		originalHeight := bounds.Dy()

		logger.WithFields(map[string]interface{}{
			"format": format,
			"width":  originalWidth,
			"height": originalHeight,
		}).Debug("Received image for upload")

		// Resize image if it's larger than 1200px on the longest side
		maxDimension := uint(1200)
		var resizedImg image.Image

		width := uint(originalWidth)
		height := uint(originalHeight)

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

		// Get caption from form (optional)
		caption := c.PostForm("caption")

		// Upload to storage provider
		metadata := map[string]string{
			"width":   strconv.Itoa(finalBounds.Dx()),
			"height":  strconv.Itoa(finalBounds.Dy()),
			"caption": caption,
		}

		storageURL, blobUUID, blobExt, err := storageProvider.UploadImage(ctx, imageData, "image/jpeg", metadata)
		var imageURL string
		var imageDataForDB []byte
		var storageProviderName string
		var blobIdentifier string

		if err != nil {
			// If storage provider upload fails, fall back to PostgreSQL
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
			storageProviderName = storageProvider.Name()
			// Combine UUID and extension for identifier
			blobIdentifier = blobUUID + blobExt
		}

		// Create database record
		animalIDUint, _ := strconv.ParseUint(animalID, 10, 32)
		userIDUint := userID.(uint)
		animalIDVal := uint(animalIDUint)

		animalImage := models.AnimalImage{
			AnimalID:        &animalIDVal,
			UserID:          userIDUint,
			ImageURL:        imageURL,
			ImageData:       imageDataForDB,
			MimeType:        "image/jpeg",
			Caption:         caption,
			Width:           finalBounds.Dx(),
			Height:          finalBounds.Dy(),
			FileSize:        len(imageData),
			StorageProvider: storageProviderName,
			BlobIdentifier:  blobIdentifier,
			BlobExtension:   blobExt,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := db.Create(&animalImage).Error; err != nil {
			logger.Error("Failed to save image to database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		// Preload user for response
		db.Preload("User").First(&animalImage, animalImage.ID)

		logger.WithFields(map[string]interface{}{
			"image_id":         animalImage.ID,
			"animal_id":        animalID,
			"url":              imageURL,
			"size":             len(imageData),
			"storage_provider": storageProviderName,
		}).Info("Image uploaded and stored")

		c.JSON(http.StatusOK, animalImage)
	}
}

// DeleteAnimalImage deletes an image (admin or owner only)
// DELETE /api/groups/:id/animals/:animalId/images/:imageId
func DeleteAnimalImage(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		imageID := c.Param("imageId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Get the image
		var animalImage models.AnimalImage
		if err := db.Where("id = ? AND animal_id = ?", imageID, animalID).First(&animalImage).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
			return
		}

		// Check if user owns the image or is admin
		userIDUint := userID.(uint)
		adminBool := isAdmin.(bool)
		if animalImage.UserID != userIDUint && !adminBool {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own images"})
			return
		}

		// Don't allow deleting profile picture without warning
		if animalImage.IsProfilePicture {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete profile picture. Please set a different profile picture first."})
			return
		}

		// Delete from storage provider if using Azure
		if animalImage.StorageProvider == "azure" && animalImage.BlobIdentifier != "" {
			if err := storageProvider.DeleteImage(ctx, animalImage.BlobIdentifier); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":           err.Error(),
					"blob_identifier": animalImage.BlobIdentifier,
				}).Warn("Failed to delete image from storage provider, continuing with database deletion")
			}
		}

		// Delete from database (soft delete)
		if err := db.Delete(&animalImage).Error; err != nil {
			logger.Error("Failed to delete image record", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":         imageID,
			"animal_id":        animalID,
			"user_id":          userIDUint,
			"storage_provider": animalImage.StorageProvider,
		}).Info("Image deleted successfully")

		c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
	}
}

// SetAnimalProfilePicture sets an image as the animal's profile picture (admin only)
// PUT /api/admin/animals/:animalId/images/:imageId/set-profile
func SetAnimalProfilePicture(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		animalID := c.Param("animalId")
		imageID := c.Param("imageId")

		// Get the image
		var animalImage models.AnimalImage
		if err := db.Where("id = ? AND animal_id = ?", imageID, animalID).First(&animalImage).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
			return
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Unset any existing profile picture for this animal
		if err := tx.Model(&models.AnimalImage{}).
			Where("animal_id = ? AND is_profile_picture = ?", animalID, true).
			Update("is_profile_picture", false).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to unset existing profile picture", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		// Set the new profile picture
		if err := tx.Model(&animalImage).Update("is_profile_picture", true).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to set new profile picture", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		// Update the animal's image_url
		if err := tx.Model(&models.Animal{}).
			Where("id = ?", animalID).
			Update("image_url", animalImage.ImageURL).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to update animal image_url", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal profile picture"})
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			logger.Error("Failed to commit transaction", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":  imageID,
			"animal_id": animalID,
		}).Info("Profile picture updated successfully")

		// Reload with user data
		db.Preload("User").First(&animalImage, animalImage.ID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Profile picture updated successfully",
			"image":   animalImage,
		})
	}
}

// SetAnimalProfilePictureGroupScoped sets an image as the animal's profile picture (group admin access)
// PUT /api/groups/:groupId/animals/:animalId/images/:imageId/set-profile
func SetAnimalProfilePictureGroupScoped(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		imageID := c.Param("imageId")

		// Get user context
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check if user is a member of this group
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify the animal belongs to this group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found in this group"})
			return
		}

		// Get the image
		var animalImage models.AnimalImage
		if err := db.Where("id = ? AND animal_id = ?", imageID, animalID).First(&animalImage).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
			return
		}

		// Start transaction
		tx := db.Begin()
		if err := tx.Error; err != nil {
			logger.Error("Failed to start transaction", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Unset any existing profile picture for this animal
		if err := tx.Model(&models.AnimalImage{}).
			Where("animal_id = ? AND is_profile_picture = ?", animalID, true).
			Update("is_profile_picture", false).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to unset existing profile picture", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		// Set the new profile picture
		if err := tx.Model(&animalImage).Update("is_profile_picture", true).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to set new profile picture", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		// Update the animal's image_url
		if err := tx.Model(&models.Animal{}).
			Where("id = ?", animalID).
			Update("image_url", animalImage.ImageURL).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to update animal image_url", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal profile picture"})
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			logger.Error("Failed to commit transaction", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":  imageID,
			"animal_id": animalID,
			"group_id":  groupID,
			"user_id":   userID,
		}).Info("Profile picture updated successfully")

		// Reload with user data
		db.Preload("User").First(&animalImage, animalImage.ID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Profile picture updated successfully",
			"image":   animalImage,
		})
	}
}

// GetDeletedImages returns all soft-deleted images for admin monitoring (admin only)
func GetDeletedImages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Get all soft-deleted images for this group (exclude binary data)
		var images []models.AnimalImage
		if err := db.Unscoped().
			Select("animal_images.id, animal_images.created_at, animal_images.updated_at, animal_images.deleted_at, animal_images.animal_id, animal_images.user_id, animal_images.image_url, animal_images.caption, animal_images.is_profile_picture, animal_images.width, animal_images.height, animal_images.file_size").
			Preload("User").
			Preload("Animal").
			Joins("JOIN animals ON animals.id = animal_images.animal_id").
			Where("animals.group_id = ? AND animal_images.deleted_at IS NOT NULL", groupID).
			Order("animal_images.deleted_at DESC").
			Find(&images).Error; err != nil {
			logger.Error("Failed to retrieve deleted images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deleted images"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"group_id": groupID,
			"count":    len(images),
		}).Info("Retrieved deleted images for admin")

		c.JSON(http.StatusOK, gin.H{"data": images})
	}
}
