# Status Report — v0.2.0 Release Train & SDK Hardening Plan Execution

> **Snapshot:** 2026-08-29 17:33 CEST · **Branch:** `master` (clean, all pushed) · **Scope:** Full execution of [docs/planning/2026-08-29_15-18_v020-release-and-sdk-hardening-plan.md](../planning/2026-08-29_15-18_v020-release-and-sdk-hardening-plan.md) (T1–T27, 27 tasks / 146 sub-tasks) after your blanket "execute the whole list" approval.
> **Outcome in one line:** v0.2.0 is **released and verified on pkg.go.dev**, all 27 plan tasks are closed or deliberately gated (T22), CI grew from 3 jobs to 8 (all green), and the three owner decisions are now recorded ADRs — with one live question for you (Delete).

## Executive summary

| Dimension                  | Before session                                            | After session                                                                           |
| -------------------------- | --------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| Latest release             | v0.1.2 (v0.2.0 content sitting untagged since 2026-08-07) | **v0.2.0 tagged, pushed, GitHub Release published, proxy-indexed, pkg.go.dev verified** |
| CI jobs                    | 3 (test/vet/lint)                                         | **8** (+ format, tidy, fuzz 30s×2, govulncheck, docs stale-refs) — all green            |
| Deprecation enforcement    | Prose                                                     | **Mechanical** (`forbidigo` gate, negative-tested)                                      |
| Contract suite credibility | 0% self-coverage, no proof it catches bugs                | **Self-tested (79.6% cov) + 4 negative tests proving detection & diagnosis**            |
| Middleware                 | Referenced in docs, nonexistent                           | **`middleware` package shipped** (`NewCommand` + `net/http` adapter, stdlib-only)       |
| Owner decisions            | 3 open blockers since 2026-08-07                          | **ADR-002/003/004 recorded** (Delete deferred pending your confirmation)                |
| Commits / tag / release    | —                                                         | 28 commits (`ac4793a…7174ce4`), 1 annotated tag, 1 GitHub Release                       |

---

## a) FULLY DONE

### P0 — v0.2.0 release train (T1–T4)

