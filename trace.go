package owl

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

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
	// Use a named tracer for the library/framework
	tracer := otel.Tracer("github.com/myuser/owl")

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
