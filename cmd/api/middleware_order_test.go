package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
)

// TestOtelginBeforeRecovery_MarksPanicSpanAsErrored proves the middleware
// ordering established by registerCoreMiddleware (otelgin registered before
// gin.Recovery()) results in a panic being recorded as an errored span, not
// silently swallowed with an Unset span status. This calls
// registerCoreMiddleware directly — the same function main() calls — so a
// future reorder inside that function fails this test instead of only
// contradicting a comment.
func TestOtelginBeforeRecovery_MarksPanicSpanAsErrored(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerCoreMiddleware(router, "test-service")
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Fatalf("expected span status Error, got %v", spans[0].Status.Code)
	}
}

// TestMiddlewareOrder_TraceLogDBCorrelation proves the full registration
// order main() actually uses — registerCoreMiddleware (otelgin), then
// LoggingMiddleware, then DBMiddleware — wires trace correlation end to end:
// the logger built by LoggingMiddleware must already carry the request
// span's trace ID, and a query run through middleware.GetDB must produce a
// child span nested under that same request span. TestOtelginBeforeRecovery
// above only covers registerCoreMiddleware in isolation; neither
// LoggingMiddleware nor DBMiddleware depend on anything that test exercises,
// so a reorder of these three in main() could silently break log/trace
// correlation or GORM span nesting without failing that test.
func TestMiddlewareOrder_TraceLogDBCorrelation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// otelgorm captures the global TracerProvider at plugin-construction
	// time, so it must be registered after SetTracerProvider above —
	// mirroring internal/database.configureTracing's real wiring.
	if err := db.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables())); err != nil {
		t.Fatalf("failed to register otelgorm plugin: %v", err)
	}

	var logBuf bytes.Buffer
	originalLogger := logging.GetDefaultLogger()
	logging.SetDefaultLogger(logging.New(logging.INFO, &logBuf, true))
	defer logging.SetDefaultLogger(originalLogger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerCoreMiddleware(router, "test-service")
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.DBMiddleware(db))
	router.GET("/query", func(c *gin.Context) {
		scoped := middleware.GetDB(c, db)
		if err := scoped.Exec("SELECT 1").Error; err != nil {
			c.String(http.StatusInternalServerError, "query failed: %v", err)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/query", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans (otelgin request span + nested gorm query span), got %d: %+v", len(spans), spans)
	}

	var root, child *tracetest.SpanStub
	for i := range spans {
		if spans[i].Parent.IsValid() {
			child = &spans[i]
		} else {
			root = &spans[i]
		}
	}
	if root == nil {
		t.Fatalf("expected one root span (otelgin's), got none among: %+v", spans)
	}
	if child == nil {
		t.Fatalf("expected one child span (the GORM query), got none among: %+v", spans)
	}
	if child.Parent.SpanID() != root.SpanContext.SpanID() || child.Parent.TraceID() != root.SpanContext.TraceID() {
		t.Fatalf("expected GORM query span to nest under the request span; child parent=%s root=%s",
			child.Parent.SpanID(), root.SpanContext.SpanID())
	}

	// Find the first logged trace_id — proves LoggingMiddleware's logger
	// (built once, before c.Next()) already saw otelgin's span on
	// c.Request.Context() rather than an empty/unsampled context.
	var loggedTraceID string
	for _, line := range strings.Split(strings.TrimSpace(logBuf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry logging.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to unmarshal log line %q: %v", line, err)
		}
		if entry.TraceID != "" {
			loggedTraceID = entry.TraceID
			break
		}
	}
	if loggedTraceID == "" {
		t.Fatalf("expected at least one log line with a non-empty trace_id, got log output: %s", logBuf.String())
	}
	if loggedTraceID != root.SpanContext.TraceID().String() {
		t.Fatalf("logged trace_id %s does not match request span trace_id %s", loggedTraceID, root.SpanContext.TraceID())
	}
}
