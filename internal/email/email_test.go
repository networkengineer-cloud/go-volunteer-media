package email

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewService(t *testing.T) {
	// Save original env vars
	origProvider := os.Getenv("EMAIL_PROVIDER")
	origHost := os.Getenv("SMTP_HOST")
	origPort := os.Getenv("SMTP_PORT")
	origUsername := os.Getenv("SMTP_USERNAME")
	origPassword := os.Getenv("SMTP_PASSWORD")
	origFromEmail := os.Getenv("SMTP_FROM_EMAIL")
	origFromName := os.Getenv("SMTP_FROM_NAME")

	// Set test env vars for SMTP provider
	os.Setenv("EMAIL_PROVIDER", "smtp")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "testuser")
	os.Setenv("SMTP_PASSWORD", "testpass")
	os.Setenv("SMTP_FROM_EMAIL", "test@example.com")
	os.Setenv("SMTP_FROM_NAME", "Test User")

	service := NewService(nil)

	if service == nil {
		t.Fatal("Expected service to be non-nil")
	}

	if service.provider == nil {
		t.Error("Expected provider to be non-nil")
	}

	if !service.IsConfigured() {
		t.Error("Expected service to be configured")
	}

	// Restore original env vars
	os.Setenv("EMAIL_PROVIDER", origProvider)
	os.Setenv("SMTP_HOST", origHost)
	os.Setenv("SMTP_PORT", origPort)
	os.Setenv("SMTP_USERNAME", origUsername)
	os.Setenv("SMTP_PASSWORD", origPassword)
	os.Setenv("SMTP_FROM_EMAIL", origFromEmail)
	os.Setenv("SMTP_FROM_NAME", origFromName)
}

func TestService_IsConfigured(t *testing.T) {
	tests := []struct {
		name    string
		service *Service
		want    bool
	}{
		{
			name: "configured service",
			service: &Service{
				provider: &SMTPProvider{
					Host:      "smtp.example.com",
					Port:      "587",
					Username:  "user",
					Password:  "pass",
					FromEmail: "test@example.com",
				},
				db: nil,
			},
			want: true,
		},
		{
			name: "nil provider",
			service: &Service{
				provider: nil,
				db:       nil,
			},
			want: false,
		},
		{
			name: "unconfigured provider",
			service: &Service{
				provider: &SMTPProvider{
					Host:      "",
					Port:      "",
					Username:  "",
					Password:  "",
					FromEmail: "",
				},
				db: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.service.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_SendEmail_NotConfigured(t *testing.T) {
	service := &Service{
		provider: nil,
		db:       nil,
	}

	err := service.SendEmail("test@example.com", "Test Subject", "<html><body>Test Body</body></html>")
	if err == nil {
		t.Error("Expected error when service is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected error to contain 'not configured', got %s", err.Error())
	}
}

func TestSendPasswordResetEmail_Structure(t *testing.T) {
	// Save original env var
	origFrontendURL := os.Getenv("FRONTEND_URL")

	tests := []struct {
		name        string
		frontendURL string
		wantBaseURL string
	}{
		{
			name:        "with custom frontend URL",
			frontendURL: "https://example.com",
			wantBaseURL: "https://example.com",
		},
		{
			name:        "with default frontend URL",
			frontendURL: "",
			wantBaseURL: "http://localhost:5173",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("FRONTEND_URL", tt.frontendURL)

			// Create unconfigured service to avoid actual email sending
			service := &Service{
				provider: nil,
				db:       nil,
			}

			err := service.SendPasswordResetEmail("test@example.com", "testuser", "test-token-123")

			// Should get "not configured" error since service is not configured
			if err == nil {
				t.Error("Expected error for unconfigured service")
			}
			if !strings.Contains(err.Error(), "not configured") {
				t.Errorf("Expected 'not configured' error, got: %v", err)
			}
		})
	}

	// Restore original env var
	os.Setenv("FRONTEND_URL", origFrontendURL)
}

func TestSendAnnouncementEmail_Structure(t *testing.T) {
	// Create unconfigured service to avoid actual email sending
	service := &Service{
		provider: nil,
		db:       nil,
	}

	err := service.SendAnnouncementEmail(
		"test@example.com",
		"Test Announcement",
		"This is a test announcement\nwith multiple lines",
	)

	// Should get "not configured" error since service is not configured
	if err == nil {
		t.Error("Expected error for unconfigured service")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected 'not configured' error, got: %v", err)
	}
}

func TestService_SendPasswordResetEmail_WithConfiguredProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := &mockEmailProvider{
		configured: true,
		sentEmails: []sentEmail{},
	}

	service := &Service{
		provider: mockProvider,
		db:       nil,
	}

	// Save and set frontend URL
	origFrontendURL := os.Getenv("FRONTEND_URL")
	os.Setenv("FRONTEND_URL", "https://test.com")
	defer os.Setenv("FRONTEND_URL", origFrontendURL)

	err := service.SendPasswordResetEmail("user@example.com", "testuser", "token123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(mockProvider.sentEmails) != 1 {
		t.Fatalf("Expected 1 email to be sent, got %d", len(mockProvider.sentEmails))
	}

	email := mockProvider.sentEmails[0]
	if email.to != "user@example.com" {
		t.Errorf("Expected to 'user@example.com', got '%s'", email.to)
	}
	if !strings.Contains(email.subject, "Password Reset") {
		t.Errorf("Expected subject to contain 'Password Reset', got '%s'", email.subject)
	}
	if !strings.Contains(email.body, "testuser") {
		t.Error("Expected body to contain username")
	}
	if !strings.Contains(email.body, "token123") {
		t.Error("Expected body to contain token")
	}
	if !strings.Contains(email.body, "https://test.com/reset-password?token=token123") {
		t.Error("Expected body to contain reset link")
	}
}

func TestService_SendAnnouncementEmail_WithConfiguredProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := &mockEmailProvider{
		configured: true,
		sentEmails: []sentEmail{},
	}

	service := &Service{
		provider: mockProvider,
		db:       nil,
	}

	err := service.SendAnnouncementEmail("user@example.com", "Important Notice", "This is the content\nLine 2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(mockProvider.sentEmails) != 1 {
		t.Fatalf("Expected 1 email to be sent, got %d", len(mockProvider.sentEmails))
	}

	email := mockProvider.sentEmails[0]
	if email.to != "user@example.com" {
		t.Errorf("Expected to 'user@example.com', got '%s'", email.to)
	}
	if !strings.Contains(email.subject, "Important Notice") {
		t.Errorf("Expected subject to contain 'Important Notice', got '%s'", email.subject)
	}
	if !strings.Contains(email.body, "Important Notice") {
		t.Error("Expected body to contain title")
	}
	if !strings.Contains(email.body, "This is the content") {
		t.Error("Expected body to contain content")
	}
	// HTML should convert newlines to <br>
	if !strings.Contains(email.body, "<br>") {
		t.Error("Expected body to contain HTML line breaks")
	}
}

// Mock provider for testing
type mockEmailProvider struct {
	configured bool
	sentEmails []sentEmail
}

type sentEmail struct {
	to      string
	subject string
	body    string
}

func (m *mockEmailProvider) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if !m.configured {
		return fmt.Errorf("mock provider not configured")
	}
	m.sentEmails = append(m.sentEmails, sentEmail{
		to:      to,
		subject: subject,
		body:    htmlBody,
	})
	return nil
}

