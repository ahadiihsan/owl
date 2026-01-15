package owl

import (
	"context"
	"fmt"
	"runtime/debug"
)

// PanicHandler is a function that handles panics.
type PanicHandler func(ctx context.Context, r any)

var panicHandler PanicHandler

// SetPanicHandler sets a global panic handler.
func SetPanicHandler(h PanicHandler) {
	panicHandler = h
}

// Go starts a safe goroutine.
func Go(ctx context.Context, fn func(ctx context.Context)) {
	// Check context before starting to avoid unnecessary goroutine spawn if already cancelled
	if ctx.Err() != nil {
		return
	}
	go func() {
		// Double-check inside in case of race during spawn
		if ctx.Err() != nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				// Log the panic
				// SAFEGUARD: If logger itself panics or is nil (though initialized in init), ensure we don't crash again.
				// We assume GetLogger() is safe as per current globals.go, but a defer here is good practice.
				func() {
					defer func() { recover() }() // Swallow panic during logging
					GetLogger().Error(ctx, "goroutine_panic", nil,
						"panic", fmt.Sprintf("%v", r),
						"stack", stack,
					)
				}()

				// Metric
				GetMonitor().Counter("goroutine_panic_total").Inc(ctx)

				// User handler
				if panicHandler != nil {
					panicHandler(ctx, r)
				}
			}
		}()
		fn(ctx)
	}()
}
