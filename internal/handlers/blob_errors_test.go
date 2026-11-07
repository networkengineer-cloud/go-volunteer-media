package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
)

// blockingImageProvider simulates an Azure download in flight: GetImage
// signals that it has started, then blocks until the request context ends
// and returns ctx.Err() wrapped the way the Azure provider wraps it
// ("azure blob storage error: %w").
type blockingImageProvider struct {
	mockStorageProvider
	started chan struct{}
}

func (p *blockingImageProvider) GetImage(ctx context.Context, _ string) ([]byte, string, error) {
	close(p.started)
	<-ctx.Done()
	return nil, "", fmt.Errorf("azure blob storage error: %w", ctx.Err())
}

// captureLogs swaps the default logger for one writing to a buffer and
// returns the levels of every line written.
func captureLogs(t *testing.T) func() []string {
	t.Helper()
	var buf bytes.Buffer
	prev := logging.GetDefaultLogger()
	logging.SetDefaultLogger(logging.New(logging.DEBUG, &buf, true))
	t.Cleanup(func() { logging.SetDefaultLogger(prev) })
	return func() []string {
		var levels []string
		sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
		for sc.Scan() {
			var entry struct {
				Level string `json:"level"`
			}
			require.NoError(t, json.Unmarshal(sc.Bytes(), &entry), "log line: %s", sc.Text())
			levels = append(levels, entry.Level)
		}
		return levels
	}
}

func setupAzureImageRoute(t *testing.T, store storage.Provider) *gin.Engine {
	t.Helper()
	db := setupVideoTestDB(t)
	require.NoError(t, db.Create(&models.AnimalImage{
		UserID:          1,
		ImageURL:        "/api/images/img-uuid",
		StorageProvider: "azure",
		BlobIdentifier:  "img-uuid.jpg",
	}).Error)

	r := gin.New()
	r.Use(middleware.LoggingMiddleware())
	r.GET("/api/images/:uuid", ServeImage(db, store))
	return r
}

func TestServeImage_ClientCancelMidDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logLevels := captureLogs(t)

	store := &blockingImageProvider{started: make(chan struct{})}
	r := setupAzureImageRoute(t, store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/images/img-uuid", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-store.started:
	case <-time.After(5 * time.Second):
		t.Fatal("GetImage was never called")
	}
	cancel() // client hangs up mid-download
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after cancellation")
	}

	assert.Less(t, w.Code, 500, "a client-cancelled request must not produce a 5xx")
	assert.Equal(t, StatusClientClosedRequest, w.Code)
	levels := logLevels()
	assert.NotEmpty(t, levels, "expected the request to be logged")
	assert.NotContains(t, levels, "ERROR", "a client-cancelled request must not log at error level")
}

func TestServeImage_DeadlineExceededStillErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logLevels := captureLogs(t)

	store := &blockingImageProvider{started: make(chan struct{})}
	r := setupAzureImageRoute(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/images/img-uuid", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Contains(t, logLevels(), "ERROR", "a storage timeout is a real failure and must log at error level")
}

func TestRespondBlobFetchError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", storage.ErrNotFound, http.StatusNotFound},
		{"wrapped not found", fmt.Errorf("x: %w", storage.ErrNotFound), http.StatusNotFound},
		{"client cancelled", fmt.Errorf("azure blob storage error: %w", context.Canceled), StatusClientClosedRequest},
		{"deadline exceeded", fmt.Errorf("azure blob storage error: %w", context.DeadlineExceeded), http.StatusGatewayTimeout},
		{"other failure", errors.New("azure blob storage error: connection refused"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			respondBlobFetchError(c, tt.err, "nf", "fail")
			assert.Equal(t, tt.want, w.Code)
		})
	}
}
