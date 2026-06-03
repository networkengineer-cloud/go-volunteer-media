# API Documentation

All endpoints are prefixed with `/api` and require a valid JWT in the `Authorization: Bearer <token>` header unless noted.

---

## Animal Media

### Get Animal Media

```
GET /api/groups/:id/animals/:animalId/media
```

Returns all images and videos for an animal. Requires group membership.

**Response `200 OK`**
```json
{
  "images": [{ "id": 1, "image_url": "...", "caption": "", "user": { "id": 1, "username": "..." }, ... }],
  "videos": [{ "id": 1, "video_url": "...", "thumbnail_url": "...", "mime_type": "video/mp4", "user": { "id": 1, "username": "..." }, ... }]
}
```

**Errors:** `403` not a group member · `404` animal not found

---

### Upload Animal Video

```
POST /api/groups/:id/animals/:animalId/videos
Content-Type: multipart/form-data
```

Uploads a video and its thumbnail to Azure Blob Storage. Requires group membership and Azure storage configuration.

**Form fields**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `video` | file | yes | MP4 or MOV, max 200 MB |
| `thumbnail` | file | yes | JPEG/PNG image, client-generated |
| `caption` | string | no | Max 500 characters |
| `duration_seconds` | integer | no | 0–3600 |

**Response `201 Created`** — the created `AnimalVideo` object with `user` preloaded.

**Errors:** `400` missing/invalid field · `403` access denied · `404` animal not found · `413` video too large · `415` unsupported format · `503` Azure not configured

---

### Delete Animal Video

```
DELETE /api/groups/:id/animals/:animalId/videos/:videoId
```

Permanently deletes a video and its Azure blobs. Only the uploader or a site admin may delete.

**Response `200 OK`**
```json
{ "message": "Video deleted successfully" }
```

**Errors:** `403` not owner or admin · `404` video or animal not found
