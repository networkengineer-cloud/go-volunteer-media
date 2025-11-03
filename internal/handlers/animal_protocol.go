package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type AnimalProtocolRequest struct {
	Content  string `json:"content" binding:"required"`
	ImageURL string `json:"image_url"`
}

// GetAnimalProtocols returns all protocol entries for an animal
func GetAnimalProtocols(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		var protocols []models.AnimalProtocol
		if err := db.Preload("User").Where("animal_id = ?", animalID).Order("created_at DESC").Find(&protocols).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch protocols"})
			return
		}

		c.JSON(http.StatusOK, protocols)
	}
}

// CreateAnimalProtocol creates a new protocol entry on an animal
func CreateAnimalProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		var req AnimalProtocolRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate content length (max 1000 chars)
		if len(strings.TrimSpace(req.Content)) > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Protocol content must be 1000 characters or less"})
			return
		}

		aid, err := strconv.ParseUint(animalID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid animal ID"})
			return
		}

		protocol := models.AnimalProtocol{
			AnimalID: uint(aid),
			UserID:   userID.(uint),
			Content:  strings.TrimSpace(req.Content),
			ImageURL: req.ImageURL,
		}

		if err := db.Create(&protocol).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create protocol"})
			return
		}

		// Reload with user info
		if err := db.Preload("User").First(&protocol, protocol.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load protocol"})
			return
		}

		c.JSON(http.StatusCreated, protocol)
	}
}

// DeleteAnimalProtocol deletes a protocol entry (admin only)
func DeleteAnimalProtocol(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		protocolID := c.Param("protocolId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Only admins can delete protocols
		if !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can delete protocols"})
			return
		}

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

		// Verify protocol exists and belongs to this animal
		var protocol models.AnimalProtocol
		if err := db.Where("id = ? AND animal_id = ?", protocolID, animalID).First(&protocol).Error; err != nil {
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
