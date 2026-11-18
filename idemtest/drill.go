package idemtest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Dispatch is the shape of a keyed, at-most-once entry point: a
// middleware.Command satisfies it without conversion, and so does any closure
// over [idempotency.Store.CheckAndRecord] or a message-consumer handler. The
// drills exercise whatever wiring the caller is holding, never a rebuilt one.
//
// It is deliberately a type alias: two named function types are not mutually
// assignable in Go, and the drills must accept middleware.Command values as-is.
type Dispatch = func(ctx context.Context, key string) error

// The expiry drill polls instead of sleeping a fixed margin, so a slow
// scheduler widens the wait instead of flaking (the shape the 2026-11-17
// consumer audit recommended for timing-based tests). Pass a small TTL.
const (
	drillPollStep  = 2 * time.Millisecond
	drillPollGrace = 100 * time.Millisecond
)

// RequireExactlyOnce drives a retry storm: attempts sequential dispatches of
// key, the shape of a client that lost the acknowledgement and retried. The
// drill fails unless exactly one attempt executes, every other attempt is
// rejected with [idempotency.ErrDuplicate], and no attempt fails for another
// reason.
func RequireExactlyOnce(tb testing.TB, dispatch Dispatch, key string, attempts int, exec *Executions) {
	tb.Helper()

	if attempts < 1 {
		tb.Fatalf("attempts = %d, want at least 1", attempts)

		return
	}

	ctx := context.Background()

	duplicates := 0

	for attempt := 1; attempt <= attempts; attempt++ {
		err := dispatch(ctx, key)
		switch {
		case err == nil:
			continue
		case errors.Is(err, idempotency.ErrDuplicate):
			duplicates++
		default:
			tb.Fatalf("attempt %d/%d: unexpected error: %v", attempt, attempts, err)

			return
		}
	}

	if duplicates != attempts-1 || exec.Total() != 1 {
		tb.Fatalf(
			"retry storm of %d attempts: duplicates = %d (want %d), executions = %d (want 1) — the side effect ran more than once",
			attempts,
			duplicates,
			attempts-1,
			exec.Total(),
		)
	}
}

// RequireOneWinner releases goroutines concurrent dispatches of key at once,
// the shape of retries racing the first delivery. The drill fails unless
// exactly one goroutine executes, every other goroutine observes
// [idempotency.ErrDuplicate], and no dispatch fails for another reason.
//
// This is the TOCTOU detector: a pipeline built on separate Seen + Record
// calls leaves a window where two goroutines both pass the check, and fails
// here loudly.
func RequireOneWinner(tb testing.TB, dispatch Dispatch, key string, goroutines int, exec *Executions) {
	tb.Helper()

	if goroutines < 1 {
		tb.Fatalf("goroutines = %d, want at least 1", goroutines)

		return
	}

	var (
		duplicates atomic.Int64
		failMu     sync.Mutex
		failures   []error
		wg         sync.WaitGroup
	)

	started := make(chan struct{})
	ctx := context.Background()

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			<-started

			err := dispatch(ctx, key)
			switch {
			case err == nil:
			case errors.Is(err, idempotency.ErrDuplicate):
				duplicates.Add(1)
			default:
				appendFailure(&failMu, &failures, err)
			}
		}()
	}

	close(started)
	wg.Wait()

	failMu.Lock()
	defer failMu.Unlock()

	if duplicates.Load() != int64(goroutines-1) || exec.Total() != 1 || len(failures) != 0 {
		var firstUnexpected error
		if len(failures) > 0 {
			firstUnexpected = failures[0]
		}

		tb.Fatalf(
			"race of %d concurrent dispatches: duplicates = %d (want %d), executions = %d (want 1), unexpected errors = %d (first: %v) — more than one dispatch executed",
			goroutines,
			duplicates.Load(),
			goroutines-1,
			exec.Total(),
			len(failures),
			firstUnexpected,
		)
	}
}

// RequireFailClosed asserts the fail-closed contract: while the store fails,
// the dispatch must return an error that is neither [idempotency.ErrDuplicate]
// nor [idempotency.ErrInvalidTTL], and the side effect must not run. A store
// failure means the duplicate state is unknown — executing anyway is the one
// unrecoverable outcome. Pair with [Fail] to drive the failure.
func RequireFailClosed(tb testing.TB, dispatch Dispatch, key string, exec *Executions) {
	tb.Helper()

	err := dispatch(context.Background(), key)
	switch {
	case err == nil:
		tb.Fatalf("dispatch succeeded while the store fails: the pipeline executed on unknown duplicate state")

		return
	case errors.Is(err, idempotency.ErrDuplicate), errors.Is(err, idempotency.ErrInvalidTTL):
		tb.Fatalf("store failure surfaced as %v: the pipeline misreported an unknown state as a sentinel outcome", err)

		return
	}

	if exec.Total() != 0 {
		tb.Fatalf("side effects executed = %d, want 0: the pipeline executed while the store failed", exec.Total())
	}
}

