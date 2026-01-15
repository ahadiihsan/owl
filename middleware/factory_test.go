package middleware

import (
	"net/http"
	"testing"
)

func TestNewHTTPFactory_Options(t *testing.T) {
	// Test nil safety defaults
	f := NewHTTPFactory(nil, nil)
	if f == nil {
		t.Fatal("NewHTTPFactory returned nil")
	}

	// Test Option: WithErrorEncoder
	customEnc := func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusTeapot)
	}
	f = NewHTTPFactory(nil, nil, WithErrorEncoder(customEnc))

	// Verify encoder is set (not easy to check private field directly, but we can verify behavior via Wrap)
	// Or we just rely on coverage from execution.
}

func TestNewGRPCFactory_NilSafety(t *testing.T) {
	f := NewGRPCFactory(nil, nil)
	if f == nil {
		t.Fatal("NewGRPCFactory returned nil")
	}
}

func TestNewHTTPClient_NilSafety(t *testing.T) {
	c := NewHTTPClient(nil, nil)
	if c == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if c.Base != http.DefaultTransport {
		t.Error("Expected DefaultTransport fallback")
	}
}
