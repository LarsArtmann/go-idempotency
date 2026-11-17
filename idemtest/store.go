package idemtest

import (
	"context"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// Store is a minimal in-memory [idempotency.Store] for tests: one mutex, one
// map of expiries, correct semantics (atomic CheckAndRecord, no TTL extension
// on a live claim, lazy expiry deletion) and nothing else — no sweeper
// goroutine, no lifecycle, no production surface (ADR-007).
//
// It is the blessed test double for consumers whose production backend is
// Redis, SQL, or anything else that survives a restart: import idemtest from
// _test.go files only, and this store never enters a production import path —
// the httptest.Server precedent, not a MemoryStore resurrection.
//
// Like other in-process stores it ignores context cancellation. The
// repository's CI continuously certifies it: contract.RunTests runs against
// NewStore on every commit.
type Store struct {
	inner *teststore.Store
}

var _ idempotency.Store = (*Store)(nil)

// NewStore returns an empty test Store.
func NewStore() *Store {
	return &Store{inner: teststore.New()}
}

// Seen reports whether the key is currently recorded and not expired.
func (s *Store) Seen(ctx context.Context, key string) (bool, error) {
	return s.inner.Seen(ctx, key)
}

// Record marks the key as seen with the given TTL; a no-op on a live claim.
func (s *Store) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.inner.Record(ctx, key, ttl)
}

// CheckAndRecord atomically claims the key: nil on the first claim,
// [idempotency.ErrDuplicate] while the claim is live.
func (s *Store) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.inner.CheckAndRecord(ctx, key, ttl)
}

// Reset empties the store. Phase-isolation helper for tests that reuse one
// instance across scenarios; prefer a fresh NewStore per subtest otherwise.
func (s *Store) Reset() {
	s.inner.Reset()
}
