package owl_test

import (
	"context"
	"testing"

	"github.com/myuser/owl"
)

func TestBaggage(t *testing.T) {
	ctx := context.Background()
	ctx = owl.SetBaggage(ctx, "user_id", "12345")

	val := owl.GetBaggage(ctx, "user_id")
	if val != "12345" {
		t.Errorf("Expected baggage '12345', got '%s'", val)
	}

	val = owl.GetBaggage(ctx, "non_existent")
	if val != "" {
		t.Errorf("Expected empty baggage, got '%s'", val)
	}
}