// RequireAllowsAfterExpiry asserts the TTL window from the caller's side: the
// first dispatch executes; a retry inside the window is rejected as a
// duplicate; once ttl has elapsed, a later attempt executes again — exactly
// two executions in total. The in-window rejection is the assertion that
// makes the drill honest: a pipeline that frees the claim instantly (a
// too-short claim, or no claim at all) would pass a reclaim-only drill and
// fails here. The reclaim is polled with a bounded deadline ([drillPollGrace]
// past the TTL) instead of sleeping a fixed margin, so slow schedulers widen
// the wait instead of flaking. Pass a small TTL (single-digit milliseconds)
// to keep the drill fast; for a variant with no wall-clock wait at all, see
// [RequireAllowsAfterExpiryPinned].
func RequireAllowsAfterExpiry(tb testing.TB, dispatch Dispatch, key string, ttl time.Duration, exec *Executions) {
	tb.Helper()

	if ttl <= 0 {
		tb.Fatalf("ttl = %v, want a positive duration", ttl)

		return
	}

	ctx := context.Background()

	if err := dispatch(ctx, key); err != nil {
		tb.Fatalf("first dispatch: %v", err)

		return
	}

	RequireExecutions(tb, exec, 1)

	if err := dispatch(ctx, key); !errors.Is(err, idempotency.ErrDuplicate) {
		tb.Fatalf("retry inside the %v window: err = %v, want ErrDuplicate — the claim did not hold", ttl, err)

		return
	}

	RequireExecutions(tb, exec, 1)

	deadline := time.Now().Add(ttl + drillPollGrace)

	for {
		err := dispatch(ctx, key)
		if err == nil {
			break // the claim expired and the retry legitimately re-processed
		}

		if !errors.Is(err, idempotency.ErrDuplicate) {
			tb.Fatalf("retry after ttl: unexpected error: %v", err)

			return
		}

		if time.Now().After(deadline) {
			tb.Fatalf(
				"key %q still claimed %v past its %v ttl: the claim outlives its window",
				key, drillPollGrace, ttl,
			)

			return
		}

		time.Sleep(drillPollStep)
	}

	RequireExecutions(tb, exec, 2)
}

func appendFailure(mu *sync.Mutex, failures *[]error, err error) {
	mu.Lock()
	defer mu.Unlock()

	*failures = append(*failures, err)
}

// RequireAllowsAfterExpiryPinned is the deterministic form of
// [RequireAllowsAfterExpiry]: the pipeline's store reads the pinned clock
// (wire it at construction with idempotency.WithClock(clock.Now)) and the
// drill walks the window on that clock — first dispatch executes, an
// in-window retry is rejected as a duplicate, clock.Advance(ttl) expires the
// claim, and the retry executes again. There is no polling and no
// wall-clock wait, so the drill cannot flake on a loaded runner and TTLs of
// any size test at the same speed. The same clock must drive the dispatch's
// store; a dispatch wired to a different clock is a wiring bug the drill
// surfaces as a stuck claim.
func RequireAllowsAfterExpiryPinned(
	tb testing.TB,
	dispatch Dispatch,
	key string,
	ttl time.Duration,
	exec *Executions,
	clock Clock,
) {
	tb.Helper()

	if ttl <= 0 {
		tb.Fatalf("ttl = %v, want a positive duration", ttl)

		return
	}

	if clock == nil {
		tb.Fatalf("clock is nil: pass the pinned clock the pipeline's store reads")

		return
	}

	ctx := context.Background()

	if err := dispatch(ctx, key); err != nil {
		tb.Fatalf("first dispatch: %v", err)

		return
	}

	RequireExecutions(tb, exec, 1)

	if err := dispatch(ctx, key); !errors.Is(err, idempotency.ErrDuplicate) {
		tb.Fatalf("retry inside the %v window: err = %v, want ErrDuplicate — the claim did not hold", ttl, err)

		return
	}

	RequireExecutions(tb, exec, 1)

	clock.Advance(ttl)

	if err := dispatch(ctx, key); err != nil {
		tb.Fatalf("retry after the clock advanced past the %v ttl: %v — the claim outlives its window", ttl, err)

		return
	}

	RequireExecutions(tb, exec, 2)
}
