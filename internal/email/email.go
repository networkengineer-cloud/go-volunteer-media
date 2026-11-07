package email

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
	"gorm.io/gorm"
)

// announcementRecipients counts every recipient of a group announcement by
// final outcome (after batching, retries and fallback). The incident that
// prompted batching was one member silently missing an email; a non-zero
// outcome=failed rate is that, made visible and alertable.
var announcementRecipients = telemetry.NewInstrument("email.announcement.recipients", func() (metric.Int64Counter, error) {
	return telemetry.Meter("internal/email").Int64Counter(
		"email.announcement.recipients",
		metric.WithDescription("Group announcement recipients by final send outcome"),
		metric.WithUnit("{recipient}"),
	)
})

// recordAnnouncementOutcomes adds one count per recipient to
// announcementRecipients, split by outcome.
func recordAnnouncementOutcomes(ctx context.Context, provider string, errs []error) {
	var sent, failed int64
	for _, err := range errs {
		if err == nil {
			sent++
		} else {
			failed++
		}
	}
	if sent > 0 {
		announcementRecipients.Add(ctx, sent, metric.WithAttributes(
			attribute.String("email.provider", provider), attribute.String("outcome", "sent")))
	}
	if failed > 0 {
		announcementRecipients.Add(ctx, failed, metric.WithAttributes(
			attribute.String("email.provider", provider), attribute.String("outcome", "failed")))
	}
}

// emailRegex is a basic RFC 5322 compliant email validation pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Service represents an email service that uses a Provider for sending emails
type Service struct {
	provider      Provider
	db            *gorm.DB
	settingsCache map[string]string
	cacheMu       sync.RWMutex
	cacheExpiry   time.Time
	refreshing    atomic.Bool
}

// NewService creates a new email service using the configured provider
func NewService(db *gorm.DB) *Service {
	provider, err := NewProvider()
	if err != nil {
		// If provider creation fails, return a service with nil provider
		// This allows the application to start even if email is misconfigured
		return &Service{provider: nil, db: db}
	}
	s := &Service{provider: provider, db: db}
	// Preload cache synchronously on initialization
	if db != nil {
		s.refreshSettingsCache()
	}
	return s
}

// NewServiceWithProvider creates a new email service with a specific provider (for testing)
// Breaking change (v2): Added db parameter for dynamic site name fetching.
// External callers (if any) must update function signature.
func NewServiceWithProvider(provider Provider, db *gorm.DB) *Service {
	s := &Service{provider: provider, db: db}
	// Preload cache synchronously on initialization
	if db != nil {
		s.refreshSettingsCache()
	}
	return s
}

// IsConfigured checks if the email service is properly configured
func (s *Service) IsConfigured() bool {
	return s.provider != nil && s.provider.IsConfigured()
}

// isValidEmail validates an email address using basic RFC 5322 rules
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// refreshSettingsCache fetches all site settings from the database and caches them
// with a 5-minute TTL. Called on service initialization and when cache expires.
func (s *Service) refreshSettingsCache() {
	if s.db == nil {
		return
	}

	var settings []models.SiteSetting
	if err := s.db.Find(&settings).Error; err != nil {
		logging.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to refresh site settings cache", err)
		// On error, keep existing cache or leave empty
		return
	}

	// Build cache map
	cache := make(map[string]string)
	for _, setting := range settings {
		cache[setting.Key] = setting.Value
	}

	// Update cache with write lock
	s.cacheMu.Lock()
	s.settingsCache = cache
	s.cacheExpiry = time.Now().Add(5 * time.Minute)
	s.cacheMu.Unlock()

	logging.WithFields(map[string]interface{}{
		"settings_count": len(settings),
		"cache_expiry":   s.cacheExpiry.Format(time.RFC3339),
	}).Debug("Site settings cache refreshed successfully")
}

// getSiteName fetches the site name from cache, falls back to default if not found
// Cache is refreshed automatically when expired (5-minute TTL).
func (s *Service) getSiteName() string {
	if s.db == nil {
		return models.DefaultSiteName
	}

	// Fast path: read from cache with read lock
	s.cacheMu.RLock()
	if time.Now().Before(s.cacheExpiry) && s.settingsCache != nil {
		if name := s.settingsCache["site_name"]; name != "" {
			s.cacheMu.RUnlock()
			return name
		}
	}
	s.cacheMu.RUnlock()

	// Slow path: cache expired or empty - refresh needed
	// Use atomic flag to prevent thundering herd (multiple goroutines refreshing simultaneously)
	if s.refreshing.CompareAndSwap(false, true) {
		s.refreshSettingsCache()
		s.refreshing.Store(false)
	}

	// Read again after refresh attempt
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if s.settingsCache != nil {
		if name := s.settingsCache["site_name"]; name != "" {
			return name
		}
	}

	// Fallback if cache is still empty
	return models.DefaultSiteName
}

