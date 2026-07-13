package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMapBlobError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantNotFound bool
	}{
		{
			name:         "blob not found maps to ErrNotFound",
			err:          &azcore.ResponseError{ErrorCode: "BlobNotFound"},
			wantNotFound: true,
		},
		{
			name: "auth failure is not collapsed to ErrNotFound",
			err:  &azcore.ResponseError{ErrorCode: "AuthenticationFailed"},
		},
		{
			name: "throttling is not collapsed to ErrNotFound",
			err:  &azcore.ResponseError{ErrorCode: "ServerBusy"},
		},
		{
			name: "non-azure error is not collapsed to ErrNotFound",
			err:  errors.New("connection reset by peer"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapBlobError(tt.err)

			if tt.wantNotFound {
				if !errors.Is(got, ErrNotFound) {
					t.Fatalf("mapBlobError(%v) = %v, want ErrNotFound", tt.err, got)
				}
				return
			}

			if errors.Is(got, ErrNotFound) {
				t.Fatalf("mapBlobError(%v) incorrectly collapsed to ErrNotFound", tt.err)
			}
			if !errors.Is(got, tt.err) {
				t.Fatalf("mapBlobError(%v) = %v, want it to wrap the original error", tt.err, got)
			}
		})
	}
}

func TestFailBlob(t *testing.T) {
	t.Run("blob-not-found is returned as-is without recording a span error", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
		_, span := tp.Tracer("test").Start(context.Background(), "test")

		err := failBlob(span, &azcore.ResponseError{ErrorCode: "BlobNotFound"}, "download failed")
		span.End()

		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("failBlob() = %v, want ErrNotFound", err)
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 recorded span, got %d", len(spans))
		}
		if len(spans[0].Events) != 0 {
			t.Fatalf("expected no recorded error events for a not-found result, got %d", len(spans[0].Events))
		}
		if spans[0].Status.Code == codes.Error {
			t.Fatalf("expected span status to not be Error for a not-found result, got %v", spans[0].Status)
		}
	})

	t.Run("a real storage error is recorded on the span and not collapsed to ErrNotFound", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
		_, span := tp.Tracer("test").Start(context.Background(), "test")

		err := failBlob(span, &azcore.ResponseError{ErrorCode: "AuthenticationFailed"}, "download failed")
		span.End()

		if errors.Is(err, ErrNotFound) {
			t.Fatalf("failBlob() incorrectly returned ErrNotFound for an authentication failure")
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 recorded span, got %d", len(spans))
		}
		if spans[0].Status.Code != codes.Error {
			t.Fatalf("expected span status Error, got %v", spans[0].Status)
		}
		if len(spans[0].Events) == 0 {
			t.Fatalf("expected the error to be recorded as a span event")
		}
	})
}
