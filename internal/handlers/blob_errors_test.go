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
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
	started     chan struct{}
	startedOnce sync.Once
}

func (p *blockingImageProvider) GetImage(ctx context.Context, _ string) ([]byte, string, error) {
	p.startedOnce.Do(func() { close(p.started) })
	<-ctx.Done()
	return nil, "", fmt.Errorf("azure blob storage error: %w", ctx.Err())
}

// captureLogs swaps the package-level default logger for one writing to a
// buffer and returns a func that reports the level of every line written.
// It mutates global state, so tests using it must not call t.Parallel().
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
	assert.Equal(t, middleware.StatusClientClosedRequest, w.Code)
	levels := logLevels()
	assert.NotEmpty(t, levels, "expected the request to be logged")
	assert.NotContains(t, levels, "ERROR", "a client-cancelled request must not log at error level")
	assert.NotContains(t, levels, "WARN", "a client cancel is routine traffic and is logged at info")
}

// A client can also hang up while the metadata lookup is still running. GORM
// is bound to the request context (middleware.GetDB), so the query fails with
// context.Canceled; that must not be reported as a 500 either.
func TestServeImage_ClientCancelDuringDBLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		route string
		path  string
		h     func(*gorm.DB, storage.Provider) gin.HandlerFunc
	}{
		{"image", "/api/images/:uuid", "/api/images/img-uuid", ServeImage},
		{"video", "/api/videos/:uuid", "/api/videos/vid-uuid", ServeVideo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logLevels := captureLogs(t)
			db := setupVideoTestDB(t)

			r := gin.New()
			r.Use(middleware.LoggingMiddleware())
			r.GET(tt.route, tt.h(db, &mockStorageProvider{}))

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // client already gone when the query runs
			req := httptest.NewRequest(http.MethodGet, tt.path, nil).WithContext(ctx)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, middleware.StatusClientClosedRequest, w.Code)
			assert.NotContains(t, logLevels(), "ERROR")
		})
	}
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
		name   string
		err    error
		reqCtx reqCtxState
		want   int
	}{
		{"not found", storage.ErrNotFound, live, http.StatusNotFound},
		{"wrapped not found", fmt.Errorf("x: %w", storage.ErrNotFound), live, http.StatusNotFound},
		{"client cancelled", fmt.Errorf("azure blob storage error: %w", context.Canceled), canceled, middleware.StatusClientClosedRequest},
		// The provider returned its own error after the client left, without
		// wrapping ctx.Err(); the request context still identifies the cancel.
		{"request cancelled, unrelated provider error", errors.New("connection reset by peer"), canceled, middleware.StatusClientClosedRequest},
		{"deadline exceeded", fmt.Errorf("azure blob storage error: %w", context.DeadlineExceeded), timedOut, http.StatusGatewayTimeout},
		// A request-context timeout is not a cancel: an unrelated error stays a 500.
		{"request timed out, unrelated provider error", errors.New("connection reset by peer"), timedOut, http.StatusInternalServerError},
		{"other failure", errors.New("azure blob storage error: connection refused"), live, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext(t, tt.reqCtx))
			respondBlobFetchError(c, tt.err, "nf", "fail")
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

func TestRespondInternalErrorUnlessCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		err    error
		reqCtx reqCtxState
		want   int
	}{
		{"db error", errors.New("database is locked"), live, http.StatusInternalServerError},
		{"query cancelled", fmt.Errorf("query: %w", context.Canceled), canceled, middleware.StatusClientClosedRequest},
		{"request cancelled, unrelated error", errors.New("driver: bad connection"), canceled, middleware.StatusClientClosedRequest},
		{"query timed out", fmt.Errorf("query: %w", context.DeadlineExceeded), timedOut, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext(t, tt.reqCtx))
			respondInternalErrorUnlessCanceled(c, tt.err, "fail")
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

// reqCtxState selects the state of the request context a test case runs with.
type reqCtxState int

const (
	live     reqCtxState = iota // not done
	canceled                    // client disconnected
	timedOut                    // deadline exceeded
)

// requestContext returns a request context in the given state.
func requestContext(t *testing.T, state reqCtxState) context.Context {
	t.Helper()
	switch state {
	case canceled:
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	case timedOut:
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		t.Cleanup(cancel)
		return ctx
	default:
		return context.Background()
	}
}
