# Features

Honest inventory of what exists, what ships with gaps, and what is planned. Every status is verified against code.

## FULLY_FUNCTIONAL

| Feature                                                                                                                                                                         | Evidence                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| **Store interface** — 3-method abstraction (`Seen`, `Record`, `CheckAndRecord`) for idempotency key tracking                                                                    | `Store` interface, `store.go`                                             |
| **MemoryStore** — in-memory `Store` reference implementation backed by `map[string]time.Time` *(deprecated — see DEPRECATED below)*                                | `MemoryStore` struct, `store.go`                                          |
| **Atomic CheckAndRecord** — single-lock check-and-set that prevents the TOCTOU race a separate Seen+Record pair would create                                                    | `MemoryStore.CheckAndRecord`, tested `TestMemoryStore_CheckAndRecord_*` |
| **TTL-based expiration (dual mechanism)** — background sweep goroutine + lazy deletion on read; map cannot grow unboundedly even with sweep disabled                            | `MemoryStore.sweep`, `MemoryStore.Seen`, `MemoryStore.Record`           |
| **Configurable sweep interval** — `sweepInterval == 0` disables background goroutine; lazy deletion still bounds growth                                                         | `NewMemoryStore` constructor                                             |
| **Idempotent Record** — calling `Record` on an existing, non-expired key is a no-op (TTL is not extended). Expired keys are re-recorded with the fresh TTL.                     | `MemoryStore.Record`, tested `store_test.go`                             |
| **ErrDuplicate as Conflict** — sentinel error classified as `Conflict` (HTTP 409, non-retryable) via `go-error-family`                                                          | `ErrDuplicate` var, tested `store_test.go`                               |
| **TTL validation** — `Record` and `CheckAndRecord` reject non-positive TTL with `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable); prevents silent exactly-once breakage | `ErrInvalidTTL` var, tested `store_test.go`                              |
| **Concurrency safety** — `sync.RWMutex` protection; exactly-one-winner tested with 200 goroutines + randomized 2–20 goroutine property test                                     | `store_test.go`, `property_test.go`                                      |
| **Graceful shutdown** — `Close()` stops sweep goroutine; idempotent via `sync.Once`. Operations still function after Close.                                                     | `MemoryStore.Close`, tested `store_test.go`                              |
| **Property-based testing** — `pgregory.net/rapid` property tests for idempotency, exact-once concurrency, key independence, TTL expiry                                          | `property_test.go`                                                        |
| **Sweep under load** — 1000-key concurrent soak test verifying sweep reclaims all expired entries                                                                               | `store_test.go`                                                           |
| **Benchmarks** — serial, contended, and parallel benchmarks for `CheckAndRecord`, `Seen`, `Record`                                                                              | `bench_test.go`                                                           |
| **Edge case coverage** — empty key, non-positive TTL rejection, post-Close operations all tested                                                                                | `store_test.go`                                                           |
| **Contract test suite** — reusable test harness that verifies any `Store` implementation against the full invariant set; consumers run it against their own backend             | `contract.RunTests`, `contract/contract.go`                              |
| **Lint configuration** — `.golangci.yml` enables `exhaustruct`, `gosec`, `revive`, `misspell`, `gocritic`                                                                       | `.golangci.yml`                                                           |
| **CI pipeline** — GitHub Actions runs `go test -race`, `go vet`, `golangci-lint` on every push and PR                                                                           | `.github/workflows/ci.yml`                                               |

## DEPRECATED

These remain functional and tested, but are scheduled for removal in a future major version. Do not build new production code on them.

| Feature | Status | Replacement | Evidence |
| ------- | ------ | ----------- | -------- |
| **MemoryStore** — in-memory `Store` with TTL expiration, background sweep, lazy deletion, graceful shutdown | Deprecated (v0.2.0); removal targeted for v1.0 | Implement the `Store` interface against your persistence backend; validate with `contract.RunTests` | `MemoryStore` struct, `NewMemoryStore`, `store.go` |

## PLANNED

No code exists for any of these. They are referenced in documentation as future work.

| Feature                                                                                                                              | Evidence of intent |
| ------------------------------------------------------------------------------------------------------------------------------------ | ------------------ |
| **Middleware package** — `CommandIdempotency`, `EventIdempotency`, `QueryIdempotency` to wire the store into CQRS dispatch pipelines | `doc.go`           |

## NOT PLANNED (Out of Scope by Design)

This library is an interface-first SDK. It does not and will not ship production backends. `MemoryStore` is **deprecated** and intended for development and testing only; for any distributed or persistent backend, implement the `Store` interface yourself.

| Feature          | Reason                                                                                                                                 |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| **Redis store**  | Driver dependency, connection-pool semantics, cluster behavior, and eviction tradeoffs are the consumer's decision, not the library's. |
| **SQL store**    | Driver selection, schema/migration strategy, and cleanup policies are the consumer's decision, not the library's.                      |
