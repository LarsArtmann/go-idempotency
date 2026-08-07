# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Features

- [ ] **Implement middleware package** — `doc.go` references `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the way to wire the store into CQRS dispatch pipelines, but no code exists. This is the primary advertised integration point and currently the library is storage-only. **Blocked on module boundary decision** (same module vs separate module — see [docs/status](docs/status/2026-08-07_21-31_docs-reframe-no-backends.md) question 2).
- [ ] **Add `Delete` method to `Store` interface** — manual key invalidation for operational and testing use. No way to force-expire a single key currently exists. **Blocked on open questions** about interface evolution (see ROADMAP.md Versioning Strategy).

## Done (this session)

- [x] ~~**Deprecate `MemoryStore`**~~ — Done. `// Deprecated:` doc comments added to `MemoryStore` and `NewMemoryStore` in `store.go`. All docs (README, FEATURES, ROADMAP, doc.go, AGENTS) updated. Removal targeted for v1.0 (see ROADMAP.md).
- [x] ~~**Add `Store` interface contract test**~~ — Done. `contract/contract.go` with `RunTests(t, factory)`, run against `MemoryStore` in `contract_test.go`.
- [x] ~~**Add fuzz tests**~~ — Done. `FuzzCheckAndRecord`, `FuzzRecord` in `fuzz_test.go`. Also added `TestMemoryStore_CloseDuringConcurrentOps` and memory benchmarks.
