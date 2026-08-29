# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Repository governance files** — SECURITY.md (private vulnerability reporting, supported-version policy, honest scope note given the zero-network design), CODE_OF_CONDUCT.md (Contributor Covenant 2.1), bug-report and feature-request issue templates oriented around doc-set expectations and contract-suite evidence, a PR checklist that includes the format/tidy/stale-refs gates, and CODEOWNERS.
- **Contract suite self-test** — `contract/` now ships a test-only internal in-memory `Store` (`contract/internal/`, internal package so consumers cannot import it) and runs `RunTests` against it in `contract/contract_test.go`. The suite is exercised in this repo's own CI (79.6% coverage of `contract/`) instead of shipping with zero coverage.
- **Negative contract tests** — four deliberately broken Stores (duplicate swallowed as nil, duplicate returning a generic error, non-positive TTL accepted by `Record` and by `CheckAndRecord`) prove the suite detects each violation AND names the violated invariant in its failure output. Each scenario re-executes the test binary in a subprocess, because an expected `t.Fatal` cannot be observed inside the parent's own test tree.
- **Enriched fuzz seed corpus** — both fuzz targets now seed an empty key, a 4 KB key, a unicode/emoji key, `math.MaxInt64` (TTL conversion overflow), and negative TTLs, up from 3 inputs per target. 60 s of live fuzzing (15.7M execs) found no failures.
- **Context-cancellation test guidance** — the `contract` package docs gain a "Testing context cancellation" section with a copy-paste test pinning the two invariants backends must honor: a canceled call returns the context error (not nil, not `ErrDuplicate`) and does NOT consume the claim, so the retry after a timeout stays processable. Linked from the README "Implementing your own backend" section.
- **Recipe: dedup + response replay (HTTP idempotency)** — closes the gap that stopped the PapDashboard consumer evaluation (feedback B1): `doc.go` now shows the composition where `CheckAndRecord` takes the atomic claim and the finished response is stored under a derived `resp:` key in the consumer's own KV backend; duplicates replay the stored response instead of getting a bare 409, claimed-but-unfinished keys return 409 until TTL, and the crash-gap between claim and response save is documented with its remedies. `Store` stays key-only; zero new dependencies. Linked from the README Documentation section.
- **Migration guide from MemoryStore** — `docs/migrating-from-memorystore.md`: why the store is deprecated, a backend-choice table, complete worked implementations (in-process map for tests, Redis `SET NX` for production), contract-suite wiring for CI, a before/after swap table, and a migration checklist. The in-process example was verified by running the full `contract.RunTests` suite against it under `-race`; the guide's snippets compile against the published module. Linked from both `Deprecated:` notices in `store.go` and the README.
- **Contract invariant list for consumers** — the README now renders all thirteen `RunTests` invariants as a per-method table (each row = one subtest), and CONTRIBUTING carries the short list plus a rule to keep the table in sync when the suite changes.

## [0.2.0] - 2026-08-29

### Changed

