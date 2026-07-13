package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

// mu guards shutdownFuncs and enabled. Init runs once at startup, but
// Shutdown can be invoked concurrently from main's deferred call and from
// logging's FATAL shutdown-hook path (a listener failure and a graceful
// shutdown can both trigger a logger.Fatal around the same time), and
// Enabled is read from arbitrary request-handling goroutines.
var mu sync.Mutex

// shutdownFuncs holds cleanup callbacks for whichever providers Init configured.
// Empty when Init was a no-op (OTEL_EXPORTER_OTLP_ENDPOINT unset).
var shutdownFuncs []func(context.Context) error

// enabled reports whether Init last configured real (non-no-op) providers.
var enabled bool

// Enabled reports whether telemetry is currently exporting to a real
// backend. False before Init runs, when OTEL_EXPORTER_OTLP_ENDPOINT is
// unset, or after a failed Init fell back to no-op providers. Callers use
// this to skip work that's only worth doing when something is actually
// listening (e.g. registering a per-query tracing plugin).
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// initStep names one provider's setup for Init's loop, so the near-identical
// "run it, log a warning and fall back on error" handling lives in one place
// instead of being copy-pasted per provider.
type initStep struct {
	name string
	fn   func(context.Context, *resource.Resource) error
}

// Init configures global OpenTelemetry trace, metric, and log providers from
// OTEL_EXPORTER_OTLP_ENDPOINT and the other standard OTEL_EXPORTER_OTLP_*
// env vars (read automatically by the OTLP exporters). If the endpoint is
// unset, Init leaves OTel's default no-op providers in place, so the app
// behaves identically with zero telemetry configuration (local dev, CI).
// Telemetry setup failure is never fatal to application startup — Init logs
// a warning and falls back to no-op providers instead of returning an error.
func Init(ctx context.Context, serviceName, environment string) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logging.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set, telemetry disabled (using no-op providers)")
		return
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironmentName(environment),
		),
	)
	if err != nil {
		logging.WithField("error", err.Error()).Warn("failed to build otel resource, falling back to no-op telemetry providers")
		return
	}

	steps := []initStep{
		{"tracing", initTraces},
		{"metrics", initMetrics},
		{"logs", initLogs},
	}
	for _, step := range steps {
		if err := step.fn(ctx, res); err != nil {
			logging.WithField("error", err.Error()).Warn(fmt.Sprintf("failed to init otel %s, falling back to no-op telemetry providers", step.name))
			fallback(ctx)
			return
		}
	}

	mu.Lock()
	enabled = true
	mu.Unlock()
	logging.WithField("endpoint", endpoint).Info("OpenTelemetry initialized")
}

// fallback tears down any providers Init already configured before a later
// step failed, then restores fresh no-op providers as the global tracer,
// meter, and logger providers. Without this, a partial failure (e.g. traces
// configured successfully but metrics failed) would leave the global tracer
// provider pointing at a real-but-shut-down provider instead of a working
// no-op, silently breaking every subsequent otel.Tracer(...) call. Shutdown
// already clears shutdownFuncs and enabled, so fallback only adds the no-op
// provider swap on top of it.
func fallback(ctx context.Context) {
	if err := Shutdown(ctx); err != nil {
		logging.WithField("error", err.Error()).Warn("failed to clean up partially-initialized telemetry providers")
	}
	otel.SetTracerProvider(tracenoop.NewTracerProvider())
	otel.SetMeterProvider(metricnoop.NewMeterProvider())
	global.SetLoggerProvider(lognoop.NewLoggerProvider())
}

func initTraces(ctx context.Context, res *resource.Resource) error {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(traceSampler()),
	)
	otel.SetTracerProvider(tp)
	mu.Lock()
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	mu.Unlock()
	return nil
}

// traceSampler builds the root sampler from OTEL_TRACES_SAMPLER_ARG, the
// standard OTel env var for a sampling ratio (see
// https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).
// Unset or invalid falls back to sampling every trace, preserving this
// package's original behavior — the knob exists so ingest volume/cost can be
// dialed down later without a code change, once traffic warrants it.
func traceSampler() sdktrace.Sampler {
	ratio := 1.0
	if arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); arg != "" {
		if parsed, err := strconv.ParseFloat(arg, 64); err == nil && parsed >= 0 && parsed <= 1 {
			ratio = parsed
		} else {
			logging.WithField("value", arg).Warn("invalid OTEL_TRACES_SAMPLER_ARG, sampling every trace")
		}
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
}

func initMetrics(ctx context.Context, res *resource.Resource) error {
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	mu.Lock()
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
	mu.Unlock()
	return nil
}

func initLogs(ctx context.Context, res *resource.Resource) error {
	exporter, err := otlploghttp.New(ctx)
	if err != nil {
		return err
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)
	mu.Lock()
	shutdownFuncs = append(shutdownFuncs, lp.Shutdown)
	mu.Unlock()
	return nil
}

// Shutdown flushes and stops all configured providers. Safe to call even if
// Init was a no-op — shutdownFuncs is simply empty in that case. Safe to
// call concurrently: the shared state is drained under mu before running
// the (potentially slow) shutdown callbacks outside the lock.
func Shutdown(ctx context.Context) error {
	mu.Lock()
	fns := shutdownFuncs
	shutdownFuncs = nil
	enabled = false
	mu.Unlock()

	var errs []error
	for _, fn := range fns {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errs)
	}
	return nil
}

// RecordError records err on span and marks it failed with msg as the status
// description. Centralizes the record-error/set-status pair so every
// instrumented call site (email, GroupMe, blob storage, ...) reports
// failures the same way.
func RecordError(span trace.Span, err error, msg string) {
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}

// Fail records err on span via RecordError and returns err unchanged.
// Collapses the "record + set status + return" sequence repeated at every
// instrumented call site's error paths into a single expression, e.g.
// `return telemetry.Fail(span, fmt.Errorf("...: %w", err), "...")`.
func Fail(span trace.Span, err error, msg string) error {
	RecordError(span, err, msg)
	return err
}

// Tracer returns a named tracer via the current global TracerProvider. A
// single entry point for instrumented packages to obtain a tracer, instead
// of each one hand-declaring its own `otel.Tracer("...")` package var with
// an independently chosen name.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
