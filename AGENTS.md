# AGENTS.md

Go library providing an idempotency/deduplication store for CQRS command keys. Module: `github.com/larsartmann/go-idempotency` (Go 1.26.5).

## Commands

```bash
go test ./...          # run all tests
go test ./... -race    # run with race detector (recommended — concurrency is core to this lib)
go test ./... -v       # verbose, shows property-test shrink traces
go vet ./...           # static analysis (currently clean)
golangci-lint run ./... # lint (no .golangci config file committed yet)
```

No `flake.nix`, Makefile, or justfile exists. This is a plain Go module — use `go` directly. `CONTRIBUTING.md` references `golangci-lint run ./...`, which is valid but there is no `.golangci.yml` committed yet (see TODO_LIST.md).

## Architecture

Single-package library (`package idempotency`), flat file layout — no subdirectories.

- **`Store` interface** (`store.go`) — three methods: `Seen`, `Record`, `CheckAndRecord`. All take `context.Context` but `MemoryStore` ignores it (params named `_`).
- **`MemoryStore`** (`store.go`) — the only implementation. In-memory `map[string]time.Time` guarded by `sync.RWMutex`, with TTL-based expiration via two mechanisms: a background sweep goroutine AND lazy deletion on read.
- **`doc.go`** — package documentation with quick-start example.

### Key Design Decisions (non-obvious)

- **`CheckAndRecord` is the atomic primitive.** Never split into `Seen` + `Record` — that creates a TOCTOU race. This is the preferred entry point for callers.
- **`ErrDuplicate` is a `Conflict`** via `go-error-family`, which maps to HTTP 409 downstream and is non-retryable. Callers check with `errors.Is(err, idempotency.ErrDuplicate)`.
- **`Record` is a no-op on existing keys** — it does NOT extend the TTL. This is intentional.
- **`Seen` takes a write lock** (`s.mu.Lock()`), not a read lock, because it performs lazy deletion of expired entries.
- **`Close` is idempotent** via `sync.Once`. Always `defer store.Close()`.
- **Context is ignored.** `MemoryStore` does not honor context cancellation — all params are `_`. A future Redis/SQL store would need to use it.

### Future Surface (does not exist yet)

`doc.go` references a "middleware package" with `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency`. This is planned but not implemented — do not assume it exists.

## Dependencies

| Dependency                               | Purpose                                                                                                                                                                             |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `github.com/larsartmann/go-error-family` | Error classification. `ErrDuplicate` uses `errorfamily.NewConflict(code, msg)`. `errorfamily.Classify(err)` returns the family; `errorfamily.IsRetryable(err)` checks retryability. |
| `pgregory.net/rapid`                     | Property-based testing framework (test-only).                                                                                                                                       |

## Testing Conventions

- **External test package** (`idempotency_test`) — imports the package as a consumer.
- **`t.Parallel()` on every test.**
- **Two test files, split by strategy:**
  - `store_test.go` — example/unit tests, one function per scenario. Concurrency tests use a `started chan struct{}` barrier to release all goroutines simultaneously.
  - `property_test.go` — property-based tests via `rapid.Check(t, func(t *rapid.T) {...})`. Generators: `rapid.String()`, `rapid.IntRange()`, `rapid.StringMatching()`.
- **Always `defer store.Close()`**, even when sweep is disabled (`sweepInterval == 0`).
- Concurrency correctness (exactly-one-winner) is tested with 200 goroutines in unit tests and randomized 2–20 goroutines in property tests.

## Code Conventions

- `//nolint:exhaustruct` is used when zero-valued struct fields are intentional (e.g., `sync.RWMutex`, `sync.Once`). This implies `exhaustruct` linter is expected if running golangci-lint.
- Go 1.22+ range syntax: `for range n` (integer) and `for i := range n`.
- Doc comments explain the _why_ (TOCTOU prevention, atomicity guarantees), not the *what`.
- Error sentinels carry a stable string code (`"idempotency.duplicate"`) as first arg to `NewConflict`.
