# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Features

- [ ] **Implement middleware package** — `doc.go:26-31` references `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the way to wire the store into CQRS dispatch pipelines, but no code exists. This is the primary advertised integration point and currently the library is storage-only.
- [ ] **Add fuzz tests** — `FuzzCheckAndRecord`, `FuzzRecord` with arbitrary keys, TTLs, and concurrency patterns. Go native fuzzing is a natural fit for this library.
- [ ] **Add `Delete` method to `Store` interface** — manual key invalidation for operational and testing use. No way to force-expire a single key currently exists.
- [ ] **Add `Store` interface contract test** — table-driven test suite that any `Store` implementation must pass. Run against `MemoryStore` now; reuse when Redis/SQL backends ship.
