package email

import (
	"os"
	"testing"
)

func TestNewProvider(t *testing.T) {
	// Save original env var
	origProvider := os.Getenv("EMAIL_PROVIDER")
	defer os.Setenv("EMAIL_PROVIDER", origProvider)

	tests := []struct {
		name          string
		envProvider   string
		wantType      string
		wantErr       bool
	}{
		{
			name:        "default to SMTP when not set",
			envProvider: "",
			wantType:    "smtp",
			wantErr:     false,
		},
		{
			name:        "explicitly set to SMTP",
			envProvider: "smtp",
			wantType:    "smtp",
			wantErr:     false,
		},
		{
			name:        "set to Resend",
			envProvider: "resend",
			wantType:    "resend",
			wantErr:     false,
		},
		{
			name:        "unsupported provider",
			envProvider: "sendgrid",
			wantType:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EMAIL_PROVIDER", tt.envProvider)

			provider, err := NewProvider()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected provider but got nil")
				return
			}

			if provider.GetProviderName() != tt.wantType {
				t.Errorf("Expected provider type %s, got %s", tt.wantType, provider.GetProviderName())
			}
		})
	}
}

func TestProviderInterface(t *testing.T) {
	// Test that all providers implement the interface correctly
	providers := []struct {
		name     string
		provider Provider
	}{
		{
			name:     "SMTP",
			provider: NewSMTPProvider(),
		},
		{
			name:     "Resend",
			provider: NewResendProvider(),
		},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			// Test that IsConfigured doesn't panic
			_ = tt.provider.IsConfigured()

			// Test that GetProviderName returns non-empty string
			if tt.provider.GetProviderName() == "" {
				t.Error("GetProviderName should return non-empty string")
			}

			// Test that SendEmail with unconfigured provider returns error
			if !tt.provider.IsConfigured() {
				err := tt.provider.SendEmail("test@example.com", "Test", "<html><body>Test</body></html>")
				if err == nil {
					t.Error("Expected error when sending email with unconfigured provider")
				}
			}
		})
	}
}
