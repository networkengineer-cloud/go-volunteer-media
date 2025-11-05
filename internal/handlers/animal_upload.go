package handlers

import (
	"fmt"
	"image"
	_ "image/gif" // Register GIF format
	"image/jpeg"
	_ "image/png" // Register PNG format
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"github.com/nfnt/resize"
)

// UploadAnimalImage handles secure animal image uploads with optimization
func UploadAnimalImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := middleware.GetLogger(c)

		file, err := c.FormFile("image")
		if err != nil {
			logger.Error("Failed to get form file", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Validate file upload (size, type, content)
		if err := upload.ValidateImageUpload(file, upload.MaxImageSize); err != nil {
			logger.Error("File validation failed", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file: " + err.Error()})
			return
		}

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			logger.Error("Failed to open uploaded file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		defer src.Close()

		// Decode the image
		img, format, err := image.Decode(src)
		if err != nil {
			logger.Error("Failed to decode image", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file"})
			return
		}
		logger.WithFields(map[string]interface{}{
			"format": format,
			"width":  img.Bounds().Dx(),
			"height": img.Bounds().Dy(),
		}).Debug("Received image for upload")

		// Resize image if it's larger than 1200px on the longest side
		maxDimension := uint(1200)
		var resizedImg image.Image

		bounds := img.Bounds()
		width := uint(bounds.Dx())
		height := uint(bounds.Dy())

		if width > maxDimension || height > maxDimension {
			if width > height {
				// Landscape - resize based on width
				resizedImg = resize.Resize(maxDimension, 0, img, resize.Lanczos3)
			} else {
				// Portrait or square - resize based on height
				resizedImg = resize.Resize(0, maxDimension, img, resize.Lanczos3)
			}
			logger.WithFields(map[string]interface{}{
				"new_width":  resizedImg.Bounds().Dx(),
				"new_height": resizedImg.Bounds().Dy(),
			}).Debug("Image resized")
		} else {
			resizedImg = img
			logger.Debug("Image dimensions acceptable, no resizing needed")
		}

		// Generate unique filename (always save as .jpg for consistency)
		fname := fmt.Sprintf("%d_%s.jpg", time.Now().UnixNano(), uuid.New().String())
		uploadPath := filepath.Join("public", "uploads", fname)

		logger.WithField("path", uploadPath).Debug("Saving optimized image")

		// Create the output file
		outFile, err := os.Create(uploadPath)
		if err != nil {
			logger.Error("Failed to create output file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}
		defer outFile.Close()

		// Encode as JPEG with quality 85 (good balance between quality and size)
		if err := jpeg.Encode(outFile, resizedImg, &jpeg.Options{Quality: 85}); err != nil {
			logger.Error("Failed to encode image", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Return public URL
		url := "/uploads/" + fname
		logger.WithField("url", url).Info("Image uploaded and optimized successfully")
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}
