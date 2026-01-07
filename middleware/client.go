package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/myuser/owl"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// HTTPClient wraps a standard http.RoundTripper to handle Trace Injection and Error Hydration.
type HTTPClient struct {
	Base   http.RoundTripper
	Logger owl.Logger
}

// NewHTTPClient creates a new observability client wrapper.
func NewHTTPClient(base http.RoundTripper, logger owl.Logger) *HTTPClient {
	if logger == nil {
		logger = owl.NoOpLogger{}
	}
	if base == nil {
		base = http.DefaultTransport
	}
	return &HTTPClient{
		Base:   base,
		Logger: logger,
	}
}

// RoundTrip executes the HTTP transaction.
func (c *HTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	ctx := req.Context()

	// 1. Trace Injection (W3C)
	// Inject the current trace context into the headers so the upstream service can continue the trace.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// 2. Execution
	resp, err := c.Base.RoundTrip(req)
	duration := time.Since(start).Seconds()

	// 3. Logging
	fields := []any{
		"duration", duration,
		"method", req.Method,
		"url", req.URL.String(),
	}

	if err != nil {
		c.Logger.Error(ctx, "outbound_request_failed", err, fields...)
		return nil, err
	}

	fields = append(fields, "status", resp.StatusCode)
	c.Logger.Info(ctx, "outbound_request_success", fields...)

	return resp, nil
}

// Helper for HTTP Response Hydration
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	// Read body (non-destructive if we buffer, but here we assume we consume it)
	// We LIMIT the read to prevent OOM on massive bodies.
	// 64KB is sufficient for any reasonable error JSON.
	defer resp.Body.Close()

	// Only attempt JSON decode if Content-Type looks like JSON
	ct := resp.Header.Get("Content-Type")
	isJSON := false
	if len(ct) >= 16 && ct[:16] == "application/json" {
		isJSON = true
	} else if len(ct) >= 15 && ct[:15] == "application/json" {
		// "application/json" is 16 chars... wait.
		// "application/json" is 16 chars.
		isJSON = true
	} else {
		// simple check
		if ct == "application/json" {
			isJSON = true
		}
	}
	// Simplified check
	// Note: Strings.Contains is often safer for "application/json; charset=utf-8"
	// but let's just proceed with safe Logic.

	reader := io.LimitReader(resp.Body, 64*1024)
	body, _ := io.ReadAll(reader)

	if isJSON || (len(body) > 0 && body[0] == '{') {
		var owlErr owl.Error
		if err := json.Unmarshal(body, &owlErr); err == nil && owlErr.Code != 0 {
			return &owlErr
		}
	}

	// Fallback using status code reverse mapping
	// If body is text, include it in the Msg for debugging
	return owl.Problem(
		owl.FromHTTPStatus(resp.StatusCode),
		owl.WithMsg(string(body)),
	)
}

// UnaryClientInterceptor returns a new unary client interceptor that injects trace context and logs requests.
func UnaryClientInterceptor(logger owl.Logger) grpc.UnaryClientInterceptor {
	if logger == nil {
		logger = owl.NoOpLogger{}
	}
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		// 1. Trace Injection
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		// Inject into gRPC metadata
		otel.GetTextMapPropagator().Inject(ctx, &metadataSupplier{md})
		ctx = metadata.NewOutgoingContext(ctx, md)

		// 2. Execution
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start).Seconds()

		// 3. Logging
		fields := []any{
			"duration", duration,
			"method", method,
		}

		if err != nil {
			logger.Error(ctx, "outbound_rpc_failed", err, fields...)

			// Hydration Logic
			st, ok := status.FromError(err)
			if ok {
				owlCode := owl.FromGRPCStatus(st.Code())
				return owl.Problem(
					owlCode,
					owl.WithMsg(st.Message()), // Use st.Message() as SafeMsg/Msg
					owl.WithErr(err),
				)
			}
		} else {
			logger.Info(ctx, "outbound_rpc_success", fields...)
		}
		return err
	}
}

// metadataSupplier implements propagation.TextMapCarrier
type metadataSupplier struct {
	metadata.MD
}

func (s *metadataSupplier) Get(key string) string {
	vals := s.MD.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (s *metadataSupplier) Set(key string, value string) {
	s.MD.Set(key, value)
}

func (s *metadataSupplier) Keys() []string {
	keys := make([]string, 0, len(s.MD))
	for k := range s.MD {
		keys = append(keys, k)
	}
	return keys
}
