package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type AnnouncementRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=200"`
	Content     string `json:"content" binding:"required,min=10"`
	SendEmail   bool   `json:"send_email"`
	SendGroupMe bool   `json:"send_groupme"`
}

// GetAnnouncements returns all announcements (accessible to all authenticated users)
func GetAnnouncements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var announcements []models.Announcement
		if err := db.WithContext(ctx).Preload("User").Order("created_at DESC").Limit(10).Find(&announcements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
			return
		}

		c.JSON(http.StatusOK, announcements)
	}
}

// CreateAnnouncement creates a new announcement and optionally sends emails and GroupMe messages (admin only)
func CreateAnnouncement(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var req AnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		announcement := models.Announcement{
			UserID:      userID.(uint),
			Title:       req.Title,
			Content:     req.Content,
			SendEmail:   req.SendEmail,
			SendGroupMe: req.SendGroupMe,
		}

		if err := db.WithContext(ctx).Create(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create announcement"})
			return
		}

		// Load the user information for the response
		if err := db.WithContext(ctx).Preload("User").First(&announcement, announcement.ID).Error; err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to load announcement user", err)
		}

		// Send emails if requested and email service is configured
		if req.SendEmail && emailService.IsConfigured() {
			// Use background context for async email sending
			go func() {
				bgCtx := context.Background()
				if err := sendAnnouncementEmails(bgCtx, db, emailService, announcement.Title, announcement.Content); err != nil {
					logging.WithContext(bgCtx).Error("Error sending announcement emails", err)
				}
			}()
		}

		// Send GroupMe messages if requested
		if req.SendGroupMe && groupMeService != nil {
			// Use background context for async GroupMe sending
			go func() {
				bgCtx := context.Background()
				if err := sendAnnouncementToGroupMe(bgCtx, db, groupMeService, announcement.Title, announcement.Content); err != nil {
					logging.WithContext(bgCtx).Error("Error sending announcement to GroupMe", err)
				}
			}()
		}

		c.JSON(http.StatusCreated, announcement)
	}
}

// DeleteAnnouncement deletes an announcement (admin only)
func DeleteAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		announcementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid announcement ID"})
			return
		}

		if err := db.WithContext(ctx).Delete(&models.Announcement{}, uint(announcementID)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete announcement"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Announcement deleted successfully"})
	}
}

// sendAnnouncementEmails sends announcement emails to all users who have opted in
func sendAnnouncementEmails(ctx context.Context, db *gorm.DB, emailService *email.Service, title, content string) error {
	logger := logging.WithContext(ctx)

	var users []models.User
	if err := db.WithContext(ctx).Where("email_notifications_enabled = ?", true).Find(&users).Error; err != nil {
		logger.Error("Failed to fetch users for email notifications", err)
		return err
	}

	logger.WithField("user_count", len(users)).Info("Sending announcement emails to users")
	successCount := 0
	for _, user := range users {
		if err := emailService.SendAnnouncementEmail(user.Email, title, content); err != nil {
			// Don't log email addresses to prevent PII leakage - just log the error
			logger.Error("Failed to send announcement email to user", err)
		} else {
			successCount++
		}
	}
	logger.WithFields(map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(users),
	}).Info("Announcement email sending completed")
	return nil
}

// sendAnnouncementToGroupMe sends announcement to all GroupMe-enabled groups
func sendAnnouncementToGroupMe(ctx context.Context, db *gorm.DB, groupMeService *groupme.Service, title, content string) error {
	logger := logging.WithContext(ctx)

	// Fetch all groups with GroupMe enabled
	var groups []models.Group
	if err := db.WithContext(ctx).Where("groupme_enabled = ? AND groupme_bot_id != ?", true, "").Find(&groups).Error; err != nil {
		logger.Error("Failed to fetch GroupMe-enabled groups", err)
		return err
	}

	logger.WithField("group_count", len(groups)).Info("Sending announcement to GroupMe groups")
	successCount := 0
	for _, group := range groups {
		if err := groupMeService.SendAnnouncement(group.GroupMeBotID, title, content); err != nil {
			logger.WithFields(map[string]interface{}{
				"group_id":   group.ID,
				"group_name": group.Name,
			}).Error("Failed to send announcement to GroupMe", err)
		} else {
			successCount++
		}
	}
	logger.WithFields(map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(groups),
	}).Info("GroupMe announcement sending completed")
	return nil
}

// CreateGroupAnnouncement creates a group-specific announcement (group admin or site admin)
// This allows group admins to send announcements with email and GroupMe notifications
// to members of their group
func CreateGroupAnnouncement(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req AnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		announcement := models.Announcement{
			UserID:      userID.(uint),
			Title:       req.Title,
			Content:     req.Content,
			SendEmail:   req.SendEmail,
			SendGroupMe: req.SendGroupMe,
		}

		if err := db.WithContext(ctx).Create(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create announcement"})
			return
		}

		// Load the user information for the response
		if err := db.WithContext(ctx).Preload("User").First(&announcement, announcement.ID).Error; err != nil {
			logger := middleware.GetLogger(c)
			logger.Error("Failed to load announcement user", err)
		}

		// Send emails if requested and email service is configured
		// Only send to group members who have opted in
		if req.SendEmail && emailService.IsConfigured() {
			go func() {
				bgCtx := context.Background()
				if err := sendGroupAnnouncementEmails(bgCtx, db, emailService, group.ID, announcement.Title, announcement.Content); err != nil {
					logging.WithContext(bgCtx).Error("Error sending group announcement emails", err)
				}
			}()
		}

		// Send GroupMe message if requested and group has GroupMe enabled
		if req.SendGroupMe && groupMeService != nil && group.GroupMeEnabled && group.GroupMeBotID != "" {
			go func() {
				bgCtx := context.Background()
				if err := groupMeService.SendAnnouncement(group.GroupMeBotID, announcement.Title, announcement.Content); err != nil {
					logging.WithContext(bgCtx).WithFields(map[string]interface{}{
						"group_id":   group.ID,
						"group_name": group.Name,
					}).Error("Failed to send announcement to GroupMe", err)
				}
			}()
		}

		c.JSON(http.StatusCreated, announcement)
	}
}

// sendGroupAnnouncementEmails sends announcement emails to group members who have opted in
func sendGroupAnnouncementEmails(ctx context.Context, db *gorm.DB, emailService *email.Service, groupID uint, title, content string) error {
	logger := logging.WithContext(ctx)

	// Fetch group members who have email notifications enabled
	var users []models.User
	if err := db.WithContext(ctx).
		Joins("JOIN user_groups ON user_groups.user_id = users.id").
		Where("user_groups.group_id = ? AND users.email_notifications_enabled = ?", groupID, true).
		Find(&users).Error; err != nil {
		logger.Error("Failed to fetch group members for email notifications", err)
		return err
	}

	logger.WithFields(map[string]interface{}{
		"user_count": len(users),
		"group_id":   groupID,
	}).Info("Sending group announcement emails to members")
	
	successCount := 0
	for _, user := range users {
		if err := emailService.SendAnnouncementEmail(user.Email, title, content); err != nil {
			// Don't log email addresses to prevent PII leakage - just log the error
			logger.Error("Failed to send announcement email to user", err)
		} else {
			successCount++
		}
	}
	logger.WithFields(map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(users),
		"group_id":      groupID,
	}).Info("Group announcement email sending completed")
	return nil
}
