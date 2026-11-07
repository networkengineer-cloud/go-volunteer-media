package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewResendProvider(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("RESEND_API_KEY")
	origFromEmail := os.Getenv("RESEND_FROM_EMAIL")
	origFromName := os.Getenv("RESEND_FROM_NAME")

	// Set test env vars
	os.Setenv("RESEND_API_KEY", "re_test_key_123")
	os.Setenv("RESEND_FROM_EMAIL", "test@example.com")
	os.Setenv("RESEND_FROM_NAME", "Test User")

	provider := NewResendProvider()

	if provider.APIKey != "re_test_key_123" {
		t.Errorf("Expected APIKey to be re_test_key_123, got %s", provider.APIKey)
	}
	if provider.FromEmail != "test@example.com" {
		t.Errorf("Expected FromEmail to be test@example.com, got %s", provider.FromEmail)
	}
	if provider.FromName != "Test User" {
		t.Errorf("Expected FromName to be Test User, got %s", provider.FromName)
	}

	// Restore original env vars
	os.Setenv("RESEND_API_KEY", origAPIKey)
	os.Setenv("RESEND_FROM_EMAIL", origFromEmail)
	os.Setenv("RESEND_FROM_NAME", origFromName)
}

func TestResendProvider_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		provider *ResendProvider
		want     bool
	}{
		{
			name: "fully configured",
			provider: &ResendProvider{
				APIKey:    "re_test_key",
				FromEmail: "test@example.com",
			},
			want: true,
		},
		{
			name: "missing API key",
			provider: &ResendProvider{
				APIKey:    "",
				FromEmail: "test@example.com",
			},
			want: false,
		},
		{
			name: "missing from email",
			provider: &ResendProvider{
				APIKey:    "re_test_key",
				FromEmail: "",
			},
			want: false,
		},
		{
			name: "completely empty",
			provider: &ResendProvider{
				APIKey:    "",
				FromEmail: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResendProvider_GetProviderName(t *testing.T) {
	provider := NewResendProvider()
	if provider.GetProviderName() != "resend" {
		t.Errorf("Expected provider name 'resend', got '%s'", provider.GetProviderName())
	}
}

func TestResendProvider_SendEmail_NotConfigured(t *testing.T) {
	provider := &ResendProvider{
		APIKey:    "",
		FromEmail: "",
	}

	err := provider.SendEmail(context.Background(), "test@example.com", "Test Subject", "<html><body>Test Body</body></html>")
	if err == nil {
		t.Error("Expected error when provider is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected error to contain 'not configured', got %s", err.Error())
	}
}

func TestResendProvider_SendEmail_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer") {
			t.Errorf("Expected Authorization header with Bearer, got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req ResendEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if req.From != "Test User <test@example.com>" {
			t.Errorf("Expected From to be 'Test User <test@example.com>', got '%s'", req.From)
		}
		if len(req.To) != 1 || req.To[0] != "recipient@example.com" {
			t.Errorf("Expected To to be ['recipient@example.com'], got %v", req.To)
		}
		if req.Subject != "Test Subject" {
			t.Errorf("Expected Subject to be 'Test Subject', got '%s'", req.Subject)
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResendEmailResponse{
			ID: "test-email-id",
		})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey:    "test-api-key",
		FromEmail: "test@example.com",
		FromName:  "Test User",
		apiURL:    server.URL, // Use mock server URL
		client:    server.Client(),
	}

	// Actually test SendEmail - this was missing!
	err := provider.SendEmail(context.Background(), "recipient@example.com", "Test Subject", "<html><body>Test</body></html>")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestResendProvider_SendEmail_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResendEmailResponse{
			Error: struct {
				Message string `json:"message"`
			}{
				Message: "Invalid email address",
			},
		})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey:    "test-api-key",
		FromEmail: "test@example.com",
		apiURL:    server.URL, // Use mock server URL
		client:    server.Client(),
	}

	err := provider.SendEmail(context.Background(), "invalid", "Test", "<html><body>Test</body></html>")
	if err == nil {
		t.Error("Expected error for API error response")
	}
	if !strings.Contains(err.Error(), "Invalid email address") {
		t.Errorf("Expected error message to contain 'Invalid email address', got: %v", err)
	}
}

func TestResendProvider_SendBatch_Success(t *testing.T) {
	var gotPayload []ResendEmailRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batch" {
			t.Errorf("Expected batch request to POST /batch, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Errorf("Failed to decode batch request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{{"id": "1"}, {"id": "2"}}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey:    "test-api-key",
		FromEmail: "test@example.com",
		FromName:  "Test User",
		apiURL:    server.URL,
		client:    server.Client(),
	}

	to := []string{"a@example.com", "b@example.com"}
	errs := provider.SendBatch(context.Background(), to, "Subject", "<p>Body</p>")

	if len(errs) != len(to) {
		t.Fatalf("Expected %d results, got %d", len(to), len(errs))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("Expected no error for recipient %d, got: %v", i, err)
		}
	}
	if len(gotPayload) != 2 {
		t.Fatalf("Expected batch payload with 2 messages, got %d", len(gotPayload))
	}
	if gotPayload[0].To[0] != "a@example.com" || gotPayload[1].To[0] != "b@example.com" {
		t.Errorf("Expected each message addressed to one recipient, got %v", gotPayload)
	}
}

