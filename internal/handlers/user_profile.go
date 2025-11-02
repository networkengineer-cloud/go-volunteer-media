package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// UserProfileResponse represents user profile information with activity data
type UserProfileResponse struct {
	ID                        uint                     `json:"id"`
	Username                  string                   `json:"username"`
	Email                     string                   `json:"email"`
	IsAdmin                   bool                     `json:"is_admin"`
	CreatedAt                 string                   `json:"created_at"`
	DefaultGroupID            *uint                    `json:"default_group_id"`
	Groups                    []models.Group           `json:"groups"`
	Statistics                UserProfileStatistics    `json:"statistics"`
	RecentComments            []UserCommentActivity    `json:"recent_comments"`
	RecentAnnouncements       []UserAnnouncementActivity `json:"recent_announcements"`
	AnimalsInteractedWith     []AnimalInteraction      `json:"animals_interacted_with"`
}

// UserProfileStatistics represents statistics for a user profile
type UserProfileStatistics struct {
	TotalComments         int64  `json:"total_comments"`
	TotalAnnouncements    int64  `json:"total_announcements"`
	AnimalsInteracted     int64  `json:"animals_interacted"`
	MostActiveGroup       *GroupActivityInfo `json:"most_active_group"`
	LastActiveDate        *string `json:"last_active_date"`
}

// GroupActivityInfo represents activity information for a group
type GroupActivityInfo struct {
	GroupID      uint   `json:"group_id"`
	GroupName    string `json:"group_name"`
	CommentCount int64  `json:"comment_count"`
}

// UserCommentActivity represents a user's comment activity
type UserCommentActivity struct {
	ID          uint   `json:"id"`
	AnimalID    uint   `json:"animal_id"`
	AnimalName  string `json:"animal_name"`
	GroupID     uint   `json:"group_id"`
	GroupName   string `json:"group_name"`
	Content     string `json:"content"`
	ImageURL    string `json:"image_url"`
	CreatedAt   string `json:"created_at"`
}

