package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	resendAPIURL = "https://api.resend.com/emails"
)

// ResendProvider implements the Provider interface using Resend
type ResendProvider struct {
	APIKey    string
	FromEmail string
	FromName  string
	client    *http.Client
}

// NewResendProvider creates a new Resend provider from environment variables
func NewResendProvider() *ResendProvider {
	return &ResendProvider{
		APIKey:    os.Getenv("RESEND_API_KEY"),
		FromEmail: os.Getenv("RESEND_FROM_EMAIL"),
		FromName:  os.Getenv("RESEND_FROM_NAME"),
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
func (p *ResendProvider) SendEmail(to, subject, htmlBody string) error {
	if !p.IsConfigured() {
		return fmt.Errorf("Resend provider is not configured")
	}

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
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var resendResp ResendEmailResponse
		if err := json.Unmarshal(body, &resendResp); err != nil {
			return fmt.Errorf("Resend API error: status %d, body: %s", resp.StatusCode, string(body))
		}
		if resendResp.Error.Message != "" {
			return fmt.Errorf("Resend API error: %s", resendResp.Error.Message)
		}
		return fmt.Errorf("Resend API error: status %d", resp.StatusCode)
	}

	return nil
}
