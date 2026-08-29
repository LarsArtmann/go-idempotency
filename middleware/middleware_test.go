package middleware_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
	"github.com/larsartmann/go-idempotency/internal/teststore"
	"github.com/larsartmann/go-idempotency/middleware"
)

// failingStore lets tests drive the store-failure paths deterministically.
type failingStore struct {
	idempotency.Store // embedded: pass-through for the other Store methods

	err error
}

func (s *failingStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if s.err != nil {
		return s.err
	}

	return s.Store.CheckAndRecord(ctx, key, ttl)
}

func (s *failingStore) Close() {}

func newTestStore(t *testing.T) *teststore.Store {
	t.Helper()

	store := teststore.New()
	t.Cleanup(store.Close)

	return store
}

func TestCommand_FirstCallExecutes(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	executed := 0
	command := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		executed++

		return nil
	})

	if err := command(context.Background(), "order-42:place"); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	if executed != 1 {
		t.Fatalf("executed = %d, want 1", executed)
	}
}

func TestCommand_DuplicateDoesNotExecute(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	executed := 0
	command := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		executed++

		return nil
	})

	ctx := context.Background()

	if err := command(ctx, "order-42:place"); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	err := command(ctx, "order-42:place")
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second dispatch: want ErrDuplicate, got %v", err)
	}

	if executed != 1 {
		t.Fatalf("executed = %d, want 1 (duplicate must not execute)", executed)
	}
}

func TestCommand_CommandErrorPropagates(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	boom := errors.New("boom") //nolint:err113 // test sentinel

	command := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		return boom
	})

	if err := command(context.Background(), "key"); !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}
}

func TestCommand_StoreErrorFailsClosed(t *testing.T) {
	t.Parallel()

	store := &failingStore{Store: teststore.New(), err: errors.New("redis down")} //nolint:err113 // test sentinel
	t.Cleanup(store.Close)

	executed := 0
	command := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		executed++

		return nil
	})

	err := command(context.Background(), "key")
	if err == nil || errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("want store failure, got %v", err)
	}

	if executed != 0 {
		t.Fatalf("executed = %d, want 0 (store failure must not execute)", executed)
	}
}

func TestCommand_AtMostOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	var (
		mu       sync.Mutex
		executed int
	)

	command := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		mu.Lock()
		executed++
		mu.Unlock()

		return nil
	})

	const goroutines = 50

	var (
		wg   sync.WaitGroup
		dups int
	)

	wg.Add(goroutines)

	started := make(chan struct{})

	for range goroutines {
		go func() {
			defer wg.Done()

			<-started

			err := command(context.Background(), "contended-key")
			if errors.Is(err, idempotency.ErrDuplicate) {
				mu.Lock()
				dups++
				mu.Unlock()
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	close(started)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if executed != 1 {
		t.Fatalf("executed = %d, want exactly 1 across %d goroutines", executed, goroutines)
	}

	if dups != goroutines-1 {
		t.Fatalf("dups = %d, want %d", dups, goroutines-1)
	}
}

func TestCommand_RejectsInvalidTTL(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	command := middleware.NewCommand(store, 0, func(_ context.Context) error {
		return nil
	})

	err := command(context.Background(), "key")
	if !errors.Is(err, idempotency.ErrInvalidTTL) {
		t.Fatalf("want ErrInvalidTTL, got %v", err)
	}
}

func TestCommandContract(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		return newTestStore(t)
	})
}
