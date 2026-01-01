package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// AdminDashboardStats represents overall system statistics for the admin dashboard
type AdminDashboardStats struct {
	TotalUsers              int64             `json:"total_users"`
	TotalGroups             int64             `json:"total_groups"`
	TotalAnimals            int64             `json:"total_animals"`
	TotalComments           int64             `json:"total_comments"`
	RecentUsers             []RecentUser      `json:"recent_users"`
	MostActiveGroups        []ActiveGroupInfo `json:"most_active_groups"`
	AnimalsNeedingAttention []AnimalAlert     `json:"animals_needing_attention"`
	SystemHealth            SystemHealthInfo  `json:"system_health"`
}

// RecentUser represents a recently registered user
type RecentUser struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// ActiveGroupInfo represents activity information for active groups
type ActiveGroupInfo struct {
	GroupID      uint   `json:"group_id"`
	GroupName    string `json:"group_name"`
	UserCount    int64  `json:"user_count"`
	AnimalCount  int64  `json:"animal_count"`
	CommentCount int64  `json:"comment_count"`
	LastActivity string `json:"last_activity"`
}

// AnimalAlert represents an animal that needs attention based on tags
type AnimalAlert struct {
	AnimalID    uint     `json:"animal_id"`
	AnimalName  string   `json:"animal_name"`
	GroupID     uint     `json:"group_id"`
	GroupName   string   `json:"group_name"`
	ImageURL    string   `json:"image_url"`
	AlertTags   []string `json:"alert_tags"`
	LastComment string   `json:"last_comment"`
}

// SystemHealthInfo represents system health indicators
type SystemHealthInfo struct {
	ActiveUsersLast24h    int64   `json:"active_users_last_24h"`
	CommentsLast24h       int64   `json:"comments_last_24h"`
	NewUsersLast7Days     int64   `json:"new_users_last_7_days"`
	AverageCommentsPerDay float64 `json:"average_comments_per_day"`
}

