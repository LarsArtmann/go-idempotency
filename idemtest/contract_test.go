package idemtest_test

import (
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/idemtest"
)

// TestContract certifies the blessed test store: every consumer trusting
// idemtest.NewStore is trusting this run. The suite is the same one backend
// implementers use, so the double is held to the identical invariant set
// (atomicity, TTL expiry, Forever semantics, concurrency, error handling).
func TestContract(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		return idemtest.NewStore()
	})
}
