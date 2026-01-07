package owl_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/myuser/owl"
	"github.com/myuser/owl/health"
	"github.com/myuser/owl/logs"
	"github.com/myuser/owl/metrics"
	"github.com/myuser/owl/middleware"
	"github.com/myuser/owl/owltest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc/codes"
)

func TestErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, http.StatusOK},
		{"generic error", errors.New("boom"), http.StatusInternalServerError},
		{"owl invalid", owl.Problem(owl.Invalid), http.StatusBadRequest},
		{"owl not found", owl.Problem(owl.NotFound), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := owl.ToHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("ToHTTPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorToGRPCStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"generic error", context.DeadlineExceeded, codes.Unknown},
		{"owl invalid", owl.Problem(owl.Invalid, owl.WithMsg("bad request")), codes.InvalidArgument},
		{"owl not found", owl.Problem(owl.NotFound, owl.WithMsg("not found")), codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := owl.ToGRPCStatus(tt.err)
			if got.Code() != tt.want {
				t.Errorf("ToGRPCStatus() code = %v, want %v", got.Code(), tt.want)
			}
		})
	}
}

func TestHTTPMiddleware(t *testing.T) {
	// Setup Dependencies
	logger := logs.NewSlogAdapter(nil) // Uses stdout

	// Start OTel Provider manually for Testing
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(mp)
	meter := otel.Meter("test-service")

	monitor := metrics.NewOTelAdapter(meter)
	factory := middleware.NewHTTPFactory(logger, monitor)

	// Custom handler
	h := func(w http.ResponseWriter, r *http.Request) error {
		return owl.Problem(
			owl.Unauthorized,
			owl.WithOp("Op.Test"),
			owl.WithMsg("internal database error"), // Internal Log
			owl.WithSafeMsg("access denied"),       // Public Safe Message
		)
	}

	// Wrap it
	handler := factory.Wrap(h)

	// Simulate request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check status
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Check JSON body
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 1. Verify Code classification
	if code, ok := resp["code"]; !ok || code != "UNAUTHORIZED" {
		t.Errorf("expected code 'UNAUTHORIZED', got %v", code)
	}

	// 2. Verify SafeMsg is used (NOT internal message)
	if msg, ok := resp["message"]; !ok || msg != "access denied" {
		t.Errorf("expected safe message 'access denied', got '%v'", msg)
	}
}

func TestClientServerHydration(t *testing.T) {
	// 1. Setup Server
	logger := logs.NewSlogAdapter(nil)
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(mp)
	meter := otel.Meter("test-service")

	monitor := metrics.NewOTelAdapter(meter)
	factory := middleware.NewHTTPFactory(logger, monitor)

	serverHandler := func(w http.ResponseWriter, r *http.Request) error {
		return owl.Problem(owl.NotFound, owl.WithMsg("document 123 not found"))
	}

	ts := httptest.NewServer(factory.Wrap(serverHandler))
	defer ts.Close()

	// 2. Setup Client
	// Wrap the transport
	client := ts.Client()
	client.Transport = middleware.NewHTTPClient(client.Transport, logger)

	// 3. Make Request
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}

	// 4. Hydrate Error
	defer resp.Body.Close()
	hydratedErr := middleware.CheckResponse(resp)

	if hydratedErr == nil {
		t.Fatal("expected hydrated error, got nil")
	}

	// 5. Verify Hydration
	var owlErr *owl.Error
	if asErr, ok := hydratedErr.(*owl.Error); ok {
		owlErr = asErr
	} else {
		t.Fatalf("expected *owl.Error, got %T", hydratedErr)
	}

	if owlErr.Code != owl.NotFound {
		t.Errorf("expected CodeNotFound, got %v", owlErr.Code)
	}
	// Verify behavior of msg when not set specifically (CheckResponse logic)
	// The problem in middleware.CheckResponse creates a new error if status is hydration fallback.
	// But if it decoded the JSON, it gets the fields from the JSON.
	// The server sent JSON with: Code="NOT_FOUND", Message="NOT_FOUND" (because SafeMsg not set).
	// So we expect "NOT_FOUND".
	if owlErr.Msg != "NOT_FOUND" {
		t.Errorf("expected public message 'NOT_FOUND', got '%v'", owlErr.Msg)
	}
}

func TestErrorIs(t *testing.T) {
	err1 := owl.Problem(owl.NotFound, owl.WithMsg("not found"))

	if !errors.Is(err1, owl.CodeNotFound) {
		t.Error("errors.Is(err1, owl.CodeNotFound) should be true")
	}

	err2 := owl.Problem(owl.Internal)
	if errors.Is(err2, owl.NotFound) {
		t.Error("errors.Is(err2, owl.NotFound) should be false")
	}

	baseErr := errors.New("underlying error")
	err3 := owl.Problem(owl.Internal, owl.WithErr(baseErr))

	if !errors.Is(err3, baseErr) {
		t.Error("errors.Is(err3, baseErr) should be true")
	}
}

func TestMiddlewareNilSafety(t *testing.T) {
	factory := middleware.NewHTTPFactory(nil, nil)

	h := func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}

	wrapped := factory.Wrap(h)

	req := httptest.NewRequest("GET", "/nil", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

// --- Roadmap Feature Tests ---

func TestOwlGoSafety(t *testing.T) {
	// Use owltest helpers
	testLogger := owltest.NewLogger()
	testMonitor := owltest.NewMonitor()

	// Set Singletons (and reset after test if we cared, but tests are isolated binaries usually)
	owl.SetLogger(testLogger)
	owl.SetMonitor(testMonitor)

	done := make(chan struct{})

	// Use PanicHandler to sync, as it runs AFTER logging
	owl.SetPanicHandler(func(ctx context.Context, r any) {
		close(done)
	})

	// Run unsafe code safely
	owl.Go(context.Background(), func(ctx context.Context) {
		panic("oops")
	})

	<-done

	// Verify it didn't crash (we are here)
	// Verify logs
	entry := testLogger.LastEntry()
	if entry == nil {
		t.Fatal("expected log entry for panic")
	}
	if entry.Msg != "goroutine_panic" || entry.Level != "ERROR" {
		t.Errorf("unexpected log: %+v", entry)
	}

	// Verify metrics
	if val := testMonitor.GetCounter("goroutine_panic_total"); val != 1 {
		t.Errorf("expected counter 1, got %v", val)
	}
}

func TestOwlStart(t *testing.T) {
	ctx := context.Background()

	// We can't easily assert OTel internal state without a global span exporter hook or mocking the TracerProvider.
	// But we can check that it doesn't panic and returns a valid context.
	ctx, end := owl.Start(ctx, "TestSpan")
	defer func() {
		// Test error recording
		err := errors.New("span error")
		end(&err)
	}()

	if ctx == nil {
		t.Error("expected non-nil context")
	}
}

func TestHealthPackage(t *testing.T) {
	// Mock Checkers
	dbCheck := health.CheckerFunc(func(ctx context.Context) error {
		return nil
	})
	redisCheck := health.CheckerFunc(func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	// 1. All Good
	h := health.Handler(map[string]health.Checker{
		"db": dbCheck,
	})
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Errorf("expected json ok:true, got %s", w.Body.String())
	}

	// 2. Failure
	hFail := health.Handler(map[string]health.Checker{
		"db":    dbCheck,
		"redis": redisCheck,
	})
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	hFail.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok":false`) {
		t.Errorf("expected json ok:false, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"connection refused"`) {
		t.Errorf("expected error details, got %s", w.Body.String())
	}
}
