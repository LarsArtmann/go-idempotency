# Status: MemoryStore Injectable Clock (Issue #14) — Session Report

**Date:** 2026-11-17 22:17 CET
**Session scope:** Execute and verify [issue #14](https://github.com/LarsArtmann/go-idempotency/issues/14) — injectable `MemoryStore` clock, deterministic expiry tests.
**Head:** `47e0ab0` (auto-commit daemon captured all work; working tree clean; **not pushed**).

## Scope line — what this report verified vs. did not

**Verified this session (command / artifact):** `go test ./... -race -count=1` all-green (root + contract + middleware + example, 1.2–1.4s each) · `go test -race -count=5` on all converted tests, stable · `go vet ./...` clean · `golangci-lint run ./...` 0 issues · `gofmt -l` clean · `go mod tidy` diff clean · `dprint check` clean · `./scripts/check-stale-refs.sh` clean · fuzz smoke 2×10s (`FuzzCheckAndRecord` 1.33M execs, `FuzzConcurrentMixed` 1.11M execs, PASS) · `BenchmarkMemoryUsage_AfterSweep -benchtime=3x` works (30.58 %-reclaimed) · `idemtest/` `go test ./... -race` green (proxy-resolved, decoupled from local root changes) · file-size caps (largest touched file: store.go 339/350).

**NOT verified (and why):** GitHub issue state (not pushed — no remote action without instruction) · full 3-minute CI fuzz pass (CI-only) · `erraudit` (skipped: no error-handling code touched) · `govulncheck` local (CI-covered) · the scheduled strict-timing contract pass at `TimingScale: 3` (contract suite untouched, but unobserved) · the anomalous 10KKeys benchmark reading (noticed, not investigated — see d-3) · pre-release gate `pre-release-check.sh` (needs a tag/CHANGELOG-parity context; no release cut).

---

## a) FULLY DONE

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | **Injectable clock API** — `now func() time.Time` field, `WithClock(func() time.Time) MemoryStoreOption`, variadic `NewMemoryStore(sweepInterval, opts...)`, nil-clock backstop, options applied before the sweeper goroutine starts | `store.go:162` (field), `store.go:182` (`WithClock`), constructor ~`store.go:210` |
| 2 | **All four expiry sites read the injected clock** — `Seen` lazy-delete, `Record` re-record, `CheckAndRecord` claim, `sweep` expiry comparison; the sweep ticker deliberately stays wall-clock (documented at the site) | `store.go` (3× `s.now()` + sweep); verified by converted tests failing without the override |
| 3 | **`fakeClock` test double** — thread-safe (`sync.Mutex`), `Advance`, `Calls`, `awaitCallAfter` (bounded poll, 2s deadline); used by 6 test files | `fakeclock_test.go` (68 lines) |
| 4 | **10 expiry-path sleeps converted to deterministic clock advances** — the issue's table listed 8; the census found 10 (missed: `store_test.go:235`, three `store_forever_test.go` sites, `property_forever_test.go:34`) | `store_test.go`, `store_forever_test.go`, `store_lifecycle_test.go`, `property_test.go`, `property_forever_test.go`, `bench_test.go` |
| 5 | **Sweep tests upgraded from "sleep and hope" to proven-tick awaits** — advance pinned clock, capture `clock.Calls()` baseline, await a real ticker tick, then assert: the sweep (not lazy deletion) demonstrably reclaimed | `store_lifecycle_test.go:20-43`, `store_forever_test.go:118+`, `bench_test.go` `AfterSweep` |
| 6 | **Property tests draw TTLs up to 1 hour** — the issue's core complaint ("hour-scale TTLs cannot test expiry at all") directly fixed | `property_test.go` `TTLExpiry`, `property_forever_test.go` |
| 7 | **`TestMemoryStore_CloseDuringConcurrentOps`** — Close now happens after a *guaranteed* ≥1 op (bounded poll), not after a guessed 20ms sleep | `fuzz_test.go` |
| 8 | **Speed + flake resistance** — converted tests run 0.00–0.02s (was 40–120ms each); stable at `-count=5` under race detector | session command output |
| 9 | **Issue claims verified against the checkout before building** (verify-external-claims) — line numbers had drifted since v0.3.2, substance confirmed; the 8-sleep table found incomplete | session read of all 8 cited sites + census |
| 10 | **Docs updated in the same change** — CHANGELOG Unreleased entry, FEATURES row, README feature line, AGENTS.md (MemoryStore bullet + twelve-file test list + `fakeclock_test.go` entry) | `CHANGELOG.md`, `FEATURES.md`, `README.md:152`, `AGENTS.md` |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
| - | ---- | ---- | ------- |
| 1 | **`WithClock(nil)` + constructor backstop** | Implemented and doc-commented ("nil clock is ignored") | **Zero dedicated tests** for nil-clock fallback, multi-option ordering, or the constructor backstop path — the only new public API with no direct test |
| 2 | **Equality boundary semantics** | Property now advances *exactly* `ttl`, pinning "at `now == expiresAt` the claim is expired" for MemoryStore | The contract suite does not pin equality (`AllowsAfterExpiry` sleeps past); the semantic decision was made implicitly by me, not confirmed by the owner |
| 3 | **Non-monotonic clock guidance** | Clock is injectable; tests only advance forward | `WithClock` docs carry no warning for backward-moving injected clocks; no test covers a clock that jumps back |
| 4 | **10KKeys benchmark anomaly** | Noted `1844674407370905 bytes/key` (absurd ≈2^61 value) in the `-benchtime=3x` run | Not investigated, not flagged in my final message (should have been mentioned as an unrelated observation) |
| 5 | **Clock seam propagation to the test stack** | `MemoryStore` injectable; `internal/teststore` and `idemtest` unchanged | The kit's `RequireAllowsAfterExpiry` still polls real time; the existing TODO "timing-scale option for the expiry drill" is now adjacent to a possible clock-aware design — not decided |

## c) NOT STARTED

| # | Item | Note |
| - | ---- | ---- |
| 1 | GitHub issue #14 comment + close | Nothing posted remotely; implementation evidence (commit SHAs, gate results) exists only locally. Not pushed, per harness rule |
| 2 | Dedicated `WithClock` option-semantics unit tests | nil-clock, ordering, backstop (see b-1) |
| 3 | Release path | Exported API added → next root tag must be ≥ v0.6.0 (minor). No decision recorded, no `pre-release-check.sh` run, no TODO_LIST row |
| 4 | `doc.go` / `DOMAIN_LANGUAGE.md` touch | Deliberately skipped — arguably a "pinned clock" glossary row and one Quick-Start sentence are warranted; unexamined |
| 5 | `docs/migrating-from-memorystore.md` note | Dev/test users keeping `MemoryStore` could learn about `WithClock` there |
| 6 | TODO_LIST harvest of this report's section (f) | Report is the input; harvest is the follow-up session step |

## d) TOTALLY FUCKED UP

**Nothing in the shipped work is broken.** All gates green, working tree clean, no data lost, nothing reverted. Honest near-misses, for the record:

| # | Near-miss | Severity | Resolution |
| - | --------- | -------- | ---------- |
| 1 | One `multiedit` failed outright (escaped-tab typo in `old_string`), one edit batch rejected for not having viewed the files via the view tool first | Process stumble, zero damage | Self-caught immediately; corrected with exact strings |
| 2 | Observed the 10KKeys benchmark emitting a garbage `bytes/key` metric and moved on without investigating or mentioning it | Small (pre-existing, benchmark-only, not touched by this change) | Recorded here; investigation queued (f-13) |
| 3 | The final summary claimed "10 expiry sleeps" converted while my own headline table earlier said 8 — both true (issue said 8, census found 10), but the count flip-flopped mid-report without one crisp statement up front | Cosmetic reporting | This report states the census once, with the missed sites named |

## e) WHAT WE SHOULD IMPROVE (brutal self-review)

1. **Forgot the test for the thing I shipped.** `WithClock` is new public API; every converted test *incidentally* proves the clock is wired (they fail against the wall clock), but nil-fallback, option ordering, and the backstop have no direct test. That is exactly the kind of gap the library's own negative-test convention exists to prevent — and the convention formally binds only `contract/` invariants and `idemtest` drills, so nothing forced me. It should have been obvious anyway.
2. **Forgot to flag an anomaly I saw.** The 10KKeys metric garbage violated my own rule ("mention unrelated issues found"). 
3. **Made a semantics decision silently.** Equality-boundary expiry (b-2) is now enshrined in a property test without an owner decision. A reviewer could read it as intended contract; it may not be.
4. **Missed a connection to existing backlog.** TODO_LIST already carried "timing-scale option for the expiry drill" (CI flake risk); with `WithClock` landed, a clock-aware kit design is now the better-shaped answer — I noticed this only while writing this report.
5. **Split-brain check (passed, with a watch item):** liveness predicates exist in three in-repo places (`MemoryStore.isLive`, `internal/teststore` copy — deliberately mirrored, `art-dupl:accept`-marked, `example`). This change did not add a fourth; it added a clock seam only in `MemoryStore`, so the two test stores now *differ* in testability (MemoryStore injectable, teststore not). Not drift yet — it is a divergence by design choice worth an explicit decision (f-7).
6. **Ghost systems (none):** `WithClock` is wired into tests the moment it landed; `fakeClock` is used by six files; nothing dead. No capability shipped unwired.
7. **Scope discipline (held, mostly):** resisted a clock *interface* (kept `func() time.Time` per the issue), resisted propagating into `contract/` and `internal/teststore`, left the ticker real. The one scope growth — drawing TTL magnitudes into the two properties — is on-theme and strictly stronger coverage.
8. **Removed nothing useful:** sleeps were replaced by deterministic or bounded-proven waits; every old assertion survives, several got stronger (sweep participation now provable, TTL magnitudes now covered).
9. **Did I lie anywhere?** Checked: "0 issues" (lint, verified), "stable at -count=5" (verified), "1.3M execs" (verified), "idemtest decoupled" (verified: no `replace` in `idemtest/go.mod`). One imprecision: the benchmark output's bytes/key figure for 10KKeys was nonsense and I called the benchmark run "healthy" based on the AfterSweep row alone — overstated; corrected here.
10. **Process:** the auto-commit daemon committed incrementally mid-work (7-file and 2-file commits before the final 11-file one), so HEAD history fragments the change. Harmless here, but it means `git log` alone is not a clean review unit — the CHANGELOG entry is the real summary.

## f) UP TO 50 THINGS TO GET DONE NEXT

*Brainstorm, not commitment — ROADMAP/TODO_LIST fuel; route through docs-health HARVEST.*

**Direct follow-ups on this session's work:**
1. Unit test: `WithClock(nil)` leaves the wall clock in place (b-1)
2. Unit test: option ordering — last `WithClock` wins when applied twice
3. Unit test: constructor backstop — a raw option setting `now = nil` still yields a working store
4. Document (or reject) non-monotonic injected clocks in `WithClock`'s doc (b-3)
5. Owner decision + pin the equality boundary (`now == expiresAt` → expired?) — align contract suite if pinned (b-2)
6. Extract the repeated advance→baseline→await dance in sweep tests into one `fakeClock` helper (4 call sites, art-dupl candidate)
7. Decide whether `internal/teststore` gets the same clock seam (kills the e-5 divergence) or stays wall-clock by contract-suite design
8. idemtest: clock-aware `RequireAllowsAfterExpiry` variant or timing-scale option (pre-existing TODO, now reshaped)
9. idemtest: expose a blessed fake-clock helper for consumer tests if (8) lands
10. Fuzz target driving a jittery/backward clock through `CheckAndRecord` invariants
11. Run `erraudit . --enforce-go-error-family` once for parity (cheap, skipped this session)
12. Local `govulncheck` run for CI parity
13. Investigate the 10KKeys `bytes/key` anomaly (negative HeapInuse delta → float64?) (b-4)
14. Capture benchmark baselines (TODO T8) — noting `AfterSweep`'s mechanism/timing changed this session
15. Push + comment on issue #14 with evidence (commit SHAs, gates); owner ticks the review checkbox
16. Release decision: cut v0.6.0 carrying `WithClock` (CHANGELOG section ready) via `pre-release-check.sh`
17. After release: decide whether `idemtest` bumps its root require past v0.4.0 (needed only if kit adopts clock helpers)
18. Add a "pinned clock / injectable clock" row to `docs/DOMAIN_LANGUAGE.md` Testing table
19. One `doc.go` Quick-Start sentence: pin `MemoryStore`'s clock for deterministic tests
20. Note `WithClock` in `docs/migrating-from-memorystore.md` (dev/test keepers)
21. Godoc example block for `WithClock` (pkg.go.dev rendering)
22. Watch `store.go` size (339/350) — split `clock.go` if it grows again
23. Verify next scheduled CI run's strict-timing pass is unaffected (contract suite untouched, but confirm)

**Pre-existing TODO_LIST items seen this session (unchanged, for completeness):**
24. idemtest: decide drill counter semantics (per-drill vs shared)
25. idemtest: strengthen `RequireAllowsAfterExpiry` (assert duplicate inside the window first)
26. idemtest: `SendConcurrently` `RequestOption` support
27. idemtest: assert failure-message fragments in the sabotage harness
28. Point `docs/migrating-from-memorystore.md` at `idemtest.NewStore`
29. idemtest demo walkthrough in `example/`
30. Nested-module coverage + Codecov in CI
31. erraudit policy decision for the nested module
32. Migrate middleware self-tests onto kit doubles (blocked on owner question)
33. Harvest the 2026-11-17 dedup-session status backlog
34. DOMAIN_LANGUAGE.md upgrades: where-citation column, bounded contexts (plan T5)
35. MemoryStore v1.0 removal mini-staircase (plan T7)
36. Benchmark baselines (plan T8 — same as 14)
37. Spec-citation pass over the 18 contract invariants (plan T9)
38. Visual render check of the 2026-11-03 review HTML
39. `EventIdempotency`/`QueryIdempotency` (YAGNI-gated, ADR-002)
40. Revisit `Delete(ctx, key)` on `Store` (ADR-004-gated)
41. Propagate `Forever` to go-cqrs-lite (`metaengine.DedupStore`, `claimkit`, `adttest`)
42. Middleware docs: mention deterministic `MemoryStore` testing where the package shows store wiring examples
43. Consider a CI job (or BuildFlow step) enforcing dprint on Markdown — md formatting is currently convention-only
44. Long-fuzz pass locally (30s × all four targets) before the next release, mirroring CI's 3m/target
45. Introduce a tiny `Clock` interface only when a second clock consumer appears (explicit YAGNI marker)
46. README "Testing" section: one line on deterministic expiry testing with `WithClock`
47. ROADMAP: record the clock seam as the test-ergonomics precedent for future `EventIdempotency`
48. Cross-check AGENTS.md "Known Tool Findings" for the `wsl_v5` blank-line-before-`defer` rule so future edits pre-empt it
49. Run docs-health VERIFY over this report's claims after the next push (evidence: session artifacts age)
50. Close the loop: HARVEST this section (f) into TODO_LIST/ROADMAP

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — Equality boundary:** Is "a claim expires **at** the exact expiry instant (`now == expiresAt` → expired)" the intended contract? My property test now pins this for `MemoryStore` (advance exactly `ttl`); the contract suite only pins *strictly past* expiry. If you intend "live through the expiry instant", I must flip the property and add a contract invariant — one-line change, wrong guess = enshrined wrong semantics.

**Q2 — Release & issue handling:** Cut **v0.6.0** now (CHANGELOG section ready, gate script available) and post the implementation evidence on #14 — or batch `WithClock` with upcoming work? I will not push, comment, or tag without your word.

**Q3 — Clock seam propagation:** Should the injectable clock reach the blessed test stack (`internal/teststore` + a clock-aware `idemtest.RequireAllowsAfterExpiry`, which would resolve the existing "timing-scale for the expiry drill" TODO)? That is a public `idemtest` API decision (separate module → `idemtest/v0.1.1+`), so it needs your YAGNI call.

---

*Awaiting instructions.*
