// Package authz is the single authorization policy for the API (roadmap
// item AR-2). Handlers ask one question, "may this caller perform this
// action in this group?", and the answer comes from one table (policy)
// instead of being re-derived inline in every handler.
//
// The model:
//
//   - A Subject is the authenticated caller (user ID + site-admin flag),
//     read from the Gin context that middleware.AuthRequired populated.
//   - A Role is the subject's standing in one group: none, member, group
//     admin, or site admin (which outranks every group role).
//   - An Action names what the caller wants to do. Each Action maps to the
//     minimum Role that may perform it.
//
// Adding a role (coordinator, kiosk device, applicant, mentor) means adding
// a Role and deciding, per Action, whether it qualifies - here, with a test -
// rather than touching every handler.
//
// Ownership rules ("a member may edit their own comment") stay in handlers:
// they depend on the resource, not on the caller's role. The handler asks
// authz for the role-level decision, then applies the ownership check.
package authz

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// Role is a subject's standing within one group. Roles are ordered: each
// one can do everything the roles below it can.
type Role int

const (
	// RoleNone is a caller with no membership in the group.
	RoleNone Role = iota
	// RoleMember is an ordinary member of the group.
	RoleMember
	// RoleGroupAdmin is a member flagged as an admin of this group
	// (UserGroup.IsGroupAdmin).
	RoleGroupAdmin
	// RoleSiteAdmin is a site-wide admin (User.IsAdmin); it applies in every
	// group whether or not the user is a member.
	RoleSiteAdmin
)

// String returns the role's name for logs and test output.
func (r Role) String() string {
	switch r {
	case RoleNone:
		return "none"
	case RoleMember:
		return "member"
	case RoleGroupAdmin:
		return "group_admin"
	case RoleSiteAdmin:
		return "site_admin"
	default:
		return "unknown(" + strconv.Itoa(int(r)) + ")"
	}
}

// Action is something a caller wants to do. Values are stable strings so
// they can appear in logs and audit records.
type Action string

// Group-scoped actions. Each is checked against the caller's Role in the
// group named by the request.
const (
	// ViewGroup reads anything in a group: animals, comments, media,
	// protocols, scripts, documents, updates, the activity feed, search,
	// tags, the schedule overview and open coverage requests.
	ViewGroup Action = "group.view"
	// PostContent adds to a group: comments, updates, photos and videos,
	// animal profile pictures. Editing or deleting one's *own* content also
	// needs this; the ownership check stays in the handler.
	PostContent Action = "group.post"
	// ManageOwnSchedule edits the caller's own recurring shifts and their own
	// coverage requests, and claims other members' open requests.
	ManageOwnSchedule Action = "schedule.own"

	// ManageAnimals creates, edits and deletes animals and assigns animal
	// tags and scripts to them.
	ManageAnimals Action = "animals.manage"
	// ManageContent creates, edits and deletes the group's reference
	// material: protocols, scripts, documents, animal tags, comment tags.
	ManageContent Action = "content.manage"
	// ModerateContent sees and acts on other members' content: comment edit
	// history, deleted comments and images, deleting others' comments and
	// updates.
	ModerateContent Action = "content.moderate"
	// ManageSchedule edits other members' shifts, reassigns shifts, sets
	// coverage priority, and cancels or reopens others' coverage requests.
	ManageSchedule Action = "schedule.manage"
	// ManageMembers adds, removes, promotes and demotes group members,
	// creates users in the group, and manages member skill tags.
	ManageMembers Action = "members.manage"
	// ViewMemberDetails sees members' private details: contact info they
	// chose to hide, last login, pending password setup.
	ViewMemberDetails Action = "members.view_private"
	// ManageGroupSettings edits group-level settings (GroupMe bot, etc.).
	ManageGroupSettings Action = "group.settings"
	// Announce sends group-wide announcements and emails.
	Announce Action = "group.announce"
	// ModerateMedia deletes photos and videos other members uploaded.
	ModerateMedia Action = "media.moderate"
	// ConfigureGroupFeatures turns group features (e.g. scheduling) on or
	// off. Site admins only.
	ConfigureGroupFeatures Action = "group.features"
)

