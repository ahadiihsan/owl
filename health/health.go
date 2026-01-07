package health

import (
	"context"
	"encoding/json"
	"net/http"
)

// Checker checks the health of a component.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckerFunc allows a simple function to be used as a Checker.
type CheckerFunc func(ctx context.Context) error

func (f CheckerFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// Handler returns a standard JSON health handler.
// It iterates over the provided checks map.
// If any check fails, it returns 503 and the error details.
// If all pass, it returns 200.
func Handler(checks map[string]Checker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		results := make(map[string]string)

		ctx := r.Context()

		for name, checker := range checks {
			if err := checker.Check(ctx); err != nil {
				status = http.StatusServiceUnavailable
				results[name] = err.Error()
			} else {
				results[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     status == http.StatusOK,
			"checks": results,
		})
	})
}
