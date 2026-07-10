package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInit_NoEndpoint_IsNoOp(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if err := Init(context.Background(), "test-service", "test"); err != nil {
		t.Fatalf("Init returned error with no endpoint configured: %v", err)
	}

	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error after no-op Init: %v", err)
	}
}

func TestInit_WithEndpoint_ConfiguresProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

	if err := Init(context.Background(), "test-service", "test"); err != nil {
		t.Fatalf("Init returned error with valid endpoint: %v", err)
	}

	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}
