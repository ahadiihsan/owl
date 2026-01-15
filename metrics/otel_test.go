package metrics

import (
	"context"
	"testing"

	"github.com/myuser/owl"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestOTelAdapter(t *testing.T) {
	// Use NoOp meter provider for basic coverage (verifies it calls OTel API without panic)
	provider := noop.NewMeterProvider()
	meter := provider.Meter("test")

	adapter := NewOTelAdapter(meter)

	ctx := context.Background()

	t.Run("Counter", func(t *testing.T) {
		counter := adapter.Counter("test_counter")
		// noop implementation doesn't record, but we verify calling conventions
		counter.Inc(ctx, owl.Attr("key", "val"))
		counter.Add(ctx, 5, owl.Attr("key", "val"))
	})

	t.Run("Histogram", func(t *testing.T) {
		histo := adapter.Histogram("test_histogram")
		histo.Record(ctx, 100, owl.Attr("key", "val"))
	})
}
