# Animal Video Upload

**Date:** 2026-06-02
**Status:** Approved

## Overview

Add video upload support for animals, integrated into the existing photo gallery. Videos are short clips (~30 seconds) sourced from standard iPhone or Android devices. They appear alongside images in the gallery as thumbnails that open a modal video player. Azure Blob Storage is required for video — no PostgreSQL fallback.

## Data Model

New `AnimalVideo` struct in `internal/models/models.go`, backed by a new `animal_videos` table:

```go
type AnimalVideo struct {
    ID              uint
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       gorm.DeletedAt
    AnimalID        *uint          // nullable, consistent with AnimalImage
    UserID          uint
    VideoURL        string         // Azure Blob public URL
    ThumbnailURL    string         // JPEG thumbnail blob URL
    MimeType        string         // video/mp4 or video/quicktime
    Caption         string
    DurationSeconds int
    FileSize        int64
    BlobIdentifier  string         // UUID for video blob
    ThumbnailBlobID string         // UUID for thumbnail blob
    BlobExtension   string         // .mp4 or .mov
}
```

**Indexes:**

- `idx_animal_video_animal` on `animal_id`

**Key constraint:** No `ImageData` / binary column. Videos are Azure-only. The backend returns a clear error if Azure is not configured when a video upload is attempted.

Thumbnail is stored as a separate Azure blob (JPEG), served from the same CDN path as images. Its UUID is tracked in `ThumbnailBlobID` so it can be cleaned up on video deletion.

## Backend

### New file: `internal/handlers/animal_video.go`

**Endpoints:**

| Method   | Path                                                      | Handler              |
|----------|-----------------------------------------------------------|----------------------|
| POST     | `/api/groups/:id/animals/:animalId/videos`                | `UploadAnimalVideo`  |
| DELETE   | `/api/groups/:id/animals/:animalId/videos/:videoId`       | `DeleteAnimalVideo`  |

**Upload flow (`UploadAnimalVideo`):**

1. Validate group membership and animal existence
2. Parse multipart form: `video` field (video file) + `thumbnail` field (JPEG) + optional `caption`
3. Validate video: MIME type whitelist, magic byte check, max 200MB
4. Validate thumbnail: existing image validation path (it's a JPEG)
5. Upload thumbnail blob to Azure → capture `ThumbnailURL` + `ThumbnailBlobID`
6. Upload video blob to Azure → capture `VideoURL` + `BlobIdentifier`
7. Insert `AnimalVideo` record
8. Return video metadata JSON

**Delete flow (`DeleteAnimalVideo`):**

1. Verify caller is video owner or site admin
2. Delete video blob from Azure
3. Delete thumbnail blob from Azure
4. Soft-delete `AnimalVideo` record

### New endpoint: `GET /api/groups/:id/animals/:animalId/media`

Returns a unified media response for the gallery:

```json
{
  "images": [ /* existing AnimalImage shape */ ],
  "videos": [ /* AnimalVideo shape */ ]
}
```

The existing `GET .../images` endpoint is unchanged.

### Validation additions: `internal/upload/validation.go`

- `MaxVideoSize`: 200MB (200 * 1024 * 1024 bytes)
- Allowed MIME types: `video/mp4`, `video/quicktime`
- Magic byte validation:
  - MP4: `ftyp` box at byte offset 4
  - MOV: QuickTime `ftyp` variants (`qt  `, `M4V `, etc.)

## Frontend

### `frontend/src/pages/PhotoGallery.tsx`

- Switch from `getImages` to `getMedia` endpoint
- Render images and videos in a unified grid
- Video items: show `ThumbnailURL` with a play button overlay
- Clicking a video thumbnail opens a modal containing an HTML5 `<video>` element with `controls` and the `VideoURL` as source

### New: `frontend/src/components/VideoUpload.tsx`

Handles the video upload flow (no crop/edit step):

1. File input — `accept="video/mp4,video/quicktime,.mp4,.mov"`
2. Client-side validation: MIME type and 200MB size limit
3. First-frame thumbnail extraction:
   - Load video into a hidden `<video>` element
   - Seek to frame 0, draw to `<canvas>`
   - Export canvas as JPEG blob
4. Read `videoElement.duration` and include as `duration_seconds` form field
5. Show progress indicator during upload (video files are large)
6. POST multipart form with `video`, `thumbnail`, `duration_seconds`, and optional `caption`

### `frontend/src/api/client.ts`

New `animalsApi` methods:

```ts
uploadVideo(groupId, animalId, videoFile, thumbnailBlob, caption): Promise<AnimalVideo>
deleteVideo(groupId, animalId, videoId): Promise<void>
getMedia(groupId, animalId): Promise<{ images: AnimalImage[], videos: AnimalVideo[] }>
```

### Upload entry point in gallery

The existing upload button detects file type on selection:

- `image/*` → existing `ImageEditor` crop/upload flow
- `video/*` → new `VideoUpload` flow

Single file input with `accept="image/*,video/mp4,video/quicktime,.mov"`.

## Error Handling

- Azure not configured: `500` with message "Video storage requires Azure Blob Storage to be configured"
- File too large: `413` with max size in message
- Unsupported format: `415` with accepted types in message
- Thumbnail extraction failure (client-side): show inline error, do not proceed with upload
- Azure upload partial failure (thumbnail succeeds, video fails): delete thumbnail blob, return error

## Testing

- Unit: video MIME type and magic byte validation
- Integration: full upload → Azure → DB record → delete cleanup (both blobs removed)
- Frontend: thumbnail extraction with mock video element, upload progress rendering, mixed gallery grid rendering, video modal open/close

## Out of Scope

- Setting a video as an animal's profile media (deferred)
- Server-side transcoding or compression
- PostgreSQL fallback for video storage
- Video duration auto-detection on the server
