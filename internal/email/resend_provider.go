package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

const (
	defaultResendAPIURL = "https://api.resend.com/emails"
)

// tracer is shared by every Provider implementation in this package (see
// also smtp_provider.go); span names ("email.resend.send",
// "email.smtp.send") distinguish providers, so one tracer per package is
// enough — no need for a differently-named tracer per file.
var tracer = telemetry.Tracer("internal/email")

// ResendProvider implements the Provider interface using Resend
type ResendProvider struct {
	APIKey    string
	FromEmail string
	FromName  string
	client    *http.Client
	apiURL    string // Configurable API URL for testing
}

// NewResendProvider creates a new Resend provider from environment variables
func NewResendProvider() *ResendProvider {
	apiURL := os.Getenv("RESEND_API_URL")
	if apiURL == "" {
		apiURL = defaultResendAPIURL
	}
	
	return &ResendProvider{
		APIKey:    os.Getenv("RESEND_API_KEY"),
		FromEmail: os.Getenv("RESEND_FROM_EMAIL"),
		FromName:  os.Getenv("RESEND_FROM_NAME"),
		apiURL:    apiURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured checks if the Resend provider is properly configured
func (p *ResendProvider) IsConfigured() bool {
	return p.APIKey != "" && p.FromEmail != ""
}

// GetProviderName returns the provider name for logging
func (p *ResendProvider) GetProviderName() string {
	return "resend"
}

// ResendEmailRequest represents the Resend API request structure
type ResendEmailRequest struct {
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// ResendEmailResponse represents the Resend API response structure
type ResendEmailResponse struct {
	ID    string `json:"id"`
	Error struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// SendEmail sends an email using Resend API
func (p *ResendProvider) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if !p.IsConfigured() {
		return fmt.Errorf("Resend provider is not configured")
	}

	ctx, span := tracer.Start(ctx, "email.resend.send", trace.WithAttributes(
		attribute.Int("email.body_size_bytes", len(htmlBody)),
	))
	defer span.End()

	// Construct from address
	from := p.FromEmail
	if p.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.FromName, p.FromEmail)
	}

	// Create request payload
	payload := ResendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to marshal request: %w", err), "failed to marshal request")
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to create request: %w", err), "failed to create request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to send request: %w", err), "request failed")
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return telemetry.Fail(span, fmt.Errorf("failed to read response: %w", err), "failed to read response")
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var resendResp ResendEmailResponse
		if err := json.Unmarshal(body, &resendResp); err != nil {
			return telemetry.Fail(span, fmt.Errorf("Resend API error: status %d, body: %s", resp.StatusCode, string(body)), "non-200 response")
		}
		if resendResp.Error.Message != "" {
			return telemetry.Fail(span, fmt.Errorf("Resend API error: %s", resendResp.Error.Message), "non-200 response")
		}
		return telemetry.Fail(span, fmt.Errorf("Resend API error: status %d", resp.StatusCode), "non-200 response")
	}

	return nil
}