// SendEmail sends an email using the configured provider
func (s *Service) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email service is not configured")
	}

	// Validate email address before attempting to send
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Bound the send with its own timeout. context.WithoutCancel detaches
	// from the caller's cancellation signal — a client disconnecting mid
	// -request must not abort an in-flight password-reset/invite email send
	// — while still carrying the caller's values (trace context, request ID)
	// so the send stays linked to the originating trace/log context.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	return s.provider.SendEmail(ctx, to, subject, htmlBody)
}

// SendPasswordResetEmail sends a password reset email
func (s *Service) SendPasswordResetEmail(ctx context.Context, to, username, resetToken string) error {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	siteName := s.getSiteName()
	subject := fmt.Sprintf("Password Reset Request - %s", siteName)
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
            <p>We received a request to reset your password for your %s account.</p>
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
            <p>© %s - This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
`, username, siteName, resetLink, resetLink, siteName)

	return s.SendEmail(ctx, to, subject, body)
}

// SendPasswordSetupEmail sends a password setup email for new user invitations
func (s *Service) SendPasswordSetupEmail(ctx context.Context, to, username, setupToken string) error {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	// URL-encode the token for safe transmission
	encodedToken := url.QueryEscape(setupToken)
	setupLink := fmt.Sprintf("%s/setup-password?token=%s", baseURL, encodedToken)

	siteName := s.getSiteName()
	subject := fmt.Sprintf("Welcome to %s - Set Your Password", siteName)
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
            <h1>Welcome to %s!</h1>
        </div>
        <div class="content">
            <p class="welcome">Hello %s,</p>
            <p>Your username for signing in is: <strong>%s</strong></p>
            <p>Your account has been created for %s. We're excited to have you join our team!</p>
            <p>To get started, please click the button below to set your password:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Set Your Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #0e6c55;">%s</p>
            <p><strong>This link will expire in 7 days.</strong></p>
            <p>Once you've set your password, you'll be able to sign in and start contributing to our mission of helping animals in need.</p>
            <p>If you have any questions or didn't expect this invitation, please contact your administrator.</p>
        </div>
        <div class="footer">
            <p>© %s - This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
`, siteName, username, username, siteName, setupLink, setupLink, siteName)

	return s.SendEmail(ctx, to, subject, body)
}

// SendAnnouncementEmail sends an announcement email
func (s *Service) SendAnnouncementEmail(ctx context.Context, to, title, content string) error {
	subject, body := s.buildAnnouncementEmail(title, content)
	return s.SendEmail(ctx, to, subject, body)
}

// buildAnnouncementEmail renders the subject/HTML body shared by
// SendAnnouncementEmail and SendAnnouncementEmailBatch, so the two single-
// vs-batch send paths can't drift into sending different-looking emails.
func (s *Service) buildAnnouncementEmail(title, content string) (subject, body string) {
	siteName := s.getSiteName()
	subject = fmt.Sprintf("Announcement: %s - %s", title, siteName)

	// Escape HTML in title and convert newlines to HTML line breaks in content
	escapedTitle := html.EscapeString(title)
	htmlContent := strings.ReplaceAll(html.EscapeString(content), "\n", "<br>")

	body = fmt.Sprintf(`
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
            <p>© %s - You're receiving this because you opted in to email notifications.</p>
            <p>You can manage your email preferences in your account settings.</p>
        </div>
    </div>
</body>
</html>
`, escapedTitle, htmlContent, siteName)

	return subject, body
}

// nonBatchSendMaxAttempts/nonBatchSendRetryDelay bound the fallback retry
// SendAnnouncementEmailBatch uses for a provider with no batch endpoint of
// its own (SMTP, or a test double). The base Provider interface exposes a
// failure as a plain error with no status code, so this can't single out a
// rate limit the way ResendProvider's own batch retry does - it treats any
// failure as possibly transient, bounded by a small retry count so one
// permanently bad address doesn't loop for long.
const (
	nonBatchSendMaxAttempts = 3
	nonBatchSendRetryDelay  = 500 * time.Millisecond
)

