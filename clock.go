package idempotency

import "time"

// MemoryStoreOption customizes a [MemoryStore] at construction time. Options
// are applied by [NewMemoryStore] before the background sweeper starts, so a
// configured clock governs expiry decisions from the first tick on.
type MemoryStoreOption func(*MemoryStore)

// WithClock overrides the clock [MemoryStore] reads for every expiry decision:
// lazy deletion in [MemoryStore.Seen], re-recording in [MemoryStore.Record],
// claiming in [MemoryStore.CheckAndRecord], and the sweep's expiry comparison.
// The default is time.Now.
//
// The reason this exists is testing: a pinned clock turns the expiry paths —
// which otherwise require a real sleep of ttl plus margin — into deterministic
// clock advances, and makes hour-scale TTLs testable at all. Supply a
// thread-safe clock when the store is used from multiple goroutines: the
// sweep goroutine reads it on every tick.
//
// Treat the injected clock as forward-moving only. A clock that jumps
// backward extends live claims past their intended window; a jump forward can
// expire claims early. Atomicity is never affected — the exactly-one-winner
// guarantee lives in the map operations, not in the clock — but window
// sizing distorts in both directions. The sweep ticker always runs on the
// real wall clock regardless of what this clock does.
//
// A nil clock is ignored and the wall-clock default stands.
func WithClock(now func() time.Time) MemoryStoreOption {
	return func(s *MemoryStore) {
		if now != nil {
			s.now = now
		}
	}
}
