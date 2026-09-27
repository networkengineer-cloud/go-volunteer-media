---
name: add-api-endpoint
description: Add a new API endpoint to the go-volunteer-media app. Covers the complete full-stack workflow: defining or updating a GORM model, adding the migration, writing the Gin handler, registering the route in main.go, adding a typed API method to the frontend client, and writing tests. Use when adding any new backend functionality.
argument-hint: [feature name or description]
---

# Add a New API Endpoint

Follow these steps in order. Each step maps to a specific file.

## Step 1 — Define or update the model (`internal/models/models.go`)

Add a new struct or update an existing one. Models in this codebase declare
their base fields explicitly — none embed `gorm.Model`, because it has no
`json` tags and would serialise as `ID`/`CreatedAt`. Copy this shape:

```go
type Foo struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
    GroupID     uint           `gorm:"not null;index" json:"group_id"`
    Name        string         `gorm:"not null" json:"name"`
    Description string         `json:"description"`
}
```

Rules:
- Use `gorm:"not null"` for required fields
- Foreign keys must have a matching `gorm:"index"` or `gorm:"uniqueIndex"`
- Include `DeletedAt gorm.DeletedAt` for soft delete; omit it only for
  append-only or join tables (e.g. `ShiftSlot`, `CommentHistory`)
- JSON field names use `snake_case`; secrets and internal bookkeeping get
  `json:"-"`
- Adding a column is safe. Renaming or dropping one is **not** handled by
  AutoMigrate — it needs explicit SQL in `database.go` and review

## Step 2 — Register the migration (`internal/database/database.go`)

Add the new model to the `db.AutoMigrate(...)` call inside `RunMigrations`:

```go
err := db.AutoMigrate(
    // ... existing models ...
    &models.Foo{},
)
```

If the feature needs default seed data, add a `createDefaultFoos(db)` call and implement the function following the existing `createDefaultGroups` / `createDefaultAnimalTags` pattern.

## Step 3 — Write the Gin handler (`internal/handlers/foo.go`)

Create a new file `internal/handlers/foo.go`. Each handler is a closure that receives `*gorm.DB` and returns `gin.HandlerFunc`:

```go
package handlers

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
    "github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// GetFoos returns all Foos for a group. Requires group membership (or site admin).
func GetFoos(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        db := middleware.GetDB(c, db)
        groupID := c.Param("id")

        userID, _ := c.Get("user_id")
        isAdmin, _ := c.Get("is_admin")
        if !checkGroupAccess(db, userID, isAdmin, groupID) {
            respondForbidden(c, "forbidden")
            return
        }

        var foos []models.Foo
        if err := db.Where("group_id = ?", groupID).Find(&foos).Error; err != nil {
            middleware.GetLogger(c).Error("Failed to fetch foos", err)
            respondInternalError(c, "Failed to fetch foos")
            return
        }
        respondOK(c, foos)
    }
}
```

Key rules for handlers:
- **First line is always `db := middleware.GetDB(c, db)`.** It returns the
  request-scoped DB (bound by `DBMiddleware`) with the request context
  already applied, so no separate `WithContext` call is needed. Shadow with
  `:=`; never assign to the closure's `db` with `=` — that mutates state
  shared by every request (150+ call sites follow this)
- Read identity from context, never from the request body:
  `middleware.GetUserID(c)` returns a typed `(uint, bool)`;
  `middleware.IsSiteAdmin(c)` returns a `bool`. The `checkGroup*Access`
  helpers take the raw `c.Get("user_id")` / `c.Get("is_admin")` values
- Call `checkGroupAccess()` (in `animal_helpers.go`) before any group-scoped query — despite the filename, these helpers are used by all group-scoped handlers, not just animal handlers
- Return a generic message to the client and log the real error with
  `middleware.GetLogger(c).Error(msg, err)` — don't send `err.Error()`,
  which can leak SQL and internals
- Admin-only operations go in a separate handler or file (e.g., `foo_admin.go`)
- Use the response helpers in `respond.go`: `respondOK(c, data)`, `respondCreated(c, data)`, `respondNoContent(c)`, and `respondBadRequest` / `respondUnauthorized` / `respondForbidden` / `respondNotFound` / `respondInternalError`, which all take `(c, msg)`
- Return early on every error path

See [handler-template.go](./handler-template.go) for a complete CRUD example.

## Step 4 — Register the route (`cmd/api/main.go`)

Find the appropriate router group and add the route. Protected routes go inside the `authRequired` group; admin routes go inside the `adminGroup`:

```go
// Under the authRequired group, inside a group-scoped block:
groupRoutes.GET("/foos", handlers.GetFoos(db))
groupRoutes.GET("/foos/:fooId", handlers.GetFooByID(db))
groupRoutes.POST("/foos", handlers.CreateFoo(db))
groupRoutes.PUT("/foos/:fooId", handlers.UpdateFoo(db))
groupRoutes.DELETE("/foos/:fooId", handlers.DeleteFoo(db))
```

Route parameter naming convention: `:id` for group ID, `:animalId`/`:fooId` for nested entity IDs.

## Step 5 — Add a typed frontend API method (`frontend/src/api/client.ts`)

Add a typed section to the centralized API client. Never call `axios` directly in components:

```typescript
// --- Foo API ---
export interface Foo {
  id: number;
  group_id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export type FooInput = Pick<Foo, 'name' | 'description'>;

export const fooApi = {
  getAll: (groupId: number) =>
    api.get<Foo[]>(`/groups/${groupId}/foos`),
  getById: (groupId: number, id: number) =>
    api.get<Foo>(`/groups/${groupId}/foos/${id}`),
  create: (groupId: number, data: FooInput) =>
    api.post<Foo>(`/groups/${groupId}/foos`, data),
  update: (groupId: number, id: number, data: Partial<FooInput>) =>
    api.put<Foo>(`/groups/${groupId}/foos/${id}`, data),
  delete: (groupId: number, id: number) =>
    api.delete(`/groups/${groupId}/foos/${id}`),
};
```

TypeScript interface rules:
- Field names exactly match the Go model's JSON tags (`snake_case`)
- `id`, `created_at`, `updated_at` are always present (from `gorm.Model`)
- Use `number` for Go `uint`/`int`, `string` for Go `string`, `boolean` for Go `bool`

See [client-template.ts](./client-template.ts) for a complete typed example.

## Step 6 — Write tests

**Backend** — create `internal/handlers/foo_test.go`. Follow the pattern in any existing `*_test.go` in that directory. Use `test_helpers.go` shared utilities.

If the behaviour depends on Postgres (JSONB, full-text, pgvector, row
locking, or two requests racing), put those tests in
`foo_postgres_test.go` using `openSearchTestPostgres(t)` — SQLite will not
catch those bugs. See the `run-tests` skill for running them locally.

**Frontend E2E** — if the feature has a user-facing UI, add a Playwright spec in `frontend/tests/foo.spec.ts`. See the `playwright-e2e-test` skill for the test authoring workflow.

Also update `API.md` with the new endpoint(s) — this is required before the PR can merge.

## Checklist

- [ ] Model added/updated in `models.go`
- [ ] Model added to `AutoMigrate` in `database.go`
- [ ] Handler file created in `internal/handlers/`
- [ ] Route registered in `cmd/api/main.go`
- [ ] TypeScript interface + API method added in `frontend/src/api/client.ts`
- [ ] Handler starts with `db := middleware.GetDB(c, db)`
- [ ] Backend test file added (plus `_postgres_test.go` if Postgres-specific)
- [ ] E2E test added (if UI is involved)
- [ ] `API.md` updated with new endpoint(s)
