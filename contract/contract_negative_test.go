package contract_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

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
			return &writeOnceRecord{Store: s}
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
			return &immortalClaims{Store: s, claimed: make(map[string]struct{})}
		},
		invariant: "AllowsAfterExpiry",
		reason:    "after expiry: want nil",
	},
	{
		name: "CheckAndRecordAllowsTwoWinners",
		sabotage: func(s *teststore.Store) idempotency.Store {
			return &doubleWinner{Store: s}
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

			cmd := exec.CommandContext(t.Context(), os.Args[0],
				"-test.run", "^TestRunTestsDetectsBrokenStores$", "-test.v")

			cmd.Env = append(os.Environ(), helperEnv+"="+scenario.name)

			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf(
					"suite passed against a deliberately broken Store (%s): the %s invariant did not catch it\noutput:\n%s",
					scenario.name,
					scenario.invariant,
					out,
				)
			}

			if !strings.Contains(string(out), scenario.invariant) || !strings.Contains(string(out), scenario.reason) {
				t.Fatalf(
					"suite failed against broken Store (%s) but the failure does not name the violated invariant %s (%q must appear)\noutput:\n%s",
					scenario.name,
					scenario.invariant,
					scenario.reason,
					out,
				)
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

			store := teststore.New()
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
	*teststore.Store
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
	*teststore.Store
}

func (s genericDuplicateError) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)
	if errors.Is(err, idempotency.ErrDuplicate) {
		//nolint:err113 // deliberately not a sentinel: this is the broken implementation under test
		return errors.New("key " + key + " already processed")
	}

	return err
}

// ttlBlindRecord never rejects a non-positive TTL.
type ttlBlindRecord struct {
	*teststore.Store
}

func (s ttlBlindRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, max(ttl, time.Nanosecond))
}

// ttlBlindCheckAndRecord never rejects a non-positive TTL.
type ttlBlindCheckAndRecord struct {
	*teststore.Store
}

func (s ttlBlindCheckAndRecord) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, max(ttl, time.Nanosecond))
}

// invertedSeen reports the opposite of the truth: unseen keys look seen and
// recorded keys look unseen. It models a backend whose existence check is
// inverted (e.g. a wrong EXISTS / NOT EXISTS query).
type invertedSeen struct {
	*teststore.Store
}

func (s invertedSeen) Seen(ctx context.Context, key string) (bool, error) {
	seen, err := s.Store.Seen(ctx, key)

	return !seen, err
}

// neverExpireMultiplier stretches every TTL far beyond any test window, so
// entries recorded through this wrapper never expire during a suite run.
const neverExpireMultiplier = 1000

// neverExpires ignores TTL expiry entirely: nothing it records ever expires.
// It models a backend that stores keys without honoring the requested expiry.
type neverExpires struct {
	*teststore.Store
}

func (s neverExpires) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, ttl*neverExpireMultiplier)
}

func (s neverExpires) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, ttl*neverExpireMultiplier)
}

// ttlExtendingRecord extends a live entry's TTL on every Record, violating the
// no-extension rule. It models a backend implemented with plain SET (no NX)
// semantics for Record.
type ttlExtendingRecord struct {
	*teststore.Store
}

func (s ttlExtendingRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.ForceRecord(ctx, key, ttl)
}

// writeOnceRecord ignores re-records of a key it has written before, even
// after the entry expired. It models a backend whose Record is a SET NX that
// never refreshes expired claims.
type writeOnceRecord struct {
	*teststore.Store

	mu      sync.Mutex
	written map[string]struct{}
}

func (s *writeOnceRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.Record(ctx, key, ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.written[key]; ok {
		return nil
	}

	if err := s.Store.Record(ctx, key, ttl); err != nil {
		return err
	}

	if s.written == nil {
		s.written = make(map[string]struct{})
	}

	s.written[key] = struct{}{}

	return nil
}

// rejectingCheckAndRecord rejects every fresh claim with ErrDuplicate while
// preserving TTL validation. It models a backend whose existence check always
// reports the key as present.
type rejectingCheckAndRecord struct {
	*teststore.Store
}

func (s rejectingCheckAndRecord) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.CheckAndRecord(ctx, key, ttl)
	}

	return idempotency.ErrDuplicate
}

// immortalClaims remembers every key it has ever claimed and reports it as a
// duplicate forever, even after the TTL expired. It models a backend that
// checks existence without honoring expiry.
type immortalClaims struct {
	*teststore.Store

	mu      sync.Mutex
	claimed map[string]struct{}
}

func (s *immortalClaims) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.CheckAndRecord(ctx, key, ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.claimed[key]; ok {
		return idempotency.ErrDuplicate
	}

	if err := s.Store.CheckAndRecord(ctx, key, ttl); err != nil {
		return err
	}

	s.claimed[key] = struct{}{}

	return nil
}

// doubleWinner allows the first two CheckAndRecord callers to win, then
// reports duplicates. It models an off-by-one claim budget: a broken atomic
// primitive that grants the claim to more than exactly one caller.
type doubleWinner struct {
	*teststore.Store

	mu     sync.Mutex
	winner int
}

func (s *doubleWinner) CheckAndRecord(_ context.Context, _ string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.winner++

	if s.winner <= 2 {
		return nil
	}

	return idempotency.ErrDuplicate
}

// keyCollapsing maps every key onto one internal entry, so recording any key
// marks all keys as seen. It models a backend that loses the key dimension
// (e.g. a fixed or hashed-to-constant storage key).
type keyCollapsing struct {
	*teststore.Store
}

func (s keyCollapsing) Seen(ctx context.Context, _ string) (bool, error) {
	return s.Store.Seen(ctx, "")
}

func (s keyCollapsing) Record(ctx context.Context, _ string, ttl time.Duration) error {
	return s.Store.Record(ctx, "", ttl)
}

func (s keyCollapsing) CheckAndRecord(ctx context.Context, _ string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, "", ttl)
}

// emptyKeyDropper silently drops empty keys instead of tracking them like any
// other key. It models a backend whose client skips empty values.
type emptyKeyDropper struct {
	*teststore.Store
}

func (s emptyKeyDropper) Record(ctx context.Context, key string, ttl time.Duration) error {
	if key == "" {
		return nil
	}

	return s.Store.Record(ctx, key, ttl)
}
