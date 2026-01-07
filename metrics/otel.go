package metrics

import (
	"context"

	"github.com/myuser/owl"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OTelAdapter implements owl.Monitor using OpenTelemetry.
type OTelAdapter struct {
	meter metric.Meter
}

// NewOTelAdapter initializes an adapter with an existing OTel Meter.
// The Application Logic (main.go) is responsible for setting up the Exporter (Prometheus/OTLP)
// and the MeterProvider.
func NewOTelAdapter(meter metric.Meter) owl.Monitor {
	return &OTelAdapter{
		meter: meter,
	}
}

func (o *OTelAdapter) Counter(name string, opts ...owl.MetricOption) owl.Counter {
	// In a real impl, we'd parse opts to set description/units/tags
	c, err := o.meter.Float64Counter(name)
	if err != nil {
		// Fallback to nil internal counter (safe due to checks below)
		return &otelCounter{c: nil}
	}
	return &otelCounter{c: c}
}

func (o *OTelAdapter) Histogram(name string, opts ...owl.MetricOption) owl.Histogram {
	h, err := o.meter.Float64Histogram(name)
	if err != nil {
		return &otelHistogram{h: nil}
	}
	return &otelHistogram{h: h}
}

// Wrappers

type otelCounter struct {
	c metric.Float64Counter
}

func (c *otelCounter) Inc(ctx context.Context, attrs ...owl.Attribute) {
	if c.c != nil {
		c.c.Add(ctx, 1, metric.WithAttributes(toOtelAttrs(attrs)...))
	}
}

func (c *otelCounter) Add(ctx context.Context, delta float64, attrs ...owl.Attribute) {
	if c.c != nil {
		c.c.Add(ctx, delta, metric.WithAttributes(toOtelAttrs(attrs)...))
	}
}

type otelHistogram struct {
	h metric.Float64Histogram
}

func (h *otelHistogram) Record(ctx context.Context, value float64, attrs ...owl.Attribute) {
	if h.h != nil {
		h.h.Record(ctx, value, metric.WithAttributes(toOtelAttrs(attrs)...))
	}
}

// Helper to convert attributes
func toOtelAttrs(attrs []owl.Attribute) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	res := make([]attribute.KeyValue, len(attrs))
	for i, a := range attrs {
		res[i] = attribute.String(a.Key, a.Value)
	}
	return res
}
