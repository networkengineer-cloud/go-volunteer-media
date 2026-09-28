package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/authz"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// callerCan reports whether the authenticated caller may perform action in
// the group named by groupID, a path parameter. It is the handler-side entry
// point to the central policy in internal/authz: a malformed group ID, a
// missing caller or a database error all deny. The handler writes its own
// 403 when this returns false.
func callerCan(c *gin.Context, db *gorm.DB, action authz.Action, groupID string) bool {
	id, ok := authz.ParseGroupID(groupID)
	if !ok {
		return false
	}
	return authz.CallerCan(c, db, action, id)
}

// manageUserDenials holds a handler's 403 messages for each reason
// authz.CheckManageUser can deny, so each endpoint keeps its wording.
type manageUserDenials struct {
	targetIsSiteAdmin string
	targetHasNoGroups string
	noSharedGroup     string
}

// callerCanManageUser applies authz.CheckManageUser to target (whose Groups
// must be loaded). On denial or error it logs, writes the response and
// returns false.
func callerCanManageUser(c *gin.Context, db *gorm.DB, target *models.User, msgs manageUserDenials) bool {
	subject, ok := authz.SubjectFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return false
	}
	denial, err := authz.CheckManageUser(c.Request.Context(), db, subject, target)
	if err != nil {
		middleware.GetLogger(c).Error("Failed to verify user management access", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return false
	}

	var msg string
	switch denial {
	case authz.UserAllowed:
		return true
	case authz.DenyTargetIsSiteAdmin:
		msg = msgs.targetIsSiteAdmin
	case authz.DenyTargetHasNoGroups:
		msg = msgs.targetHasNoGroups
	default:
		msg = msgs.noSharedGroup
	}
	middleware.GetLogger(c).WithFields(map[string]interface{}{
		"current_user_id": subject.UserID,
		"target_user_id":  target.ID,
		"endpoint":        c.FullPath(),
	}).Warn("Unauthorized user management attempt")
	c.JSON(http.StatusForbidden, gin.H{"error": msg})
	return false
}
