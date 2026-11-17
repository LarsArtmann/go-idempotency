package idemtest_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
)

func TestStore_ClaimsAndDuplicates(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "order-42:place", time.Minute); err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if err := store.CheckAndRecord(ctx, "order-42:place", time.Minute); !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second claim: want ErrDuplicate, got %v", err)
	}

	seen, err := store.Seen(ctx, "order-42:place")
	if err != nil || !seen {
		t.Fatalf("seen after claim = (%t, %v), want (true, nil)", seen, err)
	}
}

func TestStore_ResetEmptiesClaims(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	ctx := context.Background()

	if err := store.Record(ctx, "key", time.Minute); err != nil {
		t.Fatalf("record: %v", err)
	}

	store.Reset()

	if err := store.CheckAndRecord(ctx, "key", time.Minute); err != nil {
		t.Fatalf("claim after reset: %v", err)
	}
}

func TestFail_FailsEveryMethod(t *testing.T) {
	t.Parallel()

	inner := idemtest.NewStore()
	boom := errors.New("redis down") //nolint:err113 // test sentinel
	failing := idemtest.Fail(inner, boom)
	ctx := context.Background()

	if _, err := failing.Seen(ctx, "key"); !errors.Is(err, boom) {
		t.Fatalf("Seen: want boom, got %v", err)
	}

	if err := failing.Record(ctx, "key", time.Minute); !errors.Is(err, boom) {
		t.Fatalf("Record: want boom, got %v", err)
	}

	if err := failing.CheckAndRecord(ctx, "key", time.Minute); !errors.Is(err, boom) {
		t.Fatalf("CheckAndRecord: want boom, got %v", err)
	}
}

func TestFail_NilErrPassesThrough(t *testing.T) {
	t.Parallel()

	failing := idemtest.Fail(idemtest.NewStore(), nil)

	if err := failing.CheckAndRecord(context.Background(), "key", time.Minute); err != nil {
		t.Fatalf("pass-through claim: %v", err)
	}
}

func TestRecorder_RecordsCallsInOrder(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	ctx := context.Background()

	_ = recorder.CheckAndRecord(ctx, "order-42:place", time.Minute)
	_, _ = recorder.Seen(ctx, "order-42:place")
	_ = recorder.Record(ctx, "other", idempotency.Forever)

	calls := recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("calls recorded = %d, want 3: %+v", len(calls), calls)
	}

	wantOps := []idemtest.Op{
		idemtest.OpCheckAndRecord,
		idemtest.OpSeen,
		idemtest.OpRecord,
	}

	for i, want := range wantOps {
		if calls[i].Op != want {
			t.Fatalf("call %d op = %s, want %s", i, calls[i].Op, want)
		}
	}

	if calls[0].Key != "order-42:place" || calls[0].TTL != time.Minute {
		t.Fatalf("call 0 = (%s, %v), want (order-42:place, %v)", calls[0].Key, calls[0].TTL, time.Minute)
	}

	if calls[2].TTL != idempotency.Forever {
		t.Fatalf("call 2 ttl = %v, want Forever", calls[2].TTL)
	}
}

func TestRecorder_RecordsDuplicateErrors(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	ctx := context.Background()

	_ = recorder.CheckAndRecord(ctx, "key", time.Minute)
	_ = recorder.CheckAndRecord(ctx, "key", time.Minute)

	calls := recorder.Calls()
	if !errors.Is(calls[1].Err, idempotency.ErrDuplicate) {
		t.Fatalf("second call err = %v, want ErrDuplicate", calls[1].Err)
	}
}

func TestRecorder_CallsReturnsACopy(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	_, _ = recorder.Seen(context.Background(), "key")

	calls := recorder.Calls()
	calls[0].Key = "mutated"

	if got := recorder.Calls()[0].Key; got != "key" {
		t.Fatalf("Calls() returned a live slice: key mutated to %q", got)
	}
}

func TestRequireClaimed(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	ctx := context.Background()

	_ = recorder.CheckAndRecord(ctx, "order-42:place", time.Minute)
	_, _ = recorder.Seen(ctx, "unrelated")

	idemtest.RequireClaimed(t, recorder, "order-42:place", time.Minute)
}

func TestExecutions_FnCountsAndWrapDelegates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	exec := idemtest.NewExecutions()

	_ = exec.Fn()(ctx)
	_ = exec.Fn()(ctx)

	delegated := false
	_ = exec.Wrap(func(context.Context) error {
		delegated = true

		return nil
	})(ctx)

	if !delegated {
		t.Fatal("Wrap did not delegate to the wrapped function")
	}

	idemtest.RequireExecutions(t, exec, 3)
}

func TestExecutions_CountingHandlerCountsRequests(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := idemtest.Counting(exec, http.NotFoundHandler())

	_ = idemtest.Send(t, handler, "key")
	_ = idemtest.Send(t, handler, "key")

	idemtest.RequireExecutions(t, exec, 2)
}
