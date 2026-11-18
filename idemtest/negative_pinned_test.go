package idemtest_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
	"github.com/larsartmann/go-idempotency/middleware"
)

// The pinned-drill sabotage proofs: the expiry window walked on a PinnedClock
// must reject both violation classes — a claim the clock advance cannot free,
// and a claim that never holds in the first place.

func TestNegative_RequireAllowsAfterExpiryDetectsStuckClaim(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	forever := middleware.NewCommand(idemtest.NewStore(), idempotency.Forever, exec.Fn())

	expectDrillFailure(
		t,
		"a permanent claim passed off as TTL-windowed",
		"still claimed",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireAllowsAfterExpiry(tb, forever, "key", drillTTL, exec)
		},
	)
}

// sabotageInstantReclaim is the TooShortClaim violation class: the pipeline
// executes on every dispatch — the claim never holds, so every retry
// re-processes. The reclaim-only form of the expiry drill passes this
// pipeline; the in-window rejection assert exists to fail it.
func sabotageInstantReclaim(exec *idemtest.Executions) idemtest.Dispatch {
	return sabotageBypass(exec)
}

func TestNegative_RequireAllowsAfterExpiryDetectsInstantReclaim(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(
		t,
		"a pipeline whose claim never holds",
		"the claim did not hold",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireAllowsAfterExpiry(
				tb,
				sabotageInstantReclaim(exec),
				"key",
				drillTTL,
				exec,
			)
		},
	)
}

func TestNegative_RequireAllowsAfterExpiryPinned(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	exec := idemtest.NewExecutions()
	forever := middleware.NewCommand(idemtest.NewStore(), idempotency.Forever, exec.Fn())

	expectDrillFailure(
		t,
		"a claim that outlives the advanced clock",
		"the claim outlives its window",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireAllowsAfterExpiryPinned(tb, forever, "key", drillTTL, exec, clock)
		},
	)
}

func TestNegative_RequireAllowsAfterExpiryPinnedDetectsInstantReclaim(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	exec := idemtest.NewExecutions()

	expectDrillFailure(
		t,
		"a pipeline whose claim never holds",
		"the claim did not hold",
		func(tb testing.TB) {
			tb.Helper()

			idemtest.RequireAllowsAfterExpiryPinned(
				tb,
				sabotageInstantReclaim(exec),
				"key",
				drillTTL,
				exec,
				clock,
			)
		},
	)
}

func TestNegative_RequireAllowsAfterExpiryPinnedRejectsNilClock(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()

	expectDrillFailure(t, "a nil clock", "clock is nil", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireAllowsAfterExpiryPinned(
			tb,
			sabotageBypass(exec),
			"key",
			drillTTL,
			exec,
			nil,
		)
	})
}

func TestNegative_RequireAllowsAfterExpiryPinnedRejectsNonPositiveTTL(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	exec := idemtest.NewExecutions()

	expectDrillFailure(t, "a non-positive ttl", "want a positive duration", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireAllowsAfterExpiryPinned(tb, sabotageBypass(exec), "key", 0, exec, clock)
	})
}
