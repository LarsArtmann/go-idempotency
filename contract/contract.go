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
// # Testing context cancellation
//
// [RunTests] always passes [context.Background], so it cannot assert how your
// backend treats cancellation. Backends that honor context cancellation (and
// in particular any backend doing network round-trips) should add a test like
// the one below. It pins the two invariants callers rely on:
//
//   - A canceled call returns the context error, not nil and not
//     [idempotency.ErrDuplicate].
//
//   - A canceled call does NOT consume the claim: the key remains unrecorded,
//     so the retry that arrives after the cancellation can still be processed.
//     Otherwise a timed-out request would poison its key until TTL expiry.
//
//     func TestContextCancellation(t *testing.T) {
//     t.Parallel()
//
//     store := mybackend.NewStore(redisClient)
//     t.Cleanup(func() { store.Close() })
//
//     ctx, cancel := context.WithCancel(context.Background())
//     cancel()
//
//     err := store.CheckAndRecord(ctx, "canceled-key", time.Minute)
//     if !errors.Is(err, context.Canceled) {
//     t.Fatalf("canceled CheckAndRecord: want context.Canceled, got %v", err)
//     }
//
//     // The canceled call must not have recorded the key.
//     if err := store.CheckAndRecord(context.Background(), "canceled-key", time.Minute); err != nil {
//     t.Fatalf("canceled call consumed the claim: %v", err)
//     }
//     }
//
// Apply the same pattern to Seen and Record.
package contract

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Timing constants for TTL-based contract tests. These are intentionally short
// to keep the test suite fast while leaving enough margin for scheduler jitter.
const (
	shortTTL  = 20 * time.Millisecond // expires quickly; used to test expiry
	mediumTTL = 30 * time.Millisecond // slightly longer than shortTTL
	waitAfter = 50 * time.Millisecond // sleep past shortTTL/mediumTTL
	longTTL   = time.Minute           // does not expire during the test
)

// StoreFactory returns a fresh, empty [idempotency.Store] for a single test.
// The factory should register cleanup (e.g., closing connections, clearing
// data) via [testing.T.Cleanup] so that each subtest starts from a clean state.
type StoreFactory func(t *testing.T) idempotency.Store

// RunTests runs the full contract test suite against a [idempotency.Store]
// produced by factory. Each subtest receives a fresh store instance.
//
// All tests use [context.Background] and short TTLs where timing matters.
// Backends that honor context cancellation should also be tested separately
// for timeout behavior — the contract suite does not assert context semantics
// because [idempotency.MemoryStore] intentionally ignores context.
func RunTests(t *testing.T, factory StoreFactory) {
	t.Helper()

	ctx := context.Background()

	t.Run("Seen", func(t *testing.T) { runSeenTests(t, ctx, factory) })
	t.Run("Record", func(t *testing.T) { runRecordTests(t, ctx, factory) })
	t.Run("CheckAndRecord", func(t *testing.T) { runCheckAndRecordTests(t, ctx, factory) })
	t.Run("Concurrency", func(t *testing.T) { runConcurrencyTests(t, ctx, factory) })
	t.Run("Cross-cutting", func(t *testing.T) { runCrossCuttingTests(t, ctx, factory) })
}

func runSeenTests(t *testing.T, ctx context.Context, factory StoreFactory) {
	t.Helper()

	t.Run("UnseenKeyReturnsFalse", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		seen, err := store.Seen(ctx, "never-recorded")
		if err != nil {
			t.Fatalf("Seen on unseen key: unexpected error: %v", err)
		}

		if seen {
			t.Fatal("Seen on unseen key: want false, got true")
		}
	})

	t.Run("AfterRecordReturnsTrue", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "recorded-key", longTTL); err != nil {
			t.Fatalf("Record: %v", err)
		}

		seen, err := store.Seen(ctx, "recorded-key")
		if err != nil {
			t.Fatalf("Seen after Record: %v", err)
		}

		if !seen {
			t.Fatal("Seen after Record: want true, got false")
		}
	})

	t.Run("LazilyDeletesExpired", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		_ = store.Record(ctx, "lazy-key", shortTTL)

		time.Sleep(waitAfter)

		seen, _ := store.Seen(ctx, "lazy-key")

		if seen {
			t.Fatal("after TTL expiry: Seen should return false")
		}
	})
}

