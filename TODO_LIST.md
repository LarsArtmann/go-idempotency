# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped in v0.3.0 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: the owner confirmed the deferral 2026-09-05; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**

## Infrastructure

- [ ] **Coverage floor once Codecov is active** — fail CI under ~90% so erosion is loud. Blocked on the owner setting `CODECOV_TOKEN`. **Impact: Medium · Effort: S**
- [ ] **Announce v0.3.0** (owner preference) — channel and timing are the owner's call; the middleware package plus the contract-suite detection proofs are the story. **Impact: Low · Effort: S**

<!-- The v0.3.0 release train, ADR settling, plan archival, fuzz-budget raise, and post-release stale-refs refresh were executed 2026-09-05 — see CHANGELOG [0.3.0] and the archived pareto plan. -->
