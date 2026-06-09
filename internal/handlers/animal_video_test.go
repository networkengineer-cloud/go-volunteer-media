package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
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
		AnimalID:     animalIDRef,
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

func TestGetAnimalMedia_UnknownAnimalReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	user := models.User{Username: "vol", Email: "vol@test.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	r := gin.New()
	r.GET("/groups/:id/animals/:animalId/media", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)
	}, GetAnimalMedia(db))

	req := httptest.NewRequest(http.MethodGet,
		"/groups/"+itoa(group.ID)+"/animals/99999/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnimalMedia_NonMemberIsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	stranger := models.User{Username: "stranger", Email: "stranger@test.com", Password: "x"}
	assert.NoError(t, db.Create(&stranger).Error)
	// stranger is deliberately NOT added to the group
	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	r := gin.New()
	r.GET("/groups/:id/animals/:animalId/media", func(c *gin.Context) {
		c.Set("user_id", stranger.ID)
		c.Set("is_admin", false)
	}, GetAnimalMedia(db))

	req := httptest.NewRequest(http.MethodGet,
		"/groups/"+itoa(group.ID)+"/animals/"+itoa(animal.ID)+"/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetAnimalMedia_ImagesIncludeUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

	group := models.Group{Name: "Dogs2", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	user := models.User{Username: "uploader", Email: "up@test.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))
	animal := models.Animal{Name: "Buddy", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	animalIDRef := animal.ID
	assert.NoError(t, db.Create(&models.AnimalImage{
		AnimalID: &animalIDRef,
		UserID:   user.ID,
		ImageURL: "/images/buddy.jpg",
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
	assert.NotZero(t, body.Images[0].User.ID, "image should include preloaded User")
	assert.Equal(t, user.ID, body.Images[0].User.ID)
}

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

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
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

	assert.Equal(t, http.StatusCreated, w.Code)

	var video models.AnimalVideo
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &video))
	assert.NotZero(t, video.ID)
	assert.Equal(t, "Test caption", video.Caption)
	assert.Equal(t, 15, video.DurationSeconds)

	// Verify DB record
	var dbVideo models.AnimalVideo
	assert.NoError(t, db.First(&dbVideo, video.ID).Error)
	assert.Equal(t, dbVideo.AnimalID, animal.ID)
	assert.NotZero(t, video.User.ID, "response should include the preloaded User")
	assert.Equal(t, user.ID, video.User.ID)
}

// createVideoRequestWithFields builds a multipart POST with explicit caption, duration, and filename.
func createVideoRequestWithFields(t *testing.T, videoContent, thumbContent []byte, filename, caption, duration string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	vp, err := writer.CreateFormFile("video", filename)
	if err != nil {
		t.Fatalf("failed to create video form file: %v", err)
	}
	if _, err := vp.Write(videoContent); err != nil {
		t.Fatalf("failed to write video content: %v", err)
	}

	tp, err := writer.CreateFormFile("thumbnail", "thumb.jpg")
	if err != nil {
		t.Fatalf("failed to create thumbnail form file: %v", err)
	}
	if _, err := tp.Write(thumbContent); err != nil {
		t.Fatalf("failed to write thumbnail content: %v", err)
	}

	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("duration_seconds", duration)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadAnimalVideo_NonMemberGetsForbiddenNotStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{} // Name() returns "mock", not "azure"

	group := models.Group{Name: "Dogs", Description: "x"}
	assert.NoError(t, db.Create(&group).Error)
	user := models.User{Username: "stranger", Email: "s@t.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	// user is deliberately NOT added to group

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

	assert.Equal(t, http.StatusForbidden, w.Code, "access check must run before storage check so non-members don't learn about Azure configuration")
}

func TestUploadAnimalVideo_MovFile_StoresMimeTypeAsQuicktime(t *testing.T) {
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

	req := createVideoRequestWithFields(t, minimalMP4, minimalJPEG, "clip.mov", "", "5")
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var video models.AnimalVideo
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &video))
	assert.Equal(t, "video/quicktime", video.MimeType)
}

func TestUploadAnimalVideo_NegativeDuration_ReturnsBadRequest(t *testing.T) {
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

	req := createVideoRequestWithFields(t, minimalMP4, minimalJPEG, "clip.mp4", "hello", "-1")
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "duration")
}

func TestUploadAnimalVideo_DurationTooLarge_ReturnsBadRequest(t *testing.T) {
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

	req := createVideoRequestWithFields(t, minimalMP4, minimalJPEG, "clip.mp4", "", "3601")
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "duration")
}

func TestUploadAnimalVideo_CaptionTooLong_ReturnsBadRequest(t *testing.T) {
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

	longCaption := strings.Repeat("x", 501)
	req := createVideoRequestWithFields(t, minimalMP4, minimalJPEG, "clip.mp4", longCaption, "10")
	req.URL.Path = "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "caption")
}

func TestUploadAnimalVideo_VideoUploadFails_ThumbnailCleanedup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{
		ProviderName:             "azure",
		UploadImageErrForMimeType: map[string]error{"video/mp4": fmt.Errorf("azure: container unavailable")},
	}

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
	assert.Len(t, store.DeletedBlobs, 1, "only the thumbnail blob should be deleted")
}

