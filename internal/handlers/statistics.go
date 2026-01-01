package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

// GetGroupStatistics returns statistics for all groups (admin only)
func GetGroupStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

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
		`).Scan(&rawStats).Error

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

			// Parse timestamp if present
			if raw.LastActivity != "" {
				// Try parsing different timestamp formats (SQLite and PostgreSQL)
				formats := []string{
					time.RFC3339,
					"2006-01-02 15:04:05.999999999 -07:00",
					"2006-01-02 15:04:05",
					"2006-01-02T15:04:05.999999999Z",
				}
				for _, format := range formats {
					if t, err := time.Parse(format, raw.LastActivity); err == nil {
						statistics[i].LastActivity = &t
						break
					}
				}
			}
		}

		c.JSON(http.StatusOK, statistics)
	}
}

// GetUserStatistics returns statistics for all users (admin only)
func GetUserStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

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
		`).Scan(&rawStats).Error

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

			// Parse timestamp if present
			if raw.LastActive != "" {
				// Try parsing different timestamp formats (SQLite and PostgreSQL)
				formats := []string{
					time.RFC3339,
					"2006-01-02 15:04:05.999999999 -07:00",
					"2006-01-02 15:04:05",
					"2006-01-02T15:04:05.999999999Z",
				}
				for _, format := range formats {
					if t, err := time.Parse(format, raw.LastActive); err == nil {
						statistics[i].LastActive = &t
						break
					}
				}
			}
		}

		c.JSON(http.StatusOK, statistics)
	}
}

// GetCommentTagStatistics returns statistics for comment tags
// Accepts optional group_id query parameter to filter by group
func GetCommentTagStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get optional group_id filter
		groupIDStr := c.Query("group_id")

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
		`

		var rawStats []TagStatsRaw
		var err error
		if groupIDStr != "" {
			err = db.WithContext(ctx).Raw(query, groupIDStr).Scan(&rawStats).Error
		} else {
			err = db.WithContext(ctx).Raw(query).Scan(&rawStats).Error
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

			// Parse timestamp if present
			if raw.LastUsed != "" {
				// Try parsing different timestamp formats (SQLite and PostgreSQL)
				formats := []string{
					time.RFC3339,
					"2006-01-02 15:04:05.999999999 -07:00",
					"2006-01-02 15:04:05",
					"2006-01-02T15:04:05.999999999Z",
				}
				for _, format := range formats {
					if t, err := time.Parse(format, raw.LastUsed); err == nil {
						statistics[i].LastUsed = &t
						break
					}
				}
			}
		}

		c.JSON(http.StatusOK, statistics)
	}
}