- **Interface-first reframe** — the library is now documented and shipped as an SDK, not a batteries-included framework: it owns the `Store` interface, its error semantics, and the contract test suite, and deliberately ships no production backend (see ADR-001). README and package docs rewritten around "you implement the backend"; Redis `SET NX` / SQL `INSERT ... ON CONFLICT DO NOTHING` named as the atomic primitives per backend.
- **CI hardening** — GitHub Actions pinned by commit SHA; coverage is reported in CI.
- **Deprecation-consistency pass completed** — closes the P0 debt from `docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md`: the `Store` interface comment now points to the deprecation instead of recommending `MemoryStore` (`store.go`); `CONTRIBUTING.md` scope marks MemoryStore deprecated; `AGENTS.md` architecture bullet marks it deprecated; `ExampleStore`/`ExampleMemoryStore` note that the deprecated MemoryStore is illustrative and link to the custom-backend section (`example_test.go`); the contract test comment calls MemoryStore "the deprecated in-process implementation" (`contract_test.go`). Also marks the `doc.go` Redis adapter example as illustrative (the redis client is intentionally not a dependency) and documents why `BenchmarkMemoryUsage_AfterSweep` reports well under 100% reclaimed (`bench_test.go`). CONTRIBUTING.md development setup gained a short fuzz command.
- **Documentation health audit (2026-08-29)** — accuracy and freshness fixes across living docs: `AGENTS.md` (Go version no longer hardcoded — points at go.mod; "five test files" corrected to six; CI description mentions coverage reporting), `FEATURES.md` (fuzz tests and memory benchmarks added as FULLY_FUNCTIONAL rows; deprecation status corrected to "unreleased; slated for v0.2.0" since v0.2.0 is not yet tagged), `ROADMAP.md` (v0.2.0 marked planned; new "In-Process Store Evolution" section harvesting the PapDashboard consumer evaluation: response-replay recipe and bounded-store-or-documented-position ideas), `docs/DOMAIN_LANGUAGE.md` (added `ErrInvalidTTL` and `Rejection` glossary entries), `docs/adr/001-no-backends.md` (deprecation release wording corrected), `TODO_LIST.md` rebuilt (stale "Done (this session)" trophy section removed — the items live in this changelog; harvested open work: cut v0.2.0, deprecation lint gate, contract suite self-test, fuzz seed corpus, context-cancellation test guidance, coverage badge).

### Added

- **Enforced deprecation lint gate** — `forbidigo` in `.golangci.yml` now fails the build on any `MemoryStore`/`NewMemoryStore` usage outside `_test.go` (the `store.go` implementation itself is exempt). The deprecation is mechanically enforced by CI, not just documented.
- **Contract test suite** — `contract/contract.go` with `RunTests(t *testing.T, factory StoreFactory)` covering all `Store` invariants: Seen, Record, CheckAndRecord, TTL expiry, duplicate detection, concurrency safety (200 goroutines), error semantics, empty key handling. Run against `MemoryStore` in `contract_test.go`. Consumers import `github.com/larsartmann/go-idempotency/contract` to verify their own backend.
- **Fuzz tests** — `FuzzCheckAndRecord` and `FuzzRecord` in `fuzz_test.go` verifying no panics on arbitrary keys and TTLs, and that exactly-once and TTL-validation invariants hold.
- **Close+concurrent race test** — `TestMemoryStore_CloseDuringConcurrentOps` verifying no panic when `Close()` is called mid-flight during concurrent operations.
- **Memory benchmarks** — `BenchmarkMemoryUsage_10KKeys` (reports bytes/key) and `BenchmarkMemoryUsage_AfterSweep` (reports %-reclaimed) in `bench_test.go`.
- **Godoc examples** — `ExampleStore` and `ExampleMemoryStore` in `example_test.go`, rendered on pkg.go.dev.
- **Redis adapter example** — full three-method Redis `SET NX` adapter in `doc.go` package docs showing how to implement the `Store` interface.
- **ADR-001: No Production Backends** — architecture decision record documenting the interface-first choice, alternatives, and consequences (`docs/adr/001-no-backends.md`).

### Deprecated

- **`MemoryStore` and `NewMemoryStore`** — `// Deprecated:` doc comments added in `store.go`. MemoryStore is intended for development and testing only; it does not survive restarts and cannot be shared across instances. For production, implement the `Store` interface against your persistence backend and validate with `contract.RunTests`. Removal targeted for v1.0. All docs (`doc.go`, `README.md`, `FEATURES.md`, `ROADMAP.md`) updated to reflect the deprecation.

### Changed

