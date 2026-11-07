package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

const (
	defaultResendAPIURL = "https://api.resend.com/emails"

	// resendBatchMaxRecipients is Resend's own cap on how many messages one
	// batch call may contain; SendBatch chunks at this size rather than
	// relying on the API to reject an oversized call.
	resendBatchMaxRecipients = 100

	// resendBatchMaxAttempts bounds how many times one chunk's call is
	// retried on a transient failure (429, 5xx, or a network error) before
	// SendBatch gives up on it and reports the error for every recipient in
	// that chunk.
	resendBatchMaxAttempts = 3

	// resendBatchRetryBaseDelay is the backoff used when a retryable
	// response carries no Retry-After header of its own. Doubles each
	// attempt (500ms, 1s).
	resendBatchRetryBaseDelay = 500 * time.Millisecond
)

// resendSingleSendInterval paces the per-recipient fallback SendBatch uses
// when Resend rejects a whole chunk as invalid (see
// isBatchValidationRejection). Kept under Resend's default limit of 2
// requests per second, since sending a chunk one call per recipient is
// exactly the traffic pattern that produced the original 429. var, not
// const, so tests can shrink it.
var resendSingleSendInterval = 550 * time.Millisecond

// resendBatchFallbacks counts batch chunks Resend rejected for their
// content, which SendBatch then re-sent one recipient at a time. A steady
// non-zero rate means a bad address (or payload) is sitting in a group and
// every announcement to it is paying the slow per-recipient path.
var resendBatchFallbacks = telemetry.NewInstrument("email.resend.batch.per_recipient_fallback", func() (metric.Int64Counter, error) {
	return telemetry.Meter("internal/email").Int64Counter(
		"email.resend.batch.per_recipient_fallback",
		metric.WithDescription("Resend batch chunks rejected for their content and re-sent one recipient at a time"),
		metric.WithUnit("{chunk}"),
	)
})

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
	// sendBatchChunk appends "/batch"; a configured trailing slash would
	// otherwise produce ".../emails//batch".
	apiURL = strings.TrimRight(apiURL, "/")
	
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

// ResendBatchEmailResponse represents the Resend batch API response
// structure. On success Data holds one entry per message in the request,
// in the same order; on failure Error is populated and Data is empty - the
// batch endpoint accepts or rejects a whole call atomically, it does not
// partially succeed.
type ResendBatchEmailResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
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

// SendBatch sends subject/htmlBody to every address in to via Resend's
// batch endpoint, chunked at resendBatchMaxRecipients. Each chunk is one
// API call that either accepts or rejects the whole chunk. A chunk rejected
// for its content (see isBatchValidationRejection) is re-sent one recipient
// at a time so only the bad address fails; any other chunk error (after
// retries) is reported for every recipient in that chunk - see
// ResendBatchEmailResponse.
func (p *ResendProvider) SendBatch(ctx context.Context, to []string, subject, htmlBody string) []error {
	errs := make([]error, len(to))
	if !p.IsConfigured() {
		err := fmt.Errorf("resend provider is not configured")
		for i := range errs {
			errs[i] = err
		}
		return errs
	}

	from := p.FromEmail
	if p.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.FromName, p.FromEmail)
	}

	for start := 0; start < len(to); start += resendBatchMaxRecipients {
		end := start + resendBatchMaxRecipients
		if end > len(to) {
			end = len(to)
		}
		chunk := to[start:end]

		payload := make([]ResendEmailRequest, len(chunk))
		for i, addr := range chunk {
			payload[i] = ResendEmailRequest{From: from, To: []string{addr}, Subject: subject, HTML: htmlBody}
		}

		// One idempotency key per chunk, generated once and reused across
		// every retry of that same chunk (see sendBatchChunkWithRetry): if a
		// retry fires because the response to an already-accepted call was
		// lost (a network error or timeout - isRetryableResendStatus(0) -
		// not because Resend actually rejected it), Resend recognizes the
		// repeated key within its 24h window and returns the original
		// result instead of sending the chunk's recipients a second email.
		idempotencyKey := uuid.NewString()
		statusCode, chunkErr := p.sendBatchChunkWithRetry(ctx, payload, idempotencyKey)
		if chunkErr != nil && isBatchValidationRejection(statusCode) {
			// Resend validates a batch as a whole: one bad message rejects
			// the entire chunk. Without this fallback, one malformed
			// address would cost up to resendBatchMaxRecipients other
			// members their email, where the old per-recipient path lost
			// only that one. Sending the chunk individually confines the
			// failure to the recipient(s) actually at fault.
			resendBatchFallbacks.Add(ctx, 1, metric.WithAttributes(attribute.Int("http.response.status_code", statusCode)))
			logging.WithFields(map[string]interface{}{
				"status_code": statusCode,
				"chunk_size":  len(chunk),
				"error":       chunkErr.Error(),
			}).Warn("Resend rejected a batch chunk; re-sending to each recipient individually")
			p.sendChunkIndividually(ctx, chunk, subject, htmlBody, errs[start:end])
			continue
		}
		for i := range chunk {
			errs[start+i] = chunkErr
		}
	}
	return errs
}

