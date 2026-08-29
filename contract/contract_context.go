package contract

import (
	"context"
	"errors"
	"testing"
	"time"
)

// contextSuiteTTL outlives every test in the cancellation suite: these tests
// pin error and claim semantics, not expiry.
const contextSuiteTTL = time.Minute

// # Cancellation suite design notes
//
// RunTestsContextAware exists as a separate entry point rather than an
// [Options] flag on [RunTests] for three reasons:
//
//  1. The main suite must stay meaningful for context-blind stores.
//     MemoryStore (github.com/larsartmann/go-idempotency) ignores context by
//     documented design, and any local (in-process) backend legitimately does
//     the same. A cancellation check bolted into the shared suite would force
//     every such backend to fail tests that describe semantics it never
//     promised.
//
//  2. Opt-in is the honest failure mode. A backend that honors cancellation
//     promises something extra; a separate suite names that promise and is
//     only run by implementations that make it. A boolean flag would blur the
//     line between "the invariants every Store has" and "the invariants this
//     Store added".
//
//  3. The surface stays frozen. [RunTests], [RunTestsStrict], and this suite
//     share [StoreFactory]; no Options growth, no new factory shape.
//
// The suite pins exactly the two invariants callers rely on when their
// requests time out:
//
//   - A canceled call returns the context error, not nil and not the
//     duplicate sentinel (idempotency.ErrDuplicate). Callers branch on the
//     context error to decide whether to retry; any other outcome lies to
//     them.
//
//   - A canceled call does NOT consume the claim. The key remains
//     unrecorded, so the retry that arrives after the cancellation can still
//     be processed. A store that records on a canceled call turns every
//     timeout into a poisoned key that rejects legitimate retries until TTL
//     expiry: the worst possible behavior for the at-least-once delivery
//     this library exists to support.
//
// The cancellation case is deliberate: callers cancel when a client
// disconnects or a deadline fires, which is precisely when the request did
// not complete and must not have left a claim behind. Cancel the context
// before the call; the assertion target is the pre-call check (ctx.Err()),
// which every context-honoring backend performs before (or as part of) its
// round-trip.

// RunTestsContextAware runs the optional cancellation-semantics suite against
// an idempotency.Store produced by factory. Run it in addition to
// [RunTests], never instead of it, for backends that honor context
// cancellation, which should be every backend doing network round-trips
// (Redis, SQL, ...).
//
// A store that ignores context fails this suite by design: the suite is
// opt-in exactly because MemoryStore and other local stores do not make the
// cancellation promise. See the design notes above for why this is a
// separate entry point rather than an [Options] flag.
func RunTestsContextAware(t *testing.T, factory StoreFactory) {
	t.Helper()

	t.Run("Seen", func(t *testing.T) {
		t.Run("CanceledReturnsContextError", func(t *testing.T) {
			t.Parallel()

			runCanceledReturnsContextError(t, func(ctx context.Context) error {
				_, err := factory(t).Seen(ctx, "canceled-seen")

				return err //nolint:wrapcheck // the suite asserts on the store's error verbatim
			})
		})
	})

	t.Run("Record", func(t *testing.T) {
		t.Run("CanceledReturnsContextError", func(t *testing.T) {
			t.Parallel()

			runCanceledReturnsContextError(t, func(ctx context.Context) error {
				return factory(t).Record(ctx, "canceled-record", contextSuiteTTL)
			})
		})

		t.Run("CanceledDoesNotConsumeClaim", func(t *testing.T) {
			t.Parallel()

			store := factory(t)

			runCanceledDoesNotConsumeClaim(t,
				func(ctx context.Context) error {
					return store.Record(ctx, "canceled-record-claim", contextSuiteTTL)
				},
				func() error {
					return store.CheckAndRecord(context.Background(), "canceled-record-claim", contextSuiteTTL)
				})
		})
	})

	t.Run("CheckAndRecord", func(t *testing.T) {
		t.Run("CanceledReturnsContextError", func(t *testing.T) {
			t.Parallel()

			runCanceledReturnsContextError(t, func(ctx context.Context) error {
				return factory(t).CheckAndRecord(ctx, "canceled-car", contextSuiteTTL)
			})
		})

		t.Run("CanceledDoesNotConsumeClaim", func(t *testing.T) {
			t.Parallel()

			store := factory(t)

			runCanceledDoesNotConsumeClaim(t,
				func(ctx context.Context) error {
					return store.CheckAndRecord(ctx, "canceled-car-claim", contextSuiteTTL)
				},
				func() error {
					return store.CheckAndRecord(context.Background(), "canceled-car-claim", contextSuiteTTL)
				})
		})
	})
}

// runCanceledReturnsContextError asserts the error invariant: a call made on
// an already-canceled context returns the context error, so callers can
// branch on it to decide whether to retry.
func runCanceledReturnsContextError(t *testing.T, call func(ctx context.Context) error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := call(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled call: want context.Canceled, got %v", err)
	}
}

// runCanceledDoesNotConsumeClaim asserts the claim invariant: after a canceled
// call, the key is still unrecorded, so the retry that arrives after the
// cancellation can claim it. A store that fails this turns every timeout into
// a poisoned key until TTL expiry.
func runCanceledDoesNotConsumeClaim(
	t *testing.T,
	call func(ctx context.Context) error,
	retry func() error,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := call(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled call: want context.Canceled, got %v", err)
	}

	if err := retry(); err != nil {
		t.Fatalf("canceled call consumed the claim: %v", err)
	}
}
