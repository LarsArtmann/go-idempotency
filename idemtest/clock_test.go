package idemtest_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
	"github.com/larsartmann/go-idempotency/middleware"
)

// clockedStore is a minimal Store honoring a clock, existing only to prove
// RequireAllowsAfterExpiryPinned end-to-end in this test package. The
// shipped blessed double (NewStore) stays wall-clock on purpose: the contract
// suite must verify real timing, not a clock tests can move.
type clockedStore struct {
	mu     sync.Mutex
	claims map[string]time.Time
	clock  *idemtest.PinnedClock
}

func newClockedStore(clock *idemtest.PinnedClock) *clockedStore {
	return &clockedStore{ //nolint:exhaustruct // mu is zero-valued on purpose
		claims: map[string]time.Time{},
		clock:  clock,
	}
}

func (s *clockedStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.claims[key]

	return ok && s.clock.Now().Before(exp), nil
}

func (s *clockedStore) Record(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.claims[key] = s.clock.Now().Add(ttl)

	return nil
}

func (s *clockedStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.claims[key]; ok && s.clock.Now().Before(exp) {
		return idempotency.ErrDuplicate
	}

	s.claims[key] = s.clock.Now().Add(ttl)

	return nil
}

func TestRequireAllowsAfterExpiryPinned(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	store := newClockedStore(clock)
	exec := idemtest.NewExecutions()
	place := middleware.NewCommand(store, time.Minute, exec.Fn())

	idemtest.RequireAllowsAfterExpiryPinned(t, place, "order-42:place", time.Minute, exec, clock)
}

func TestPinnedClockStartsAtInstantAndAdvancesForward(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(42, 0))

	if got := clock.Now(); !got.Equal(time.Unix(42, 0)) {
		t.Fatalf("initial Now = %v, want %v", got, time.Unix(42, 0))
	}

	clock.Advance(5 * time.Second)

	if got := clock.Now(); !got.Equal(time.Unix(47, 0)) {
		t.Fatalf("Now after advance = %v, want %v", got, time.Unix(47, 0))
	}
}

func TestPinnedClockIgnoresBackwardAdvance(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(42, 0))
	clock.Advance(-5 * time.Second)

	if got := clock.Now(); !got.Equal(time.Unix(42, 0)) {
		t.Fatalf("Now after backward advance = %v, want %v (clock is forward-moving only)", got, time.Unix(42, 0))
	}
}

func TestPinnedClockIsConcurrencySafe(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()

		for range 100 {
			clock.Advance(time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()

		for range 100 {
			_ = clock.Now()
		}
	}()

	wg.Wait()

	if got := clock.Now(); got.Before(time.Unix(0, 50_000_000)) {
		t.Fatalf("final Now = %v, want at least 50ms past the start", got)
	}
}

func TestClockedStoreHonorsWindow(t *testing.T) {
	t.Parallel()

	clock := idemtest.NewPinnedClock(time.Unix(0, 0))
	store := newClockedStore(clock)
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "k", time.Minute); err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if err := store.CheckAndRecord(ctx, "k", time.Minute); !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("in-window claim: want ErrDuplicate, got %v", err)
	}

	clock.Advance(time.Minute)

	if err := store.CheckAndRecord(ctx, "k", time.Minute); err != nil {
		t.Fatalf("post-expiry claim: %v", err)
	}
}
