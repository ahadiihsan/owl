package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

func TestSlogAdapter(t *testing.T) {
	// Setup capture buffer
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := NewSlogAdapter(logger)
	ctx := context.Background()

	t.Run("Info Log", func(t *testing.T) {
		buf.Reset()
		adapter.Info(ctx, "hello world", "key", "value")

		var logEntry map[string]any
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to unmarshal log: %v", err)
		}

		if logEntry["msg"] != "hello world" {
			t.Errorf("Expected msg 'hello world', got %v", logEntry["msg"])
		}
		if logEntry["level"] != "INFO" {
			t.Errorf("Expected level INFO, got %v", logEntry["level"])
		}
		if logEntry["key"] != "value" {
			t.Errorf("Expected key 'value', got %v", logEntry["key"])
		}
	})

	t.Run("Debug Log", func(t *testing.T) {
		buf.Reset()
		adapter.Debug(ctx, "debugging")

		var logEntry map[string]any
		json.Unmarshal(buf.Bytes(), &logEntry)

		if logEntry["level"] != "DEBUG" {
			t.Errorf("Expected level DEBUG, got %v", logEntry["level"])
		}
	})

	t.Run("Warn Log", func(t *testing.T) {
		buf.Reset()
		adapter.Warn(ctx, "warning")

		var logEntry map[string]any
		json.Unmarshal(buf.Bytes(), &logEntry)

		if logEntry["level"] != "WARN" {
			t.Errorf("Expected level WARN, got %v", logEntry["level"])
		}
	})

	t.Run("Error Log", func(t *testing.T) {
		buf.Reset()
		mockErr := errors.New("mock error")
		adapter.Error(ctx, "oops", mockErr, "user", "123")

		var logEntry map[string]any
		json.Unmarshal(buf.Bytes(), &logEntry)

		if logEntry["level"] != "ERROR" {
			t.Errorf("Expected level ERROR, got %v", logEntry["level"])
		}
		if logEntry["error"] != "mock error" {
			t.Errorf("Expected error 'mock error', got %v", logEntry["error"])
		}
	})
}

func TestSlogAdapter_Sanitizer(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	// Sanitizer that redacts "token"
	sanitizer := func(key string, value any) any {
		if key == "token" {
			return "***"
		}
		return value
	}

	adapter := NewSlogAdapter(logger, WithSanitizer(sanitizer))
	ctx := context.Background()

	adapter.Info(ctx, "login", "token", "secret123")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["token"] != "***" {
		t.Errorf("Expected redacted token, got %v", logEntry["token"])
	}
}
