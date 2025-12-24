package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

// UploadAnimalProtocolDocument handles uploading a protocol document (PDF or DOCX) for an animal
func UploadAnimalProtocolDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		// Get animal ID from URL parameter
		groupIDStr := c.Param("id")
		animalIDStr := c.Param("animalId")

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

		// Get user ID from context
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
		userID := userIDVal.(uint)

		// Check if animal exists and belongs to the group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found in this group"})
			} else {
				logger.Error("Failed to query animal", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify animal"})
			}
			return
		}

		// Get uploaded file
		file, err := c.FormFile("document")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No document uploaded"})
			return
		}

		// Validate file upload (size, type, content)
		if err := upload.ValidateDocumentUpload(file, upload.MaxDocumentSize); err != nil {
			logger.Error("Document validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document: " + err.Error()})
			return
		}

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process document"})
			return
		}
		defer src.Close()

		// Read document data into buffer
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			logger.Error("Failed to read document data", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process document"})
			return
		}

		documentData := buf.Bytes()

		// Determine MIME type based on file extension
		var mimeType string
		ext := path.Ext(file.Filename)
		switch ext {
		case ".pdf":
			mimeType = "application/pdf"
		case ".docx":
			mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		default:
			mimeType = "application/octet-stream"
		}

		// Generate unique document identifier
		documentUUID := uuid.New().String()

		// Upload to storage provider
		storageURL, blobIdentifier, err := storageProvider.UploadDocument(ctx, documentData, mimeType, file.Filename)
		var documentURL string
		var documentDataForDB []byte
		var storageProviderName string

		if err != nil {
			// If storage provider upload fails, fall back to PostgreSQL
			logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Failed to upload to storage provider, falling back to PostgreSQL")

			documentURL = fmt.Sprintf("/api/documents/%s", documentUUID)
			documentDataForDB = documentData
			storageProviderName = "postgres"
			blobIdentifier = ""
		} else {
			// Successfully uploaded to storage provider
			documentURL = storageURL
			documentDataForDB = nil // Don't store in DB when using external storage
			storageProviderName = "azure"
		}

		// Update animal record with protocol document
		animal.ProtocolDocumentURL = documentURL
		animal.ProtocolDocumentName = file.Filename
		animal.ProtocolDocumentData = documentDataForDB
		animal.ProtocolDocumentType = mimeType
		animal.ProtocolDocumentSize = len(documentData)
		animal.ProtocolDocumentUserID = &userID
		animal.ProtocolDocumentProvider = storageProviderName
		animal.ProtocolDocumentBlobIdentifier = blobIdentifier
		animal.UpdatedAt = time.Now()

		if err := db.Save(&animal).Error; err != nil {
			logger.Error("Failed to save animal with protocol document", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save protocol document"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"animal_id":        animalID,
			"group_id":         groupID,
			"document_url":     documentURL,
			"document_name":    file.Filename,
			"size":             len(documentData),
			"storage_provider": storageProviderName,
		}).Info("Protocol document uploaded successfully")

		c.JSON(http.StatusOK, gin.H{
			"url":         documentURL,
			"name":        file.Filename,
			"size":        len(documentData),
			"type":        mimeType,
			"uploaded_by": userID,
		})
	}
}

// ServeAnimalProtocolDocument serves a protocol document using the configured storage provider
// Requires authentication and verifies user is member of the animal's group
func ServeAnimalProtocolDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		documentUUID := c.Param("uuid")
		documentURL := fmt.Sprintf("/api/documents/%s", documentUUID)

		// AuthRequired middleware stores user claims in context
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDValue.(uint)

		isAdminValue, _ := c.Get("is_admin")
		isAdmin, _ := isAdminValue.(bool)

		// Find animal with the protocol document
		var animal models.Animal
		if err := db.WithContext(ctx).Where("protocol_document_url = ?", documentURL).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}

		// Authorization: Verify user is member of the animal's group (or is admin)
		if !isAdmin {
			var count int64
			if err := db.WithContext(ctx).
				Model(&models.UserGroup{}).
				Where("user_id = ? AND group_id = ?", userID, animal.GroupID).
				Count(&count).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
				return
			}
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: You must be a member of this group to view this document"})
				return
			}
		}

		// Check which storage provider was used for this document
		if animal.ProtocolDocumentProvider == "azure" && animal.ProtocolDocumentBlobIdentifier != "" {
			// Retrieve from Azure Blob Storage
			data, mimeType, err := storageProvider.GetDocument(ctx, animal.ProtocolDocumentBlobIdentifier)
			if err != nil {
				if err == storage.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Document not found in storage"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve document"})
				}
				return
			}

			// Set headers for document download
			c.Header("Content-Type", mimeType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", animal.ProtocolDocumentName))
			c.Header("Content-Length", strconv.Itoa(len(data)))
			c.Header("Cache-Control", "private, max-age=3600") // Cache for 1 hour
			c.Data(http.StatusOK, mimeType, data)
		} else {
			// Legacy: retrieve from PostgreSQL database
			if len(animal.ProtocolDocumentData) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Document data not available"})
				return
			}

			// Set headers for document download
			c.Header("Content-Type", animal.ProtocolDocumentType)
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", animal.ProtocolDocumentName))
			c.Header("Content-Length", strconv.Itoa(len(animal.ProtocolDocumentData)))
			c.Header("Cache-Control", "private, max-age=3600") // Cache for 1 hour
			c.Data(http.StatusOK, animal.ProtocolDocumentType, animal.ProtocolDocumentData)
		}
	}
}

// DeleteAnimalProtocolDocument removes the protocol document from an animal
func DeleteAnimalProtocolDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)

		// Get animal ID from URL parameter
		groupIDStr := c.Param("id")
		animalIDStr := c.Param("animalId")

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

		// Check if animal exists and belongs to the group
		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found in this group"})
			} else {
				logger.Error("Failed to query animal", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify animal"})
			}
			return
		}

		// Delete from storage provider if using Azure
		if animal.ProtocolDocumentProvider == "azure" && animal.ProtocolDocumentBlobIdentifier != "" {
			if err := storageProvider.DeleteDocument(ctx, animal.ProtocolDocumentBlobIdentifier); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":           err.Error(),
					"blob_identifier": animal.ProtocolDocumentBlobIdentifier,
				}).Warn("Failed to delete document from storage provider, continuing with database deletion")
			}
		}

		// Clear protocol document fields
		animal.ProtocolDocumentURL = ""
		animal.ProtocolDocumentName = ""
		animal.ProtocolDocumentData = nil
		animal.ProtocolDocumentType = ""
		animal.ProtocolDocumentSize = 0
		animal.ProtocolDocumentUserID = nil
		animal.ProtocolDocumentProvider = ""
		animal.ProtocolDocumentBlobIdentifier = ""
		animal.UpdatedAt = time.Now()

		if err := db.Save(&animal).Error; err != nil {
			logger.Error("Failed to remove protocol document from animal", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove protocol document"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"animal_id": animalID,
			"group_id":  groupID,
		}).Info("Protocol document removed successfully")

		c.JSON(http.StatusOK, gin.H{"message": "Protocol document removed successfully"})
	}
}
