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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				// Log the panic
				GetLogger().Error(ctx, "goroutine_panic", nil,
					"panic", fmt.Sprintf("%v", r),
					"stack", stack,
				)

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
