package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/myuser/owl"
	"github.com/myuser/owl/logs"
	"github.com/myuser/owl/metrics"
	"github.com/myuser/owl/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	// 1. Setup Logger (Slog Adapter)
	logger := logs.NewSlogAdapter(nil) // Defaults to JSON/Stdout
	ctx := context.Background()
	logger.Info(ctx, "starting example server...")

	// 2. Setup Metrics (OTel)
	// In production, use an OTLP exporter. Here we use stdout for demonstration.
	exp, err := stdoutmetric.New()
	if err != nil {
		log.Fatalf("failed to create metric exporter: %v", err)
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exp, metric.WithInterval(10*time.Second))),
	)
	defer func() { _ = meterProvider.Shutdown(ctx) }()
	otel.SetMeterProvider(meterProvider)

	// usage: provide the actual Meter interface to the adapter
	meter := otel.Meter("example-service")
	monitor := metrics.NewOTelAdapter(meter)

	// 3. Setup Middleware
	factory := middleware.NewHTTPFactory(logger, monitor)

	// 4. Create Handler
	// This handler sometimes returns success, sometimes specific errors
	h := func(w http.ResponseWriter, r *http.Request) error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		// Check for query param 'error'
		q := r.URL.Query().Get("error")
		switch q {
		case "not_found":
			return owl.Problem(owl.NotFound, owl.WithMsg("requested resource X missing"), owl.WithSafeMsg("Item not found"))
		case "internal":
			return owl.Problem(owl.Internal, owl.WithMsg("db connection failed"), owl.WithSafeMsg("Something went wrong"))
		case "panic":
			panic("unexpected crash")
		}

		w.Write([]byte("Hello, Owl!"))
		return nil
	}

	// 5. Wrap & Start Server
	mux := http.NewServeMux()
	mux.Handle("/", factory.Wrap(h))

	addr := ":8080"
	logger.Info(ctx, "server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error(ctx, "server failed", err)
		os.Exit(1)
	}
}
