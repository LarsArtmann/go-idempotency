package idempotency_test

// fakeClock is a thread-safe, manually advanced clock for pinning
// MemoryStore's expiry decisions via idempotency.WithClock. Advance replaces
// the real sleeps the expiry-path tests used to pay for; Calls counts how
// many times the store has read the clock, which lets the sweep tests wait
// for a proven background tick instead of sleeping past one.

import (
	"sync"
	"time"
)

type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	calls int
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

// Now is the func() time.Time the store invokes on every expiry decision.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++

	return c.now
}

// Advance moves the clock forward by d, simulating the passage of
// TTL-sized time without sleeping.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// Calls reports how many times the store has read the clock so far.
func (c *fakeClock) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

// awaitCallAfter polls until the store reads the clock again beyond baseline
// — for the sweep tests: the background ticker fired after the clock was
// advanced — or the timeout elapses. The wait uses real time because the
// sweep ticker is deliberately wall-clock, but it is bounded, never a bare
// sleep of a guessed duration.
func (c *fakeClock) awaitCallAfter(baseline int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if c.Calls() > baseline {
			return true
		}

		time.Sleep(time.Millisecond)
	}

	return false
}
