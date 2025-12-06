// isValidGroupMeBotID validates the GroupMe bot ID format (40-char hex string)
package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)

type GroupRequest struct {
	Name           string `json:"name" binding:"required,min=2,max=100"`
	Description    string `json:"description" binding:"max=500"`
	ImageURL       string `json:"image_url,omitempty"`
	HeroImageURL   string `json:"hero_image_url,omitempty"`
	HasProtocols   bool   `json:"has_protocols"`
	GroupMeBotID   string `json:"groupme_bot_id,omitempty"`
	GroupMeEnabled bool   `json:"groupme_enabled"`
}

// isValidGroupMeBotID validates the GroupMe bot ID format (26-char hex string)
func isValidGroupMeBotID(id string) bool {
	if id == "" {
		return true // allow empty (not configured)
	}
	if len(id) != 26 {
		return false
	}
	for _, c := range id {
		if !(('a' <= c && c <= 'f') || ('A' <= c && c <= 'F') || ('0' <= c && c <= '9')) {
			return false
		}
	}
	return true
}

// UploadGroupImage handles secure group image uploads (admin only)
func UploadGroupImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		file, err := c.FormFile("image")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Validate file upload (size, type, content)
		if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil {
			logger.Error("File validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Get validated extension
		ext := strings.ToLower(filepath.Ext(file.Filename))

		// Generate unique filename
		fname := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), ext)
		uploadPath := filepath.Join("public", "uploads", fname)

		logger.WithField("path", uploadPath).Debug("Saving group image")

		// Save file
		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			logger.Error("Failed to save file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
			return
		}

		// Return public URL
		url := "/uploads/" + fname
		logger.WithField("url", url).Info("Group image uploaded successfully")
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

// GetGroups returns all groups the user has access to
func GetGroups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		isAdmin, exists := c.Get("is_admin")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin context not found"})
			return
		}

		var groups []models.Group

		adminFlag, ok := isAdmin.(bool)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid admin flag"})
			return
		}
		if adminFlag {
			// Admins can see all groups
			if err := db.WithContext(ctx).Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
		} else {
			// Regular users see only their groups
			var user models.User
			if err := db.WithContext(ctx).Preload("Groups").First(&user, userID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user groups"})
				return
			}
			groups = user.Groups
		}

		c.JSON(http.StatusOK, groups)
	}
}

// GetGroup returns a specific group by ID
func GetGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user has access to this group
		if !isAdmin.(bool) {
			var user models.User
			if err := db.Preload("Groups", "id = ?", groupID).First(&user, userID).Error; err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
			if len(user.Groups) == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
		}

		c.JSON(http.StatusOK, group)
	}
}

// CreateGroup creates a new group (admin only)
func CreateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set default hero image if not provided
		heroImageURL := req.HeroImageURL
		if heroImageURL == "" {
			heroImageURL = "/default-hero.svg"
		}

		// Validate GroupMeBotID
		if !isValidGroupMeBotID(req.GroupMeBotID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GroupMe bot ID. Must be a 26-character hexadecimal string."})
			return
		}

		group := models.Group{
			Name:           req.Name,
			Description:    req.Description,
			ImageURL:       req.ImageURL,
			HeroImageURL:   heroImageURL,
			HasProtocols:   req.HasProtocols,
			GroupMeBotID:   req.GroupMeBotID,
			GroupMeEnabled: req.GroupMeEnabled,
		}

		if err := db.Create(&group).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
			return
		}

		c.JSON(http.StatusCreated, group)
	}
}

// UpdateGroup updates an existing group (admin only)
func UpdateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		var req GroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		group.Name = req.Name
		group.Description = req.Description
		group.ImageURL = req.ImageURL
		group.HeroImageURL = req.HeroImageURL
		group.HasProtocols = req.HasProtocols
		// Validate GroupMeBotID
		if !isValidGroupMeBotID(req.GroupMeBotID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GroupMe bot ID. Must be a 26-character hexadecimal string."})
			return
		}
		group.GroupMeBotID = req.GroupMeBotID
		group.GroupMeEnabled = req.GroupMeEnabled

		if err := db.Save(&group).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

// DeleteGroup deletes a group (admin only)
func DeleteGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")

		if err := db.Delete(&models.Group{}, groupID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
	}
}

// AddUserToGroup adds a user to a group (admin only)
func AddUserToGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var user models.User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		var group models.Group
		if err := db.First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if err := db.Model(&user).Association("Groups").Append(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User added to group successfully"})
	}
}

// RemoveUserFromGroup removes a user from a group (admin only)
func RemoveUserFromGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var user models.User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		var group models.Group
		if err := db.First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if err := db.Model(&user).Association("Groups").Delete(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User removed from group successfully"})
	}
}

