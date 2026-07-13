package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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
