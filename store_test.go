package idempotency_test

import (
	"context"
	"errors"
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

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now)) // no sweep goroutine
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-3", 20*time.Millisecond)

	clock.Advance(40 * time.Millisecond)

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

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now)) // no sweep: expired key stays in map
	defer store.Close()

	ctx := context.Background()

	_ = store.Record(ctx, "cmd-record-expiry", 20*time.Millisecond)

	clock.Advance(40 * time.Millisecond)

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

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
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

	clock.Advance(40 * time.Millisecond)

	// After expiry: a fresh recording is allowed.
	if err := store.CheckAndRecord(ctx, "cmd-6", time.Minute); err != nil {
		t.Fatalf("after expiry: want nil, got %v", err)
	}
}
