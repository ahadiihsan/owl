package owl_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/myuser/owl"
)

func TestGo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var executed bool
	var wg sync.WaitGroup
	wg.Add(1)

	// We can't easily wait for "not running", so we wait a bit.
	// But since we want to test that it *doesn't* run, we can set a flag.
	// To verify logic, we must ensure owl.Go doesn't block.

	// Because owl.Go is async, if we cancel first, it should return immediately.
	// But how do we prove the goroutine didn't start?
	// We can't prove negative easily without internal hooks.
	// However, we can prove that IF logic runs, it respects context.

	done := make(chan struct{})

	owl.Go(ctx, func(ctx context.Context) {
		// This should theoretically NOT run if our fix works at the top level
		// Or if it runs, it checks context inside.
		executed = true
		close(done)
	})

	select {
	case <-done:
		t.Error("owl.Go executed function despite context cancellation")
	case <-time.After(50 * time.Millisecond):
		// Pass implies it didn't run or hasn't run yet.
	}

	if executed {
		t.Error("Variable executed set to true")
	}
}

func TestGo_PanicRecovery(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx := context.Background()
	owl.SetPanicHandler(func(ctx context.Context, r any) {
		defer wg.Done()
		if r != "boom" {
			t.Errorf("Expected panic 'boom', got %v", r)
		}
	})

	owl.Go(ctx, func(ctx context.Context) {
		panic("boom")
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for panic handler")
	}
}
