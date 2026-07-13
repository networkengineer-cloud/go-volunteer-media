package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

// defaultSMTPTimeout bounds the SMTP conversation when ctx carries no
// deadline of its own (e.g. a caller that didn't wrap ctx with a timeout).
const defaultSMTPTimeout = 25 * time.Second

// smtpDeadline returns ctx's own deadline if it has one, else a deadline
// defaultSMTPTimeout from now. net/smtp's Client methods are blocking,
// context-oblivious I/O — the connection's own deadline is the only thing
// that can bound a server that accepts the TCP connection but then stalls
// mid-handshake or mid-conversation.
func smtpDeadline(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(defaultSMTPTimeout)
}

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
	deadline := smtpDeadline(ctx)

	dialer := &tls.Dialer{NetDialer: &net.Dialer{}, Config: tlsConfig}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		// If TLS connection fails, try STARTTLS
		if err := p.sendWithSTARTTLS(ctx, addr, auth, p.FromEmail, to, msg); err != nil {
			return telemetry.Fail(span, err, "starttls fallback failed")
		}
		return nil
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to set connection deadline: %w", err), "deadline failed")
	}

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

// sendWithSTARTTLS sends email using STARTTLS, as a fallback for servers
// that don't accept a direct TLS connection on the configured port. This
// walks the same steps net/smtp.SendMail does internally, but dials via ctx
// (so an attempt to an unreachable host is cancellable) and sets a deadline
// on the connection (so a server that accepts the TCP connection but then
// stalls mid-handshake or mid-conversation can't hang the caller
// indefinitely — net/smtp's blocking Client methods have no ctx-awareness of
// their own).
func (p *SMTPProvider) sendWithSTARTTLS(ctx context.Context, addr string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(smtpDeadline(ctx)); err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, p.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: p.Host}); err != nil {
			return err
		}
	}

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
