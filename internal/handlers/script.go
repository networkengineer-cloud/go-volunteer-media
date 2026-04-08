package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

// GetScripts returns all scripts for a group (group members only, group must have has_protocols enabled)
func GetScripts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if !group.HasProtocols {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scripts not enabled for this group"})
			return
		}

		var scripts []models.Script
		if err := db.WithContext(ctx).
			Where("group_id = ?", groupID).
			Order("order_index ASC, created_at ASC").
			Find(&scripts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scripts"})
			return
		}

		c.JSON(http.StatusOK, scripts)
	}
}

// GetScript returns a single script by ID (group members only)
func GetScript(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		scriptID := c.Param("scriptId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var script models.Script
		if err := db.Where("id = ? AND group_id = ?", scriptID, groupID).First(&script).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Script not found"})
			return
		}

		c.JSON(http.StatusOK, script)
	}
}

// CreateScript creates a new script with file upload (group admin or site admin)
func CreateScript(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		groupIDStr := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDStr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		// Verify group exists and has scripts enabled
		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		if !group.HasProtocols {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Scripts not enabled for this group"})
			return
		}

		title := c.PostForm("title")
		if len(title) < 2 || len(title) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title must be between 2 and 200 characters"})
			return
		}
		description := c.PostForm("description")
		if len(description) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Description must be 500 characters or fewer"})
			return
		}
		orderIndexStr := c.DefaultPostForm("order_index", "0")
		orderIndex, _ := strconv.Atoi(orderIndexStr)

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		if err := upload.ValidateDocumentUpload(file, upload.MaxDocumentSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		defer src.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			logger.Error("Failed to read file data", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		fileData := buf.Bytes()

		mimeType := mimeTypeFromFilename(file.Filename)
		uploaderID := userID.(uint)

		// Pre-generate a UUID for fallback postgres path
		scriptUUID := uuid.New().String()

		// Upload to storage provider
		_, blobUUID, blobExt, uploadErr := storageProvider.UploadDocument(ctx, fileData, mimeType, file.Filename)
		var fileURL, blobIdentifier, fileProvider string
		var fileDataForDB []byte

		if uploadErr != nil {
			// Fall back to PostgreSQL storage
			logger.WithFields(map[string]interface{}{"error": uploadErr.Error()}).
				Warn("Failed to upload script to storage provider, falling back to PostgreSQL")
			fileURL = fmt.Sprintf("/api/script-files/%s", scriptUUID)
			blobIdentifier = scriptUUID
			fileProvider = "postgres"
			fileDataForDB = fileData
		} else {
			// Always use our own serve URL so the frontend hits a consistent endpoint
			blobIdentifier = blobUUID + blobExt
			fileURL = fmt.Sprintf("/api/script-files/%s", blobIdentifier)
			fileProvider = storageProvider.Name()
			fileDataForDB = nil
		}

		script := models.Script{
			GroupID:              uint(groupID),
			Title:                title,
			Description:          description,
			OrderIndex:           orderIndex,
			FileURL:              fileURL,
			FileName:             file.Filename,
			FileType:             mimeType,
			FileSize:             len(fileData),
			FileProvider:         fileProvider,
			FileBlobIdentifier:   blobIdentifier,
			FileBlobExtension:    blobExt,
			FileData:             fileDataForDB,
			FileUploadedByUserID: &uploaderID,
		}

		if err := db.Create(&script).Error; err != nil {
			logger.Error("Failed to create script", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create script"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"script_id": script.ID,
			"group_id":  groupID,
			"file_name": file.Filename,
		}).Info("Script created successfully")

		c.JSON(http.StatusCreated, script)
	}
}