func TestUploadAnimalVideo_ThumbnailUploadFails_VideoCleanedup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{
		ProviderName:             "azure",
		UploadImageErrForMimeType: map[string]error{"image/jpeg": fmt.Errorf("azure: container unavailable")},
	}

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
	assert.Len(t, store.DeletedBlobs, 1, "only the video blob should be deleted")
}

func TestUploadAnimalVideo_BothUploadsFail_NoBlobsDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	store := &mockStorageProvider{
		ProviderName:   "azure",
		UploadImageErr: fmt.Errorf("azure: service unavailable"),
	}

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
	assert.Empty(t, store.DeletedBlobs, "no blobs to clean up when both uploads failed")
}

func TestUploadAnimalVideo_DBCreateFails_BothBlobsCleanedup(t *testing.T) {
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

	// Drop the video table so db.Create fails while earlier queries still work.
	db.Exec("DROP TABLE animal_videos")

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
	assert.Len(t, store.DeletedBlobs, 2, "both blobs should be cleaned up on DB failure")
}

func TestServeImage_ThumbnailFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("serves thumbnail blob via thumbnail_url lookup", func(t *testing.T) {
		db := setupVideoTestDB(t)
		imgData := []byte{0xFF, 0xD8, 0xFF} // fake JPEG bytes
		store := &mockStorageProvider{GetImageData: imgData, GetImageMime: "image/jpeg"}

		assert.NoError(t, db.Create(&models.AnimalVideo{
			AnimalID:        1,
			UserID:          1,
			VideoURL:        "/api/videos/video-uuid",
			ThumbnailURL:    "/api/images/thumb-uuid",
			ThumbnailBlobID: "thumb-uuid.jpg",
		}).Error)

		r := gin.New()
		r.GET("/api/images/:uuid", ServeImage(db, store))

		req := httptest.NewRequest(http.MethodGet, "/api/images/thumb-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
		assert.Equal(t, imgData, w.Body.Bytes())
	})

	t.Run("returns 404 when ThumbnailBlobID is empty", func(t *testing.T) {
		db := setupVideoTestDB(t)
		store := &mockStorageProvider{}

		assert.NoError(t, db.Create(&models.AnimalVideo{
			AnimalID:        1,
			UserID:          1,
			VideoURL:        "/api/videos/video-uuid2",
			ThumbnailURL:    "/api/images/thumb-no-blob",
			ThumbnailBlobID: "",
		}).Error)

		r := gin.New()
		r.GET("/api/images/:uuid", ServeImage(db, store))

		req := httptest.NewRequest(http.MethodGet, "/api/images/thumb-no-blob", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 on non-ErrNotFound storage error", func(t *testing.T) {
		db := setupVideoTestDB(t)
		store := &mockStorageProvider{GetImageErr: fmt.Errorf("azure: connection refused")}

		assert.NoError(t, db.Create(&models.AnimalVideo{
			AnimalID:        1,
			UserID:          1,
			VideoURL:        "/api/videos/video-uuid3",
			ThumbnailURL:    "/api/images/thumb-infra-err",
			ThumbnailBlobID: "thumb-infra-err.jpg",
		}).Error)

		r := gin.New()
		r.GET("/api/images/:uuid", ServeImage(db, store))

		req := httptest.NewRequest(http.MethodGet, "/api/images/thumb-infra-err", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 404 when storage returns ErrNotFound", func(t *testing.T) {
		db := setupVideoTestDB(t)
		store := &mockStorageProvider{GetImageErr: storage.ErrNotFound}

		assert.NoError(t, db.Create(&models.AnimalVideo{
			AnimalID:        1,
			UserID:          1,
			VideoURL:        "/api/videos/video-uuid4",
			ThumbnailURL:    "/api/images/thumb-missing",
			ThumbnailBlobID: "thumb-missing.jpg",
		}).Error)

		r := gin.New()
		r.GET("/api/images/:uuid", ServeImage(db, store))

		req := httptest.NewRequest(http.MethodGet, "/api/images/thumb-missing", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 when DB query itself fails", func(t *testing.T) {
		db := setupVideoTestDB(t)
		store := &mockStorageProvider{}

		// Drop the table so the thumbnail_url query returns a DB error, not ErrRecordNotFound.
		db.Exec("DROP TABLE animal_videos")

		r := gin.New()
		r.GET("/api/images/:uuid", ServeImage(db, store))

		req := httptest.NewRequest(http.MethodGet, "/api/images/thumb-db-err", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestDeleteAnimalVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

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

	videoBlob := "video-blob-id.mp4"
	thumbBlob := "thumb-blob-id.png"

	t.Run("non-member is forbidden", func(t *testing.T) {
		stranger := models.User{Username: "stranger", Email: "stranger@t.com", Password: "x"}
		assert.NoError(t, db.Create(&stranger).Error)
		// stranger is deliberately NOT added to group

		store := &mockStorageProvider{ProviderName: "azure"}
		animalIDRef := animal.ID
		video := models.AnimalVideo{
			AnimalID:        animalIDRef,
			UserID:          owner.ID,
			VideoURL:        "/video.mp4",
			ThumbnailURL:    "/thumb.jpg",
			BlobIdentifier:  videoBlob,
			ThumbnailBlobID: thumbBlob,
		}
		assert.NoError(t, db.Create(&video).Error)

		r := gin.New()
		r.DELETE("/groups/:id/animals/:animalId/videos/:videoId", func(c *gin.Context) {
			c.Set("user_id", stranger.ID)
			c.Set("is_admin", false)
		}, DeleteAnimalVideo(db, store))

		path := "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos/" + itoa(video.ID)
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Empty(t, store.DeletedBlobs)
	})

	t.Run("cross-group delete rejected when animal belongs to different group", func(t *testing.T) {
		group2 := models.Group{Name: "Cats", Description: "x"}
		assert.NoError(t, db.Create(&group2).Error)
		animalGroup2 := models.Animal{Name: "Luna", Species: "Cat", GroupID: group2.ID, Status: "available"}
		assert.NoError(t, db.Create(&animalGroup2).Error)

		animalIDRef := animalGroup2.ID
		video := models.AnimalVideo{
			AnimalID:        animalIDRef,
			UserID:          owner.ID,
			VideoURL:        "/video.mp4",
			ThumbnailURL:    "/thumb.jpg",
			BlobIdentifier:  "cross-group-video.mp4",
			ThumbnailBlobID: "cross-group-thumb.png",
		}
		assert.NoError(t, db.Create(&video).Error)

		store := &mockStorageProvider{ProviderName: "azure"}
		r := gin.New()
		r.DELETE("/groups/:id/animals/:animalId/videos/:videoId", func(c *gin.Context) {
			c.Set("user_id", owner.ID)
			c.Set("is_admin", false)
		}, DeleteAnimalVideo(db, store))

		path := "/groups/" + itoa(group.ID) + "/animals/" + itoa(animalGroup2.ID) + "/videos/" + itoa(video.ID)
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Empty(t, store.DeletedBlobs, "no blobs should be deleted when animal is in a different group")
	})

	t.Run("site admin can delete video without being the uploader", func(t *testing.T) {
		admin := models.User{Username: "admin", Email: "admin@t.com", Password: "x", IsAdmin: true}
		assert.NoError(t, db.Create(&admin).Error)

		animalIDRef := animal.ID
		video := models.AnimalVideo{
			AnimalID:        animalIDRef,
			UserID:          owner.ID,
			VideoURL:        "/video.mp4",
			ThumbnailURL:    "/thumb.jpg",
			BlobIdentifier:  videoBlob,
			ThumbnailBlobID: thumbBlob,
		}
		assert.NoError(t, db.Create(&video).Error)

		store := &mockStorageProvider{ProviderName: "azure"}
		r := gin.New()
		r.DELETE("/groups/:id/animals/:animalId/videos/:videoId", func(c *gin.Context) {
			c.Set("user_id", admin.ID)
			c.Set("is_admin", true)
		}, DeleteAnimalVideo(db, store))

		path := "/groups/" + itoa(group.ID) + "/animals/" + itoa(animal.ID) + "/videos/" + itoa(video.ID)
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, store.DeletedBlobs, videoBlob)
		assert.Contains(t, store.DeletedBlobs, thumbBlob)
	})

	t.Run("non-owner is forbidden", func(t *testing.T) {
		store := &mockStorageProvider{ProviderName: "azure"}
		animalIDRef := animal.ID
		video := models.AnimalVideo{
			AnimalID:        animalIDRef,
			UserID:          owner.ID,
			VideoURL:        "/video.mp4",
			ThumbnailURL:    "/thumb.jpg",
			BlobIdentifier:  videoBlob,
			ThumbnailBlobID: thumbBlob,
		}
		assert.NoError(t, db.Create(&video).Error)

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
		assert.Empty(t, store.DeletedBlobs, "no blobs should be deleted on forbidden request")
	})

	t.Run("owner can delete and blobs are cleaned up", func(t *testing.T) {
		store := &mockStorageProvider{ProviderName: "azure"}
		animalIDRef := animal.ID
		video := models.AnimalVideo{
			AnimalID:        animalIDRef,
			UserID:          owner.ID,
			VideoURL:        "/video.mp4",
			ThumbnailURL:    "/thumb.jpg",
			BlobIdentifier:  videoBlob,
			ThumbnailBlobID: thumbBlob,
		}
		assert.NoError(t, db.Create(&video).Error)

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

		assert.Contains(t, store.DeletedBlobs, videoBlob)
		assert.Contains(t, store.DeletedBlobs, thumbBlob)

		var count int64
		db.Model(&models.AnimalVideo{}).Where("id = ?", video.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
