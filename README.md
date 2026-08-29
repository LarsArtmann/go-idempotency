# go-idempotency

**Turns at-least-once delivery into at-most-once processing.**

[![CI](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml/badge.svg)](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-idempotency.svg)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-idempotency)](https://goreportcard.com/report/github.com/larsartmann/go-idempotency)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A focused Go library that deduplicates retried operations using idempotency keys, so a retried command never executes twice. Built for CQRS command handling, it works for any at-most-once-processing need.

## Why

Delivery in a CQRS system is at-least-once: a client submits a command, loses the acknowledgement, and retries. Without deduplication, the retried command executes twice, producing duplicate events and side effects.

This library closes that gap. The client attaches a stable key to each logical command; the store records the key before processing and rejects retries whose key has already been recorded.

## How it works

```
1. client → submit (key=K)  →  CheckAndRecord(K)  →  key unseen   →  process  →  ack
2. ack lost; client retries →  CheckAndRecord(K)  →  ErrDuplicate  →  drop the retry
```

The entire correctness guarantee rests on `CheckAndRecord` being **atomic** — the check and the record happen under a single lock, so concurrent callers with the same key are serialized: exactly one wins.

```go
// ✅ Atomic — exactly one concurrent caller wins
err := store.CheckAndRecord(ctx, key, ttl)

// ❌ Racy — a TOCTOU window lets two callers both pass Seen before either records
seen, _ := store.Seen(ctx, key)
if !seen {
    _ = store.Record(ctx, key, ttl)
}
```

Prefer `CheckAndRecord`. Only reach for `Seen` / `Record` separately when you understand the race you are accepting.

## Design philosophy

This is an **SDK, not a batteries-included framework**. The library ships:

- The **`Store` interface** (three methods, well-defined error semantics, atomicity contracts)
- **`MemoryStore`** _(deprecated)_ — in-memory reference implementation for development and testing only; it will be removed in a future major version

It deliberately does **not** ship production backends. There will never be a Redis store, SQL store, or any other concrete backend in this module. Each backend has its own driver, connection-pool semantics, deployment constraints, and operational tradeoffs. Encoding those decisions here would bloat the dependency tree and force choices on you that you should make yourself.

**You implement the `Store` interface** against whatever backend fits your system. The interface is small and the [API docs](https://pkg.go.dev/github.com/larsartmann/go-idempotency) name the atomic primitive each backend should use (Redis `SET NX`, SQL `INSERT ... ON CONFLICT DO NOTHING`, etc.).

For example, a Redis `CheckAndRecord` is a single `SET NX` call:

```go
func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
    ok, err := s.client.SetNX(ctx, "idem:"+key, "1", ttl).Result()
    if err != nil {
        return err
    }
    if !ok {
        return idempotency.ErrDuplicate
    }
    return nil
}
```

See the [package docs](https://pkg.go.dev/github.com/larsartmann/go-idempotency) for the full Redis adapter (all three methods) and use the [contract test suite](contract/) to verify your implementation.

## Quick start

> **Note:** `MemoryStore` is deprecated and intended for development/testing only. The example illustrates the API — in production, substitute your own `Store` implementation.

```bash
go get github.com/larsartmann/go-idempotency
```

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-idempotency"
)

func main() {
	store := idempotency.NewMemoryStore(5 * time.Minute)
	defer store.Close()

	ctx := context.Background()
	cmdKey := "order-123-create"

	// Check-and-record in a single atomic step (preferred over Seen + Record).
	err := store.CheckAndRecord(ctx, cmdKey, 10*time.Minute)
	if errors.Is(err, idempotency.ErrDuplicate) {
		fmt.Println("already processed — dropping retry")
		return
	}
	if errors.Is(err, idempotency.ErrInvalidTTL) {
		fmt.Printf("programmer error: %v\n", err)
		return
	}
	if err != nil {
		fmt.Printf("store failure: %v\n", err)
		return
	}

	// Process the command...
	fmt.Println("processing command")
}
```

## Errors

Errors are classified by [go-error-family](https://github.com/larsartmann/go-error-family), so they map cleanly onto HTTP statuses and retry decisions downstream. Always check with `errors.Is`.

| Error           | Returned by                | Family    | HTTP | Retryable |
| --------------- | -------------------------- | --------- | ---- | --------- |
| `ErrDuplicate`  | `CheckAndRecord`           | Conflict  | 409  | No        |
| `ErrInvalidTTL` | `Record`, `CheckAndRecord` | Rejection | 400  | No        |

`ErrDuplicate` means a prior, still-valid recording exists — return it to the client, do not retry. `ErrInvalidTTL` means the caller passed `ttl <= 0`; a non-positive TTL records an already-past expiry that protects nothing, so the store rejects it loudly instead of silently breaking exactly-once.

## Features

- **`Store` interface** — three-method contract (`Seen`, `Record`, `CheckAndRecord`) with well-defined error semantics and atomicity requirements
- **Contract test suite** — `contract.RunTests` verifies any `Store` implementation against the full invariant set (atomicity, TTL expiry, concurrency safety, error handling)
- **`MemoryStore`** _(deprecated)_ — in-memory reference implementation for development and testing, with TTL-based expiration (background sweep + lazy deletion), configurable sweep interval, and graceful shutdown; will be removed in a future major version
- **Conflict-classified errors** — `ErrDuplicate` is HTTP 409, non-retryable; `ErrInvalidTTL` is HTTP 400
- **Concurrency-safe** — exactly-one-winner verified with 200 goroutines, property-based tests, and fuzz tests

See [FEATURES.md](FEATURES.md) for the full, code-evidenced inventory.

## Implementing your own backend

The `Store` interface is three methods. Each maps to a single round-trip on a typical backend. The critical requirement is that `CheckAndRecord` is **atomic** — use your backend's native check-and-set primitive.

| Method           | What it does                           | Typical backend primitive                                      |
| ---------------- | -------------------------------------- | -------------------------------------------------------------- |
| `Seen`           | Check if key exists and is not expired | `EXISTS` (Redis), `SELECT COUNT(*)` (SQL)                      |
| `Record`         | Store key with TTL (no-op if exists)   | `SET NX` (Redis), `INSERT ... ON CONFLICT DO NOTHING` (SQL)    |
| `CheckAndRecord` | Atomic check-and-set                   | `SET NX EX` (Redis), `INSERT ... ON CONFLICT DO NOTHING` (SQL) |

**Error mapping:** when the key already exists, return `idempotency.ErrDuplicate`. When `ttl <= 0`, return `idempotency.ErrInvalidTTL`. These are sentinel errors from `go-error-family` with stable HTTP status mappings.

**Verify your implementation:** import the contract package and run the test suite:

```go
func TestContract(t *testing.T) {
    t.Parallel()
    contract.RunTests(t, func(t *testing.T) idempotency.Store {
        store := mybackend.NewStore(conn)
        t.Cleanup(func() { store.Close() })
        return store
    })
}
```

See the [package docs](https://pkg.go.dev/github.com/larsartmann/go-idempotency) for a full Redis adapter example (all three methods).

The suite is self-tested: this repository runs `RunTests` against its own internal in-memory Store (`contract/internal/`, test-only) in CI, so the suite is exercised on every commit — it is not untested code you are asked to trust.

**What `RunTests` checks** — the thirteen invariants, each a named subtest:

| Method | Invariant (subtest) | Requirement |
| --- | --- | --- |
| `Seen` | `UnseenKeyReturnsFalse` | An unseen key reports `(false, nil)`. |
| `Seen` | `AfterRecordReturnsTrue` | A recorded, unexpired key reports true. |
| `Seen` | `LazilyDeletesExpired` | After TTL expiry, `Seen` reports false. |
| `Record` | `NoopOnExistingKey` | Re-recording a live key must NOT extend its TTL. |
| `Record` | `ReRecordsAfterExpiry` | An expired key accepts a fresh TTL. |
| `Record` | `RejectsNonPositiveTTL` | `ttl <= 0` returns `ErrInvalidTTL`; nothing is recorded. |
| `CheckAndRecord` | `FirstCallSucceeds` | The first claim returns nil. |
| `CheckAndRecord` | `DuplicateReturnsErrDuplicate` | A second claim inside the TTL returns `ErrDuplicate` (`errors.Is`-compatible). |
| `CheckAndRecord` | `AllowsAfterExpiry` | After expiry the key can be claimed again. |
| `CheckAndRecord` | `RejectsNonPositiveTTL` | `ttl <= 0` returns `ErrInvalidTTL`; nothing is recorded. |
| Concurrency | `AtomicUnderConcurrency` | 200 goroutines racing one key: exactly one nil win, all others `ErrDuplicate`, no other errors. |
| Cross-cutting | `KeysAreIndependent` | Operations on one key never affect another. |
| Cross-cutting | `EmptyKey` | The empty string is a valid key across all methods. |

**If your backend honors context cancellation** (any network round-trip should), test it separately with the pattern documented in the [contract package docs](https://pkg.go.dev/github.com/larsartmann/go-idempotency/contract#hdr-Testing_context_cancellation): a canceled call must return the context error and must NOT consume the claim, so the retry after a timeout can still be processed.

## Status & roadmap

`MemoryStore` (single-process, in-memory) is **deprecated**. It remains functional and concurrency-tested but is intended for development and testing only. For production, implement the `Store` interface against your persistence backend and validate with `contract.RunTests` — the [migration guide](docs/migrating-from-memorystore.md) walks the full path with a worked example. `MemoryStore` will be removed in a future major version.

This library will **not** add production backends (Redis, SQL, etc.). That is by design. The `Store` interface and the `contract` test suite let you implement and verify your own backend. The [middleware package](TODO_LIST.md) (CommandIdempotency, EventIdempotency, QueryIdempotency) is the next planned addition to this module.

Versioning: **v0.x** — the error sentinels are stable, but `MemoryStore` is deprecated and the `Store` interface may gain methods (`Delete`, `Stats`) before **v1.0**. See [ROADMAP.md](ROADMAP.md).

## Documentation

- [API reference (pkg.go.dev)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
- [Recipe: dedup + response replay](https://pkg.go.dev/github.com/larsartmann/go-idempotency#hdr-Recipe-Dedup_Response_Replay_HTTP_Idempotency) — HTTP idempotency: atomic claim + replaying the original response to retriers
- [Migrating from MemoryStore](docs/migrating-from-memorystore.md) — pick a backend, implement it, validate with the contract suite, swap the type
- [Features](FEATURES.md) — honest feature inventory with code evidence
- [Domain language](docs/DOMAIN_LANGUAGE.md) — glossary of idempotency terms
- [ADR-001: Why no backends](docs/adr/001-no-backends.md) — architecture decision record
- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)

## Development

```bash
go test ./... -race    # tests with race detector (mandatory)
go vet ./...           # static analysis
golangci-lint run ./... # lint (60+ linters, see .golangci.yml)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full setup, testing strategy, and conventions.

## License

[MIT](LICENSE)
