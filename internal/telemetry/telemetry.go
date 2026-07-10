package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"

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

// shutdownFuncs holds cleanup callbacks for whichever providers Init configured.
// Empty when Init was a no-op (OTEL_EXPORTER_OTLP_ENDPOINT unset).
var shutdownFuncs []func(context.Context) error

// Init configures global OpenTelemetry trace, metric, and log providers from
// OTEL_EXPORTER_OTLP_ENDPOINT and the other standard OTEL_EXPORTER_OTLP_*
// env vars (read automatically by the OTLP exporters). If the endpoint is
// unset, Init leaves OTel's default no-op providers in place, so the app
// behaves identically with zero telemetry configuration (local dev, CI).
// Init never returns an error that should stop application startup — errors
// are logged and treated as "telemetry disabled".
func Init(ctx context.Context, serviceName, environment string) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logging.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set, telemetry disabled (using no-op providers)")
		return nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironmentName(environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to build otel resource: %w", err)
	}

	if err := initTraces(ctx, res); err != nil {
		return fmt.Errorf("failed to init tracing: %w", err)
	}
	if err := initMetrics(ctx, res); err != nil {
		return fmt.Errorf("failed to init metrics: %w", err)
	}
	if err := initLogs(ctx, res); err != nil {
		return fmt.Errorf("failed to init logs: %w", err)
	}

	logging.WithField("endpoint", endpoint).Info("OpenTelemetry initialized")
	return nil
}

func initTraces(ctx context.Context, res *resource.Resource) error {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	return nil
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
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
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
	shutdownFuncs = append(shutdownFuncs, lp.Shutdown)
	return nil
}

// Shutdown flushes and stops all configured providers. Safe to call even if
// Init was a no-op — shutdownFuncs is simply empty in that case.
func Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range shutdownFuncs {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	shutdownFuncs = nil
	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errs)
	}
	return nil
}
