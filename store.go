package idempotency

import (
	"context"
	"math"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrDuplicate is returned by [Store.CheckAndRecord] when the key was already
// recorded and has not expired. It is classified as a Conflict: a retried
// command with the same idempotency key conflicts with a prior, still-valid
// recording.
//
// The declared type is the error interface, not the concrete classification
// type: callers contract on the sentinel value (via errors.Is), never on the
// implementation behind it, so a future classification change cannot break
// compilation of consumer code.
var ErrDuplicate error = errorfamily.NewConflict(
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
// [Forever] is a positive duration and is therefore accepted. Zero is not a
// shorthand for forever.
//
// It is classified as a Rejection (HTTP 400, non-retryable): the caller passed
// bad input. Check with errors.Is(err, idempotency.ErrInvalidTTL).
//
// Like [ErrDuplicate], the declared type is the error interface: the sentinel
// is the contract, the classification type is an implementation detail.
var ErrInvalidTTL error = errorfamily.NewRejection(
	"idempotency.invalid-ttl",
	"ttl must be positive",
)

// Forever is a TTL sentinel meaning "never expires". Its value is
// math.MaxInt64. Pass it to [Store.Record] or [Store.CheckAndRecord] to create
// a permanent claim.
//
// Scope: permanent claims are for consumer and event deduplication, and for
// permanent domain invariants (for example, "an invoice may only be submitted
// once"). Request-facing idempotency keys should stay TTL-windowed: a
// permanent claim on a crashed request becomes a permanent lockout (ADR-004).
//
// Implementations must store the sentinel directly and must not compute
// now.Add(Forever). time.Time can represent that instant (about year 2318),
// but UnixNano overflows in 2262, so backends that persist epoch nanoseconds
// need an explicit encoding: no Redis expiry, a NULL SQL expiry column, or a
// max sentinel stored as-is. See ADR-006 and the package documentation.
//
// Forever is a positive duration, so the ttl <= 0 validation still applies:
// zero is NOT a shorthand for forever.
const Forever time.Duration = math.MaxInt64

// Store tracks opaque keys (typically command idempotency keys) to prevent
// duplicate processing. When a key is seen for the first time, the store
// records it with a TTL. Subsequent lookups for the same key report it as seen
// until the TTL expires, or never if the key was recorded with [Forever].
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
	// Expired entries are treated as absent: implementations return false
	// (and may garbage-collect the entry).
	//
	// Verified by the contract test suite
	// (github.com/larsartmann/go-idempotency/contract), subtests
	// Seen/UnseenKeyReturnsFalse, Seen/AfterRecordReturnsTrue,
	// Seen/LazilyDeletesExpired, and Seen/ForeverSurvives.
	Seen(ctx context.Context, key string) (bool, error)

	// Record marks the key as seen with the given TTL. If the key is already
	// recorded and unexpired, it is a no-op (the TTL is not extended — the
	// original expiry stands). An expired key is re-recorded with a fresh
	// TTL. Returns [ErrInvalidTTL] if ttl is not positive.
	//
	// Pass [Forever] for a permanent claim. The no-op rule applies in both
	// directions: a live short-TTL claim is not upgraded to permanent, and a
	// live permanent claim is not downgraded to a short TTL.
	//
	// Verified by the contract test suite subtests Record/NoopOnExistingKey,
	// Record/ReRecordsAfterExpiry, Record/RejectsNonPositiveTTL,
	// Record/ForeverNeverExpires, Record/ForeverNoopOnExistingKey, and
	// Record/ForeverNotDowngraded.
	Record(ctx context.Context, key string, ttl time.Duration) error

	// CheckAndRecord atomically reports whether the key was already seen and,
	// if not, records it. Returns [ErrDuplicate] if the key was already
	// recorded and not expired. Returns [ErrInvalidTTL] if ttl is not
	// positive.
	//
	// Pass [Forever] for a permanent claim. A live permanent claim is a
	// duplicate for every later caller.
	//
	// Implementations MUST make this atomic (single lock or single round-trip)
	// to prevent the TOCTOU race that a separate Seen + Record pair would
	// create. For [MemoryStore] this is a single mutex. When implementing the
	// interface against your own backend, use its native atomic primitive:
	// Redis SET NX EX, or a single SQL upsert whose DO UPDATE only fires when
	// the existing row has expired. A bare ON CONFLICT DO NOTHING only frees
	// the key once the expired row is purged, so the claim outlives its TTL.
	//
	// Verified by the contract test suite subtests
	// CheckAndRecord/FirstCallSucceeds,
	// CheckAndRecord/DuplicateReturnsErrDuplicate,
	// CheckAndRecord/AllowsAfterExpiry,
	// CheckAndRecord/RejectsNonPositiveTTL,
	// CheckAndRecord/Forever,
	// Concurrency/AtomicUnderConcurrency (200-goroutine race, exactly one
	// winner), and Cross-cutting/KeysAreIndependent and
	// Cross-cutting/EmptyKey.
	CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}

// MemoryStore is an in-memory [Store] with TTL-based expiration. It runs an
// optional background goroutine that sweeps expired entries. Safe for
// concurrent use.
//
// Expired entries are also deleted lazily on read, so the map cannot grow
// unboundedly even when the sweep goroutine is disabled (sweepInterval == 0).
//
// Entries recorded with [Forever] are stored as the zero time.Time and are
// never expired, lazily deleted, or swept.
//
// Every expiry decision (lazy deletion, re-recording, claiming, sweeping)
// reads the clock installed at construction: the wall clock by default, or
// the function passed to [WithClock]. Tests pin the clock to make TTL
// behavior deterministic instead of sleeping real time.
//
// MemoryStore ignores the context.Context parameter on all methods. The
// parameter exists on the [Store] interface so that custom backend
// implementations (Redis, SQL, etc.) can honor cancellation and timeouts.
//
// Deprecated: MemoryStore is intended for development, testing, and
// single-process use only. It does not survive restarts and cannot be shared
// across instances. For production, implement the [Store] interface against
// your persistence backend and validate it with the contract test suite (see
// package contract). A step-by-step migration guide lives in
// docs/migrating-from-memorystore.md. MemoryStore will be removed in a future
// major version.
type MemoryStore struct {
	mu       sync.RWMutex
	entries  map[string]time.Time // key → expiresAt
	stop     chan struct{}
	stopOnce sync.Once
	now      func() time.Time
}

// NewMemoryStore creates an in-memory idempotency store and, when sweepInterval
// is positive, starts a background goroutine that sweeps expired entries every
// sweepInterval. Call [MemoryStore.Close] to stop the sweeper.
//
// Pass sweepInterval == 0 to disable the background sweep; lazy deletion on
// read still bounds growth.
//
// Options customize construction; see [WithClock] to override the clock the
// store reads for expiry decisions.
//
// Deprecated: Use only for development and testing. See [MemoryStore] for the
// rationale and docs/migrating-from-memorystore.md for the migration path.
func NewMemoryStore(sweepInterval time.Duration, opts ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{ //nolint:exhaustruct_v5 // mu, stopOnce are zero-valued
		entries: make(map[string]time.Time),
		stop:    make(chan struct{}),
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}

	if s.now == nil {
		s.now = time.Now
	}

	if sweepInterval > 0 {
		go s.sweep(sweepInterval)
	}

	return s
}

// Seen reports whether the key is currently recorded and not expired.
// Expired entries are lazily deleted on read; [Forever] entries are never
// expired or deleted.
func (s *MemoryStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.entries[key]
	if !ok {
		return false, nil
	}

	if isLive(exp, s.now()) {
		return true, nil
	}

	delete(s.entries, key)

	return false, nil
}

// isLive reports whether exp represents a claim that has not expired. A zero
// time is the in-process representation of a [Forever] claim.
//
// A claim is live strictly before its expiry: at the exact expiry instant
// (now == exp) it is already expired. The property test
// TestProperty_TTLExpiry pins this boundary for [MemoryStore]; backends with
// independent clocks should treat the edge as an implementation detail and
// pin their own semantics.
func isLive(exp, now time.Time) bool {
	return exp.IsZero() || now.Before(exp)
}

// expiryFor converts a TTL into the stored expiry. [Forever] is stored as the
// zero time so no caller ever performs now.Add(Forever) arithmetic.
func expiryFor(now time.Time, ttl time.Duration) time.Time {
	if ttl == Forever {
		return time.Time{}
	}

	return now.Add(ttl)
}

// Record marks the key as seen with the given TTL. If the key is already
// recorded and not expired, it is a no-op (the existing expiry is not extended).
// If the key is expired (even if still in the map before sweep), Record sets a
// fresh TTL. Returns [ErrInvalidTTL] if ttl is not positive.
//
// Pass [Forever] for a permanent claim. The no-op rule applies in both
// directions: a live short-TTL claim is not upgraded, and a live [Forever]
// claim is not downgraded.
func (s *MemoryStore) Record(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if exp, ok := s.entries[key]; !ok || !isLive(exp, now) {
		s.entries[key] = expiryFor(now, ttl)
	}

	return nil
}

// CheckAndRecord atomically claims a key. Returns [ErrDuplicate] if the key was
// already recorded and not expired, or [ErrInvalidTTL] if ttl is not positive.
// The check and the record happen under a single write lock, so concurrent
// callers with the same key are serialized: exactly one wins.
//
// Pass [Forever] for a permanent claim; the stored expiry is the zero time.
func (s *MemoryStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if exp, ok := s.entries[key]; ok && isLive(exp, now) {
		return ErrDuplicate
	}

	s.entries[key] = expiryFor(now, ttl)

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
			// The ticker itself stays on the real wall clock; only the expiry
			// comparison reads the store's clock (see [WithClock]).
			now := s.now()

			s.mu.Lock()
			for key, exp := range s.entries {
				if exp.IsZero() {
					continue // Forever: never expires, never swept
				}

				if now.After(exp) {
					delete(s.entries, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