// policy maps every Action to the minimum Role allowed to perform it.
// An Action missing from this table is denied to everyone, so a typo fails
// closed.
var policy = map[Action]Role{
	ViewGroup:         RoleMember,
	PostContent:       RoleMember,
	ManageOwnSchedule: RoleMember,

	ManageAnimals:       RoleGroupAdmin,
	ManageContent:       RoleGroupAdmin,
	ModerateContent:     RoleGroupAdmin,
	ManageSchedule:      RoleGroupAdmin,
	ManageMembers:       RoleGroupAdmin,
	ViewMemberDetails:   RoleGroupAdmin,
	ManageGroupSettings: RoleGroupAdmin,
	Announce:            RoleGroupAdmin,

	// Site admins only, unlike ModerateContent: group admins can delete
	// others' comments but not their photos or videos. Kept as it was
	// before the central policy; lowering it to RoleGroupAdmin is a one-line
	// product decision.
	ModerateMedia:          RoleSiteAdmin,
	ConfigureGroupFeatures: RoleSiteAdmin,
}

// Allows reports whether role may perform action. It is a pure function of
// the policy table; unknown actions are denied.
func Allows(role Role, action Action) bool {
	min, ok := policy[action]
	if !ok {
		return false
	}
	return role >= min
}

// Subject is the authenticated caller.
type Subject struct {
	UserID      uint
	IsSiteAdmin bool
}

// SubjectFromContext reads the caller that middleware.AuthRequired stored on
// the Gin context. ok is false when there is no authenticated user.
func SubjectFromContext(c *gin.Context) (Subject, bool) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == 0 {
		return Subject{}, false
	}
	return Subject{UserID: userID, IsSiteAdmin: middleware.IsSiteAdmin(c)}, true
}

// GroupRole resolves s's Role in groupID. Site admins are RoleSiteAdmin
// without a query. Otherwise the caller must have a UserGroup row for the
// group, and neither the user nor the group may be soft-deleted.
func GroupRole(ctx context.Context, db *gorm.DB, s Subject, groupID uint) (Role, error) {
	if s.IsSiteAdmin {
		return RoleSiteAdmin, nil
	}
	if s.UserID == 0 || groupID == 0 {
		return RoleNone, nil
	}

	var rows []struct{ IsGroupAdmin bool }
	err := db.WithContext(ctx).
		Model(&models.UserGroup{}).
		Select("user_groups.is_group_admin").
		Joins("JOIN users ON users.id = user_groups.user_id AND users.deleted_at IS NULL").
		Joins(`JOIN "groups" ON "groups".id = user_groups.group_id AND "groups".deleted_at IS NULL`).
		Where("user_groups.user_id = ? AND user_groups.group_id = ?", s.UserID, groupID).
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return RoleNone, err
	}
	if len(rows) == 0 {
		return RoleNone, nil
	}
	if rows[0].IsGroupAdmin {
		return RoleGroupAdmin, nil
	}
	return RoleMember, nil
}

// Can reports whether s may perform action in groupID.
func Can(ctx context.Context, db *gorm.DB, s Subject, action Action, groupID uint) (bool, error) {
	role, err := GroupRole(ctx, db, s, groupID)
	if err != nil {
		return false, err
	}
	return Allows(role, action), nil
}

// CallerCan is Can for a handler: it reads the caller from c and fails
// closed - no authenticated caller, or a database error, is a denial (the
// error is logged). The handler writes its own 403 when this returns false.
func CallerCan(c *gin.Context, db *gorm.DB, action Action, groupID uint) bool {
	s, ok := SubjectFromContext(c)
	if !ok {
		return false
	}
	allowed, err := Can(requestContext(c), db, s, action, groupID)
	if err != nil {
		logging.WithFields(map[string]interface{}{
			"user_id":  s.UserID,
			"group_id": groupID,
			"action":   string(action),
		}).Error("authorization check failed", err)
		return false
	}
	return allowed
}

// CallerRole resolves the caller's Role in groupID, for handlers that shape
// their response by role (e.g. showing admin-only fields) rather than
// allowing or denying outright. It fails closed to RoleNone.
func CallerRole(c *gin.Context, db *gorm.DB, groupID uint) Role {
	s, ok := SubjectFromContext(c)
	if !ok {
		return RoleNone
	}
	role, err := GroupRole(requestContext(c), db, s, groupID)
	if err != nil {
		logging.WithFields(map[string]interface{}{
			"user_id":  s.UserID,
			"group_id": groupID,
		}).Error("authorization role lookup failed", err)
		return RoleNone
	}
	return role
}

