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

// itoa converts uint to string for URL building in tests.
func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
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

func TestUploadAnimalVideo_VideoUploadFails_ThumbnailCleanedup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)
	// Call 1 (thumbnail) succeeds; call 2 (video) fails.
	store := &mockStorageProvider{
		ProviderName:          "azure",
		UploadImageCallErrors: map[int]error{2: fmt.Errorf("azure: container unavailable")},
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
	assert.Contains(t, store.DeletedBlobs, "test-uuid-1.png", "thumbnail blob should be cleaned up after video upload failure")
	assert.Len(t, store.DeletedBlobs, 1, "only the thumbnail should be deleted")
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
	assert.Contains(t, store.DeletedBlobs, "test-uuid-1.png", "thumbnail blob should be cleaned up on DB failure")
	assert.Contains(t, store.DeletedBlobs, "test-uuid-2.png", "video blob should be cleaned up on DB failure")
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
