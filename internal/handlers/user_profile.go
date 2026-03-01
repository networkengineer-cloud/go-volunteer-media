package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// SkillTagEntry represents a skill tag assigned to a user with group context
type SkillTagEntry struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

// UserProfileResponse represents user profile information with activity data
type UserProfileResponse struct {
	ID                    uint                       `json:"id"`
	Username              string                     `json:"username"`
	FirstName             string                     `json:"first_name"`
	LastName              string                     `json:"last_name"`
	Email                 string                     `json:"email"`
	PhoneNumber           string                     `json:"phone_number"`
	IsAdmin               bool                       `json:"is_admin"`
	CreatedAt             string                     `json:"created_at"`
	DefaultGroupID        *uint                      `json:"default_group_id"`
	Groups                []models.Group             `json:"groups"`
	Statistics            UserProfileStatistics      `json:"statistics"`
	RecentComments        []UserCommentActivity      `json:"recent_comments"`
	RecentAnnouncements   []UserAnnouncementActivity `json:"recent_announcements"`
	AnimalsInteractedWith []AnimalInteraction        `json:"animals_interacted_with"`
	SkillTags             []SkillTagEntry            `json:"skill_tags"`
}

// UserProfileStatistics represents statistics for a user profile
type UserProfileStatistics struct {
	TotalComments      int64              `json:"total_comments"`
	TotalAnnouncements int64              `json:"total_announcements"`
	AnimalsInteracted  int64              `json:"animals_interacted"`
	MostActiveGroup    *GroupActivityInfo `json:"most_active_group"`
	LastActiveDate     *string            `json:"last_active_date"`
}

// GroupActivityInfo represents activity information for a group
type GroupActivityInfo struct {
	GroupID      uint   `json:"group_id"`
	GroupName    string `json:"group_name"`
	CommentCount int64  `json:"comment_count"`
}