// UpdateScript updates a script's metadata and optionally replaces the file (group admin or site admin)
func UpdateScript(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		groupIDStr := c.Param("id")
		scriptIDStr := c.Param("scriptId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDStr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var script models.Script
		if err := db.Where("id = ? AND group_id = ?", scriptIDStr, groupIDStr).First(&script).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Script not found"})
			return
		}

		title := c.PostForm("title")
		if len(title) < 2 || len(title) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title must be between 2 and 200 characters"})
			return
		}
		description := c.PostForm("description")
		if len(description) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Description must be 500 characters or fewer"})
			return
		}
		orderIndexStr := c.DefaultPostForm("order_index", strconv.Itoa(script.OrderIndex))
		orderIndex, _ := strconv.Atoi(orderIndexStr)

		script.Title = title
		script.Description = description
		script.OrderIndex = orderIndex
		script.UpdatedAt = time.Now()

		// Replace file if a new one was provided
		if file, err := c.FormFile("file"); err == nil {
			if err := upload.ValidateDocumentUpload(file, upload.MaxDocumentSize); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
				return
			}

			src, err := file.Open()
			if err != nil {
				logger.Error("Failed to open uploaded file", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
				return
			}
			defer src.Close()

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, src); err != nil {
				logger.Error("Failed to read file data", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
				return
			}
			fileData := buf.Bytes()

			// Delete old blob if it exists
			if script.FileProvider == "azure" && script.FileBlobIdentifier != "" {
				if err := storageProvider.DeleteDocument(ctx, script.FileBlobIdentifier); err != nil {
					logger.WithFields(map[string]interface{}{
						"error":           err.Error(),
						"blob_identifier": script.FileBlobIdentifier,
					}).Warn("Failed to delete old script file from storage")
				}
			}

			mimeType := mimeTypeFromFilename(file.Filename)

			// Pre-generate fallback UUID for postgres path
			replacementUUID := uuid.New().String()

			_, newBlobUUID, newBlobExt, newUploadErr := storageProvider.UploadDocument(ctx, fileData, mimeType, file.Filename)
			var newFileURL, newBlobIdentifier, newFileProvider string
			var newFileData []byte

			if newUploadErr != nil {
				logger.WithFields(map[string]interface{}{"error": newUploadErr.Error()}).
					Warn("Failed to upload replacement script file, falling back to PostgreSQL")
				newBlobIdentifier = replacementUUID
				newFileURL = fmt.Sprintf("/api/script-files/%s", replacementUUID)
				newFileProvider = "postgres"
				newFileData = fileData
			} else {
				newBlobIdentifier = newBlobUUID + newBlobExt
				newFileURL = fmt.Sprintf("/api/script-files/%s", newBlobIdentifier)
				newFileProvider = storageProvider.Name()
				newFileData = nil
			}

			uploaderID := userID.(uint)
			script.FileURL = newFileURL
			script.FileName = file.Filename
			script.FileType = mimeType
			script.FileSize = len(fileData)
			script.FileProvider = newFileProvider
			script.FileBlobIdentifier = newBlobIdentifier
			script.FileBlobExtension = newBlobExt
			script.FileData = newFileData
			script.FileUploadedByUserID = &uploaderID
		}

		if err := db.Save(&script).Error; err != nil {
			logger.Error("Failed to save script", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update script"})
			return
		}

		c.JSON(http.StatusOK, script)
	}
}

// DeleteScript soft-deletes a script and removes its file from storage (group admin or site admin)
func DeleteScript(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		groupIDStr := c.Param("id")
		scriptIDStr := c.Param("scriptId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDStr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var script models.Script
		if err := db.Where("id = ? AND group_id = ?", scriptIDStr, groupIDStr).First(&script).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Script not found"})
			return
		}

		// Delete file from storage if stored in Azure
		if script.FileProvider == "azure" && script.FileBlobIdentifier != "" {
			if err := storageProvider.DeleteDocument(ctx, script.FileBlobIdentifier); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":           err.Error(),
					"blob_identifier": script.FileBlobIdentifier,
				}).Warn("Failed to delete script file from storage, continuing with DB deletion")
			}
		}

		// For postgres-backed scripts, clear the binary data before soft-deleting
		// to avoid retaining potentially large bytea blobs in soft-deleted rows.
		if script.FileProvider == "postgres" && len(script.FileData) > 0 {
			if err := db.Model(&script).Update("file_data", nil).Error; err != nil {
				logger.WithFields(map[string]interface{}{"script_id": script.ID}).
					Warn("Failed to clear file data before delete, proceeding anyway")
			}
		}

		if err := db.Delete(&script).Error; err != nil {
			logger.Error("Failed to delete script", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete script"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Script deleted successfully"})
	}
}

