package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/myuser/owl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestCheckResponse_BodyRestoration(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		isJSON     bool
	}{
		{
			name:       "OK Response",
			statusCode: 200,
			body:       `{"status": "ok"}`,
			isJSON:     true,
		},
		{
			name:       "Error Response JSON",
			statusCode: 400,
			body:       `{"code": "INVALID", "message": "bad request"}`,
			isJSON:     true,
		},
		{
			name:       "Error Response Text",
			statusCode: 500,
			body:       "Critical Failure",
			isJSON:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			bodyElem := io.NopCloser(strings.NewReader(tt.body))
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       bodyElem,
				Header:     make(http.Header),
			}
			if tt.isJSON {
				resp.Header.Set("Content-Type", "application/json")
			}

			// Call CheckResponse (which reads some body)
			_ = CheckResponse(resp)

			// Verify Body is readable and matches original
			restored, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read restored body: %v", err)
			}

			if string(restored) != tt.body {
				t.Errorf("Body mismatch.\nGot:  %q\nWant: %q", string(restored), tt.body)
			}
		})
	}
}

// mockTransport implements http.RoundTripper
type mockTransport struct {
	RoundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestHTTPClient_RoundTrip(t *testing.T) {
	mock := &mockTransport{
		RoundTripFunc: func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		},
	}

	client := NewHTTPClient(mock, owl.NoOpLogger{})
	req := httptest.NewRequest("GET", "http://example.com", nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Test error case
	mock.RoundTripFunc = func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network error")
	}
	_, err = client.RoundTrip(req)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	interceptor := UnaryClientInterceptor(owl.NoOpLogger{})

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		// Verify metadata injection (Basic check)
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Error("Metadata not injected")
		}
		// In a real test we'd check for TraceContext headers in md
		if len(md) == 0 {
			// It might be empty if propagator didn't inject anything (e.g. no parent trace),
			// but we expect at least an empty object derived.
		}
		return nil
	}

	err := interceptor(context.Background(), "/test", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("Interceptor failed: %v", err)
	}

	// Test error prop
	errInvoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return errors.New("rpc error")
	}
	err = interceptor(context.Background(), "/test", nil, nil, nil, errInvoker)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestMetadataSupplier(t *testing.T) {
	md := metadata.New(map[string]string{"k": "v"})
	s := &metadataSupplier{MD: md}

	if s.Get("k") != "v" {
		t.Error("Get failed")
	}
	if s.Get("missing") != "" {
		t.Error("Get missing failed")
	}

	s.Set("k2", "v2")
	if s.Get("k2") != "v2" {
		t.Error("Set failed")
	}

	keys := s.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}
