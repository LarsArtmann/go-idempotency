package contract_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/internal/teststore"
)

// The deliberately broken Stores sabotaged by the negative scenarios in
// contract_negative_test.go (and the cancellation variants in
// contract_context_negative_test.go). Each models a distinct real-world
// backend bug class; each scenario's sabotage wraps a correct internal Store
// in one of these.

// silentDuplicate swallows duplicates: CheckAndRecord reports success for a
// key that is already recorded.
type silentDuplicate struct {
	*teststore.Store
}

func (s silentDuplicate) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)
	if errors.Is(err, idempotency.ErrDuplicate) {
		return nil
	}

	return err
}

// genericDuplicateError returns an ad-hoc error instead of the ErrDuplicate
// sentinel on duplicates.
type genericDuplicateError struct {
	*teststore.Store
}

func (s genericDuplicateError) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := s.Store.CheckAndRecord(ctx, key, ttl)
	if errors.Is(err, idempotency.ErrDuplicate) {
		//nolint:err113 // deliberately not a sentinel: this is the broken implementation under test
		return errors.New("key " + key + " already processed")
	}

	return err
}

// ttlBlindRecord never rejects a non-positive TTL.
type ttlBlindRecord struct {
	*teststore.Store
}

func (s ttlBlindRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, max(ttl, time.Nanosecond))
}

// ttlBlindCheckAndRecord never rejects a non-positive TTL.
type ttlBlindCheckAndRecord struct {
	*teststore.Store
}

func (s ttlBlindCheckAndRecord) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, max(ttl, time.Nanosecond))
}

// invertedSeen reports the opposite of the truth: unseen keys look seen and
// recorded keys look unseen. It models a backend whose existence check is
// inverted (e.g. a wrong EXISTS / NOT EXISTS query).
type invertedSeen struct {
	*teststore.Store
}

func (s invertedSeen) Seen(ctx context.Context, key string) (bool, error) {
	seen, err := s.Store.Seen(ctx, key)

	return !seen, err
}

// neverExpireMultiplier stretches every TTL far beyond any test window, so
// entries recorded through this wrapper never expire during a suite run.
const neverExpireMultiplier = 1000

// neverExpires ignores TTL expiry entirely: nothing it records ever expires.
// It models a backend that stores keys without honoring the requested expiry.
type neverExpires struct {
	*teststore.Store
}

func (s neverExpires) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.Record(ctx, key, ttl*neverExpireMultiplier)
}

func (s neverExpires) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, key, ttl*neverExpireMultiplier)
}

// ttlExtendingRecord extends a live entry's TTL on every Record, violating the
// no-extension rule. It models a backend implemented with plain SET (no NX)
// semantics for Record.
type ttlExtendingRecord struct {
	*teststore.Store
}

func (s ttlExtendingRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	return s.ForceRecord(ctx, key, ttl)
}

// writeOnceRecord ignores re-records of a key it has written before, even
// after the entry expired. It models a backend whose Record is a SET NX that
// never refreshes expired claims.
type writeOnceRecord struct {
	*teststore.Store

	mu      sync.Mutex
	written map[string]struct{}
}

func (s *writeOnceRecord) Record(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.Record(ctx, key, ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.written[key]; ok {
		return nil
	}

	if err := s.Store.Record(ctx, key, ttl); err != nil {
		return err
	}

	if s.written == nil {
		s.written = make(map[string]struct{})
	}

	s.written[key] = struct{}{}

	return nil
}

// rejectingCheckAndRecord rejects every fresh claim with ErrDuplicate while
// preserving TTL validation. It models a backend whose existence check always
// reports the key as present.
type rejectingCheckAndRecord struct {
	*teststore.Store
}

func (s rejectingCheckAndRecord) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.CheckAndRecord(ctx, key, ttl)
	}

	return idempotency.ErrDuplicate
}

// immortalClaims remembers every key it has ever claimed and reports it as a
// duplicate forever, even after the TTL expired. It models a backend that
// checks existence without honoring expiry.
type immortalClaims struct {
	*teststore.Store

	mu      sync.Mutex
	claimed map[string]struct{}
}

func (s *immortalClaims) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Store.CheckAndRecord(ctx, key, ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.claimed[key]; ok {
		return idempotency.ErrDuplicate
	}

	if err := s.Store.CheckAndRecord(ctx, key, ttl); err != nil {
		return err
	}

	s.claimed[key] = struct{}{}

	return nil
}

// doubleWinner allows the first two CheckAndRecord callers to win, then
// reports duplicates. It models an off-by-one claim budget: a broken atomic
// primitive that grants the claim to more than exactly one caller.
type doubleWinner struct {
	*teststore.Store

	mu     sync.Mutex
	winner int
}

func (s *doubleWinner) CheckAndRecord(_ context.Context, _ string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.winner++

	if s.winner <= 2 {
		return nil
	}

	return idempotency.ErrDuplicate
}

// keyCollapsing maps every key onto one internal entry, so recording any key
// marks all keys as seen. It models a backend that loses the key dimension
// (e.g. a fixed or hashed-to-constant storage key).
type keyCollapsing struct {
	*teststore.Store
}

func (s keyCollapsing) Seen(ctx context.Context, _ string) (bool, error) {
	return s.Store.Seen(ctx, "")
}

func (s keyCollapsing) Record(ctx context.Context, _ string, ttl time.Duration) error {
	return s.Store.Record(ctx, "", ttl)
}

func (s keyCollapsing) CheckAndRecord(ctx context.Context, _ string, ttl time.Duration) error {
	return s.Store.CheckAndRecord(ctx, "", ttl)
}

// emptyKeyDropper silently drops empty keys instead of tracking them like any
// other key. It models a backend whose client skips empty values.
type emptyKeyDropper struct {
	*teststore.Store
}

func (s emptyKeyDropper) Record(ctx context.Context, key string, ttl time.Duration) error {
	if key == "" {
		return nil
	}

	return s.Store.Record(ctx, key, ttl)
}
