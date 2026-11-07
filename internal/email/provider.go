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

// BatchEmailProvider is implemented by providers that can send the same
// subject/body to many recipients in fewer round trips than one call per
// recipient (currently only ResendProvider, via Resend's batch endpoint).
// A caller fanning an announcement out to a whole group checks for this via
// a type assertion on Provider, so a provider that doesn't implement it
// (SMTPProvider, any test mock) falls back to sending one at a time.
type BatchEmailProvider interface {
	// SendBatch sends subject/htmlBody to every address in to. Returns one
	// error per recipient, in the same order as to (nil for a recipient
	// whose send succeeded).
	SendBatch(ctx context.Context, to []string, subject, htmlBody string) []error
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
