package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/myuser/owl"
)

func TestHTTPFactory_Wrap(t *testing.T) {
	logger := owl.NoOpLogger{}
	monitor := owl.NoOpMonitor{}

	f := NewHTTPFactory(logger, monitor)

	handler := func(w http.ResponseWriter, r *http.Request) error {
		if r.URL.Path == "/panic" {
			panic("oops")
		}
		if r.URL.Path == "/error" {
			return errors.New("fail")
		}
		if r.URL.Path == "/owl_error" {
			return owl.Problem(owl.CodeInvalid, owl.WithMsg("invalid input"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return nil
	}

	h := f.Wrap(handler)

	// Case 1: OK
	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Case 2: Error
	req = httptest.NewRequest("GET", "/error", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}

	// Case 3: Owl Error
	req = httptest.NewRequest("GET", "/owl_error", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}

	// Case 4: Panic
	req = httptest.NewRequest("GET", "/panic", nil)
	w = httptest.NewRecorder()
	// Panic is recovered by middleware
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Should not happen if middleware catches it
				t.Error("Middleware failed to catch panic")
			}
		}()
		h.ServeHTTP(w, req)
	}()
	if w.Code != 500 {
		t.Errorf("Expected 500 for panic, got %d", w.Code)
	}

	// ResponseWriter flush coverage
	rw := &responseWriter{ResponseWriter: w}
	rw.Flush() // Should not panic
}
