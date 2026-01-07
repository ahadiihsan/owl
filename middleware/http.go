package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/myuser/owl"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// HTTPHandler is a signature that returns an error, allowing specific error handling.
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// responseWriter is a wrapper to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher interface to allow streaming.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// ErrorEncoder defines how errors are written to the response.
type ErrorEncoder func(w http.ResponseWriter, r *http.Request, err error)

// HTTPFactory allows injecting dependencies (Logger, Monitor) into the middleware.
type HTTPFactory struct {
	logger       owl.Logger
	monitor      owl.Monitor
	errorEncoder ErrorEncoder
}

// NewHTTPFactory creates a factory for middlewares.
func NewHTTPFactory(l owl.Logger, m owl.Monitor, opts ...func(*HTTPFactory)) *HTTPFactory {
	if l == nil {
		l = owl.NoOpLogger{}
	}
	if m == nil {
		m = owl.NoOpMonitor{}
	}

	f := &HTTPFactory{
		logger:       l,
		monitor:      m,
		errorEncoder: defaultErrorEncoder,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// WithErrorEncoder sets a custom error encoder.
func WithErrorEncoder(enc ErrorEncoder) func(*HTTPFactory) {
	return func(f *HTTPFactory) {
		f.errorEncoder = enc
	}
}

// defaultErrorEncoder writes JSON responses.
func defaultErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	status := owl.ToHTTPStatus(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var obsErr *owl.Error
	if errors.As(err, &obsErr) {
		// Marshal semantic error
		_ = json.NewEncoder(w).Encode(obsErr)
	} else {
		// Obscure internal errors
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "INTERNAL",
			"message": "Internal Server Error",
		})
	}
}

// Wrap wraps a custom HTTPHandler and converts it to standard http.Handler.
func (f *HTTPFactory) Wrap(h HTTPHandler) http.Handler {
	// Pre-allocate metrics
	reqCount := f.monitor.Counter("http_requests_total")
	reqLatency := f.monitor.Histogram("http_request_duration_seconds")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 1. Trace Extraction
		// Extract trace context from headers and inject into request context
		ctx := r.Context()
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// 2. Panic Recovery
		defer func() {
			if rec := recover(); rec != nil {
				duration := time.Since(start).Seconds()
				f.logger.Error(ctx, "panic recovered", nil, "panic", rec)

				// Metrics
				reqCount.Inc(ctx, owl.Attr("status", "500"), owl.Attr("panic", "true"))
				reqLatency.Record(ctx, duration, owl.Attr("status", "500"), owl.Attr("panic", "true"))

				// Return 500
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":    "INTERNAL",
					"message": "Internal Server Error",
				})
			}
		}()

		// 2. Execution
		err := h(rw, r)
		duration := time.Since(start).Seconds()

		// 3. Error Handling
		if err != nil {
			status := owl.ToHTTPStatus(err)
			rw.status = status // Update status for access logs if needed

			// Determine log level and content
			// We log the FULL details (Msg, Err) internally
			var obsErr *owl.Error
			isObsErr := false
			if asErr, ok := err.(*owl.Error); ok {
				obsErr = asErr
				isObsErr = true
			}

			// Fields for structured logging
			fields := []any{
				"status", status,
				"duration", duration,
				"method", r.Method,
				"path", r.URL.Path,
			}

			if isObsErr {
				// Log the internal message + details
				f.logger.Error(ctx, obsErr.Msg, obsErr.Err, fields...)
			} else {
				f.logger.Error(ctx, "request_failed", err, fields...)
			}

			// Write Response for Client using Encoder
			f.errorEncoder(w, r, err)
		} else {
			// 4. Success Logging
			f.logger.Info(ctx, "request_success",
				"status", rw.status,
				"duration", duration,
				"method", r.Method,
				"path", r.URL.Path,
			)
		}

		// Update Metrics
		reqCount.Inc(ctx,
			owl.Attr("method", r.Method),
			owl.Attr("path", r.URL.Path),
			// Convert status to string (Improvement: use numeric code, not StatusText)
			owl.Attr("status", strconv.Itoa(rw.status)),
		)
		reqLatency.Record(ctx, duration,
			owl.Attr("method", r.Method),
			owl.Attr("path", r.URL.Path),
			owl.Attr("status", strconv.Itoa(rw.status)),
		)
	})
}