- **Documentation reframed to interface-first SDK philosophy.** All docs now make clear that go-idempotency provides the `Store` interface and `MemoryStore` (a reference implementation) only — it intentionally does NOT and will NOT ship production backends (Redis, SQL, etc.). Consumers implement the interface against their own backend. Affected files: `doc.go`, `store.go`, `README.md`, `ROADMAP.md`, `FEATURES.md`, `TODO_LIST.md`, `docs/DOMAIN_LANGUAGE.md`, `AGENTS.md`.
  - `ROADMAP.md`: "Distributed Backends" section replaced with "Backend Implementations (Out of Scope by Design)".
  - `FEATURES.md`: Redis/SQL moved from PLANNED to a new "NOT PLANNED" section; evidence column switched from brittle line numbers to durable symbol-name references.
  - `README.md`: added "Design philosophy" section with Redis code snippet; added "Implementing your own backend" guide with backend-primitive mapping table; rewrote "Features" section to lead with Store interface and contract test suite.
  - `CONTRIBUTING.md`: added "Scope" section stating backends are out of scope and PRs adding backends will not be accepted.
  - `docs/DOMAIN_LANGUAGE.md`: `Store` reference updated from line numbers to symbol name.
  - `AGENTS.md`: typo fixed on line 66 (`*what`.`→`_what_`).

### Fixed

- **Stale line-number references in FEATURES.md and DOMAIN_LANGUAGE.md** — replaced all `store.go:NN-MM` references with durable symbol-name references that don't break when code shifts.

## [0.1.2] - 2026-08-07

### Fixed

- **`Record` and `CheckAndRecord` now reject non-positive TTL** — a zero or negative TTL recorded an expiry already in the past, so the key protected nothing: the next caller would also succeed, silently breaking the exactly-once guarantee that is this library's purpose. Both methods now return `ErrInvalidTTL` (a `Rejection`, HTTP 400, non-retryable) for `ttl <= 0` instead of accepting a useless recording (`store.go`).
- Benchmark loops modernized to `b.Loop()` (Go 1.24+ idiom), removing gopls `bloop` warnings (`bench_test.go`).

### Changed

- README rewritten with badges, an error reference table, a retry-to-dedup flow diagram, and an honest v0.x status section.

## [0.1.1] - 2026-08-03

### Fixed

- **`Record` now checks TTL expiry** — `MemoryStore.Record` previously checked map presence only, not expiry. An expired-but-not-yet-swept key was treated as "already recorded" and `Record` was a no-op. Now expired keys are re-recorded with the fresh TTL, matching `CheckAndRecord`'s behavior (`store.go:103-114`).
- Lint compliance: `intrange` fix in `property_test.go`, `makezero` nolint in `bench_test.go`, idiomatic Go names added to `varnamelen` ignore list.

### Added

- `.golangci.yml` — lint configuration enabling 60+ linters including `exhaustruct`, `gosec`, `revive`, `misspell`, `gocritic`
- `.github/workflows/ci.yml` — CI pipeline: `go test -race`, `go vet`, `golangci-lint` on push and PR
- `bench_test.go` — benchmarks for `CheckAndRecord` (serial, contended, parallel-unique), `Seen` (hit, miss), `Record`
- `docs/DOMAIN_LANGUAGE.md` — domain glossary for CQRS idempotency terms
- Test: `Record` after expiry re-records with new TTL (`store_test.go`)
- Test: post-`Close` operations still function (`store_test.go`)
- Test: empty key and zero TTL edge cases (`store_test.go`)

### Changed

- **License switched from proprietary to MIT.**
- `doc.go` middleware reference softened to "planned, not yet implemented".
- `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `AGENTS.md` rebuilt with accurate project-specific content.
- `FEATURES.md`, `TODO_LIST.md`, `ROADMAP.md` added with honest status tracking.

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

[Unreleased]: https://github.com/larsartmann/go-idempotency/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/larsartmann/go-idempotency/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/larsartmann/go-idempotency/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/larsartmann/go-idempotency/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/larsartmann/go-idempotency/releases/tag/v0.1.0
