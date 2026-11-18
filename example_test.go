package idempotency_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// ExampleStore shows the primary usage pattern: CheckAndRecord as a single
// atomic step that prevents duplicate processing of a retried command.
//
// It uses the deprecated MemoryStore for illustration only; for production,
// see the "Implementing a Custom Backend" section in the package docs.
func ExampleStore() {
	store := idempotency.NewMemoryStore(5 * time.Minute)
	defer store.Close()

	ctx := context.Background()

	// First submission of a command with idempotency key "order-123".
	err := store.CheckAndRecord(ctx, "order-123", 10*time.Minute)
	if err != nil {
		fmt.Println("unexpected error on first call:", err)

		return
	}

	fmt.Println("first call: processing")

	// Client lost the ack and retries the same command.
	err = store.CheckAndRecord(ctx, "order-123", 10*time.Minute)

	if errors.Is(err, idempotency.ErrDuplicate) {
		fmt.Println("retry: dropped (duplicate)")

		return
	}

	// Output:
	// first call: processing
	// retry: dropped (duplicate)
}

// ExampleMemoryStore demonstrates the full lifecycle: create, use, close.
// The deprecated MemoryStore keeps the example runnable; see the package docs
// for the production path (implement Store against your own backend).
func ExampleMemoryStore() {
	store := idempotency.NewMemoryStore(5 * time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Record a key, then check it.
	_ = store.Record(ctx, "task-abc", time.Hour)

	seen, _ := store.Seen(ctx, "task-abc")
	fmt.Println("seen:", seen)

	// An unrecorded key is not seen.
	seen, _ = store.Seen(ctx, "task-xyz")
	fmt.Println("seen:", seen)

	// Output:
	// seen: true
	// seen: false
}

// ExampleWithClock pins the store's clock so expiry becomes a deterministic
// state transition: advance the clock instead of sleeping past the TTL.
func ExampleWithClock() {
	now := time.Unix(0, 0)

	store := idempotency.NewMemoryStore(0,
		idempotency.WithClock(func() time.Time { return now }),
	)
	defer store.Close()

	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "pinned", time.Hour); err != nil {
		fmt.Println("unexpected error on first call:", err)

		return
	}

	fmt.Println("first call: processing")

	// Advance the clock past the TTL — no sleep, no flake.
	now = now.Add(2 * time.Hour)

	if err := store.CheckAndRecord(ctx, "pinned", time.Hour); err != nil {
		fmt.Println("unexpected error after expiry:", err)

		return
	}

	fmt.Println("after expiry: reclaimed")

	// Output:
	// first call: processing
	// after expiry: reclaimed
}

// ExampleForever shows a permanent claim: the duplicate window is the lifetime
// of the store, which is the correct shape for event/consumer dedupe. Request
// idempotency keys should stay TTL-windowed (ADR-006).
func ExampleForever() {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	eventKey := "event:evt-42"

	_ = store.CheckAndRecord(ctx, eventKey, idempotency.Forever)

	err := store.CheckAndRecord(ctx, eventKey, idempotency.Forever)
	fmt.Println("duplicate:", errors.Is(err, idempotency.ErrDuplicate))

	// Output:
	// duplicate: true
}
