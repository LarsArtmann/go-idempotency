# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

> Standing rule (added 2026-11-03): a plan older than two weeks with zero tasks executed gets executed or archived — never left to rot. The 2026-10-19 pareto plan's fate decision lives at the bottom of the plan file.

## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped in v0.3.0 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: the owner confirmed the deferral 2026-09-05; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**
- [ ] **Propagate `Forever` to go-cqrs-lite** (`repo: go-cqrs-lite`) — `metaengine.DedupStore`/`claimkit` and `MapDedupStore` must map the sentinel (store MaxInt64/NULL, never `now.Add`), and `adttest.AssertDedupStore` must pin the mapped semantics; consume `github.com/larsartmann/go-idempotency@v0.6.0`. **Deferred 2026-11-17 to its own cross-repo session** — it touches another repo's store engines and its own release train; doing it inline here would half-land a cross-repo change. **Impact: High · Effort: M**

## Consumer test kit (`idemtest`) follow-ups

_Backlog from the kit's 2026-11-17 status report and the v0.5.0 release verification; being executed in the v0.6.0 session._

- [ ] **Decide drill counter semantics** — each drill asserts its `Executions` counter from zero (`Total() == 1`), and the v0.5.0 README quick start initially shared one counter across drills (caught by the clean-room smoke test, fixed in the README). Document "one counter per drill" in `idemtest/doc.go` explicitly. **Impact: Medium · Effort: S** (source: `docs/status/2026-11-17_21-20_…idemtest…` §f; `idemtest/doc.go`)
- [ ] **Strengthen `RequireAllowsAfterExpiry`** — assert a duplicate inside the TTL window before polling for reclaim, so a too-short claim (instantly freed key) fails the drill instead of passing; add the clock-aware drill variant (pinned clock, no wall-clock grace) and, if it lands, expose the blessed clock helper to consumers. **Impact: High · Effort: S** (source: `docs/status/2026-11-17_22-17_withclock…` §f item 8)
- [ ] **`SendConcurrently` `RequestOption` support** — `idemtest/http.go` currently pins request defaults for the concurrent burst; assess and either implement or decide-no with rationale. **Impact: Low · Effort: S**
- [ ] **Assert failure-message fragments in the sabotage harness** — `idemtest/negative_test.go`'s `captureTB` captures `msg` but never asserts on it; asserting the violation name makes the proofs stronger. **Impact: Medium · Effort: S**
- [ ] **Nested-module coverage + Codecov in CI** — the race/vet/lint/tidy/govulncheck legs cover `idemtest/` since v0.5.0; the coverage steps and Codecov upload still run root-only. **Impact: Medium · Effort: S**
- [ ] **erraudit policy for the nested module** — the enforced `--enforce-go-error-family` run gates the root only; wire `idemtest` into the same policy. **Impact: Low · Effort: S**

## Documentation

- [ ] **Middleware docs: deterministic-testing line** — one sentence where the middleware package doc shows store wiring: pin `MemoryStore`'s clock with `WithClock` in tests (`docs/status/2026-11-17_22-17…` §f item 42). **Impact: Low · Effort: S**
- [ ] **Annotate pre-`idemtest` status reports** — sweep `docs/status/` for reports claiming "no idemtest package exists" and resolve those items inline per docs-health ANNOTATE (non-destructive markers, not rewrites). **Impact: Low · Effort: S**

## Quality

- [ ] **art-dupl full-sensitivity pass + `idemtest` clone adjudication** — re-run `art-dupl -t 1 --suggest-generics`, adjudicate the unidentified 11th group and the post-`idemtest` candidates (`drill.go`/`http.go` `mu.Lock` generics, `tb.Helper` prologue clones); diff against the 2026-11-17 group set — any NEW group is a regression to catch (`docs/status/2026-11-17_21-20_dedup-all…` §f items 1–3, 30). **Impact: High · Effort: S**
- [ ] **Verify art-dupl HTML-template accept-comment leak upstream** — minimal repro; file upstream if real (`…dedup-all…` §f item 4). **Impact: Medium · Effort: M**
- [ ] **Verify golines writer-vs-checker `//nolint` gap** — 3-line repro; file to BuildFlow/golines or document as known behavior (`…dedup-all…` §f item 5). **Impact: Medium · Effort: M**
- [ ] **Fuzz target for non-monotonic clocks** — drive a jittery/backward clock through `CheckAndRecord` invariants via `WithClock` (`…withclock…` §f item 10). **Impact: Medium · Effort: S**
- [ ] **Visual browser render check of the 2026-11-03 review HTML** — anchors and tag balance were validated programmatically; one human render pass confirms the Bauhaus layout renders as intended. **Impact: Low · Effort: S** (human-gated)

