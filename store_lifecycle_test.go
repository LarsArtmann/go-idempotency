package idempotency_test

// Sweep and Close lifecycle tests for the deprecated MemoryStore: background
// sweep reclamation (single-shot and under load), idempotent Close, and
// post-Close operability. The sweep tests run inside testing/synctest bubbles:
// the bubble's native fake clock virtualizes both time.Sleep and the
// sweeper's real ticker, so expiry + sweep happen in zero wall time without
// polling for a background tick.

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/larsartmann/go-idempotency"
)

func TestMemoryStore_Sweep_RemovesExpiredEntries(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		store := idempotency.NewMemoryStore(15 * time.Millisecond)
		t.Cleanup(store.Close)

		ctx := context.Background()
		_ = store.Record(ctx, "cmd-5", 10*time.Millisecond)

		// Virtual time: the Sleep advances past the TTL and several sweep
		// intervals in zero wall time — the bubble's sweeper (started inside
		// the bubble) ticks on the virtual clock and reclaims the entry
		// before any read.
		time.Sleep(60 * time.Millisecond)

		seen, _ := store.Seen(ctx, "cmd-5")
		if seen {
			t.Fatal("sweep did not remove expired entry")
		}
	})
}

func TestMemoryStore_Close_Idempotent(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(time.Hour)
	store.Close()
	store.Close() // must not panic
}

// TestMemoryStore_OperationsAfterClose verifies that Seen, Record, and
// CheckAndRecord all still function after Close. Close only stops the sweep
// goroutine; the map and mutex remain usable.
func TestMemoryStore_OperationsAfterClose(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(time.Hour)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "pre-close-key", time.Minute)

	store.Close()

	// Seen still works after Close.
	seen, err := store.Seen(ctx, "pre-close-key")
	if err != nil || !seen {
		t.Fatalf("Seen after Close: want true, nil; got %v, %v", seen, err)
	}

	// Record still works after Close.
	if err := store.Record(ctx, "post-close-key", time.Minute); err != nil {
		t.Fatalf("Record after Close: %v", err)
	}

	// CheckAndRecord still works after Close.
	if err := store.CheckAndRecord(ctx, "post-close-check", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord after Close: %v", err)
	}
}

// TestMemoryStore_Sweep_ReclaimsAllKeysUnderLoad is a TTL-sweep soak test: it
// records a large batch of short-TTL keys concurrently and verifies the
// background sweeper reclaims every one of them (no volume-induced leak,
// no key survives past its expiry under load). Runs in a synctest bubble:
// virtual time advances as soon as every bubble goroutine is durably blocked,
// so the recorder goroutines finish, the sweeper ticks repeatedly, and the
// whole soak costs zero wall time.
func TestMemoryStore_Sweep_ReclaimsAllKeysUnderLoad(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()

		store := idempotency.NewMemoryStore(10 * time.Millisecond)
		t.Cleanup(store.Close)

		const n = 1000

		var wg sync.WaitGroup
		wg.Add(n)

		for i := range n {
			go func(i int) {
				defer wg.Done()

				_ = store.Record(ctx, strconv.Itoa(i), 5*time.Millisecond)
			}(i)
		}

		wg.Wait()

		// Virtual time: past several sweep intervals and every claim's TTL.
		time.Sleep(120 * time.Millisecond)

		// Every sampled key must be reclaimed (unseen) after expiry.
		for i := 0; i < n; i += 50 {
			seen, _ := store.Seen(ctx, strconv.Itoa(i))
			if seen {
				t.Fatalf("key %d still seen after sweep under load (volume leak)", i)
			}
		}
	})
}
