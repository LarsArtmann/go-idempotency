# AGENTS.md

Go library providing an idempotency/deduplication store for CQRS command keys. Module: `github.com/larsartmann/go-idempotency` (Go 1.26.5).

## Commands

```bash
go test ./...          # run all tests
go test ./... -race    # run with race detector (MANDATORY — concurrency is core to this lib)
go test ./... -v       # verbose, shows property-test shrink traces
go vet ./...           # static analysis (currently clean)
golangci-lint run ./... # lint (uses .golangci.yml: 60+ linters, see file for full list)
```

No `flake.nix`, Makefile, or justfile exists. This is a plain Go module — use `go` directly. CI (`.github/workflows/ci.yml`) runs `go test -race`, `go vet`, and `golangci-lint` on every push and PR.

## Architecture

Single-package library (`package idempotency`) with a `contract/` subpackage for reusable test infrastructure. Root package is flat — no subdirectories except `contract/`.

- **`Store` interface** (`store.go`) — three methods: `Seen`, `Record`, `CheckAndRecord`. All take `context.Context` but `MemoryStore` ignores it (params named `_`).
- **`MemoryStore`** (`store.go`) — the reference implementation. In-memory `map[string]time.Time` guarded by `sync.RWMutex`, with TTL-based expiration via two mechanisms: a background sweep goroutine AND lazy deletion on read.
- **`doc.go`** — package documentation with quick-start example and Redis adapter implementation example.
- **`contract/`** — reusable contract test suite (`RunTests`) that verifies any `Store` implementation against the full invariant set. Consumers import it to verify their own backend.

### Key Design Decisions (non-obvious)

- **`CheckAndRecord` is the atomic primitive.** Never split into `Seen` + `Record` — that creates a TOCTOU race. This is the preferred entry point for callers.
- **Non-positive TTL is rejected.** `Record` and `CheckAndRecord` return `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable) for `ttl <= 0`. A zero/negative TTL records an already-past expiry, which silently breaks the exactly-once guarantee — the store rejects the bad input instead.
- **`ErrDuplicate` is a `Conflict`** via `go-error-family`, which maps to HTTP 409 downstream and is non-retryable. Callers check with `errors.Is(err, idempotency.ErrDuplicate)`.
- **`Record` is a no-op on existing non-expired keys** — it does NOT extend the TTL. Expired keys (even if not yet swept) are re-recorded with a fresh TTL.
- **`Seen` takes a write lock** (`s.mu.Lock()`), not a read lock, because it performs lazy deletion of expired entries.
- **`Close` is idempotent** via `sync.Once`. Always `defer store.Close()`.
- **Context is ignored.** `MemoryStore` does not honor context cancellation — all params are `_`. The parameter exists on the interface so that custom backend implementations (Redis, SQL, etc.) can honor cancellation and timeouts.
- **`MemoryStore` is deprecated.** It carries `// Deprecated:` doc comments and is intended for development and testing only. Removal targeted for v1.0. The library is interface-first: consumers implement the `Store` interface against their own backend (validated by `contract.RunTests`). Never add backend code or driver dependencies to this module.

### Future Surface (does not exist yet)

`doc.go` references a "middleware package" with `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency`. This is planned but not implemented — do not assume it exists.

### Backend Implementations (Out of Scope)

This library will NOT ship production backends (Redis, SQL, etc.). `MemoryStore` is **deprecated** (dev/test only, removal targeted for v1.0). Consumers implement the `Store` interface against their own backend. The atomic primitives for common backends are documented in the `CheckAndRecord` comment: Redis `SET NX`, SQL `INSERT ... ON CONFLICT DO NOTHING`.

## Dependencies

| Dependency                               | Purpose                                                                                                                                                                             |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `github.com/larsartmann/go-error-family` | Error classification. `ErrDuplicate` uses `errorfamily.NewConflict(code, msg)`. `errorfamily.Classify(err)` returns the family; `errorfamily.IsRetryable(err)` checks retryability. |
| `pgregory.net/rapid`                     | Property-based testing framework (test-only).                                                                                                                                       |

## Testing Conventions

- **External test package** (`idempotency_test`) — imports the package as a consumer.
- **`t.Parallel()` on every test.**
- **Five test files in root, split by strategy:**
  - `store_test.go` — unit and concurrency tests, one function per scenario. Concurrency tests use a `started chan struct{}` barrier to release all goroutines simultaneously. Includes edge cases (empty key, zero TTL, post-Close behavior).
  - `property_test.go` — property-based tests via `rapid.Check(t, func(t *rapid.T) {...})`. Generators: `rapid.String()`, `rapid.IntRange()`, `rapid.StringMatching()`.
  - `fuzz_test.go` — `FuzzCheckAndRecord`, `FuzzRecord` for panic-safety and invariant checking on arbitrary inputs. Also `TestMemoryStore_CloseDuringConcurrentOps` for use-after-close race safety.
  - `bench_test.go` — benchmarks for `CheckAndRecord`, `Seen`, `Record` under serial, contended, and parallel workloads. Memory benchmarks (`BenchmarkMemoryUsage_*`) report bytes/key and %-reclaimed.
  - `example_test.go` — godoc `ExampleStore` and `ExampleMemoryStore` functions that render on pkg.go.dev.
  - `contract_test.go` — runs `contract.RunTests` against `MemoryStore`.
- **Always `defer store.Close()`**, even when sweep is disabled (`sweepInterval == 0`).
- Concurrency correctness (exactly-one-winner) is tested with 200 goroutines in unit tests, randomized 2–20 goroutines in property tests, and in the contract test suite.

## Code Conventions

- `//nolint:exhaustruct` is used when zero-valued struct fields are intentional (e.g., `sync.RWMutex`, `sync.Once`). The `exhaustruct` linter is enabled in `.golangci.yml`.
- Go 1.22+ range syntax: `for range n` (integer) and `for i := range n`.
- Doc comments explain the _why_ (TOCTOU prevention, atomicity guarantees), not the _what_.
- Error sentinels carry a stable string code (`"idempotency.duplicate"`, `"idempotency.invalid-ttl"`) as first arg to the `NewConflict`/`NewRejection` constructors.
