package owl_test

import (
	"context"
	"testing"

	"github.com/myuser/owl"
)

func TestSetTracerName(t *testing.T) {
	// Default name is "github.com/myuser/owl" (from init)
	// We can't easily check the internal tracer name via public API without reflection or a mock provider.
	// But we can verify setting it doesn't panic.

	owl.SetTracerName("my-service")

	ctx, end := owl.Start(context.Background(), "test-span")
	defer end(nil)

	if ctx == nil {
		t.Error("Start returned nil context")
	}
}
