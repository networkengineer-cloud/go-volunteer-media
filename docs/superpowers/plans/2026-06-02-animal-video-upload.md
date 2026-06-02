# Animal Video Upload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add video upload support to the animal photo gallery, storing videos in Azure Blob Storage and displaying them as thumbnails with a modal player alongside existing images.

**Architecture:** A new `AnimalVideo` model backs a dedicated `animal_videos` table; a new `GetAnimalMedia` endpoint returns both images and videos in a single response. The frontend `VideoUpload` component extracts a first-frame JPEG thumbnail client-side before uploading both blobs via a new multipart endpoint. `PhotoGallery` is updated to consume the media endpoint and render video thumbnails with a play overlay that opens a modal player.

**Tech Stack:** Go 1.21, Gin, GORM (SQLite in tests, Postgres in production), Azure Blob Storage, React 18, TypeScript, Axios, Vitest

---

## File Map

| Action | Path | Responsibility |
| --- | --- | --- |
| Modify | `internal/upload/validation.go` | Add `MaxVideoSize`, `AllowedVideoTypes`, `ValidateVideoUpload` |
| Create | `internal/upload/validation_video_test.go` | Unit tests for video validation |
| Modify | `internal/models/models.go` | Add `AnimalVideo` struct |
| Modify | `internal/database/database.go` | Add `AnimalVideo` to `AutoMigrate` |
| Modify | `internal/handlers/test_helpers.go` | Add `ProviderName`/`DeletedBlobs` to mock; add `createVideoMultipartRequest` |
| Create | `internal/handlers/animal_video.go` | `GetAnimalMedia`, `UploadAnimalVideo`, `DeleteAnimalVideo` handlers |
| Create | `internal/handlers/animal_video_test.go` | Integration tests for all three handlers |
| Modify | `cmd/api/main.go` | Register three new routes |
| Modify | `frontend/src/api/client.ts` | Add `AnimalVideo`, `AnimalMedia` types; `getMedia`, `uploadVideo`, `deleteVideo` |
| Create | `frontend/src/components/VideoUpload.tsx` | Client-side thumbnail extraction and upload UI |
| Modify | `frontend/src/pages/PhotoGallery.tsx` | Unified media grid, video modal, video delete |

---

## Task 1: Video validation

**Files:**
- Modify: `internal/upload/validation.go`
- Create: `internal/upload/validation_video_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/upload/validation_video_test.go`:

```go
package upload

import (
	"bytes"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
)

func TestValidateVideoUpload(t *testing.T) {
	// MP4: 4-byte size, "ftyp" at bytes 4-7, brand "isom" at 8-11
	minimalMP4 := make([]byte, 20)
	copy(minimalMP4[4:8], []byte("ftyp"))
	copy(minimalMP4[8:12], []byte("isom"))

	// MOV: "ftyp" at bytes 4-7, brand "qt  " at 8-11
	minimalMOV := make([]byte, 20)
	copy(minimalMOV[4:8], []byte("ftyp"))
	copy(minimalMOV[8:12], []byte("qt  "))

	tests := []struct {
		name        string
		fileSize    int64
		filename    string
		content     []byte
		wantErr     error
		errContains string
	}{
		{
			name:     "valid MP4",
			fileSize: 10 * 1024 * 1024,
			filename: "clip.mp4",
			content:  minimalMP4,
		},
		{
			name:     "valid MOV",
			fileSize: 10 * 1024 * 1024,
			filename: "clip.mov",
			content:  minimalMOV,
		},
		{
			name:        "file too large",
			fileSize:    201 * 1024 * 1024,
			filename:    "big.mp4",
			content:     minimalMP4,
			wantErr:     ErrFileTooLarge,
			errContains: "file size exceeds maximum limit",
		},
		{
			name:        "unsupported extension",
			fileSize:    1024,
			filename:    "clip.avi",
			content:     minimalMP4,
			wantErr:     ErrInvalidFileType,
			errContains: "extension .avi is not allowed",
		},
		{
			name:        "wrong magic bytes",
			fileSize:    1024,
			filename:    "clip.mp4",
			content:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x00, 0x00, 0x00}, // JPEG bytes
			wantErr:     ErrInvalidFileType,
			errContains: "does not appear to be a valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("video", tt.filename)
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			if _, err := part.Write(tt.content); err != nil {
				t.Fatalf("failed to write content: %v", err)
			}
			writer.Close()

			reader := multipart.NewReader(body, writer.Boundary())
			form, err := reader.ReadForm(32 << 20)
			if err != nil {
				t.Fatalf("failed to read form: %v", err)
			}
			defer form.RemoveAll()

			files := form.File["video"]
			if len(files) == 0 {
				t.Fatal("no files in form")
			}
			fh := files[0]
			fh.Size = tt.fileSize

			err = ValidateVideoUpload(fh, MaxVideoSize)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got %v, want %v", err, tt.wantErr)
				}
				if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
					t.Errorf("error %q does not contain %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/upload/... -run TestValidateVideoUpload -v
```

Expected: FAIL — `ValidateVideoUpload` undefined.

- [ ] **Step 3: Add validation constants and functions to `internal/upload/validation.go`**

Add after the existing `MaxDocumentSize` constant block:

```go
// MaxVideoSize is the maximum allowed video upload size (200MB)
MaxVideoSize = 200 * 1024 * 1024 // 200 MB
```

Add after the existing `AllowedDocumentTypes` var:

```go
// AllowedVideoTypes maps file extensions to their MIME types for video uploads
var AllowedVideoTypes = map[string][]string{
	".mp4": {"video/mp4"},
	".mov": {"video/quicktime"},
}
```

Add at the end of the file (before the final newline):

```go
// ValidateVideoUpload validates an uploaded video file (size, extension, magic bytes).
// Videos must be MP4 or MOV and fit within maxSize bytes.
func ValidateVideoUpload(file *multipart.FileHeader, maxSize int64) error {
	if file.Size > maxSize {
		return fmt.Errorf("%w: file size is %d bytes, maximum is %d bytes",
			ErrFileTooLarge, file.Size, maxSize)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if _, ok := AllowedVideoTypes[ext]; !ok {
		return fmt.Errorf("%w: extension %s is not allowed (only .mp4 and .mov are supported)",
			ErrInvalidFileType, ext)
	}

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	buf := make([]byte, 12)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if !isVideoContent(buf[:n]) {
		return fmt.Errorf("%w: file does not appear to be a valid MP4 or MOV video",
			ErrInvalidFileType)
	}

	return nil
}

// isVideoContent checks the ISO Base Media File Format box type at bytes 4-7.
// Both MP4 and MOV are built on this format.
func isVideoContent(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	boxType := data[4:8]
	for _, valid := range [][]byte{
		[]byte("ftyp"), []byte("moov"), []byte("wide"), []byte("mdat"), []byte("free"),
	} {
		if bytes.Equal(boxType, valid) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/upload/... -run TestValidateVideoUpload -v
```

Expected: all 5 subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/upload/validation.go internal/upload/validation_video_test.go
git commit -m "feat: add video upload validation (MP4/MOV, 200MB limit)"
```

---

## Task 2: `AnimalVideo` model and migration

**Files:**
- Modify: `internal/models/models.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Add `AnimalVideo` struct to `internal/models/models.go`**

Add after the `AnimalImage` struct (around line 391):

```go
// AnimalVideo represents a video uploaded for an animal
type AnimalVideo struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	AnimalID        *uint          `gorm:"index:idx_animal_video_animal" json:"animal_id"`
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	VideoURL        string         `gorm:"not null" json:"video_url"`
	ThumbnailURL    string         `gorm:"not null" json:"thumbnail_url"`
	MimeType        string         `gorm:"default:'video/mp4'" json:"mime_type"`
	Caption         string         `json:"caption"`
	DurationSeconds int            `json:"duration_seconds"`
	FileSize        int64          `json:"file_size"`
	BlobIdentifier  string         `json:"-"` // UUID+ext of video blob in Azure
	ThumbnailBlobID string         `json:"-"` // UUID+ext of thumbnail blob in Azure
	BlobExtension   string         `json:"-"` // e.g. ".mp4"
	User            User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

- [ ] **Step 2: Add `AnimalVideo` to `AutoMigrate` in `internal/database/database.go`**

In the `AutoMigrate(...)` call (around line 150), add `&models.AnimalVideo{}` after `&models.AnimalImage{}`:

```go
&models.AnimalImage{},
&models.AnimalVideo{},
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go build ./...
```

Expected: exits 0 with no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/models/models.go internal/database/database.go
git commit -m "feat: add AnimalVideo model and AutoMigrate entry"
```

---

## Task 3: Update test helpers for video tests

**Files:**
- Modify: `internal/handlers/test_helpers.go`

- [ ] **Step 1: Extend `mockStorageProvider` with `ProviderName` and `DeletedBlobs` tracking**

In `internal/handlers/test_helpers.go`, replace the `mockStorageProvider` struct and its `Name()` / `DeleteImage()` methods with:

```go
// mockStorageProvider is a test double for storage.Provider.
// Set ProviderName to control what Name() returns (default: "mock").
// Set UploadImageErr or UploadDocumentErr to simulate failures.
// DeletedBlobs records every identifier passed to DeleteImage.
type mockStorageProvider struct {
	ProviderName      string
	UploadImageErr    error
	UploadDocumentErr error
	LastMimeType      string
	DeletedBlobs      []string
}

func (m *mockStorageProvider) Name() string {
	if m.ProviderName != "" {
		return m.ProviderName
	}
	return "mock"
}
func (m *mockStorageProvider) UploadImage(_ context.Context, _ []byte, mimeType string, _ map[string]string) (string, string, string, error) {
	m.LastMimeType = mimeType
	if m.UploadImageErr != nil {
		return "", "", "", m.UploadImageErr
	}
	return "/api/images/test-uuid", "test-uuid", ".png", nil
}
func (m *mockStorageProvider) UploadDocument(_ context.Context, _ []byte, _, _ string) (string, string, string, error) {
	if m.UploadDocumentErr != nil {
		return "", "", "", m.UploadDocumentErr
	}
	return "/api/documents/test-uuid", "test-uuid", ".pdf", nil
}
func (m *mockStorageProvider) GetImage(_ context.Context, _ string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *mockStorageProvider) GetDocument(_ context.Context, _ string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *mockStorageProvider) DeleteImage(_ context.Context, identifier string) error {
	m.DeletedBlobs = append(m.DeletedBlobs, identifier)
	return nil
}
func (m *mockStorageProvider) DeleteDocument(_ context.Context, _ string) error { return nil }
func (m *mockStorageProvider) GetImageURL(_ string) string                      { return "" }
func (m *mockStorageProvider) GetDocumentURL(_ string) string                   { return "" }
```

