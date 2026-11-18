package idempotency_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// jumpClock is a manually set clock for fuzzing non-monotonic behavior.
type jumpClock struct {
	now time.Time
}

func (c *jumpClock) Now() time.Time { return c.now }

func (c *jumpClock) set(t time.Time) { c.now = t }

// FuzzClockJumps drives a store whose clock jumps anywhere — backward and
// forward, by nanoseconds or by millennia — and pins the invariant the
// WithClock documentation promises: window semantics stay exact and
// atomicity is never affected. After every jump, CheckAndRecord must return
// ErrDuplicate exactly when the clock stands before the live claim's expiry,
// and must succeed exactly when it stands at or past it. A backward jump
// extends a claim's observed window; a forward jump can expire it early —
// but the decision itself is always expiry vs clock, never garbage.
func FuzzClockJumps(f *testing.F) {
	base := time.Unix(0, 0)

	f.Add(int64(60), int64(30), int64(-5))
	f.Add(int64(3600), int64(-3600), int64(3600))
	f.Add(int64(1), int64(0), int64(0))
	f.Add(int64(60), int64(math.MaxInt64/2), int64(math.MinInt64/2))
	f.Add(int64(1), 9e18/int64(time.Second), -(9e18 / int64(time.Second)))

	f.Fuzz(func(t *testing.T, ttlSeconds, jump1Seconds, jump2Seconds int64) {
		if ttlSeconds <= 0 || ttlSeconds > 366*24*3600 {
			return // invalid TTL is ErrInvalidTTL territory (covered by FuzzCheckAndRecord); year-scale bounds the arithmetic
		}

		clock := &jumpClock{now: base}

		store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
		defer store.Close()

		ctx := context.Background()
		ttl := time.Duration(ttlSeconds) * time.Second

		if err := store.CheckAndRecord(ctx, "k", ttl); err != nil {
			t.Fatalf("first claim: %v", err)
		}

		expiry := base.Add(ttl)

		for i, jump := range []time.Duration{
			time.Duration(jump1Seconds) * time.Second,
			time.Duration(jump2Seconds) * time.Second,
		} {
			clock.set(base.Add(jump))

			now := clock.Now()

			if overflowedClock(now, base, jump) {
				continue // int64 overflow in the jump arithmetic; not a store property
			}

			err := store.CheckAndRecord(ctx, "k", ttl)

			if now.Before(expiry) {
				if !errors.Is(err, idempotency.ErrDuplicate) {
					t.Fatalf("jump %d: clock %v < expiry %v: want ErrDuplicate, got %v", i, now, expiry, err)
				}

				continue
			}

			if err != nil {
				t.Fatalf("jump %d: clock %v >= expiry %v: want reclaim, got %v", i, now, expiry, err)
			}

			expiry = now.Add(ttl) // the reclaim installed a fresh claim
		}
	})
}

func overflowedClock(now, base time.Time, jump time.Duration) bool {
	return jump != 0 && now.Equal(base)
}
