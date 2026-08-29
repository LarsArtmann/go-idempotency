package contract_test

import (
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

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