// ServeScriptFile serves the binary file for a script (group members only)
// The URL parameter :uuid is the FileBlobIdentifier for Azure or the script numeric ID for postgres storage.
func ServeScriptFile(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uuidOrID := c.Param("uuid")

		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDValue.(uint)
		isAdminValue, _ := c.Get("is_admin")
		isAdmin, _ := isAdminValue.(bool)

		// Try to look up script by blob identifier (UUID string set at upload time)
		var script models.Script
		if err := db.WithContext(ctx).Where("file_blob_identifier = ?", uuidOrID).First(&script).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Script not found"})
			return
		}

		// Authorization: verify user is a member of the script's group or is a site admin
		if !isAdmin {
			var count int64
			if err := db.WithContext(ctx).
				Model(&models.UserGroup{}).
				Where("user_id = ? AND group_id = ?", userID, script.GroupID).
				Count(&count).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
				return
			}
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: You must be a member of this group to view this script"})
				return
			}
		}

		if script.FileProvider == "azure" && script.FileBlobIdentifier != "" {
			data, mimeType, err := storageProvider.GetDocument(ctx, script.FileBlobIdentifier)
			if err != nil {
				if err == storage.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Script file not found in storage"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve script file"})
				}
				return
			}
			c.Header("Content-Type", mimeType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(script.FileName)))
			c.Header("Content-Length", strconv.Itoa(len(data)))
			c.Header("Cache-Control", "private, max-age=3600")
			c.Data(http.StatusOK, mimeType, data)
		} else {
			if len(script.FileData) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Script file data not available"})
				return
			}
			c.Header("Content-Type", script.FileType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(script.FileName)))
			c.Header("Content-Length", strconv.Itoa(len(script.FileData)))
			c.Header("Cache-Control", "private, max-age=3600")
			c.Data(http.StatusOK, script.FileType, script.FileData)
		}
	}
}

// SetAnimalScripts replaces all linked scripts for an animal in a single call (group admin or site admin)
// Body: { "script_ids": [1, 2, 3] }  — passing an empty array removes all links
func SetAnimalScripts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupIDStr := c.Param("id")
		animalIDStr := c.Param("animalId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDStr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		animalID, err := strconv.ParseUint(animalIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid animal ID"})
			return
		}

		var req struct {
			ScriptIDs []uint `json:"script_ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		var scripts []models.Script
		if len(req.ScriptIDs) > 0 {
			if err := db.Where("id IN ? AND group_id = ?", req.ScriptIDs, groupID).Find(&scripts).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate scripts"})
				return
			}
			if len(scripts) != len(req.ScriptIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "One or more scripts not found in this group"})
				return
			}
		}

		// Replace replaces the entire association set atomically
		if err := db.Model(&animal).Association("Scripts").Replace(&scripts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update animal scripts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Animal scripts updated successfully"})
	}
}

// mimeTypeFromFilename returns a MIME type for common document extensions.
func mimeTypeFromFilename(filename string) string {
	switch path.Ext(filename) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// sanitizeFilename strips characters that could be used for HTTP header injection
// (CR, LF, and double-quote). This prevents a malicious filename from breaking
// the Content-Disposition header.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"\r", "",
		"\n", "",
		"\"", "'",
	)
	return r.Replace(name)
}
