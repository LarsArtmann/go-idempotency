// Package middleware wires an [idempotency.Store] into CQRS dispatch
// pipelines. The core is transport-agnostic; the HTTP adapter uses only
// net/http. Per ADR-002 this package is stdlib-only: it imports nothing
// outside the standard library and this module, so using it adds zero new
// dependencies to a consumer's tree.
//
// EventIdempotency and QueryIdempotency (named in earlier docs) are planned,
// not implemented — they wait for a consumer that needs them.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Command executes a command exactly once per idempotency key for the
// configured TTL. Implementations must be safe for concurrent use.
type Command func(ctx context.Context, key string) error

// NewCommand wraps a command-execution function so the side effect runs at
// most once per key: the first call claims the key via
// [idempotency.Store.CheckAndRecord] and executes; concurrent or later calls
// with the same key are rejected with [idempotency.ErrDuplicate] without
// executing.
//
// The store errors are passed through unchanged so callers can reuse their
// existing [idempotency.ErrDuplicate] handling:
//
//   - nil: the command was claimed and executed.
//   - [idempotency.ErrDuplicate]: a live claim already exists; execute was
//     NOT called. Drop the retry or, for HTTP-style semantics, replay the
//     original response (see the response-replay recipe in the package docs
//     of github.com/larsartmann/go-idempotency).
//   - [idempotency.ErrInvalidTTL]: ttl is not positive — programmer error.
//   - any other error: the store itself failed; the command did not execute.
//     Fail the delivery and let redelivery retry; do not execute on unknown
//     state.
//
// The key is caller-owned. Derive it from the entity and operation
// ("order-42:place"), never from per-request values; namespace it
// ("idem:") when the backend is shared.
func NewCommand(store idempotency.Store, ttl time.Duration, execute func(ctx context.Context) error) Command {
	return func(ctx context.Context, key string) error {
		err := store.CheckAndRecord(ctx, key, ttl)
		switch {
		case err == nil:
			return execute(ctx)
		case errors.Is(err, idempotency.ErrDuplicate), errors.Is(err, idempotency.ErrInvalidTTL):
			return err //nolint:wrapcheck // sentinels pass through unchanged by design
		default:
			// Store failure: the command was not executed. Wrap to mark
			// where it surfaced; errors.Is still reaches the cause.
			return fmt.Errorf("command idempotency check failed (command not executed): %w", err)
		}
	}
}
