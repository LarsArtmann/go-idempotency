# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Release


## Features

- [ ] **Implement middleware package** — `doc.go` references `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the way to wire the store into CQRS dispatch pipelines, but no code exists. This is the primary advertised integration point and currently the library is storage-only. **Blocked on module boundary decision** (same module vs separate module — see [docs/status/2026-08-07_21-31_docs-reframe-no-backends.md](docs/status/2026-08-07_21-31_docs-reframe-no-backends.md) question 2). **Impact: High · Effort: L**
- [ ] **Add `Delete` method to `Store` interface** — manual key invalidation for operational and testing use. No way to force-expire a single key currently exists. **Blocked on open questions** about interface evolution (see [ROADMAP.md](ROADMAP.md) Versioning Strategy). **Impact: Medium · Effort: S**

## Testing

## Infrastructure

- [ ] **Wire a coverage badge** — CI already reports coverage and uploads the artifact (`.github/workflows/ci.yml`); choose Codecov or Coveralls and add the README badge. Evidence: [docs/status/2026-08-07_22-15_execution-plan-completion.md](docs/status/2026-08-07_22-15_execution-plan-completion.md) b.1. **Impact: Low · Effort: S**
