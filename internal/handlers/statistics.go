package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"gorm.io/gorm"
)

// GroupStatistics represents statistics for a group
type GroupStatistics struct {
	GroupID      uint       `json:"group_id"`
	UserCount    int64      `json:"user_count"`
	AnimalCount  int64      `json:"animal_count"`
	LastActivity *time.Time `json:"last_activity"`
}

// UserStatistics represents statistics for a user
type UserStatistics struct {
	UserID                uint       `json:"user_id"`
	CommentCount          int64      `json:"comment_count"`
	LastActive            *time.Time `json:"last_active"`
	AnimalsInteractedWith int64      `json:"animals_interacted_with"`
}

// CommentTagStatistics represents statistics for a comment tag
type CommentTagStatistics struct {
	TagID                uint       `json:"tag_id"`
	UsageCount           int64      `json:"usage_count"`
	LastUsed             *time.Time `json:"last_used"`
	MostTaggedAnimalID   *uint      `json:"most_tagged_animal_id"`
	MostTaggedAnimalName *string    `json:"most_tagged_animal_name"`
}

// parseTimestamp attempts to parse a timestamp string using multiple common formats
// Returns nil if the string is empty or cannot be parsed
func parseTimestamp(s string) *time.Time {
	if s == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	return nil
}

// GetGroupStatistics returns statistics for all groups with pagination support (admin only)
func GetGroupStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add explicit timeout for query execution
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Get pagination parameters
		limit := 20 // Default limit
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100 // Max 100 per page
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get total count of groups
		var total int64
		if err := db.WithContext(ctx).Raw(`
			SELECT COUNT(*) FROM groups WHERE deleted_at IS NULL
		`).Scan(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count groups"})
			return
		}

		// Use a single aggregated query to fetch all group statistics at once
		// This eliminates the N+1 query problem by using subqueries
		// Note: We use string for last_activity to handle different databases (SQLite vs PostgreSQL)
		type GroupStatsRaw struct {
			GroupID      uint   `json:"group_id"`
			UserCount    int64  `json:"user_count"`
			AnimalCount  int64  `json:"animal_count"`
			LastActivity string `json:"last_activity"` // String to handle both SQLite and PostgreSQL
		}

		var rawStats []GroupStatsRaw
		start := time.Now()
		err := db.WithContext(ctx).Raw(`
			SELECT 
				g.id as group_id,
				COALESCE((SELECT COUNT(DISTINCT ug.user_id) 
					FROM user_groups ug 
					WHERE ug.group_id = g.id), 0) as user_count,
				COALESCE((SELECT COUNT(*) 
					FROM animals a 
					WHERE a.group_id = g.id AND a.deleted_at IS NULL), 0) as animal_count,
				(SELECT MAX(ac.created_at) 
					FROM animal_comments ac 
					JOIN animals a ON a.id = ac.animal_id 
					WHERE a.group_id = g.id AND ac.deleted_at IS NULL) as last_activity
			FROM groups g
			WHERE g.deleted_at IS NULL
			ORDER BY g.id
			LIMIT ? OFFSET ?
		`, limit, offset).Scan(&rawStats).Error

		duration := time.Since(start)
		if duration > 1*time.Second {
			logging.WithField("duration_ms", duration.Milliseconds()).Warn("Slow GetGroupStatistics query")
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group statistics"})
			return
		}

		// Convert raw stats to final format with proper time parsing
		statistics := make([]GroupStatistics, len(rawStats))
		for i, raw := range rawStats {
			statistics[i] = GroupStatistics{
				GroupID:     raw.GroupID,
				UserCount:   raw.UserCount,
				AnimalCount: raw.AnimalCount,
			}

			// Parse timestamp if present using helper function
			statistics[i].LastActivity = parseTimestamp(raw.LastActivity)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    statistics,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": offset+len(statistics) < int(total),
		})
	}
}

// GetUserStatistics returns statistics for all users with pagination support (admin only)
func GetUserStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add explicit timeout for query execution
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Get pagination parameters
		limit := 20 // Default limit
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100 // Max 100 per page
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get total count of users
		var total int64
		if err := db.WithContext(ctx).Raw(`
			SELECT COUNT(*) FROM users WHERE deleted_at IS NULL
		`).Scan(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
			return
		}

		// Use a single aggregated query to fetch all user statistics at once
		// This eliminates the N+1 query problem by using subqueries
		// Note: We use string for last_active to handle different databases (SQLite vs PostgreSQL)
		type UserStatsRaw struct {
			UserID                uint   `json:"user_id"`
			CommentCount          int64  `json:"comment_count"`
			LastActive            string `json:"last_active"` // String to handle both SQLite and PostgreSQL
			AnimalsInteractedWith int64  `json:"animals_interacted_with"`
		}

		var rawStats []UserStatsRaw
		start := time.Now()
		err := db.WithContext(ctx).Raw(`
			SELECT 
				u.id as user_id,
				COALESCE((SELECT COUNT(*) 
					FROM animal_comments ac 
					WHERE ac.user_id = u.id AND ac.deleted_at IS NULL), 0) as comment_count,
				(SELECT MAX(ac.created_at) 
					FROM animal_comments ac 
					WHERE ac.user_id = u.id AND ac.deleted_at IS NULL) as last_active,
				COALESCE((SELECT COUNT(DISTINCT ac.animal_id) 
					FROM animal_comments ac 
					WHERE ac.user_id = u.id AND ac.deleted_at IS NULL), 0) as animals_interacted_with
			FROM users u
			WHERE u.deleted_at IS NULL
			ORDER BY u.id
			LIMIT ? OFFSET ?
		`, limit, offset).Scan(&rawStats).Error

		duration := time.Since(start)
		if duration > 1*time.Second {
			logging.WithField("duration_ms", duration.Milliseconds()).Warn("Slow GetUserStatistics query")
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user statistics"})
			return
		}

		// Convert raw stats to final format with proper time parsing
		statistics := make([]UserStatistics, len(rawStats))
		for i, raw := range rawStats {
			statistics[i] = UserStatistics{
				UserID:                raw.UserID,
				CommentCount:          raw.CommentCount,
				AnimalsInteractedWith: raw.AnimalsInteractedWith,
			}

			// Parse timestamp if present using helper function
			statistics[i].LastActive = parseTimestamp(raw.LastActive)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    statistics,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": offset+len(statistics) < int(total),
		})
	}
}

