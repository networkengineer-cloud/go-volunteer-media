---
name: image-upload
description: Handle image and document uploads in go-volunteer-media. Explains the storageProvider abstraction that supports both local filesystem and Azure Blob Storage, the correct handler patterns for gallery images, profile pictures, and protocol documents, and how to read/delete stored files. Use when adding any file upload or media-management feature.
user-invokable: false
---

# Image and Document Upload Pattern

**Never write files directly to the filesystem.** All uploads must go through the `storageProvider` interface so they work in both local dev (filesystem) and production (Azure Blob Storage).

## The Storage Abstraction

The storage interface is defined in `internal/storage/`. Two implementations exist:
- **Local**: writes to `public/uploads/` and serves via `/uploads` static route
- **Azure Blob**: stores in Azure Blob Storage; images are served via `GET /api/images/:uuid`

The active provider is injected into handlers at startup. Handlers receive it as a parameter.

## Handler Pattern — Image Upload

```go
func UploadFooImage(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        groupID := c.Param("id")
        userID, _ := c.Get("user_id")
        isAdmin, _ := c.Get("is_admin")
        isAdminBool, _ := isAdmin.(bool)

        // 1. Auth check
        if !checkGroupAccess(db, userID, isAdminBool, groupID) {
            respondForbidden(c, "forbidden")
            return
        }

        // 1b. Parse the entity ID
        parsedAnimalID, err := strconv.ParseUint(c.Param("animalId"), 10, 64)
        if err != nil {
            respondBadRequest(c, "invalid animal id")
            return
        }

        // 2. Parse the multipart file header
        file, err := c.FormFile("image")
        if err != nil {
            respondBadRequest(c, "no file uploaded")
            return
        }

        // 3. Validate MIME type and size
        if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil {
            respondBadRequest(c, err.Error())
            return
        }

        // 4. Open and read file bytes
        src, err := file.Open()
        if err != nil {
            respondInternalError(c, err.Error())
            return
        }
        defer src.Close()
        data, err := io.ReadAll(src)
        if err != nil {
            respondInternalError(c, err.Error())
            return
        }

        // 5. Upload via the provider
        // UploadImage returns (publicURL, identifier, extension, error)
        imageURL, identifier, _, err := storageProvider.UploadImage(ctx, data, file.Header.Get("Content-Type"), map[string]string{})
        if err != nil {
            respondInternalError(c, err.Error())
            return
        }

        // 6. Persist the reference in the DB
        img := models.AnimalImage{
            AnimalID:       uint(parsedAnimalID),
            ImageURL:       imageURL,
            BlobIdentifier: identifier,
        }
        if err := db.WithContext(ctx).Create(&img).Error; err != nil {
            respondInternalError(c, err.Error())
            return
        }
        respondCreated(c, img)
    }
}
```

## Handler Pattern — Reading an Image

Images stored by the provider are served through the UUID endpoint, not by exposing filesystem paths:

```
GET /api/images/:uuid  →  handlers in cmd/api/main.go, served directly from storage provider
```

In the frontend, always reference images via `/api/images/<uuid>`, not with `/uploads/<filename>`.

## Handler Pattern — Deleting an Image

```go
// Retrieve the record first, then delete from storage, then delete the DB row.
// Always scope the lookup to the authorized group to prevent cross-group deletes.
// Trade-off: if DeleteImage succeeds but db.Delete fails, the blob is gone but the
// DB row remains pointing to a missing file. This is acceptable because the next
// delete attempt will simply return a storage 404. For strict atomicity, soft-delete
// the DB row first, then clean up storage asynchronously.
var img models.AnimalImage
if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", id, groupID).First(&img).Error; err != nil {
    respondNotFound(c, "not found")
    return
}
if err := storageProvider.DeleteImage(ctx, img.BlobIdentifier); err != nil {
    respondInternalError(c, err.Error())
    return
}
if err := db.WithContext(ctx).Delete(&img).Error; err != nil {
    respondInternalError(c, err.Error())
    return
}
```

## File Validation (use `internal/upload/`)

The `upload` package provides shared validation:

```go
import "github.com/networkengineer-cloud/go-volunteer-media/internal/upload"

// Images: validates MIME (image/jpeg, image/png, image/gif) and size (max 10MB)
if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil { ... }

// Documents: validates MIME (application/pdf) and size (max 20MB)
if err := upload.ValidateDocumentUpload(file, upload.MaxDocumentSize); err != nil { ... }
```

Never write your own MIME or size validation — always use `upload.ValidateImageUpload` / `upload.ValidateDocumentUpload`.

## Existing Upload Handlers (Reference)

| File | What it handles |
|---|---|
| `internal/handlers/animal_image.go` | Gallery images: upload, delete, set profile picture |
| `internal/handlers/animal_document.go` | Protocol documents: upload, serve, delete |
| `internal/handlers/animal_upload.go` | Simple single image upload (not gallery) |
| `internal/handlers/settings.go` | Hero image upload for site settings |
| `internal/handlers/group.go` | Group avatar/image upload |

Read these existing handlers before writing a new one — they show the exact pattern used in production.

## Key Files

- `internal/storage/` — `Provider` interface + local/Azure implementations
- `internal/upload/` — `ValidateImageUpload`, `ValidateDocumentUpload` helpers
- `internal/models/models.go` — `AnimalImage`, `AnimalDocument` model structs
- `cmd/api/main.go` — where `storageProvider` is initialized and injected
