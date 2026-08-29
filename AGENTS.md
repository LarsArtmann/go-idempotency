# AGENTS.md

Go library providing an idempotency/deduplication store for CQRS command keys. Module: `github.com/larsartmann/go-idempotency` (Go 1.26+ — see go.mod).

## Commands

```bash
go test ./...          # run all tests
go test ./... -race    # run with race detector (MANDATORY — concurrency is core to this lib)
go test ./... -v       # verbose, shows property-test shrink traces
go vet ./...           # static analysis
golangci-lint run ./... # lint (see .golangci.yml for the enabled linters)
./scripts/check-stale-refs.sh  # fail on known-stale doc phrases (patterns listed in the script)
```

No `flake.nix`, Makefile, or justfile exists. This is a plain Go module — use `go` directly. CI (`.github/workflows/ci.yml`) runs 8 jobs on every push and PR: `go test -race` with coverage (Codecov upload self-activates only when the `CODECOV_TOKEN` secret exists), `go vet`, `golangci-lint`, a `gofmt` check, a `go mod tidy` diff check, 30s fuzzing per fuzz target (store and middleware), `govulncheck`, and a docs job.

Environment gotcha: the login shell may export `GOCACHE`/`GOMODCACHE`/`GOLANGCI_LINT_CACHE` pointing at a nonexistent `/mnt/buildcache/...`. Prefix go/golangci-lint commands with `export GOCACHE=/tmp/go-build-cache-idem GOMODCACHE=/tmp/go-mod-cache-idem GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache-idem`. Editor LSP diagnostics mentioning `/mnt/buildcache` are ghosts from the same cause — the CLI is the source of truth.

## Architecture

Multi-package SDK: root `package idempotency` (the `Store` contract plus the deprecated `MemoryStore`), `middleware/` (consumer-facing dispatch adapters), `contract/` (reusable test suite), `example/` (runnable walkthrough), `internal/teststore/` (shared test backend), and `scripts/` (repo tooling).

- **`Store` interface** (`store.go`) — three methods: `Seen`, `Record`, `CheckAndRecord`. All take `context.Context` but `MemoryStore` ignores it (params named `_`).
- **`MemoryStore`** (`store.go`) — **deprecated** (development/testing only; removal targeted for v1.0). In-memory `map[string]time.Time` guarded by `sync.RWMutex`, with TTL-based expiration via two mechanisms: a background sweep goroutine AND lazy deletion on read.
- **`doc.go`** — package documentation: table of contents, quick start, Redis + SQL adapter examples, and the response-replay recipe.
- **`middleware/`** — `Command`/`NewCommand` (transport-agnostic at-most-once dispatch) and the `net/http` adapter (`http.go`, `HeaderKey = "Idempotency-Key"`). Stdlib-only per ADR-002; fails closed on store errors.
- **`contract/`** — reusable contract test suite verifying any `Store` implementation against 13 invariants: `RunTests` (fast defaults) and `RunTestsStrict(t, factory, Options{TimingScale})` for slow CI runners. `contract_context.go` adds `RunTestsContextAware`, the opt-in cancellation-semantics suite (canceled call returns the context error and does not consume the claim); context-blind stores such as `MemoryStore` must NOT run it. Consumers import the package to verify their own backend.
- **`contract/contract_negative_test.go` + `contract/contract_context_negative_test.go`** — 19 deliberately broken Stores (14 main suite + 5 cancellation suite) proving each suite fails against every invariant's violation and names it. Both share the `runNegativeScenarios` subprocess harness (one env var per suite; sibling collateral failures are expected and harmless).
- **`example/`** — runnable walkthrough (`go run ./example`): a demo backend validated by the contract suite, then exactly-once processing from the caller's view. Deliberately no `MemoryStore`.
- **`internal/teststore/`** — module-internal test Store shared by the contract self-test, the negative tests, and the middleware tests (`ContextAware` wrapper adds the context checks for the cancellation-suite self-test). Consumers cannot import it; never add production-backend code here.
- **`scripts/check-stale-refs.sh`** — greps docs for known-stale phrases (the pattern list lives in the script); run before committing doc changes.

### Key Design Decisions (non-obvious)

