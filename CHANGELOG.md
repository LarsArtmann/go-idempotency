# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **`Record` now checks TTL expiry** — `MemoryStore.Record` previously checked map presence only, not expiry. An expired-but-not-yet-swept key was treated as "already recorded" and `Record` was a no-op. Now expired keys are re-recorded with the fresh TTL, matching `CheckAndRecord`'s behavior (`store.go:103-114`).

### Added

- Project scaffolding: `.editorconfig`, `.gitattributes`, `.gitignore`
- `LICENSE` — proprietary license
- `CONTRIBUTING.md` — development setup and contribution workflow
- `README.md` — project overview and quick start
- `AGENTS.md` — non-obvious context for AI coding sessions
- `FEATURES.md` — honest feature inventory with `file:line` evidence
- `TODO_LIST.md` — short-term actionable work
- `ROADMAP.md` — long-term direction and raw ideas
- `.golangci.yml` — lint configuration enabling `exhaustruct`, `gosec`, `revive`, `misspell`, `gocritic`
- `.github/workflows/ci.yml` — CI pipeline: `go test -race`, `go vet`, `golangci-lint` on push and PR
- `bench_test.go` — benchmarks for `CheckAndRecord` (serial, contended, parallel-unique), `Seen` (hit, miss), `Record`
- Test: `Record` after expiry re-records with new TTL (`store_test.go`)

### Changed

- Refined `property_test.go` test patterns
- `README.md` rebuilt with real description, verified quick-start example, and accurate development commands
- `CHANGELOG.md` rebuilt with detailed v0.1.0 feature breakdown
- `CONTRIBUTING.md` rebuilt with project-specific guidance, testing strategy, and code conventions

## [0.1.0] - 2026-08-03

Tagged `v0.1.0`: initial release — idempotency store with TTL-based dedup.

### Added

- **Store interface** (`store.go:30-48`) — 3-method abstraction: `Seen`, `Record`, `CheckAndRecord`
- **MemoryStore** (`store.go:56-156`) — in-memory implementation backed by `map[string]time.Time`
- **Atomic CheckAndRecord** — single-write-lock check-and-set preventing the TOCTOU race that a separate `Seen` + `Record` pair would create
- **TTL-based expiration with dual mechanism** — background sweep goroutine (configurable interval) + lazy deletion on read; map cannot grow unboundedly even when sweep is disabled (`sweepInterval == 0`)
- **Idempotent Record** — recording an existing key is a no-op; TTL is never extended
- **ErrDuplicate as Conflict** — sentinel error classified as `Conflict` (HTTP 409, non-retryable) via `go-error-family`
- **Graceful shutdown** — `Close()` stops the sweep goroutine; idempotent via `sync.Once`
- **Comprehensive test suite** — unit tests (`store_test.go`), property-based tests via `pgregory.net/rapid` (`property_test.go`)
- **Concurrency correctness tests** — exactly-one-winner verified with 200 concurrent goroutines and randomized 2–20 goroutine property tests
- **Sweep soak test** — 1000-key concurrent load test verifying sweep reclaims all expired entries
- **Package documentation** (`doc.go`) with quick-start example and design rationale

[Unreleased]: https://github.com/larsartmann/go-idempotency/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/larsartmann/go-idempotency/releases/tag/v0.1.0