// IsGroupAdmin checks if a user is an admin for a specific group
// Returns true if user is a site admin OR a group admin for the specified group
func IsGroupAdmin(db *gorm.DB, userID uint, groupID uint) bool {
	var userGroup models.UserGroup
	if err := db.Where("user_id = ? AND group_id = ?", userID, groupID).First(&userGroup).Error; err != nil {
		return false
	}
	return userGroup.IsGroupAdmin
}

// IsGroupAdminOrSiteAdmin checks if a user is a site admin OR a group admin for the specified group
func IsGroupAdminOrSiteAdmin(c *gin.Context, db *gorm.DB, groupID uint) bool {
	// Check if site admin
	if middleware.IsSiteAdmin(c) {
		return true
	}

	// Check if group admin
	userID, exists := c.Get("user_id")
	if !exists {
		return false
	}

	return IsGroupAdmin(db, userID.(uint), groupID)
}

// PromoteGroupAdmin promotes a user to group admin status for a specific group
// Accessible by site admins or group admins of the specific group
func PromoteGroupAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		// Get current user
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Check authorization: must be site admin OR group admin of this group
		var currentUser models.User
		if err := db.WithContext(ctx).First(&currentUser, currentUserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		if !currentUser.IsAdmin && !IsGroupAdmin(db, currentUserID.(uint), uint(groupID)) {
			logger.WithFields(map[string]interface{}{
				"current_user_id": currentUserID,
				"group_id":        groupID,
			}).Warn("Unauthorized attempt to promote group admin")
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be a site admin or group admin to promote users"})
			return
		}

		// Verify user exists
		var user models.User
		if err := db.WithContext(ctx).First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is a member of the group
		var userGroup models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).First(&userGroup).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Check if already a group admin
		if userGroup.IsGroupAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is already a group admin"})
			return
		}

		// Promote to group admin
		if err := db.WithContext(ctx).Model(&userGroup).Update("is_group_admin", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user to group admin"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User promoted to group admin"})
	}
}

// DemoteGroupAdmin removes group admin status from a user for a specific group
// Accessible by site admins or group admins of the specific group
func DemoteGroupAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		
		userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		// Get current user
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Check authorization: must be site admin OR group admin of this group
		var currentUser models.User
		if err := db.WithContext(ctx).First(&currentUser, currentUserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		if !currentUser.IsAdmin && !IsGroupAdmin(db, currentUserID.(uint), uint(groupID)) {
			logger.WithFields(map[string]interface{}{
				"current_user_id": currentUserID,
				"group_id":        groupID,
			}).Warn("Unauthorized attempt to demote group admin")
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be a site admin or group admin to demote users"})
			return
		}

		// Verify user exists
		var user models.User
		if err := db.WithContext(ctx).First(&user, uint(userID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, uint(groupID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is a member of the group
		var userGroup models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).First(&userGroup).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Check if user is a group admin
		if !userGroup.IsGroupAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a group admin"})
			return
		}

		// Demote from group admin
		if err := db.WithContext(ctx).Model(&userGroup).Update("is_group_admin", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote user from group admin"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User demoted from group admin"})
	}
}

// GetGroupMembers returns all members of a group with their group admin status
func GetGroupMembers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		// Check if user has access to this group (is member or site admin)
		currentUserID, _ := c.Get("user_id")
		isSiteAdmin := middleware.IsSiteAdmin(c)

		var currentUserGroupAdmin bool
		if !isSiteAdmin {
			var userGroup models.UserGroup
			if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", currentUserID, groupID).First(&userGroup).Error; err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
			currentUserGroupAdmin = userGroup.IsGroupAdmin
		}

		// Get all members with their group admin status
		var userGroups []models.UserGroup
		if err := db.WithContext(ctx).Preload("User").Where("group_id = ?", groupID).Find(&userGroups).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
			return
		}

		// Build response with user info and group admin status
		type MemberInfo struct {
			UserID       uint   `json:"user_id"`
			Username     string `json:"username"`
			Email        string `json:"email"`
			PhoneNumber  string `json:"phone_number"`
			IsGroupAdmin bool   `json:"is_group_admin"`
			IsSiteAdmin  bool   `json:"is_site_admin"`
		}

		var members []MemberInfo
		for _, ug := range userGroups {
			// Show email and phone number respecting privacy settings
			// Site admins always see all contact info
			// Group admins of this group see all contact info for their members
			// Users viewing their own profile always see their contact info
			// Other users see contact info only if not hidden

			email := ""
			phoneNumber := ""

			if isSiteAdmin || currentUserGroupAdmin || currentUserID.(uint) == ug.UserID {
				// Site admins, group admins, and users viewing their own profile always see all contact info
				email = ug.User.Email
				phoneNumber = ug.User.PhoneNumber
			} else {
				// Regular users see contact info only if not hidden by the target user
				if !ug.User.HideEmail {
					email = ug.User.Email
				}
				if !ug.User.HidePhoneNumber {
					phoneNumber = ug.User.PhoneNumber
				}
			}

			members = append(members, MemberInfo{
				UserID:       ug.UserID,
				Username:     ug.User.Username,
				Email:        email,
				PhoneNumber:  phoneNumber,
				IsGroupAdmin: ug.IsGroupAdmin,
				IsSiteAdmin:  ug.User.IsAdmin,
			})
		}

		c.JSON(http.StatusOK, members)
	}
}

