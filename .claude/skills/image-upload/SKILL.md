---
name: image-upload
description: Handle image and document uploads in go-volunteer-media. Explains the storage.Provider abstraction (Postgres bytea by default, Azure Blob Storage in deployed environments), where the bytes actually live for each provider, the upload/serve/delete handler patterns, and the upload validators. Use when adding any file upload or media-management feature.
user-invocable: false
---

# Image and Document Upload Pattern

**Never write files to the local filesystem.** All uploads go through the
`storage.Provider` interface (`internal/storage/storage.go`) so they work with
either backend.

## The two providers

Selected at startup by `STORAGE_PROVIDER` (`storage.LoadConfig`) and injected
into handlers as `storageProvider storage.Provider`.

| Provider | Used by | Where the bytes live |
|---|---|---|
| `postgres` (`storage.ProviderPostgres`, the **default**) | Local dev (`.env.example`) | **In the model's own row** — a `bytea` column such as `ImageData`, `FileData`, `ProtocolDocumentData`. The provider stores nothing: `UploadImage`/`UploadDocument` only mint a UUID + URL, and `Delete*` are no-ops. |
| `azure` (`storage.ProviderAzure`) | dev/prod via Terraform | In Azure Blob Storage, keyed by `<uuid><ext>`. The row stores the identifier; the `bytea` column stays `nil`. |

**The consequence:** after calling the provider, the handler must decide what
to persist. If `storageProvider.Name() == storage.ProviderPostgres`, write the
bytes into the row; otherwise store `nil` and the blob identifier. Forgetting
this silently loses the file in local dev.

## Reference handler

`internal/handlers/group_document.go` (`UploadGroupDocument`) is the pattern
to copy — it handles both providers and the fallback correctly:

```go
_, blobUUID, blobExt, uploadErr := storageProvider.UploadDocument(ctx, fileData, mimeType, file.Filename)
var fileURL, blobIdentifier, fileProvider string
var fileDataForDB []byte

if uploadErr != nil {
    // Provider failed: fall back to storing the bytes in Postgres.
    logger.WithFields(map[string]interface{}{"error": uploadErr.Error()}).
        Warn("Failed to upload document to storage provider, falling back to PostgreSQL")
    fileURL = fmt.Sprintf("/api/group-documents/%s", docUUID)
    blobIdentifier = docUUID
    fileProvider = storage.ProviderPostgres
    fileDataForDB = fileData
} else {
    blobIdentifier = blobUUID + blobExt
    fileURL = fmt.Sprintf("/api/group-documents/%s", blobIdentifier)
    fileProvider = storageProvider.Name()
    if fileProvider == storage.ProviderPostgres {
        fileDataForDB = fileData // postgres provider stores nothing itself
    } else {
        fileDataForDB = nil
    }
}

doc := models.GroupDocument{
    // ...
    FileURL:            fileURL,
    FileProvider:       fileProvider,
    FileBlobIdentifier: blobIdentifier,
    FileBlobExtension:  blobExt,
    FileData:           fileDataForDB,
}
```

A new model that stores a file needs the same column set:
`<X>Provider`, `<X>BlobIdentifier`, `<X>BlobExtension`, and
`<X>Data []byte \`gorm:"type:bytea" json:"-"\``. Never expose the bytes in
JSON.

> **Known inconsistency:** `animal_image.go` (`UploadAnimalImage`) sets
> `ImageData = nil` whenever the provider call succeeds — including with the
> `postgres` provider, which never fails — so gallery images uploaded under
> `STORAGE_PROVIDER=postgres` are served as 404 by `ServeImage`. Don't copy
> that branch; follow `group_document.go`.

## Serving files

Files are always served through an API route that looks up the row, checks
the provider, and returns either the row's bytes or the blob:

| Route | Handler | Auth |
|---|---|---|
| `GET /api/images/:uuid` | `ServeImage` (`animal_upload.go`) | Public |
| `GET /api/videos/:uuid` | `ServeVideo` (`animal_upload.go`) | Public |
| `GET /api/documents/:uuid` | `ServeAnimalProtocolDocument` (`animal_document.go`) | Authenticated |
| `GET /api/group-documents/:uuid` | `ServeGroupDocument` (`group_document.go`) | Authenticated + group check |

Anything private (waivers, incident attachments, background-check
documents) must be served through an **authenticated** route with an access
check, like `ServeGroupDocument` — never through the public image route, and
never by handing out a direct blob URL. `group_document.go` discards the
provider's URL for exactly this reason.

In the frontend, reference files by the URL stored on the record
(`/api/images/<uuid>` etc.), never a filesystem path.

## Deleting files

Look up the row scoped to the authorized group, delete the blob only when the
provider is not Postgres, then delete the row:

```go
if doc.FileProvider != storage.ProviderPostgres && doc.FileBlobIdentifier != "" {
    if err := storageProvider.DeleteDocument(ctx, doc.FileBlobIdentifier); err != nil {
        // log; decide whether to continue
    }
}
if err := db.Delete(&doc).Error; err != nil { ... }
```

For Postgres-stored files, deleting (or soft-deleting) the row is the delete.

## Validation (`internal/upload/validation.go`)

Always use the shared validators — never hand-roll MIME or size checks:

| Function | Accepts | Size constant |
|---|---|---|
| `upload.ValidateImageUpload(file, max)` | jpg/jpeg, png, gif, webp, heic/heif | `MaxImageSize` (10 MB), `MaxHeroImageSize` (5 MB) |
| `upload.ValidateImageContent(data)` | Checks decoded bytes really are an image | — |
| `upload.ValidateDocumentUpload(file, max)` | pdf, docx, xlsx | `MaxDocumentSize` (20 MB) |
| `upload.ValidateVideoUpload(file, max)` | Video types | `MaxVideoSize` (200 MB) |

Also useful: `upload.SanitizeFilename`, `upload.MimeTypeFromFilename`.
Routes accepting large bodies raise the per-route body limit in
`cmd/api/main.go` (see the document routes).

## Existing upload handlers

| File | What it handles |
|---|---|
| `internal/handlers/group_document.go` | Group documents — **reference pattern** |
| `internal/handlers/script.go` | Script files (same column set as group documents) |
| `internal/handlers/animal_document.go` | Per-animal protocol documents |
| `internal/handlers/animal_image.go` | Gallery images: resize, upload, delete, profile picture (see inconsistency above) |
| `internal/handlers/animal_video.go` | Videos + thumbnails (Azure only) |
| `internal/handlers/animal_upload.go` | `ServeImage` / `ServeVideo` |
| `internal/handlers/settings.go` | Site hero image |

## Key files

- `internal/storage/storage.go` — `Provider` interface, `LoadConfig`, `NewProvider`
- `internal/storage/postgres.go`, `azure.go` — the two implementations
- `internal/upload/validation.go` — validators and size limits
- `cmd/api/main.go` — provider construction and the serve routes
