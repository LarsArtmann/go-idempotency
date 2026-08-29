# Contributing

Thanks for your interest in contributing to go-idempotency.

## Scope

This library is an interface-first SDK. It provides the `Store` interface and `MemoryStore` (a **deprecated** reference implementation for development and testing only). **Backend implementations (Redis, SQL, DynamoDB, etc.) are out of scope and will not be accepted as PRs.** Implement the `Store` interface in your own project; use the `contract` package test suite to verify correctness.

## How to Contribute

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes with tests
4. Ensure all checks pass (see below)
5. Submit a pull request

## Development Setup

Requires Go 1.26+.

```bash
go test ./... -race -count=1   # run all tests with race detector (mandatory)
go vet ./...                   # static analysis
golangci-lint run ./...        # lint (uses .golangci.yml)
go test -bench=. -benchmem     # run benchmarks
go test -fuzz=FuzzCheckAndRecord -fuzztime=30s   # short fuzz run
```

All three checks (test, vet, lint) must pass. CI runs them automatically on push and PR.

## Testing Strategy

Tests live in `idempotency_test` (external test package) and are split by strategy:

- **`store_test.go`** — unit and concurrency tests, one function per scenario. Concurrency tests use a `started chan struct{}` barrier to release all goroutines simultaneously. Every test calls `t.Parallel()`.
- **`property_test.go`** — property-based tests via `pgregory.net/rapid`. Invariants tested: idempotent Record, exact-once CheckAndRecord, key independence, TTL expiry.
- **`fuzz_test.go`** — `FuzzCheckAndRecord`, `FuzzRecord`, and `TestMemoryStore_CloseDuringConcurrentOps`.
- **`bench_test.go`** — benchmarks for `CheckAndRecord`, `Seen`, and `Record` under serial, contended, and parallel workloads, plus memory usage benchmarks.
- **`example_test.go`** — godoc `Example()` functions rendered on pkg.go.dev.
- **`contract_test.go`** — runs the `contract.RunTests` suite against `MemoryStore`.
- **`contract/contract.go`** — the reusable contract test suite itself (importable by consumers). It checks 13 invariants: Seen reports unseen/false, recorded/true, expired/false; Record is a no-op on live keys (never extends TTL), re-records after expiry, rejects `ttl <= 0` with `ErrInvalidTTL`; CheckAndRecord succeeds once, returns `ErrDuplicate` on duplicates, allows after expiry, rejects `ttl <= 0`; 200-goroutine atomicity (exactly one winner); key independence; empty-key validity. The consumer-facing table is in README ("What RunTests checks"). If you change the suite, update that table in the same PR.
- **`contract/contract_test.go`** — self-test: runs `RunTests` against `contract/internal`, an in-memory test Store, so the suite itself is exercised in this repo's CI.
- **`contract/contract_negative_test.go`** — negative tests: deliberately broken Stores (duplicate swallowed/generic, TTL-blind Record/CheckAndRecord) prove the suite fails against each violation and names the violated invariant.
- **`contract/internal/`** — minimal test-only in-memory `Store` (internal package; consumers cannot import it). Not a production backend — see ADR-001.

Always `defer store.Close()`, even when sweep is disabled (`sweepInterval == 0`).

## Code Conventions

- **`//nolint:exhaustruct`** marks structs where zero-valued fields are intentional (e.g., `sync.RWMutex`, `sync.Once`). The `exhaustruct` linter is enabled in `.golangci.yml`.
- **Go 1.22+ range syntax**: `for range n` (integer) and `for i := range n`.
- **Doc comments explain the _why_** (TOCTOU prevention, atomicity guarantees), not the _what_.
- **Error sentinels** carry a stable string code (e.g., `"idempotency.duplicate"`) as the first argument to `errorfamily.NewConflict`. This code is a public API contract.

## Project Documentation

- [AGENTS.md](AGENTS.md) — architecture, design decisions, and non-obvious context
- [FEATURES.md](FEATURES.md) — honest feature inventory with code evidence
- [ROADMAP.md](ROADMAP.md) — long-term direction and raw ideas
- [TODO_LIST.md](TODO_LIST.md) — short-term actionable work

## Reporting Issues

Please use [GitHub Issues](https://github.com/larsartmann/go-idempotency/issues) to report bugs or request features.
