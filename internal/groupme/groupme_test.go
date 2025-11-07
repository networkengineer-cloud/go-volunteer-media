package groupme

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Run("creates service instance", func(t *testing.T) {
		service := NewService()
		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
	})
}

func TestService_SendMessage(t *testing.T) {
	tests := []struct {
		name           string
		botID          string
		text           string
		setupServer    func() *httptest.Server
		expectedError  string
		validateResult func(t *testing.T, err error)
	}{
		{
			name:  "successful message send",
			botID: "test-bot-123",
			text:  "Test announcement",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request method
					assert.Equal(t, http.MethodPost, r.Method)
					
					// Validate URL path
					assert.Equal(t, "/v3/bots/post", r.URL.Path)
					
					// Validate content type
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					
					// Validate request body
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					
					var payload map[string]string
					err = json.Unmarshal(body, &payload)
					require.NoError(t, err)
					
					assert.Equal(t, "test-bot-123", payload["bot_id"])
					assert.Equal(t, "Test announcement", payload["text"])
					
					// Return success response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code": 201,
						},
					})
				}))
			},
			validateResult: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:  "empty bot ID",
			botID: "",
			text:  "Test message",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("should not make request with empty bot ID")
				}))
			},
			expectedError: "bot ID is required",
		},
		{
			name:  "empty message text",
			botID: "test-bot-123",
			text:  "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("should not make request with empty text")
				}))
			},
			expectedError: "message text is required",
		},
		{
			name:  "API returns 400 error",
			botID: "invalid-bot-id",
			text:  "Test message",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code":   400,
							"errors": []string{"Invalid bot_id"},
						},
					})
				}))
			},
			expectedError: "GroupMe API error",
		},
		{
			name:  "network error",
			botID: "test-bot-123",
			text:  "Test message",
			setupServer: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("should not reach handler")
				}))
				server.Close() // Close immediately to simulate network error
				return server
			},
			expectedError: "failed to send GroupMe message",
		},
		{
			name:  "message text too long (truncated)",
			botID: "test-bot-123",
			text:  string(make([]byte, 2000)), // 2000 characters
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					
					var payload map[string]string
					err = json.Unmarshal(body, &payload)
					require.NoError(t, err)
					
					// Should be truncated to 1000 characters
					assert.LessOrEqual(t, len(payload["text"]), 1000)
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code": 201,
						},
					})
				}))
			},
			validateResult: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:  "API returns 429 rate limit",
			botID: "test-bot-123",
			text:  "Test message",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code":   429,
							"errors": []string{"Rate limit exceeded"},
						},
					})
				}))
			},
			expectedError: "GroupMe API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer func() {
				// Only close if not already closed
				if server != nil {
					server.Close()
				}
			}()

			service := NewService()
			service.apiURL = server.URL + "/v3/bots/post"

			err := service.SendMessage(tt.botID, tt.text)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, err)
			}
		})
	}
}

func TestService_SendAnnouncement(t *testing.T) {
	tests := []struct {
		name           string
		botID          string
		title          string
		content        string
		setupServer    func() *httptest.Server
		expectedError  string
		validateResult func(t *testing.T, err error)
	}{
		{
			name:    "successful announcement send",
			botID:   "test-bot-123",
			title:   "Important Update",
			content: "This is the announcement content",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					
					var payload map[string]string
					err = json.Unmarshal(body, &payload)
					require.NoError(t, err)
					
					// Verify formatted message
					assert.Contains(t, payload["text"], "📢 Important Update")
					assert.Contains(t, payload["text"], "This is the announcement content")
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code": 201,
						},
					})
				}))
			},
			validateResult: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:    "empty title",
			botID:   "test-bot-123",
			title:   "",
			content: "Content only",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("should not make request with empty title")
				}))
			},
			expectedError: "title is required",
		},
		{
			name:    "empty content",
			botID:   "test-bot-123",
			title:   "Title",
			content: "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("should not make request with empty content")
				}))
			},
			expectedError: "content is required",
		},
		{
			name:    "long title and content (truncated)",
			botID:   "test-bot-123",
			title:   string(make([]byte, 300)),
			content: string(make([]byte, 1500)),
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					
					var payload map[string]string
					err = json.Unmarshal(body, &payload)
					require.NoError(t, err)
					
					// Should be truncated to 1000 characters total
					assert.LessOrEqual(t, len(payload["text"]), 1000)
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"meta": map[string]interface{}{
							"code": 201,
						},
					})
				}))
			},
			validateResult: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			service := NewService()
			service.apiURL = server.URL + "/v3/bots/post"

			err := service.SendAnnouncement(tt.botID, tt.title, tt.content)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, err)
			}
		})
	}
}

func TestFormatAnnouncementMessage(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		content  string
		expected string
	}{
		{
			name:     "basic formatting",
			title:    "Test Title",
			content:  "Test content",
			expected: "📢 Test Title\n\nTest content",
		},
		{
			name:     "with newlines in content",
			title:    "Multiline",
			content:  "Line 1\nLine 2\nLine 3",
			expected: "📢 Multiline\n\nLine 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAnnouncementMessage(tt.title, tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{
			name:      "text shorter than max",
			text:      "Short text",
			maxLength: 100,
			expected:  "Short text",
		},
		{
			name:      "text equal to max",
			text:      "Exact",
			maxLength: 5,
			expected:  "Exact",
		},
		{
			name:      "text longer than max",
			text:      "This is a very long message that needs truncation",
			maxLength: 20,
			expected:  "This is a very lo...",
		},
		{
			name:      "max length less than ellipsis",
			text:      "Test",
			maxLength: 2,
			expected:  "Te",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateMessage(tt.text, tt.maxLength)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLength)
		})
	}
}
