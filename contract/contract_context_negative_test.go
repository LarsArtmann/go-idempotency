package contract_test

import (
	"context"
	"testing"
	"time"

	idempotency "github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// The negative tests here mirror TestRunTestsDetectsBrokenStores for the
// optional cancellation suite: RunTestsContextAware must FAIL against Stores
// that violate one of its two invariants, and the failure must name the
// violated subtest. Because the embedded teststore is context-blind in every
// method, some scenarios also trip sibling subtests (collateral failures);
// the harness only requires the target invariant and reason to appear.

// contextHelperEnv selects cancellation-negative-test helper mode in a
// re-executed test binary.
const contextHelperEnv = "GO_IDEMPOTENCY_CONTRACT_CONTEXT_NEGATIVE_HELPER"

// contextNegativeScenarios describes deliberately broken Stores for
// RunTestsContextAware: how to sabotage a correct internal Store, which suite
// subtest must catch it, and which fragment of the failure output proves the
// catch. Same shape as negativeScenarios in contract_negative_test.go.
var contextNegativeScenarios = []negativeScenario{
	{
		// The bare teststore ignores context: the default for local
		// stores, and exactly why the cancellation suite is opt-in.
		name: "SeenBlindToCancellation",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return s
		},
		invariant: "CanceledReturnsContextError",
		reason:    "want context.Canceled",
	},
	{
		name: "RecordBlindToCancellation",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return recordBlindToCancellation{Store: s}
		},
		invariant: "CanceledReturnsContextError",
		reason:    "want context.Canceled",
	},
	{
		name: "RecordPoisonsClaimOnCancellation",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return recordPoisoner{Store: s}
		},
		invariant: "CanceledDoesNotConsumeClaim",
		reason:    "consumed the claim",
	},
	{
		name: "CheckAndRecordBlindToCancellation",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return checkAndRecordBlindToCancellation{Store: s}
		},
		invariant: "CanceledReturnsContextError",
		reason:    "want context.Canceled",
	},
	{
		name: "CheckAndRecordPoisonsClaimOnCancellation",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return checkAndRecordPoisoner{Store: s}
		},
		invariant: "CanceledDoesNotConsumeClaim",
		reason:    "consumed the claim",
	},
}

// TestRunTestsContextAwareDetectsBrokenStores re-executes the test binary
// once per scenario (see TestRunTestsDetectsBrokenStores for why the expected
// failure needs a subprocess): the child must exit non-zero AND its output
// must name the violated subtest, proving the cancellation suite both detects
// the break and diagnoses it.
func TestRunTestsContextAwareDetectsBrokenStores(t *testing.T) {
	t.Parallel()

	runNegativeScenarios(t, contextNegativeScenarios, contextHelperEnv,
		"^TestRunTestsContextAwareDetectsBrokenStores$", "cancellation suite",
		func(t *testing.T, sabotage func(*teststore.Store) idempotency.Store) {
			t.Helper()

			contract.RunTestsContextAware(t, sabotagedStoreFactory(t, sabotage))
		})
}

// recordBlindToCancellation delegates Record without checking the context:
// a canceled call proceeds and reports success.
type recordBlindToCancellation struct {
	*teststore.Store
}

func (s recordBlindToCancellation) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, ttl)
}

// recordPoisoner returns the context error but records the claim anyway,
// poisoning the key: the retry after the cancellation hits ErrDuplicate
// until TTL expiry.
type recordPoisoner struct {
	*teststore.Store
}

func (s recordPoisoner) Record(ctx context.Context, key string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		_ = s.Store.Record(context.Background(), key, ttl)

		return err
	}

	return s.Store.Record(ctx, key, ttl)
}

// checkAndRecordBlindToCancellation delegates CheckAndRecord without
// checking the context: a canceled call proceeds and reports success.
type checkAndRecordBlindToCancellation struct {
	*teststore.Store
}

func (s checkAndRecordBlindToCancellation) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, ttl)
}

// checkAndRecordPoisoner returns the context error but consumes the claim
// anyway, so the retry after the cancellation hits ErrDuplicate until TTL
// expiry.
type checkAndRecordPoisoner struct {
	*teststore.Store
}

func (s checkAndRecordPoisoner) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		_ = s.Store.CheckAndRecord(context.Background(), key, ttl)

		return err
	}

	return s.Store.CheckAndRecord(ctx, key, ttl)
}
