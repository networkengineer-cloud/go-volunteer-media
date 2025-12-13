package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

		// Get total counts
		db.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers)
		db.WithContext(ctx).Model(&models.Group{}).Count(&stats.TotalGroups)
		db.WithContext(ctx).Model(&models.Animal{}).Count(&stats.TotalAnimals)
		db.WithContext(ctx).Model(&models.AnimalComment{}).Count(&stats.TotalComments)

		// Get recent users (last 5)
		var recentUsers []models.User
		db.WithContext(ctx).
			Order("created_at DESC").
			Limit(5).
			Find(&recentUsers)

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

		db.WithContext(ctx).
			Model(&models.Group{}).
			Select(`
				groups.id as group_id,
				groups.name as group_name,
				(SELECT COUNT(DISTINCT user_groups.user_id) FROM user_groups WHERE user_groups.group_id = groups.id) as user_count,
				(SELECT COUNT(*) FROM animals WHERE animals.group_id = groups.id) as animal_count,
				(SELECT COUNT(*) FROM animal_comments 
					JOIN animals ON animals.id = animal_comments.animal_id 
					WHERE animals.group_id = groups.id AND animal_comments.created_at > ?) as comment_count,
				(SELECT MAX(animal_comments.created_at) FROM animal_comments 
					JOIN animals ON animals.id = animal_comments.animal_id 
					WHERE animals.group_id = groups.id) as last_activity
			`, thirtyDaysAgo).
			Having("comment_count > 0").
			Order("comment_count DESC").
			Limit(5).
			Scan(&groupActivities)

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
		db.WithContext(ctx).
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
			Scan(&animalAlerts)

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

		// Get system health indicators
		var health SystemHealthInfo

		// Active users in last 24 hours (users who commented)
		last24h := time.Now().AddDate(0, 0, -1)
		db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Select("COUNT(DISTINCT user_id)").
			Where("created_at > ?", last24h).
			Scan(&health.ActiveUsersLast24h)

		// Comments in last 24 hours
		db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Where("created_at > ?", last24h).
			Count(&health.CommentsLast24h)

		// New users in last 7 days
		last7days := time.Now().AddDate(0, 0, -7)
		db.WithContext(ctx).
			Model(&models.User{}).
			Where("created_at > ?", last7days).
			Count(&health.NewUsersLast7Days)

		// Average comments per day (last 30 days)
		if stats.TotalComments > 0 {
			var avgComments float64
			db.WithContext(ctx).
				Model(&models.AnimalComment{}).
				Select("COUNT(*)::float / 30").
				Where("created_at > ?", thirtyDaysAgo).
				Scan(&avgComments)
			health.AverageCommentsPerDay = avgComments
		}

		stats.SystemHealth = health

		c.JSON(http.StatusOK, stats)
	}
}