- [ ] **Step 2: Add `createVideoMultipartRequest` helper and minimal video bytes**

Add at the bottom of `internal/handlers/test_helpers.go`:

```go
// minimalMP4 passes isVideoContent — "ftyp" box type at bytes 4-7.
var minimalMP4 = func() []byte {
	b := make([]byte, 20)
	copy(b[4:8], []byte("ftyp"))
	copy(b[8:12], []byte("isom"))
	return b
}()

// minimalJPEG is a valid JPEG header that passes ValidateImageUpload.
var minimalJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

// createVideoMultipartRequest builds a multipart POST with "video", "thumbnail",
// "caption", and "duration_seconds" fields.
func createVideoMultipartRequest(t *testing.T, videoContent, thumbContent []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	videoPart, err := writer.CreateFormFile("video", "clip.mp4")
	if err != nil {
		t.Fatalf("failed to create video form file: %v", err)
	}
	if _, err := videoPart.Write(videoContent); err != nil {
		t.Fatalf("failed to write video content: %v", err)
	}

	thumbPart, err := writer.CreateFormFile("thumbnail", "thumb.jpg")
	if err != nil {
		t.Fatalf("failed to create thumbnail form file: %v", err)
	}
	if _, err := thumbPart.Write(thumbContent); err != nil {
		t.Fatalf("failed to write thumbnail content: %v", err)
	}

	_ = writer.WriteField("caption", "Test caption")
	_ = writer.WriteField("duration_seconds", "15")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go build ./internal/handlers/...
```

Expected: exits 0.

- [ ] **Step 4: Confirm existing handler tests still pass**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -timeout 60s 2>&1 | tail -5
```

Expected: `ok  github.com/networkengineer-cloud/go-volunteer-media/internal/handlers`

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/test_helpers.go
git commit -m "test: extend mockStorageProvider with ProviderName and DeletedBlobs tracking"
```

---

## Task 4: `GetAnimalMedia` handler

**Files:**
- Create: `internal/handlers/animal_video.go`
- Create: `internal/handlers/animal_video_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/handlers/animal_video_test.go`:

```go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupVideoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	assert.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Animal{},
		&models.AnimalImage{},
		&models.AnimalVideo{},
	))
	return db
}

func TestGetAnimalMedia(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

	group := models.Group{Name: "Dogs", Description: "Dog group"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{Username: "vol", Email: "vol@test.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	// Seed one image and one video
	animalIDRef := animal.ID
	assert.NoError(t, db.Create(&models.AnimalImage{
		AnimalID: &animalIDRef,
		UserID:   user.ID,
		ImageURL: "/images/test.jpg",
	}).Error)
	assert.NoError(t, db.Create(&models.AnimalVideo{
		AnimalID:     &animalIDRef,
		UserID:       user.ID,
		VideoURL:     "/videos/test.mp4",
		ThumbnailURL: "/images/thumb.jpg",
	}).Error)

	r := gin.New()
	r.GET("/groups/:id/animals/:animalId/media", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)
	}, GetAnimalMedia(db))

	req := httptest.NewRequest(http.MethodGet,
		"/groups/"+itoa(group.ID)+"/animals/"+itoa(animal.ID)+"/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Images []models.AnimalImage `json:"images"`
		Videos []models.AnimalVideo `json:"videos"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Images, 1)
	assert.Len(t, body.Videos, 1)
}

// itoa converts uint to string for URL building in tests.
func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -run TestGetAnimalMedia -v
```

Expected: FAIL — `GetAnimalMedia` undefined.

- [ ] **Step 3: Create `internal/handlers/animal_video.go` with `GetAnimalMedia`**

Only include imports used by `GetAnimalMedia` — Go will refuse to compile unused imports.

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

type mediaResponse struct {
	Images []models.AnimalImage `json:"images"`
	Videos []models.AnimalVideo `json:"videos"`
}

// GetAnimalMedia returns all images and videos for an animal.
// GET /api/groups/:id/animals/:animalId/media
func GetAnimalMedia(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userIDUint, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		var images []models.AnimalImage
		if err := db.
			Select("id, created_at, updated_at, animal_id, user_id, image_url, caption, is_profile_picture, width, height, file_size").
			Where("animal_id = ?", animalID).
			Order("is_profile_picture DESC, created_at DESC").
			Find(&images).Error; err != nil {
			logger.Error("Failed to fetch animal images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}

		var videos []models.AnimalVideo
		if err := db.Preload("User").
			Where("animal_id = ?", animalID).
			Order("created_at DESC").
			Find(&videos).Error; err != nil {
			logger.Error("Failed to fetch animal videos", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}

		c.JSON(http.StatusOK, mediaResponse{Images: images, Videos: videos})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -run TestGetAnimalMedia -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/animal_video.go internal/handlers/animal_video_test.go
git commit -m "feat: add GetAnimalMedia handler returning unified images and videos"
```

