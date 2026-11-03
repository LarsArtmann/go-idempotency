# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

> Standing rule (added 2026-11-03): a plan older than two weeks with zero tasks executed gets executed or archived — never left to rot. The 2026-10-19 pareto plan's fate decision lives at the bottom of the plan file.

## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped in v0.3.0 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: the owner confirmed the deferral 2026-09-05; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**

## Documentation

- [ ] **DOMAIN_LANGUAGE.md upgrades (plan T5)** — Where-citation column for every term, Bounded contexts section (Store / middleware / contract vocabulary), Deprecated terms section (MemoryStore); the go-etag glossary pattern. **Impact: Medium · Effort: M**
- [ ] **MemoryStore v1.0 removal mini-staircase (plan T7)** — preconditions, steps, and the major-bump plan modeled on go-etag's validated v1.0.0 deletion plan, plus a fleet `rg MemoryStore` usage audit; link from the ROADMAP v1.0 bullet. **Impact: High · Effort: M**

## Quality

- [ ] **Benchmark baselines (plan T8)** — capture `-count=6 -benchmem` timing and memory baselines to `reports/bench/<date>_<name>.txt`; unblocks the ROADMAP lock-strategy work, which is meaningless without baselines. **Impact: Medium · Effort: M**
- [ ] **Spec-citation pass over the 13 contract invariants (plan T9)** — each invariant cites its authority (TOCTOU, claim semantics, ADR); verify the negative-test README rows stay 1:1. **Impact: Medium · Effort: M**
- [ ] **Visual browser render check of the 2026-11-03 review HTML** — anchors and tag balance were validated programmatically; one human render pass confirms the Bauhaus layout renders as intended. **Impact: Low · Effort: S**

## Infrastructure

- [ ] **Coverage floor once Codecov is active** — fail CI under ~90% so erosion is loud. Keep the Codecov threshold step drafted as a ready-to-flip diff (plan T10). Blocked on the owner setting `CODECOV_TOKEN`. **Impact: Medium · Effort: S**
- [ ] **Announce v0.3.x** (owner preference) — channel and timing are the owner's call; the middleware package plus the contract-suite detection proofs are the story. **Impact: Low · Effort: S**
- [ ] **Watch `release.yml` first live fire** — the tag-triggered release workflow is untested-by-default; watch its first run end-to-end on the next root tag (go-etag's own rule, inherited from the 2026-10-19 learning review). **Impact: Medium · Effort: S**

## Cleanup

- [ ] **ROADMAP demand-gated annotations (plan T12)** — annotate demand-gated items with the 2026-10-19 fleet evidence; re-confirm the YAGNI gates. **Impact: Low · Effort: S**

<!-- The v0.3.0 release train, ADR settling, plan archival, fuzz-budget raise, and post-release stale-refs refresh were executed 2026-09-05 — see CHANGELOG [0.3.0] and the archived pareto plan. The 2026-10-19 plan's head (T1–T4, T6, T11 plus both GitHub-issue doc fixes) was executed 2026-11-03 — see CHANGELOG [Unreleased] and the fate decision in docs/planning/2026-10-19_12-17_release-hardening-and-go-etag-adoption-pareto-plan.md. -->
