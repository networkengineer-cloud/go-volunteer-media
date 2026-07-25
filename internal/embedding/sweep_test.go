package embedding

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func heartbeatTotal(t *testing.T, reader *sdkmetric.ManualReader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "embedding.sweep.heartbeat" {
				continue
			}
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				for _, dp := range sum.DataPoints {
					total += dp.Value
				}
			}
		}
	}
	return total
}

// TestStartReconciliationSweep_EmitsHeartbeatEveryTickEvenWhenUnusable proves
// the heartbeat fires unconditionally — including on ticks where the sweep
// itself does nothing because the embedder isn't usable. This is the case
// that matters: the heartbeat exists to detect the scheduler goroutine dying
// entirely, so it must not depend on a successful (or even attempted) sweep.
func TestStartReconciliationSweep_EmitsHeartbeatEveryTickEvenWhenUnusable(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(prev)

	interval := 5 * time.Millisecond
	// nil db is safe here: Unconfigured makes Usable() false, so
	// sweepAnimals/sweepComments/sweepUpdates (the only callers that touch
	// db) are never invoked.
	stop := StartReconciliationSweep(nil, &StubEmbedder{Unconfigured: true}, interval)
	defer stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for heartbeatTotal(t, reader) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("expected at least 2 heartbeat ticks within 500ms at a %s interval, got %d", interval, heartbeatTotal(t, reader))
		}
		time.Sleep(interval)
	}
}
