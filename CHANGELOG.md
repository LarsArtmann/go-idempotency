# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Project scaffolding: `.editorconfig`, `.gitattributes`, `.gitignore`
- `LICENSE` — proprietary license
- `CONTRIBUTING.md` — development setup and contribution workflow
- `README.md` — project overview and quick start
- `AGENTS.md` — non-obvious context for AI coding sessions

### Changed

- Refined `property_test.go` test patterns

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
