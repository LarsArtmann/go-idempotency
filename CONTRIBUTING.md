# Contributing

Thanks for your interest in contributing to go-idempotency.

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
```

All three checks (test, vet, lint) must pass. CI runs them automatically on push and PR.

## Testing Strategy

Tests live in `idempotency_test` (external test package) and are split by strategy:

- **`store_test.go`** — unit and concurrency tests, one function per scenario. Concurrency tests use a `started chan struct{}` barrier to release all goroutines simultaneously. Every test calls `t.Parallel()`.
- **`property_test.go`** — property-based tests via `pgregory.net/rapid`. Invariants tested: idempotent Record, exact-once CheckAndRecord, key independence, TTL expiry.
- **`bench_test.go`** — benchmarks for `CheckAndRecord`, `Seen`, and `Record` under serial, contended, and parallel workloads.

Always `defer store.Close()`, even when sweep is disabled (`sweepInterval == 0`).

## Code Conventions

- **`//nolint:exhaustruct`** marks structs where zero-valued fields are intentional (e.g., `sync.RWMutex`, `sync.Once`). The `exhaustruct` linter is enabled in `.golangci.yml`.
- **Go 1.22+ range syntax**: `for range n` (integer) and `for i := range n`.
- **Doc comments explain the *why*** (TOCTOU prevention, atomicity guarantees), not the *what*.
- **Error sentinels** carry a stable string code (e.g., `"idempotency.duplicate"`) as the first argument to `errorfamily.NewConflict`. This code is a public API contract.

## Project Documentation

- [AGENTS.md](AGENTS.md) — architecture, design decisions, and non-obvious context
- [FEATURES.md](FEATURES.md) — honest feature inventory with code evidence
- [ROADMAP.md](ROADMAP.md) — long-term direction and raw ideas
- [TODO_LIST.md](TODO_LIST.md) — short-term actionable work

## Reporting Issues

Please use [GitHub Issues](https://github.com/larsartmann/go-idempotency/issues) to report bugs or request features.
