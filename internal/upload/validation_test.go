package upload

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestValidateImageUpload(t *testing.T) {
	tests := []struct {
		name        string
		fileSize    int64
		filename    string
		content     []byte
		maxSize     int64
		wantErr     error
		errContains string
	}{
		{
			name:     "valid JPEG image",
			fileSize: 1024 * 1024, // 1MB
			filename: "test.jpg",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}, // JPEG header
			maxSize:  MaxImageSize,
			wantErr:  nil,
		},
		{
			name:     "valid PNG image",
			fileSize: 2 * 1024 * 1024, // 2MB
			filename: "test.png",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
			maxSize:  MaxImageSize,
			wantErr:  nil,
		},
		{
			name:        "file too large",
			fileSize:    11 * 1024 * 1024, // 11MB
			filename:    "large.jpg",
			content:     []byte{0xFF, 0xD8, 0xFF},
			maxSize:     MaxImageSize,
			wantErr:     ErrFileTooLarge,
			errContains: "file size exceeds maximum limit",
		},
		{
			name:        "invalid extension",
			fileSize:    1024,
			filename:    "test.exe",
			content:     []byte{0xFF, 0xD8, 0xFF},
			maxSize:     MaxImageSize,
			wantErr:     ErrInvalidFileType,
			errContains: "extension .exe is not allowed",
		},
		{
			name:        "mismatched content type",
			fileSize:    1024,
			filename:    "test.jpg",
			content:     []byte("This is not an image file"),
			maxSize:     MaxImageSize,
			wantErr:     ErrInvalidFileType,
			errContains: "file content type is",
		},
		{
			name:     "valid WebP image",
			fileSize: 1024,
			filename: "test.webp",
			// Need a more complete WebP header for detection
			content:  append([]byte{0x52, 0x49, 0x46, 0x46, 0x20, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38}, make([]byte, 500)...),
			maxSize:  MaxImageSize,
			wantErr:  nil,
		},
		{
			name:     "valid GIF image",
			fileSize: 1024,
			filename: "test.gif",
			content:  []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, // GIF89a header
			maxSize:  MaxImageSize,
			wantErr:  nil,
		},
		{
			name:     "lowercase extension",
			fileSize: 1024,
			filename: "test.JPEG",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0},
			maxSize:  MaxImageSize,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock multipart.FileHeader
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", tt.filename)
			if err != nil {
				t.Fatalf("Failed to create form file: %v", err)
			}

			// Write test content
			if _, err := part.Write(tt.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			writer.Close()

			// Parse the multipart form
			req := &http.Request{
				Method: "POST",
				Header: map[string][]string{
					"Content-Type": {writer.FormDataContentType()},
				},
				Body: http.NoBody,
			}
			req.Body = http.NoBody

			// Create a reader from the body
			reader := multipart.NewReader(body, writer.Boundary())
			form, err := reader.ReadForm(32 << 20) // 32MB max
			if err != nil {
				t.Fatalf("Failed to read form: %v", err)
			}
			defer form.RemoveAll()

			// Get the file header
			files := form.File["file"]
			if len(files) == 0 {
				t.Fatal("No files in form")
			}
			fileHeader := files[0]

			// Override size for testing
			fileHeader.Size = tt.fileSize

			// Test validation
			err = ValidateImageUpload(fileHeader, tt.maxSize)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "clean filename",
			filename: "test-image_123.jpg",
			want:     "test-image_123.jpg",
		},
		{
			name:     "filename with spaces",
			filename: "test image file.png",
			want:     "test image file.png",
		},
		{
			name:     "filename with special characters",
			filename: "test!@#$%^&*().jpg",
			want:     "test----------.jpg",
		},
		{
			name:     "filename with unicode",
			filename: "test_файл_ファイル.png",
			want:     "test_----_----.png",
		},
		{
			name:     "very long filename",
			filename: strings.Repeat("a", 200) + ".jpg",
			want:     strings.Repeat("a", 100) + ".jpg",
		},
		{
			name:     "filename with path separators",
			filename: "../../../etc/passwd.jpg",
			want:     "---------etc-passwd.jpg",
		},
		{
			name:     "filename with null bytes",
			filename: "test\x00file.jpg",
			want:     "test-file.jpg",
		},
		{
			name:     "multiple extensions",
			filename: "test.tar.gz.jpg",
			want:     "test-tar-gz.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.filename)
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateImageContent(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     error
		errContains string
	}{
		{
			name:    "valid JPEG",
			data:    append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}, make([]byte, 88)...),
			wantErr: nil,
		},
		{
			name:    "valid PNG",
			data:    append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 92)...),
			wantErr: nil,
		},
		{
			name:    "valid GIF",
			data:    append([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, make([]byte, 94)...),
			wantErr: nil,
		},
		{
			name:    "valid WebP",
			data:    append([]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}, make([]byte, 88)...),
			wantErr: nil,
		},
		{
			name:        "file too small",
			data:        []byte{0xFF, 0xD8},
			wantErr:     ErrInvalidFile,
			errContains: "file too small",
		},
		{
			name:        "unrecognized format",
			data:        append([]byte{0x00, 0x00, 0x00, 0x00}, make([]byte, 96)...),
			wantErr:     ErrInvalidFile,
			errContains: "unrecognized image format",
		},
		{
			name:        "text file",
			data:        []byte("This is a text file, not an image"),
			wantErr:     ErrInvalidFile,
			errContains: "file too small",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageContent(tt.data)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestAllowedImageTypes(t *testing.T) {
	// Test that all expected extensions are present
	expectedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

	for _, ext := range expectedExtensions {
		if _, ok := AllowedImageTypes[ext]; !ok {
			t.Errorf("Expected extension %s to be in AllowedImageTypes", ext)
		}
	}

	// Test that MIME types are correct
	if len(AllowedImageTypes[".jpg"]) == 0 || AllowedImageTypes[".jpg"][0] != "image/jpeg" {
		t.Error("Expected .jpg to map to image/jpeg")
	}
	if len(AllowedImageTypes[".png"]) == 0 || AllowedImageTypes[".png"][0] != "image/png" {
		t.Error("Expected .png to map to image/png")
	}
}

func TestConstants(t *testing.T) {
	// Test that constants have expected values
	if MaxImageSize != 10*1024*1024 {
		t.Errorf("Expected MaxImageSize to be 10MB, got %d", MaxImageSize)
	}

	if MaxHeroImageSize != 5*1024*1024 {
		t.Errorf("Expected MaxHeroImageSize to be 5MB, got %d", MaxHeroImageSize)
	}
}
