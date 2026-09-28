---
name: group-auth-pattern
description: Background knowledge for implementing authorization in go-volunteer-media Gin handlers. Explains the four authorization tiers (public, group member, group admin, site admin), how to read identity from context, when to use AdminRequired middleware vs inline group checks, and common security anti-patterns to avoid. Automatically loaded when writing or reviewing handler code.
user-invocable: false
---

# Group Authorization Pattern

This app has four authorization tiers. Choose the right one for every handler.

> **Planned change:** the volunteer-system roadmap
> (`docs/VOLUNTEER_SYSTEM_ROADMAP.md`, item AR-2) replaces these inline
> checks with a central policy helper before new roles (coordinator, kiosk
> device, applicant) are added. Until that lands, use the helpers below and
> do not invent new role checks inline.

## Tier 1 — Public (no auth required)

Routes outside the `authRequired` middleware group. Only a handful:
- `POST /api/login`
- `POST /api/request-password-reset`, `/reset-password`, `/setup-password`
- `GET /api/settings`, `GET /api/images/:uuid`, `GET /api/videos/:uuid`
- Health check endpoints

**Do not** add new routes here unless they genuinely need no authentication.

## Tier 2 — Authenticated + Group Member

Routes inside the `authRequired` middleware group. The middleware sets two context keys:

```go
userID, _ := c.Get("user_id")    // type: uint
isAdmin, _ := c.Get("is_admin")  // type: bool
```

For group-scoped resources, **always** call `checkGroupAccess()` from `animal_helpers.go` before operating on data. Despite the filename, these helpers are used by all group-scoped handlers across the codebase — not just animal endpoints.

```go
func GetFoos(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        db := middleware.GetDB(c, db) // request-scoped DB; never reassign the closure's db
        groupID := c.Param("id")
        userID, _ := c.Get("user_id")
        isAdmin, _ := c.Get("is_admin")

        // This returns true for: site admins OR members of the group OR group admins
        if !checkGroupAccess(db, userID, isAdmin, groupID) {
            respondForbidden(c, "forbidden")  // 403, not 404 — do not leak resource existence
            return
        }
        // ... safe to query group data
    }
}
```

`checkGroupAccess()` approves the request if **any** of these is true:
- `isAdmin == true` (site admin)
- The user has a `UserGroup` record for this group (any role)

## Tier 3 — Group Admin or Site Admin

For write operations that only group admins should perform (create/update/delete animals, manage group members, etc.), use `checkGroupAdminAccess()` from `animal_helpers.go` — there is **no separate middleware** for this tier:

```go
// checkGroupAdminAccess returns true for site admins OR group admins of the specific group
if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
    respondForbidden(c, "forbidden")
    return
}
```

## Tier 4 — Site Admin Only

Route lives under `/api/admin/...` and uses `AdminRequired()` middleware. Nothing extra needed inside the handler — the middleware has already verified `is_admin == true`.

Routes in this tier: all `/admin/users`, `/admin/groups`, `/admin/announcements`, `/admin/settings`, `/admin/animals`, statistics, dashboard, seed-database.

## Critical Rules

### ✅ Always read identity from context

```go
// CORRECT
userID, _ := c.Get("user_id")
isAdmin, _ := c.Get("is_admin")
```

### ❌ Never read identity from request body or query params

```go
// WRONG — an attacker can elevate their own privileges
var body struct { UserID uint `json:"user_id"` }
c.ShouldBindJSON(&body)
```

### ✅ Return 403 for authorization failures

Use `respondForbidden(c, "forbidden")` — not 404. Do not reveal whether the resource exists.

### ✅ Check group membership before querying

Always validate group access before running any GORM query on group-scoped data. Don't rely on GORM's WHERE clause alone to enforce access control.

### ✅ Type-assert context values safely

`checkGroupAccess()` and `checkGroupAdminAccess()` accept `interface{}` for `userID` and `isAdmin`, so you can pass them directly from `c.Get()`. However, if you need to pass `userID` to a model-layer function that requires a typed `uint` (e.g., `models.IsGroupAdmin`), assert it explicitly:

```go
// Pass the raw c.Get values directly to checkGroupAccess — it accepts interface{}.
// For code that needs a typed uint, use the middleware helper:
uid, ok := middleware.GetUserID(c)
if !ok {
    respondUnauthorized(c, "unauthorized")
    return
}
isSiteAdmin := middleware.IsSiteAdmin(c)
```

## Auth Flow Summary

```
Request
  │
  ├─ AuthRequired() middleware  →  sets user_id + is_admin in context
  │
  ├─ handler reads user_id/is_admin from c.Get()
  │
  ├─ checkGroupAccess(db, userID, isAdmin, groupID)  →  403 if not a member
  │
  └─ (write ops) checkGroupAdminAccess(db, uid, isAdmin, groupID)  →  403 if not admin
```

## Key Files

- `internal/middleware/middleware.go` — `AuthRequired()`, `AdminRequired()`, `GetUserID()`, `IsSiteAdmin()`
- `internal/handlers/animal_helpers.go` — `checkGroupAccess()`, `checkGroupAdminAccess()`
- `internal/models/models.go` — `UserGroup` struct (has `IsGroupAdmin bool`)
- `cmd/api/main.go` — route groups showing which middleware applies to which routes
