package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

type ProtocolRequest struct {
	Title      string `json:"title" binding:"required,min=2,max=200"`
	Content    string `json:"content" binding:"required,min=10,max=1000"`
	ImageURL   string `json:"image_url,omitempty"`
	OrderIndex int    `json:"order_index"`
}

// UploadProtocolImage handles secure protocol image uploads (group admin or site admin)
func UploadProtocolImage(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
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

		// Detect MIME type from file content
		mimeType := http.DetectContentType(data)

		// Upload to storage provider
		imageURL, _, _, err := storageProvider.UploadImage(ctx, data, mimeType, nil)
		if err != nil {
			logger.Error("Failed to upload image to storage", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}

		logger.WithField("url", imageURL).Info("Protocol image uploaded successfully")
		c.JSON(http.StatusOK, gin.H{"url": imageURL})
	}
}

// GetProtocols returns all protocols for a group
func GetProtocols(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// Verify group has protocols enabled
		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if !group.HasProtocols {
			c.JSON(http.StatusNotFound, gin.H{"error": "Protocols not enabled for this group"})
			return
		}

		var protocols []models.Protocol
		if err := db.WithContext(ctx).
			Where("group_id = ?", groupID).
			Order("order_index ASC, created_at ASC").
			Find(&protocols).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch protocols"})
			return
		}

		c.JSON(http.StatusOK, protocols)
	}
}

// GetProtocol returns a specific protocol by ID
func GetProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		protocolID := c.Param("protocolId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check group access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var protocol models.Protocol
		if err := db.Where("id = ? AND group_id = ?", protocolID, groupID).First(&protocol).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Protocol not found"})
			return
		}

		c.JSON(http.StatusOK, protocol)
	}
}

// CreateProtocol creates a new protocol (group admin or site admin)
func CreateProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req ProtocolRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Verify group exists and has protocols enabled
		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if !group.HasProtocols {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Protocols not enabled for this group"})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		protocol := models.Protocol{
			GroupID:    uint(gid),
			Title:      req.Title,
			Content:    req.Content,
			ImageURL:   req.ImageURL,
			OrderIndex: req.OrderIndex,
		}

		if err := db.Create(&protocol).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create protocol"})
			return
		}

		c.JSON(http.StatusCreated, protocol)
	}
}

// UpdateProtocol updates an existing protocol (group admin or site admin)
func UpdateProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		protocolID := c.Param("protocolId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req ProtocolRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		var protocol models.Protocol
		if err := db.Where("id = ? AND group_id = ?", protocolID, groupID).First(&protocol).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Protocol not found"})
			return
		}

		protocol.Title = req.Title
		protocol.Content = req.Content
		protocol.ImageURL = req.ImageURL
		protocol.OrderIndex = req.OrderIndex

		if err := db.Save(&protocol).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update protocol"})
			return
		}

		c.JSON(http.StatusOK, protocol)
	}
}

// DeleteProtocol deletes a protocol (group admin or site admin)
func DeleteProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		protocolID := c.Param("protocolId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var protocol models.Protocol
		if err := db.Where("id = ? AND group_id = ?", protocolID, groupID).First(&protocol).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Protocol not found"})
			return
		}

		if err := db.Delete(&protocol).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete protocol"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Protocol deleted successfully"})
	}
}
