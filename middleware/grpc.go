package middleware

import (
	"context"
	"time"

	"github.com/myuser/owl"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCFactory allows injecting dependencies.
type GRPCFactory struct {
	logger  owl.Logger
	monitor owl.Monitor
}

// NewGRPCFactory creates a new factory.
func NewGRPCFactory(l owl.Logger, m owl.Monitor) *GRPCFactory {
	if l == nil {
		l = owl.NoOpLogger{}
	}
	if m == nil {
		m = owl.NoOpMonitor{}
	}
	return &GRPCFactory{logger: l, monitor: m}
}

// UnaryServerInterceptor returns a new interceptor.
func (f *GRPCFactory) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	reqCount := f.monitor.Counter("grpc_requests_total")
	reqLatency := f.monitor.Histogram("grpc_request_duration_seconds")

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 1. Trace Extraction
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataSupplier{md})
		}

		start := time.Now()

		// 2. Execution
		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		// 3. Match code
		codeStr := "OK"
		if err != nil {
			if s, ok := status.FromError(err); ok {
				codeStr = s.Code().String()
			} else {
				codeStr = "UNKNOWN"
			}
		}

		// 4. Metrics
		reqCount.Inc(ctx,
			owl.Attr("method", info.FullMethod),
			owl.Attr("code", codeStr),
		)
		reqLatency.Record(ctx, duration,
			owl.Attr("method", info.FullMethod),
			owl.Attr("code", codeStr),
		)

		// 5. Error Handling
		if err != nil {
			// Convert to gRPC Status
			gst := owl.ToGRPCStatus(err)

			// Log internal error with full details
			// If it's an ObsError, we have rich details
			var obsErr *owl.Error
			if e, ok := err.(*owl.Error); ok {
				obsErr = e
				f.logger.Error(ctx, obsErr.Msg, obsErr.Err,
					"code", gst.Code().String(),
					"duration", duration,
					"method", info.FullMethod,
				)
			} else {
				f.logger.Error(ctx, "grpc_request_failed", err,
					"code", gst.Code().String(),
					"duration", duration,
					"method", info.FullMethod,
				)
			}

			// Return the converted status error (which contains SafeMsg)
			return nil, gst.Err()
		}

		// 4. Success Logging
		f.logger.Info(ctx, "grpc_request_success",
			"code", "OK",
			"duration", duration,
			"method", info.FullMethod,
		)

		return resp, nil
	}
}
