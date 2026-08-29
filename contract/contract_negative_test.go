package contract_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/contract/internal"
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
	sabotage  func(*internal.Store) idempotency.Store
	invariant string
	reason    string
}

var negativeScenarios = []negativeScenario{
	{
		name: "CheckAndRecordDuplicateReturnsNil",
		sabotage: func(s *internal.Store) idempotency.Store {
			return silentDuplicate{Store: s}
		},
		invariant: "DuplicateReturnsErrDuplicate",
		reason:    "want ErrDuplicate",
	},
	{
		name: "CheckAndRecordDuplicateGenericError",
		sabotage: func(s *internal.Store) idempotency.Store {
			return genericDuplicateError{Store: s}
		},
		invariant: "DuplicateReturnsErrDuplicate",
		reason:    "want ErrDuplicate",
	},
	{
		name: "RecordAcceptsNonPositiveTTL",
		sabotage: func(s *internal.Store) idempotency.Store {
			return ttlBlindRecord{Store: s}
		},
		invariant: "RejectsNonPositiveTTL",
		reason:    "want ErrInvalidTTL",
	},
	{
		name: "CheckAndRecordAcceptsNonPositiveTTL",
		sabotage: func(s *internal.Store) idempotency.Store {
			return ttlBlindCheckAndRecord{Store: s}
		},
		invariant: "RejectsNonPositiveTTL",
		reason:    "want ErrInvalidTTL",
	},
}

// TestRunTestsDetectsBrokenStores re-executes the test binary once per
// scenario. The child runs the full suite against the broken Store; because a
// contract violation fails the given *testing.T (t.Fatal), an expected failure
// cannot be observed inside the parent's own test tree (a failed subtest
// always fails its ancestors). The subprocess isolates the expected failure so
// this parent test can assert on it: the child must exit non-zero AND its
// output must name the violated invariant, proving the suite both detects
// the break and diagnoses it.
func TestRunTestsDetectsBrokenStores(t *testing.T) {
	t.Parallel()

	if scenario := os.Getenv(helperEnv); scenario != "" {
		runBroken(t, scenario)
		return
	}

	for _, scenario := range negativeScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command(os.Args[0], "-test.run", "^TestRunTestsDetectsBrokenStores$", "-test.v")

			cmd.Env = append(os.Environ(), helperEnv+"="+scenario.name)

			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("suite passed against a deliberately broken Store (%s): the %s invariant did not catch it\noutput:\n%s",
					scenario.name, scenario.invariant, out)
			}

			if !strings.Contains(string(out), scenario.invariant) || !strings.Contains(string(out), scenario.reason) {
				t.Fatalf("suite failed against broken Store (%s) but the failure does not name the violated invariant %s (%q must appear)\noutput:\n%s",
					scenario.name, scenario.invariant, scenario.reason, out)
			}
		})
	}
}

// runBroken runs the full suite against the scenario's broken Store. It is
// executed in a child process (see TestRunTestsDetectsBrokenStores) where its
// failure is the expected, asserted outcome.
func runBroken(t *testing.T, scenarioName string) {
	t.Helper()

	for _, scenario := range negativeScenarios {
		if scenario.name != scenarioName {
			continue
		}

		contract.RunTests(t, func(t *testing.T) idempotency.Store {
			t.Helper()

			store := internal.New()
			t.Cleanup(store.Close)

			return scenario.sabotage(store)
		})

		return
	}

	t.Fatalf("unknown scenario %q", scenarioName)
}

// silentDuplicate swallows duplicates: CheckAndRecord reports success for a
// key that is already recorded.
type silentDuplicate struct {
	*internal.Store
}

func (s silentDuplicate) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)
	if errors.Is(err, idempotency.ErrDuplicate) {
		return nil
	}

	return err
}

// genericDuplicateError returns an ad-hoc error instead of the ErrDuplicate
// sentinel on duplicates.
type genericDuplicateError struct {
	*internal.Store
}

func (s genericDuplicateError) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)
	if errors.Is(err, idempotency.ErrDuplicate) {
		return errors.New("key " + key + " already processed")
	}

	return err
}

// ttlBlindRecord never rejects a non-positive TTL.
type ttlBlindRecord struct {
	*internal.Store
}

func (s ttlBlindRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, max(ttl, time.Nanosecond))
}

// ttlBlindCheckAndRecord never rejects a non-positive TTL.
type ttlBlindCheckAndRecord struct {
	*internal.Store
}

func (s ttlBlindCheckAndRecord) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, max(ttl, time.Nanosecond))
}
