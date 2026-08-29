package contract_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// defaultTimingScale is the stretch used when the timing-scale env knob is
// unset: noticeably slower than the defaults, fast enough for every commit.
const defaultTimingScale = 2.0

// timingScaleEnv lets CI stretch the scaled self-test beyond the default
// without code changes; the weekly scheduled run uses 3 to exercise the
// slow-machine path harder.
const timingScaleEnv = "GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE"

// timingScale resolves the TimingScale for the scaled self-test.
func timingScale(t *testing.T) float64 {
	t.Helper()

	raw := os.Getenv(timingScaleEnv)
	if raw == "" {
		return defaultTimingScale
	}

	scale, err := strconv.ParseFloat(raw, 64)
	if err != nil || scale <= 0 {
		t.Fatalf("%s=%q: want a positive number", timingScaleEnv, raw)
	}

	return scale
}

// TestRunTestsSelfVerification runs the full contract suite against the
// internal in-memory Store. This exercises the suite itself: without it,
// contract/ ships with zero coverage and a broken RunTests would go unnoticed
// by this repository's own CI.
func TestRunTestsSelfVerification(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := teststore.New()
		t.Cleanup(store.Close)

		return store
	})
}

// TestRunTestsStrict_ScaledTimings proves the options path: the full suite
// passes with stretched timings (exercise the slow-CI configuration
// occasionally so it cannot rot). The stretch defaults to 2 and can be raised
// via the timing-scale env knob for scheduled slow-machine runs.
func TestRunTestsStrict_ScaledTimings(t *testing.T) {
	t.Parallel()

	contract.RunTestsStrict(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := teststore.New()
		t.Cleanup(store.Close)

		return store
	}, contract.Options{TimingScale: timingScale(t)})
}

// TestRunTestsContextAwareSelfVerification runs the optional cancellation
// suite against the context-honoring Store wrapper, exercising the suite the
// same way the main-suite self-test exercises RunTests.
func TestRunTestsContextAwareSelfVerification(t *testing.T) {
	t.Parallel()

	contract.RunTestsContextAware(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := teststore.NewContextAware()
		t.Cleanup(store.Close)

		return store
	})
}
