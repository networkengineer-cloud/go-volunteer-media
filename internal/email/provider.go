package email

import (
	"fmt"
	"os"
)

// Provider defines the interface that all email providers must implement
// This allows easy swapping between different email services (SMTP, Resend, SendGrid, etc.)
type Provider interface {
	// SendEmail sends an email to a single recipient
	SendEmail(to, subject, htmlBody string) error
	
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
func NewProvider() (Provider, error) {
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