// UserAnnouncementActivity represents a user's announcement activity
type UserAnnouncementActivity struct {
	ID        uint   `json:"id"`
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// AnimalInteraction represents an animal a user has interacted with
type AnimalInteraction struct {
	AnimalID      uint   `json:"animal_id"`
	AnimalName    string `json:"animal_name"`
	GroupID       uint   `json:"group_id"`
	GroupName     string `json:"group_name"`
	ImageURL      string `json:"image_url"`
	CommentCount  int64  `json:"comment_count"`
	LastCommentAt string `json:"last_comment_at"`
}

// GetUserProfile returns detailed profile information for a user
// Accessible by the user themselves or admins
func GetUserProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Get user ID from URL parameter
		userIDParam := c.Param("id")
		targetUserID, err := strconv.ParseUint(userIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Get current user's ID and admin status
		currentUserID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check authorization: users can only view their own profile unless they're admin
		if !isAdmin.(bool) && currentUserID.(uint) != uint(targetUserID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own profile"})
			return
		}

		// Fetch user details
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").First(&user, targetUserID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		// Build profile response
		profile := UserProfileResponse{
			ID:             user.ID,
			Username:       user.Username,
			Email:          user.Email,
			IsAdmin:        user.IsAdmin,
			CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			DefaultGroupID: user.DefaultGroupID,
			Groups:         user.Groups,
		}

		// Calculate statistics
		var stats UserProfileStatistics

		// Total comments
		var commentCount int64
		db.WithContext(ctx).Model(&models.AnimalComment{}).
			Where("user_id = ?", targetUserID).
			Count(&commentCount)
		stats.TotalComments = commentCount

		// Total announcements
		var announcementCount int64
		db.WithContext(ctx).Model(&models.Announcement{}).
			Where("user_id = ?", targetUserID).
			Count(&announcementCount)
		stats.TotalAnnouncements = announcementCount

		// Animals interacted with
		var animalCount int64
		db.WithContext(ctx).Model(&models.AnimalComment{}).
			Select("DISTINCT animal_id").
			Where("user_id = ?", targetUserID).
			Count(&animalCount)
		stats.AnimalsInteracted = animalCount

		// Last active date (most recent comment or announcement)
		var lastCommentDate *string
		var lastComment models.AnimalComment
		if err := db.WithContext(ctx).Where("user_id = ?", targetUserID).
			Order("created_at DESC").First(&lastComment).Error; err == nil {
			dateStr := lastComment.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
			lastCommentDate = &dateStr
		}
		stats.LastActiveDate = lastCommentDate

		// Most active group (group with most comments)
		type GroupActivity struct {
			GroupID      uint
			GroupName    string
			CommentCount int64
		}
		var groupActivity GroupActivity
		err = db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Select("groups.id as group_id, groups.name as group_name, COUNT(*) as comment_count").
			Joins("JOIN animals ON animals.id = animal_comments.animal_id").
			Joins("JOIN groups ON groups.id = animals.group_id").
			Where("animal_comments.user_id = ?", targetUserID).
			Group("groups.id, groups.name").
			Order("comment_count DESC").
			First(&groupActivity).Error

		if err == nil {
			stats.MostActiveGroup = &GroupActivityInfo{
				GroupID:      groupActivity.GroupID,
				GroupName:    groupActivity.GroupName,
				CommentCount: groupActivity.CommentCount,
			}
		}

		profile.Statistics = stats

		// Recent comments (last 10)
		type CommentWithDetails struct {
			CommentID   uint
			AnimalID    uint
			AnimalName  string
			GroupID     uint
			GroupName   string
			Content     string
			ImageURL    string
			CreatedAt   string
		}

		var commentDetails []CommentWithDetails
		db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Select(`
				animal_comments.id as comment_id,
				animals.id as animal_id,
				animals.name as animal_name,
				groups.id as group_id,
				groups.name as group_name,
				animal_comments.content,
				animal_comments.image_url,
				animal_comments.created_at
			`).
			Joins("JOIN animals ON animals.id = animal_comments.animal_id").
			Joins("JOIN groups ON groups.id = animals.group_id").
			Where("animal_comments.user_id = ?", targetUserID).
			Order("animal_comments.created_at DESC").
			Limit(10).
			Scan(&commentDetails)

		recentComments := make([]UserCommentActivity, len(commentDetails))
		for i, cd := range commentDetails {
			recentComments[i] = UserCommentActivity{
				ID:         cd.CommentID,
				AnimalID:   cd.AnimalID,
				AnimalName: cd.AnimalName,
				GroupID:    cd.GroupID,
				GroupName:  cd.GroupName,
				Content:    cd.Content,
				ImageURL:   cd.ImageURL,
				CreatedAt:  cd.CreatedAt,
			}
		}
		profile.RecentComments = recentComments

		// Recent announcements (last 10, if admin)
		// Note: Announcements are site-wide and don't belong to specific groups
		if user.IsAdmin {
			var announcements []models.Announcement
			db.WithContext(ctx).
				Where("user_id = ?", targetUserID).
				Order("created_at DESC").
				Limit(10).
				Find(&announcements)

			recentAnnouncements := make([]UserAnnouncementActivity, len(announcements))
			for i, announcement := range announcements {
				recentAnnouncements[i] = UserAnnouncementActivity{
					ID:        announcement.ID,
					GroupID:   0, // Announcements are site-wide
					GroupName: "Site-wide",
					Content:   announcement.Content,
					CreatedAt: announcement.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			profile.RecentAnnouncements = recentAnnouncements
		} else {
			profile.RecentAnnouncements = []UserAnnouncementActivity{}
		}

		// Animals interacted with (showing comment count per animal)
		type AnimalInteractionQuery struct {
			AnimalID      uint
			AnimalName    string
			GroupID       uint
			GroupName     string
			ImageURL      string
			CommentCount  int64
			LastCommentAt string
		}

		var animalInteractions []AnimalInteractionQuery
		db.WithContext(ctx).
			Model(&models.AnimalComment{}).
			Select(`
				animals.id as animal_id,
				animals.name as animal_name,
				groups.id as group_id,
				groups.name as group_name,
				animals.image_url,
				COUNT(*) as comment_count,
				MAX(animal_comments.created_at) as last_comment_at
			`).
			Joins("JOIN animals ON animals.id = animal_comments.animal_id").
			Joins("JOIN groups ON groups.id = animals.group_id").
			Where("animal_comments.user_id = ?", targetUserID).
			Group("animals.id, animals.name, groups.id, groups.name, animals.image_url").
			Order("comment_count DESC, last_comment_at DESC").
			Limit(20).
			Scan(&animalInteractions)

		animalsInteracted := make([]AnimalInteraction, len(animalInteractions))
		for i, ai := range animalInteractions {
			animalsInteracted[i] = AnimalInteraction{
				AnimalID:      ai.AnimalID,
				AnimalName:    ai.AnimalName,
				GroupID:       ai.GroupID,
				GroupName:     ai.GroupName,
				ImageURL:      ai.ImageURL,
				CommentCount:  ai.CommentCount,
				LastCommentAt: ai.LastCommentAt,
			}
		}
		profile.AnimalsInteractedWith = animalsInteracted

		c.JSON(http.StatusOK, profile)
	}
}
