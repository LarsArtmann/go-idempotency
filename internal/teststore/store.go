// Package teststore provides a minimal in-memory idempotency.Store used to
// self-test the contract suite. It is test infrastructure, not a production
// backend: no sweep goroutine, a no-op Close, and just enough state to mirror
// the documented Store semantics (lazy expiry deletion, no TTL extension,
// atomic CheckAndRecord).
package teststore

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Store is a minimal in-memory implementation of [idempotency.Store] with the
// same observable semantics as the deprecated root-package MemoryStore,
// without the background sweeper: expired entries are deleted lazily on read,
// Record never extends the TTL of a live entry, and CheckAndRecord is a
// single atomic critical section.
type Store struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		entries: make(map[string]time.Time),
	}
}

// Seen reports whether the key is currently recorded and not expired.
// Expired entries are lazily deleted on read.
func (s *Store) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.entries[key]
	if !ok {
		return false, nil
	}

	if time.Now().Before(exp) {
		return true, nil
	}

	delete(s.entries, key)

	return false, nil
}

// Record marks the key as seen with the given TTL. If the key is already
// recorded and not expired, it is a no-op (the existing expiry is not
// extended). If the key is expired, Record sets a fresh TTL. Returns
// [idempotency.ErrInvalidTTL] if ttl is not positive.
func (s *Store) Record(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; !ok || !now.Before(exp) {
		s.entries[key] = now.Add(ttl)
	}

	return nil
}

// CheckAndRecord atomically claims a key. Returns
// [idempotency.ErrDuplicate] if the key was already recorded and not expired,
// or [idempotency.ErrInvalidTTL] if ttl is not positive. The check and the
// record happen under a single lock, so concurrent callers with the same key
// are serialized: exactly one wins.
func (s *Store) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; ok && now.Before(exp) {
		return idempotency.ErrDuplicate
	}

	s.entries[key] = now.Add(ttl)

	return nil
}

// Close is a no-op; the Store holds no background resources. It exists so the
// Store satisfies factory contracts that register Close via t.Cleanup.
func (s *Store) Close() {}

// Reset empties the store. Test-isolation helper for tests that reuse one
// instance across phases.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]time.Time)
}
