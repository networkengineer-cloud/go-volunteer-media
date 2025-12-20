package email

import (
	"context"
	"fmt"
	"os"
)

// Provider defines the interface that all email providers must implement
// This allows easy swapping between different email services (SMTP, Resend, SendGrid, etc.)
type Provider interface {
	// SendEmail sends an email to a single recipient
	// Context can be used for timeouts, cancellation, and tracing
	SendEmail(ctx context.Context, to, subject, htmlBody string) error
	
	// IsConfigured returns true if the provider is properly configured
	IsConfigured() bool
	
	// GetProviderName returns the name of the provider for logging
	GetProviderName() string
}

// ProviderType represents the type of email provider
type ProviderType string

const (
	ProviderTypeSMTP   ProviderType = "smtp"
	ProviderTypeResend ProviderType = "resend"
)

// NewProvider creates an email provider based on environment configuration
// Returns nil provider if EMAIL_ENABLED is set to "false" or "0"
func NewProvider() (Provider, error) {
	// Check if email is explicitly disabled
	emailEnabled := os.Getenv("EMAIL_ENABLED")
	if emailEnabled == "false" || emailEnabled == "0" {
		return nil, nil // Email disabled - return nil provider without error
	}

	providerType := os.Getenv("EMAIL_PROVIDER")
	if providerType == "" {
		providerType = string(ProviderTypeSMTP) // Default to SMTP for backwards compatibility
	}

	switch ProviderType(providerType) {
	case ProviderTypeSMTP:
		return NewSMTPProvider(), nil
	case ProviderTypeResend:
		return NewResendProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", providerType)
	}
}
