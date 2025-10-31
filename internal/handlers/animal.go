package handlers

import (
	"fmt"
	"image"
	_ "image/gif" // Register GIF format
	"image/jpeg"
	_ "image/png" // Register PNG format
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

// UploadAnimalImage handles secure animal image uploads with optimization
func UploadAnimalImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		
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
				// Landscape - resize based on width
				resizedImg = resize.Resize(maxDimension, 0, img, resize.Lanczos3)
			} else {
				// Portrait or square - resize based on height
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

		// Generate unique filename (always save as .jpg for consistency)
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

		// Encode as JPEG with quality 85 (good balance between quality and size)
		if err := jpeg.Encode(outFile, resizedImg, &jpeg.Options{Quality: 85}); err != nil {
			logger.Error("Failed to encode image", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Return public URL
		url := "/uploads/" + fname
		logger.WithField("url", url).Info("Image uploaded and optimized successfully")
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

type AnimalRequest struct {
	Name        string `json:"name" binding:"required"`
	Species     string `json:"species"`
	Breed       string `json:"breed"`
	Age         int    `json:"age"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url,omitempty"`
	Status      string `json:"status"`
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

// GetAnimals returns all animals in a group with optional filtering
func GetAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Build query with filters
		query := db.Where("group_id = ?", groupID)

		// Status filter (default to "available" if not specified)
		status := c.Query("status")
		if status == "" {
			query = query.Where("status = ?", "available")
		} else if status != "all" {
			query = query.Where("status = ?", status)
		}

		// Name search filter
		nameSearch := c.Query("name")
		if nameSearch != "" {
			query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(nameSearch)+"%")
		}

		var animals []models.Animal
		if err := query.Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		c.JSON(http.StatusOK, animals)
	}
}

// GetAnimal returns a specific animal by ID
func GetAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		c.JSON(http.StatusOK, animal)
	}
}

// CreateAnimal creates a new animal in a group
func CreateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		animal := models.Animal{
			GroupID:     uint(gid),
			Name:        req.Name,
			Species:     req.Species,
			Breed:       req.Breed,
			Age:         req.Age,
			Description: req.Description,
			ImageURL:    req.ImageURL,
			Status:      req.Status,
		}

		if animal.Status == "" {
			animal.Status = "available"
		}

		if err := db.Create(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal"})
			return
		}

		c.JSON(http.StatusCreated, animal)
	}
}

// UpdateAnimal updates an existing animal
func UpdateAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		animal.Name = req.Name
		animal.Species = req.Species
		animal.Breed = req.Breed
		animal.Age = req.Age
		animal.Description = req.Description
		animal.ImageURL = req.ImageURL
		if req.Status != "" {
			animal.Status = req.Status
		}

		if err := db.Save(&animal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal"})
			return
		}

		c.JSON(http.StatusOK, animal)
	}
}

// DeleteAnimal deletes an animal
func DeleteAnimal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).Delete(&models.Animal{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete animal"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Animal deleted successfully"})
	}
}
