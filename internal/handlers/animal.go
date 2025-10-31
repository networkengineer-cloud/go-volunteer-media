package handlers

import (
	"encoding/csv"
	"fmt"
	"image"
	_ "image/gif" // Register GIF format
	"image/jpeg"
	_ "image/png" // Register PNG format
	"io"
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
	GroupID     uint   `json:"group_id,omitempty"`
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

// UpdateAnimalAdmin updates an existing animal by ID (admin only, no group check needed)
func UpdateAnimalAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		animalID := c.Param("animalId")

		var req AnimalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var animal models.Animal
		if err := db.First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		// Build update map with only provided fields
		updates := make(map[string]interface{})
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Species != "" {
			updates["species"] = req.Species
		}
		if req.Breed != "" {
			updates["breed"] = req.Breed
		}
		if req.Age > 0 {
			updates["age"] = req.Age
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.ImageURL != "" {
			updates["image_url"] = req.ImageURL
		}
		if req.Status != "" {
			updates["status"] = req.Status
		}
		if req.GroupID != 0 {
			updates["group_id"] = req.GroupID
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
			return
		}

		if err := db.Model(&animal).Updates(updates).Error; err != nil {
			logger.Error("Failed to update animal", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal"})
			return
		}

		// Reload animal to get updated data
		if err := db.First(&animal, animalID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload animal"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"animal_id": animal.ID,
			"updates":   updates,
		}).Info("Updated animal")

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

// BulkUpdateAnimalsRequest represents the bulk update request
type BulkUpdateAnimalsRequest struct {
	AnimalIDs []uint  `json:"animal_ids" binding:"required"`
	GroupID   *uint   `json:"group_id,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// BulkUpdateAnimals updates multiple animals at once (admin only)
func BulkUpdateAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		var req BulkUpdateAnimalsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.AnimalIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No animal IDs provided"})
			return
		}

		// Build update map
		updates := make(map[string]interface{})
		if req.GroupID != nil {
			updates["group_id"] = *req.GroupID
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
			return
		}

		// Perform bulk update
		if err := db.Model(&models.Animal{}).Where("id IN ?", req.AnimalIDs).Updates(updates).Error; err != nil {
			logger.Error("Failed to bulk update animals", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animals"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"count":    len(req.AnimalIDs),
			"group_id": req.GroupID,
			"status":   req.Status,
		}).Info("Bulk updated animals")

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Successfully updated %d animals", len(req.AnimalIDs)),
			"count":   len(req.AnimalIDs),
		})
	}
}

// ExportAnimalsCSV exports animals to CSV format
func ExportAnimalsCSV(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Query("group_id")

		query := db.Model(&models.Animal{})
		if groupID != "" {
			query = query.Where("group_id = ?", groupID)
		}

		var animals []models.Animal
		if err := query.Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"count":    len(animals),
			"group_id": groupID,
		}).Info("Exporting animals to CSV")

		// Set response headers for CSV download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=animals.csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write CSV header
		if err := writer.Write([]string{"id", "group_id", "name", "species", "breed", "age", "description", "status", "image_url"}); err != nil {
			logger.Error("Failed to write CSV header", err)
			return
		}

		// Write animal data
		for _, animal := range animals {
			record := []string{
				strconv.FormatUint(uint64(animal.ID), 10),
				strconv.FormatUint(uint64(animal.GroupID), 10),
				animal.Name,
				animal.Species,
				animal.Breed,
				strconv.Itoa(animal.Age),
				animal.Description,
				animal.Status,
				animal.ImageURL,
			}
			if err := writer.Write(record); err != nil {
				logger.Error("Failed to write CSV record", err)
				return
			}
		}
	}
}

// ImportAnimalsCSV imports animals from CSV file
func ImportAnimalsCSV(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		logger.WithField("filename", file.Filename).Info("Processing CSV import")

		// Validate file extension
		if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a CSV"})
			return
		}

		// Open the file
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		defer src.Close()

		// Parse CSV
		reader := csv.NewReader(src)

		// Read header
		header, err := reader.Read()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header"})
			return
		}

		// Validate header has minimum required fields
		if len(header) < 2 { // At minimum: group_id, name
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV format. Expected headers: group_id, name, species, breed, age, description, status, image_url"})
			return
		}

		// Create header index map
		headerMap := make(map[string]int)
		for i, h := range header {
			headerMap[strings.TrimSpace(strings.ToLower(h))] = i
		}

		// Validate required headers
		if _, ok := headerMap["group_id"]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required column: group_id"})
			return
		}
		if _, ok := headerMap["name"]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required column: name"})
			return
		}

		var animals []models.Animal
		var errors []string
		lineNum := 1

		// Read data rows
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errors = append(errors, fmt.Sprintf("Line %d: Failed to read row", lineNum))
				lineNum++
				continue
			}
			lineNum++

			// Parse group_id
			groupIDStr := strings.TrimSpace(record[headerMap["group_id"]])
			groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Line %d: Invalid group_id '%s'", lineNum, groupIDStr))
				continue
			}

			// Parse name (required)
			name := strings.TrimSpace(record[headerMap["name"]])
			if name == "" {
				errors = append(errors, fmt.Sprintf("Line %d: Name is required", lineNum))
				continue
			}

			animal := models.Animal{
				GroupID: uint(groupID),
				Name:    name,
			}

			// Parse optional fields
			if idx, ok := headerMap["species"]; ok && idx < len(record) {
				animal.Species = strings.TrimSpace(record[idx])
			}
			if idx, ok := headerMap["breed"]; ok && idx < len(record) {
				animal.Breed = strings.TrimSpace(record[idx])
			}
			if idx, ok := headerMap["age"]; ok && idx < len(record) {
				ageStr := strings.TrimSpace(record[idx])
				if ageStr != "" {
					age, err := strconv.Atoi(ageStr)
					if err == nil {
						animal.Age = age
					}
				}
			}
			if idx, ok := headerMap["description"]; ok && idx < len(record) {
				animal.Description = strings.TrimSpace(record[idx])
			}
			if idx, ok := headerMap["status"]; ok && idx < len(record) {
				status := strings.TrimSpace(record[idx])
				if status != "" {
					animal.Status = status
				} else {
					animal.Status = "available"
				}
			} else {
				animal.Status = "available"
			}
			if idx, ok := headerMap["image_url"]; ok && idx < len(record) {
				animal.ImageURL = strings.TrimSpace(record[idx])
			}

			animals = append(animals, animal)
		}

		if len(animals) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "No valid animals to import",
				"errors": errors,
			})
			return
		}

		// Insert animals in batch
		if err := db.Create(&animals).Error; err != nil {
			logger.Error("Failed to import animals", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import animals"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"count":    len(animals),
			"warnings": len(errors),
		}).Info("Successfully imported animals from CSV")

		response := gin.H{
			"message": fmt.Sprintf("Successfully imported %d animals", len(animals)),
			"count":   len(animals),
		}
		if len(errors) > 0 {
			response["warnings"] = errors
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetAllAnimals returns all animals (admin only, for bulk edit page)
func GetAllAnimals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build query with filters
		query := db.Model(&models.Animal{})

		// Status filter
		status := c.Query("status")
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}

		// Group filter
		groupID := c.Query("group_id")
		if groupID != "" {
			query = query.Where("group_id = ?", groupID)
		}

		// Name search filter
		nameSearch := c.Query("name")
		if nameSearch != "" {
			query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(nameSearch)+"%")
		}

		var animals []models.Animal
		if err := query.Order("group_id, name").Find(&animals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
			return
		}

		c.JSON(http.StatusOK, animals)
	}
}

// ExportAnimalCommentsCSV exports all animal comments with animal details to CSV format (admin only)
func ExportAnimalCommentsCSV(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Query("group_id")
		animalID := c.Query("animal_id")

		// Build query to get comments with related data
		query := db.Preload("User").Preload("Tags")

		// If animal_id filter is provided, filter by specific animal
		if animalID != "" {
			query = query.Where("animal_comments.animal_id = ?", animalID)
		} else if groupID != "" {
			// If only group_id filter is provided, join with animals to filter by group
			query = query.Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animals.group_id = ?", groupID)
		}

		var comments []models.AnimalComment
		if err := query.Order("animal_comments.created_at DESC").Find(&comments).Error; err != nil {
			logger.Error("Failed to fetch comments", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
			return
		}

	// Load animal details for each comment
	animalIDs := make([]uint, 0, len(comments))
	for _, comment := range comments {
		animalIDs = append(animalIDs, comment.AnimalID)
	}

	// Get all animals in one query
	var animals []models.Animal
	if len(animalIDs) > 0 {
		if err := db.Where("id IN ?", animalIDs).Find(&animals).Error; err != nil {
			logger.Error("Failed to fetch animals", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animal details"})
			return
		}
	}		// Create animal lookup map
		animalMap := make(map[uint]models.Animal)
		for _, animal := range animals {
			animalMap[animal.ID] = animal
		}

		// Get group details for animals
		groupIDs := make([]uint, 0)
		for _, animal := range animals {
			groupIDs = append(groupIDs, animal.GroupID)
		}

		var groups []models.Group
		if len(groupIDs) > 0 {
			if err := db.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
				logger.Error("Failed to fetch groups", err)
				// Continue without group names
			}
		}

		// Create group lookup map
		groupMap := make(map[uint]string)
		for _, group := range groups {
			groupMap[group.ID] = group.Name
		}

		logger.WithFields(map[string]interface{}{
			"comment_count": len(comments),
			"group_id":      groupID,
			"animal_id":     animalID,
		}).Info("Exporting animal comments to CSV")

		// Set response headers for CSV download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=animal-comments.csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write CSV header
		header := []string{
			"comment_id",
			"animal_id",
			"animal_name",
			"animal_species",
			"animal_breed",
			"animal_status",
			"group_id",
			"group_name",
			"comment_content",
			"comment_author",
			"comment_tags",
			"created_at",
			"updated_at",
		}
		if err := writer.Write(header); err != nil {
			logger.Error("Failed to write CSV header", err)
			return
		}

		// Write comment data
		for _, comment := range comments {
			animal, ok := animalMap[comment.AnimalID]
			if !ok {
				// Skip if animal not found
				continue
			}

			groupName := groupMap[animal.GroupID]

			// Collect tag names
			tagNames := make([]string, 0, len(comment.Tags))
			for _, tag := range comment.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			tagsStr := strings.Join(tagNames, "; ")

			authorName := ""
			if comment.User.Username != "" {
				authorName = comment.User.Username
			}

			record := []string{
				strconv.FormatUint(uint64(comment.ID), 10),
				strconv.FormatUint(uint64(animal.ID), 10),
				animal.Name,
				animal.Species,
				animal.Breed,
				animal.Status,
				strconv.FormatUint(uint64(animal.GroupID), 10),
				groupName,
				comment.Content,
				authorName,
				tagsStr,
				comment.CreatedAt.Format(time.RFC3339),
				comment.UpdatedAt.Format(time.RFC3339),
			}
			if err := writer.Write(record); err != nil {
				logger.Error("Failed to write CSV record", err)
				return
			}
		}
	}
}
