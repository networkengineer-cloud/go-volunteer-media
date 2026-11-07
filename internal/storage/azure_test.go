package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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

		err := failBlob(context.Background(), span, &azcore.ResponseError{ErrorCode: "BlobNotFound"}, "download failed")
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

		err := failBlob(context.Background(), span, &azcore.ResponseError{ErrorCode: "AuthenticationFailed"}, "download failed")
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

func TestMapBlobError_PreservesContextErrors(t *testing.T) {
	for _, target := range []error{context.Canceled, context.DeadlineExceeded} {
		// The SDK surfaces transport cancellation wrapped (e.g. in *url.Error);
		// simulate one extra layer of wrapping on top of ours.
		got := mapBlobError(fmt.Errorf("Get \"https://acct.blob.core.windows.net/x\": %w", target))
		if !errors.Is(got, target) {
			t.Fatalf("mapBlobError lost %v through wrapping: %v", target, got)
		}
		if errors.Is(got, ErrNotFound) {
			t.Fatalf("mapBlobError collapsed %v to ErrNotFound", target)
		}
	}
}

func TestFailBlob_ContextErrors(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantSpanError bool
	}{
		{name: "client cancellation is not a span error", err: context.Canceled, wantSpanError: false},
		{name: "deadline exceeded is still a span error", err: context.DeadlineExceeded, wantSpanError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
			_, span := tp.Tracer("test").Start(context.Background(), "test")

			err := failBlob(context.Background(), span, fmt.Errorf("transport: %w", tt.err), "download failed")
			span.End()

			if !errors.Is(err, tt.err) {
				t.Fatalf("failBlob() = %v, want errors.Is(_, %v)", err, tt.err)
			}
			gotSpanError := exporter.GetSpans()[0].Status.Code == codes.Error
			if gotSpanError != tt.wantSpanError {
				t.Fatalf("span error = %v, want %v", gotSpanError, tt.wantSpanError)
			}
		})
	}
}

// newTestSpan returns a recording span and a func that ends it and reports
// whether its status was set to Error.
func newTestSpan(t *testing.T) (trace.Span, func() bool) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	_, span := tp.Tracer("test").Start(context.Background(), "test")
	return span, func() bool {
		span.End()
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 recorded span, got %d", len(spans))
		}
		return spans[0].Status.Code == codes.Error
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// The SDK may surface its own error (not wrapping context.Canceled) once the
// caller has gone; the cancelled context alone must keep it off the span.
func TestFailBlob_CanceledContextWithUnrelatedError(t *testing.T) {
	span, spanErrored := newTestSpan(t)
	err := failBlob(canceledContext(), span, errors.New("connection reset by peer"), "download failed")

	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("failBlob() = %v, want the mapped storage error", err)
	}
	if spanErrored() {
		t.Fatalf("span marked as error for a request the caller cancelled")
	}
}

func TestReadBlob(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		body          io.Reader
		wantErrIs     error
		wantSpanError bool
	}{
		{name: "success", ctx: context.Background(), body: strings.NewReader("img")},
		{name: "body read cancelled", ctx: context.Background(), body: iotest.ErrReader(context.Canceled), wantErrIs: context.Canceled},
		{name: "context cancelled, unrelated read error", ctx: canceledContext(), body: iotest.ErrReader(io.ErrUnexpectedEOF), wantErrIs: io.ErrUnexpectedEOF},
		{name: "deadline exceeded is a span error", ctx: context.Background(), body: iotest.ErrReader(context.DeadlineExceeded), wantErrIs: context.DeadlineExceeded, wantSpanError: true},
		{name: "other read failure is a span error", ctx: context.Background(), body: iotest.ErrReader(io.ErrUnexpectedEOF), wantErrIs: io.ErrUnexpectedEOF, wantSpanError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span, spanErrored := newTestSpan(t)
			data, err := readBlob(tt.ctx, span, tt.body)

			if tt.wantErrIs == nil {
				if err != nil || string(data) != "img" {
					t.Fatalf("readBlob() = %q, %v; want \"img\", nil", data, err)
				}
			} else if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("readBlob() error = %v, want errors.Is(_, %v)", err, tt.wantErrIs)
			}
			if got := spanErrored(); got != tt.wantSpanError {
				t.Fatalf("span error = %v, want %v", got, tt.wantSpanError)
			}
		})
	}
}
