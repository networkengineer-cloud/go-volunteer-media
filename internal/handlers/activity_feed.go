package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// ActivityItem represents a unified activity feed item
type ActivityItem struct {
	ID        uint                `json:"id"`
	Type      string              `json:"type"` // "comment", "announcement"
	CreatedAt time.Time           `json:"created_at"`
	UserID    uint                `json:"user_id"`
	User      *models.User        `json:"user,omitempty"`
	Content   string              `json:"content"`
	Title     string              `json:"title,omitempty"` // For announcements
	ImageURL  string              `json:"image_url,omitempty"`
	AnimalID  *uint               `json:"animal_id,omitempty"` // For comments
	Animal    *models.Animal      `json:"animal,omitempty"`    // For comments
	Tags      []models.CommentTag `json:"tags,omitempty"`      // For comments
}

// GetGroupActivityFeed returns a unified activity feed combining updates/announcements and comments
func GetGroupActivityFeed(db *gorm.DB) gin.HandlerFunc {
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

		// Get limit and offset parameters for pagination
		limit := 20
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get filter type (all, comments, announcements)
		filterType := c.Query("type")

		// Initialize with empty slice to ensure we never return nil
		activityItems := make([]ActivityItem, 0)

		// Fetch announcements (Updates) if not filtering for comments only
		if filterType == "" || filterType == "all" || filterType == "announcements" {
			var updates []models.Update
			err := db.WithContext(ctx).
				Where("group_id = ?", groupID).
				Preload("User").
				Order("created_at DESC").
				Find(&updates).Error

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
				return
			}

			for _, update := range updates {
				activityItems = append(activityItems, ActivityItem{
					ID:        update.ID,
					Type:      "announcement",
					CreatedAt: update.CreatedAt,
					UserID:    update.UserID,
					User:      &update.User,
					Content:   update.Content,
					Title:     update.Title,
					ImageURL:  update.ImageURL,
				})
			}
		}

		// Fetch comments if not filtering for announcements only
		if filterType == "" || filterType == "all" || filterType == "comments" {
			// First get all animals in this group
			var animals []models.Animal
			if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&animals).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
				return
			}

			// Get animal IDs
			var animalIDs []uint
			animalMap := make(map[uint]models.Animal)
			for _, animal := range animals {
				animalIDs = append(animalIDs, animal.ID)
				animalMap[animal.ID] = animal
			}

			if len(animalIDs) > 0 {
				// Get comments from these animals
				var comments []models.AnimalComment
				err := db.WithContext(ctx).
					Where("animal_id IN ?", animalIDs).
					Preload("User").
					Preload("Tags").
					Order("created_at DESC").
					Find(&comments).Error

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
					return
				}

				for _, comment := range comments {
					animal := animalMap[comment.AnimalID]
					activityItems = append(activityItems, ActivityItem{
						ID:        comment.ID,
						Type:      "comment",
						CreatedAt: comment.CreatedAt,
						UserID:    comment.UserID,
						User:      &comment.User,
						Content:   comment.Content,
						ImageURL:  comment.ImageURL,
						AnimalID:  &comment.AnimalID,
						Animal:    &animal,
						Tags:      comment.Tags,
					})
				}
			}
		}

		// Sort all items by creation time (newest first)
		// This is done in-memory for simplicity; for production consider DB-level sorting
		for i := 0; i < len(activityItems); i++ {
			for j := i + 1; j < len(activityItems); j++ {
				if activityItems[i].CreatedAt.Before(activityItems[j].CreatedAt) {
					activityItems[i], activityItems[j] = activityItems[j], activityItems[i]
				}
			}
		}

		// Apply pagination
		total := len(activityItems)
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}

		paginatedItems := activityItems[start:end]

		// Ensure we return an empty array instead of nil
		if paginatedItems == nil {
			paginatedItems = []ActivityItem{}
		}

		// Return response with pagination metadata
		c.JSON(http.StatusOK, gin.H{
			"items":   paginatedItems,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": end < total,
		})
	}
}
