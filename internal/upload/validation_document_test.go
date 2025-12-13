package upload

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"testing"
)

func TestValidateDocumentUpload(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     []byte
		maxSize     int64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid PDF",
			filename:    "protocol.pdf",
			content:     []byte("%PDF-1.4\n%âãÏÓ\ntest content"),
			maxSize:     MaxDocumentSize,
			expectError: false,
		},
		{
			name:        "Valid DOCX (ZIP format)",
			filename:    "protocol.docx",
			content:     []byte("PK\x03\x04test content"),
			maxSize:     MaxDocumentSize,
			expectError: false,
		},
		{
			name:        "File too large",
			filename:    "large.pdf",
			content:     bytes.Repeat([]byte("x"), 100),
			maxSize:     50, // Set max size smaller than content
			expectError: true,
			errorMsg:    "file size exceeds maximum limit",
		},
		{
			name:        "Invalid file extension",
			filename:    "document.txt",
			content:     []byte("test content"),
			maxSize:     MaxDocumentSize,
			expectError: true,
			errorMsg:    "extension .txt is not allowed",
		},
		{
			name:        "Wrong content type for PDF",
			filename:    "fake.pdf",
			content:     []byte("This is not a PDF file"),
			maxSize:     MaxDocumentSize,
			expectError: true,
			errorMsg:    "file does not appear to be a valid PDF document",
		},
		{
			name:        "Empty file",
			filename:    "empty.pdf",
			content:     []byte{},
			maxSize:     MaxDocumentSize,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart file header
			file := createMockFileHeader(tt.filename, tt.content)

			err := ValidateDocumentUpload(file, tt.maxSize)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// createMockFileHeader creates a mock multipart.FileHeader for testing
func createMockFileHeader(filename string, content []byte) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/octet-stream")
	
	part, _ := writer.CreatePart(h)
	part.Write(content)
	writer.Close()
	
	// Parse the multipart data to get a FileHeader
	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(int64(len(content)) + 1024)
	
	if len(form.File["file"]) > 0 {
		return form.File["file"][0]
	}
	
	// Fallback: create a minimal FileHeader
	return &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(content)),
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		bytes.Contains([]byte(s), []byte(substr)))
}
