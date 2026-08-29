# Migrating away from MemoryStore

`MemoryStore` is deprecated: it is intended for development and testing only.
It does not survive restarts (every in-flight client retry re-executes after a
crash), it cannot be shared across instances, and it is unbounded. It will be
removed in a future major version. This guide moves you to your own backend in
four steps:

1. [Pick a backend](#1-pick-a-backend)
2. [Implement the three methods](#2-implement-the-three-methods)
3. [Validate with the contract suite](#3-validate-with-the-contract-suite)
4. [Swap the type](#4-swap-the-type)

## 1. Pick a backend

The library deliberately ships no production backend (see
[ADR-001](adr/001-no-backends.md)). You implement `idempotency.Store` against
whatever storage you already run:

| Your situation | Backend to target | Atomic primitive |
| --- | --- | --- |
| Multiple service instances, already run Redis | Redis | `SET NX EX` |
| Multiple instances, relational stack | PostgreSQL/MySQL | `INSERT ... ON CONFLICT DO NOTHING` / `INSERT IGNORE` |
| AWS-native | DynamoDB | `PutItem` with `attribute_not_exists` condition |
| Single process, non-critical dedup (dev/test, best-effort) | In-process map (like MemoryStore, but yours) | `sync.Mutex` critical section |
| Single process, must survive restart | SQLite/bbolt | transaction with unique key constraint |

Pick the smallest thing that survives the failures you care about. The rule of
thumb: **the store must outlive every process that might claim a key** —
otherwise a restart re-executes whatever commands are retried after it.

## 2. Implement the three methods

The whole interface is three methods. Below, a worked migration: an in-process
store for tests (your replacement for MemoryStore in unit tests) and the Redis
implementation for production. Both are complete.

### Test double: in-process map

```go
// teststore.go
package teststore

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-idempotency"
)

type Store struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func New() *Store {
	return &Store{entries: make(map[string]time.Time)}
}

func (s *Store) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.entries[key]
	if !ok {
		return false, nil
	}
	if time.Now().Before(exp) {
		return true, nil
	}
	delete(s.entries, key) // lazy expiry
	return false, nil
}

func (s *Store) Record(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; !ok || !now.Before(exp) {
		s.entries[key] = now.Add(ttl) // live keys are NOT extended
	}
	return nil
}

func (s *Store) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if exp, ok := s.entries[key]; ok && now.Before(exp) {
		return idempotency.ErrDuplicate
	}
	s.entries[key] = now.Add(ttl)
	return nil
}

func (s *Store) Close() {}
```

### Production: Redis

```go
// redistore.go
package redistore

import (
	"context"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/redis/go-redis/v9" // your dependency, not this module's
)

type Store struct {
	client *redis.Client
	prefix string // e.g. "idem:"
}

func New(client *redis.Client) *Store {
	return &Store{client: client, prefix: "idem:"}
}

func (s *Store) Seen(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, s.prefix+key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) Record(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}
	// NX leaves an existing live key alone: Record never extends a TTL.
	return s.client.SetNX(ctx, s.prefix+key, "1", ttl).Err()
}

func (s *Store) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}
	ok, err := s.client.SetNX(ctx, s.prefix+key, "1", ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return idempotency.ErrDuplicate
	}
	return nil
}

func (s *Store) Close() error { return s.client.Close() }
```

The one rule that matters: **`CheckAndRecord` must be a single atomic
operation.** A separate `Seen` call followed by `Record` re-introduces the
TOCTOU race where two concurrent first-attempts both win. Use your backend's
native check-and-set (`SET NX`, `INSERT ... ON CONFLICT DO NOTHING`,
conditional `PutItem`), not two round-trips.

## 3. Validate with the contract suite

Do not trust your implementation — prove it. The
[`contract`](https://pkg.go.dev/github.com/larsartmann/go-idempotency/contract)
package runs the full invariant set (atomicity, TTL expiry, duplicate
detection, 200-goroutine concurrency, error semantics) against any `Store`:

```go
// redistore_contract_test.go
package redistore_test

import (
	"testing"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/contract"
)

func TestContract(t *testing.T) {
	t.Parallel()

	contract.RunTests(t, func(t *testing.T) idempotency.Store {
		t.Helper()

		store := redistore.New(testClient(t)) // dedicated DB index / prefix
		t.Cleanup(func() { _ = store.Close() })

		return store
	})
}
```

Requirements on the factory: it must return an **empty** store (each subtest
starts clean) and register cleanup via `t.Cleanup`. Run it in CI next to your
normal tests — the same binary, no extra tooling. If your backend honors
context cancellation, also cover the
[cancellation pattern](https://pkg.go.dev/github.com/larsartmann/go-idempotency/contract#hdr-Testing_context_cancellation).

## 4. Swap the type

With tests green, replace the type and delete the MemoryStore call sites:

| Before (deprecated) | After |
| --- | --- |
| `idempotency.NewMemoryStore(5 * time.Minute)` | `redistore.New(client)` (or `teststore.New()` in tests) |
| `store.CheckAndRecord(ctx, key, ttl)` | unchanged |
| `errors.Is(err, idempotency.ErrDuplicate)` | unchanged |
| `defer store.Close()` | unchanged |

The method set you call does not change — only the constructor does. This
repository's own lint config (`forbidigo` in `.golangci.yml`) fails any new
`MemoryStore` usage outside `_test.go`; adopt the same rule so the deprecated
type cannot creep back in.

### Migration checklist

- [ ] Backend chosen: survives the failures you care about (restarts, at minimum)
- [ ] `CheckAndRecord` implemented as a single atomic operation
- [ ] Non-positive TTL rejected with `idempotency.ErrInvalidTTL` in `Record` and `CheckAndRecord`
- [ ] `contract.RunTests` green against your implementation, running in CI
- [ ] All `NewMemoryStore` call sites replaced; forbidigo (or review) keeps them out
- [ ] Key namespacing agreed with everything else sharing the backend (`idem:` prefix or table)
