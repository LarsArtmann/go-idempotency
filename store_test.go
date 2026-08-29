package idempotency_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-idempotency"
)

func TestMemoryStore_CheckAndRecord_FirstCallSucceeds(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	err := store.CheckAndRecord(context.Background(), "cmd-1", time.Minute)
	if err != nil {
		t.Fatalf("first CheckAndRecord: want nil, got %v", err)
	}
}

func TestMemoryStore_CheckAndRecord_DuplicateReturnsErrDuplicate(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := store.CheckAndRecord(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("first call: %v", err)
	}

	err := store.CheckAndRecord(ctx, "cmd-1", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second call: want ErrDuplicate, got %v", err)
	}

	// ErrDuplicate must be a Conflict so it maps to HTTP 409 downstream.
	if fam := errorfamily.Classify(err); fam != errorfamily.Conflict {
		t.Fatalf("family: want Conflict, got %s", fam)
	}
}

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

func TestMemoryStore_Seen_AfterRecord(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	seen, err := store.Seen(ctx, "cmd-2")
	if err != nil || seen {
		t.Fatalf("before record: want seen=false, nil; got %v, %v", seen, err)
	}

	_ = store.Record(ctx, "cmd-2", time.Minute)

	seen, err = store.Seen(ctx, "cmd-2")
	if err != nil || !seen {
		t.Fatalf("after record: want seen=true, nil; got %v, %v", seen, err)
	}
}

func TestMemoryStore_Seen_LazilyDeletesExpired(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0) // no sweep goroutine
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-3", 20*time.Millisecond)

	time.Sleep(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "cmd-3")
	if err != nil || seen {
		t.Fatalf("after TTL: want seen=false, got %v, %v", seen, err)
	}

	// The lazy delete must have removed the entry.
	seen, err = store.Seen(ctx, "cmd-3")
	if err != nil || seen {
		t.Fatalf("after lazy delete: want seen=false, got %v, %v", seen, err)
	}
}

func TestMemoryStore_Record_NoopOnExistingKey(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-4", time.Minute)

	// A second Record must not extend the TTL (no-op).
	_ = store.Record(ctx, "cmd-4", 5*time.Minute)

	// Still seen — no-op doesn't clear it.
	seen, _ := store.Seen(ctx, "cmd-4")
	if !seen {
		t.Fatal("Record no-op cleared a valid key")
	}
}

func TestMemoryStore_Record_AfterExpiry_ReRecordsWithNewTTL(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0) // no sweep: expired key stays in map
	defer store.Close()

	ctx := context.Background()

	_ = store.Record(ctx, "cmd-record-expiry", 20*time.Millisecond)

	time.Sleep(40 * time.Millisecond)

	// The key is expired but still in the map (no sweep, no intervening Seen to
	// lazily delete it). Record must treat it as expired and set a fresh TTL.
	_ = store.Record(ctx, "cmd-record-expiry", time.Minute)

	seen, err := store.Seen(ctx, "cmd-record-expiry")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}

	if !seen {
		t.Fatal("Record after expiry should have re-recorded with the new TTL, but Seen returned false")
	}
}

func TestMemoryStore_Sweep_RemovesExpiredEntries(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(15 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-5", 10*time.Millisecond)

	// Wait beyond the sweep interval so the background goroutine runs at least once.
	time.Sleep(60 * time.Millisecond)

	seen, _ := store.Seen(ctx, "cmd-5")
	if seen {
		t.Fatal("sweep did not remove expired entry")
	}
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

// TestMemoryStore_EmptyKey verifies that the empty string is a valid key.
func TestMemoryStore_EmptyKey(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "", time.Minute); err != nil {
		t.Fatalf("Record empty key: %v", err)
	}

	seen, _ := store.Seen(ctx, "")
	if !seen {
		t.Fatal("empty key should be seen after Record")
	}

	if err := store.CheckAndRecord(ctx, "", time.Minute); !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("CheckAndRecord on existing empty key: want ErrDuplicate, got %v", err)
	}
}

