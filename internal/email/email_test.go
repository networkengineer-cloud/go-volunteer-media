package email

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
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

	service := NewService()

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
			},
			want: true,
		},
		{
			name: "nil provider",
			service: &Service{
				provider: nil,
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
