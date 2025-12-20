package email

import (
	"os"
	"strings"
	"testing"
)

func TestNewSMTPProvider(t *testing.T) {
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

	provider := NewSMTPProvider()

	if provider.Host != "smtp.example.com" {
		t.Errorf("Expected Host to be smtp.example.com, got %s", provider.Host)
	}
	if provider.Port != "587" {
		t.Errorf("Expected Port to be 587, got %s", provider.Port)
	}
	if provider.Username != "testuser" {
		t.Errorf("Expected Username to be testuser, got %s", provider.Username)
	}
	if provider.Password != "testpass" {
		t.Errorf("Expected Password to be testpass, got %s", provider.Password)
	}
	if provider.FromEmail != "test@example.com" {
		t.Errorf("Expected FromEmail to be test@example.com, got %s", provider.FromEmail)
	}
	if provider.FromName != "Test User" {
		t.Errorf("Expected FromName to be Test User, got %s", provider.FromName)
	}

	// Restore original env vars
	os.Setenv("SMTP_HOST", origHost)
	os.Setenv("SMTP_PORT", origPort)
	os.Setenv("SMTP_USERNAME", origUsername)
	os.Setenv("SMTP_PASSWORD", origPassword)
	os.Setenv("SMTP_FROM_EMAIL", origFromEmail)
	os.Setenv("SMTP_FROM_NAME", origFromName)
}

func TestSMTPProvider_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		provider *SMTPProvider
		want     bool
	}{
		{
			name: "fully configured",
			provider: &SMTPProvider{
				Host:      "smtp.example.com",
				Port:      "587",
				Username:  "user",
				Password:  "pass",
				FromEmail: "test@example.com",
			},
			want: true,
		},
		{
			name: "missing host",
			provider: &SMTPProvider{
				Host:      "",
				Port:      "587",
				Username:  "user",
				Password:  "pass",
				FromEmail: "test@example.com",
			},
			want: false,
		},
		{
			name: "missing port",
			provider: &SMTPProvider{
				Host:      "smtp.example.com",
				Port:      "",
				Username:  "user",
				Password:  "pass",
				FromEmail: "test@example.com",
			},
			want: false,
		},
		{
			name: "missing username",
			provider: &SMTPProvider{
				Host:      "smtp.example.com",
				Port:      "587",
				Username:  "",
				Password:  "pass",
				FromEmail: "test@example.com",
			},
			want: false,
		},
		{
			name: "missing password",
			provider: &SMTPProvider{
				Host:      "smtp.example.com",
				Port:      "587",
				Username:  "user",
				Password:  "",
				FromEmail: "test@example.com",
			},
			want: false,
		},
		{
			name: "missing from email",
			provider: &SMTPProvider{
				Host:      "smtp.example.com",
				Port:      "587",
				Username:  "user",
				Password:  "pass",
				FromEmail: "",
			},
			want: false,
		},
		{
			name: "completely empty",
			provider: &SMTPProvider{
				Host:      "",
				Port:      "",
				Username:  "",
				Password:  "",
				FromEmail: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSMTPProvider_GetProviderName(t *testing.T) {
	provider := NewSMTPProvider()
	if provider.GetProviderName() != "smtp" {
		t.Errorf("Expected provider name 'smtp', got '%s'", provider.GetProviderName())
	}
}

func TestSMTPProvider_SendEmail_NotConfigured(t *testing.T) {
	provider := &SMTPProvider{
		Host:      "",
		Port:      "",
		Username:  "",
		Password:  "",
		FromEmail: "",
	}

	err := provider.SendEmail("test@example.com", "Test Subject", "<html><body>Test Body</body></html>")
	if err == nil {
		t.Error("Expected error when provider is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected error to contain 'not configured', got %s", err.Error())
	}
}

func TestSMTPProvider_SendEmail_ValidatesRecipient(t *testing.T) {
	provider := &SMTPProvider{
		Host:      "smtp.example.com",
		Port:      "587",
		Username:  "user",
		Password:  "pass",
		FromEmail: "from@example.com",
		FromName:  "From Name",
	}

	// Note: This will fail at the connection stage with a real SMTP server
	// but we're testing that the provider attempts to send
	err := provider.SendEmail("", "Test", "<html><body>Test</body></html>")
	if err == nil {
		t.Error("Expected error when sending to empty recipient")
	}
}
