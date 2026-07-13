package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/otel"
)

// weakTestCertPEM is a throwaway self-signed cert (not a secret; borrowed
// from the otlptracehttp test suite) used only to give the OTLP exporter's
// TLS config a non-nil cert pool so we can force a real, synchronous
// construction-time error (see TestInit_ExporterSetupError_FallsBackToNoOp).
const weakTestCertPEM = `
-----BEGIN CERTIFICATE-----
MIIBhzCCASygAwIBAgIRANHpHgAWeTnLZpTSxCKs0ggwCgYIKoZIzj0EAwIwEjEQ
MA4GA1UEChMHb3RlbC1nbzAeFw0yMTA0MDExMzU5MDNaFw0yMTA0MDExNDU5MDNa
MBIxEDAOBgNVBAoTB290ZWwtZ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAS9
nWSkmPCxShxnp43F+PrOtbGV7sNfkbQ/kxzi9Ego0ZJdiXxkmv/C05QFddCW7Y0Z
sJCLHGogQsYnWJBXUZOVo2MwYTAOBgNVHQ8BAf8EBAMCB4AwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDAYDVR0TAQH/BAIwADAsBgNVHREEJTAjgglsb2NhbGhvc3SHEAAA
AAAAAAAAAAAAAAAAAAGHBH8AAAEwCgYIKoZIzj0EAwIDSQAwRgIhANwZVVKvfvQ/
1HXsTvgH+xTQswOwSSKYJ1cVHQhqK7ZbAiEAus8NxpTRnp5DiTMuyVmhVNPB+bVH
Lhnm4N/QDk5rek0=
-----END CERTIFICATE-----
`

func TestInit_NoEndpoint_IsNoOp(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	Init(context.Background(), "test-service", "test")

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

	Init(context.Background(), "test-service", "test")

	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

// TestInit_ExporterSetupError_FallsBackToNoOp forces a real, synchronous
// error out of the OTLP trace exporter's construction and asserts Init falls
// back to a fresh no-op tracer provider instead of leaving the global tracer
// provider pointing at whatever partially-initialized (and now shut-down)
// real provider Init had configured so far.
//
// otlptracehttp.New(ctx) is otherwise "lazy" — a merely unreachable or even
// malformed OTEL_EXPORTER_OTLP_ENDPOINT does not error synchronously,
// because the exporter's env-var URL parsing swallows its own errors (see
// otlptracehttp's internal envconfig.WithURL, which logs via
// otel's global error handler and returns rather than surfacing the parse
// error to the caller) and network reachability is only checked on first
// export. The one construction-time check that *does* return an error
// synchronously is otlptracehttp's client.Start: it rejects an exporter
// configured with OTEL_EXPORTER_OTLP_INSECURE=true and a non-nil TLS
// config at the same time (contradictory: insecure transport + a
// configured cert pool). Supplying OTEL_EXPORTER_OTLP_CERTIFICATE
// (pointing at a valid PEM file) alongside OTEL_EXPORTER_OTLP_INSECURE=true
// reliably triggers that check without any network I/O.
func TestInit_ExporterSetupError_FallsBackToNoOp(t *testing.T) {
	certPath := filepath.Join(t.TempDir(), "weak-cert.pem")
	if err := os.WriteFile(certPath, []byte(weakTestCertPEM), 0o600); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_CERTIFICATE", certPath)

	Init(context.Background(), "test-service", "test")

	_, span := otel.Tracer("test").Start(context.Background(), "test-span")
	defer span.End()
	if span.IsRecording() {
		t.Fatal("expected global tracer provider to fall back to a non-recording no-op provider after a failed Init")
	}

	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error after failed Init: %v", err)
	}
}
