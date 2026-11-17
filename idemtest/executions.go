package idemtest

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
)

// Executions counts how many times a side effect ran. Wire its adapters into
// the pipeline under test, then assert with [RequireExecutions]: the store
// reports which attempts were rejected, the counter reports what actually
// executed — the observable half of at-most-once processing.
//
// Counting is safe for concurrent use, so one Executions serves both the
// command adapter ([Executions.Fn], [Executions.Wrap]) and the HTTP adapter
// ([Counting]).
type Executions struct {
	count atomic.Int64
}

// NewExecutions returns a zeroed counter.
func NewExecutions() *Executions {
	return &Executions{} //nolint:exhaustruct_v5 // count is a valid zero atomic
}

// Fn returns a no-op side effect that counts as one execution. Pass it where
// a middleware.NewCommand execute function is expected and the drills can
// then assert how many times the command body ran.
func (e *Executions) Fn() func(context.Context) error {
	return func(context.Context) error {
		e.count.Add(1)

		return nil
	}
}

// Wrap returns fn wrapped so every call counts as one execution before fn
// runs. Use it when the side effect under test has real work to do.
func (e *Executions) Wrap(fn func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		e.count.Add(1)

		return fn(ctx)
	}
}

// Total reports how many executions have been counted.
func (e *Executions) Total() int {
	return int(e.count.Load())
}

// Counting returns an http.Handler that counts one execution per request
// served, then delegates to next — the adapter for HTTP drills, where a
// processed request is an executed side effect.
func Counting(executions *Executions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executions.count.Add(1)

		next.ServeHTTP(w, r)
	})
}

// RequireExecutions fails tb unless exactly want side effects have run.
func RequireExecutions(tb testing.TB, executions *Executions, want int) {
	tb.Helper()

	if got := executions.Total(); got != want {
		tb.Fatalf("side effects executed = %d, want %d", got, want)
	}
}
