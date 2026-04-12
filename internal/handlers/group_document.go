package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/convert"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

// GetGroupDocuments returns all documents for a group (group members only).
// Unlike Scripts, documents are available to all groups regardless of has_protocols.
func GetGroupDocuments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var documents []models.GroupDocument
		if err := db.WithContext(ctx).
			Select("id, created_at, updated_at, group_id, title, description, order_index, "+
				"file_url, file_name, file_type, file_size, file_provider, "+
				"file_blob_identifier, file_blob_extension, file_uploaded_by_user_id").
			Where("group_id = ?", groupID).
			Order("order_index ASC, created_at ASC").
			Find(&documents).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
			return
		}

		c.JSON(http.StatusOK, documents)
	}
}

// UploadGroupDocument uploads a new document to a group (group admin or site admin only).
func UploadGroupDocument(db *gorm.DB, storageProvider storage.Provider, converter convert.Converter) gin.HandlerFunc {
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

		// Verify group exists
		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
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
		orderIndex, orderIndexErr := strconv.Atoi(orderIndexStr)
		if orderIndexErr != nil || orderIndex < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order_index: must be a non-negative integer"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		if err := upload.ValidateDocumentUpload(file, upload.MaxDocumentSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

		// Convert DOCX and XLSX to PDF so all documents can be viewed inline in the browser.
		// PDFs pass through unchanged. Conversion failure rejects the upload.
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".pdf" {
			pdfData, convErr := converter.ToPDF(ctx, fileData, ext)
			if convErr != nil {
				logger.WithFields(map[string]interface{}{"error": convErr.Error(), "ext": ext}).
					Warn("Document conversion to PDF failed")
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error": "File could not be converted to PDF. Please check the file and try again.",
				})
				return
			}
			fileData = pdfData
			file.Filename = strings.TrimSuffix(file.Filename, ext) + ".pdf"
		}

		mimeType := upload.MimeTypeFromFilename(file.Filename)
		uploaderID := userID.(uint)

		// Pre-generate a UUID for fallback postgres path
		docUUID := uuid.New().String()

		// Upload to storage provider.
		// The first return value (provider URL) is intentionally discarded: all document
		// downloads are proxied through /api/group-documents/:uuid so that the auth check
		// in ServeGroupDocument is always enforced, even for Azure-backed storage.
		_, blobUUID, blobExt, uploadErr := storageProvider.UploadDocument(ctx, fileData, mimeType, file.Filename)
		var fileURL, blobIdentifier, fileProvider string
		var fileDataForDB []byte

		if uploadErr != nil {
			// Fall back to PostgreSQL storage
			logger.WithFields(map[string]interface{}{"error": uploadErr.Error()}).
				Warn("Failed to upload document to storage provider, falling back to PostgreSQL")
			fileURL = fmt.Sprintf("/api/group-documents/%s", docUUID)
			blobIdentifier = docUUID
			fileProvider = storage.ProviderPostgres
			fileDataForDB = fileData
		} else {
			blobIdentifier = blobUUID + blobExt
			fileURL = fmt.Sprintf("/api/group-documents/%s", blobIdentifier)
			fileProvider = storageProvider.Name()
			if fileProvider == storage.ProviderPostgres {
				fileDataForDB = fileData
			} else {
				fileDataForDB = nil
			}
		}

		doc := models.GroupDocument{
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

		if err := db.Create(&doc).Error; err != nil {
			logger.Error("Failed to create group document", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"document_id": doc.ID,
			"group_id":    groupID,
			"file_name":   file.Filename,
		}).Info("Group document created successfully")

		c.JSON(http.StatusCreated, doc)
	}
}

// DeleteGroupDocument soft-deletes a group document and removes its file from storage (group admin or site admin).
func DeleteGroupDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		groupIDStr := c.Param("id")
		docIDStr := c.Param("docId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDStr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var doc models.GroupDocument
		if err := db.Where("id = ? AND group_id = ?", docIDStr, groupIDStr).First(&doc).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}

		// Delete file from storage if stored externally (non-postgres provider)
		if doc.FileProvider == storage.ProviderAzure && doc.FileBlobIdentifier != "" {
			if err := storageProvider.DeleteDocument(ctx, doc.FileBlobIdentifier); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":           err.Error(),
					"blob_identifier": doc.FileBlobIdentifier,
				}).Warn("Failed to delete document file from storage, continuing with DB deletion")
			}
		}

		// For postgres-backed documents, clear the binary data before soft-deleting
		if doc.FileProvider == storage.ProviderPostgres && len(doc.FileData) > 0 {
			if err := db.Model(&doc).Update("file_data", nil).Error; err != nil {
				logger.WithFields(map[string]interface{}{"document_id": doc.ID}).
					Warn("Failed to clear file data before delete, proceeding anyway")
			}
		}

		if err := db.Delete(&doc).Error; err != nil {
			logger.Error("Failed to delete group document", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
	}
}

// ServeGroupDocument serves the binary file for a group document (group members only).
// The URL parameter :uuid is the FileBlobIdentifier.
func ServeGroupDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		uuidParam := c.Param("uuid")

		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDValue.(uint)
		isAdminValue, _ := c.Get("is_admin")
		isAdmin, _ := isAdminValue.(bool)

		// Look up document by blob identifier
		var doc models.GroupDocument
		if err := db.WithContext(ctx).Where("file_blob_identifier = ?", uuidParam).First(&doc).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}

		// Authorization: verify user is a member of the document's group or is a site admin
		if !isAdmin {
			var count int64
			if err := db.WithContext(ctx).
				Model(&models.UserGroup{}).
				Where("user_id = ? AND group_id = ?", userID, doc.GroupID).
				Count(&count).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
				return
			}
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: You must be a member of this group to view this document"})
				return
			}
		}

		if doc.FileProvider == storage.ProviderAzure && doc.FileBlobIdentifier != "" {
			data, mimeType, err := storageProvider.GetDocument(ctx, doc.FileBlobIdentifier)
			if err != nil {
				if err == storage.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Document file not found in storage"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve document file"})
				}
				return
			}
			c.Header("Content-Type", mimeType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(doc.FileName)))
			c.Header("Content-Length", strconv.Itoa(len(data)))
			c.Header("Cache-Control", "private, max-age=3600")
			c.Data(http.StatusOK, mimeType, data)
		} else {
			if len(doc.FileData) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Document file data not available"})
				return
			}
			c.Header("Content-Type", doc.FileType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(doc.FileName)))
			c.Header("Content-Length", strconv.Itoa(len(doc.FileData)))
			c.Header("Cache-Control", "private, max-age=3600")
			c.Data(http.StatusOK, doc.FileType, doc.FileData)
		}
	}
}
