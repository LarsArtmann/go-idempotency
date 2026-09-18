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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// timings holds the durations the TTL-based subtests sleep on. Defaults are
// intentionally short to keep the suite fast while leaving enough margin for
// scheduler jitter; TimingScale stretches them on slow or busy machines.
type timings struct {
	shortTTL  time.Duration // expires quickly; used to test expiry
	mediumTTL time.Duration // slightly longer than shortTTL
	waitAfter time.Duration // sleep past shortTTL/mediumTTL
	longTTL   time.Duration // does not expire during the test
}

func defaultTimings() timings {
	//nolint:mnd // the tuned suite timings; named by their struct fields
	return timings{
		shortTTL:  20 * time.Millisecond,
		mediumTTL: 30 * time.Millisecond,
		waitAfter: 50 * time.Millisecond,
		longTTL:   time.Minute,
	}
}

func scaledTimings(scale float64) timings {
	tm := defaultTimings()

	if scale <= 0 {
		return tm
	}

	stretch := func(d time.Duration) time.Duration {
		return time.Duration(float64(d) * scale)
	}

	return timings{
		shortTTL:  stretch(tm.shortTTL),
		mediumTTL: stretch(tm.mediumTTL),
		waitAfter: stretch(tm.waitAfter),
		longTTL:   stretch(tm.longTTL),
	}
}

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

// mustSeen returns the store's Seen answer for key, failing the test if Seen
// itself errors. A backend error must never be swallowed: the zero value
// (false) would otherwise masquerade as a passing assertion and certify a
// backend that violates the very contract under test.
func mustSeen(t *testing.T, store idempotency.Store, ctx context.Context, key string) bool {
	t.Helper()

	seen, err := store.Seen(ctx, key)
	if err != nil {
		t.Fatalf("Seen(%q): %v", key, err)
	}

	return seen
}

func runSeenTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("UnseenKeyReturnsFalse", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if mustSeen(t, store, ctx, "never-recorded") {
			t.Fatal("Seen on unseen key: want false, got true")
		}
	})

	t.Run("AfterRecordReturnsTrue", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "recorded-key", tm.longTTL); err != nil {
			t.Fatalf("Record: %v", err)
		}

		if !mustSeen(t, store, ctx, "recorded-key") {
			t.Fatal("Seen after Record: want true, got false")
		}
	})

	t.Run("LazilyDeletesExpired", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "lazy-key", tm.shortTTL); err != nil {
			t.Fatalf("Record: %v", err)
		}

		time.Sleep(tm.waitAfter)

		if mustSeen(t, store, ctx, "lazy-key") {
			t.Fatal("after TTL expiry: Seen should return false")
		}
	})
}

func runRecordTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("NoopOnExistingKey", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "noop-key", tm.shortTTL); err != nil {
			t.Fatalf("first Record: %v", err)
		}

		// Second Record with a longer TTL must NOT extend the expiry.
		if err := store.Record(ctx, "noop-key", tm.longTTL); err != nil {
			t.Fatalf("second Record: %v", err)
		}

		// Wait past the original short TTL but well within the long TTL.
		time.Sleep(tm.waitAfter)

		if mustSeen(t, store, ctx, "noop-key") {
			t.Fatal("Record extended the TTL: key should have expired under the original short TTL")
		}
	})

	t.Run("ReRecordsAfterExpiry", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "expiry-key", tm.shortTTL); err != nil {
			t.Fatalf("first Record: %v", err)
		}

		time.Sleep(tm.waitAfter)

		// After expiry, Record must set a fresh TTL.
		if err := store.Record(ctx, "expiry-key", tm.longTTL); err != nil {
			t.Fatalf("Record after expiry: %v", err)
		}

		if !mustSeen(t, store, ctx, "expiry-key") {
			t.Fatal("Record after expiry: key should be seen with fresh TTL")
		}
	})

	t.Run("RejectsNonPositiveTTL", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		for _, ttl := range []time.Duration{0, -time.Second} {
			err := store.Record(ctx, "bad-ttl-record", ttl)

			if !errors.Is(err, idempotency.ErrInvalidTTL) {
				t.Fatalf("Record(%v): want ErrInvalidTTL, got %v", ttl, err)
			}
		}

		// No key must have been recorded.
		if mustSeen(t, store, ctx, "bad-ttl-record") {
			t.Fatal("key must not be recorded after a rejected TTL")
		}
	})
}

func runCheckAndRecordTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("FirstCallSucceeds", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		err := store.CheckAndRecord(ctx, "fresh-key", tm.longTTL)
		if err != nil {
			t.Fatalf("first CheckAndRecord: want nil, got %v", err)
		}
	})

	t.Run("DuplicateReturnsErrDuplicate", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.CheckAndRecord(ctx, "dup-key", tm.longTTL); err != nil {
			t.Fatalf("first CheckAndRecord: %v", err)
		}

		err := store.CheckAndRecord(ctx, "dup-key", tm.longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("second CheckAndRecord: want ErrDuplicate, got %v", err)
		}
	})

	t.Run("AllowsAfterExpiry", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.CheckAndRecord(ctx, "exp-key", tm.shortTTL); err != nil {
			t.Fatalf("first CheckAndRecord: %v", err)
		}

		// Within TTL: must be a duplicate.
		err := store.CheckAndRecord(ctx, "exp-key", tm.longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("within TTL: want ErrDuplicate, got %v", err)
		}

		time.Sleep(tm.waitAfter)

		// After expiry: a fresh recording must succeed.
		if err := store.CheckAndRecord(ctx, "exp-key", tm.longTTL); err != nil {
			t.Fatalf("after expiry: want nil, got %v", err)
		}
	})

	t.Run("RejectsNonPositiveTTL", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		for _, ttl := range []time.Duration{0, -time.Second} {
			err := store.CheckAndRecord(ctx, "bad-ttl-car", ttl)

			if !errors.Is(err, idempotency.ErrInvalidTTL) {
				t.Fatalf("CheckAndRecord(%v): want ErrInvalidTTL, got %v", ttl, err)
			}
		}

		// No key must have been recorded.
		if mustSeen(t, store, ctx, "bad-ttl-car") {
			t.Fatal("key must not be recorded after a rejected TTL")
		}
	})
}

func runConcurrencyTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("AtomicUnderConcurrency", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		const goroutines = 200

		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			wins    int
			dups    int
			started = make(chan struct{})
		)

		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()

				<-started

				err := store.CheckAndRecord(ctx, "concurrent-key", tm.longTTL)

				mu.Lock()

				switch {
				case err == nil:
					wins++
				case errors.Is(err, idempotency.ErrDuplicate):
					dups++
				default:
					t.Errorf("unexpected error: %v", err)
				}

				mu.Unlock()
			}()
		}

		close(started)
		wg.Wait()

		if wins != 1 {
			t.Fatalf("wins: want exactly 1, got %d", wins)
		}

		if dups != goroutines-1 {
			t.Fatalf("dups: want %d, got %d", goroutines-1, dups)
		}
	})
}

func runCrossCuttingTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("KeysAreIndependent", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "key-A", tm.longTTL); err != nil {
			t.Fatalf("Record A: %v", err)
		}

		if mustSeen(t, store, ctx, "key-B") {
			t.Fatal("key-B should not be seen (only key-A was recorded)")
		}

		if err := store.CheckAndRecord(ctx, "key-B", tm.longTTL); err != nil {
			t.Fatalf("CheckAndRecord B: %v", err)
		}

		if !mustSeen(t, store, ctx, "key-A") {
			t.Fatal("key-A should still be seen after operating on key-B")
		}
	})

	t.Run("EmptyKey", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "", tm.longTTL); err != nil {
			t.Fatalf("Record empty key: %v", err)
		}

		if !mustSeen(t, store, ctx, "") {
			t.Fatal("empty key should be seen after Record")
		}

		err := store.CheckAndRecord(ctx, "", tm.longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("CheckAndRecord on existing empty key: want ErrDuplicate, got %v", err)
		}
	})
}
