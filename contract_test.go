package idempotency_test

import (
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
)

// TestContract_MemoryStore runs the full contract test suite against
// MemoryStore, the deprecated in-process implementation. This verifies that it
// satisfies every invariant that consumers will verify against their own
// backends.
func TestContract_MemoryStore(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := idempotency.NewMemoryStore(0)
		t.Cleanup(store.Close)

		return store
	})
}
