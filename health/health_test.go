package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Create handler with a mock check
	handler := Handler(map[string]Checker{
		"db": CheckerFunc(func(ctx context.Context) error {
			return nil
		}),
	})

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	checks := body["checks"].(map[string]any)

	if checks["db"] != "ok" {
		t.Errorf("Expected db ok, got %v", checks["db"])
	}
}
