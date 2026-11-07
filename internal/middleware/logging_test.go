package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

// Mutates the package-level default logger, so it must not run in parallel.
func TestLoggingMiddleware_LevelByStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{"success", http.StatusOK, "INFO"},
		{"client error", http.StatusNotFound, "WARN"},
		{"client closed request", StatusClientClosedRequest, "INFO"},
		{"server error", http.StatusInternalServerError, "ERROR"},
		{"gateway timeout", http.StatusGatewayTimeout, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := logging.GetDefaultLogger()
			logging.SetDefaultLogger(logging.New(logging.DEBUG, &buf, true))
			t.Cleanup(func() { logging.SetDefaultLogger(prev) })

			r := gin.New()
			r.Use(LoggingMiddleware())
			r.GET("/", func(c *gin.Context) { c.AbortWithStatus(tt.status) })
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			var entry struct {
				Level string `json:"level"`
			}
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
				t.Fatalf("expected exactly one JSON log line, got %q: %v", buf.String(), err)
			}
			if entry.Level != tt.wantLevel {
				t.Fatalf("status %d logged at %s, want %s", tt.status, entry.Level, tt.wantLevel)
			}
		})
	}
}
