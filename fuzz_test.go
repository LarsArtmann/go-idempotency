package idempotency_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// FuzzCheckAndRecord verifies that CheckAndRecord never panics on arbitrary
// keys and TTLs, and that the exactly-once invariant holds: for any two
// sequential calls with the same key, the first succeeds and the second
// returns ErrDuplicate (or both succeed if the TTL expired between them).
func FuzzCheckAndRecord(f *testing.F) {
	f.Add("fuzz-key", int64(60))
	f.Add("", int64(1))
	f.Add("key-with-unicode", int64(0))
	f.Add("🔑🚀-emoji-ключ-中文", int64(30)) //nolint:gosmopolitan // unicode key is the point of the seed
	f.Add(strings.Repeat("K", 4096), int64(30))
	f.Add("max-ttl", int64(math.MaxInt64))
	f.Add("negative-ttl", int64(-1))

	f.Fuzz(func(t *testing.T, key string, ttlSeconds int64) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		ctx := context.Background()
		ttl := time.Duration(ttlSeconds) * time.Second

		err := store.CheckAndRecord(ctx, key, ttl)

		if ttl <= 0 {
			if !errors.Is(err, idempotency.ErrInvalidTTL) {
				t.Fatalf("non-positive TTL: want ErrInvalidTTL, got %v", err)
			}

			return
		}

		if err != nil {
			t.Fatalf("first CheckAndRecord: %v", err)
		}

		// Second call with the same key must be a duplicate.
		err = store.CheckAndRecord(ctx, key, ttl)

		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("second CheckAndRecord: want ErrDuplicate, got %v", err)
		}
	})
}

// FuzzRecord verifies that Record never panics on arbitrary inputs and that
// it rejects non-positive TTLs correctly.
func FuzzRecord(f *testing.F) {
	f.Add("fuzz-key", int64(30))
	f.Add("", int64(-5))
	f.Add("another-key", int64(0))
	f.Add("🔑🚀-emoji-ключ-中文", int64(-1)) //nolint:gosmopolitan // unicode key is the point of the seed
	f.Add(strings.Repeat("R", 4096), int64(15))
	f.Add("max-ttl", int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, key string, ttlSeconds int64) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		ctx := context.Background()
		ttl := time.Duration(ttlSeconds) * time.Second

		err := store.Record(ctx, key, ttl)

		if ttl <= 0 {
			if !errors.Is(err, idempotency.ErrInvalidTTL) {
				t.Fatalf("non-positive TTL: want ErrInvalidTTL, got %v", err)
			}

			return
		}

		if err != nil {
			t.Fatalf("Record: %v", err)
		}

		// Idempotent: second Record must not error.
		err = store.Record(ctx, key, ttl)
		if err != nil {
			t.Fatalf("second Record: %v", err)
		}
	})
}

// TestMemoryStore_CloseDuringConcurrentOps verifies that calling Close while
// goroutines are actively using the store does not panic. Close only stops the
// sweep goroutine; the map and mutex remain usable, so concurrent operations
// during Close must be safe.
func TestMemoryStore_CloseDuringConcurrentOps(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(10 * time.Millisecond)

	ctx := context.Background()

	var ops atomic.Int64

	stop := make(chan struct{})

	// Start 50 goroutines hammering the store.
	for range 50 {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}

				_ = store.CheckAndRecord(ctx, "race-key", time.Minute)
				_ = store.Record(ctx, "race-key", time.Minute)

				_, _ = store.Seen(ctx, "race-key")

				ops.Add(1)
			}
		}()
	}

	// Let goroutines run, then Close mid-flight.
	time.Sleep(20 * time.Millisecond)

	store.Close()
	close(stop)

	// Operations after Close must still work (no panic).
	if err := store.CheckAndRecord(ctx, "post-close", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord after Close during concurrent ops: %v", err)
	}

	if ops.Load() == 0 {
		t.Fatal("no operations completed before Close")
	}
}

// FuzzConcurrentMixed drives a randomized goroutine count doing mixed
// CheckAndRecord/Record/Seen operations against one store, hunting for panics
// and data races under arbitrary interleavings. The exactly-once and
// TTL-validation invariants are covered by the sequential fuzz targets; this
// one is about concurrent-state safety, so non-positive TTLs exit early.
func FuzzConcurrentMixed(f *testing.F) {
	f.Add("seed-key", int64(5), uint8(4))
	f.Add("", int64(1), uint8(19))
	f.Add("🔑-mixed", int64(60), uint8(2))

	f.Fuzz(func(t *testing.T, key string, ttlSeconds int64, goroutines uint8) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		ttl := time.Duration(ttlSeconds) * time.Second
		if ttl <= 0 {
			return
		}

		workers := int(goroutines)%19 + 2 // 2..20
		ctx := context.Background()

		var wg sync.WaitGroup

		wg.Add(workers)

		for i := range workers {
			go func(i int) {
				defer wg.Done()

				switch i % 3 {
				case 0:
					_ = store.CheckAndRecord(ctx, key, ttl)
				case 1:
					_ = store.Record(ctx, key, ttl)
				default:
					_, _ = store.Seen(ctx, key)
				}
			}(i)
		}

		wg.Wait()
	})
}
