package middleware_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/internal/teststore"
	"github.com/larsartmann/go-idempotency/middleware"
)

// FuzzDispatch fuzzes middleware.NewCommand with arbitrary keys and TTLs. It
// pins the dispatch contract: a non-positive TTL surfaces ErrInvalidTTL
// without executing, the first valid dispatch executes exactly once, a
// redelivery of the same key is rejected with ErrDuplicate without executing,
// and a concurrent burst on a fresh store executes the side effect exactly
// once.
func FuzzDispatch(f *testing.F) {
	f.Add("order-42:place", int64(60))
	f.Add("", int64(0))
	f.Add("🔑🚀-emoji-ключ-中文", int64(-1)) //nolint:gosmopolitan // unicode key is the point of the seed
	f.Add(strings.Repeat("D", 4096), int64(30))
	f.Add("max-ttl", int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, key string, ttlSeconds int64) {
		ttl := time.Duration(ttlSeconds) * time.Second
		ctx := context.Background()

		store := teststore.New()
		t.Cleanup(store.Close)

		var executed atomic.Int32

		command := middleware.NewCommand(store, ttl, func(_ context.Context) error {
			executed.Add(1)

			return nil
		})

		if err := command(ctx, key); err != nil {
			if ttl <= 0 && errors.Is(err, idempotency.ErrInvalidTTL) {
				if executed.Load() != 0 {
					t.Fatal("invalid TTL must not execute")
				}

				return
			}

			t.Fatalf("first dispatch: %v", err)
		}

		if ttl <= 0 {
			t.Fatal("non-positive TTL: want ErrInvalidTTL, got nil")
		}

		if executed.Load() != 1 {
			t.Fatalf("executed = %d, want 1 after first dispatch", executed.Load())
		}

		if err := command(ctx, key); !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("redelivery: want ErrDuplicate, got %v", err)
		}

		if executed.Load() != 1 {
			t.Fatalf("executed = %d, want 1 after redelivery", executed.Load())
		}

		// Concurrent burst on a fresh store: exactly one dispatch may claim
		// the key and execute; every other dispatch sees ErrDuplicate.
		burstStore := teststore.New()
		t.Cleanup(burstStore.Close)

		var burstExecuted atomic.Int32

		burst := middleware.NewCommand(burstStore, ttl, func(_ context.Context) error {
			burstExecuted.Add(1)

			return nil
		})

		const goroutines = 8

		var wg sync.WaitGroup

		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()

				switch err := burst(ctx, key); {
				case err == nil, errors.Is(err, idempotency.ErrDuplicate):
				default:
					t.Errorf("burst dispatch: unexpected error %v", err)
				}
			}()
		}

		wg.Wait()

		if burstExecuted.Load() != 1 {
			t.Fatalf("burst executed = %d, want exactly 1", burstExecuted.Load())
		}
	})
}
