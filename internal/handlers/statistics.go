package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
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

		var groups []models.Group
		if err := db.WithContext(ctx).Find(&groups).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
			return
		}

		statistics := make([]GroupStatistics, len(groups))
		for i, group := range groups {
			stats := GroupStatistics{
				GroupID: group.ID,
			}

			// Count users in group
			var userCount int64
			db.WithContext(ctx).Model(&models.User{}).
				Joins("JOIN user_groups ON user_groups.user_id = users.id").
				Where("user_groups.group_id = ?", group.ID).
				Count(&userCount)
			stats.UserCount = userCount

			// Count animals in group
			var animalCount int64
			db.WithContext(ctx).Model(&models.Animal{}).
				Where("group_id = ?", group.ID).
				Count(&animalCount)
			stats.AnimalCount = animalCount

			// Get last activity (most recent comment or announcement in this group)
			var lastCommentTime *time.Time
			var comment models.AnimalComment
			err := db.WithContext(ctx).
				Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animals.group_id = ?", group.ID).
				Order("animal_comments.created_at DESC").
				First(&comment).Error

			if err == nil {
				lastCommentTime = &comment.CreatedAt
			}

			stats.LastActivity = lastCommentTime
			statistics[i] = stats
		}

		c.JSON(http.StatusOK, statistics)
	}
}

// GetUserStatistics returns statistics for all users (admin only)
func GetUserStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var users []models.User
		if err := db.WithContext(ctx).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}

		statistics := make([]UserStatistics, len(users))
		for i, user := range users {
			stats := UserStatistics{
				UserID: user.ID,
			}

			// Count comments by user
			var commentCount int64
			db.WithContext(ctx).Model(&models.AnimalComment{}).
				Where("user_id = ?", user.ID).
				Count(&commentCount)
			stats.CommentCount = commentCount

			// Get last activity (most recent comment)
			var lastComment models.AnimalComment
			err := db.WithContext(ctx).
				Where("user_id = ?", user.ID).
				Order("created_at DESC").
				First(&lastComment).Error

			if err == nil {
				stats.LastActive = &lastComment.CreatedAt
			}

			// Count distinct animals the user has commented on
			var animalCount int64
			db.WithContext(ctx).Model(&models.AnimalComment{}).
				Where("user_id = ?", user.ID).
				Distinct("animal_id").
				Count(&animalCount)
			stats.AnimalsInteractedWith = animalCount

			statistics[i] = stats
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

		// Build query for tags
		tagsQuery := db.WithContext(ctx)
		if groupIDStr != "" {
			tagsQuery = tagsQuery.Where("group_id = ?", groupIDStr)
		}

		var tags []models.CommentTag
		if err := tagsQuery.Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
			return
		}

		statistics := make([]CommentTagStatistics, len(tags))
		for i, tag := range tags {
			stats := CommentTagStatistics{
				TagID: tag.ID,
			}

			// Build base query for statistics (filter by group if specified)
			statsBaseQuery := db.WithContext(ctx).Table("animal_comment_tags").
				Joins("JOIN animal_comments ON animal_comments.id = animal_comment_tags.animal_comment_id").
				Joins("JOIN animals ON animals.id = animal_comments.animal_id")

			if groupIDStr != "" {
				statsBaseQuery = statsBaseQuery.Where("animals.group_id = ?", groupIDStr)
			}

			// Count usage of this tag
			var usageCount int64
			statsBaseQuery.
				Where("animal_comment_tags.comment_tag_id = ?", tag.ID).
				Count(&usageCount)
			stats.UsageCount = usageCount

			// Get last time this tag was used
			var lastUsedComment struct {
				CreatedAt time.Time
			}
			lastUsedQuery := db.WithContext(ctx).Table("animal_comments").
				Select("animal_comments.created_at").
				Joins("JOIN animal_comment_tags ON animal_comment_tags.animal_comment_id = animal_comments.id").
				Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animal_comment_tags.comment_tag_id = ?", tag.ID)

			if groupIDStr != "" {
				lastUsedQuery = lastUsedQuery.Where("animals.group_id = ?", groupIDStr)
			}

			err := lastUsedQuery.
				Order("animal_comments.created_at DESC").
				First(&lastUsedComment).Error

			if err == nil {
				stats.LastUsed = &lastUsedComment.CreatedAt
			}

			// Get most tagged animal for this tag
			var mostTaggedAnimals []struct {
				AnimalID   uint
				AnimalName string
				Count      int64
			}
			mostTaggedQuery := db.WithContext(ctx).Table("animal_comment_tags").
				Select("animals.id as animal_id, animals.name as animal_name, COUNT(*) as count").
				Joins("JOIN animal_comments ON animal_comments.id = animal_comment_tags.animal_comment_id").
				Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animal_comment_tags.comment_tag_id = ?", tag.ID)

			if groupIDStr != "" {
				mostTaggedQuery = mostTaggedQuery.Where("animals.group_id = ?", groupIDStr)
			}

			err = mostTaggedQuery.
				Group("animals.id, animals.name").
				Order("count DESC").
				Limit(1).
				Find(&mostTaggedAnimals).Error

			if err == nil && len(mostTaggedAnimals) > 0 && mostTaggedAnimals[0].Count > 0 {
				stats.MostTaggedAnimalID = &mostTaggedAnimals[0].AnimalID
				stats.MostTaggedAnimalName = &mostTaggedAnimals[0].AnimalName
			}

			statistics[i] = stats
		}

		c.JSON(http.StatusOK, statistics)
	}
}
