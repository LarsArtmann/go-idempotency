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
	return &demoStore{expires: make(map[string]time.Time)}
}

func (s *demoStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.expires[key]

	return ok && time.Now().Before(exp), nil
}

func (s *demoStore) Record(_ context.Context, key string, ttl time.Duration) error {
	// art-dupl:accept example intentionally re-implements the store to demonstrate the contract standalone
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.expires[key]; !ok || time.Now().Before(exp) {
		if ok {
			return nil // live key: never extend
		}
	}

	s.expires[key] = time.Now().Add(ttl)

	return nil
}

func (s *demoStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	// art-dupl:accept example intentionally re-implements the store to demonstrate the contract standalone
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.expires[key]; ok && time.Now().Before(exp) {
		return idempotency.ErrDuplicate
	}

	s.expires[key] = time.Now().Add(ttl)

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
}
