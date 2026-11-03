package contract

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

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
			t.Fatalf("wins: want exactly 1 winner, got %d", wins)
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
