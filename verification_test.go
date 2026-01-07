package owl_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/myuser/owl"
	"github.com/myuser/owl/logs"
	"github.com/myuser/owl/metrics"
	"github.com/myuser/owl/middleware"
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

func TestMiddlewareFlusher(t *testing.T) {
	factory := middleware.NewHTTPFactory(nil, nil)

	handler := func(w http.ResponseWriter, r *http.Request) error {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return owl.Problem(owl.Internal, owl.WithMsg("Response does not support Flusher"))
		}
		w.Header().Set("X-Custom", "1")
		w.WriteHeader(200)
		flusher.Flush()
		return nil
	}

	wrapped := factory.Wrap(handler)

	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