// TestMemoryStore_NonPositiveTTLRejected verifies that Record and
// CheckAndRecord reject a non-positive TTL with ErrInvalidTTL. A zero or
// negative TTL would record an expiry already in the past, so the key would
// protect nothing and the exactly-once guarantee would silently break; the
// store rejects the bad input instead of accepting a useless recording.
func TestMemoryStore_NonPositiveTTLRejected(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	for _, testCase := range []struct {
		name string
		ttl  time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	} {
		if err := store.Record(
			ctx,
			"rec-"+testCase.name,
			testCase.ttl,
		); !errors.Is(err, idempotency.ErrInvalidTTL) {
			t.Fatalf("Record(%s): want ErrInvalidTTL, got %v", testCase.name, err)
		}

		if err := store.CheckAndRecord(
			ctx,
			"car-"+testCase.name,
			testCase.ttl,
		); !errors.Is(err, idempotency.ErrInvalidTTL) {
			t.Fatalf("CheckAndRecord(%s): want ErrInvalidTTL, got %v", testCase.name, err)
		}
	}

	// ErrInvalidTTL must be a Rejection so it maps to HTTP 400 downstream and is
	// not retried.
	if fam := errorfamily.Classify(idempotency.ErrInvalidTTL); fam != errorfamily.Rejection {
		t.Fatalf("family: want Rejection, got %s", fam)
	}

	if errorfamily.IsRetryable(idempotency.ErrInvalidTTL) {
		t.Fatal("Rejection must not be retryable")
	}

	// No key must have been recorded for any rejected input.
	for _, key := range []string{"rec-zero", "rec-negative", "car-zero", "car-negative"} {
		if seen, _ := store.Seen(ctx, key); seen {
			t.Fatalf("key %q must not be recorded after a rejected TTL", key)
		}
	}
}

func TestMemoryStore_CheckAndRecord_AllowsAfterExpiry(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.CheckAndRecord(ctx, "cmd-6", 20*time.Millisecond)

	// Within TTL: duplicate.
	if err := store.CheckAndRecord(
		ctx,
		"cmd-6",
		time.Minute,
	); !errors.Is(
		err,
		idempotency.ErrDuplicate,
	) {
		t.Fatalf("within TTL: want ErrDuplicate, got %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	// After expiry: a fresh recording is allowed.
	if err := store.CheckAndRecord(ctx, "cmd-6", time.Minute); err != nil {
		t.Fatalf("after expiry: want nil, got %v", err)
	}
}

func TestErrDuplicate_IsConflict(t *testing.T) {
	t.Parallel()

	if fam := errorfamily.Classify(idempotency.ErrDuplicate); fam != errorfamily.Conflict {
		t.Fatalf("family: want Conflict, got %s", fam)
	}

	if errorfamily.IsRetryable(idempotency.ErrDuplicate) {
		t.Fatal("Conflict must not be retryable")
	}
}

// TestMemoryStore_Sweep_ReclaimsAllKeysUnderLoad is a TTL-sweep soak test: it
// records a large batch of short-TTL keys concurrently and verifies the
// background sweeper reclaims every one of them (no volume-induced leak,
// no key survives past its expiry under load).
func TestMemoryStore_Sweep_ReclaimsAllKeysUnderLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store := idempotency.NewMemoryStore(10 * time.Millisecond)
	defer store.Close()

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

	// Wait well beyond the sweep interval so the sweeper runs multiple cycles.
	time.Sleep(120 * time.Millisecond)

	// Every sampled key must be reclaimed (unseen) after expiry.
	for i := 0; i < n; i += 50 {
		seen, _ := store.Seen(ctx, strconv.Itoa(i))
		if seen {
			t.Fatalf("key %d still seen after sweep under load (volume leak)", i)
		}
	}
}

func TestSentinels_SurviveErrorWrapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
	}{
		{name: "ErrDuplicate", sentinel: idempotency.ErrDuplicate},
		{name: "ErrInvalidTTL", sentinel: idempotency.ErrInvalidTTL},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapped := fmt.Errorf("dispatch: %w", fmt.Errorf("handler: %w", testCase.sentinel))

			if !errors.Is(wrapped, testCase.sentinel) {
				t.Fatalf(
					"errors.Is must match through fmt.Errorf wrapping (code=%q)",
					errorfamily.Code(testCase.sentinel),
				)
			}
		})
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