func TestResendProvider_SendBatch_ChunksAtRecipientCap(t *testing.T) {
	var callCount atomic.Int32
	var chunkSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		var payload []ResendEmailRequest
		json.NewDecoder(r.Body).Decode(&payload)
		chunkSizes = append(chunkSizes, len(payload))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	to := make([]string, 150)
	for i := range to {
		to[i] = "recipient@example.com"
	}

	errs := provider.SendBatch(context.Background(), to, "Subject", "<p>Body</p>")
	for _, err := range errs {
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	}
	if callCount.Load() != 2 {
		t.Fatalf("Expected 150 recipients to need 2 batch calls, got %d", callCount.Load())
	}
	if len(chunkSizes) != 2 || chunkSizes[0] != 100 || chunkSizes[1] != 50 {
		t.Fatalf("Expected chunk sizes [100, 50], got %v", chunkSizes)
	}
}

func TestResendProvider_SendBatch_RetriesOn429ThenSucceeds(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ResendBatchEmailResponse{Error: struct {
				Message string `json:"message"`
			}{Message: "rate limited"}})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{{"id": "1"}}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	errs := provider.SendBatch(context.Background(), []string{"a@example.com"}, "Subject", "<p>Body</p>")
	if errs[0] != nil {
		t.Fatalf("Expected the retry to succeed, got: %v", errs[0])
	}
	if callCount.Load() != 2 {
		t.Fatalf("Expected exactly one retry (2 calls total), got %d", callCount.Load())
	}
}

func TestResendProvider_SendBatch_GivesUpAfterMaxAttemptsOn429(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ResendBatchEmailResponse{Error: struct {
			Message string `json:"message"`
		}{Message: "rate limited"}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	errs := provider.SendBatch(context.Background(), []string{"a@example.com", "b@example.com"}, "Subject", "<p>Body</p>")
	if callCount.Load() != resendBatchMaxAttempts {
		t.Fatalf("Expected exactly %d attempts, got %d", resendBatchMaxAttempts, callCount.Load())
	}
	for i, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "rate limited") {
			t.Errorf("Expected recipient %d to carry the rate-limit error, got: %v", i, err)
		}
	}
}

func TestResendProvider_SendBatch_DoesNotRetryOnBadRequest(t *testing.T) {
	withShortSingleSendInterval(t)
	var batchCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/batch") {
			batchCalls.Add(1)
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResendBatchEmailResponse{Error: struct {
			Message string `json:"message"`
		}{Message: "invalid recipient"}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	errs := provider.SendBatch(context.Background(), []string{"a@example.com"}, "Subject", "<p>Body</p>")
	if batchCalls.Load() != 1 {
		t.Fatalf("Expected a non-transient 400 batch call to be tried only once, got %d calls", batchCalls.Load())
	}
	if errs[0] == nil {
		t.Fatal("Expected an error for the bad request")
	}
}

func withShortSingleSendInterval(t *testing.T) {
	t.Helper()
	orig := resendSingleSendInterval
	resendSingleSendInterval = time.Millisecond
	t.Cleanup(func() { resendSingleSendInterval = orig })
}

// Resend validates a batch as a whole, so one bad address rejects the
// chunk. The others in that chunk must still get their email, and only the
// bad one should report an error.
func TestResendProvider_SendBatch_FallsBackPerRecipientWhenChunkRejected(t *testing.T) {
	withShortSingleSendInterval(t)
	var mu sync.Mutex
	var singleTo []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/batch") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"Invalid to field"}`))
			return
		}
		var req ResendEmailRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		singleTo = append(singleTo, req.To...)
		mu.Unlock()
		if req.To[0] == "bad@example.com" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"Invalid to field"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"x"}`))
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	to := []string{"a@example.com", "bad@example.com", "c@example.com"}
	errs := provider.SendBatch(context.Background(), to, "Subject", "<p>Body</p>")

	if len(singleTo) != 3 {
		t.Fatalf("Expected each recipient re-sent individually, got %v", singleTo)
	}
	if errs[0] != nil || errs[2] != nil {
		t.Errorf("Expected the valid recipients to succeed, got %v / %v", errs[0], errs[2])
	}
	if errs[1] == nil {
		t.Error("Expected the bad recipient to report an error")
	}
}

// Auth failures and the like are not content rejections: re-sending one by
// one would just fail N more times, so the chunk error stands.
func TestResendProvider_SendBatch_NoPerRecipientFallbackOnAuthError(t *testing.T) {
	var singleCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/batch") {
			singleCalls.Add(1)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"API key is invalid"}`))
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}
	errs := provider.SendBatch(context.Background(), []string{"a@example.com", "b@example.com"}, "Subject", "<p>Body</p>")
	if singleCalls.Load() != 0 {
		t.Errorf("Expected no per-recipient fallback on a 401, got %d single sends", singleCalls.Load())
	}
	if errs[0] == nil || errs[1] == nil {
		t.Error("Expected the chunk error for every recipient")
	}
}