func (m *mockEmailProvider) IsConfigured() bool {
	return m.configured
}

func (m *mockEmailProvider) GetProviderName() string {
	return "mock"
}

// setupTestDB creates an in-memory SQLite database for integration testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate the SiteSetting model
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// TestGetSiteName_WithDatabase tests getSiteName with a real database
func TestGetSiteName_WithDatabase(t *testing.T) {
	t.Run("fetches site name from database", func(t *testing.T) {
		db := setupTestDB(t)

		// Insert test setting
		setting := models.SiteSetting{
			Key:   "site_name",
			Value: "Test Organization",
		}
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("Failed to create test setting: %v", err)
		}

		mockProvider := &mockEmailProvider{configured: true}
		service := NewServiceWithProvider(mockProvider, db)

		siteName := service.getSiteName()

		if siteName != "Test Organization" {
			t.Errorf("Expected site name 'Test Organization', got '%s'", siteName)
		}
	})

	t.Run("uses default when database returns empty", func(t *testing.T) {
		db := setupTestDB(t)
		// No settings in database

		mockProvider := &mockEmailProvider{configured: true}
		service := NewServiceWithProvider(mockProvider, db)

		siteName := service.getSiteName()

		if siteName != models.DefaultSiteName {
			t.Errorf("Expected default site name '%s', got '%s'", models.DefaultSiteName, siteName)
		}
	})

	t.Run("uses default when db is nil", func(t *testing.T) {
		mockProvider := &mockEmailProvider{configured: true}
		service := NewServiceWithProvider(mockProvider, nil)

		siteName := service.getSiteName()

		if siteName != models.DefaultSiteName {
			t.Errorf("Expected default site name '%s', got '%s'", models.DefaultSiteName, siteName)
		}
	})

	t.Run("cache works across multiple calls", func(t *testing.T) {
		db := setupTestDB(t)

		// Insert test setting
		setting := models.SiteSetting{
			Key:   "site_name",
			Value: "Cached Name",
		}
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("Failed to create test setting: %v", err)
		}

		mockProvider := &mockEmailProvider{configured: true}
		service := NewServiceWithProvider(mockProvider, db)

		// First call - should fetch from DB and cache
		siteName1 := service.getSiteName()
		if siteName1 != "Cached Name" {
			t.Errorf("Expected 'Cached Name', got '%s'", siteName1)
		}

		// Update the database value
		db.Model(&models.SiteSetting{}).Where("key = ?", "site_name").Update("value", "Updated Name")

		// Second call - should still return cached value (not expired yet)
		siteName2 := service.getSiteName()
		if siteName2 != "Cached Name" {
			t.Errorf("Expected cached 'Cached Name', got '%s'", siteName2)
		}
	})

	t.Run("cache expires after TTL", func(t *testing.T) {
		db := setupTestDB(t)

		// Insert test setting
		setting := models.SiteSetting{
			Key:   "site_name",
			Value: "Original Name",
		}
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("Failed to create test setting: %v", err)
		}

		mockProvider := &mockEmailProvider{configured: true}
		service := NewServiceWithProvider(mockProvider, db)

		// First call - should fetch from DB and cache
		siteName1 := service.getSiteName()
		if siteName1 != "Original Name" {
			t.Errorf("Expected 'Original Name', got '%s'", siteName1)
		}

		// Update the database value
		db.Model(&models.SiteSetting{}).Where("key = ?", "site_name").Update("value", "Refreshed Name")

		// Manually expire the cache
		service.cacheMu.Lock()
		service.cacheExpiry = time.Now().Add(-1 * time.Minute)
		service.cacheMu.Unlock()

		// Next call should refresh cache and get new value
		siteName2 := service.getSiteName()
		if siteName2 != "Refreshed Name" {
			t.Errorf("Expected refreshed 'Refreshed Name', got '%s'", siteName2)
		}
	})
}

