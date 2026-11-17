// Command example is a runnable walkthrough of the go-idempotency SDK: it
// implements the Store interface against a trivial in-process backend (the
// same shape you would swap for Redis or SQL in production — see
// docs/migrating-from-memorystore.md), validates that backend with the
// contract suite, and then shows exactly-once processing from the caller's
// point of view.
//
// Run it with:
//
//	go run ./example
//
// The deprecated MemoryStore is intentionally not used here: production and
// example code should speak to a Store the consumer owns.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// demoStore is a minimal in-memory idempotency.Store: one mutex, one map of
// expiries. Good enough for a demo and for tests; for production swap in a
// backend that survives restarts.
type demoStore struct {
	mu      sync.Mutex
	expires map[string]time.Time
}

func newDemoStore() *demoStore {
	return &demoStore{mu: sync.Mutex{}, expires: make(map[string]time.Time)}
}

// demoLive reports whether exp is unexpired. Zero time means Forever.
func demoLive(exp, now time.Time) bool {
	return exp.IsZero() || now.Before(exp)
}

// demoExpiry stores Forever as zero time, never now.Add(Forever).
func demoExpiry(now time.Time, ttl time.Duration) time.Time {
	// art-dupl:accept example intentionally re-implements the store to demonstrate the contract standalone
	if ttl == idempotency.Forever {
		return time.Time{}
	}

	return now.Add(ttl)
}

func (s *demoStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.expires[key]

	return ok && demoLive(exp, time.Now()), nil
}

func (s *demoStore) Record(_ context.Context, key string, ttl time.Duration) error {
	// art-dupl:accept example intentionally re-implements the store to demonstrate the contract standalone
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.expires[key]; !ok || !demoLive(exp, now) {
		s.expires[key] = demoExpiry(now, ttl)
	}

	return nil
}

func (s *demoStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	// art-dupl:accept example intentionally re-implements the store to demonstrate the contract standalone
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.expires[key]; ok && demoLive(exp, now) {
		return idempotency.ErrDuplicate
	}

	s.expires[key] = demoExpiry(now, ttl)

	return nil
}

func (s *demoStore) Close() {}

func main() {
	store := newDemoStore()
	defer store.Close()

	ctx := context.Background()
	key := "order-42:place" // stable key: entity + operation, not a request UUID

	// Three delivery attempts of the same logical command (at-least-once
	// delivery): only the first one may execute.
	for attempt := 1; attempt <= 3; attempt++ {
		err := store.CheckAndRecord(ctx, key, time.Minute)
		switch {
		case err == nil:
			fmt.Printf("attempt %d: claimed the key -> processing command\n", attempt)
		case errors.Is(err, idempotency.ErrDuplicate):
			fmt.Printf("attempt %d: duplicate -> dropped (already processed)\n", attempt)
		default:
			fmt.Printf("attempt %d: store failure -> fail the request, do NOT process: %v\n", attempt, err)
		}
	}

	// A permanent claim: event/consumer dedupe, where the duplicate window is
	// the lifetime of the store, not a request-retry horizon.
	eventKey := "event:evt-42"

	for attempt := 1; attempt <= 2; attempt++ {
		err := store.CheckAndRecord(ctx, eventKey, idempotency.Forever)
		switch {
		case err == nil:
			fmt.Printf("event attempt %d: claimed permanently -> apply the event\n", attempt)
		case errors.Is(err, idempotency.ErrDuplicate):
			fmt.Printf("event attempt %d: duplicate -> already applied (the claim never expires)\n", attempt)
		default:
			fmt.Printf("event attempt %d: store failure: %v\n", attempt, err)
		}
	}
}
