package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// ActivityItem represents a unified activity feed item
type ActivityItem struct {
	ID        uint                    `json:"id"`
	Type      string                  `json:"type"` // "comment", "announcement"
	CreatedAt time.Time               `json:"created_at"`
	UserID    uint                    `json:"user_id"`
	User      *models.User            `json:"user,omitempty"`
	Content   string                  `json:"content"`
	Title     string                  `json:"title,omitempty"` // For announcements
	ImageURL  string                  `json:"image_url,omitempty"`
	AnimalID  *uint                   `json:"animal_id,omitempty"` // For comments
	Animal    *models.Animal          `json:"animal,omitempty"`    // For comments
	Tags      []models.CommentTag     `json:"tags,omitempty"`      // For comments
	Metadata  *models.SessionMetadata `json:"metadata,omitempty"`  // For session reports
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
		db := middleware.GetDB(c, db)
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
		filterType := c.Query("type")     // all, comments, announcements
		filterAnimal := c.Query("animal") // animal ID
		filterTags := c.Query("tags")     // comma-separated tag names
		filterRating := c.Query("rating") // 1-5 or "poor" (1-2)
		filterDateFrom := c.Query("from") // ISO date
		filterDateTo := c.Query("to")     // ISO date

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

		// totalAnnouncements/totalComments are true counts, independent of
		// any bounded fetch below - len(activityItems) can no longer be used
		// for "total"/"hasMore" once the comments query is limited, since it
		// would silently undercount past the fetched window.
		var totalAnnouncements int
		summary := ActivityFeedSummary{}

		// Fetch announcements (Updates) if not filtering for comments only.
		// Tags, ratings, and the animal filter are all comment-only concepts
		// (models.Update has no tag relation, no metadata/rating, and no
		// animal association at all - announcements are group-wide), so an
		// active tag, rating, or animal filter must exclude announcements
		// entirely - otherwise every announcement in the group is returned
		// regardless of which tag/rating/animal was requested.
		if (filterType == "" || filterType == "all" || filterType == "announcements") && filterTags == "" && filterRating == "" && filterAnimal == "" {
			var updates []models.Update
			query := db.Where("group_id = ?", groupID)

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
			totalAnnouncements = len(updates)

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

		var totalComments int64

		// Fetch comments if not filtering for announcements only
		if filterType == "" || filterType == "all" || filterType == "comments" {
			// First get all animals in this group
			var animals []models.Animal
			animalQuery := db.Where("group_id = ?", groupID)

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
				commentQuery := db.Where("animal_id IN ?", animalIDs)

				// Apply date filters
				if dateFrom != nil {
					commentQuery = commentQuery.Where("created_at >= ?", dateFrom)
				}
				if dateTo != nil {
					commentQuery = commentQuery.Where("created_at <= ?", dateTo)
				}

				// Apply tag filter if specified
				if filterTags != "" {
					commentQuery = applyTagFilter(commentQuery, splitAndTrim(filterTags))
				}

				// Apply the rating filter in SQL (against the metadata jsonb
				// column) rather than after fetching every row - this has to
				// happen in SQL, not after Find(), for the Limit(offset+
				// limit) below to be safe: a post-fetch filter could still
				// drop rows out of an already-limited page, silently
				// under-filling it.
				//
				// Replicates the old post-fetch Go loop's semantics exactly,
				// including its structure: a comment needs *some* rating for
				// any non-empty filterRating value (checked unconditionally,
				// before branching - not just within the poor/exact cases),
				// then poor/exact further narrow that, and a non-numeric,
				// non-"poor" value is a deliberate no-op on top of the
				// has-a-rating gate (the old code's strconv.Atoi failure
				// branch didn't skip the comment either, so this preserves
				// that quirk rather than silently tightening it).
				//
				// SessionRating is `json:"session_rating,omitempty"`, so a
				// zero rating is never actually present as JSON 0 - the key
				// is omitted from the document entirely - meaning
				// metadata->>'session_rating' IS NOT NULL alone is
				// equivalent to "has a nonzero rating", covering both a NULL
				// metadata column and a present-but-keyless one.
				if filterRating != "" {
					commentQuery = commentQuery.Where("(animal_comments.metadata->>'session_rating') IS NOT NULL")
					if filterRating == "poor" {
						commentQuery = commentQuery.Where("(animal_comments.metadata->>'session_rating')::int BETWEEN 1 AND 2")
					} else if ratingVal, err := strconv.Atoi(filterRating); err == nil {
						commentQuery = commentQuery.Where("(animal_comments.metadata->>'session_rating')::int = ?", ratingVal)
					}
				}

				// True count of every matching comment (all filters above,
				// no Limit) - see totalComments' declaration comment.
				if err := commentQuery.Session(&gorm.Session{}).
					Model(&models.AnimalComment{}).
					Count(&totalComments).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count comments"})
					return
				}

				// Summary stats need every matching comment's metadata, not
				// just the current page - fetched separately (and leanly:
				// just the metadata column, no User/Tags preload) from the
				// bounded page query below so bounding that query for
				// performance can't also silently narrow the summary counts
				// to whatever happened to fit on this page.
				var summaryRows []models.AnimalComment
				if err := commentQuery.Session(&gorm.Session{}).
					Select("metadata").
					Find(&summaryRows).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute activity feed summary"})
					return
				}
				for _, row := range summaryRows {
					if row.Metadata == nil {
						continue
					}
					if row.Metadata.BehaviorNotes != "" {
						summary.BehaviorConcernsCount++
					}
					if row.Metadata.MedicalNotes != "" {
						summary.MedicalConcernsCount++
					}
					if row.Metadata.SessionRating > 0 && row.Metadata.SessionRating <= 2 {
						summary.PoorSessionsCount++
					}
				}

				// The actual page of comments to display. Bounded to
				// offset+limit rather than fetched in full - this is the
				// query that caused the incident this fix addresses (it
				// previously had no Limit at all, pulling every comment
				// across every animal in the group). offset+limit (not just
				// limit) is required, not a rounding-friendly shortcut: the
				// final response interleaves comments with announcements by
				// created_at and only then slices out [offset:offset+limit],
				// so anything that could land in that slice - across both
				// sources - has to be fetched first. Any comment that ends
				// up in the true top offset+limit of the *merged* feed must
				// itself be within the top offset+limit of the
				// comments-only list too (at most offset+limit-1 items,
				// comments or announcements, can be more recent than it) -
				// the standard bounded top-K merge argument. Announcements
				// are left unbounded (fetched in full, as before), which
				// trivially satisfies the same "at least top-K" requirement
				// for that side.
				var comments []models.AnimalComment
				if err := commentQuery.Session(&gorm.Session{}).
					Preload("User").
					Preload("Tags").
					Order("created_at DESC").
					Limit(offset + limit).
					Find(&comments).Error; err != nil {
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
						Metadata:  comment.Metadata,
					})
				}
			}
		}

		// Sort all items by creation time (newest first) - O(n log n)
		sort.Slice(activityItems, func(i, j int) bool {
			return activityItems[i].CreatedAt.After(activityItems[j].CreatedAt)
		})

		// Apply pagination. Slicing is bounded against len(activityItems)
		// purely to stay in-range (activityItems no longer holds every
		// matching comment, just the bounded page fetched above) - the
		// reported total/hasMore below use the true counts instead, not
		// this length.
		sliceLen := len(activityItems)
		start := offset
		if start > sliceLen {
			start = sliceLen
		}
		end := start + limit
		if end > sliceLen {
			end = sliceLen
		}

		paginatedItems := activityItems[start:end]

		// Ensure we return an empty array instead of nil
		if paginatedItems == nil {
			paginatedItems = []ActivityItem{}
		}

		// True total across both sources: announcements are fetched in full
		// above (len(updates) is already an accurate count), comments use
		// the dedicated count query above (totalComments) since the comment
		// fetch itself is now bounded.
		total := int64(totalAnnouncements) + totalComments
		hasMore := int64(offset+len(paginatedItems)) < total

		// Return response with pagination metadata and summary
		c.JSON(http.StatusOK, gin.H{
			"items":   paginatedItems,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": hasMore,
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
