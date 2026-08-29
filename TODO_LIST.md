# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Release

- [ ] **Cut v0.2.0** — the v0.2.0 content (contract test suite, fuzz tests, memory benchmarks, godoc examples, ADR-001, docs reframe, `MemoryStore` deprecation) is complete on `master` and staged in CHANGELOG `[Unreleased]`; the deprecation-consistency pass is done. Tag, push, and verify pkg.go.dev. Evidence: [ROADMAP.md](ROADMAP.md) Versioning Strategy, [CHANGELOG.md](CHANGELOG.md) `[Unreleased]`, [docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md](docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md) P7.38. **Impact: High · Effort: S**

## Features

- [ ] **Implement middleware package** — `doc.go` references `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the way to wire the store into CQRS dispatch pipelines, but no code exists. This is the primary advertised integration point and currently the library is storage-only. **Blocked on module boundary decision** (same module vs separate module — see [docs/status/2026-08-07_21-31_docs-reframe-no-backends.md](docs/status/2026-08-07_21-31_docs-reframe-no-backends.md) question 2). **Impact: High · Effort: L**
- [ ] **Add `Delete` method to `Store` interface** — manual key invalidation for operational and testing use. No way to force-expire a single key currently exists. **Blocked on open questions** about interface evolution (see [ROADMAP.md](ROADMAP.md) Versioning Strategy). **Impact: Medium · Effort: S**

## Deprecation enforcement

- [ ] **Add a lint gate against new `MemoryStore` usage** — `forbidigo` (already enabled in `.golangci.yml`) or `depguard` should fail on `NewMemoryStore`/`MemoryStore{}` outside `_test.go`, so the deprecation is enforced by CI instead of by post-mortem. Evidence: [docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md](docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md) P1.8–P1.9. **Impact: Medium · Effort: S**

## Testing

- [ ] **Self-test the contract suite** — `contract/` reports `[no test files]` / 0% coverage because it is only exercised from the root package. Ship a tiny internal test-only `Store` and run `RunTests` against it (plus one negative test proving the suite catches a deliberately broken implementation). Evidence: [contract/contract.go](contract/contract.go), [docs/status/2026-08-07_22-15_execution-plan-completion.md](docs/status/2026-08-07_22-15_execution-plan-completion.md) e.5. **Impact: Medium · Effort: M**
- [ ] **Enrich the fuzz seed corpus** — `fuzz_test.go` seeds only 3 inputs per fuzz target; add empty string, unicode, very long keys, `math.MaxInt64` and negative TTL durations. Evidence: [fuzz_test.go](fuzz_test.go), [docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md](docs/status/2026-08-07_22-30_memorystore-deprecation-self-critique.md) P2.15. **Impact: Low · Effort: XS**
- [ ] **Document the context-cancellation test pattern for custom backends** — `RunTests` says backends honoring context "should be tested separately" but provides no guidance or template. Evidence: [contract/contract.go](contract/contract.go) `RunTests` doc comment, [docs/status/2026-08-07_22-15_execution-plan-completion.md](docs/status/2026-08-07_22-15_execution-plan-completion.md) b.3. **Impact: Low · Effort: S**

## Infrastructure

- [ ] **Wire a coverage badge** — CI already reports coverage and uploads the artifact (`.github/workflows/ci.yml`); choose Codecov or Coveralls and add the README badge. Evidence: [docs/status/2026-08-07_22-15_execution-plan-completion.md](docs/status/2026-08-07_22-15_execution-plan-completion.md) b.1. **Impact: Low · Effort: S**