// GetGroupMembership returns the current user's membership info for a specific group
// including whether they are a group admin
func GetGroupMembership(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		isSiteAdmin := middleware.IsSiteAdmin(c)

		// Get user's membership in this group
		var userGroup models.UserGroup
		err = db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).First(&userGroup).Error
		if err != nil {
			// If not a member but is site admin, still return info
			if isSiteAdmin {
				c.JSON(http.StatusOK, gin.H{
					"user_id":        userID,
					"group_id":       groupID,
					"is_member":      false,
					"is_group_admin": false,
					"is_site_admin":  true,
				})
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":        userID,
			"group_id":       groupID,
			"is_member":      true,
			"is_group_admin": userGroup.IsGroupAdmin,
			"is_site_admin":  isSiteAdmin,
		})
	}
}

// AddMemberToGroup adds a user to a group (group admin or site admin)
// This allows group admins to add new members to their group
func AddMemberToGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Verify target user exists
		var targetUser models.User
		if err := db.WithContext(ctx).First(&targetUser, uint(targetUserID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is already a member
		var existingMembership models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", targetUserID, groupID).First(&existingMembership).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of this group"})
			return
		}

		// Add user to group
		if err := db.WithContext(ctx).Model(&targetUser).Association("Groups").Append(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User added to group successfully"})
	}
}

// RemoveMemberFromGroup removes a user from a group (group admin or site admin)
// This allows group admins to remove members from their group
func RemoveMemberFromGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Verify target user exists
		var targetUser models.User
		if err := db.WithContext(ctx).First(&targetUser, uint(targetUserID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is a member
		var existingMembership models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", targetUserID, groupID).First(&existingMembership).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Remove user from group
		if err := db.WithContext(ctx).Model(&targetUser).Association("Groups").Delete(&group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User removed from group successfully"})
	}
}

// PromoteMemberToGroupAdmin promotes a user to group admin status (group admin or site admin)
func PromoteMemberToGroupAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Verify target user exists
		var targetUser models.User
		if err := db.WithContext(ctx).First(&targetUser, uint(targetUserID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is a member of the group
		var userGroup models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", targetUserID, groupID).First(&userGroup).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Check if already a group admin
		if userGroup.IsGroupAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is already a group admin"})
			return
		}

		// Promote to group admin
		if err := db.WithContext(ctx).Model(&userGroup).Update("is_group_admin", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user to group admin"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User promoted to group admin"})
	}
}

// DemoteMemberFromGroupAdmin removes group admin status from a user (group admin or site admin)
func DemoteMemberFromGroupAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Verify target user exists
		var targetUser models.User
		if err := db.WithContext(ctx).First(&targetUser, uint(targetUserID)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Verify group exists
		var group models.Group
		if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		// Check if user is a member of the group
		var userGroup models.UserGroup
		if err := db.WithContext(ctx).Where("user_id = ? AND group_id = ?", targetUserID, groupID).First(&userGroup).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
			return
		}

		// Check if user is a group admin
		if !userGroup.IsGroupAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a group admin"})
			return
		}

		// Demote from group admin
		if err := db.WithContext(ctx).Model(&userGroup).Update("is_group_admin", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote user from group admin"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User demoted from group admin"})
	}
}

// UpdateGroupSettings updates group settings (group admin or site admin)
// Group admins can update settings for their own group
func UpdateGroupSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		// Check for group admin or site admin access
		if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req GroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		group.Name = req.Name
		group.Description = req.Description
		group.ImageURL = req.ImageURL
		group.HeroImageURL = req.HeroImageURL
		group.HasProtocols = req.HasProtocols
		// Validate GroupMeBotID
		if !isValidGroupMeBotID(req.GroupMeBotID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GroupMe bot ID. Must be a 26-character hexadecimal string."})
			return
		}
		group.GroupMeBotID = req.GroupMeBotID
		group.GroupMeEnabled = req.GroupMeEnabled

		if err := db.Save(&group).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
			return
		}

		c.JSON(http.StatusOK, group)
	}
}
