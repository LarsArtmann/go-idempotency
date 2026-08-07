# Features

Honest inventory of what exists, what ships with gaps, and what is planned. Every status is verified against code.

## FULLY_FUNCTIONAL

| Feature                                                                                                                                                     | Evidence                                               |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| **Store interface** — 3-method abstraction (`Seen`, `Record`, `CheckAndRecord`) for idempotency key tracking                                                | `store.go:30-48`                                       |
| **MemoryStore** — in-memory `Store` implementation backed by `map[string]time.Time`                                                                         | `store.go:56-61`                                       |
| **Atomic CheckAndRecord** — single-lock check-and-set that prevents the TOCTOU race a separate Seen+Record pair would create                                | `store.go:118-129`, tested `store_test.go:50-97`       |
| **TTL-based expiration (dual mechanism)** — background sweep goroutine + lazy deletion on read; map cannot grow unboundedly even with sweep disabled        | `store.go:74-76`, `store.go:83-99`, `store.go:136-156` |
| **Configurable sweep interval** — `sweepInterval == 0` disables background goroutine; lazy deletion still bounds growth                                     | `store.go:63-79`                                       |
| **Idempotent Record** — calling `Record` on an existing, non-expired key is a no-op (TTL is not extended). Expired keys are re-recorded with the fresh TTL. | `store.go:103-114`, tested `store_test.go`             |
| **ErrDuplicate as Conflict** — sentinel error classified as `Conflict` (HTTP 409, non-retryable) via `go-error-family`                                      | `store.go:15-18`, tested `store_test.go`               |
| **TTL validation** — `Record` and `CheckAndRecord` reject non-positive TTL with `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable); prevents silent exactly-once breakage | `store.go`, tested `store_test.go`                     |
| **Concurrency safety** — `sync.RWMutex` protection; exactly-one-winner tested with 200 goroutines + randomized 2–20 goroutine property test                 | `store_test.go`, `property_test.go`                    |
| **Graceful shutdown** — `Close()` stops sweep goroutine; idempotent via `sync.Once`. Operations still function after Close.                                 | `store.go:132-134`, tested `store_test.go`             |
| **Property-based testing** — `pgregory.net/rapid` property tests for idempotency, exact-once concurrency, key independence, TTL expiry                      | `property_test.go`                                     |
| **Sweep under load** — 1000-key concurrent soak test verifying sweep reclaims all expired entries                                                           | `store_test.go`                                        |
| **Benchmarks** — serial, contended, and parallel benchmarks for `CheckAndRecord`, `Seen`, `Record`                                                          | `bench_test.go`                                        |
| **Edge case coverage** — empty key, non-positive TTL rejection, post-Close operations all tested                                                              | `store_test.go`                                        |
| **Lint configuration** — `.golangci.yml` enables `exhaustruct`, `gosec`, `revive`, `misspell`, `gocritic`                                                   | `.golangci.yml`                                        |
| **CI pipeline** — GitHub Actions runs `go test -race`, `go vet`, `golangci-lint` on every push and PR                                                       | `.github/workflows/ci.yml`                             |

## PLANNED

No code exists for any of these. They are referenced in documentation as future work.

| Feature                                                                                                                              | Evidence of intent |
| ------------------------------------------------------------------------------------------------------------------------------------ | ------------------ |
| **Middleware package** — `CommandIdempotency`, `EventIdempotency`, `QueryIdempotency` to wire the store into CQRS dispatch pipelines | `doc.go:26-31`     |
| **Redis store** — distributed idempotency using `SET NX` for atomic check-and-record                                                 | `store.go:42-43`   |
| **SQL store** — persistent idempotency using `INSERT ... ON CONFLICT DO NOTHING`                                                     | `store.go:44-46`   |
