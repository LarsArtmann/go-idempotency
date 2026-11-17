package idemtest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
	"github.com/larsartmann/go-idempotency/middleware"
)

func TestRequireExactlyOnce(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, time.Minute, exec.Fn())

	idemtest.RequireExactlyOnce(t, place, "order-42:place", 5, exec)
}

func TestRequireExactlyOnce_SingleAttemptStillExecutes(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, time.Minute, exec.Fn())

	idemtest.RequireExactlyOnce(t, place, "order-42:place", 1, exec)
}

func TestRequireOneWinner(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, time.Minute, exec.Fn())

	const goroutines = 50

	idemtest.RequireOneWinner(t, place, "contended-key", goroutines, exec)
}

func TestRequireOneWinner_RawCheckAndRecordClosure(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()

	dispatch := func(ctx context.Context, key string) error {
		if err := store.CheckAndRecord(ctx, key, time.Minute); err != nil {
			return err
		}

		return exec.Fn()(ctx)
	}

	const goroutines = 50

	idemtest.RequireOneWinner(t, dispatch, "event-7:apply", goroutines, exec)
}

func TestRequireFailClosed(t *testing.T) {
	t.Parallel()

	store := idemtest.Fail(idemtest.NewStore(), errors.New("redis down")) //nolint:err113 // test sentinel
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, time.Minute, exec.Fn())

	idemtest.RequireFailClosed(t, place, "order-42:place", exec)
}

func TestFailing_RecoversMidTest(t *testing.T) {
	t.Parallel()

	failing := idemtest.Fail(idemtest.NewStore(), errors.New("redis down")) //nolint:err113 // test sentinel
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(failing, time.Minute, exec.Fn())

	idemtest.RequireFailClosed(t, place, "key", exec)

	failing.Err = nil

	if err := place(context.Background(), "key"); err != nil {
		t.Fatalf("dispatch after recovery: %v", err)
	}

	idemtest.RequireExecutions(t, exec, 1)
}

func TestRequireAllowsAfterExpiry(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, drillTTL, exec.Fn())

	idemtest.RequireAllowsAfterExpiry(t, place, "order-42:place", drillTTL, exec)
}

// drillTTL is small enough to keep the expiry drill fast and large enough to
// survive scheduler jitter on a busy CI runner.
const drillTTL = 5 * time.Millisecond

func TestForeverClaimNeverFrees(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "event-7:apply", idempotency.Forever); err != nil {
		t.Fatalf("forever claim: %v", err)
	}

	if err := store.CheckAndRecord(ctx, "event-7:apply", idempotency.Forever); !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("forever duplicate: want ErrDuplicate, got %v", err)
	}
}
