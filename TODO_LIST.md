# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Release

- [ ] **Cut v0.3.0** per [RELEASING.md](RELEASING.md) — content is staging in CHANGELOG `[Unreleased]` (middleware, RunTestsStrict, proven-detection negative tests, runnable example, governance files, recipes, docs batches). Gated on the owner's ADR veto window; finalize the changelog against `git log v0.2.0..HEAD` and verify pkg.go.dev renders the new packages. **Impact: High · Effort: S**

## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped 2026-08-29 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: owner raised the domain concern that claim invalidation must not become request-path API; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**

## Infrastructure

- [ ] **Raise the CI fuzz budget 30 s → 2–5 min per target** once a clean soak week exists (the 4×15 min soak of 2026-08-29 started that clock; re-verify no findings before raising). **Impact: Low · Effort: S**
- [ ] **Coverage floor once Codecov is active** — fail CI under ~90% so erosion is loud. Blocked on the owner setting `CODECOV_TOKEN`. **Impact: Medium · Effort: S**
- [ ] **Review Dependabot PRs** (e.g. actions/checkout 4.4.0 → 7.0.1, PR #6) — owner decision; refreshes also clear the Node 20 runner warnings. **Impact: Low · Effort: S**
- [ ] **After the ADR veto window closes**: flip provisional ADR statuses to settled and archive executed plans to `docs/planning/archived/`. **Impact: Low · Effort: S**
- [ ] **After v0.3.0 is tagged**: re-run `scripts/check-stale-refs.sh` pattern review against new release-status phrases (the checker's patterns must learn each new "stale once tagged" wording), and flip the README middleware link back to its pkg.go.dev page. **Impact: Low · Effort: S**
