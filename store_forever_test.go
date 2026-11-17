package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

func TestMemoryStore_CheckAndRecord_ForeverIsDuplicate(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "forever-cmd", idempotency.Forever); err != nil {
		t.Fatalf("first CheckAndRecord(Forever): want nil, got %v", err)
	}

	err := store.CheckAndRecord(ctx, "forever-cmd", idempotency.Forever)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second CheckAndRecord(Forever): want ErrDuplicate, got %v", err)
	}
}

func TestMemoryStore_Record_ForeverSurvivesAndIsNotDowngraded(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "forever-record", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever): %v", err)
	}

	// A later short Record must not shorten a live permanent claim.
	if err := store.Record(ctx, "forever-record", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short) after Forever: %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "forever-record")
	if err != nil || !seen {
		t.Fatalf("Forever claim after short Record: want seen=true, got %v, %v", seen, err)
	}
}

func TestMemoryStore_Record_ForeverDoesNotUpgradeLiveClaim(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "short-then-forever", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	// A live short claim must not be upgraded to permanent.
	if err := store.Record(ctx, "short-then-forever", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever) after short: %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "short-then-forever")
	if err != nil || seen {
		t.Fatalf("short claim upgraded to Forever: want seen=false, got %v, %v", seen, err)
	}
}

func TestMemoryStore_CheckAndRecord_ForeverReclaimsExpiredClaim(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "expired-to-forever", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	if err := store.CheckAndRecord(ctx, "expired-to-forever", idempotency.Forever); err != nil {
		t.Fatalf("CheckAndRecord(Forever) after expiry: want nil, got %v", err)
	}

	seen, err := store.Seen(ctx, "expired-to-forever")
	if err != nil || !seen {
		t.Fatalf("reclaimed Forever claim: want seen=true, got %v, %v", seen, err)
	}
}

func TestMemoryStore_Sweep_PreservesForeverEntries(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(10 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "forever-sweep", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever): %v", err)
	}

	if err := store.Record(ctx, "short-sweep", 5*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	// Let the sweeper run several cycles.
	time.Sleep(60 * time.Millisecond)

	seen, err := store.Seen(ctx, "forever-sweep")
	if err != nil || !seen {
		t.Fatalf("sweep removed a Forever claim: want seen=true, got %v, %v", seen, err)
	}

	seen, err = store.Seen(ctx, "short-sweep")
	if err != nil || seen {
		t.Fatalf("sweep did not remove an expired claim: want seen=false, got %v, %v", seen, err)
	}
}
