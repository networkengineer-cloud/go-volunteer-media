package email

import (
	"context"
	"os"
	"testing"
)

func TestNewProvider(t *testing.T) {
	// Save original env vars
	origProvider := os.Getenv("EMAIL_PROVIDER")
	origEnabled := os.Getenv("EMAIL_ENABLED")
	defer func() {
		os.Setenv("EMAIL_PROVIDER", origProvider)
		os.Setenv("EMAIL_ENABLED", origEnabled)
	}()

	tests := []struct {
		name          string
		envProvider   string
		envEnabled    string
		wantType      string
		wantNil       bool
		wantErr       bool
	}{
		{
			name:        "default to SMTP when not set",
			envProvider: "",
			envEnabled:  "",
			wantType:    "smtp",
			wantNil:     false,
			wantErr:     false,
		},
		{
			name:        "explicitly set to SMTP",
			envProvider: "smtp",
			envEnabled:  "",
			wantType:    "smtp",
			wantNil:     false,
			wantErr:     false,
		},
		{
			name:        "set to Resend",
			envProvider: "resend",
			envEnabled:  "",
			wantType:    "resend",
			wantNil:     false,
			wantErr:     false,
		},
		{
			name:        "email disabled with false",
			envProvider: "smtp",
			envEnabled:  "false",
			wantType:    "",
			wantNil:     true,
			wantErr:     false,
		},
		{
			name:        "email disabled with 0",
			envProvider: "resend",
			envEnabled:  "0",
			wantType:    "",
			wantNil:     true,
			wantErr:     false,
		},
		{
			name:        "unsupported provider",
			envProvider: "sendgrid",
			envEnabled:  "",
			wantType:    "",
			wantNil:     false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EMAIL_PROVIDER", tt.envProvider)
			os.Setenv("EMAIL_ENABLED", tt.envEnabled)

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

			if tt.wantNil {
				if provider != nil {
					t.Error("Expected nil provider when email is disabled")
				}
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
				ctx := context.Background()
				err := tt.provider.SendEmail(ctx, "test@example.com", "Test", "<html><body>Test</body></html>")
				if err == nil {
					t.Error("Expected error when sending email with unconfigured provider")
				}
			}
		})
	}
}