// UserCommentActivity represents a user's comment activity
type UserCommentActivity struct {
	ID         uint   `json:"id"`
	AnimalID   uint   `json:"animal_id"`
	AnimalName string `json:"animal_name"`
	GroupID    uint   `json:"group_id"`
	GroupName  string `json:"group_name"`
	Content    string `json:"content"`
	ImageURL   string `json:"image_url"`
	CreatedAt  string `json:"created_at"`
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

// fetchSkillTagsForUser returns all skill tags assigned to targetUserID, optionally
// restricted to groups where currentUserID is a group admin.
func fetchSkillTagsForUser(db *gorm.DB, targetUserID uint, restrictToGroupAdminOf uint) []SkillTagEntry {
	type row struct {
		GroupID   uint
		GroupName string
		TagID     uint
		Name      string
		Color     string
	}
	var rows []row

	q := db.Table("user_skill_tag_assignments a").
		Select("g.id as group_id, g.name as group_name, t.id as tag_id, t.name, t.color").
		Joins("JOIN user_skill_tags t ON t.id = a.user_skill_tag_id AND t.deleted_at IS NULL").
		Joins("JOIN groups g ON g.id = t.group_id AND g.deleted_at IS NULL").
		Where("a.user_id = ?", targetUserID)

	if restrictToGroupAdminOf != 0 {
		// Restrict to groups where the caller is a group admin
		q = q.Joins("JOIN user_groups ug ON ug.group_id = t.group_id AND ug.user_id = ? AND ug.is_group_admin = ?", restrictToGroupAdminOf, true)
	}

	if err := q.Scan(&rows).Error; err != nil {
		return []SkillTagEntry{}
	}

	entries := make([]SkillTagEntry, len(rows))
	for i, r := range rows {
		entries[i] = SkillTagEntry{
			GroupID:   r.GroupID,
			GroupName: r.GroupName,
			ID:        r.TagID,
			Name:      r.Name,
			Color:     r.Color,
		}
	}
	return entries
}

// GetUserProfile returns detailed profile information for a user
// - Users can view their own profile with full details
// - Site admins can view any profile with full details
// - Group admins can view extended profile info for members in their groups
// - Regular group members can view limited profile info (username only) for members in their shared groups
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

		// Determine access level
		currentUserIDUint, _ := currentUserID.(uint)
		isOwnProfile := currentUserIDUint == uint(targetUserID)
		isSiteAdmin, _ := isAdmin.(bool)

		// Check if current user is a group admin for any shared group with target user
		var isGroupAdminForSharedGroup bool
		if !isSiteAdmin && !isOwnProfile {
			// Check if current user is admin of any shared group
			type SharedGroupInfo struct {
				GroupID      uint
				IsGroupAdmin bool
			}
			var sharedGroups []SharedGroupInfo
			err := db.WithContext(ctx).Raw(`
				SELECT ug1.group_id, ug1.is_group_admin
				FROM user_groups ug1
				JOIN user_groups ug2 ON ug1.group_id = ug2.group_id
				WHERE ug1.user_id = ? AND ug2.user_id = ?
			`, currentUserID, targetUserID).Scan(&sharedGroups).Error

			if err == nil && len(sharedGroups) > 0 {
				for _, sg := range sharedGroups {
					if sg.IsGroupAdmin {
						isGroupAdminForSharedGroup = true
						break
					}
				}
			}
		}

		// Fetch user details
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups", activeGroupsPreload).First(&user, targetUserID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		// For regular users viewing another user's profile - return info respecting privacy settings
		if !isOwnProfile && !isSiteAdmin && !isGroupAdminForSharedGroup {
			// Return profile info respecting privacy settings
			type RegularUserProfileResponse struct {
				ID          uint           `json:"id"`
				Username    string         `json:"username"`
				FirstName   string         `json:"first_name,omitempty"`
				LastName    string         `json:"last_name,omitempty"`
				Email       string         `json:"email,omitempty"`
				PhoneNumber string         `json:"phone_number,omitempty"`
				CreatedAt   string         `json:"created_at"`
				Groups      []models.Group `json:"groups"`
			}
			response := RegularUserProfileResponse{
				ID:        user.ID,
				Username:  user.Username,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Groups:    user.Groups,
			}
			// Only include email if user hasn't hidden it
			if !user.HideEmail {
				response.Email = user.Email
			}
			// Only include phone if user hasn't hidden it
			if !user.HidePhoneNumber {
				response.PhoneNumber = user.PhoneNumber
			}
			c.JSON(http.StatusOK, response)
			return
		}

		// For group admins viewing members - return extended info respecting privacy settings
		if !isOwnProfile && !isSiteAdmin && isGroupAdminForSharedGroup {
			// Return extended profile for group admins
			// Group admins can see all members but still respect privacy settings
			// Actually, based on PERMISSIONS.md: "Group admins can always see contact info"
			// So group admins bypass privacy settings for their group members
			type GroupAdminProfileResponse struct {
				ID          uint            `json:"id"`
				Username    string          `json:"username"`
				FirstName   string          `json:"first_name"`
				LastName    string          `json:"last_name"`
				Email       string          `json:"email"`
				PhoneNumber string          `json:"phone_number"`
				CreatedAt   string          `json:"created_at"`
				Groups      []models.Group  `json:"groups"`
				SkillTags   []SkillTagEntry `json:"skill_tags"`
			}
			c.JSON(http.StatusOK, GroupAdminProfileResponse{
				ID:          user.ID,
				Username:    user.Username,
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				Email:       user.Email,
				PhoneNumber: user.PhoneNumber,
				CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Groups:      user.Groups,
				SkillTags:   fetchSkillTagsForUser(db, user.ID, currentUserIDUint),
			})
			return
		} // Build full profile response for own profile or admin viewing
		profile := UserProfileResponse{
			ID:             user.ID,
			Username:       user.Username,
			FirstName:      user.FirstName,
			LastName:       user.LastName,
			Email:          user.Email,
			PhoneNumber:    user.PhoneNumber,
			IsAdmin:        user.IsAdmin,
			CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			DefaultGroupID: user.DefaultGroupID,
			Groups:         user.Groups,
			SkillTags:      fetchSkillTagsForUser(db, user.ID, 0),
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
			CommentID  uint
			AnimalID   uint
			AnimalName string
			GroupID    uint
			GroupName  string
			Content    string
			ImageURL   string
			CreatedAt  string
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
