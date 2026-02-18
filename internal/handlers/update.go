package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type UpdateRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	ImageURL    string `json:"image_url"`
	SendGroupMe bool   `json:"send_groupme"`
}

// GetUpdates returns all updates for a group
func GetUpdates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var updates []models.Update
		if err := db.Preload("User").Where("group_id = ?", groupID).Order("created_at DESC").Find(&updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updates"})
			return
		}

		c.JSON(http.StatusOK, updates)
	}
}

// CreateUpdate creates a new update/post in a group
func CreateUpdate(db *gorm.DB, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check access
		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var req UpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		gid, err := strconv.ParseUint(groupID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		userIDUint, _ := userID.(uint)
		update := models.Update{
			GroupID:     uint(gid),
			UserID:      userIDUint,
			Title:       req.Title,
			Content:     req.Content,
			ImageURL:    req.ImageURL,
			SendGroupMe: req.SendGroupMe,
		}

		if err := db.WithContext(ctx).Create(&update).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create update"})
			return
		}

		// Reload with user info
		if err := db.WithContext(ctx).Preload("User").First(&update, update.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load update"})
			return
		}

		// Send to GroupMe if requested and service is available
		if req.SendGroupMe && groupMeService != nil {
			// Use background context for async GroupMe sending
			go func() {
				bgCtx := context.Background()
				if err := sendUpdateToGroupMe(bgCtx, db, groupMeService, uint(gid), update.Title, update.Content); err != nil {
					logging.WithContext(bgCtx).Error("Error sending update to GroupMe", err)
				}
			}()
		}

		c.JSON(http.StatusCreated, update)
	}
}

// sendUpdateToGroupMe sends an update to a group's GroupMe chat
func sendUpdateToGroupMe(ctx context.Context, db *gorm.DB, groupMeService *groupme.Service, groupID uint, title, content string) error {
	logger := logging.WithContext(ctx)

	// Fetch the group
	var group models.Group
	if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		logger.Error("Failed to fetch group for GroupMe send", err)
		return err
	}

	// Check if GroupMe is enabled for this group
	if !group.GroupMeEnabled || group.GroupMeBotID == "" {
		logger.WithFields(map[string]interface{}{
			"group_id":   groupID,
			"group_name": group.Name,
		}).Info("GroupMe not enabled for group, skipping send")
		return nil
	}

	// Send the announcement
	if err := groupMeService.SendAnnouncement(group.GroupMeBotID, title, content); err != nil {
		logger.WithFields(map[string]interface{}{
			"group_id":   groupID,
			"group_name": group.Name,
		}).Error("Failed to send update to GroupMe", err)
		return err
	}

	logger.WithFields(map[string]interface{}{
		"group_id":   groupID,
		"group_name": group.Name,
	}).Info("Update sent to GroupMe successfully")
	return nil
}