// requestContext returns the request's context, or Background for a Gin
// context built without a request (as some handler tests do).
func requestContext(c *gin.Context) context.Context {
	if c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}

// ParseGroupID parses a group ID path parameter. ok is false for anything
// that is not a positive 32-bit integer, which callers treat as a denial.
func ParseGroupID(raw string) (uint, bool) {
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// UserDenial explains why a user-management action on another user was
// denied, so handlers can keep their specific messages.
type UserDenial int

const (
	// UserAllowed means the caller may manage the target user.
	UserAllowed UserDenial = iota
	// DenyTargetIsSiteAdmin: only site admins may manage site admins.
	DenyTargetIsSiteAdmin
	// DenyTargetHasNoGroups: a user in no group can only be managed by a
	// site admin.
	DenyTargetHasNoGroups
	// DenyNoSharedAdminGroup: the caller administers none of the target's
	// groups.
	DenyNoSharedAdminGroup
)

// ErrNoSubject is returned when there is no authenticated caller.
var ErrNoSubject = errors.New("authz: no authenticated caller")

// CheckManageUser decides whether s may manage target's account (update
// profile, reset password, resend invitation, unlock). Site admins may
// manage anyone. Otherwise the caller needs ManageMembers in at least one of
// the target's groups, and the target must not be a site admin.
// target.Groups must be loaded.
func CheckManageUser(ctx context.Context, db *gorm.DB, s Subject, target *models.User) (UserDenial, error) {
	if s.IsSiteAdmin {
		return UserAllowed, nil
	}
	if s.UserID == 0 {
		return DenyNoSharedAdminGroup, ErrNoSubject
	}
	if target.IsAdmin {
		return DenyTargetIsSiteAdmin, nil
	}
	if len(target.Groups) == 0 {
		return DenyTargetHasNoGroups, nil
	}

	scope, err := GroupsWhere(ctx, db, s, ManageMembers)
	if err != nil {
		return DenyNoSharedAdminGroup, err
	}
	for _, g := range target.Groups {
		if scope.Contains(g.ID) {
			return UserAllowed, nil
		}
	}
	return DenyNoSharedAdminGroup, nil
}

// GroupScope is the set of groups in which a caller may perform an action.
type GroupScope struct {
	// All is true for callers whose role applies in every group (site
	// admins); GroupIDs is then nil.
	All      bool
	GroupIDs []uint
}

// Empty reports whether the scope contains no group at all.
func (g GroupScope) Empty() bool { return !g.All && len(g.GroupIDs) == 0 }

// Contains reports whether groupID is in the scope.
func (g GroupScope) Contains(groupID uint) bool {
	if g.All {
		return true
	}
	for _, id := range g.GroupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

// GroupsWhere returns every group in which s may perform action, for
// endpoints that span groups (bulk edits, cross-group listings). Filter
// queries with GroupIDs unless All is set. Soft-deleted groups are excluded.
func GroupsWhere(ctx context.Context, db *gorm.DB, s Subject, action Action) (GroupScope, error) {
	min, ok := policy[action]
	if !ok {
		return GroupScope{}, nil
	}
	if s.IsSiteAdmin {
		return GroupScope{All: true}, nil
	}
	if s.UserID == 0 || min > RoleGroupAdmin {
		return GroupScope{}, nil
	}

	q := db.WithContext(ctx).
		Model(&models.UserGroup{}).
		Joins("JOIN users ON users.id = user_groups.user_id AND users.deleted_at IS NULL").
		Joins(`JOIN "groups" ON "groups".id = user_groups.group_id AND "groups".deleted_at IS NULL`).
		Where("user_groups.user_id = ?", s.UserID)
	if min == RoleGroupAdmin {
		q = q.Where("user_groups.is_group_admin = ?", true)
	}
	var ids []uint
	if err := q.Order("user_groups.group_id").Pluck("user_groups.group_id", &ids).Error; err != nil {
		return GroupScope{}, err
	}
	return GroupScope{GroupIDs: ids}, nil
}