---

## Task 5: `UploadAnimalVideo` handler

**Files:**
- Modify: `internal/handlers/animal_video.go`
- Modify: `internal/handlers/animal_video_test.go`

- [ ] **Step 1: Add upload tests to `internal/handlers/animal_video_test.go`**

Add these two test functions to the existing file:

```go
func TestUploadAnimalVideo_AzureRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{} // Name() returns "mock", not "azure"

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	user := models.User{Username: "vol", Email: "v@t.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))
	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	r := gin.New()
	r.POST("/groups/:id/animals/:animalId/videos", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)
	}, UploadAnimalVideo(db, store))

	req := createVideoMultipartRequest(t, minimalMP4, minimalJPEG)
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Video upload is not available right now")
}

func TestUploadAnimalVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{ProviderName: "azure"}

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	user := models.User{Username: "vol", Email: "v@t.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))
	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	r := gin.New()
	r.POST("/groups/:id/animals/:animalId/videos", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)
	}, UploadAnimalVideo(db, store))

	req := createVideoMultipartRequest(t, minimalMP4, minimalJPEG)
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var video models.AnimalVideo
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &video))
	assert.NotZero(t, video.ID)
	assert.Equal(t, "Test caption", video.Caption)
	assert.Equal(t, 15, video.DurationSeconds)

	// Verify DB record
	var dbVideo models.AnimalVideo
	assert.NoError(t, db.First(&dbVideo, video.ID).Error)
	assert.Equal(t, *dbVideo.AnimalID, animal.ID)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -run "TestUploadAnimalVideo" -v
```

Expected: FAIL — `UploadAnimalVideo` undefined.

- [ ] **Step 3: Add `UploadAnimalVideo` to `internal/handlers/animal_video.go`**

First, replace the import block at the top of `internal/handlers/animal_video.go` with the full set needed by all three handlers:

```go
import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)
```

Then append after `GetAnimalMedia`:

```go
// UploadAnimalVideo handles video uploads to the animal gallery.
// Azure Blob Storage is required — videos are never stored in PostgreSQL.
// POST /api/groups/:id/animals/:animalId/videos
func UploadAnimalVideo(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		isAdmin, _ := c.Get("is_admin")

		if storageProvider.Name() != storage.ProviderAzure {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Video upload is not available right now. Please contact support."})
			return
		}

		if !checkGroupAccess(db, userIDUint, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var animal models.Animal
		if err := db.Where("id = ? AND group_id = ?", animalID, groupID).First(&animal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
			return
		}

		videoFile, err := c.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No video file uploaded"})
			return
		}
		if err := upload.ValidateVideoUpload(videoFile, upload.MaxVideoSize); err != nil {
			if errors.Is(err, upload.ErrFileTooLarge) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "This video is too large. Please use a clip under 200MB."})
			} else {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Only MP4 and MOV videos are supported."})
			}
			return
		}

		thumbnailFile, err := c.FormFile("thumbnail")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No thumbnail file uploaded"})
			return
		}
		if err := upload.ValidateImageUpload(thumbnailFile, upload.MaxImageSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid thumbnail image"})
			return
		}

		videoSrc, err := videoFile.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process video"})
			return
		}
		defer videoSrc.Close()
		videoData, err := io.ReadAll(videoSrc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process video"})
			return
		}

		thumbSrc, err := thumbnailFile.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process thumbnail"})
			return
		}
		defer thumbSrc.Close()
		thumbData, err := io.ReadAll(thumbSrc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process thumbnail"})
			return
		}

		caption := c.PostForm("caption")
		durationSeconds, _ := strconv.Atoi(c.PostForm("duration_seconds"))

		thumbURL, thumbBlobID, thumbExt, err := storageProvider.UploadImage(ctx, thumbData, "image/jpeg", map[string]string{"caption": caption})
		if err != nil {
			logger.Error("Failed to upload thumbnail", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed. Please try again."})
			return
		}

		videoExt := strings.ToLower(filepath.Ext(videoFile.Filename))
		videoMimeType := "video/mp4"
		if videoExt == ".mov" {
			videoMimeType = "video/quicktime"
		}

		videoURL, videoBlobID, videoBlobExt, err := storageProvider.UploadImage(ctx, videoData, videoMimeType, map[string]string{"caption": caption})
		if err != nil {
			logger.Error("Failed to upload video, cleaning up thumbnail", err)
			if delErr := storageProvider.DeleteImage(ctx, thumbBlobID+thumbExt); delErr != nil {
				logger.Error("Failed to clean up thumbnail after video upload failure", delErr)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed. Please try again."})
			return
		}

		animalIDUint, _ := strconv.ParseUint(animalID, 10, 32)
		animalIDVal := uint(animalIDUint)

		animalVideo := models.AnimalVideo{
			AnimalID:        &animalIDVal,
			UserID:          userIDUint,
			VideoURL:        videoURL,
			ThumbnailURL:    thumbURL,
			MimeType:        videoMimeType,
			Caption:         caption,
			DurationSeconds: durationSeconds,
			FileSize:        videoFile.Size,
			BlobIdentifier:  videoBlobID + videoBlobExt,
			ThumbnailBlobID: thumbBlobID + thumbExt,
			BlobExtension:   videoBlobExt,
		}

		if err := db.Create(&animalVideo).Error; err != nil {
			logger.Error("Failed to save video to database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video"})
			return
		}

		db.Preload("User").First(&animalVideo, animalVideo.ID)

		logger.WithFields(map[string]interface{}{
			"video_id":  animalVideo.ID,
			"animal_id": animalID,
			"size":      videoFile.Size,
		}).Info("Video uploaded and stored")

		c.JSON(http.StatusOK, animalVideo)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -run "TestUploadAnimalVideo" -v
```

