# Features

Honest inventory of what exists, what ships with gaps, and what is planned. Every status is verified against code.

## FULLY_FUNCTIONAL

| Feature                                                                                                                                                                         | Evidence                                                                |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Store interface** — 3-method abstraction (`Seen`, `Record`, `CheckAndRecord`) for idempotency key tracking                                                                    | `Store` interface, `store.go`                                           |
| **MemoryStore** — in-memory `Store` implementation backed by `map[string]time.Time` _(deprecated — see DEPRECATED below)_                                             | `MemoryStore` struct, `store.go`                                        |
| **Atomic CheckAndRecord** — single-lock check-and-set that prevents the TOCTOU race a separate Seen+Record pair would create                                                    | `MemoryStore.CheckAndRecord`, tested `TestMemoryStore_CheckAndRecord_*` |
| **TTL-based expiration (dual mechanism)** — background sweep goroutine + lazy deletion on read; map cannot grow unboundedly even with sweep disabled                            | `MemoryStore.sweep`, `MemoryStore.Seen`, `MemoryStore.Record`           |
| **Configurable sweep interval** — `sweepInterval == 0` disables background goroutine; lazy deletion still bounds growth                                                         | `NewMemoryStore` constructor                                            |
| **Idempotent Record** — calling `Record` on an existing, non-expired key is a no-op (TTL is not extended). Expired keys are re-recorded with the fresh TTL.                     | `MemoryStore.Record`, tested `store_test.go`                            |
| **ErrDuplicate as Conflict** — sentinel error classified as `Conflict` (HTTP 409, non-retryable) via `go-error-family`                                                          | `ErrDuplicate` var, tested `store_test.go`                              |
| **TTL validation** — `Record` and `CheckAndRecord` reject non-positive TTL with `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable); prevents silent exactly-once breakage | `ErrInvalidTTL` var, tested `store_test.go`                             |
| **Concurrency safety** — `sync.RWMutex` protection; exactly-one-winner tested with 200 goroutines + randomized 2–20 goroutine property test                                     | `store_test.go`, `property_test.go`                                     |
| **Graceful shutdown** — `Close()` stops sweep goroutine; idempotent via `sync.Once`. Operations still function after Close.                                                     | `MemoryStore.Close`, tested `store_test.go`                             |
| **Property-based testing** — `pgregory.net/rapid` property tests for idempotency, exact-once concurrency, key independence, TTL expiry                                          | `property_test.go`                                                      |
| **Fuzz tests** — `FuzzCheckAndRecord`, `FuzzRecord`, `FuzzConcurrentMixed` on the store plus `FuzzDispatch` on the middleware: panic-safety, invariants, and at-most-once execution on arbitrary keys/TTLs | `fuzz_test.go`, `middleware/fuzz_test.go`                               |
| **Sweep under load** — 1000-key concurrent soak test verifying sweep reclaims all expired entries                                                                               | `store_test.go`                                                         |
| **Benchmarks** — serial, contended, and parallel benchmarks for `CheckAndRecord`, `Seen`, `Record`, plus memory-usage benchmarks reporting bytes/key and %-reclaimed            | `bench_test.go`                                                         |
| **Edge case coverage** — empty key, non-positive TTL rejection, post-Close operations all tested                                                                                | `store_test.go`                                                         |
| **Contract test suite** — reusable test harness that verifies any `Store` implementation against the full invariant set; consumers run it against their own backend             | `contract.RunTests`, `contract/contract.go`                             |
| **Strict contract run** — `RunTestsStrict` with `Options.TimingScale` stretches every wall-clock timing so TTL subtests stay stable on slow or loaded CI runners      | `contract.RunTestsStrict`, `contract/contract.go`                       |
| **Context-aware contract suite** — opt-in `RunTestsContextAware` asserts that a canceled call returns the context error and does not consume the claim; for backends honoring cancellation (context-blind stores must not run it) | `contract.RunTestsContextAware`, `contract/contract_context.go`         |
| **Proven detection (negative tests)** — 19 deliberately broken Stores (14 main suite + 5 cancellation suite), at least one per invariant; the suites are proven to fail against each violation and name it | `contract/contract_negative_test.go`, `contract/contract_context_negative_test.go` |
| **Middleware command dispatch** — `NewCommand` wraps any command function for at-most-once execution per idempotency key; sentinel errors pass through unchanged, store failures fail closed | `middleware.NewCommand`, `middleware/middleware.go`                     |
| **HTTP adapter** — `net/http` middleware reading the `Idempotency-Key` header: first request processes, duplicates get `409 Conflict`, missing header is `400`                 | `middleware/http.go`, `middleware.HeaderKey`                            |
| **Runnable example** — end-to-end walkthrough: a demo backend validated by the contract suite, then exactly-once processing from the caller's view (`go run ./example`)        | `example/main.go`                                                       |
| **Lint configuration** — `.golangci.yml` enables `exhaustruct`, `gosec`, `revive`, `misspell`, `gocritic`                                                                       | `.golangci.yml`                                                         |
| **CI pipeline** — 8 GitHub Actions jobs on every push/PR: test (`-race` + coverage, Codecov upload gated on the secret), vet, lint (60+ linters), gofmt, tidy-diff, fuzz (30s/target), govulncheck, docs | `.github/workflows/ci.yml`                                              |

## DEPRECATED

These remain functional and tested, but are scheduled for removal in a future major version. Do not build new production code on them.

| Feature                                                                                                     | Status                                                                | Replacement                                                                                         | Evidence                                           |
| ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **MemoryStore** — in-memory `Store` with TTL expiration, background sweep, lazy deletion, graceful shutdown | Deprecated since v0.2.0; removal targeted for v1.0 | Implement the `Store` interface against your persistence backend; validate with `contract.RunTests` | `MemoryStore` struct, `NewMemoryStore`, `store.go` |

## PLANNED

No code exists for any of these. They are referenced in documentation as future work.

| Feature                                                                                                                              | Evidence of intent |
| ------------------------------------------------------------------------------------------------------------------------------------ | ------------------ |
| **`EventIdempotency` / `QueryIdempotency` middleware** — event-delivery and query-path adapters; YAGNI-gated until a consumer needs them | ADR-002, `middleware` package doc |

## NOT PLANNED (Out of Scope by Design)

This library is an interface-first SDK. It does not and will not ship production backends. `MemoryStore` is **deprecated** and intended for development and testing only; for any distributed or persistent backend, implement the `Store` interface yourself.

| Feature         | Reason                                                                                                                                 |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| **Redis store** | Driver dependency, connection-pool semantics, cluster behavior, and eviction tradeoffs are the consumer's decision, not the library's. |
| **SQL store**   | Driver selection, schema/migration strategy, and cleanup policies are the consumer's decision, not the library's.                      |