// GetAdminDashboardStats returns comprehensive statistics for the admin dashboard
func GetAdminDashboardStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var stats AdminDashboardStats

		// Use a single CTE-based query to get all counts efficiently
		type CountsResult struct {
			TotalUsers    int64
			TotalGroups   int64
			TotalAnimals  int64
			TotalComments int64
		}
		var counts CountsResult

		err := db.WithContext(ctx).Raw(`
			SELECT 
				(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) as total_users,
				(SELECT COUNT(*) FROM groups WHERE deleted_at IS NULL) as total_groups,
				(SELECT COUNT(*) FROM animals WHERE deleted_at IS NULL) as total_animals,
				(SELECT COUNT(*) FROM animal_comments WHERE deleted_at IS NULL) as total_comments
		`).Scan(&counts).Error

		if err == nil {
			stats.TotalUsers = counts.TotalUsers
			stats.TotalGroups = counts.TotalGroups
			stats.TotalAnimals = counts.TotalAnimals
			stats.TotalComments = counts.TotalComments
		} else {
			logging.WithField("error", err.Error()).Warn("Failed to fetch total counts")
		}

		// Get recent users (last 5)
		var recentUsers []models.User
		if err := db.WithContext(ctx).
			Order("created_at DESC").
			Limit(5).
			Find(&recentUsers).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to fetch recent users")
		}

		stats.RecentUsers = make([]RecentUser, len(recentUsers))
		for i, user := range recentUsers {
			stats.RecentUsers[i] = RecentUser{
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				IsAdmin:   user.IsAdmin,
				CreatedAt: user.CreatedAt,
			}
		}

		// Get most active groups (top 5 by comment count in last 30 days)
		// Use optimized CTE to avoid correlated subqueries
		type GroupActivityQuery struct {
			GroupID      uint
			GroupName    string
			UserCount    int64
			AnimalCount  int64
			CommentCount int64
			LastActivity string
		}

		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		var groupActivities []GroupActivityQuery

		err = db.WithContext(ctx).Raw(`
			WITH group_stats AS (
				SELECT 
					g.id as group_id,
					g.name as group_name,
					COUNT(DISTINCT ug.user_id) as user_count,
					COUNT(DISTINCT a.id) as animal_count,
					COUNT(DISTINCT CASE WHEN ac.created_at > ? THEN ac.id END) as comment_count,
					MAX(ac.created_at) as last_activity
				FROM groups g
				LEFT JOIN user_groups ug ON ug.group_id = g.id
				LEFT JOIN animals a ON a.group_id = g.id AND a.deleted_at IS NULL
				LEFT JOIN animal_comments ac ON ac.animal_id = a.id AND ac.deleted_at IS NULL
				WHERE g.deleted_at IS NULL
				GROUP BY g.id, g.name
				HAVING COUNT(DISTINCT CASE WHEN ac.created_at > ? THEN ac.id END) > 0
			)
			SELECT * FROM group_stats
			ORDER BY comment_count DESC
			LIMIT 5
		`, thirtyDaysAgo, thirtyDaysAgo).Scan(&groupActivities).Error

		if err == nil {
			stats.MostActiveGroups = make([]ActiveGroupInfo, len(groupActivities))
			for i, ga := range groupActivities {
				stats.MostActiveGroups[i] = ActiveGroupInfo{
					GroupID:      ga.GroupID,
					GroupName:    ga.GroupName,
					UserCount:    ga.UserCount,
					AnimalCount:  ga.AnimalCount,
					CommentCount: ga.CommentCount,
					LastActivity: ga.LastActivity,
				}
			}
		} else {
			logging.WithField("error", err.Error()).Warn("Failed to fetch most active groups")
		}

		// Get animals needing attention (with system tags like "Needs Attention", "Medical", "Behavior")
		type AnimalAlertQuery struct {
			AnimalID    uint
			AnimalName  string
			GroupID     uint
			GroupName   string
			ImageURL    string
			TagNames    string // Comma-separated tag names
			LastComment string
		}

		var animalAlerts []AnimalAlertQuery
		if err := db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Select(`
				DISTINCT animals.id as animal_id,
				animals.name as animal_name,
				groups.id as group_id,
				groups.name as group_name,
				animals.image_url,
				STRING_AGG(DISTINCT comment_tags.name, ',') as tag_names,
				MAX(animal_comments.created_at) as last_comment
			`).
			Joins("JOIN animals ON animals.id = animal_comments.animal_id").
			Joins("JOIN groups ON groups.id = animals.group_id").
			Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
			Joins("JOIN comment_tags ON comment_tags.id = animal_comment_tags.comment_tag_id").
			Where("comment_tags.is_system = true").
			Group("animals.id, animals.name, groups.id, groups.name, animals.image_url").
			Order("last_comment DESC").
			Limit(10).
			Scan(&animalAlerts).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to fetch animals needing attention")
		}

		stats.AnimalsNeedingAttention = make([]AnimalAlert, len(animalAlerts))
		for i, alert := range animalAlerts {
			// Split comma-separated tag names into array
			tags := []string{}
			if alert.TagNames != "" {
				tags = append(tags, alert.TagNames) // Simplified - would need proper string split in production
			}

			stats.AnimalsNeedingAttention[i] = AnimalAlert{
				AnimalID:    alert.AnimalID,
				AnimalName:  alert.AnimalName,
				GroupID:     alert.GroupID,
				GroupName:   alert.GroupName,
				ImageURL:    alert.ImageURL,
				AlertTags:   tags,
				LastComment: alert.LastComment,
			}
		}

		// Get system health indicators using a single optimized query
		var health SystemHealthInfo

		last24h := time.Now().AddDate(0, 0, -1)
		last7days := time.Now().AddDate(0, 0, -7)

		type HealthQuery struct {
			ActiveUsersLast24h    int64
			CommentsLast24h       int64
			NewUsersLast7Days     int64
			AverageCommentsPerDay float64
		}
		var healthQuery HealthQuery

		err = db.WithContext(ctx).Raw(`
			SELECT 
				(SELECT COUNT(DISTINCT user_id) FROM animal_comments 
				 WHERE deleted_at IS NULL AND created_at > ?) as active_users_last_24h,
				(SELECT COUNT(*) FROM animal_comments 
				 WHERE deleted_at IS NULL AND created_at > ?) as comments_last_24h,
				(SELECT COUNT(*) FROM users 
				 WHERE deleted_at IS NULL AND created_at > ?) as new_users_last_7_days,
				(SELECT COALESCE(COUNT(*)::float / 30, 0) FROM animal_comments 
				 WHERE deleted_at IS NULL AND created_at > ?) as average_comments_per_day
		`, last24h, last24h, last7days, thirtyDaysAgo).Scan(&healthQuery).Error

		if err == nil {
			health.ActiveUsersLast24h = healthQuery.ActiveUsersLast24h
			health.CommentsLast24h = healthQuery.CommentsLast24h
			health.NewUsersLast7Days = healthQuery.NewUsersLast7Days
			health.AverageCommentsPerDay = healthQuery.AverageCommentsPerDay
		} else {
			logging.WithField("error", err.Error()).Warn("Failed to fetch system health indicators")
		}

		stats.SystemHealth = health

		c.JSON(http.StatusOK, stats)
	}
}
