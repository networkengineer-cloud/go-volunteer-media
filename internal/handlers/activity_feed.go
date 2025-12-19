package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// ActivityItem represents a unified activity feed item
type ActivityItem struct {
	ID        uint                     `json:"id"`
	Type      string                   `json:"type"` // "comment", "announcement"
	CreatedAt time.Time                `json:"created_at"`
	UserID    uint                     `json:"user_id"`
	User      *models.User             `json:"user,omitempty"`
	Content   string                   `json:"content"`
	Title     string                   `json:"title,omitempty"` // For announcements
	ImageURL  string                   `json:"image_url,omitempty"`
	AnimalID  *uint                    `json:"animal_id,omitempty"` // For comments
	Animal    *models.Animal           `json:"animal,omitempty"`    // For comments
	Tags      []models.CommentTag      `json:"tags,omitempty"`      // For comments
	Metadata  *models.SessionMetadata  `json:"metadata,omitempty"`  // For session reports
}

// ActivityFeedSummary provides quick stats about concerns
type ActivityFeedSummary struct {
	BehaviorConcernsCount int `json:"behavior_concerns_count"`
	MedicalConcernsCount  int `json:"medical_concerns_count"`
	PoorSessionsCount     int `json:"poor_sessions_count"` // Sessions rated 1-2
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

		// Get pagination parameters
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

		// Get filter parameters
		filterType := c.Query("type")       // all, comments, announcements
		filterAnimal := c.Query("animal")   // animal ID
		filterTags := c.Query("tags")       // comma-separated tag names
		filterRating := c.Query("rating")   // 1-5 or "poor" (1-2)
		filterDateFrom := c.Query("from")   // ISO date
		filterDateTo := c.Query("to")       // ISO date

		// Initialize with empty slice to ensure we never return nil
		activityItems := make([]ActivityItem, 0)

		// Parse date filters
		var dateFrom, dateTo *time.Time
		if filterDateFrom != "" {
			if t, err := time.Parse(time.RFC3339, filterDateFrom); err == nil {
				dateFrom = &t
			}
		}
		if filterDateTo != "" {
			if t, err := time.Parse(time.RFC3339, filterDateTo); err == nil {
				dateTo = &t
			}
		}

		// Fetch announcements (Updates) if not filtering for comments only
		if filterType == "" || filterType == "all" || filterType == "announcements" {
			var updates []models.Update
			query := db.WithContext(ctx).Where("group_id = ?", groupID)
			
			if dateFrom != nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
			if dateTo != nil {
				query = query.Where("created_at <= ?", dateTo)
			}
			
			err := query.Preload("User").
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
			animalQuery := db.WithContext(ctx).Where("group_id = ?", groupID)
			
			// Filter by specific animal if requested
			if filterAnimal != "" {
				animalQuery = animalQuery.Where("id = ?", filterAnimal)
			}
			
			if err := animalQuery.Find(&animals).Error; err != nil {
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
				commentQuery := db.WithContext(ctx).Where("animal_id IN ?", animalIDs)
				
				// Apply date filters
				if dateFrom != nil {
					commentQuery = commentQuery.Where("created_at >= ?", dateFrom)
				}
				if dateTo != nil {
					commentQuery = commentQuery.Where("created_at <= ?", dateTo)
				}
				
				// Apply tag filter if specified
				if filterTags != "" {
					tagNames := []string{}
					for _, tag := range splitAndTrim(filterTags) {
						tagNames = append(tagNames, tag)
					}
					commentQuery = commentQuery.Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
						Joins("JOIN comment_tags ON comment_tags.id = animal_comment_tags.comment_tag_id").
						Where("comment_tags.name IN ?", tagNames).
						Group("animal_comments.id")
				}
				
				err := commentQuery.Preload("User").
					Preload("Tags").
					Order("created_at DESC").
					Find(&comments).Error

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
					return
				}

				for _, comment := range comments {
					// Apply rating filter if specified
					if filterRating != "" {
						if comment.Metadata == nil || comment.Metadata.SessionRating == 0 {
							continue // Skip if no rating
						}
						
						if filterRating == "poor" {
							if comment.Metadata.SessionRating > 2 {
								continue // Skip if not poor rating (1-2)
							}
						} else {
							if ratingVal, err := strconv.Atoi(filterRating); err == nil {
								if comment.Metadata.SessionRating != ratingVal {
									continue // Skip if rating doesn't match
								}
							}
						}
					}
					
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
						Metadata:  comment.Metadata,
					})
				}
			}
		}

		// Sort all items by creation time (newest first) - O(n log n)
		sort.Slice(activityItems, func(i, j int) bool {
			return activityItems[i].CreatedAt.After(activityItems[j].CreatedAt)
		})

		// Calculate summary statistics
		summary := ActivityFeedSummary{}
		for _, item := range activityItems {
			if item.Type == "comment" && item.Metadata != nil {
				if item.Metadata.BehaviorNotes != "" {
					summary.BehaviorConcernsCount++
				}
				if item.Metadata.MedicalNotes != "" {
					summary.MedicalConcernsCount++
				}
				if item.Metadata.SessionRating > 0 && item.Metadata.SessionRating <= 2 {
					summary.PoorSessionsCount++
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

		// Return response with pagination metadata and summary
		c.JSON(http.StatusOK, gin.H{
			"items":   paginatedItems,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": end < total,
			"summary": summary,
		})
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace
func splitAndTrim(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
