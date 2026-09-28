---
name: group-auth-pattern
description: Background knowledge for implementing authorization in go-volunteer-media Gin handlers. Explains the central policy in internal/authz (roles, actions, callerCan / CallerRole / GroupsWhere / CheckManageUser), the route-level tiers (public, authenticated, AdminRequired), how to add an action or a role, and common security anti-patterns to avoid. Automatically loaded when writing or reviewing handler code.
user-invocable: false
---

# Group Authorization Pattern

Every authorization decision goes through **one policy table** in
`internal/authz` (roadmap item AR-2). Handlers ask "may the caller do
*this action* in *this group*?" — they never check `is_admin` or
`is_group_admin` themselves.

## The model

| Concept | What it is |
| --- | --- |
| `authz.Subject` | The caller: user ID + site-admin flag, read from the auth context. |
| `authz.Role` | The caller's standing in one group: `RoleNone` < `RoleMember` < `RoleGroupAdmin` < `RoleSiteAdmin`. Site admin applies in every group, member or not. |
| `authz.Action` | What the caller wants to do, e.g. `ViewGroup`, `ManageAnimals`. |
| `policy` | `map[Action]Role` — the minimum role for each action. Unknown actions are denied. |

Role resolution (`authz.GroupRole`) needs a `UserGroup` row for the group,
and neither the user nor the group may be soft-deleted.

### Actions

| Action | Minimum role | Covers |
| --- | --- | --- |
| `ViewGroup` | member | Reading anything in a group: animals, comments, media, protocols, scripts, documents, updates, feed, search, tags, schedule overview, open coverage requests |
| `PostContent` | member | Comments, updates, photos/videos, profile pictures; editing/deleting *own* content (ownership checked in the handler) |
| `ManageOwnSchedule` | member | Own shifts and coverage requests; claiming others' open requests |
| `ManageAnimals` | group admin | Animal CRUD, assigning tags/scripts to animals, bulk edit |
| `ManageContent` | group admin | Protocols, scripts, documents, animal tags, comment tags |
| `ModerateContent` | group admin | Comment history, deleted comments/images, deleting others' comments and updates |
| `ManageSchedule` | group admin | Others' shifts, reassigning, priority, cancelling/reopening others' requests, reminders |
| `ManageMembers` | group admin | Add/remove/promote/demote members, create users, skill tags, managing member accounts |
| `ViewMemberDetails` | group admin | Hidden contact info, last login, setup status |
| `ManageGroupSettings` | group admin | Group settings (GroupMe bot, …) |
| `Announce` | group admin | Announcements and update emails |
| `ModerateMedia` | site admin | Deleting others' photos/videos |
| `ConfigureGroupFeatures` | site admin | Turning group features (scheduling) on/off |

The authoritative list is the `policy` map in `internal/authz/authz.go`, and
`TestPolicyMatrix` pins every action × role cell.

## Group-scoped handler

```go
func GetFoos(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        db := middleware.GetDB(c, db) // request-scoped DB; never reassign the closure's db
        groupID := c.Param("id")

        if !callerCan(c, db, authz.ViewGroup, groupID) {
            respondForbidden(c, "forbidden") // 403, not 404 — do not leak resource existence
            return
        }
        // ... safe to query group data
    }
}
```

`callerCan` (in `internal/handlers/authz.go`) parses the group ID path
parameter and calls `authz.CallerCan`. It **fails closed**: a malformed group
ID, a missing caller, or a database error is a denial (errors are logged).
The handler writes its own 403 so existing messages stay the same.

When the group ID is already a `uint` (e.g. from a loaded row), call
`authz.CallerCan(c, db, action, groupID)` directly:

```go
if !authz.CallerCan(c, db, authz.ViewGroup, doc.GroupID) { ... }
```

### Ownership plus a role

Keep the ownership check in the handler and ask authz for the override:

```go
if comment.UserID != userID && !callerCan(c, db, authz.ModerateContent, groupID) {
    respondForbidden(c, "You can only delete your own comments")
    return
}
```

### Shaping a response by role

When a handler returns different fields by role rather than allowing or
denying, resolve the role once:

```go
role := authz.CallerRole(c, db, groupID)
if !authz.Allows(role, authz.ViewGroup) { /* 403 */ }
showPrivate := authz.Allows(role, authz.ViewMemberDetails)
```

### Cross-group endpoints

For endpoints that span groups (bulk edits, listings), get the set of groups
where the caller may act and filter by it:

```go
scope, err := authz.GroupsWhere(ctx, db, subject, authz.ManageAnimals)
if scope.Empty() { /* 403 */ }
if !scope.All {
    query = query.Where("group_id IN ?", scope.GroupIDs)
}
```

Check **every** group the request touches — including destinations (e.g. a
bulk "move to group X" must check X is in scope).

### Managing another user's account

Updating, resetting the password of, resending the invitation for,
unlocking, or deleting another user goes through `authz.CheckManageUser`
(via `callerCanManageUser` in handlers). Site admins may manage anyone; a
group admin may manage a non-site-admin who belongs to a group where the
caller has `ManageMembers`.

## Route-level tiers (middleware)

- **Public** — routes outside `AuthRequired`: login, password reset/setup,
  `GET /api/settings`, image/video serving, health checks. Don't add routes
  here unless they genuinely need no authentication.
- **Authenticated** — `AuthRequired(db)` sets `user_id` (uint) and
  `is_admin` (bool) from a JWT or API token. Group checks happen in the
  handler via `callerCan`.
- **Site admin only** — `/api/admin/**` uses `AdminRequired()`. Nothing
  extra is needed in the handler.

## Adding an action or a role

- **New action:** add the constant and its minimum role to `policy`, and a
  row to `TestPolicyMatrix`. Choose the narrowest name that describes the
  capability, not the role (`ManageSchedule`, not `GroupAdminOnly`).
- **New role** (coordinator, kiosk device, applicant, mentor — roadmap
  FD-2, TT-2, ON-1, ON-6): add it to `Role`, teach `GroupRole` to resolve
  it, and decide per action whether it qualifies — in the policy and the
  matrix test. If a role doesn't fit the linear ordering, change `Allows`
  to consult an explicit per-role set; handlers don't change either way.

## Critical rules

- ❌ **Never check roles inline** — no `c.Get("is_admin")`,
  `middleware.IsSiteAdmin(c)`, or `is_group_admin` queries in handlers. The
  `handler-inline-role-check` Semgrep rule flags them. (The one exception,
  `GetGroupMembership`, *reports* the flags to the client and is annotated.)
- ❌ **Never read identity from the request body or query** — an attacker
  can set any user ID they like.
- ✅ **Return 403 for authorization failures**, not 404.
- ✅ **Authorize before querying** group-scoped data; a `WHERE group_id = ?`
  alone is not access control.
- ✅ **Test the wiring** for new endpoints (member allowed, outsider denied,
  admin-only action denied to a member). Policy cells are already covered by
  `internal/authz` tests.

## Key files

- `internal/authz/authz.go` — roles, actions, policy, `CallerCan`,
  `CallerRole`, `GroupsWhere`, `CheckManageUser`
- `internal/authz/authz_test.go` — the policy matrix and resolution tests
- `internal/handlers/authz.go` — `callerCan`, `callerCanManageUser`
- `internal/middleware/middleware.go` — `AuthRequired()`, `AdminRequired()`,
  `GetUserID()`
- `internal/models/models.go` — `UserGroup` (`IsGroupAdmin`), `User` (`IsAdmin`)
- `tools/semgrep/go-handler-conventions.yaml` — `handler-inline-role-check`
