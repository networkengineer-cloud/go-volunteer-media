package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// PromoteUser sets is_admin to true for a user
func PromoteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if user.IsAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is already admin"})
			return
		}
		if err := db.WithContext(ctx).Model(&user).Update("is_admin", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User promoted to admin"})
	}
}

// DemoteUser sets is_admin to false for a user
func DemoteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if !user.IsAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not admin"})
			return
		}
		if err := db.WithContext(ctx).Model(&user).Update("is_admin", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User demoted from admin"})
	}
}

// GetDeletedUsers returns all soft-deleted users (admin only)
func GetDeletedUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var users []models.User
		if err := db.WithContext(ctx).Unscoped().Preload("Groups").Where("deleted_at IS NOT NULL").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch deleted users"})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

// RestoreUser restores a soft-deleted user (admin only)
func RestoreUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).Unscoped().First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if user.DeletedAt.Valid {
			if err := db.WithContext(ctx).Unscoped().Model(&user).Update("deleted_at", nil).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore user"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "User restored"})
	}
}

// AdminDeleteUser soft-deletes (deactivates) a user (marks as deleted, disables login)
func AdminDeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")
		var user models.User
		if err := db.WithContext(ctx).First(&user, userId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if err := db.WithContext(ctx).Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
	}
}

// (removed duplicate import block)

type AdminCreateUserRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=50,usernamechars"`
	FirstName      string `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName       string `json:"last_name" binding:"omitempty,min=1,max=100"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"omitempty,min=8,max=72"` // Optional - if empty, send setup email
	SendSetupEmail bool   `json:"send_setup_email"`                          // If true and no password, send setup email
	IsAdmin        bool   `json:"is_admin"`
	GroupIDs       []uint `json:"group_ids"`
}

type AdminResetPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
}

// AdminCreateUser allows an admin to create a new user
// If no password is provided and SendSetupEmail is true, sends a password setup email
func AdminCreateUser(db *gorm.DB, emailService *email.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req AdminCreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate that either password is provided OR email setup is requested
		if req.Password == "" && !req.SendSetupEmail {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Either password must be provided or send_setup_email must be true"})
			return
		}

		// Check if username or email already exists
		var existing models.User
		if err := db.WithContext(ctx).Where("username = ? OR email = ?", req.Username, req.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}

		var hashedPassword string
		var setupToken string
		var setupTokenExpiry *time.Time

		if req.Password != "" {
			// Password provided - hash it
			var err error
			hashedPassword, err = auth.HashPassword(req.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}
		} else if req.SendSetupEmail {
			// No password - generate setup token for email
			if !emailService.IsConfigured() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email service is not configured. Please provide a password instead."})
				return
			}

			// Generate a temporary password that cannot be used for login
			// (user must set their own password via email)
			tempPassword, err := generateSecureToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate setup token"})
				return
			}
			hashedPassword, err = auth.HashPassword(tempPassword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process setup"})
				return
			}

			// Generate setup token
			setupToken, err = generateSecureToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate setup token"})
				return
			}

			// Hash the setup token before storing
			hashedSetupToken, err := auth.HashPassword(setupToken)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process setup token"})
				return
			}

			// Setup token expires in 24 hours (longer than password reset)
			expiry := time.Now().Add(24 * time.Hour)
			setupTokenExpiry = &expiry

			// Store hashed token in dedicated setup_token field (separate from reset tokens)
			user := models.User{
				Username:              req.Username,
				FirstName:             strings.TrimSpace(req.FirstName),
				LastName:              strings.TrimSpace(req.LastName),
				Email:                 req.Email,
				Password:              hashedPassword,
				IsAdmin:               req.IsAdmin,
				SetupToken:            hashedSetupToken,
				SetupTokenExpiry:      setupTokenExpiry,
				RequiresPasswordSetup: true, // Block login until password is set
			}

			// If group IDs are provided, fetch and associate groups
			if len(req.GroupIDs) > 0 {
				var groups []models.Group
				if err := db.WithContext(ctx).Where("id IN ?", req.GroupIDs).Find(&groups).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
					return
				}
				user.Groups = groups
			}

			if err := db.WithContext(ctx).Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}

			// Preload groups for response
			if err := db.WithContext(ctx).Preload("Groups").First(&user, user.ID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user groups"})
				return
			}

			// Send setup email (use unhashed token)
			if err := emailService.SendPasswordSetupEmail(user.Email, user.Username, setupToken); err != nil {
				// Log error but don't fail the request - user is created
				logger := middleware.GetLogger(c)
				logger.Error("Failed to send password setup email", err)
				
				// Provide actionable error message - admin can resend invitation
				c.JSON(http.StatusCreated, gin.H{
					"user": user,
					"warning": "User created successfully, but the setup email could not be sent. " +
						"You can use the 'Reset Password' button on the user's profile to send a new setup email, " +
						"or manually provide them with a temporary password.",
				})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"user":    user,
				"message": "User created successfully. Password setup email sent to " + user.Email,
			})
			return
		}

		// Regular user creation with password
		user := models.User{
			Username:  req.Username,
			FirstName: strings.TrimSpace(req.FirstName),
			LastName:  strings.TrimSpace(req.LastName),
			Email:     req.Email,
			Password:  hashedPassword,
			IsAdmin:   req.IsAdmin,
		}

		// If group IDs are provided, fetch and associate groups
		if len(req.GroupIDs) > 0 {
			var groups []models.Group
			if err := db.WithContext(ctx).Where("id IN ?", req.GroupIDs).Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
			user.Groups = groups
		}

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Preload groups for response
		if err := db.WithContext(ctx).Preload("Groups").First(&user, user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user groups"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

// GroupAdminCreateUserRequest is the request body for group admins creating users
type GroupAdminCreateUserRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=50,usernamechars"`
	FirstName      string `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName       string `json:"last_name" binding:"omitempty,min=1,max=100"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"omitempty,min=8,max=72"` // Optional - if empty, send setup email
	SendSetupEmail bool   `json:"send_setup_email"`                          // If true and no password, send setup email
	GroupIDs       []uint `json:"group_ids" binding:"required,min=1"`        // At least one group required
}

// GroupAdminCreateUser allows a group admin to create a new user and assign them to groups they administer
func GroupAdminCreateUser(db *gorm.DB, emailService *email.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		
		// Get current user ID
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		
		var req GroupAdminCreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate that either password is provided OR email setup is requested
		if req.Password == "" && !req.SendSetupEmail {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Either password must be provided or send_setup_email must be true"})
			return
		}

		// Get current user to check admin status
		var currentUser models.User
		if err := db.WithContext(ctx).First(&currentUser, currentUserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current user"})
			return
		}

		// Verify that the current user is a group admin of ALL specified groups
		// (or is a site admin, in which case they can create users for any group)
		if !currentUser.IsAdmin {
			for _, groupID := range req.GroupIDs {
				if !IsGroupAdmin(db, currentUserID.(uint), groupID) {
					logger.WithFields(map[string]interface{}{
						"current_user_id": currentUserID,
						"group_id":        groupID,
					}).Warn("Unauthorized attempt to create user for group")
					c.JSON(http.StatusForbidden, gin.H{"error": "You can only create users for groups you administer"})
					return
				}
			}
		}

		// Check if username or email already exists
		var existing models.User
		if err := db.WithContext(ctx).Where("username = ? OR email = ?", req.Username, req.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}

		var hashedPassword string
		var setupToken string
		var setupTokenExpiry *time.Time

		if req.Password != "" {
			// Password provided - hash it
			var err error
			hashedPassword, err = auth.HashPassword(req.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}
		} else if req.SendSetupEmail {
			// No password - generate setup token for email
			if !emailService.IsConfigured() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email service is not configured. Please provide a password instead."})
				return
			}

			// Generate a temporary password that cannot be used for login
			tempPassword, err := generateSecureToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate setup token"})
				return
			}
			hashedPassword, err = auth.HashPassword(tempPassword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process setup"})
				return
			}

			// Generate setup token
			setupToken, err = generateSecureToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate setup token"})
				return
			}

			// Hash the setup token before storing
			hashedSetupToken, err := auth.HashPassword(setupToken)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process setup token"})
				return
			}

			// Setup token expires in 24 hours
			expiry := time.Now().Add(24 * time.Hour)
			setupTokenExpiry = &expiry

			// Create user with setup token
			user := models.User{
				Username:              req.Username,
				FirstName:             strings.TrimSpace(req.FirstName),
				LastName:              strings.TrimSpace(req.LastName),
				Email:                 req.Email,
				Password:              hashedPassword,
				IsAdmin:               false, // Group admins cannot create site admins
				SetupToken:            hashedSetupToken,
				SetupTokenExpiry:      setupTokenExpiry,
				RequiresPasswordSetup: true,
			}

			// Fetch and associate groups
			var groups []models.Group
			if err := db.WithContext(ctx).Where("id IN ?", req.GroupIDs).Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
				return
			}
			user.Groups = groups

			if err := db.WithContext(ctx).Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}

			// Preload groups for response
			if err := db.WithContext(ctx).Preload("Groups").First(&user, user.ID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user groups"})
				return
			}

			// Send setup email
			if err := emailService.SendPasswordSetupEmail(user.Email, user.Username, setupToken); err != nil {
				logger.Error("Failed to send password setup email", err)
				
				c.JSON(http.StatusCreated, gin.H{
					"user": user,
					"warning": "User created successfully, but the setup email could not be sent. " +
						"You can use the 'Reset Password' button on the user's profile to send a new setup email.",
				})
				return
			}

			logger.WithFields(map[string]interface{}{
				"user_id":    user.ID,
				"created_by": currentUserID,
				"groups":     req.GroupIDs,
			}).Info("User created by group admin")

			c.JSON(http.StatusCreated, gin.H{
				"user":    user,
				"message": "User created successfully. Password setup email sent to " + user.Email,
			})
			return
		}

		// Regular user creation with password
		user := models.User{
			Username:  req.Username,
			FirstName: strings.TrimSpace(req.FirstName),
			LastName:  strings.TrimSpace(req.LastName),
			Email:     req.Email,
			Password:  hashedPassword,
			IsAdmin:   false, // Group admins cannot create site admins
		}

		// Fetch and associate groups
		var groups []models.Group
		if err := db.WithContext(ctx).Where("id IN ?", req.GroupIDs).Find(&groups).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
			return
		}
		user.Groups = groups

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Preload groups for response
		if err := db.WithContext(ctx).Preload("Groups").First(&user, user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user groups"})
			return
		}

		logger.WithFields(map[string]interface{}{
			"user_id":    user.ID,
			"created_by": currentUserID,
			"groups":     req.GroupIDs,
		}).Info("User created by group admin")

		c.JSON(http.StatusCreated, user)
	}
}

// AdminResetUserPassword allows an admin, group admin, or the user themselves to reset a password
func AdminResetUserPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		userId := c.Param("userId")

		userIdInt, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		var req AdminResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current user ID
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Find the target user with their groups
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").First(&user, userIdInt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			}
			return
		}

		// Self-reset is always allowed (user is already authenticated via JWT)
		isSelf := currentUserID.(uint) == uint(userIdInt)

		// For self-resets, verify the current password server-side
		if isSelf {
			if req.CurrentPassword == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is required"})
				return
			}
			if err := auth.CheckPassword(user.Password, req.CurrentPassword); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
				return
			}
		}

		if !isSelf && !middleware.IsSiteAdmin(c) {
			// Group admin path: cannot reset password of admin users
			if isTargetUserAdmin(ctx, db, &user) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Group admins can only reset passwords for regular volunteers"})
				return
			}

			// Check if caller is group admin of any shared group.
			// Regular (non-admin) members are excluded by the is_group_admin=true filter.
			hasAccess := false
			for _, targetGroup := range user.Groups {
				var userGroup models.UserGroup
				err := db.WithContext(ctx).Where("user_id = ? AND group_id = ? AND is_group_admin = ?",
					currentUserID, targetGroup.ID, true).First(&userGroup).Error
				if err == nil {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				logger.WithFields(map[string]interface{}{
					"current_user_id": currentUserID,
					"target_user_id":  userId,
				}).Warn("Unauthorized attempt to reset password")
				c.JSON(http.StatusForbidden, gin.H{"error": "You must be a site admin or group admin to reset passwords"})
				return
			}
		}

		// Hash the new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Update password and clear any lockouts or reset tokens
		if err := db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
			"password":              hashedPassword,
			"reset_token":           "",
			"reset_token_expiry":    nil,
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
	}
}

// UpdateUserRequest is the request body for updating user information.
// Empty strings for FirstName/LastName are allowed to clear those fields.
type UpdateUserRequest struct {
	FirstName   string `json:"first_name" binding:"omitempty,max=100"`
	LastName    string `json:"last_name" binding:"omitempty,max=100"`
	Email       string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phone_number" binding:"omitempty,max=20"`
}

