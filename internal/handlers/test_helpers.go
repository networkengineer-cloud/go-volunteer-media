package handlers

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database for testing with all models migrated
func SetupTestDB(t *testing.T) *gorm.DB {
	// Set JWT_SECRET for testing - must be random and secure for validation to pass
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")

	// IMPORTANT: SQLite in-memory databases are per-connection.
	// GORM's connection pool may open multiple connections, which can lead to
	// flaky "no such table" errors if migrations run on one connection and
	// queries execute on another.
	// We force a single connection for deterministic tests.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Run migrations for all models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Animal{},
		&models.Update{},
		&models.Announcement{},
		&models.CommentTag{},
		&models.AnimalComment{},
		&models.SiteSetting{},
		&models.Protocol{},
		&models.AnimalTag{},
		&models.AnimalNameHistory{},
	)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// CreateTestUser creates a user in the test database
func CreateTestUser(t *testing.T, db *gorm.DB, username, email, password string, isAdmin bool) *models.User {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		IsAdmin:  isAdmin,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateTestGroup creates a group in the test database
func CreateTestGroup(t *testing.T, db *gorm.DB, name, description string) *models.Group {
	group := &models.Group{
		Name:        name,
		Description: description,
	}

	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create test group: %v", err)
	}

	return group
}

// CreateTestAnimal creates an animal in the test database
func CreateTestAnimal(t *testing.T, db *gorm.DB, groupID uint, name, species string) *models.Animal {
	animal := &models.Animal{
		GroupID: groupID,
		Name:    name,
		Species: species,
		Status:  "available",
	}

	if err := db.Create(animal).Error; err != nil {
		t.Fatalf("Failed to create test animal: %v", err)
	}

	return animal
}

// AddUserToGroupWithAdmin adds a user to a group and optionally makes them a group admin
func AddUserToGroupWithAdmin(t *testing.T, db *gorm.DB, userID, groupID uint, isGroupAdmin bool) {
	userGroup := &models.UserGroup{
		UserID:       userID,
		GroupID:      groupID,
		IsGroupAdmin: isGroupAdmin,
	}

	if err := db.Create(userGroup).Error; err != nil {
		t.Fatalf("Failed to add user to group: %v", err)
	}
}

// minimalPNG is a valid PNG header sufficient to pass ValidateImageUpload without a real image.
var minimalPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

// minimalPDF is a valid PDF header sufficient to pass ValidateDocumentUpload without a real PDF.
var minimalPDF = []byte{'%', 'P', 'D', 'F', '-', '1', '.', '4', '\n'}

// mockConverter is a test double for convert.Converter.
// Set ConvertErr to simulate a conversion failure.
// When ConvertErr is nil, ToPDF returns minimalPDF bytes.
type mockConverter struct {
	ConvertErr error
}

func (m *mockConverter) ToPDF(_ context.Context, _ []byte, _ string) ([]byte, error) {
	if m.ConvertErr != nil {
		return nil, m.ConvertErr
	}
	return minimalPDF, nil
}

// mockStorageProvider is a test double for storage.Provider.
// Set ProviderName to control what Name() returns (default: "mock").
// Set UploadImageErr to fail every call.
// Set UploadImageErrForMimeType to fail uploads for a specific MIME type
// (e.g. "image/jpeg" for thumbnail, "video/mp4" for video) — safe for
// concurrent callers because it reads a pre-set map without mutation.
// Each successful call returns a unique identifier ("test-uuid-N").
// DeletedBlobs records every identifier passed to DeleteImage.
type mockStorageProvider struct {
	ProviderName             string
	UploadImageErr           error
	UploadImageErrForMimeType map[string]error // mime type → error; safe for concurrent use
	UploadDocumentErr        error
	GetImageData             []byte
	GetImageMime             string
	GetImageErr              error
	LastMimeType             string
	DeletedBlobs             []string
	mu                       sync.Mutex
	uploadCallCount          int
}

func (m *mockStorageProvider) Name() string {
	if m.ProviderName != "" {
		return m.ProviderName
	}
	return "mock"
}
func (m *mockStorageProvider) UploadImage(_ context.Context, _ []byte, mimeType string, _ map[string]string) (string, string, string, error) {
	if err, ok := m.UploadImageErrForMimeType[mimeType]; ok {
		return "", "", "", err
	}
	if m.UploadImageErr != nil {
		return "", "", "", m.UploadImageErr
	}
	m.mu.Lock()
	m.uploadCallCount++
	id := fmt.Sprintf("test-uuid-%d", m.uploadCallCount)
	m.mu.Unlock()
	return "/api/images/" + id, id, ".png", nil
}
func (m *mockStorageProvider) UploadDocument(_ context.Context, _ []byte, _, _ string) (string, string, string, error) {
	if m.UploadDocumentErr != nil {
		return "", "", "", m.UploadDocumentErr
	}
	return "/api/documents/test-uuid", "test-uuid", ".pdf", nil
}
func (m *mockStorageProvider) GetImage(_ context.Context, _ string) ([]byte, string, error) {
	if m.GetImageErr != nil {
		return nil, "", m.GetImageErr
	}
	return m.GetImageData, m.GetImageMime, nil
}
func (m *mockStorageProvider) GetDocument(_ context.Context, _ string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *mockStorageProvider) DeleteImage(_ context.Context, identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeletedBlobs = append(m.DeletedBlobs, identifier)
	return nil
}
func (m *mockStorageProvider) DeleteDocument(_ context.Context, _ string) error { return nil }
func (m *mockStorageProvider) GetImageURL(_ string) string                      { return "" }
func (m *mockStorageProvider) GetDocumentURL(_ string) string                   { return "" }

// itoa converts a uint to its decimal string representation for URL construction in tests.
func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
}

// createImageMultipartRequest builds a multipart/form-data POST request containing one image file.
func createImageMultipartRequest(t *testing.T, fieldName, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

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
