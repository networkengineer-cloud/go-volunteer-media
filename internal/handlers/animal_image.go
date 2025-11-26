package handlers

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
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

		// Get all images for this animal
		var images []models.AnimalImage
		if err := db.Preload("User").
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
func UploadAnimalImageToGallery(db *gorm.DB) gin.HandlerFunc {
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

		// Generate unique filename
		fname := fmt.Sprintf("%d_%s.jpg", time.Now().UnixNano(), uuid.New().String())
		uploadPath := filepath.Join("public", "uploads", fname)

		logger.WithField("path", uploadPath).Debug("Saving optimized image")

		// Create the output file
		outFile, err := os.Create(uploadPath)
		if err != nil {
			logger.Error("Failed to create output file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}
		defer outFile.Close()

		// Encode as JPEG with quality 85
		if err := jpeg.Encode(outFile, resizedImg, &jpeg.Options{Quality: 85}); err != nil {
			logger.Error("Failed to encode image", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Get file size
		fileInfo, _ := outFile.Stat()
		fileSize := int(fileInfo.Size())

		// Get caption from form (optional)
		caption := c.PostForm("caption")

		// Create database record
		imageURL := "/uploads/" + fname
		animalIDUint, _ := strconv.ParseUint(animalID, 10, 32)
		userIDUint := userID.(uint)

		animalImage := models.AnimalImage{
			AnimalID: uint(animalIDUint),
			UserID:   userIDUint,
			ImageURL: imageURL,
			Caption:  caption,
			Width:    resizedImg.Bounds().Dx(),
			Height:   resizedImg.Bounds().Dy(),
			FileSize: fileSize,
		}

		if err := db.Create(&animalImage).Error; err != nil {
			logger.Error("Failed to save image record", err)
			// Try to delete the uploaded file
			os.Remove(uploadPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image record"})
			return
		}

		// Preload user for response
		db.Preload("User").First(&animalImage, animalImage.ID)

		logger.WithField("url", imageURL).Info("Image uploaded and saved to gallery successfully")
		c.JSON(http.StatusOK, animalImage)
	}
}

// DeleteAnimalImage deletes an image (admin or owner only)
// DELETE /api/groups/:id/animals/:animalId/images/:imageId
func DeleteAnimalImage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Delete from database (soft delete)
		if err := db.Delete(&animalImage).Error; err != nil {
			logger.Error("Failed to delete image record", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"image_id":  imageID,
			"animal_id": animalID,
			"user_id":   userIDUint,
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

// GetDeletedImages returns all soft-deleted images for admin monitoring (admin only)
func GetDeletedImages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Only admins can view deleted images
		if !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Get all soft-deleted images for this group
		var images []models.AnimalImage
		if err := db.Unscoped().
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
