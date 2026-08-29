package teststore

import (
	"context"
	"time"
)

// ContextAware wraps [Store] with the context checks the plain Store omits:
// every method first reports the context error (context.Canceled or the
// deadline error) for an already-canceled context instead of silently
// proceeding. It exists so the contract package can self-test
// RunTestsContextAware; production backends implement the checks inside their
// own round-trip paths.
type ContextAware struct {
	*Store
}

// NewContextAware returns an empty context-honoring Store.
func NewContextAware() *ContextAware {
	return &ContextAware{Store: New()}
}

// Seen reports the context error when ctx is already canceled, and otherwise
// delegates to [Store.Seen].
func (s *ContextAware) Seen(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // the context error is returned verbatim by design
	}

	return s.Store.Seen(ctx, key)
}

// Record reports the context error when ctx is already canceled, and
// otherwise delegates to [Store.Record].
func (s *ContextAware) Record(ctx context.Context, key string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // the context error is returned verbatim by design
	}

	return s.Store.Record(ctx, key, ttl)
}

// CheckAndRecord reports the context error when ctx is already canceled, and
// otherwise delegates to [Store.CheckAndRecord]. A canceled call never
// records the key.
func (s *ContextAware) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // the context error is returned verbatim by design
	}

	return s.Store.CheckAndRecord(ctx, key, ttl)
}