- **`CheckAndRecord` is the atomic primitive.** Never split into `Seen` + `Record` — that creates a TOCTOU race. This is the preferred entry point for callers.
- **Non-positive TTL is rejected.** `Record` and `CheckAndRecord` return `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable) for `ttl <= 0`. A zero/negative TTL records an already-past expiry, which silently breaks the exactly-once guarantee — the store rejects the bad input instead.
- **`ErrDuplicate` is a `Conflict`** via `go-error-family`, which maps to HTTP 409 downstream and is non-retryable. Callers check with `errors.Is(err, idempotency.ErrDuplicate)`.
- **`Record` is a no-op on existing non-expired keys** — it does NOT extend the TTL. Expired keys (even if not yet swept) are re-recorded with a fresh TTL.
- **`Seen` takes a write lock** (`s.mu.Lock()`), not a read lock, because it performs lazy deletion of expired entries.
- **`Close` is idempotent** via `sync.Once`. Always `defer store.Close()`.
- **Context is ignored.** `MemoryStore` does not honor context cancellation — all params are `_`. The parameter exists on the interface so that custom backend implementations (Redis, SQL, etc.) can honor cancellation and timeouts.
- **`MemoryStore` is deprecated.** It carries `// Deprecated:` doc comments and is intended for development and testing only. Removal targeted for v1.0. The library is interface-first: consumers implement the `Store` interface against their own backend (validated by `contract.RunTests`). Never add backend code or driver dependencies to this module.
- **Middleware fails closed.** `middleware.NewCommand` executes the side effect only when `CheckAndRecord` returns `nil`. Store failures (network down, timeout) return a wrapped error and the command does NOT execute: unknown state is never treated as "not a duplicate", because executing twice is the one unrecoverable outcome.

### Future Surface (does not exist yet)

`EventIdempotency` and `QueryIdempotency` are named in docs but not implemented — YAGNI-gated per ADR-002 until a consumer needs them. `Delete` on `Store` is deferred per ADR-004. Do not assume either exists.

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
- **Six test files in root, split by strategy:**
  - `store_test.go` — unit and concurrency tests, one function per scenario. Concurrency tests use a `started chan struct{}` barrier to release all goroutines simultaneously. Includes edge cases (empty key, zero TTL, post-Close behavior).
  - `property_test.go` — property-based tests via `rapid.Check(t, func(t *rapid.T) {...})`. Generators: `rapid.String()`, `rapid.IntRange()`, `rapid.StringMatching()`.
  - `fuzz_test.go` — `FuzzCheckAndRecord`, `FuzzRecord`, `FuzzConcurrentMixed` for panic-safety and invariant checking on arbitrary inputs and concurrent interleavings. Also `TestMemoryStore_CloseDuringConcurrentOps` for use-after-close race safety.
  - `bench_test.go` — benchmarks for `CheckAndRecord`, `Seen`, `Record` under serial, contended, and parallel workloads. Memory benchmarks (`BenchmarkMemoryUsage_*`) report bytes/key and %-reclaimed.
  - `example_test.go` — godoc `ExampleStore` and `ExampleMemoryStore` functions that render on pkg.go.dev.
  - `contract_test.go` — runs `contract.RunTests` against `MemoryStore`.
- **Package-level test suites** (same conventions): `contract/contract_test.go` (suite self-tests: `RunTests` + scaled `RunTestsStrict` + `RunTestsContextAware` against `internal/teststore`; the scaled run reads `GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE`), `contract/contract_negative_test.go` + `contract/contract_context_negative_test.go` (detection proofs), `middleware/middleware_test.go` + `middleware/http_test.go` (unit + 50-goroutine at-most-once), `middleware/fuzz_test.go` (`FuzzDispatch`), `middleware/example_test.go` (`ExampleNewCommand`), `example/example_test.go` (contract suite over the demo backend).
- **Contract invariants pair with negative tests.** Every new invariant in `contract/contract.go` (or `contract/contract_context.go`) must ship with a broken-Store scenario in the matching negative-test file and a README row in the same PR (see "Extending the suite" in `contract.go`).
- **Always `defer store.Close()`**, even when sweep is disabled (`sweepInterval == 0`).
- Concurrency correctness (exactly-one-winner) is tested with 200 goroutines in unit tests, randomized 2–20 goroutines in property tests, and in the contract test suite.

## Code Conventions

- `//nolint:exhaustruct` is used when zero-valued struct fields are intentional (e.g., `sync.RWMutex`, `sync.Once`). The `exhaustruct` linter is enabled in `.golangci.yml`.
- Go 1.22+ range syntax: `for range n` (integer) and `for i := range n`.
- Doc comments explain the _why_ (TOCTOU prevention, atomicity guarantees), not the _what_.
- Error sentinels carry a stable string code (`"idempotency.duplicate"`, `"idempotency.invalid-ttl"`) as first arg to the `NewConflict`/`NewRejection` constructors.
