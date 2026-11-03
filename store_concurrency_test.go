package idempotency_test

// Concurrency tests for the deprecated MemoryStore: the exactly-one-winner
// race (released simultaneously via a started barrier) and sweeper-goroutine
// hygiene after Close.

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

func TestMemoryStore_CheckAndRecord_AtomicUnderConcurrency(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	const (
		goroutines = 200
		key        = "contended-cmd"
	)

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

			<-started // release all goroutines at once

			err := store.CheckAndRecord(context.Background(), key, time.Minute)

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
}

func TestMemoryStore_Close_StopsSweeperGoroutine(t *testing.T) {
	t.Parallel()

	// Warm up any lazily-initialized runtime goroutines before measuring.
	before := runtime.NumGoroutine()

	stores := make([]*idempotency.MemoryStore, 0, 10)

	for range 10 {
		store := idempotency.NewMemoryStore(time.Millisecond)
		stores = append(stores, store)
	}

	for _, store := range stores {
		store.Close()
	}

	// Sweeper goroutines exit asynchronously; poll instead of sleeping a
	// fixed, jitter-prone interval. A small tolerance absorbs unrelated
	// runtime goroutines (GC, test framework) appearing mid-test.
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf(
		"goroutine count did not return to baseline: before=%d after=%d (sweeper leak?)",
		before, runtime.NumGoroutine(),
	)
}
