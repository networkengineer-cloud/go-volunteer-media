package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// UserSkillTagRequest is the request body for creating or updating a user skill tag
type UserSkillTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=50"`
	Color string `json:"color" binding:"required"`
}

// GetUserSkillTags returns all skill tags defined for a group
// Route: GET /api/groups/:id/user-skill-tags
func GetUserSkillTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var tags []models.UserSkillTag
		if err := db.Where("group_id = ?", groupID).Order("name").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill tags"})
			return
		}

		c.JSON(http.StatusOK, tags)
	}
}

// CreateUserSkillTag creates a new skill tag for a group (group admin or site admin only)
// Route: POST /api/groups/:id/user-skill-tags
func CreateUserSkillTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can create skill tags"})
			return
		}

		var req UserSkillTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		groupIDUint, _ := strconv.ParseUint(groupID, 10, 32)
		tag := models.UserSkillTag{
			GroupID: uint(groupIDUint),
			Name:    req.Name,
			Color:   req.Color,
		}

		if err := db.Create(&tag).Error; err != nil {
			logger.Error("Failed to create skill tag", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create skill tag"})
			return
		}

		c.JSON(http.StatusCreated, tag)
	}
}

// UpdateUserSkillTag updates an existing skill tag (group admin or site admin only)
// Route: PUT /api/groups/:id/user-skill-tags/:tagId
func UpdateUserSkillTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		tagID := c.Param("tagId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can update skill tags"})
			return
		}

		var tag models.UserSkillTag
		if err := db.Where("id = ? AND group_id = ?", tagID, groupID).First(&tag).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Skill tag not found"})
			return
		}

		var req UserSkillTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		if err := db.Model(&tag).Updates(map[string]interface{}{"name": req.Name, "color": req.Color}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill tag"})
			return
		}

		c.JSON(http.StatusOK, tag)
	}
}

// DeleteUserSkillTag deletes a skill tag and removes it from all users (group admin or site admin only)
// Route: DELETE /api/groups/:id/user-skill-tags/:tagId
func DeleteUserSkillTag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		tagID := c.Param("tagId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can delete skill tags"})
			return
		}

		var tag models.UserSkillTag
		if err := db.Where("id = ? AND group_id = ?", tagID, groupID).First(&tag).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Skill tag not found"})
			return
		}

		// Remove all user assignments for this tag first
		if err := db.Exec("DELETE FROM user_skill_tag_assignments WHERE user_skill_tag_id = ?", tag.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove tag assignments"})
			return
		}

		if err := db.Delete(&tag).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete skill tag"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Skill tag deleted"})
	}
}

// AssignUserSkillTagsRequest is the request body for assigning skill tags to a user
type AssignUserSkillTagsRequest struct {
	TagIDs []uint `json:"tag_ids" binding:"required"`
}

// AssignUserSkillTags sets the skill tags for a user within a group (group admin or site admin only)
// Route: PUT /api/groups/:id/members/:userId/skill-tags
func AssignUserSkillTags(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		targetUserID := c.Param("userId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only group admins can assign skill tags"})
			return
		}

		var req AssignUserSkillTagsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		// Verify the target user is a member of the group
		var ug models.UserGroup
		if err := db.Where("user_id = ? AND group_id = ?", targetUserID, groupID).First(&ug).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Validate that all tag IDs belong to this group
		if len(req.TagIDs) > 0 {
			var count int64
			db.Model(&models.UserSkillTag{}).Where("id IN ? AND group_id = ?", req.TagIDs, groupID).Count(&count)
			if int(count) != len(req.TagIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "One or more tag IDs do not belong to this group"})
				return
			}
		}

		// Load the target user
		var targetUser models.User
		if err := db.First(&targetUser, targetUserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Build the list of new tags
		var newTags []models.UserSkillTag
		if len(req.TagIDs) > 0 {
			if err := db.Where("id IN ? AND group_id = ?", req.TagIDs, groupID).Find(&newTags).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
				return
			}
		}

		// Remove existing skill tags for this group, then add new ones
		groupIDUint, _ := strconv.ParseUint(groupID, 10, 64)
		if err := db.Exec(
			"DELETE FROM user_skill_tag_assignments WHERE user_id = ? AND user_skill_tag_id IN (SELECT id FROM user_skill_tags WHERE group_id = ? AND deleted_at IS NULL)",
			targetUser.ID, uint(groupIDUint),
		).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill tags"})
			return
		}

		if len(newTags) > 0 {
			if err := db.Model(&targetUser).Association("SkillTags").Append(newTags); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign skill tags"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Skill tags updated", "tags": newTags})
	}
}
