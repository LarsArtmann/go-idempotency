package main

import (
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
)

// TestContract validates the example's demoStore against the full contract
// suite — the same wiring a real backend would use in CI. Keeping it here
// proves the example code is not just compilable but contractually correct.
func TestContract(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := newDemoStore()
		t.Cleanup(store.Close)

		return store
	})
}
