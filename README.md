# go-idempotency

**Turns at-least-once delivery into at-most-once processing.**

[![CI](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml/badge.svg)](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/LarsArtmann/go-idempotency/graph/badge.svg)](https://codecov.io/gh/LarsArtmann/go-idempotency)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-idempotency.svg)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-idempotency)](https://goreportcard.com/report/github.com/larsartmann/go-idempotency)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev)
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
- **`MemoryStore`** _(deprecated)_ — in-memory store kept for development and testing only; it will be removed in a future major version

It deliberately does **not** ship production backends. There will never be a Redis store, SQL store, or any other concrete backend in this module. Each backend has its own driver, connection-pool semantics, deployment constraints, and operational tradeoffs. Encoding those decisions here would bloat the dependency tree and force choices on you that you should make yourself.

**You implement the `Store` interface** against whatever backend fits your system. The interface is small and the [API docs](https://pkg.go.dev/github.com/larsartmann/go-idempotency) name the atomic primitive each backend should use (Redis `SET NX EX`, one SQL conditional upsert, etc.).

**On the go-cqrs-lite stack? You do not need to write a backend.** The [`metaengine.DedupStore` engine capability](https://github.com/larsartmann/go-cqrs-lite) (ADR-0142 there) is implemented by every go-cqrs-lite storage engine — SQL engines natively, map-shaped engines via the degraded `MapDedupStore` runtime — with cross-engine semantics pinned by its conformance suite. That repo's own `idempotency/{kvstore,sqlstore}` modules were retired in the capability's favor (ADR-0158 there). Everyone else implements `Store` as described below.

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

## Installation

```bash
go get github.com/larsartmann/go-idempotency
```

Requires Go 1.27 or newer. The only runtime dependency is [go-error-family](https://github.com/larsartmann/go-error-family) (error classification).

## Quick start

> **Note:** `MemoryStore` is deprecated and intended for development/testing only. The example illustrates the API — in production, substitute your own `Store` implementation.

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

**Transient store errors.** Any error that is neither sentinel means the store itself failed (network, timeout, backend down). The correct reaction is to **fail the request without processing** — treating a store failure as "probably not a duplicate" is how duplicate payments happen. For transport-style backends, retry policies belong in your client: `go-error-family` classifies the underlying error, so `errorfamily.IsRetryable(err)` tells you whether a retry has a chance (connection reset: yes; auth failure: no). Retry with backoff only for retryable errors; never retry `ErrDuplicate` or `ErrInvalidTTL`.

## Common pitfalls

- **Never implement `CheckAndRecord` as `Seen` + `Record`.** Two round-trips reintroduce the TOCTOU race: two concurrent first-attempts both see "not seen" and both process. The suite's `Concurrency/AtomicUnderConcurrency` subtest (200 goroutines, exactly one winner) exists to catch exactly this.
- **A store error is not permission to process.** `CheckAndRecord` failing does not mean the command is fresh — it means you don't know. Fail the request (or the delivery); let the redelivery mechanism try again.
- **Size the TTL to the retry window, not the request timeout.** Retries can arrive from client backoff, message-redelivery, and outbox replay — often hours after the first attempt. If the TTL expires first, the retry executes again. When in doubt, longer.
- **Namespace your keys when sharing a backend.** Prefix with `idem:` (or a dedicated table/hash) so idempotency keys cannot collide with other data in the same store, and so operational cleanup cannot sweep them by accident.
- **Watch the clock.** Expiry is judged by whoever owns the clock. Redis evaluates TTL server-side (app clock irrelevant); SQL/DynamoDB columns compare against the backend's clock; in-process maps use the instance's clock. With multiple instances writing absolute timestamps, sync matters; server-side TTLs avoid the whole class.
- **A claimed key that never finishes stays claimed until TTL.** If your process crashes between `CheckAndRecord` winning and the effect completing, retries get `ErrDuplicate` until expiry. Plan for it: keep TTLs tunable per key, store a failure marker as the response (see the response-replay recipe), or invalidate the claim manually once your backend exposes a way to.
- **"Isn't idempotency supposed to be forever?"** It depends on the claim's scope. Request-facing claims are TTL-windowed by design: the window must cover the retry horizon (redelivery, outbox replay, client backoff), and a permanent claim on a crashed request would be a permanent lockout for that key. For consumer/event dedupe and permanent domain invariants, pass `idempotency.Forever`: the claim never expires, which is the correct semantic when re-processing a duplicate corrupts state. See [ADR-006](docs/adr/006-forever-permanent-claims.md).

## Features

- **`Store` interface** — three-method contract (`Seen`, `Record`, `CheckAndRecord`) with well-defined error semantics and atomicity requirements
- **Permanent claims** — `idempotency.Forever` records a claim that never expires, for event/consumer dedupe and permanent domain invariants; request-facing keys stay TTL-windowed (ADR-006)
- **Contract test suite** — `contract.RunTests` verifies any `Store` implementation against the full invariant set (atomicity, TTL expiry, concurrency safety, error handling)
- **Consumer test kit** — `idemtest` (ADR-007): a contract-certified test store, failure injection, a store spy, and sabotage-proven drills for retry storms, one-winner races, fail-closed behavior, TTL windows, and the HTTP adapter's 409/400/503 contract
- **`MemoryStore`** _(deprecated)_ — in-memory store for development and testing, with TTL-based expiration (background sweep + lazy deletion), configurable sweep interval, and graceful shutdown; will be removed in a future major version
- **Conflict-classified errors** — `ErrDuplicate` is HTTP 409, non-retryable; `ErrInvalidTTL` is HTTP 400
- **Concurrency-safe** — exactly-one-winner verified with 200 goroutines, property-based tests, and fuzz tests

See [FEATURES.md](FEATURES.md) for the full, code-evidenced inventory.

## Implementing your own backend

The `Store` interface is three methods. Each maps to a single round-trip on a typical backend. The critical requirement is that `CheckAndRecord` is **atomic** — use your backend's native check-and-set primitive.

| Backend        | `CheckAndRecord` primitive                           | TTL semantics                                                                        | Gotchas                                                                                                         |
| -------------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| Redis          | `SET key 1 NX EX <ttl>`                              | Server-side expiry — clock-safe                                                      | Prefix keys (`idem:`); NX returning "not set" means duplicate                                                   |
| PostgreSQL     | `INSERT ... ON CONFLICT DO UPDATE ... WHERE expired` | `expires_at` column; NULL means permanent; live when `expires_at IS NULL OR > now()` | A bare `ON CONFLICT DO NOTHING` only frees the key once the expired row is purged                               |
| MySQL          | `INSERT ... ON DUPLICATE KEY UPDATE ... IF(expired)` | `expires_at` column, as above                                                        | `INSERT IGNORE` swallows unrelated errors and never reclaims an expired key; zero rows affected means duplicate |
| DynamoDB       | `PutItem` with `attribute_not_exists(key)`           | TTL attribute is best-effort (deletion can lag hours) — always check expiry on read  | Conditional writes are extra cost on duplicate retries                                                          |
| SQLite / bbolt | Conditional upsert in a transaction                  | `expires_at` column, as above                                                        | `INSERT OR IGNORE` never reclaims an expired key; single-writer, dead after a crash by definition               |

**If you accept `Forever`, map the sentinel at the boundary and never compute `now.Add(Forever)`.** `time.Time.Add` reaches about year 2318, but `UnixNano` overflows in 2262. Use no expiry at all (Redis `SET key 1` without `EX`), a NULL/SQL-infinity expiry column, or a max sentinel stored as-is. The package docs carry the per-backend detail.

| Method           | What it does                           | Typical backend primitive                   |
| ---------------- | -------------------------------------- | ------------------------------------------- |
| `Seen`           | Check if key exists and is not expired | `EXISTS` (Redis), `SELECT COUNT(*)` (SQL)   |
| `Record`         | Store key with TTL (no-op if exists)   | `SET NX EX` (Redis), conditional SQL upsert |
| `CheckAndRecord` | Atomic check-and-set                   | `SET NX EX` (Redis), conditional SQL upsert |

**Error mapping:** when the key already exists, return `idempotency.ErrDuplicate`. When `ttl <= 0`, return `idempotency.ErrInvalidTTL`. When the TTL is exactly `idempotency.Forever`, store the backend's permanent representation (no expiry or NULL). These are sentinel errors from `go-error-family` with stable HTTP status mappings.

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

The suite is self-tested: this repository runs `RunTests` against its own internal in-memory Store (`internal/teststore/`, test-only) in CI, so the suite is exercised on every commit — it is not untested code you are asked to trust.

**What `RunTests` checks** — the eighteen invariants, each a named subtest:

| Method           | Invariant (subtest)            | Requirement                                                                                     |
| ---------------- | ------------------------------ | ----------------------------------------------------------------------------------------------- |
| `Seen`           | `UnseenKeyReturnsFalse`        | An unseen key reports `(false, nil)`.                                                           |
| `Seen`           | `AfterRecordReturnsTrue`       | A recorded, unexpired key reports true.                                                         |
| `Seen`           | `LazilyDeletesExpired`         | After TTL expiry, `Seen` reports false.                                                         |
| `Record`         | `NoopOnExistingKey`            | Re-recording a live key must NOT extend its TTL.                                                |
| `Record`         | `ReRecordsAfterExpiry`         | An expired key accepts a fresh TTL.                                                             |
| `Record`         | `RejectsNonPositiveTTL`        | `ttl <= 0` returns `ErrInvalidTTL`; nothing is recorded.                                        |
| `CheckAndRecord` | `FirstCallSucceeds`            | The first claim returns nil.                                                                    |
| `CheckAndRecord` | `DuplicateReturnsErrDuplicate` | A second claim inside the TTL returns `ErrDuplicate` (`errors.Is`-compatible).                  |
| `CheckAndRecord` | `AllowsAfterExpiry`            | After expiry the key can be claimed again.                                                      |
| `CheckAndRecord` | `RejectsNonPositiveTTL`        | `ttl <= 0` returns `ErrInvalidTTL`; nothing is recorded.                                        |
| `Forever`        | `SeenSurvives`                 | A claim recorded with `Forever` is still seen after the sleep window.                           |
| `Forever`        | `RecordNeverExpires`           | A `Forever` claim remains a duplicate for `CheckAndRecord` after the sleep window.              |
| `Forever`        | `RecordNoopOnExistingKey`      | A live short-TTL claim is not upgraded to `Forever`.                                            |
| `Forever`        | `RecordNotDowngraded`          | A live `Forever` claim is not shortened by a later short TTL.                                   |
| `Forever`        | `CheckAndRecord`               | The first `Forever` claim returns nil; later claims return `ErrDuplicate`.                      |
| Concurrency      | `AtomicUnderConcurrency`       | 200 goroutines racing one key: exactly one nil win, all others `ErrDuplicate`, no other errors. |
| Cross-cutting    | `KeysAreIndependent`           | Operations on one key never affect another.                                                     |
| Cross-cutting    | `EmptyKey`                     | The empty string is a valid key across all methods.                                             |

Slow or heavily loaded CI runners can stretch the suite's wall-clock timings with `contract.RunTestsStrict` and `contract.Options{TimingScale: 3}` instead of debugging expiry flakes.

## Testing your integration

The contract suite verifies a backend. To test everything **above** it — your command wiring, your HTTP adapter — the [`idemtest`](idemtest/) kit ships the drills and doubles every consumer otherwise hand-rolls ([ADR-007](docs/adr/007-consumer-test-kit.md)). Import it from `_test.go` files only.

A contract-certified store for tests (no deprecated `MemoryStore`, no hand-rolled map):

```go
store := idemtest.NewStore()
```

Prove at-most-once through your own dispatch, in one line each:

```go
exec := idemtest.NewExecutions()
place := middleware.NewCommand(store, time.Minute, exec.Fn())

idemtest.RequireExactlyOnce(t, place, "order-42:place", 5, exec)  // retry storm: 5 attempts, 1 execution
idemtest.RequireOneWinner(t, place, "contended-key", 50, exec)   // concurrent race: exactly one winner
idemtest.RequireFailClosed(t, place, "key", exec)                 // pair with idemtest.Fail(store, err)
idemtest.RequireAllowsAfterExpiry(t, place, "key", 5*time.Millisecond, exec) // the TTL window frees
```

`RequireOneWinner` is the TOCTOU detector: a pipeline built on separate `Seen` + `Record` calls fails it loudly. The drills accept any `func(ctx, key) error` dispatch — a `middleware.Command` passes as-is, and so does a raw `CheckAndRecord` closure.

Drive the HTTP adapter end-to-end (in-process, no sockets):

```go
exec := idemtest.NewExecutions()
handler := middleware.HTTP(store, time.Minute, idemtest.Counting(exec, appHandler))

first := idemtest.Send(t, handler, "order-42:place")
idemtest.RequireStatus(t, first, http.StatusCreated)
idemtest.RequireExecutions(t, exec, 1)

idemtest.RequireDuplicate(t, idemtest.Send(t, handler, "order-42:place")) // 409, still 1 execution
```

`idemtest.Send(t, handler, "")` sends no header (the 400 path); `idemtest.Fail` + `RequireStoreUnavailable` drills the 503 fail-closed path; `idemtest.SendConcurrently` releases N same-key requests at once. `idemtest.NewRecorder` spies on every store call so key-derivation and TTL bugs surface as assertion failures (`idemtest.RequireClaimed`), not silences.

Every drill is sabotage-proven in this repository's CI: each assertion ships with a negative test showing it fail against the violation it names.

**If your backend honors context cancellation** (any network round-trip should), also run the opt-in cancellation suite: `contract.RunTestsContextAware(t, factory)` asserts that a canceled call returns the context error and does NOT consume the claim, so the retry after a timeout can still be processed. Context-blind stores (like the deprecated `MemoryStore`) must not run it: ignoring cancellation is their documented behavior.

## Status & roadmap

`MemoryStore` (single-process, in-memory) is **deprecated**. It remains functional and concurrency-tested but is intended for development and testing only. For production, implement the `Store` interface against your persistence backend and validate with `contract.RunTests` — the [migration guide](docs/migrating-from-memorystore.md) walks the full path with a worked example. `MemoryStore` will be removed in a future major version.

This library will **not** add production backends (Redis, SQL, etc.). That is by design. The `Store` interface and the `contract` test suite let you implement and verify your own backend. The [`middleware` package](https://pkg.go.dev/github.com/larsartmann/go-idempotency/middleware) ships `Command`/`NewCommand` and a `net/http` adapter (stdlib-only per [ADR-002](docs/adr/002-middleware-module-boundary.md)); `EventIdempotency`/`QueryIdempotency` wait for a consumer that needs them.

Versioning: **v0.x** — the error sentinels are stable, but `MemoryStore` is deprecated and the `Store` interface may gain methods (`Delete`, `Stats`) before **v1.0**. See [ROADMAP.md](ROADMAP.md).

## Documentation

- [API reference (pkg.go.dev)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
- [Recipe: dedup + response replay](https://pkg.go.dev/github.com/larsartmann/go-idempotency#hdr-Recipe__Dedup___Response_Replay__HTTP_Idempotency_) — HTTP idempotency: atomic claim + replaying the original response to retriers
- [Client-side vs receiver-side posture](https://pkg.go.dev/github.com/larsartmann/go-idempotency#hdr-Client_side_vs_receiver_side_posture) — when the API you call already dedupes server-side and replays the resource (Wise, Stripe), the replay recipe is the only correct client-side shape; key-only guards fit the receiving end of a request
- [Migrating from MemoryStore](docs/migrating-from-memorystore.md) — pick a backend, implement it, validate with the contract suite, swap the type
- [Test kit (`idemtest`)](idemtest/) — drills and doubles for testing your integration: retry storms, one-winner races, fail-closed, HTTP 409/400/503 (pkg.go.dev page follows the next release tag)
- [Runnable example](example/main.go) — end-to-end walkthrough (`go run ./example`)
- [Features](FEATURES.md) — honest feature inventory with code evidence
- [Domain language](docs/DOMAIN_LANGUAGE.md) — glossary of idempotency terms
- [ADR index](docs/adr/README.md) — all architecture decision records in one table
- [Security policy](SECURITY.md) — how to report vulnerabilities
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)

## Development

```bash
go test ./... -race    # tests with race detector (mandatory)
go vet ./...           # static analysis
golangci-lint run ./... # lint (see .golangci.yml for the enabled linters)
./scripts/check-stale-refs.sh # fail on doc phrases known to go stale (also a CI job)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full setup, testing strategy, and conventions.

## License

[MIT](LICENSE)