// batchSendTimeout bounds a call to the provider's batch endpoint (see
// BatchEmailProvider.SendBatch) - longer than SendEmail's own 30s single-
// send timeout because it may retry several of Resend's own chunked calls.
// It does NOT bound SendAnnouncementEmailBatch's per-recipient fallback
// loop for a provider with no batch endpoint - see that loop's own
// comment for why a single fixed cap doesn't fit a call whose duration
// scales with recipient count.
const batchSendTimeout = 2 * time.Minute

// SendAnnouncementEmailBatch sends the same announcement to every address
// in to, preferring the provider's batch endpoint (currently only Resend,
// via BatchEmailProvider) when it has one - turning what would be one HTTP
// call per recipient into one call per up-to-100 recipients, which is what
// actually keeps a group-wide announcement under Resend's per-second rate
// limit instead of just retrying around hitting it. A provider without a
// batch endpoint (SMTP) falls back to sending one at a time, each with a
// short retry.
//
// Returns one error per recipient, in the same order as to (nil for a
// recipient whose send succeeded), mirroring BatchEmailProvider.SendBatch.
func (s *Service) SendAnnouncementEmailBatch(ctx context.Context, to []string, title, content string) (errs []error) {
	errs = make([]error, len(to))
	defer func() {
		providerName := "none"
		if s.provider != nil {
			providerName = s.provider.GetProviderName()
		}
		recordAnnouncementOutcomes(context.WithoutCancel(ctx), providerName, errs)
	}()
	if !s.IsConfigured() {
		err := fmt.Errorf("email service is not configured")
		for i := range errs {
			errs[i] = err
		}
		return errs
	}

	subject, body := s.buildAnnouncementEmail(title, content)

	valid := make([]string, 0, len(to))
	validIdx := make([]int, 0, len(to))
	for i, addr := range to {
		if !isValidEmail(addr) {
			errs[i] = fmt.Errorf("invalid email address: %s", addr)
			continue
		}
		valid = append(valid, addr)
		validIdx = append(validIdx, i)
	}
	if len(valid) == 0 {
		return errs
	}

	// Mirrors SendEmail's own context.WithoutCancel reasoning: this is
	// always called off the request path already (see
	// sendGroupAnnouncementEmails), but detaching here too means a caller
	// that ever stops doing so doesn't silently abort a group's worth of
	// sends on client disconnect.
	detachedCtx := context.WithoutCancel(ctx)

	if batchProvider, ok := s.provider.(BatchEmailProvider); ok {
		// batchSendTimeout only bounds the batch-endpoint call: a handful
		// of chunked HTTP calls with their own bounded retries, so a fixed
		// cap is the right shape here.
		batchCtx, cancel := context.WithTimeout(detachedCtx, batchSendTimeout)
		defer cancel()
		results := batchProvider.SendBatch(batchCtx, valid, subject, body)
		for i, err := range results {
			errs[validIdx[i]] = err
		}
		return errs
	}

	// No batchSendTimeout wrapping the fallback loop: unlike the batch
	// path, its total duration scales with len(valid), which a single
	// fixed cap can't account for - a large group would otherwise have its
	// later recipients fail outright with "context deadline exceeded"
	// once the cap elapsed, never actually attempted. Each recipient's own
	// attempts are already bounded (nonBatchSendPerAttemptTimeout *
	// nonBatchSendMaxAttempts, plus the small inter-attempt delay), so the
	// loop as a whole can't hang - it just isn't capped at one number that
	// has no good value for every group size.
	for i, addr := range valid {
		errs[validIdx[i]] = sendWithRetry(detachedCtx, s.provider, addr, subject, body)
	}
	return errs
}

// nonBatchSendPerAttemptTimeout bounds a single SendEmail call inside
// sendWithRetry, mirroring Service.SendEmail's own 30s per-send timeout.
// Without a timeout of its own, one attempt sharing the caller's much
// longer batchSendTimeout could burn most of it on a single hung
// connection (SMTP in particular takes its deadline from ctx), starving
// every recipient still waiting behind it in the loop.
const nonBatchSendPerAttemptTimeout = 30 * time.Second

// sendWithRetry retries a single-recipient send a couple of times for a
// provider with no batch endpoint of its own. See nonBatchSendMaxAttempts.
func sendWithRetry(ctx context.Context, provider Provider, to, subject, htmlBody string) error {
	var err error
	for attempt := 0; attempt < nonBatchSendMaxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, nonBatchSendPerAttemptTimeout)
		err = provider.SendEmail(attemptCtx, to, subject, htmlBody)
		cancel()
		if err == nil {
			return nil
		}
		if attempt == nonBatchSendMaxAttempts-1 {
			break
		}
		select {
		case <-time.After(nonBatchSendRetryDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