// applyUserUpdate validates email uniqueness, applies the update, reloads the
// user with groups, and writes the JSON response to c. Callers should return
// immediately after calling this function.
func applyUserUpdate(ctx context.Context, db *gorm.DB, c *gin.Context, user *models.User, req UpdateUserRequest) {
	if req.Email != user.Email {
		if err := validateEmailUniqueness(ctx, db, req.Email, user.ID); err != nil {
			if errors.Is(err, ErrEmailInUse) {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate email"})
			}
			return
		}
	}

	updates := map[string]interface{}{
		"first_name":   strings.TrimSpace(req.FirstName),
		"last_name":    strings.TrimSpace(req.LastName),
		"phone_number": strings.TrimSpace(req.PhoneNumber),
		"email":        req.Email,
	}

	if err := db.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	if err := db.WithContext(ctx).Preload("Groups").First(user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AdminUpdateUser allows an admin to update a user's information
func AdminUpdateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.Param("userId")

		// Parse and validate userId
		userIdInt, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		var req UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		if err := db.WithContext(ctx).First(&user, userIdInt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			}
			return
		}

		applyUserUpdate(ctx, db, c, &user, req)
	}
}

// isTargetUserAdmin checks if the target user holds any admin role (site admin
// or group admin in any group). Group admins may only modify regular volunteers.
func isTargetUserAdmin(ctx context.Context, db *gorm.DB, user *models.User) bool {
	if user.IsAdmin {
		return true
	}
	var count int64
	db.WithContext(ctx).Model(&models.UserGroup{}).
		Where("user_id = ? AND is_group_admin = ?", user.ID, true).
		Count(&count)
	return count > 0
}

