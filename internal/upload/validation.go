package upload

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	// MaxImageSize is the maximum allowed image upload size (10MB)
	MaxImageSize = 10 * 1024 * 1024 // 10 MB

	// MaxHeroImageSize is the maximum size for hero images (5MB)
	MaxHeroImageSize = 5 * 1024 * 1024 // 5 MB
)

var (
	// ErrFileTooLarge is returned when uploaded file exceeds size limit
	ErrFileTooLarge = errors.New("file size exceeds maximum limit")

	// ErrInvalidFileType is returned when file type is not allowed
	ErrInvalidFileType = errors.New("file type not allowed")

	// ErrInvalidFile is returned when file is invalid or corrupted
	ErrInvalidFile = errors.New("invalid or corrupted file")
)

// AllowedImageTypes maps file extensions to their MIME types
// Note: Browsers often convert HEIC to JPEG automatically
var AllowedImageTypes = map[string][]string{
	".jpg":  {"image/jpeg", "image/jpg"},
	".jpeg": {"image/jpeg", "image/jpg"},
	".png":  {"image/png"},
	".gif":  {"image/gif"},
	".webp": {"image/webp"},
	".heic": {"image/heic", "image/heif"},
	".heif": {"image/heic", "image/heif"},
}

// ValidateImageUpload validates an uploaded image file
func ValidateImageUpload(file *multipart.FileHeader, maxSize int64) error {
	// Check file size
	if file.Size > maxSize {
		return fmt.Errorf("%w: file size is %d bytes, maximum is %d bytes",
			ErrFileTooLarge, file.Size, maxSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedMimeTypes, ok := AllowedImageTypes[ext]
	if !ok {
		return fmt.Errorf("%w: extension %s is not allowed", ErrInvalidFileType, ext)
	}

	// Open file to check content type
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type from file content
	contentType := http.DetectContentType(buffer[:n])

	// Be permissive with content type validation for mobile uploads
	// Many mobile browsers convert images automatically (e.g., HEIC to JPEG)
	// Just ensure it's some kind of image
	if !strings.HasPrefix(contentType, "image/") && contentType != "application/octet-stream" {
		// Only reject if it's clearly not an image
		// Note: application/octet-stream is allowed because some mobile browsers
		// don't send proper MIME types for converted images
		validContentType := false
		for _, allowedType := range allowedMimeTypes {
			if contentType == allowedType {
				validContentType = true
				break
			}
		}
		
		if !validContentType {
			return fmt.Errorf("%w: file does not appear to be a valid image (detected: %s)",
				ErrInvalidFileType, contentType)
		}
	}

	return nil
}

// SanitizeFilename removes potentially dangerous characters from filename
func SanitizeFilename(filename string) string {
	// Get extension
	ext := filepath.Ext(filename)

	// Remove extension and clean the name
	name := strings.TrimSuffix(filename, ext)

	// Replace problematic characters
	name = strings.Map(func(r rune) rune {
		// Allow alphanumeric, dash, underscore, and space
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return '-'
	}, name)

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	// Return sanitized name with extension
	return name + ext
}

// ValidateImageContent performs additional validation on image content
// This can be used with image decoding to ensure file is a valid image
func ValidateImageContent(data []byte) error {
	// Check minimum file size (100 bytes)
	if len(data) < 100 {
		return fmt.Errorf("%w: file too small to be a valid image", ErrInvalidFile)
	}

	// Verify magic numbers for common image formats
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		// JPEG
		return nil
	} else if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		// PNG
		return nil
	} else if bytes.HasPrefix(data, []byte{0x47, 0x49, 0x46}) {
		// GIF
		return nil
	} else if bytes.HasPrefix(data, []byte{0x52, 0x49, 0x46, 0x46}) {
		// WebP (RIFF format)
		if len(data) > 12 && bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
			return nil
		}
	}

	return fmt.Errorf("%w: unrecognized image format", ErrInvalidFile)
}
