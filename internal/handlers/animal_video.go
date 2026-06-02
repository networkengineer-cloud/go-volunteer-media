package handlers

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

		if err := db.Preload("User").First(&animalVideo, animalVideo.ID).Error; err != nil {
			logger.Error("Failed to preload user for video response", err)
		}

		logger.WithFields(map[string]interface{}{
			"video_id":  animalVideo.ID,
			"animal_id": animalID,
			"size":      videoFile.Size,
		}).Info("Video uploaded and stored")

		c.JSON(http.StatusOK, animalVideo)
	}
}
