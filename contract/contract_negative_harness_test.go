package contract_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// runNegativeScenarios is the shared negative-test harness, used by the main
// suite and the cancellation suite. In the parent process it re-executes the
// test binary once per scenario; in the child (env var set) it runs runSuite
// against the sabotaged Store, where failing is the expected, asserted
// outcome.
func runNegativeScenarios(
	t *testing.T,
	scenarios []negativeScenario,
	envVar, childPattern, description string,
	runSuite func(t *testing.T, sabotage func(*teststore.Store) idempotency.Store),
) {
	t.Helper()

	if scenarioName := os.Getenv(envVar); scenarioName != "" {
		for _, scenario := range scenarios {
			if scenario.name == scenarioName {
				runSuite(t, scenario.sabotage)

				return
			}
		}

		t.Fatalf("unknown scenario %q", scenarioName)
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run", childPattern, "-test.v")

			cmd.Env = append(os.Environ(), envVar+"="+scenario.name)

			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf(
					"%s passed against a deliberately broken Store (%s): the %s invariant did not catch it\noutput:\n%s",
					description,
					scenario.name,
					scenario.invariant,
					out,
				)
			}

			if !strings.Contains(string(out), scenario.invariant) || !strings.Contains(string(out), scenario.reason) {
				t.Fatalf(
					"%s failed against broken Store (%s) but the failure does not name the violated invariant %s (%q must appear)\noutput:\n%s",
					description,
					scenario.name,
					scenario.invariant,
					scenario.reason,
					out,
				)
			}
		})
	}
}

// sabotagedStoreFactory adapts a sabotage into a [StoreFactory]: a fresh
// internal Store, closed via t.Cleanup, wrapped in the broken behavior.
func sabotagedStoreFactory(
	t *testing.T,
	sabotage func(*teststore.Store) idempotency.Store,
) func(t *testing.T) idempotency.Store {
	t.Helper()

	return func(t *testing.T) idempotency.Store {
		t.Helper()

		store := teststore.New()
		t.Cleanup(store.Close)

		return sabotage(store)
	}
}
