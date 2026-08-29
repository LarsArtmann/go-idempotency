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