Expected: both subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/animal_video.go internal/handlers/animal_video_test.go
git commit -m "feat: add UploadAnimalVideo handler with Azure-only enforcement"
```

---

## Task 6: `DeleteAnimalVideo` handler

**Files:**
- Modify: `internal/handlers/animal_video.go`
- Modify: `internal/handlers/animal_video_test.go`

- [ ] **Step 1: Add delete tests to `internal/handlers/animal_video_test.go`**

```go
func TestDeleteAnimalVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{ProviderName: "azure"}

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	owner := models.User{Username: "owner", Email: "owner@t.com", Password: "x"}
	assert.NoError(t, db.Create(&owner).Error)
	assert.NoError(t, db.Model(&owner).Association("Groups").Append(&group))
	other := models.User{Username: "other", Email: "other@t.com", Password: "x"}
	assert.NoError(t, db.Create(&other).Error)
	assert.NoError(t, db.Model(&other).Association("Groups").Append(&group))

	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)
	animalIDRef := animal.ID

	videoBlob := "video-blob-id.mp4"
	thumbBlob := "thumb-blob-id.png"
	video := models.AnimalVideo{
		AnimalID:        &animalIDRef,
		UserID:          owner.ID,
		VideoURL:        "/video.mp4",
		ThumbnailURL:    "/thumb.jpg",
		BlobIdentifier:  videoBlob,
		ThumbnailBlobID: thumbBlob,
	}
	assert.NoError(t, db.Create(&video).Error)

	t.Run("non-owner is forbidden", func(t *testing.T) {
		r := gin.New()
		r.DELETE("/groups/:id/animals/:animalId/videos/:videoId", func(c *gin.Context) {
			c.Set("user_id", other.ID)
			c.Set("is_admin", false)
		}, DeleteAnimalVideo(db, store))

		path := "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos/" + itoa(video.ID)
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("owner can delete and blobs are cleaned up", func(t *testing.T) {
		r := gin.New()
		r.DELETE("/groups/:id/animals/:animalId/videos/:videoId", func(c *gin.Context) {
			c.Set("user_id", owner.ID)
			c.Set("is_admin", false)
		}, DeleteAnimalVideo(db, store))

		path := "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos/" + itoa(video.ID)
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Both blobs must be queued for deletion
		assert.Contains(t, store.DeletedBlobs, videoBlob)
		assert.Contains(t, store.DeletedBlobs, thumbBlob)

		// Record should be soft-deleted
		var count int64
		db.Model(&models.AnimalVideo{}).Where("id = ?", video.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -run "TestDeleteAnimalVideo" -v
```

Expected: FAIL — `DeleteAnimalVideo` undefined.

- [ ] **Step 3: Add `DeleteAnimalVideo` to `internal/handlers/animal_video.go`**

Append after `UploadAnimalVideo`:

```go
// DeleteAnimalVideo deletes a video and its Azure blobs. Only the uploader or a site admin may delete.
// DELETE /api/groups/:id/animals/:animalId/videos/:videoId
func DeleteAnimalVideo(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := middleware.GetLogger(c)
		groupID := c.Param("id")
		animalID := c.Param("animalId")
		videoID := c.Param("videoId")
		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userIDUint, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		var video models.AnimalVideo
		if err := db.Where("id = ?", videoID).First(&video).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		if video.AnimalID == nil || strconv.FormatUint(uint64(*video.AnimalID), 10) != animalID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		isAdminBool, _ := isAdmin.(bool)
		if video.UserID != userIDUint && !isAdminBool {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own videos"})
			return
		}

		if video.BlobIdentifier != "" {
			if err := storageProvider.DeleteImage(ctx, video.BlobIdentifier); err != nil {
				logger.WithFields(map[string]interface{}{"error": err.Error(), "blob": video.BlobIdentifier}).
					Warn("Failed to delete video blob")
			}
		}

		if video.ThumbnailBlobID != "" {
			if err := storageProvider.DeleteImage(ctx, video.ThumbnailBlobID); err != nil {
				logger.WithFields(map[string]interface{}{"error": err.Error(), "blob": video.ThumbnailBlobID}).
					Warn("Failed to delete thumbnail blob")
			}
		}

		if err := db.Delete(&video).Error; err != nil {
			logger.Error("Failed to delete video from database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
	}
}
```

- [ ] **Step 4: Run all handler tests**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./internal/handlers/... -timeout 60s 2>&1 | tail -5
```

Expected: `ok  github.com/networkengineer-cloud/go-volunteer-media/internal/handlers`

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/animal_video.go internal/handlers/animal_video_test.go
git commit -m "feat: add DeleteAnimalVideo handler with blob cleanup"
```

---

## Task 7: Route registration

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Register the three new routes in `cmd/api/main.go`**

Find the block that registers animal image routes (around line 296-301):

```go
group.GET("/animals/:animalId/images", handlers.GetAnimalImages(db))
group.POST("/animals/:animalId/images", handlers.UploadAnimalImageToGallery(db, storageProvider))
group.DELETE("/animals/:animalId/images/:imageId", handlers.DeleteAnimalImage(db, storageProvider))
group.PUT("/animals/:animalId/images/:imageId/set-profile", handlers.SetAnimalProfilePictureGroupScoped(db))
```

Add immediately after those four lines:

```go
group.GET("/animals/:animalId/media", handlers.GetAnimalMedia(db))
group.POST("/animals/:animalId/videos", handlers.UploadAnimalVideo(db, storageProvider))
group.DELETE("/animals/:animalId/videos/:videoId", handlers.DeleteAnimalVideo(db, storageProvider))
```

- [ ] **Step 2: Verify the binary builds**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go build ./cmd/api/...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat: register animal video routes"
```

---

## Task 8: Frontend API types and client methods

**Files:**
- Modify: `frontend/src/api/client.ts`

- [ ] **Step 1: Add `AnimalVideo` and `AnimalMedia` interfaces**

In `frontend/src/api/client.ts`, find the `AnimalImage` interface (around line 244) and add after it:

```typescript
export interface AnimalVideo {
  id: number;
  animal_id: number | null;
  user_id: number;
  video_url: string;
  thumbnail_url: string;
  mime_type: string;
  caption: string;
  duration_seconds: number;
  file_size: number;
  created_at: string;
  user?: User;
}

export interface AnimalMedia {
  images: AnimalImage[];
  videos: AnimalVideo[];
}
```

- [ ] **Step 2: Add `getMedia`, `uploadVideo`, `deleteVideo` to `animalsApi`**

In `frontend/src/api/client.ts`, find `animalsApi` (around line 555). After the existing `deleteImage` method, add:

```typescript
  getMedia: (groupId: number, animalId: number) =>
    api.get<AnimalMedia>('/groups/' + groupId + '/animals/' + animalId + '/media'),
  uploadVideo: (
    groupId: number,
    animalId: number,
    videoFile: File,
    thumbnailBlob: Blob,
    caption?: string,
    durationSeconds?: number,
  ) => {
    const formData = new FormData();
    formData.append('video', videoFile);
    formData.append('thumbnail', new File([thumbnailBlob], 'thumbnail.jpg', { type: 'image/jpeg' }));
    if (caption) formData.append('caption', caption);
    formData.append('duration_seconds', String(durationSeconds ?? 0));
    return api.post<AnimalVideo>(
      '/groups/' + groupId + '/animals/' + animalId + '/videos',
      formData,
    );
  },
  deleteVideo: (groupId: number, animalId: number, videoId: number) =>
    api.delete('/groups/' + groupId + '/animals/' + animalId + '/videos/' + videoId),
```

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd /Users/terrywallace/projects/go-volunteer-media/frontend && npx tsc --noEmit
```

Expected: exits 0 with no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/client.ts
git commit -m "feat: add AnimalVideo type and animalsApi video methods"
```

---

## Task 9: `VideoUpload` component

**Files:**
- Create: `frontend/src/components/VideoUpload.tsx`

- [ ] **Step 1: Create `frontend/src/components/VideoUpload.tsx`**

```tsx
import React, { useState } from 'react';
import { animalsApi } from '../api/client';

const MAX_VIDEO_SIZE = 200 * 1024 * 1024;
const ALLOWED_TYPES = ['video/mp4', 'video/quicktime'];

interface VideoUploadProps {
  groupId: number;
  animalId: number;
  onSuccess: () => void;
  onCancel: () => void;
}

const VideoUpload: React.FC<VideoUploadProps> = ({ groupId, animalId, onSuccess, onCancel }) => {
  const [videoFile, setVideoFile] = useState<File | null>(null);
  const [caption, setCaption] = useState('');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError('Only MP4 and MOV videos are supported.');
      return;
    }
    if (file.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      return;
    }
    setError(null);
    setVideoFile(file);
  };

  const extractThumbnail = (file: File): Promise<{ blob: Blob; duration: number }> =>
    new Promise((resolve, reject) => {
      const video = document.createElement('video');
      video.preload = 'metadata';
      video.muted = true;
      const objectUrl = URL.createObjectURL(file);

      video.onloadeddata = () => {
        video.currentTime = 0;
      };

      video.onseeked = () => {
        const canvas = document.createElement('canvas');
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Canvas context unavailable'));
          return;
        }
        ctx.drawImage(video, 0, 0);
        canvas.toBlob(
          (blob) => {
            URL.revokeObjectURL(objectUrl);
            if (!blob) {
              reject(new Error('Failed to generate thumbnail'));
              return;
            }
            resolve({ blob, duration: Math.round(video.duration) });
          },
          'image/jpeg',
          0.85,
        );
      };

      video.onerror = () => {
        URL.revokeObjectURL(objectUrl);
        reject(new Error('Failed to load video'));
      };

      video.src = objectUrl;
    });

  const handleUpload = async () => {
    if (!videoFile) return;
    setUploading(true);
    setError(null);

    let thumbnailBlob: Blob;
    let duration: number;
    try {
      ({ blob: thumbnailBlob, duration } = await extractThumbnail(videoFile));
    } catch {
      setError("Couldn't generate a preview for this video. Please try a different file.");
      setUploading(false);
      return;
    }

    try {
      await animalsApi.uploadVideo(groupId, animalId, videoFile, thumbnailBlob, caption, duration);
      onSuccess();
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || 'Upload failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="video-upload">
      <h3>Upload Video</h3>
      <input
        type="file"
        accept="video/mp4,video/quicktime,.mp4,.mov"
        onChange={handleFileSelect}
        disabled={uploading}
      />
      {videoFile && (
        <p className="file-selected">{videoFile.name} ({(videoFile.size / 1024 / 1024).toFixed(1)} MB)</p>
      )}
      <input
        type="text"
        placeholder="Caption (optional)"
        value={caption}
        onChange={(e) => setCaption(e.target.value)}
        disabled={uploading}
      />
      {error && <p className="upload-error">{error}</p>}
      <div className="upload-actions">
        <button onClick={onCancel} disabled={uploading} className="btn-secondary">
          Cancel
        </button>
        <button onClick={handleUpload} disabled={!videoFile || uploading} className="btn-primary">
          {uploading ? 'Uploading…' : 'Upload Video'}
        </button>
      </div>
    </div>
  );
};

export default VideoUpload;
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd /Users/terrywallace/projects/go-volunteer-media/frontend && npx tsc --noEmit
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/VideoUpload.tsx
git commit -m "feat: add VideoUpload component with client-side thumbnail extraction"
```

---

## Task 10: `PhotoGallery` integration

**Files:**
- Modify: `frontend/src/pages/PhotoGallery.tsx`

- [ ] **Step 1: Update imports and state in `PhotoGallery.tsx`**

Replace the import line:
```typescript
import { animalsApi } from '../api/client';
import type { Animal, AnimalImage } from '../api/client';
```
with:
```typescript
import { animalsApi } from '../api/client';
import type { Animal, AnimalImage, AnimalVideo } from '../api/client';
import VideoUpload from '../components/VideoUpload';
```

In the component body, add new state variables after the existing `images` state:
```typescript
const [videos, setVideos] = useState<AnimalVideo[]>([]);
const [selectedVideo, setSelectedVideo] = useState<AnimalVideo | null>(null);
const [showVideoUpload, setShowVideoUpload] = useState(false);
```

- [ ] **Step 2: Update `loadData` to use `getMedia`**

In `loadData`, replace:
```typescript
animalsApi.getImages(gId, animalId),
```
with:
```typescript
animalsApi.getMedia(gId, animalId),
```

And replace the destructuring and setImages call:
```typescript
const [animalRes, imagesRes, deletedRes] = results as [
  AxiosResponse<Animal>,
  AxiosResponse<AnimalImage[]>,
  AxiosResponse<{ data?: AnimalImage[] }> | undefined,
];

setAnimal(animalRes.data);
setImages(imagesRes.data);
```
with:
```typescript
const [animalRes, mediaRes, deletedRes] = results as [
  AxiosResponse<Animal>,
  AxiosResponse<{ images: AnimalImage[]; videos: AnimalVideo[] }>,
  AxiosResponse<{ data?: AnimalImage[] }> | undefined,
];

setAnimal(animalRes.data);
setImages(mediaRes.data.images);
setVideos(mediaRes.data.videos);
```

- [ ] **Step 3: Add video delete handler**

After the existing `handleDeleteImage` function, add:
```typescript
const handleDeleteVideo = (videoId: number) => {
  if (!groupId || !id) return;
  openConfirmDialog(
    'Delete Video',
    'Are you sure you want to delete this video?',
    async () => {
      try {
        await animalsApi.deleteVideo(Number(groupId), Number(id), videoId);
        showToast('Video deleted successfully', 'success');
        setSelectedVideo(null);
        await loadData(Number(groupId), Number(id));
      } catch {
        showToast('Failed to delete video', 'error');
      }
    },
  );
};
```

- [ ] **Step 4: Update the gallery header count and upload button**

Replace the `<p className="photo-count">` and `<button>` block:
```tsx
<p className="photo-count">
  {images.length} {images.length === 1 ? 'photo' : 'photos'}
</p>
<button className="btn-primary" onClick={() => setShowUploadModal(true)}>
  + Upload Photo
</button>
```
with:
```tsx
<p className="photo-count">
  {images.length} {images.length === 1 ? 'photo' : 'photos'}, {videos.length}{' '}
  {videos.length === 1 ? 'video' : 'videos'}
</p>
<input
  id="media-upload-input"
  type="file"
  accept="image/*,video/mp4,video/quicktime,.mov"
  style={{ display: 'none' }}
  onChange={(e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    if (file.type.startsWith('video/')) {
      setShowVideoUpload(true);
    } else {
      setEditingImageUrl(URL.createObjectURL(file));
      setShowImageEditor(true);
    }
  }}
/>
<button className="btn-primary" onClick={() => document.getElementById('media-upload-input')?.click()}>
  + Upload Media
</button>
```

- [ ] **Step 5: Update the empty state and add videos to the grid**

Replace the existing `images.length === 0` conditional and grid:
```tsx
{images.length === 0 ? (
  <div className="no-photos">
    <p>No photos have been shared for {animal.name} yet.</p>
    <button className="btn-primary" onClick={() => setShowUploadModal(true)}>
      Upload First Photo
    </button>
  </div>
) : (
  <div className="photos-grid">
    {images.map((image, index) => (
      <div
        key={image.id}
        className="photo-card"
        onClick={() => openLightbox(index)}
      >
        {image.is_profile_picture && (
          <div className="profile-badge">Profile Picture</div>
        )}
        <img src={image.image_url} alt={image.caption || `Photo ${index + 1}`} />
        <div className="photo-info">
          <span className="photo-author">{image.user?.username}</span>
          <span className="photo-date">
            {new Date(image.created_at).toLocaleDateString()}
          </span>
        </div>
        {image.caption && (
          <div className="photo-caption">{image.caption}</div>
        )}
      </div>
    ))}
  </div>
)}
```
with:
```tsx
{images.length === 0 && videos.length === 0 ? (
  <div className="no-photos">
    <p>No media has been shared for {animal.name} yet.</p>
    <button className="btn-primary" onClick={() => document.getElementById('media-upload-input')?.click()}>
      Upload First Media
    </button>
  </div>
) : (
  <div className="photos-grid">
    {images.map((image, index) => (
      <div
        key={`img-${image.id}`}
        className="photo-card"
        onClick={() => openLightbox(index)}
      >
        {image.is_profile_picture && (
          <div className="profile-badge">Profile Picture</div>
        )}
        <img src={image.image_url} alt={image.caption || `Photo ${index + 1}`} />
        <div className="photo-info">
          <span className="photo-author">{image.user?.username}</span>
          <span className="photo-date">{new Date(image.created_at).toLocaleDateString()}</span>
        </div>
        {image.caption && <div className="photo-caption">{image.caption}</div>}
      </div>
    ))}
    {videos.map((video) => (
      <div
        key={`vid-${video.id}`}
        className="photo-card video-card"
        onClick={() => setSelectedVideo(video)}
      >
        <div className="video-thumbnail-wrapper">
          <img src={video.thumbnail_url} alt={video.caption || 'Video thumbnail'} />
          <div className="play-overlay">▶</div>
        </div>
        <div className="photo-info">
          <span className="photo-author">{video.user?.username}</span>
          <span className="photo-date">{new Date(video.created_at).toLocaleDateString()}</span>
        </div>
        {video.caption && <div className="photo-caption">{video.caption}</div>}
      </div>
    ))}
  </div>
)}
```

- [ ] **Step 6: Add video modal and VideoUpload modal**

After the existing lightbox JSX block, add:

```tsx
{/* Video player modal */}
{selectedVideo && (
  <div className="lightbox" onClick={() => setSelectedVideo(null)}>
    <div className="lightbox-content" onClick={(e) => e.stopPropagation()}>
      <button className="lightbox-close" onClick={() => setSelectedVideo(null)}>✕</button>
      <div className="lightbox-image-container">
        <video
          src={selectedVideo.video_url}
          controls
          autoPlay
          style={{ maxWidth: '100%', maxHeight: '80vh' }}
        />
        <div className="lightbox-info">
          {selectedVideo.caption && (
            <p className="lightbox-caption">{selectedVideo.caption}</p>
          )}
          <div className="lightbox-meta">
            <span>By {selectedVideo.user?.username}</span>
            <span>{new Date(selectedVideo.created_at).toLocaleDateString()}</span>
          </div>
          <div className="lightbox-actions">
            {(isAdmin || selectedVideo.user?.id === user?.id) && (
              <button
                className="btn-danger"
                onClick={() => handleDeleteVideo(selectedVideo.id)}
              >
                Delete Video
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  </div>
)}

{/* Video upload modal */}
{showVideoUpload && groupId && id && (
  <div className="modal-overlay">
    <div className="modal-content">
      <VideoUpload
        groupId={Number(groupId)}
        animalId={Number(id)}
        onSuccess={() => {
          setShowVideoUpload(false);
          showToast('Video uploaded successfully', 'success');
          loadData(Number(groupId), Number(id));
        }}
        onCancel={() => setShowVideoUpload(false)}
      />
    </div>
  </div>
)}
```

- [ ] **Step 7: Verify TypeScript compilation**

```bash
cd /Users/terrywallace/projects/go-volunteer-media/frontend && npx tsc --noEmit
```

Expected: exits 0. Fix any type errors before continuing.

- [ ] **Step 8: Run the full backend test suite**

```bash
cd /Users/terrywallace/projects/go-volunteer-media && go test ./... -timeout 120s 2>&1 | tail -10
```

Expected: all packages pass.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/pages/PhotoGallery.tsx
git commit -m "feat: integrate video upload and playback into PhotoGallery"
```
