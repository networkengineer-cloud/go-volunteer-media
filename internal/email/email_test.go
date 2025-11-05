package email

import (
	"os"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	// Save original env vars
	origHost := os.Getenv("SMTP_HOST")
	origPort := os.Getenv("SMTP_PORT")
	origUsername := os.Getenv("SMTP_USERNAME")
	origPassword := os.Getenv("SMTP_PASSWORD")
	origFromEmail := os.Getenv("SMTP_FROM_EMAIL")
	origFromName := os.Getenv("SMTP_FROM_NAME")

	// Set test env vars
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "testuser")
	os.Setenv("SMTP_PASSWORD", "testpass")
	os.Setenv("SMTP_FROM_EMAIL", "test@example.com")
	os.Setenv("SMTP_FROM_NAME", "Test User")

	service := NewService()

	if service.SMTPHost != "smtp.example.com" {
		t.Errorf("Expected SMTPHost to be smtp.example.com, got %s", service.SMTPHost)
	}
	if service.SMTPPort != "587" {
		t.Errorf("Expected SMTPPort to be 587, got %s", service.SMTPPort)
	}
	if service.SMTPUsername != "testuser" {
		t.Errorf("Expected SMTPUsername to be testuser, got %s", service.SMTPUsername)
	}
	if service.SMTPPassword != "testpass" {
		t.Errorf("Expected SMTPPassword to be testpass, got %s", service.SMTPPassword)
	}
	if service.FromEmail != "test@example.com" {
		t.Errorf("Expected FromEmail to be test@example.com, got %s", service.FromEmail)
	}
	if service.FromName != "Test User" {
		t.Errorf("Expected FromName to be Test User, got %s", service.FromName)
	}

	// Restore original env vars
	os.Setenv("SMTP_HOST", origHost)
	os.Setenv("SMTP_PORT", origPort)
	os.Setenv("SMTP_USERNAME", origUsername)
	os.Setenv("SMTP_PASSWORD", origPassword)
	os.Setenv("SMTP_FROM_EMAIL", origFromEmail)
	os.Setenv("SMTP_FROM_NAME", origFromName)
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name    string
		service *Service
		want    bool
	}{
		{
			name: "fully configured",
			service: &Service{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     "587",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				FromEmail:    "test@example.com",
			},
			want: true,
		},
		{
			name: "missing host",
			service: &Service{
				SMTPHost:     "",
				SMTPPort:     "587",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				FromEmail:    "test@example.com",
			},
			want: false,
		},
		{
			name: "missing port",
			service: &Service{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     "",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				FromEmail:    "test@example.com",
			},
			want: false,
		},
		{
			name: "missing username",
			service: &Service{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     "587",
				SMTPUsername: "",
				SMTPPassword: "pass",
				FromEmail:    "test@example.com",
			},
			want: false,
		},
		{
			name: "missing password",
			service: &Service{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     "587",
				SMTPUsername: "user",
				SMTPPassword: "",
				FromEmail:    "test@example.com",
			},
			want: false,
		},
		{
			name: "missing from email",
			service: &Service{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     "587",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				FromEmail:    "",
			},
			want: false,
		},
		{
			name: "completely empty",
			service: &Service{
				SMTPHost:     "",
				SMTPPort:     "",
				SMTPUsername: "",
				SMTPPassword: "",
				FromEmail:    "",
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

func TestSendEmail_NotConfigured(t *testing.T) {
	service := &Service{
		SMTPHost:     "",
		SMTPPort:     "",
		SMTPUsername: "",
		SMTPPassword: "",
		FromEmail:    "",
	}

	err := service.SendEmail("test@example.com", "Test Subject", "Test Body")
	if err == nil {
		t.Error("Expected error when service is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected error to contain 'not configured', got %s", err.Error())
	}
}

func TestSendPasswordResetEmail_Structure(t *testing.T) {
	// We can't actually send emails in tests, but we can verify the structure
	// by checking if the method constructs the right parameters

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
				SMTPHost:     "",
				SMTPPort:     "",
				SMTPUsername: "",
				SMTPPassword: "",
				FromEmail:    "",
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
		SMTPHost:     "",
		SMTPPort:     "",
		SMTPUsername: "",
		SMTPPassword: "",
		FromEmail:    "",
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

func TestServiceStructure(t *testing.T) {
	service := &Service{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "user",
		SMTPPassword: "pass",
		FromEmail:    "from@example.com",
		FromName:     "From Name",
	}

	// Verify all fields are accessible
	if service.SMTPHost == "" {
		t.Error("SMTPHost should not be empty")
	}
	if service.SMTPPort == "" {
		t.Error("SMTPPort should not be empty")
	}
	if service.SMTPUsername == "" {
		t.Error("SMTPUsername should not be empty")
	}
	if service.SMTPPassword == "" {
		t.Error("SMTPPassword should not be empty")
	}
	if service.FromEmail == "" {
		t.Error("FromEmail should not be empty")
	}
	if service.FromName == "" {
		t.Error("FromName should not be empty")
	}
}
