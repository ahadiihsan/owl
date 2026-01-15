package owl

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the name of the tracer.
var tracerName atomic.Value

func init() {
	tracerName.Store("github.com/myuser/owl")
}

// SetTracerName sets the name of the tracer used by owl.Start.
// apt for attributing traces to specific services.
func SetTracerName(name string) {
	tracerName.Store(name)
}

// Start starts a new span using the global OTel tracer.
// It returns a context with the span and a function to end it.
// The returned end function should be deferred, optionally passing a pointer to the error
// to automatically record it on the span.
//
// Usage:
//
//	ctx, end := owl.Start(ctx, "OperationName")
//	defer end(&err)
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func(*error)) {
	// Use the configured tracer name
	tn := tracerName.Load().(string)
	tracer := otel.Tracer(tn)

	ctx, span := tracer.Start(ctx, name, opts...)

	return ctx, func(errPtr *error) {
		if errPtr != nil && *errPtr != nil {
			err := *errPtr
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// SetBaggage sets a baggage member in the context.
func SetBaggage(ctx context.Context, key, value string) context.Context {
	m, _ := baggage.NewMember(key, value)
	b, _ := baggage.New(m) // Note: This creates new baggage with just this member.
	// Properly we should add to existing baggage.
	current := baggage.FromContext(ctx)
	// baggage.FromContext returns 'Baggage'. 'New' returns 'Baggage'.
	// To add member... OTel Baggage is immutable.
	// b, _ = current.SetMember(m) // SetMember returns Baggage, error.
	b, _ = current.SetMember(m)
	return baggage.ContextWithBaggage(ctx, b)
}

// GetBaggage returns a baggage member value from the context.
func GetBaggage(ctx context.Context, key string) string {
	b := baggage.FromContext(ctx)
	m := b.Member(key)
	return m.Value()
}