func TestNewResendProvider_TrimsTrailingSlashFromAPIURL(t *testing.T) {
	t.Setenv("RESEND_API_URL", "https://example.test/emails/")
	if got := NewResendProvider().apiURL; got != "https://example.test/emails" {
		t.Errorf("Expected trailing slash trimmed, got %q", got)
	}
}

func TestResendProvider_SendBatch_HonorsRetryAfterHeader(t *testing.T) {
	var callCount atomic.Int32
	var secondCallAt time.Time
	firstCallAt := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ResendBatchEmailResponse{})
			return
		}
		secondCallAt = time.Now()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{{"id": "1"}}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	errs := provider.SendBatch(context.Background(), []string{"a@example.com"}, "Subject", "<p>Body</p>")
	if errs[0] != nil {
		t.Fatalf("Expected eventual success, got: %v", errs[0])
	}
	if secondCallAt.Sub(firstCallAt) < 900*time.Millisecond {
		t.Fatalf("Expected the retry to wait for the Retry-After header's 1s, only waited %s", secondCallAt.Sub(firstCallAt))
	}
}

func TestResendProvider_SendBatch_SendsStableIdempotencyKeyAcrossRetries(t *testing.T) {
	var callCount atomic.Int32
	var keys []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		mu.Unlock()
		if callCount.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ResendBatchEmailResponse{})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{{"id": "1"}}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	errs := provider.SendBatch(context.Background(), []string{"a@example.com"}, "Subject", "<p>Body</p>")
	if errs[0] != nil {
		t.Fatalf("Expected eventual success, got: %v", errs[0])
	}
	if len(keys) != 2 {
		t.Fatalf("Expected 2 calls (1 retry), got %d", len(keys))
	}
	if keys[0] == "" {
		t.Error("Expected a non-empty Idempotency-Key header")
	}
	if keys[0] != keys[1] {
		t.Errorf("Expected the retry to reuse the same idempotency key, got %q then %q", keys[0], keys[1])
	}
}

func TestResendProvider_SendBatch_UsesDifferentIdempotencyKeysPerChunk(t *testing.T) {
	var keys []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{}})
	}))
	defer server.Close()

	provider := &ResendProvider{
		APIKey: "test-api-key", FromEmail: "test@example.com",
		apiURL: server.URL, client: server.Client(),
	}

	to := make([]string, 150)
	for i := range to {
		to[i] = "recipient@example.com"
	}
	provider.SendBatch(context.Background(), to, "Subject", "<p>Body</p>")

	if len(keys) != 2 {
		t.Fatalf("Expected 2 chunk calls, got %d", len(keys))
	}
	if keys[0] == "" || keys[1] == "" {
		t.Fatalf("Expected non-empty idempotency keys, got %q and %q", keys[0], keys[1])
	}
	if keys[0] == keys[1] {
		t.Errorf("Expected each chunk to get its own idempotency key, both were %q", keys[0])
	}
}

func TestResendProvider_SendBatch_NotConfigured(t *testing.T) {
	provider := &ResendProvider{}
	errs := provider.SendBatch(context.Background(), []string{"a@example.com", "b@example.com"}, "Subject", "<p>Body</p>")
	for i, err := range errs {
		if err == nil {
			t.Errorf("Expected an error for recipient %d when unconfigured", i)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
	}{
		{"", 0},
		{"2", 2 * time.Second},
		{"0", 0},
		{"-1", 0},
		{"not-a-number", 0},
	}
	for _, tt := range tests {
		if got := parseRetryAfter(tt.header); got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestResendEmailRequest_JSONMarshaling(t *testing.T) {
	req := ResendEmailRequest{
		From:    "Test User <test@example.com>",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		HTML:    "<html><body>Test</body></html>",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Failed to marshal ResendEmailRequest: %v", err)
	}

	var unmarshaled ResendEmailRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal ResendEmailRequest: %v", err)
	}

	if unmarshaled.From != req.From {
		t.Errorf("From mismatch: got %s, want %s", unmarshaled.From, req.From)
	}
	if len(unmarshaled.To) != len(req.To) || unmarshaled.To[0] != req.To[0] {
		t.Errorf("To mismatch: got %v, want %v", unmarshaled.To, req.To)
	}
	if unmarshaled.Subject != req.Subject {
		t.Errorf("Subject mismatch: got %s, want %s", unmarshaled.Subject, req.Subject)
	}
	if unmarshaled.HTML != req.HTML {
		t.Errorf("HTML mismatch: got %s, want %s", unmarshaled.HTML, req.HTML)
	}
}
