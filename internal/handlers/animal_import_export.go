package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

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
		if err := writer.Write([]string{"id", "group_id", "name", "species", "breed", "age", "estimated_birth_date", "description", "trainer_notes", "status", "image_url"}); err != nil {
			logger.Error("Failed to write CSV header", err)
			return
		}

		// Write animal data
		for _, animal := range animals {
			// Format estimated birth date as ISO date string
			estimatedBirthDate := ""
			if animal.EstimatedBirthDate != nil {
				estimatedBirthDate = animal.EstimatedBirthDate.Format("2006-01-02")
			}

			record := []string{
				strconv.FormatUint(uint64(animal.ID), 10),
				strconv.FormatUint(uint64(animal.GroupID), 10),
				animal.Name,
				animal.Species,
				animal.Breed,
				strconv.Itoa(animal.Age),
				estimatedBirthDate,
				animal.Description,
				animal.TrainerNotes,
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
			if idx, ok := headerMap["estimated_birth_date"]; ok && idx < len(record) {
				dateStr := strings.TrimSpace(record[idx])
				if dateStr != "" {
					if parsedDate, parseErr := time.Parse("2006-01-02", dateStr); parseErr == nil {
						animal.EstimatedBirthDate = &parsedDate
						// Auto-compute Age from birth date
						animal.Age = animal.AgeYearsFromBirthDate()
					}
				}
			}
			if idx, ok := headerMap["trainer_notes"]; ok && idx < len(record) {
				animal.TrainerNotes = strings.TrimSpace(record[idx])
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

// ExportAnimalCommentsCSV exports all animal comments with animal details to CSV format (admin only)
func ExportAnimalCommentsCSV(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Query("group_id")
		animalID := c.Query("animal_id")
		tagFilter := c.Query("tags") // Comma-separated tag names

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

		// Apply tag filter if provided (multiple tags = OR logic)
		if tagFilter != "" {
			tagNames := strings.Split(tagFilter, ",")
			// Trim whitespace from tag names
			for i, name := range tagNames {
				tagNames[i] = strings.TrimSpace(name)
			}

			// Join with tag tables to filter by tags
			query = query.Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
				Joins("JOIN comment_tags ON comment_tags.id = animal_comment_tags.comment_tag_id").
				Where("comment_tags.name IN ?", tagNames).
				Group("animal_comments.id")
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
		}

		// Create animal lookup map
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
			"tag_filter":    tagFilter,
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
