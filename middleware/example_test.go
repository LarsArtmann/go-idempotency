package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/middleware"
)

// exampleStore is the smallest correct [idempotency.Store]: one mutex, one
// map of expiries. Swap it for Redis, SQL, or any backend that survives
// restarts in production.
type exampleStore struct {
	mu      sync.Mutex
	expires map[string]time.Time
}

func newExampleStore() *exampleStore {
	return &exampleStore{expires: make(map[string]time.Time)}
}

func (s *exampleStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.expires[key]

	return ok && time.Now().Before(exp), nil
}

func (s *exampleStore) Record(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.expires[key]; ok && time.Now().Before(exp) {
		return nil
	}

	s.expires[key] = time.Now().Add(ttl)

	return nil
}

func (s *exampleStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.expires[key]; ok && time.Now().Before(exp) {
		return idempotency.ErrDuplicate
	}

	s.expires[key] = time.Now().Add(ttl)

	return nil
}

func (s *exampleStore) Close() {}

// ExampleNewCommand shows at-most-once command dispatch: the first delivery
// claims the key and executes, the retry with the same idempotency key is
// rejected with ErrDuplicate and never touches the side effect.
func ExampleNewCommand() {
	store := newExampleStore()
	defer store.Close()

	var processed []string

	place := middleware.NewCommand(store, time.Minute, func(_ context.Context) error {
		processed = append(processed, "order-42")

		return nil
	})

	ctx := context.Background()

	first := place(ctx, "order-42:place")
	retry := place(ctx, "order-42:place")

	fmt.Println("first dispatch executed:", first == nil && len(processed) == 1)
	fmt.Println("retry rejected as duplicate:", errors.Is(retry, idempotency.ErrDuplicate))
	fmt.Println("times processed:", len(processed))

	// Output:
	// first dispatch executed: true
	// retry rejected as duplicate: true
	// times processed: 1
}