1. **T1 · forbidigo deprecation gate** — `.golangci.yml` fails lint on any `MemoryStore`/`NewMemoryStore` outside `_test.go` (`store.go` exempt); proven with a planted violation (2 forbidigo hits), then removed. `ac4793a`
2. **T2 · pre-release verification** — CHANGELOG `[Unreleased]` diffed against `git log v0.1.2..HEAD`; two gaps filled (interface-first reframe, CI hardening); renamed `[0.2.0] - 2026-08-29` with compare links; `RELEASING.md` checklist written; release notes drafted at [docs/releases/v0.2.0-notes.md](../releases/v0.2.0-notes.md). `7752fd8`
3. **T3 · tag + GitHub Release** — annotated tag `v0.2.0` pushed; module proxy confirmed (`go list -m -versions` lists v0.2.0); Release published and marked Latest: https://github.com/LarsArtmann/go-idempotency/releases/tag/v0.2.0 ; CI green on the release commit.
4. **T4 · pkg.go.dev verification (proof captured)** — the [v0.2.0 page](https://pkg.go.dev/github.com/larsartmann/go-idempotency@v0.2.0) renders with `Deprecated:` markers on **both** the `MemoryStore` type and `NewMemoryStore`, godoc examples with output, `ErrInvalidTTL`/`ErrDuplicate` docs, and the `contract` directory. (The "not in the latest version of its module" banner is normal: master is now ahead of the tag.)

### P1 — contract credibility (T5–T9)

5. **T5 · internal test Store** — minimal in-memory `Store` mirroring MemoryStore semantics (lazy expiry, no TTL extension, atomic claim), no sweeper, no-op `Close`. `df28388`
6. **T6 · contract self-test** — `contract.RunTests` runs against it in CI; `contract/` coverage **0% → 79.6%**. `7d11b50`
7. **T7 · negative contract tests** — four deliberately broken Stores (duplicate→nil, duplicate→generic error, TTL-blind `Record`, TTL-blind `CheckAndRecord`); each scenario **re-executes the test binary in a subprocess** and asserts non-zero exit **plus** the failure text naming the violated invariant (`"want ErrDuplicate"`, `"want ErrInvalidTTL"`). `13c3ad9`, `51ad5a8`
8. **T8 · fuzz corpus enrichment** — 3 → 7 seeds per target (empty, 4 KB, unicode/emoji, `math.MaxInt64` overflow, negative TTLs); 60 s live fuzz (15.7M execs) clean. `c558241`, `c65d5d8`
9. **T9 · context-cancellation guidance** — "Testing context cancellation" godoc section with a copy-paste test (canceled call returns ctx error AND must not consume the claim); linked from README. `daaacd1`

### P2 — consumer enablement (T10–T18)

10. **T10 · response-replay recipe** — "Recipe: Dedup + Response Replay (HTTP Idempotency)" in `doc.go`; closes PapDashboard B1 (feedback doc carries a routing note; ROADMAP idea marked done). `99774d8`
11. **T11 · migration guide** — [docs/migrating-from-memorystore.md](../migrating-from-memorystore.md) with backend-choice table, complete in-process + Redis implementations, contract wiring, swap table, checklist. **Verified, not aspirational:** the guide's snippets were compiled against the _published_ v0.2.0 in a scratch module and the in-process store **passes the full contract suite under `-race`**. Linked from both `Deprecated:` notices. `2794435`
12. **T12 · contract invariant list** — README renders all 13 invariants as a per-method table (each row = one subtest); CONTRIBUTING carries the short list + sync rule. `9357430`
13. **T13 · CI hardening** — format, tidy-diff, fuzz (30 s × 2 targets), govulncheck jobs. **Verified green end-to-end** (run 33258873726: all 7 jobs ✓). `1abf86d`
14. **T14 · stale-refs checker** — `scripts/check-stale-refs.sh` + CI docs job. It immediately caught real drift from the v0.2.0 tag: 8 "reference implementation" endorsements, "unreleased; slated for v0.2.0" ×2, ROADMAP "not yet tagged", stale AGENTS CI description, stale TODO release item — all fixed before first red run. Negative-tested. `d9da384`
15. **T15 · coverage badge wiring** — Codecov action pinned by SHA, README badge added; upload later made **conditional on `CODECOV_TOKEN`** after observing Codecov reject tokenless uploads (see b-2). `31ceb55`, `7174ce4`
16. **T16 · repo governance** — SECURITY.md (private reporting + honest scope note), CODE_OF_CONDUCT (Covenant 2.1), bug/feature issue templates (evidence-oriented), PR checklist encoding the gates, CODEOWNERS. `0a51748`
17. **T17 · topics + FUNDING decision** — topics `go golang idempotency cqrs deduplication` set and verified rendering; FUNDING deliberately omitted (owner preference; recorded). `2728f4e`
18. **T18 · docs polish batch** — README "Common pitfalls" (TOCTOU, fail-closed on store errors, TTL sizing to the retry window, key namespacing, clock ownership, claimed-but-unfinished keys); backend feature matrix (Redis/Postgres/MySQL/DynamoDB/SQLite); transient-error/retry guidance via `errorfamily.IsRetryable`; Store-method godoc → contract-invariant cross-links; `doc.go` contents section; **`example/`** — `go run ./example` works and is contract-tested. `5f0f15e`

### P3/P4 — decisions + implementations (T19–T27)

19. **T19 · ADR-002 middleware boundary** — subpackage, stdlib-only, HTTP-first; split-out trigger defined (first transport dependency). Accepted (provisional). `3f228af`
20. **T20 · ADR-003 bounded store** — documented position: restart durability is table stakes; an LRU-capped claim store silently sacrifices exactly-once; **no `BoundedStore`**. `3f228af`
21. **T21 · ADR-004 interface evolution** — `Delete` **deferred** (your domain objection recorded verbatim-in-spirit: claim invalidation must not become request-path API; trigger = demonstrated poisoned-claim need), `Stats`/`Reset`/`ErrStoreClosed`/return-shape rejected or deferred with rationale. `3f228af`
22. **T23 · middleware package** — `middleware.NewCommand` (at-most-once command wrapper; sentinels pass through; fail-closed on store errors) + `middleware.HTTP` (`Idempotency-Key` header; 400 missing / 409 duplicate / 503 store failure). Tests include 50-goroutine exactly-once and 20-request concurrent HTTP races; a contract test is embedded. **Test-store consolidation:** `contract/internal` → module-root `internal/teststore` (one test store for contract, negative, and middleware tests — no split brain). `df3270d`
23. **T24 · SQL adapter example** — complete PostgreSQL adapter in `doc.go` (transactional `CheckAndRecord` via `INSERT … ON CONFLICT DO UPDATE … RETURNING` with row-lock serialization); ADR-001 supplement scopes examples to Redis+SQL (DynamoDB/Mongo declined). `4b8d3ab`
24. **T25 · testing hardening** — property test: arbitrary non-positive Durations (drawn overflow-safe) always reject with `ErrInvalidTTL` and record nothing; `errors.Is` across nested `fmt.Errorf` wrapping; goroutine-leak test polling 10 sweepers back to baseline. `5c19b27`
25. **T26 · fuzz/strict/bench batch** — `RunTestsStrict` + `Options.TimingScale` (timings threaded through the suite, `RunTests` delegates); `FuzzConcurrentMixed` (2–20 goroutines mixed ops; 20 s clean); `BenchmarkCheckAndRecord_SweepEnabled/Disabled` quantify sweeper cost (~20% on the contended workload). `81dc739`
26. **T27 · tooling cleanup** — dprint decision recorded (kept for local use, not CI-wired — Go formatting is enforced by gofmt/gofumpt/golines); **LSP fix**: `golangci_lint_ls` now gets writable cache env overrides in the global Crush config (root cause: shell env points `GOCACHE`/`GOLANGCI_LINT_CACHE` at the nonexistent `/mnt/buildcache`; gopls already overrode it, the golangci LSP didn't); go.mod bump verified committed; plan annotated as executed. `ee4a4d3`
27. **Session hygiene** — TODO_LIST, ROADMAP, AGENTS.md, CHANGELOG, CONTRIBUTING, README synced throughout; `[Unreleased]` is already staging the v0.3.0 content. Final state: **all gates green** (gofmt, vet, 0 lint issues, `go test ./... -race`, stale-refs, `go run ./example`), CI green on master, working tree clean, everything pushed.

## b) PARTIALLY DONE

1. **T15 Codecov activation** — wiring is done and green, but uploads are rejected with _"Token required — not valid tokenless upload"_ until you activate the repo on codecov.io (one-time GitHub login) or set the `CODECOV_TOKEN` secret. The step now **skips silently** until the secret exists; the badge honestly shows "unknown" rather than a fake number.
2. **LSP ghost diagnostics** — the fix is committed to your global Crush config but **takes effect next session**; this session's editor LSP still spews the `/mnt/buildcache` errors. The repo-side truth (CLI gates) is green; ignore the ghosts meanwhile.
3. **Docs sync for the newest features** — FEATURES.md and AGENTS.md's file/architecture lists predate the `middleware/` package, `internal/teststore/`, `example/`, and `RunTestsStrict` (CONTRIBUTING's file list _is_ current). Small HARVEST pass pending — logged in f-10/f-11.
4. **v0.2.0 as a frozen snapshot** — the tagged pkg.go.dev page necessarily shows the pre-hardening README ("middleware … next planned addition"). Expected behavior; next tag catches up (f-6).

## c) NOT STARTED (all deliberately gated — nothing forgotten)

1. **T22 `Delete(ctx, key)`** — deferred by ADR-004 pending your confirmation (see g-1). Implementation plan is pre-written in the ADR (semantics, contract subtests, docs framing).
2. **T24b `BoundedStore`** — rejected by ADR-003.
3. **v0.3.0 release train** — content is staging in CHANGELOG `[Unreleased]`; nothing tagged yet.
4. **`EventIdempotency`/`QueryIdempotency` middleware** — YAGNI-gated per ADR-002 until a consumer needs them.
5. **Plan Deferred table** (unchanged): semver-breaking-change CI check, Redis reference-backend repo, key-generation utilities, metrics hooks/`slog`, clock injection, All Contributors/FUNDING.
6. **Dependabot PR #6** (actions/checkout 4.4.0 → 7.0.1) opened mid-session — foreign to my task; not reviewed or merged (see g-2).

## d) TOTALLY FUCKED UP (nothing unrecoverable; honest near-misses)

1. **Pipe-masked lint gates — 3 commits landed momentarily dirty.** `golangci-lint run | tail -1` in gate chains swallowed the exit code, so `thelper`/`wsl_v5` (T6), `noctx`/`nlreturn`/`err113`/`golines` (T7), and `gosmopolitan` (T8) were committed before being noticed and fixed in follow-ups (`51ad5a8`, `c65d5d8`). Root cause fixed mid-session: all later gate checks use bare exit codes. Lesson: **never put gate commands behind a pipe without `PIPESTATUS`.**
2. **I clobbered my own uncommitted work with `git restore README.md`.** During the T14 negative test I planted a stale ref, then restored README from HEAD — which also reverted my _own uncommitted_ deprecation rewording. Caught immediately and re-applied byte-for-byte; no user work was ever touched. It still stings: the "never restore over a dirty tree" rule exists precisely because a dirty tree looks like a clean one at 17:00.
3. **The T7 negative-test harness was wrong twice** before it was right: (a) `t.Parallel()` inside the inner `t.Run` made the outer check run before the suite finished (false "not detected"), then (b) the failed-subtest-fails-ancestors rule made expected failures unobservable inside my own test tree at all. The subprocess re-exec design is the correct, deterministic solution — but it took two failed designs to reach.
4. **One unformatted commit (T9)** — `gofmt`/gci formatting slipped past the first commit; caught and **amended** within the same task. Gate chain now includes `gofmt -l .` by exit code.
5. **Unfinished work at interrupt** — the conditional Codecov step was left uncommitted when you paused me; completed and pushed as `7174ce4` before this report.

## e) WHAT WE SHOULD IMPROVE

1. **Single gate script** — `scripts/gate.sh` (fmt, vet, lint, `-race` test, stale-refs, tidy) used by me, CI, and humans, so "all gates green" is one command with exit-code semantics, never an ad-hoc chain.
2. **Commit-atomically-after-gates** — the pipe incident argues for a pre-push/pre-commit gate or, minimally, never chaining commit with `&&` after a piped check.
3. **Docs-before-tagging** — the v0.2.0 page froze pre-hardening docs; FEATURES/AGENTS lagged the newest code. Make "sync FEATURES + AGENTS file lists" an explicit RELEASING.md step.
4. **Codecov economics** — a self-hosted or token-free alternative (or Coveralls) could remove the activation dependency; Codecov is fine, but the one-time owner login is a real dependency for a "visible quality" badge.
5. **Negative-test maintenance** — each new contract invariant should add a matching broken-Store scenario (currently 4 of 13 covered); consider a comment in `contract.go` reminding the author of the negative-test pairing.
6. **`/mnt/buildcache` root cause** — the shell env still exports broken cache paths; every tool that doesn't override them will trip. Fix the environment (outside my reach) rather than adding per-tool overrides forever.
7. **Release cadence** — v0.2.0 sat 22 days complete-but-untagged. Now that RELEASING.md exists, cut releases on content-complete rather than batching.
8. **Middleware API breadth** — `NewCommand` is deliberately minimal; real usage will surface wants (key prefixes, replay hooks, per-route TTLs). Let demand drive, don't pre-build (same discipline that killed BoundedStore).

## f) Top #50 things to get done next

**Owner actions (fastest, unblocks everything else)**

| # | Task                                                                                                                                | Why now                                                                     |
| - | ----------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| 1 | Activate Codecov (GitHub login at app.codecov.io) or `gh secret set CODECOV_TOKEN`                                                  | Makes the coverage badge real; upload step self-activates                   |
| 2 | Review **Dependabot PR #6** (checkout 4.4.0→7.0.1) and merge or close                                                               | PR queue hygiene; also fixes the Node 20 runner warnings                    |
| 3 | Confirm or veto **ADR-002/003/004** (middleware boundary, no BoundedStore, Delete deferral) — they're marked provisionally accepted | Converts provisional decisions into settled ones; flip statuses in the ADRs |
| 4 | Decide **Delete**: accept per the ADR-004 trigger, or kill it outright                                                              | Gates T22 and any recovery-tooling roadmap items                            |
| 5 | FUNDING.yml: give a yes/no + mechanism, or leave omitted (current state is deliberate)                                              | Closes the last open T17 thread                                             |

**v0.3.0 release train (content is already staging in `[Unreleased]`)**

| #  | Task                                                                                                                            | Why now                                                                     |
| -- | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| 6  | Cut **v0.3.0** per RELEASING.md (middleware, RunTestsStrict, governance files, recipes, docs batches are all in `[Unreleased]`) | Ships a session's worth of consumer-facing work                             |
| 7  | Finalize changelog against `git log v0.2.0..HEAD`, set date + compare link                                                      | Keep-a-Changelog accuracy                                                   |
| 8  | Verify pkg.go.dev _latest_ renders `middleware`, `example`, `internal` hidden, cross-linked godoc                               | Post-release proof, per RELEASING.md step 12                                |
| 9  | Update **FEATURES.md** rows: middleware, RunTestsStrict, example dir, negative tests, fuzz concurrency                          | FEATURE inventory honesty (currently lags)                                  |
| 10 | Update **AGENTS.md** architecture section: `middleware/`, `internal/teststore/`, `example/`, `scripts/`                         | AGENTS is the AI-session context; it lags the tree                          |
| 11 | Verify the GitHub Release v0.2.0 page renders notes/markdown as you expect                                                      | Human eyeball I can't fully do                                              |
| 12 | Consider a short release announcement (GitHub Discussion / social) naming the enforced deprecation + contract suite             | v0.2.0 is the pitch; nobody sees it otherwise                               |
| 13 | After v0.3.0: re-run `scripts/check-stale-refs.sh` patterns against new release-status phrases                                  | The checker's patterns need to learn "not yet tagged"-class drift each time |
| 14 | Archive both executed plans to `docs/planning/archived/` once the veto window closes                                            | Keeps docs/planning = active work only                                      |
| 15 | HARVEST this report's section f into TODO_LIST (keep actionable, route the rest to ROADMAP)                                     | TODO_LIST is the living source; this report is a snapshot                   |

**Middleware evolution (demand-gated)**

| #  | Task                                                                                                | Why now                                                                   |
| -- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 16 | Opt-in **replay-on-duplicate** for `middleware.HTTP` (compose with a response store per the recipe) | The most likely real consumer ask; keep it opt-in to preserve 409 default |
| 17 | **Key-prefix option** (e.g. `WithKeyPrefix("idem:")`) for shared backends                           | Operational safety; currently documented-only                             |
| 18 | **Fuzz target for the middleware** (concurrent `Dispatch` interleavings, arbitrary keys/TTLs)       | Extends the panic/race hunt to the new hot path                           |
| 19 | Package-doc examples for `NewCommand` (godoc `Example` functions)                                   | pkg.go.dev currently shows the API without runnable examples              |
| 20 | Per-route TTLs or TTL-from-context option                                                           | Only if a consumer asks — otherwise skip                                  |
| 21 | `EventIdempotency` when an event-delivery consumer appears (ADR-002 YAGNI gate)                     | Pre-declared scope                                                        |
| 22 | `QueryIdempotency` ditto — or delete the mentions from docs entirely if nothing ever materializes   | Docs currently say "planned" in two places                                |
| 23 | gRPC adapter — only as its own module per ADR-002's split trigger; write the ADR supplement first   | Prevents accidental dependency creep into core                            |

**Contract / SDK evolution**

| #  | Task                                                                                                                                                            | Why now                                                                    |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| 24 | Implement **T22 `Delete`** if ADR-004's trigger fires (plan pre-written: semantics, 4 contract subtests, MemoryStore + teststore + negative-test updates, docs) | The one big gated feature                                                  |
| 25 | Add **negative-test scenarios** for the remaining invariants (Seen-on-live, KeysAreIndependent violation, EmptyKey mishandling)                                 | Prove-detection coverage is 4/13 invariants                                |
| 26 | Consider `contract.RunTestsContextAware` (optional suite asserting cancellation semantics for backends that honor ctx)                                          | Turns the cancellation _guidance_ into an executable option                |
| 27 | Pair each new contract invariant with its negative test in the same PR (add a CONTRIBUTING rule)                                                                | Stops detection-coverage rot                                               |
| 28 | Keep README invariant table + `contract.go` in sync mechanically (extract-and-diff in the docs CI job)                                                          | The sync rule is currently prose                                           |
| 29 | `ErrStoreClosed` revisit when a pooled backend needs it (ADR-004 trigger)                                                                                       | Pre-declared                                                               |
| 30 | Clock injection decision (deterministic TTL tests) — bundle with any interface evolution                                                                        | Pre-declared                                                               |
| 31 | Error-classification property test: `errorfamily.Classify(ErrDuplicate)==Conflict` / `IsRetryable==false` locked as API contract                                | Sentinels are public contract; only family membership is unit-tested today |

**Testing / CI**

| #  | Task                                                                                                                      | Why now                                           |
| -- | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------- |
| 32 | Raise CI fuzz budget after a soak week (30 s → 2–5 min) if zero findings                                                  | Budget vs. signal tradeoff                        |
| 33 | Exercise `RunTestsStrict{TimingScale: 3}` occasionally in CI (a scheduled job)                                            | The slow-machine path must not rot                |
| 34 | Add a Go **compatibility matrix** job (go.mod version + previous Go release)                                              | Currently single-version                          |
| 35 | Link checker for README/ADR links in the docs job (lychee or equivalent)                                                  | Docs rot includes link rot                        |
| 36 | Coverage floor check once Codecov is active (e.g. fail under 90%)                                                         | Prevents silent erosion                           |
| 37 | Watch `TestMemoryStore_Close_StopsSweeperGoroutine` tolerance on loaded CI; scale it with `TimingScale` if it ever flakes | Polling test with +2 tolerance                    |
| 38 | Refresh remaining pinned actions when Dependabot PRs arrive (upload-artifact v4→v5 etc.)                                  | Node 20 deprecation warnings already visible      |
| 39 | Benchmark-regression guard (benchstat on hot-path PRs) — only if a regression is ever observed                            | Anti-Verschlimmbesserung: don't build prematurely |
| 40 | Long-soak fuzz locally before v0.3.0 (10–30 min per target)                                                               | CI budget stays small; soaks catch the long tail  |

**Docs / hygiene**

| #  | Task                                                                                                                                           | Why now                                          |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| 41 | README FAQ entry: **"isn't idempotency supposed to be forever?"** — distill the TTL-window + poisoned-claim answer from the ADR-004 discussion | You asked it; the next consumer will too         |
| 42 | ADR index (docs/adr/README.md listing 001–004 one-liners)                                                                                      | Four ADRs now; discovery is nontrivial           |
| 43 | DOMAIN_LANGUAGE.md: add `CommandIdempotency`, `claim`, `response replay`, `poisoned claim` glossary rows                                       | New vocabulary shipped this session              |
| 44 | Example dir README (or link it from README "Documentation") — currently only the doc comment explains it                                       | Discoverability                                  |
| 45 | Migration guide: re-verify snippets against the next published tag (the verify-script pattern is in the commit history)                        | Snippet rot is the quiet killer                  |
| 46 | Consider dropping "60+ linters" phrasing (README/CONTRIBUTING say different numbers; AGENTS said 105) — standardize on one source              | Small accuracy nit I left behind                 |
| 47 | README "Documentation" section: add SECURITY/CoC links alongside ADRs                                                                          | Community files deserve discoverability          |
| 48 | Sweep godoc for `[MemoryStore]` references that now resolve to a deprecated type and re-target where misleading                                | The Store docs reference it as the mutex example |
| 49 | Verify the Codecov badge flips from "unknown" to a real % after activation + one push                                                          | Honest-number guarantee (T15.3)                  |
| 50 | Groom ROADMAP: mark response-replay/evolution items resolved (done), promote v0.3.0 items, prune anything ADR-killed                           | ROADMAP should reflect post-ADR reality          |

## g) Top #3 questions I can NOT figure out myself

1. **`Delete` — confirm the deferral or override it?** ADR-004 defers `Delete` because of your "isn't idempotency supposed to be forever?" concern, with a documented trigger (repeated poisoned-claim incidents that TTL tuning can't absorb). My recommendation was _accept-with-ops-recovery-framing_, but your instinct is the owner's voice on record, so: **defer (current state), or should I implement it in v0.3.0 with strict "operational recovery only" semantics and docs?**
2. **ADR-002/003 — veto window.** The middleware ships as a stdlib-only subpackage (ADR-002) and BoundedStore is rejected in favor of a documented durability position (ADR-003), both marked _provisionally accepted_ under your blanket "do the whole list" approval. Confirm, or tell me what to move (e.g., middleware → sibling module at v0.3.0 is still cheap).
3. **Codecov + Dependabot:** Will you activate Codecov / provide a `CODECOV_TOKEN` (one `gh secret set` away, step self-activates) — or should I drop the badge and upload step entirely? And may I merge or close **Dependabot PR #6** (actions/checkout 4.4.0 → 7.0.1, currently running CI), or do you want to handle Dependabot PRs yourself?

---

_Point-in-time snapshot. Section (f) is the HARVEST input for TODO_LIST/ROADMAP; this file itself goes stale by design._
