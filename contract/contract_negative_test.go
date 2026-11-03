package contract_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// The negative tests in this file prove the opposite of TestRunTestsSelfVerification:
// not only does the suite pass against a correct Store, it FAILS against Stores
// that violate a single documented invariant, and the failure names the
// violated invariant. Without these tests the suite could silently rot into a
// no-op that passes everything.

// helperEnv selects negative-test helper mode in a re-executed test binary.
const helperEnv = "GO_IDEMPOTENCY_CONTRACT_NEGATIVE_HELPER"

// negativeScenario describes one deliberately broken Store: how to sabotage a
// correct internal Store, which suite invariant must catch it, and which
// fragment of the failure output proves the catch. Violations are value-based
// (wrong return values), never timing-based: a non-atomic Seen+Record
// CheckAndRecord is only observably different under a race, which would make
// these tests flaky; the suite's Concurrency subtest covers the race against
// correct implementations.
type negativeScenario struct {
	name      string
	sabotage  func(*teststore.Store) idempotency.Store
	invariant string
	reason    string
}

var negativeScenarios = []negativeScenario{
	{
		name: "CheckAndRecordDuplicateReturnsNil",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return silentDuplicate{Store: s}
		},
		invariant: "DuplicateReturnsErrDuplicate",
		reason:    "want ErrDuplicate",
	},
	{
		name: "CheckAndRecordDuplicateGenericError",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return genericDuplicateError{Store: s}
		},
		invariant: "DuplicateReturnsErrDuplicate",
		reason:    "want ErrDuplicate",
	},
	{
		name: "RecordAcceptsNonPositiveTTL",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return ttlBlindRecord{Store: s}
		},
		invariant: "RejectsNonPositiveTTL",
		reason:    "want ErrInvalidTTL",
	},
	{
		name: "CheckAndRecordAcceptsNonPositiveTTL",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return ttlBlindCheckAndRecord{Store: s}
		},
		invariant: "RejectsNonPositiveTTL",
		reason:    "want ErrInvalidTTL",
	},
	{
		name: "SeenReportsUnseenKeysAsSeen",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return invertedSeen{Store: s}
		},
		invariant: "UnseenKeyReturnsFalse",
		reason:    "want false, got true",
	},
	{
		name: "SeenHidesRecordedKeys",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return invertedSeen{Store: s}
		},
		invariant: "AfterRecordReturnsTrue",
		reason:    "want true, got false",
	},
	{
		name: "IgnoresTTLExpiry",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return neverExpires{Store: s}
		},
		invariant: "LazilyDeletesExpired",
		reason:    "Seen should return false",
	},
	{
		name: "RecordExtendsLiveTTL",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return ttlExtendingRecord{Store: s}
		},
		invariant: "NoopOnExistingKey",
		reason:    "Record extended the TTL",
	},
	{
		name: "RecordIgnoresRerecordAfterExpiry",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return &writeOnceRecord{Store: s, mu: sync.Mutex{}, written: nil}
		},
		invariant: "ReRecordsAfterExpiry",
		reason:    "key should be seen with fresh TTL",
	},
	{
		name: "CheckAndRecordRejectsEveryFreshClaim",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return rejectingCheckAndRecord{Store: s}
		},
		invariant: "FirstCallSucceeds",
		reason:    "want nil",
	},
	{
		name: "CheckAndRecordKeepsExpiredClaims",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return &immortalClaims{Store: s, mu: sync.Mutex{}, claimed: make(map[string]struct{})}
		},
		invariant: "AllowsAfterExpiry",
		reason:    "after expiry: want nil",
	},
	{
		name: "CheckAndRecordAllowsTwoWinners",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return &doubleWinner{Store: s, mu: sync.Mutex{}, winner: 0}
		},
		invariant: "AtomicUnderConcurrency",
		reason:    "want exactly 1",
	},
	{
		name: "CollapsesDistinctKeys",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return keyCollapsing{Store: s}
		},
		invariant: "KeysAreIndependent",
		reason:    "key-B should not be seen",
	},
	{
		name: "DropsEmptyKeys",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return emptyKeyDropper{Store: s}
		},
		invariant: "EmptyKey",
		reason:    "empty key should be seen after Record",
	},
}

// TestRunTestsDetectsBrokenStores re-executes the test binary once per
// scenario. The child runs the suite against the broken Store; because a
// contract violation fails the given *testing.T (t.Fatal), an expected failure
// cannot be observed inside the parent's own test tree (a failed subtest
// always fails its ancestors). The subprocess isolates the expected failure so
// this parent test can assert on it: the child must exit non-zero AND its
// output must name the violated invariant, proving the suite both detects
// the break and diagnoses it.
func TestRunTestsDetectsBrokenStores(t *testing.T) {
	t.Parallel()

	runNegativeScenarios(t, negativeScenarios, helperEnv,
		"^TestRunTestsDetectsBrokenStores$", "contract suite",
		func(t *testing.T, sabotage func(*teststore.Store) idempotency.Store) {
			t.Helper()

			contract.RunTests(t, sabotagedStoreFactory(t, sabotage))
		})
}
