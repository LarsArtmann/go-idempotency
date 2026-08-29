package idempotency

import (
	"context"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrDuplicate is returned by [Store.CheckAndRecord] when the key was already
// recorded and has not expired. It is classified as a Conflict: a retried
// command with the same idempotency key conflicts with a prior, still-valid
// recording.
var ErrDuplicate = errorfamily.NewConflict(
	"idempotency.duplicate",
	"key has already been recorded",
)

// ErrInvalidTTL is returned by [Store.Record] and [Store.CheckAndRecord] when
// the TTL is not positive. A non-positive TTL records an expiry that is
// already in the past, so the key protects nothing: the very next caller would
// also succeed, silently breaking the exactly-once guarantee that is this
// library's purpose. Rejecting the value up front makes the misuse a loud,
// fixable error instead of a silent correctness hole.
//
// It is classified as a Rejection (HTTP 400, non-retryable): the caller passed
// bad input. Check with errors.Is(err, idempotency.ErrInvalidTTL).
var ErrInvalidTTL = errorfamily.NewRejection(
	"idempotency.invalid-ttl",
	"ttl must be positive",
)

// Store tracks opaque keys (typically command idempotency keys) to prevent
// duplicate processing. When a key is seen for the first time, the store
// records it with a TTL. Subsequent lookups for the same key report it as seen
// until the TTL expires.
//
// This is essential for at-least-once delivery: if a client submits a command,
// loses the acknowledgement, and retries, the store prevents the command from
// executing twice.
//
// Implementations must be safe for concurrent use. [MemoryStore] is provided
// for development and testing only (it is deprecated); implement this
// interface against your own backend (Redis, SQL, etc.) for production.
type Store interface {
	// Seen reports whether the key is currently recorded and not expired.
	Seen(ctx context.Context, key string) (bool, error)

	// Record marks the key as seen with the given TTL. If the key is already
	// recorded, it is a no-op (the TTL is not extended). Returns [ErrInvalidTTL]
	// if ttl is not positive.
	Record(ctx context.Context, key string, ttl time.Duration) error

	// CheckAndRecord atomically reports whether the key was already seen and,
	// if not, records it. Returns [ErrDuplicate] if the key was already
	// recorded and not expired. Returns [ErrInvalidTTL] if ttl is not positive.
	//
	// Implementations MUST make this atomic (single lock or single round-trip)
	// to prevent the TOCTOU race that a separate Seen + Record pair would
	// create. For [MemoryStore] this is a single mutex. When implementing the
	// interface against your own backend, use its native atomic primitive:
	// Redis SET NX, SQL INSERT ... ON CONFLICT DO NOTHING, etc.
	CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}

// MemoryStore is an in-memory [Store] with TTL-based expiration. It runs an
// optional background goroutine that sweeps expired entries. Safe for
// concurrent use.
//
// Expired entries are also deleted lazily on read, so the map cannot grow
// unboundedly even when the sweep goroutine is disabled (sweepInterval == 0).
//
// MemoryStore ignores the context.Context parameter on all methods. The
// parameter exists on the [Store] interface so that custom backend
// implementations (Redis, SQL, etc.) can honor cancellation and timeouts.
//
// Deprecated: MemoryStore is intended for development, testing, and
// single-process use only. It does not survive restarts and cannot be shared
// across instances. For production, implement the [Store] interface against
// your persistence backend and validate it with the contract test suite (see
// package contract). MemoryStore will be removed in a future major version.
type MemoryStore struct {
	mu       sync.RWMutex
	entries  map[string]time.Time // key → expiresAt
	stop     chan struct{}
	stopOnce sync.Once
}

// NewMemoryStore creates an in-memory idempotency store and, when sweepInterval
// is positive, starts a background goroutine that sweeps expired entries every
// sweepInterval. Call [MemoryStore.Close] to stop the sweeper.
//
// Pass sweepInterval == 0 to disable the background sweep; lazy deletion on
// read still bounds growth.
//
// Deprecated: Use only for development and testing. See [MemoryStore] for the
// rationale and the production alternative.
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore {
	s := &MemoryStore{ //nolint:exhaustruct // mu, stopOnce are zero-valued
		entries: make(map[string]time.Time),
		stop:    make(chan struct{}),
	}
	if sweepInterval > 0 {
		go s.sweep(sweepInterval)
	}

	return s
}

// Seen reports whether the key is currently recorded and not expired.
// Expired entries are lazily deleted on read.
func (s *MemoryStore) Seen(_ context.Context, key string) (bool, error) {
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
// recorded and not expired, it is a no-op (the existing expiry is not extended).
// If the key is expired (even if still in the map before sweep), Record sets a
// fresh TTL. Returns [ErrInvalidTTL] if ttl is not positive.
func (s *MemoryStore) Record(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; !ok || !now.Before(exp) {
		s.entries[key] = now.Add(ttl)
	}

	return nil
}

// CheckAndRecord atomically claims a key. Returns [ErrDuplicate] if the key was
// already recorded and not expired, or [ErrInvalidTTL] if ttl is not positive.
// The check and the record happen under a single write lock, so concurrent
// callers with the same key are serialized: exactly one wins.
func (s *MemoryStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; ok && now.Before(exp) {
		return ErrDuplicate
	}

	s.entries[key] = now.Add(ttl)

	return nil
}

// Close stops the background sweep goroutine. Safe to call multiple times.
func (s *MemoryStore) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *MemoryStore) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now := time.Now()

			s.mu.Lock()
			for key, exp := range s.entries {
				if now.After(exp) {
					delete(s.entries, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
