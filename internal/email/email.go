package email

import (
	"context"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"time"
)

// emailRegex is a basic RFC 5322 compliant email validation pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Service represents an email service that uses a Provider for sending emails
type Service struct {
	provider Provider
}

// NewService creates a new email service using the configured provider
func NewService() *Service {
	provider, err := NewProvider()
	if err != nil {
		// If provider creation fails, return a service with nil provider
		// This allows the application to start even if email is misconfigured
		return &Service{provider: nil}
	}
	return &Service{provider: provider}
}

// NewServiceWithProvider creates a new email service with a specific provider (for testing)
func NewServiceWithProvider(provider Provider) *Service {
	return &Service{provider: provider}
}

// IsConfigured checks if the email service is properly configured
func (s *Service) IsConfigured() bool {
	return s.provider != nil && s.provider.IsConfigured()
}

// isValidEmail validates an email address using basic RFC 5322 rules
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// SendEmail sends an email using the configured provider
func (s *Service) SendEmail(to, subject, htmlBody string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email service is not configured")
	}

	// Validate email address before attempting to send
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.provider.SendEmail(ctx, to, subject, htmlBody)
}

// SendPasswordResetEmail sends a password reset email
func (s *Service) SendPasswordResetEmail(to, username, resetToken string) error {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	subject := "Password Reset Request - Haws Volunteers"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #0e6c55; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f8fafc; }
        .button { display: inline-block; padding: 12px 24px; background-color: #0e6c55; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hello %s,</p>
            <p>We received a request to reset your password for your Haws Volunteers account.</p>
            <p>Click the button below to reset your password:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #0e6c55;">%s</p>
            <p><strong>This link will expire in 1 hour.</strong></p>
            <p>If you didn't request a password reset, you can safely ignore this email.</p>
        </div>
        <div class="footer">
            <p>© Haws Volunteers - This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
`, username, resetLink, resetLink)

	return s.SendEmail(to, subject, body)
}

// SendPasswordSetupEmail sends a password setup email for new user invitations
func (s *Service) SendPasswordSetupEmail(to, username, setupToken string) error {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	setupLink := fmt.Sprintf("%s/setup-password?token=%s", baseURL, setupToken)

	subject := "Welcome to Haws Volunteers - Set Your Password"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #0e6c55; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f8fafc; }
        .button { display: inline-block; padding: 12px 24px; background-color: #0e6c55; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .welcome { font-size: 18px; font-weight: bold; color: #0e6c55; margin-bottom: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Haws Volunteers!</h1>
        </div>
        <div class="content">
            <p class="welcome">Hello %s,</p>
            <p>Your account has been created for Haws Volunteers. We're excited to have you join our team!</p>
            <p>To get started, please click the button below to set your password:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Set Your Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #0e6c55;">%s</p>
            <p><strong>This link will expire in 24 hours.</strong></p>
            <p>Once you've set your password, you'll be able to sign in and start contributing to our mission of helping animals in need.</p>
            <p>If you have any questions or didn't expect this invitation, please contact your administrator.</p>
        </div>
        <div class="footer">
            <p>© Haws Volunteers - This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
`, username, setupLink, setupLink)

	return s.SendEmail(to, subject, body)
}

// SendAnnouncementEmail sends an announcement email
func (s *Service) SendAnnouncementEmail(to, title, content string) error {
	subject := fmt.Sprintf("Announcement: %s - Haws Volunteers", title)

	// Escape HTML in title and convert newlines to HTML line breaks in content
	escapedTitle := html.EscapeString(title)
	htmlContent := strings.ReplaceAll(html.EscapeString(content), "\n", "<br>")

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #0e6c55; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f8fafc; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            %s
        </div>
        <div class="footer">
            <p>© Haws Volunteers - You're receiving this because you opted in to email notifications.</p>
            <p>You can manage your email preferences in your account settings.</p>
        </div>
    </div>
</body>
</html>
`, escapedTitle, htmlContent)

	return s.SendEmail(to, subject, body)
}