// GroupAdminUpdateUser allows a group admin to update a user's information for users in their groups
func GroupAdminUpdateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		userId := c.Param("userId")

		// Parse and validate userId
		userIdInt, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Get current user ID from auth context
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find the target user with their groups
		var user models.User
		if err := db.WithContext(ctx).Preload("Groups").First(&user, userIdInt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			}
			return
		}

		// Authorization: a group admin can update a user if they admin ANY group the
		// target user belongs to. This intentionally differs from GroupAdminCreateUser
		// which requires admin of ALL specified groups. The update case is more
		// permissive because the user already exists in those groups — the group admin
		// is only modifying profile fields, not group assignments.
		// Users without groups can only be managed by site admins.
		if !middleware.IsSiteAdmin(c) {
			if len(user.Groups) == 0 {
				logger.WithFields(map[string]interface{}{
					"current_user_id": currentUserID,
					"target_user_id":  userId,
				}).Warn("Group admin attempted to update user with no groups")
				c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update users with no group assignments. Please contact a site administrator."})
				return
			}

			// Group admins cannot modify site admins or other group admins
			if isTargetUserAdmin(ctx, db, &user) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Group admins can only update regular volunteers"})
				return
			}

			hasAccess := false
			for _, targetGroup := range user.Groups {
				var userGroup models.UserGroup
				err := db.WithContext(ctx).Where("user_id = ? AND group_id = ? AND is_group_admin = ?",
					currentUserID, targetGroup.ID, true).First(&userGroup).Error
				if err == nil {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				logger.WithFields(map[string]interface{}{
					"current_user_id": currentUserID,
					"target_user_id":  userId,
				}).Warn("Unauthorized attempt to update user")
				c.JSON(http.StatusForbidden, gin.H{"error": "You must be a site admin or group admin to update user information"})
				return
			}
		}

		applyUserUpdate(ctx, db, c, &user, req)
	}
}
