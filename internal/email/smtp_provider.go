package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

// SMTPProvider implements the Provider interface using SMTP
type SMTPProvider struct {
	Host     string
	Port     string
	Username string
	Password string
	FromEmail string
	FromName  string
}

// NewSMTPProvider creates a new SMTP provider from environment variables
func NewSMTPProvider() *SMTPProvider {
	return &SMTPProvider{
		Host:      os.Getenv("SMTP_HOST"),
		Port:      os.Getenv("SMTP_PORT"),
		Username:  os.Getenv("SMTP_USERNAME"),
		Password:  os.Getenv("SMTP_PASSWORD"),
		FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
		FromName:  os.Getenv("SMTP_FROM_NAME"),
	}
}

// IsConfigured checks if the SMTP provider is properly configured
func (p *SMTPProvider) IsConfigured() bool {
	return p.Host != "" && p.Port != "" && p.Username != "" && p.Password != "" && p.FromEmail != ""
}

// GetProviderName returns the provider name for logging
func (p *SMTPProvider) GetProviderName() string {
	return "smtp"
}

// getTLSConfig returns the TLS configuration for SMTP connections
func (p *SMTPProvider) getTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         p.Host,
		InsecureSkipVerify: false, // Always verify certificates in production
	}
}

// SendEmail sends an email using SMTP
func (p *SMTPProvider) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if !p.IsConfigured() {
		return fmt.Errorf("SMTP provider is not configured")
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ctx, span := tracer.Start(ctx, "email.smtp.send", trace.WithAttributes(
		attribute.Int("email.body_size_bytes", len(htmlBody)),
	))
	defer span.End()

	from := p.FromEmail
	if p.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.FromName, p.FromEmail)
	}

	// Build email message
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, htmlBody))

	// Set up authentication
	auth := smtp.PlainAuth("", p.Username, p.Password, p.Host)

	// Connect to SMTP server with TLS
	addr := fmt.Sprintf("%s:%s", p.Host, p.Port)

	// Try to send with TLS
	tlsConfig := p.getTLSConfig()

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// If TLS connection fails, try STARTTLS
		if err := p.sendWithSTARTTLS(addr, auth, p.FromEmail, to, msg); err != nil {
			return telemetry.Fail(span, err, "starttls fallback failed")
		}
		return nil
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.Host)
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to create SMTP client: %w", err), "client creation failed")
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to authenticate: %w", err), "authentication failed")
	}

	if err = client.Mail(p.FromEmail); err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to set sender: %w", err), "set sender failed")
	}

	if err = client.Rcpt(to); err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to set recipient: %w", err), "set recipient failed")
	}

	w, err := client.Data()
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to get data writer: %w", err), "data writer failed")
	}

	_, err = w.Write(msg)
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to write message: %w", err), "write failed")
	}

	err = w.Close()
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to close writer: %w", err), "close writer failed")
	}

	if err := client.Quit(); err != nil {
		return telemetry.Fail(span, err, "quit failed")
	}
	return nil
}

// sendWithSTARTTLS sends email using STARTTLS
func (p *SMTPProvider) sendWithSTARTTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
