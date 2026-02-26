---
name: add-api-endpoint
description: Add a new API endpoint to the go-volunteer-media app. Covers the complete full-stack workflow: defining or updating a GORM model, adding the migration, writing the Gin handler, registering the route in main.go, adding a typed API method to the frontend client, and writing tests. Use when adding any new backend functionality.
argument-hint: [feature name or description]
---

# Add a New API Endpoint

Follow these steps in order. Each step maps to a specific file.

## Step 1 — Define or update the model (`internal/models/models.go`)

Add a new struct or update an existing one. All models must use `gorm.Model` (or an explicit `ID`/`CreatedAt`/`UpdatedAt`/`DeletedAt`) and `json` tags:

```go
type Foo struct {
    gorm.Model
    GroupID     uint   `json:"group_id" gorm:"not null;index"`
    Name        string `json:"name" gorm:"not null"`
    Description string `json:"description"`
}
```

Rules:
- Use `gorm:"not null"` for required fields
- Foreign keys must have a matching `gorm:"index"` or `gorm:"uniqueIndex"`
- Soft-deletes are provided by `gorm.Model`; do not add a manual deleted_at field
- JSON field names use `snake_case`

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

    "github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// GetFoos returns all Foos for a group.
func GetFoos(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        groupID := c.Param("id")
        userID, _ := c.Get("user_id")
        isAdmin, _ := c.Get("is_admin")
        isAdminBool, _ := isAdmin.(bool)

        if !checkGroupAccess(db, userID, isAdminBool, groupID) {
            respondForbidden(c, "forbidden")
            return
        }

        var foos []models.Foo
        if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&foos).Error; err != nil {
            respondInternalError(c, err.Error())
            return
        }
        respondOK(c, foos)
    }
}
```

Key rules for handlers:
- Always use `db.WithContext(ctx)` — pass the request context to GORM
- Read identity from `c.Get("user_id")` and `c.Get("is_admin")` — never from the request body
- Call `checkGroupAccess()` (in `animal_helpers.go`) before any group-scoped query
- Admin-only operations go in a separate handler or file (e.g., `foo_admin.go`)
- Use the response helpers: `respondOK`, `respondCreated`, `respondForbidden(c, msg)`, `respondNotFound(c, msg)`, `respondBadRequest`, `respondInternalError(c, msg)`, `respondNoContent` (all in `respond.go`)
- Return early on every error path

See [handler-template.go](./handler-template.go) for a complete CRUD example.

## Step 4 — Register the route (`cmd/api/main.go`)

Find the appropriate router group and add the route. Protected routes go inside the `authRequired` group; admin routes go inside the `adminGroup`:

```go
// Under the authRequired group, inside a group-scoped block:
groupRoutes.GET("/foos", handlers.GetFoos(db))
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

export const fooApi = {
  getAll: (groupId: number) =>
    api.get<Foo[]>(`/groups/${groupId}/foos`),
  getById: (groupId: number, id: number) =>
    api.get<Foo>(`/groups/${groupId}/foos/${id}`),
  create: (groupId: number, data: Partial<Foo>) =>
    api.post<Foo>(`/groups/${groupId}/foos`, data),
  update: (groupId: number, id: number, data: Partial<Foo>) =>
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

**Frontend E2E** — if the feature has a user-facing UI, add a Playwright spec in `frontend/tests/foo.spec.ts`. See the `playwright-e2e-test` skill for the test authoring workflow.

## Checklist

- [ ] Model added/updated in `models.go`
- [ ] Model added to `AutoMigrate` in `database.go`
- [ ] Handler file created in `internal/handlers/`
- [ ] Route registered in `cmd/api/main.go`
- [ ] TypeScript interface + API method added in `frontend/src/api/client.ts`
- [ ] Backend test file added
- [ ] E2E test added (if UI is involved)
- [ ] `API.md` updated with new endpoint(s)
