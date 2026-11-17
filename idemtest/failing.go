package idemtest

import (
	"context"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Failing is a [idempotency.Store] whose methods return Err instead of
// consulting Next — the deterministic driver for fail-closed drills. With Err
// set, every call fails: a middleware.NewCommand dispatch must refuse to
// execute and a middleware.HTTP handler must answer 503, because a store
// failure means the duplicate state is unknown, not absent.
//
// Set Err to nil mid-test to pass through to Next: fail one delivery, then
// prove the retry succeeds once the store heals.
type Failing struct {
	Next idempotency.Store
	Err  error
}

var _ idempotency.Store = (*Failing)(nil)

// Fail wraps next so every store call returns err. Pair the wrapper with
// [RequireFailClosed] to prove a pipeline never executes on unknown state.
func Fail(next idempotency.Store, err error) *Failing {
	return &Failing{Next: next, Err: err}
}

// Seen returns Err when set; otherwise it delegates to Next.
func (f *Failing) Seen(ctx context.Context, key string) (bool, error) {
	if f.Err != nil {
		return false, f.Err
	}

	return f.Next.Seen(ctx, key)
}

// Record returns Err when set; otherwise it delegates to Next.
func (f *Failing) Record(ctx context.Context, key string, ttl time.Duration) error {
	if f.Err != nil {
		return f.Err
	}

	return f.Next.Record(ctx, key, ttl)
}

// CheckAndRecord returns Err when set; otherwise it delegates to Next.
func (f *Failing) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if f.Err != nil {
		return f.Err
	}

	return f.Next.CheckAndRecord(ctx, key, ttl)
}
