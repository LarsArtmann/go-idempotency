package idempotency_test

// Option-semantics tests for WithClock: nil handling, application order, and
// the wall-clock fallback. The expiry-path tests elsewhere pin the clock for
// determinism; these pin the OPTION itself.

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// TestMemoryStore_WithClock_NilIsIgnored: a nil clock must neither replace an
// earlier option's clock nor be stored — a stored nil func would panic on the
// first expiry read.
func TestMemoryStore_WithClock_NilIsIgnored(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0,
		idempotency.WithClock(clock.Now),
		idempotency.WithClock(nil),
	)
	defer store.Close()

	ctx := context.Background()
	if err := store.Record(ctx, "nil-clock", 20*time.Millisecond); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// If nil had replaced the fake with the wall clock, only microseconds
	// would have passed and the entry would still read seen. If nil had been
	// stored, Record above would have panicked. Only the ignored-nil path
	// reaches here with a working fake clock.
	clock.Advance(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "nil-clock")
	if err != nil || seen {
		t.Fatalf("after advance: want seen=false, nil; got %v, %v", seen, err)
	}
}

// TestMemoryStore_WithClock_LastOptionWins: options apply in order, so the
// last clock installed governs every expiry decision.
func TestMemoryStore_WithClock_LastOptionWins(t *testing.T) {
	t.Parallel()

	first := newFakeClock(time.Unix(0, 0))
	second := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0,
		idempotency.WithClock(first.Now),
		idempotency.WithClock(second.Now),
	)
	defer store.Close()

	ctx := context.Background()
	if err := store.Record(ctx, "last-wins", 20*time.Millisecond); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Advance only the second clock: expiry must follow it. If the first
	// clock had won, the entry would still be live here.
	second.Advance(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "last-wins")
	if err != nil || seen {
		t.Fatalf("after advancing second clock: want seen=false, nil; got %v, %v", seen, err)
	}
}

// TestMemoryStore_WithClock_NilOnlyFallsBackToWallClock: WithClock(nil) as the
// only option behaves exactly like the default store — no panic, and expiry
// runs on the real wall clock (bounded poll, never a bare sleep).
func TestMemoryStore_WithClock_NilOnlyFallsBackToWallClock(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0, idempotency.WithClock(nil))
	defer store.Close()

	ctx := context.Background()
	if err := store.Record(ctx, "wall-clock", 20*time.Millisecond); err != nil {
		t.Fatalf("Record: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		seen, err := store.Seen(ctx, "wall-clock")
		if err != nil {
			t.Fatalf("Seen: %v", err)
		}

		if !seen {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("claim never expired on the wall clock: WithClock(nil) did not fall back to time.Now")
}
