package idempotency_test

import (
	"context"
	"errors"
	"math"
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

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "forever-record", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever): %v", err)
	}

	// A later short Record must not shorten a live permanent claim.
	if err := store.Record(ctx, "forever-record", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short) after Forever: %v", err)
	}

	// Long enough that the claim would have expired had the short Record
	// downgraded it.
	clock.Advance(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "forever-record")
	if err != nil || !seen {
		t.Fatalf("Forever claim after short Record: want seen=true, got %v, %v", seen, err)
	}
}

func TestMemoryStore_Record_ForeverDoesNotUpgradeLiveClaim(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "short-then-forever", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	// A live short claim must not be upgraded to permanent.
	if err := store.Record(ctx, "short-then-forever", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever) after short: %v", err)
	}

	// Past the original short expiry: Seen=false proves the claim really
	// expired, i.e. Forever never took effect.
	clock.Advance(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "short-then-forever")
	if err != nil || seen {
		t.Fatalf("short claim upgraded to Forever: want seen=false, got %v, %v", seen, err)
	}
}

func TestMemoryStore_CheckAndRecord_ForeverReclaimsExpiredClaim(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "expired-to-forever", 20*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	clock.Advance(40 * time.Millisecond)

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

	clock := newFakeClock(time.Unix(0, 0))
	store := idempotency.NewMemoryStore(10*time.Millisecond, idempotency.WithClock(clock.Now))
	defer store.Close()

	ctx := context.Background()

	if err := store.Record(ctx, "forever-sweep", idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever): %v", err)
	}

	if err := store.Record(ctx, "short-sweep", 5*time.Millisecond); err != nil {
		t.Fatalf("Record(short): %v", err)
	}

	// Expire the short claim on the pinned clock, then wait for one
	// wall-clock sweep tick past the advance: the tick must reclaim the
	// short claim but never touch the Forever entry.
	clock.Advance(10 * time.Millisecond)

	baseline := clock.Calls()
	if !clock.awaitCallAfter(baseline, 2*time.Second) {
		t.Fatal("sweeper did not tick after the clock was advanced")
	}

	seen, err := store.Seen(ctx, "forever-sweep")
	if err != nil || !seen {
		t.Fatalf("sweep removed a Forever claim: want seen=true, got %v, %v", seen, err)
	}

	seen, err = store.Seen(ctx, "short-sweep")
	if err != nil || seen {
		t.Fatalf("sweep did not remove an expired claim: want seen=false, got %v, %v", seen, err)
	}
}

func TestForever_ValueIsRepresentableInProcessButNotAsUnixNano(t *testing.T) {
	t.Parallel()

	if idempotency.Forever != math.MaxInt64 {
		t.Fatalf("Forever: want math.MaxInt64, got %d", idempotency.Forever)
	}

	// A time.Time can carry now+Forever (about year 2318), so the sentinel is a
	// legal in-process duration and the ttl <= 0 guard accepts it.
	expiry := time.Now().Add(idempotency.Forever)
	if expiry.IsZero() || !expiry.After(time.Now()) {
		t.Fatalf("now.Add(Forever): want a future instant, got %v", expiry)
	}

	if expiry.Year() < 2262 {
		t.Fatalf("now.Add(Forever): want an instant past 2262, got %v", expiry)
	}

	// UnixNano is only defined for instants between 1678 and 2262, so a backend
	// persisting epoch nanoseconds would overflow. That is why adapters store a
	// permanent representation (no Redis expiry, NULL SQL column) and never
	// compute now.Add(Forever).
}
