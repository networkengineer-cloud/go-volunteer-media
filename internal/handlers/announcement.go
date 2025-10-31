package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type AnnouncementRequest struct {
	Title     string `json:"title" binding:"required,min=2,max=200"`
	Content   string `json:"content" binding:"required,min=10"`
	SendEmail bool   `json:"send_email"`
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

// CreateAnnouncement creates a new announcement and optionally sends emails (admin only)
func CreateAnnouncement(db *gorm.DB, emailService *email.Service) gin.HandlerFunc {
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
			UserID:    userID.(uint),
			Title:     req.Title,
			Content:   req.Content,
			SendEmail: req.SendEmail,
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
