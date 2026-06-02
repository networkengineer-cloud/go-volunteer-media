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
