package owltest

import (
	"context"
	"errors"
	"testing"
)

func TestOwlHelpers(t *testing.T) {
	// 1. Logger
	logger := NewLogger()
	ctx := context.Background()

	logger.Info(ctx, "info msg", "key", "val")
	entry := logger.LastEntry()
	if entry.Msg != "info msg" {
		t.Error("LastEntry mismatch")
	}
	if entry.Level != "INFO" {
		t.Error("Level mismatch")
	}

	logger.Error(ctx, "error msg", errors.New("err"))
	entry = logger.LastEntry()
	if entry.Level != "ERROR" {
		t.Error("Level mismatch")
	}

	if logger.String() == "" {
		t.Error("String() returned empty")
	}

	logger.Reset()
	if logger.LastEntry() != nil {
		t.Error("Reset failed")
	}

	logger.Debug(ctx, "debug")
	logger.Warn(ctx, "warn")

	// 2. Monitor
	monitor := NewMonitor()

	c := monitor.Counter("c")
	c.Inc(ctx)
	c.Add(ctx, 5)

	if monitor.GetCounter("c") != 6 {
		t.Errorf("Counter mismatch, got %v", monitor.GetCounter("c"))
	}

	h := monitor.Histogram("h")
	h.Record(ctx, 10)

	// Helper methods coverage
	// monitor.Inc("c2", nil) // Removed as it doesn't exist on TestMonitor directly
}
