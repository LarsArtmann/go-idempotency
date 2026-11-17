package contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// runForeverTests pins the permanent-claim semantics (ADR-006). Forever is a
// TTL sentinel: a claim recorded with it never expires, and the no-extension
// rule applies in both directions. A live short-TTL claim must not be
// upgraded to permanent, and a live permanent claim must not be downgraded to
// a short TTL.
func runForeverTests(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	t.Run("SeenSurvives", func(t *testing.T) {
		foreverSeenSurvives(t, ctx, factory, tm)
	})
	t.Run("RecordNeverExpires", func(t *testing.T) {
		foreverRecordNeverExpires(t, ctx, factory, tm)
	})
	t.Run("RecordNoopOnExistingKey", func(t *testing.T) {
		foreverNoopOnExistingKey(t, ctx, factory, tm)
	})
	t.Run("RecordNotDowngraded", func(t *testing.T) {
		foreverNotDowngraded(t, ctx, factory, tm)
	})
	t.Run("CheckAndRecord", func(t *testing.T) {
		foreverCheckAndRecord(t, ctx, factory, tm)
	})
}

// claimedForever returns a fresh store that already holds a live permanent
// claim on key, after sleeping tm.waitAfter — long enough for a short-TTL
// claim to expire, so a key still seen is held by Forever alone. Fatals if
// the claim cannot be recorded.
func claimedForever( //nolint:ireturn // Store handouts in this suite are the interface
	t *testing.T,
	ctx context.Context,
	factory StoreFactory,
	tm timings,
	key string,
) idempotency.Store {
	t.Helper()
	t.Parallel()

	store := factory(t)

	if err := store.Record(ctx, key, idempotency.Forever); err != nil {
		t.Fatalf("Record(Forever): %v", err)
	}

	time.Sleep(tm.waitAfter)

	return store
}

func foreverSeenSurvives(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	store := claimedForever(t, ctx, factory, tm, "forever-seen")

	if !mustSeen(t, store, ctx, "forever-seen") {
		t.Fatal("Forever claim expired: want seen=true, got false")
	}
}

func foreverRecordNeverExpires(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()

	store := claimedForever(t, ctx, factory, tm, "forever-check")

	err := store.CheckAndRecord(ctx, "forever-check", idempotency.Forever)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("CheckAndRecord after wait: want ErrDuplicate, got %v", err)
	}
}

func foreverNoopOnExistingKey(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()
	t.Parallel()

	store := factory(t)

	if err := store.Record(ctx, "forever-noop", tm.shortTTL); err != nil {
		t.Fatalf("first Record: %v", err)
	}

	// A live short-TTL claim must not be upgraded to permanent.
	if err := store.Record(ctx, "forever-noop", idempotency.Forever); err != nil {
		t.Fatalf("second Record(Forever): %v", err)
	}

	time.Sleep(tm.waitAfter)

	if mustSeen(t, store, ctx, "forever-noop") {
		t.Fatal("Record upgraded a live claim to Forever: key should have expired under the original short TTL")
	}
}

func foreverNotDowngraded(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()
	t.Parallel()

	store := factory(t)

	if err := store.Record(ctx, "forever-downgrade", idempotency.Forever); err != nil {
		t.Fatalf("first Record(Forever): %v", err)
	}

	// A live permanent claim must not be shortened.
	if err := store.Record(ctx, "forever-downgrade", tm.shortTTL); err != nil {
		t.Fatalf("second Record(short): %v", err)
	}

	time.Sleep(tm.waitAfter)

	if !mustSeen(t, store, ctx, "forever-downgrade") {
		t.Fatal("Record downgraded a Forever claim: key should still be seen")
	}
}

func foreverCheckAndRecord(t *testing.T, ctx context.Context, factory StoreFactory, tm timings) {
	t.Helper()
	t.Parallel()

	store := factory(t)

	if err := store.CheckAndRecord(ctx, "forever-claim", idempotency.Forever); err != nil {
		t.Fatalf("CheckAndRecord(Forever): want nil, got %v", err)
	}

	err := store.CheckAndRecord(ctx, "forever-claim", idempotency.Forever)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second CheckAndRecord(Forever): want ErrDuplicate, got %v", err)
	}

	time.Sleep(tm.waitAfter)

	err = store.CheckAndRecord(ctx, "forever-claim", idempotency.Forever)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("CheckAndRecord after wait: want ErrDuplicate, got %v", err)
	}
}