## Infrastructure

- [ ] **art-dupl baseline gate decision** — harden the recurring dedup acceptance into an enforced baseline gate (committed baseline, CI fails on any new group — the go-etag pattern), or keep prose `art-dupl:accept` markers. Gated on the owner question in `…dedup-all…` §g item 2. **Impact: High · Effort: M**
- [ ] **Close the three red docs-gates** — `annotate`, `freshness --audit-all`, `cadence` (`…dedup-all…` §f items 7–9): run `nix run github:Artmann-Technologies/docs-gates -- run .`, resolve the verdicts. **Impact: High · Effort: M**
- [ ] **Rebuild the buildflow binary** — doctor: built `3eae2b6`, HEAD `d85812b`; results may lag until rebuilt (`…dedup-all…` §f item 11). **Impact: Medium · Effort: S**
- [ ] **Coverage floor once Codecov is active** — fail CI under ~90% so erosion is loud. Keep the Codecov threshold step drafted as a ready-to-flip diff (plan T10). Blocked on the owner setting `CODECOV_TOKEN`. **Impact: Medium · Effort: S**
- [ ] **Announce v0.6.0** (owner preference) — channel and timing are the owner's call; the story: injectable `MemoryStore` clock (`WithClock`) + deterministic expiry tests, on top of the consumer test kit (`idemtest`, its own module) and the 18-invariant contract suite. Supersedes the v0.5.0/v0.4.0 announcement rows (one narrative, current tag). **Impact: Low · Effort: S**
- [ ] **Confirm the next weekly scheduled CI run is fully green** — the 2026-11-16 scheduled run proved the `GOTOOLCHAIN` fix live; that run's Docs job failed only on private-upstream issue links (removed by `44077357195`). Watch the 2026-11-23 run: strict-timing pass at `TimingScale: 3`, lychee clean, and the `release-${{ github.ref }}` concurrency group not deadlocking the tag-push-then-rerun flow. **Impact: Medium · Effort: S**

## Cleanup

- [ ] **ROADMAP demand-gated annotations (plan T12)** — annotate demand-gated items with the 2026-10-19 fleet evidence; re-confirm the YAGNI gates. **Impact: Low · Effort: S**
- [ ] **dprint-on-Markdown enforcement** — Markdown formatting is convention-only; consider a CI job or BuildFlow step (`…withclock…` §f item 43; needs an owner stance on fleet convention vs this-repo-only). **Impact: Low · Effort: S**

<!-- Executed 2026-11-17 (withclock/issue-14 session): WithClock shipped (store.go + clock.go after the 350-line split), 23 expiry tests converted to fakeClock/synctest bubbles, bench metric int64 fixes, T5 DOMAIN_LANGUAGE rewrite, T7 removal staircase (docs/planning/2026-11-17_memorystore-v1.0-removal-staircase.md, fleet consumer audit), T8 bench baselines (reports/bench/2026-11-17_withclock-v0.6.0-baseline.txt), T9 spec-citation pass (18/18 README↔suite), migrating-doc idemtest.NewStore pointer, cache policy → Go defaults. Decided NO: idemtest demo in example/ (root go.mod would need a self-referential require of its own nested module — proxy-pinned dev drift for zero demo value); middleware self-tests onto kit doubles (same self-require cascade, test-only benefit); internal/teststore clock seam (a seam would let tests cheat the very timing the contract suite verifies — stays wall-clock by design). Environment chores from the dedup report (buildflow.db VACUUM, dprint wasm plugins, GOEXPERIMENT check) dropped as fleet-env, not repo work. -->
