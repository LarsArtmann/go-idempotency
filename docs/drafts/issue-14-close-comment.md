Shipped in v0.6.0. `idempotency.WithClock(func() time.Time)` overrides the clock behind all four expiry sites (`clock.go:29`, default stays `time.Now`), options apply before the sweeper starts. The suite's expiry sleeps are gone — clock advances instead, sweep tests run in `testing/synctest` bubbles, and `bench_test.go` got the pinned clock too. `idemtest/v0.1.1` adds `RequireAllowsAfterExpiryPinned` + `PinnedClock` so consumers drill the window with zero wall-clock wait.

Gate: 12/12 green, `-race`, 4×3m fuzz. Closing as completed.
