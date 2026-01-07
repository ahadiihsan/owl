package logs

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// Sanitizer is a function that can redact or modify field values.
type Sanitizer func(key string, value any) any

// SlogAdapter implements owl.Logger using log/slog.
type SlogAdapter struct {
	logger    *slog.Logger
	sanitizer Sanitizer
}

// NewSlogAdapter creates a new logger adapter.
func NewSlogAdapter(l *slog.Logger, opts ...func(*SlogAdapter)) *SlogAdapter {
	if l == nil {
		l = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	s := &SlogAdapter{logger: l}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithSanitizer sets the sanitizer hook.
func WithSanitizer(fn Sanitizer) func(*SlogAdapter) {
	return func(s *SlogAdapter) {
		s.sanitizer = fn
	}
}

// helper to extract context
func (s *SlogAdapter) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	// 1. Sanitize Args
	if s.sanitizer != nil && len(args) > 1 {
		// Args are key-value pairs (string, any)
		for i := 0; i < len(args)-1; i += 2 {
			key, ok := args[i].(string)
			if ok {
				args[i+1] = s.sanitizer(key, args[i+1])
			}
		}
	}

	logger := s.logger

	// Extract TraceID
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger = logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	// Extract Baggage (Business Context)
	bag := baggage.FromContext(ctx)
	if bag.Len() > 0 {
		// Baggage members to log fields
		// We sort them for consistency or just iterate
		// Note: Baggage iteration order is not guaranteed by default in all versions,
		// but typically we just add them.
		members := bag.Members()
		// To be deterministic for testing or cleaner logs, we could sort, but OTel baggage doesn't expose a sorted slice directly easily
		// Let's just iterate.
		for _, member := range members {
			logger = logger.With(slog.String(member.Key(), member.Value()))
		}
	}

	logger.Log(ctx, level, msg, args...)
}

func (s *SlogAdapter) Debug(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelDebug, msg, args...)
}

func (s *SlogAdapter) Info(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelInfo, msg, args...)
}

func (s *SlogAdapter) Warn(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelWarn, msg, args...)
}

func (s *SlogAdapter) Error(ctx context.Context, msg string, err error, args ...any) {
	// We append the error to args automatically if it's not nil
	if err != nil {
		args = append(args, "error", err.Error())
	}
	s.log(ctx, slog.LevelError, msg, args...)
}

// Global default for convenience (optional)
var Default = NewSlogAdapter(nil)