// TestEmailTemplates_UseDynamicSiteName tests that emails use the dynamic site name
func TestEmailTemplates_UseDynamicSiteName(t *testing.T) {
	db := setupTestDB(t)

	// Insert custom site name
	setting := models.SiteSetting{
		Key:   "site_name",
		Value: "Custom Shelter",
	}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatalf("Failed to create test setting: %v", err)
	}

	mockProvider := &mockEmailProvider{configured: true}
	service := NewServiceWithProvider(mockProvider, db)

	// Set frontend URL for tests
	origFrontendURL := os.Getenv("FRONTEND_URL")
	os.Setenv("FRONTEND_URL", "http://test.example.com")
	defer os.Setenv("FRONTEND_URL", origFrontendURL)

	t.Run("password reset email uses custom site name", func(t *testing.T) {
		mockProvider.sentEmails = nil // Clear sent emails

		err := service.SendPasswordResetEmail("user@example.com", "testuser", "reset-token-123")
		if err != nil {
			t.Fatalf("Failed to send password reset email: %v", err)
		}

		if len(mockProvider.sentEmails) != 1 {
			t.Fatalf("Expected 1 email, got %d", len(mockProvider.sentEmails))
		}

		email := mockProvider.sentEmails[0]

		// Check subject contains custom site name
		if !strings.Contains(email.subject, "Custom Shelter") {
			t.Errorf("Expected subject to contain 'Custom Shelter', got: %s", email.subject)
		}

		// Check body contains custom site name
		if !strings.Contains(email.body, "Custom Shelter") {
			t.Errorf("Expected body to contain 'Custom Shelter'")
		}
	})

	t.Run("password setup email uses custom site name", func(t *testing.T) {
		mockProvider.sentEmails = nil // Clear sent emails

		err := service.SendPasswordSetupEmail("user@example.com", "newuser", "setup-token-456")
		if err != nil {
			t.Fatalf("Failed to send password setup email: %v", err)
		}

		if len(mockProvider.sentEmails) != 1 {
			t.Fatalf("Expected 1 email, got %d", len(mockProvider.sentEmails))
		}

		email := mockProvider.sentEmails[0]

		// Check subject contains custom site name
		if !strings.Contains(email.subject, "Custom Shelter") {
			t.Errorf("Expected subject to contain 'Custom Shelter', got: %s", email.subject)
		}

		// Check body contains custom site name (appears multiple times)
		bodyCount := strings.Count(email.body, "Custom Shelter")
		if bodyCount < 2 {
			t.Errorf("Expected body to contain 'Custom Shelter' at least 2 times, found %d", bodyCount)
		}
	})

	t.Run("announcement email uses custom site name", func(t *testing.T) {
		mockProvider.sentEmails = nil // Clear sent emails

		err := service.SendAnnouncementEmail("user@example.com", "Important Update", "This is the announcement content")
		if err != nil {
			t.Fatalf("Failed to send announcement email: %v", err)
		}

		if len(mockProvider.sentEmails) != 1 {
			t.Fatalf("Expected 1 email, got %d", len(mockProvider.sentEmails))
		}

		email := mockProvider.sentEmails[0]

		// Check subject contains custom site name
		if !strings.Contains(email.subject, "Custom Shelter") {
			t.Errorf("Expected subject to contain 'Custom Shelter', got: %s", email.subject)
		}

		// Check footer contains custom site name
		if !strings.Contains(email.body, "© Custom Shelter") {
			t.Errorf("Expected footer to contain '© Custom Shelter'")
		}
	})
}
