package idemtest_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
)

// errSabotageStoreDown is the sabotage sentinel for the fail-open proof: a
// package-level static error, because err113 rightly rejects dynamic errors
// at the call site.
var errSabotageStoreDown = errors.New("redis down")

// distinguish "the drill failed the fake TB" from a real panic.
type fatalSentinel struct{ msg string }

// captureTB records a Fatalf without stopping the process. It exists because
// every Require* helper must be sabotage-proven: an assertion that cannot be
// shown to fail is a check that passes everything. Any testing.TB method other
// than Helper/Fatalf panics on the nil embedded interface — itself a finding
// about the helper under test.
type captureTB struct {
	testing.TB

	failed bool
	msg    string
}

func (c *captureTB) Helper() {}

func (c *captureTB) Fatalf(format string, args ...any) {
	c.failed = true
	c.msg = fmt.Sprintf(format, args...)

	panic(fatalSentinel{msg: c.msg})
}

// expectDrillFailure runs drill against a captureTB and fails t unless the
// drill reported a failure whose message names the violated contract via
// wantFragment — the proof that the assertion detects this violation and
// reports it as the violation it is, not as some unrelated crash.
func expectDrillFailure(t *testing.T, violation, wantFragment string, drill func(tb testing.TB)) {
	t.Helper()

	capture := &captureTB{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(fatalSentinel); !ok {
					panic(r)
				}
			}
		}()

		drill(capture)
	}()

	if !capture.failed {
		t.Fatalf("drill passed despite %s: the assertion does not detect this violation", violation)
	}

	if !strings.Contains(capture.msg, wantFragment) {
		t.Fatalf(
			"drill failed with %q, want message naming the violation (%q): the failure does not report %s",
			capture.msg,
			wantFragment,
			violation,
		)
	}
}

// sabotageBypass executes on every dispatch: the store is never consulted, so
// a retry storm processes every attempt.
func sabotageBypass(exec *idemtest.Executions) idemtest.Dispatch {
	return func(ctx context.Context, _ string) error {
		return exec.Fn()(ctx)
	}
}

// sabotageTOCTOU splits the claim into Seen + Record with a gap between them,
// modeling the round-trip latency that makes the race realistic: goroutines
// released together all pass Seen before the first Record lands.
func sabotageTOCTOU(store *idemtest.Store, exec *idemtest.Executions) idemtest.Dispatch {
	return func(ctx context.Context, key string) error {
		seen, err := store.Seen(ctx, key)
		if err != nil {
			return err
		}

		if seen {
			return idempotency.ErrDuplicate
		}

		time.Sleep(sabotageGap)

		if err := store.Record(ctx, key, time.Minute); err != nil {
			return err
		}

		return exec.Fn()(ctx)
	}
}

// sabotageGap widens the TOCTOU window to a realistic store round-trip.
const sabotageGap = 2 * time.Millisecond

// sabotageFailOpen swallows the store failure and executes anyway.
func sabotageFailOpen(store idempotency.Store, exec *idemtest.Executions) idemtest.Dispatch {
	return func(ctx context.Context, key string) error {
		_ = store.CheckAndRecord(ctx, key, time.Minute)

		return exec.Fn()(ctx)
	}
}

func TestNegative_RequireExactlyOnceDetectsStoreBypass(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(
		t,
		"a dispatch that bypasses the store",
		"the side effect ran more than once",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireExactlyOnce(tb, sabotageBypass(exec), "order-42:place", 5, exec)
		},
	)
}

func TestNegative_RequireExactlyOnceDetectsUnexpectedError(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	attempts := 0

	failingMidway := func(ctx context.Context, _ string) error {
		attempts++

		if attempts == 2 {
			return errors.New("connection reset") //nolint:err113 // sabotage sentinel
		}

		return exec.Fn()(ctx)
	}

	expectDrillFailure(
		t,
		"a dispatch that errors mid-storm",
		"unexpected error",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireExactlyOnce(tb, failingMidway, "key", 3, exec)
		},
	)
}

func TestNegative_RequireExactlyOnceRejectsZeroAttempts(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(t, "a zero-attempt storm", "want at least 1", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireExactlyOnce(tb, sabotageBypass(exec), "key", 0, exec)
	})
}

func TestNegative_RequireOneWinnerDetectsTOCTOU(t *testing.T) {
	t.Parallel()

	store := idemtest.NewStore()
	exec := idemtest.NewExecutions()

	expectDrillFailure(
		t,
		"a Seen + Record split (TOCTOU race)",
		"more than one dispatch executed",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireOneWinner(tb, sabotageTOCTOU(store, exec), "contended-key", 100, exec)
		},
	)
}

func TestNegative_RequireOneWinnerRejectsZeroGoroutines(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(t, "a zero-goroutine race", "want at least 1", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireOneWinner(tb, sabotageBypass(exec), "key", 0, exec)
	})
}

func TestNegative_RequireFailClosedDetectsFailOpen(t *testing.T) {
	t.Parallel()

	store := idemtest.Fail(idemtest.NewStore(), errSabotageStoreDown)
	exec := idemtest.NewExecutions()

	expectDrillFailure(
		t,
		"a pipeline that executes on unknown store state",
		"executed on unknown duplicate state",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireFailClosed(tb, sabotageFailOpen(store, exec), "key", exec)
		},
	)
}

func TestNegative_RequireFailClosedDetectsSentinelMisreport(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	misreporting := func(context.Context, string) error {
		return idempotency.ErrDuplicate
	}

	expectDrillFailure(
		t,
		"a store failure misreported as ErrDuplicate",
		"misreported an unknown state",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireFailClosed(tb, misreporting, "key", exec)
		},
	)
}

func TestNegative_RequireAllowsAfterExpiryRejectsNonPositiveTTL(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(t, "a non-positive ttl", "want a positive duration", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireAllowsAfterExpiry(tb, sabotageBypass(exec), "key", 0, exec)
	})
}

func TestNegative_RequireExecutionsDetectsWrongCount(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	_ = exec.Fn()(context.Background())

	expectDrillFailure(t, "an execution-count mismatch", "want 2", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireExecutions(tb, exec, 2)
	})
}

func TestNegative_RequireClaimedDetectsWrongKey(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	_ = recorder.CheckAndRecord(context.Background(), "per-request-uuid-7", time.Minute)

	expectDrillFailure(
		t,
		"a claim recorded under the wrong key",
		`no CheckAndRecord(key="order-42:place"`,
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireClaimed(tb, recorder, "order-42:place", time.Minute)
		},
	)
}

func TestNegative_RequireClaimedDetectsWrongTTL(t *testing.T) {
	t.Parallel()

	recorder := idemtest.NewRecorder(idemtest.NewStore())
	_ = recorder.CheckAndRecord(context.Background(), "order-42:place", time.Second)

	expectDrillFailure(t, "a claim recorded with the wrong TTL", "ttl=1h0m0s", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireClaimed(tb, recorder, "order-42:place", time.Hour)
	})
}
