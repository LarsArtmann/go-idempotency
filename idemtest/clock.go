package idemtest

import (
	"sync"
	"time"
)

// Clock is the minimal control surface the pinned-clock drill drives. Wire a
// pipeline's store to read [Clock.Now] — pass clock.Now to
// [idempotency.WithClock] at construction — and hand the same clock to
// [RequireAllowsAfterExpiryPinned]; the drill advances it to move time past
// the TTL deterministically, with no wall-clock wait at all.
type Clock interface {
	Now() time.Time
	Advance(d time.Duration)
}

// PinnedClock is the blessed [Clock] for consumer tests: a thread-safe clock
// that starts at a fixed instant and moves only when [PinnedClock.Advance] is
// called. A store wired with clock.Now (via [idempotency.WithClock]) makes
// every expiry decision a deterministic state transition.
type PinnedClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewPinnedClock returns a [PinnedClock] standing at now.
func NewPinnedClock(now time.Time) *PinnedClock {
	return &PinnedClock{now: now}
}

// Now reports the clock's current instant. Safe for concurrent use: the sweep
// goroutine of a store wired to this clock reads it on every tick.
func (c *PinnedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

// Advance moves the clock forward by d. Backward moves are deliberately not
// supported: the [WithClock] contract treats the clock as forward-moving only.
func (c *PinnedClock) Advance(d time.Duration) {
	if d < 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}
