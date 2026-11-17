package contract_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// foreverExpires treats the Forever sentinel as a short TTL. It models a
// backend that computes now.Add(ttl) for every value and then truncates or
// mis-stores the result, so a permanent claim expires almost immediately.
type foreverExpires struct {
	*teststore.Store
}

const foreverExpiresTTL = 10 * time.Millisecond

func (s foreverExpires) ttlFor(ttl time.Duration) time.Duration {
	if ttl == idempotency.Forever {
		return foreverExpiresTTL
	}

	return ttl
}

func (s foreverExpires) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, s.ttlFor(ttl))
}

func (s foreverExpires) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, s.ttlFor(ttl))
}

// foreverRejected rejects the Forever sentinel with ErrInvalidTTL. It models a
// backend that preserves the ttl > 0 guard but treats the max duration as bad
// input (for example, an adapter that validates against a maximum TTL).
type foreverRejected struct {
	*teststore.Store
}

func (s foreverRejected) Record(ctx context.Context, key string, ttl time.Duration) error {
	if ttl == idempotency.Forever {
		return idempotency.ErrInvalidTTL
	}

	return s.Store.Record(ctx, key, ttl)
}

func (s foreverRejected) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl == idempotency.Forever {
		return idempotency.ErrInvalidTTL
	}

	return s.Store.CheckAndRecord(ctx, key, ttl)
}

// foreverDuplicateSwallowed returns nil for a duplicate whose original claim
// was permanent. It models a backend that special-cases Forever in its write
// path and forgets to report the duplicate on the read path.
type foreverDuplicateSwallowed struct {
	*teststore.Store

	mu      sync.Mutex
	forever map[string]struct{}
}

func newForeverDuplicateSwallowed(s *teststore.Store) *foreverDuplicateSwallowed {
	return &foreverDuplicateSwallowed{Store: s, mu: sync.Mutex{}, forever: make(map[string]struct{})}
}

func (s *foreverDuplicateSwallowed) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)

	if err == nil && ttl == idempotency.Forever {
		s.mu.Lock()
		s.forever[key] = struct{}{}
		s.mu.Unlock()
	}

	if errors.Is(err, idempotency.ErrDuplicate) {
		s.mu.Lock()
		_, permanent := s.forever[key]
		s.mu.Unlock()

		if permanent {
			return nil
		}
	}

	return err
}

// foreverUpgrades turns a live short-TTL claim into a permanent one on a
// later Record(Forever). It models a backend that checks the sentinel before
// checking whether the key is already live.
type foreverUpgrades struct {
	*teststore.Store
}

func (s foreverUpgrades) Record(ctx context.Context, key string, ttl time.Duration) error {
	if ttl == idempotency.Forever {
		return s.ForceRecord(ctx, key, ttl)
	}

	return s.Store.Record(ctx, key, ttl)
}
