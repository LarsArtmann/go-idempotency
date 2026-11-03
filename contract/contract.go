// Package contract provides a reusable test suite that verifies any
// [idempotency.Store] implementation against the full set of behavioral
// invariants: atomicity, TTL expiry, duplicate detection, concurrency safety,
// and error semantics.
//
// # Usage
//
// To verify your own backend, create a _test.go file in your package and call
// [RunTests] with a factory that returns a fresh [idempotency.Store]:
//
//	func TestContract(t *testing.T) {
//	    t.Parallel()
//	    contract.RunTests(t, func(t *testing.T) idempotency.Store {
//	        store := mybackend.NewStore(redisClient)
//	        t.Cleanup(func() { store.Close() })
//	        return store
//	    })
//	}
//
// The factory must return an empty store (no pre-existing keys) and register
// any necessary cleanup via [testing.T.Cleanup] so that each subtest starts
// from a clean state.
//
// On slow or heavily loaded CI runners, use [RunTestsStrict] with
// Options.TimingScale > 1 to stretch the wall-clock timings instead of
// debugging expiry flakes.
//
// # Testing context cancellation
//
// [RunTests] always passes [context.Background], so it cannot assert how your
// backend treats cancellation. That is deliberate: [idempotency.MemoryStore]
// and other local stores ignore context, and the main suite must stay
// meaningful for them. Backends that DO honor cancellation (every backend
// doing network round-trips should) should additionally run
// [RunTestsContextAware], which pins the two invariants callers rely on:
//
//   - A canceled call returns the context error, not nil and not
//     [idempotency.ErrDuplicate].
//
//   - A canceled call does NOT consume the claim: the key remains unrecorded,
//     so the retry that arrives after the cancellation can still be processed.
//     Otherwise a timed-out request would poison its key until TTL expiry.
//
// # Extending the suite
//
// Every invariant added here must ship with a matching broken-Store scenario
// in contract_negative_test.go. The negative suite is what proves an
// invariant actually detects its violation class; an invariant without a
// negative scenario can quietly rot into a check that passes everything.
package contract

import (
	"context"
	"testing"

	"github.com/larsartmann/go-idempotency"
)

// StoreFactory returns a fresh, empty [idempotency.Store] for a single test.
// The factory should register cleanup (e.g., closing connections, clearing
// data) via [testing.T.Cleanup] so that each subtest starts from a clean state.
type StoreFactory func(t *testing.T) idempotency.Store

// Options tunes [RunTestsStrict]. The zero value is a fast, default-timed run.
type Options struct {
	// TimingScale multiplies every wall-clock timing in the suite. 1 (or 0)
	// keeps the fast defaults; 2–5 suits slow CI runners or busy machines.
	// There is no upper bound, but scale also stretches the suite runtime.
	TimingScale float64
}

// RunTests runs the full contract test suite against a [idempotency.Store]
// produced by factory. Each subtest receives a fresh store instance.
//
// All tests use [context.Background] and short TTLs where timing matters.
// Backends that honor context cancellation should additionally run
// [RunTestsContextAware]; the main suite deliberately does not assert
// context semantics because [idempotency.MemoryStore] intentionally ignores
// context.
func RunTests(t *testing.T, factory StoreFactory) {
	t.Helper()
	RunTestsStrict(t, factory, Options{TimingScale: 0})
}

// RunTestsStrict behaves exactly like [RunTests] but accepts options.
//
// Use it instead of RunTests when the machine running the tests is slow or
// heavily loaded (some CI runners, containers with throttled CPU): the
// TTL-based subtests sleep on wall-clock timings, and a TimingScale above 1
// stretches every sleep proportionally so expiry still lands inside the
// window instead of flaking.
func RunTestsStrict(t *testing.T, factory StoreFactory, opts Options) {
	t.Helper()

	tm := scaledTimings(opts.TimingScale)
	ctx := context.Background()

	t.Run("Seen", func(t *testing.T) { runSeenTests(t, ctx, factory, tm) })
	t.Run("Record", func(t *testing.T) { runRecordTests(t, ctx, factory, tm) })
	t.Run("CheckAndRecord", func(t *testing.T) { runCheckAndRecordTests(t, ctx, factory, tm) })
	t.Run("Concurrency", func(t *testing.T) { runConcurrencyTests(t, ctx, factory, tm) })
	t.Run("Cross-cutting", func(t *testing.T) { runCrossCuttingTests(t, ctx, factory, tm) })
}