// sendChunkIndividually sends to each address in chunk with its own
// single-recipient call, paced by resendSingleSendInterval, writing each
// result into the matching slot of errs. Recipients not yet attempted when
// ctx ends get ctx's error.
func (p *ResendProvider) sendChunkIndividually(ctx context.Context, chunk []string, subject, htmlBody string, errs []error) {
	for i, addr := range chunk {
		if i > 0 {
			select {
			case <-time.After(resendSingleSendInterval):
			case <-ctx.Done():
				for j := i; j < len(chunk); j++ {
					errs[j] = ctx.Err()
				}
				return
			}
		}
		errs[i] = p.SendEmail(ctx, addr, subject, htmlBody)
	}
}

// isBatchValidationRejection reports whether a failed batch call was
// rejected for its content (400/422) rather than for auth, rate limit or a
// server fault - the only case where retrying recipient by recipient can
// succeed for the others.
func isBatchValidationRejection(statusCode int) bool {
	return statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity
}

// sendBatchChunkWithRetry retries sendBatchChunk on a transient failure
// (429 or 5xx), honoring the response's Retry-After header when it sends
// one and otherwise backing off by resendBatchRetryBaseDelay, doubled each
// attempt. A non-transient failure (e.g. a malformed request) is returned
// immediately without burning the remaining attempts. idempotencyKey is
// sent unchanged on every attempt - see SendBatch's comment on why.
//
// Returns the last attempt's HTTP status (0 if none was received) alongside
// its error, so SendBatch can tell a content rejection from other failures.
func (p *ResendProvider) sendBatchChunkWithRetry(ctx context.Context, payload []ResendEmailRequest, idempotencyKey string) (int, error) {
	var err error
	var statusCode int
	for attempt := 0; attempt < resendBatchMaxAttempts; attempt++ {
		var retryAfter time.Duration
		statusCode, retryAfter, err = p.sendBatchChunk(ctx, payload, idempotencyKey)
		if err == nil {
			return statusCode, nil
		}
		if !isRetryableResendStatus(statusCode) || attempt == resendBatchMaxAttempts-1 {
			return statusCode, err
		}

		delay := retryAfter
		if delay <= 0 {
			delay = resendBatchRetryBaseDelay << attempt
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return statusCode, err
}

// isRetryableResendStatus reports whether a failed batch call is worth
// retrying: a rate limit or a server-side error is transient, but a 4xx
// like a malformed request or bad API key will fail again identically.
// statusCode 0 means the request never got an HTTP response at all (a
// network error), which is also worth retrying.
func isRetryableResendStatus(statusCode int) bool {
	return statusCode == 0 || statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

// sendBatchChunk makes one call to Resend's batch endpoint. statusCode is
// the HTTP response's status (0 if no response was received at all, e.g. a
// network error); retryAfter is the delay the response asked for via its
// Retry-After header (0 if absent or unparseable). idempotencyKey is sent
// as the Idempotency-Key header, documented at
// resend.com/docs/dashboard/emails/idempotency-keys: a retried call with
// the same key and the same payload returns the original result instead of
// sending again.
func (p *ResendProvider) sendBatchChunk(ctx context.Context, payload []ResendEmailRequest, idempotencyKey string) (statusCode int, retryAfter time.Duration, err error) {
	ctx, span := tracer.Start(ctx, "email.resend.send_batch", trace.WithAttributes(
		attribute.Int("email.batch_size", len(payload)),
	))
	defer span.End()

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, 0, telemetry.Fail(span, fmt.Errorf("failed to marshal batch request: %w", err), "failed to marshal batch request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL+"/batch", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, 0, telemetry.Fail(span, fmt.Errorf("failed to create batch request: %w", err), "failed to create batch request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, 0, telemetry.Fail(span, fmt.Errorf("failed to send batch request: %w", err), "batch request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, 0, telemetry.Fail(span, fmt.Errorf("failed to read batch response: %w", err), "failed to read batch response")
	}

	if resp.StatusCode == http.StatusOK {
		return resp.StatusCode, 0, nil
	}

	retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	var resendResp ResendBatchEmailResponse
	if err := json.Unmarshal(body, &resendResp); err != nil {
		return resp.StatusCode, retryAfter, telemetry.Fail(span, fmt.Errorf("resend batch API error: status %d, body: %s", resp.StatusCode, string(body)), "non-200 batch response")
	}
	if resendResp.Error.Message != "" {
		return resp.StatusCode, retryAfter, telemetry.Fail(span, fmt.Errorf("resend batch API error: %s", resendResp.Error.Message), "non-200 batch response")
	}
	return resp.StatusCode, retryAfter, telemetry.Fail(span, fmt.Errorf("resend batch API error: status %d", resp.StatusCode), "non-200 batch response")
}

// parseRetryAfter parses a Retry-After header's value as whole seconds,
// per Resend's rate-limit documentation. Returns 0 if the header is absent
// or not a positive integer, leaving the caller to fall back to its own
// backoff.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	secs, err := strconv.Atoi(header)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}
