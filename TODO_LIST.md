# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

> Standing rule (added 2026-11-03): a plan older than two weeks with zero tasks executed gets executed or archived — never left to rot. The 2026-10-19 pareto plan's fate decision lives at the bottom of the plan file.

## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped in v0.3.0 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: the owner confirmed the deferral 2026-09-05; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**
- [ ] **Propagate `Forever` to go-cqrs-lite** — `metaengine.DedupStore`/`claimkit` and `MapDedupStore` must map the sentinel (store MaxInt64/NULL, never `now.Add`), and `adttest.AssertDedupStore` must pin the mapped semantics; consume `github.com/larsartmann/go-idempotency@v0.4.0`, which carries `Forever`. **Impact: High · Effort: M**

## Consumer test kit (`idemtest`) follow-ups

_Backlog from the kit's 2026-11-17 status report and the v0.5.0 release verification._

- [ ] **Decide drill counter semantics** — each drill asserts its `Executions` counter from zero (`Total() == 1`), and the v0.5.0 README quick start initially shared one counter across drills (caught by the clean-room smoke test, fixed in the README). Either document "one counter per drill" in `idemtest/doc.go` explicitly, or snapshot the baseline inside each drill so shared counters work (behavior change → `idemtest/v0.1.1`). **Impact: Medium · Effort: S**
- [ ] **Strengthen `RequireAllowsAfterExpiry`** — assert a duplicate inside the TTL window before polling for reclaim, so a too-short claim (instantly freed key) fails the drill instead of passing. **Impact: High · Effort: S**
- [ ] **Timing-scale option for the expiry drill** — the fixed 100ms poll grace is a CI flake risk on loaded runners; mirror the contract suite's `TimingScale`. **Impact: Medium · Effort: S**
- [ ] **`SendConcurrently` `RequestOption` support** — `idemtest/http.go` currently pins request defaults for the concurrent burst. **Impact: Low · Effort: S**
- [ ] **Assert failure-message fragments in the sabotage harness** — `idemtest/negative_test.go`'s `captureTB` captures `msg` but never asserts on it; asserting the violation name makes the proofs stronger. **Impact: Medium · Effort: S**
- [ ] **Point `docs/migrating-from-memorystore.md` at `idemtest.NewStore`** — consumers who used `MemoryStore` in tests need the blessed double, not a hand-rolled map. **Impact: Medium · Effort: S**
- [ ] **Add an `idemtest` demo to `example/`** — one runnable drill walkthrough next to the contract-suite demo. **Impact: Low · Effort: S**
- [ ] **Nested-module coverage + Codecov in CI** — the race/vet/lint/tidy/govulncheck legs cover `idemtest/` since v0.5.0; the coverage steps and Codecov upload still run root-only. **Impact: Medium · Effort: S**
- [ ] **erraudit policy for the nested module** — the enforced `--enforce-go-error-family` run gates the root only; decide whether `idemtest` joins it. **Impact: Low · Effort: S**
- [ ] **Migrate middleware self-tests onto kit doubles** — pending the owner question from the 2026-11-17 20:51 report: may `middleware`'s own tests import `idemtest` (dedup of `failingStore`/barrier harness), or stay self-contained? **Impact: Low · Effort: M**
- [ ] **Harvest the dedup-session backlog** — route `docs/status/2026-11-17_21-20_dedup-all-session-status.md` section (f) through docs-health HARVEST into TODO_LIST/ROADMAP. **Impact: Medium · Effort: S**

## Documentation

- [ ] **DOMAIN_LANGUAGE.md upgrades (plan T5)** — Where-citation column for every term, Bounded contexts section (Store / middleware / contract vocabulary), Deprecated terms section (MemoryStore); the go-etag glossary pattern. **Impact: Medium · Effort: M**
- [ ] **MemoryStore v1.0 removal mini-staircase (plan T7)** — preconditions, steps, and the major-bump plan modeled on go-etag's validated v1.0.0 deletion plan, plus a fleet `rg MemoryStore` usage audit; link from the ROADMAP v1.0 bullet. **Impact: High · Effort: M**

## Quality

- [ ] **Benchmark baselines (plan T8)** — capture `-count=6 -benchmem` timing and memory baselines to `reports/bench/<date>_<name>.txt`; unblocks the ROADMAP lock-strategy work, which is meaningless without baselines. **Impact: Medium · Effort: M**
- [ ] **Spec-citation pass over the 18 contract invariants (plan T9)** — each invariant cites its authority (TOCTOU, claim semantics, ADR); verify the negative-test README rows stay 1:1. **Impact: Medium · Effort: M**
- [ ] **Visual browser render check of the 2026-11-03 review HTML** — anchors and tag balance were validated programmatically; one human render pass confirms the Bauhaus layout renders as intended. **Impact: Low · Effort: S**

## Infrastructure

- [ ] **Coverage floor once Codecov is active** — fail CI under ~90% so erosion is loud. Keep the Codecov threshold step drafted as a ready-to-flip diff (plan T10). Blocked on the owner setting `CODECOV_TOKEN`. **Impact: Medium · Effort: S**
- [ ] **Announce v0.5.0** (owner preference) — channel and timing are the owner's call; the consumer test kit (`idemtest`, its own module) plus the 18-invariant contract suite are the story. Supersedes the un-executed v0.4.0 announcement row (permanent claims are part of the same narrative). **Impact: Low · Effort: S**
- [ ] **Confirm the next weekly scheduled CI run is fully green** — the 2026-11-16 scheduled run proved the `GOTOOLCHAIN` fix live: `Test (oldstable)` passed with `GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE: 3`. That run's Docs job still failed, but only on the private-upstream issue links in `AGENTS.md`, which `44077357195` removed; no scheduled run has executed since. Watch the 2026-11-23 run. **Impact: Medium · Effort: S**

## Cleanup

- [ ] **ROADMAP demand-gated annotations (plan T12)** — annotate demand-gated items with the 2026-10-19 fleet evidence; re-confirm the YAGNI gates. **Impact: Low · Effort: S**

<!-- The v0.3.0 release train, ADR settling, plan archival, fuzz-budget raise, and post-release stale-refs refresh were executed 2026-09-05 — see CHANGELOG [0.3.0] and the archived pareto plan. The 2026-10-19 plan's head (T1–T4, T6, T11 plus both GitHub-issue doc fixes) was executed 2026-11-03 — see CHANGELOG [0.3.2] and the fate decision in docs/planning/2026-10-19_12-17_release-hardening-and-go-etag-adoption-pareto-plan.md. v0.3.2 was released 2026-11-03 through the new gate (12/12 green pre- and post-tag) and release.yml (first live fire: success); release-watch retired. -->