func runRecordTests(t *testing.T, ctx context.Context, factory StoreFactory) {
	t.Helper()

	t.Run("NoopOnExistingKey", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "noop-key", shortTTL); err != nil {
			t.Fatalf("first Record: %v", err)
		}

		// Second Record with a longer TTL must NOT extend the expiry.
		if err := store.Record(ctx, "noop-key", longTTL); err != nil {
			t.Fatalf("second Record: %v", err)
		}

		// Wait past the original short TTL but well within the long TTL.
		time.Sleep(waitAfter)

		seen, _ := store.Seen(ctx, "noop-key")

		if seen {
			t.Fatal("Record extended the TTL: key should have expired under the original short TTL")
		}
	})

	t.Run("ReRecordsAfterExpiry", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "expiry-key", shortTTL); err != nil {
			t.Fatalf("first Record: %v", err)
		}

		time.Sleep(waitAfter)

		// After expiry, Record must set a fresh TTL.
		if err := store.Record(ctx, "expiry-key", longTTL); err != nil {
			t.Fatalf("Record after expiry: %v", err)
		}

		seen, _ := store.Seen(ctx, "expiry-key")

		if !seen {
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
		seen, _ := store.Seen(ctx, "bad-ttl-record")

		if seen {
			t.Fatal("key must not be recorded after a rejected TTL")
		}
	})
}

func runCheckAndRecordTests(t *testing.T, ctx context.Context, factory StoreFactory) {
	t.Helper()

	t.Run("FirstCallSucceeds", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		err := store.CheckAndRecord(ctx, "fresh-key", longTTL)
		if err != nil {
			t.Fatalf("first CheckAndRecord: want nil, got %v", err)
		}
	})

	t.Run("DuplicateReturnsErrDuplicate", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.CheckAndRecord(ctx, "dup-key", longTTL); err != nil {
			t.Fatalf("first CheckAndRecord: %v", err)
		}

		err := store.CheckAndRecord(ctx, "dup-key", longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("second CheckAndRecord: want ErrDuplicate, got %v", err)
		}
	})

	t.Run("AllowsAfterExpiry", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.CheckAndRecord(ctx, "exp-key", shortTTL); err != nil {
			t.Fatalf("first CheckAndRecord: %v", err)
		}

		// Within TTL: must be a duplicate.
		err := store.CheckAndRecord(ctx, "exp-key", longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("within TTL: want ErrDuplicate, got %v", err)
		}

		time.Sleep(waitAfter)

		// After expiry: a fresh recording must succeed.
		if err := store.CheckAndRecord(ctx, "exp-key", longTTL); err != nil {
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
		seen, _ := store.Seen(ctx, "bad-ttl-car")

		if seen {
			t.Fatal("key must not be recorded after a rejected TTL")
		}
	})
}

func runConcurrencyTests(t *testing.T, ctx context.Context, factory StoreFactory) {
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

				err := store.CheckAndRecord(ctx, "concurrent-key", longTTL)

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

func runCrossCuttingTests(t *testing.T, ctx context.Context, factory StoreFactory) {
	t.Helper()

	t.Run("KeysAreIndependent", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "key-A", longTTL); err != nil {
			t.Fatalf("Record A: %v", err)
		}

		seenB, _ := store.Seen(ctx, "key-B")

		if seenB {
			t.Fatal("key-B should not be seen (only key-A was recorded)")
		}

		if err := store.CheckAndRecord(ctx, "key-B", longTTL); err != nil {
			t.Fatalf("CheckAndRecord B: %v", err)
		}

		seenA, _ := store.Seen(ctx, "key-A")

		if !seenA {
			t.Fatal("key-A should still be seen after operating on key-B")
		}
	})

	t.Run("EmptyKey", func(t *testing.T) {
		t.Parallel()
		store := factory(t)

		if err := store.Record(ctx, "", longTTL); err != nil {
			t.Fatalf("Record empty key: %v", err)
		}

		seen, _ := store.Seen(ctx, "")

		if !seen {
			t.Fatal("empty key should be seen after Record")
		}

		err := store.CheckAndRecord(ctx, "", longTTL)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("CheckAndRecord on existing empty key: want ErrDuplicate, got %v", err)
		}
	})
}