// GetCommentTagStatistics returns statistics for comment tags with pagination support
// Accepts optional group_id query parameter to filter by group
func GetCommentTagStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add explicit timeout for query execution
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Get optional group_id filter
		groupIDStr := c.Query("group_id")

		// Validate group_id parameter if provided
		if groupIDStr != "" {
			if _, err := strconv.ParseUint(groupIDStr, 10, 32); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group_id parameter"})
				return
			}
		}

		// Get pagination parameters
		limit := 20 // Default limit
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 100 {
					limit = 100 // Max 100 per page
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get total count of tags
		var total int64
		countQuery := `SELECT COUNT(*) FROM comment_tags WHERE deleted_at IS NULL`
		if groupIDStr != "" {
			countQuery += ` AND group_id = ?`
			if err := db.WithContext(ctx).Raw(countQuery, groupIDStr).Scan(&total).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count tags"})
				return
			}
		} else {
			if err := db.WithContext(ctx).Raw(countQuery).Scan(&total).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count tags"})
				return
			}
		}

		// Use a single aggregated query with window functions to get all tag statistics
		// This eliminates N+1 queries by computing all stats in a single query
		// Note: We use string for last_used to handle different databases (SQLite vs PostgreSQL)
		type TagStatsRaw struct {
			TagID                uint   `json:"tag_id"`
			UsageCount           int64  `json:"usage_count"`
			LastUsed             string `json:"last_used"` // String to handle both SQLite and PostgreSQL
			MostTaggedAnimalID   *uint  `json:"most_tagged_animal_id"`
			MostTaggedAnimalName string `json:"most_tagged_animal_name"`
		}

		query := `
			WITH tag_usage AS (
				SELECT 
					ct.id as tag_id,
					COUNT(act.animal_comment_id) as usage_count,
					MAX(ac.created_at) as last_used,
					a.id as animal_id,
					a.name as animal_name,
					ROW_NUMBER() OVER (PARTITION BY ct.id ORDER BY COUNT(act.animal_comment_id) DESC) as rn
				FROM comment_tags ct
				LEFT JOIN animal_comment_tags act ON act.comment_tag_id = ct.id
				LEFT JOIN animal_comments ac ON ac.id = act.animal_comment_id AND ac.deleted_at IS NULL
				LEFT JOIN animals a ON a.id = ac.animal_id AND a.deleted_at IS NULL
				WHERE ct.deleted_at IS NULL
		`

		// Add group filter if specified
		if groupIDStr != "" {
			query += " AND (a.group_id = ? OR a.group_id IS NULL)"
		}

		query += `
				GROUP BY ct.id, a.id, a.name
			)
			SELECT 
				tag_id,
				SUM(usage_count) as usage_count,
				MAX(last_used) as last_used,
				MAX(CASE WHEN rn = 1 THEN animal_id END) as most_tagged_animal_id,
				MAX(CASE WHEN rn = 1 THEN animal_name END) as most_tagged_animal_name
			FROM tag_usage
			GROUP BY tag_id
			ORDER BY tag_id
			LIMIT ? OFFSET ?
		`

		var rawStats []TagStatsRaw
		var err error
		start := time.Now()
		if groupIDStr != "" {
			err = db.WithContext(ctx).Raw(query, groupIDStr, limit, offset).Scan(&rawStats).Error
		} else {
			err = db.WithContext(ctx).Raw(query, limit, offset).Scan(&rawStats).Error
		}

		duration := time.Since(start)
		if duration > 1*time.Second {
			logging.WithField("duration_ms", duration.Milliseconds()).Warn("Slow GetCommentTagStatistics query")
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tag statistics"})
			return
		}

		// Convert raw stats to final format with proper time parsing
		statistics := make([]CommentTagStatistics, len(rawStats))
		for i, raw := range rawStats {
			statistics[i] = CommentTagStatistics{
				TagID:              raw.TagID,
				UsageCount:         raw.UsageCount,
				MostTaggedAnimalID: raw.MostTaggedAnimalID,
			}

			// Set most tagged animal name if present
			if raw.MostTaggedAnimalName != "" {
				statistics[i].MostTaggedAnimalName = &raw.MostTaggedAnimalName
			}

			// Parse timestamp if present using helper function
			statistics[i].LastUsed = parseTimestamp(raw.LastUsed)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    statistics,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": offset+len(statistics) < int(total),
		})
	}
}
