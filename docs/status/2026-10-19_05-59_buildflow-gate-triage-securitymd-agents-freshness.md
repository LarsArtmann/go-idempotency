# Status Report — BuildFlow Gate Triage: securitymd + AGENTS.md Freshness (go-idempotency)

| | |
| --- | --- |
| **Date** | 2026-10-19 05:59 CEST |
| **Repo** | `/home/lars/projects/go-idempotency` (module `github.com/larsartmann/go-idempotency`) |
| **Trigger** | `buildflow --fix --build-mode=full` exited **69**: 2 error-severity findings remained after auto-fix (`go-structure-linter` 1, `securitymd` 1) |
| **Outcome** | Gate **green (exit 0)**, verified twice (full mode + fast mode). All fixes committed by the auto-commit daemon (`ab2b2e7` + follow-up). |
| **Scope discipline** | Session was triage-and-fix of the BuildFlow findings gate only; no Go code was touched. |

---

## a) FULLY DONE

1. **securitymd error + warning cleared — SECURITY.md now passes with 0 findings.**
   - Evidence: `buildflow -s securitymd --format finding` → `"total": 0` (verified twice, second time cold-cache).
   - Root causes found by reading the validator source (`securitymd/pkg/policy/validate.go`): the file had NO `## Security Practices` section (rule `missing-practices`, error), and the existing commitment was phrased "acknowledgement within 7 days" — which matches **none** of the rule's literal substrings (`respond within`, `response time`, `within 24/48/72 hours`, …), so `missing-response-time` (warning) fired despite a commitment existing.
   - Fix: added `## Security Practices` (real practices: minimal attack surface, CI govulncheck/fuzz/race gates, fail-closed dispatch, TTL validation at the boundary, hardened releases) and rephrased to "We respond within 7 days …".
   - Files: `SECURITY.md` (committed in `ab2b2e7`).

2. **AGENTS.md staleness error cleared — genuine refresh, not a clock reset.**
   - Evidence: cold-cache `buildflow -s go-structure-linter` → age finding gone; only 1 warning left (see below).
   - Real drift found and corrected while verifying claims against the codebase:
     - Header said "Go 1.26+" but `go.mod` says `go 1.27` (changed before 2026-09-11, missed in the last refresh).
     - `doc.go` grew the SQLite connection-pool pitfalls section (2026-10-03/05, battle-tested in KeyHolderAI) — now mentioned.
     - `docs/migrating-from-memorystore.md` existed but was absent from the Architecture list — added.
     - New section "Known Tool Findings (deliberate non-fixes)" documenting: the `examples-directory` false positive, the 14-day AGENTS.md age gate policy, the securitymd literal-substring requirements, and the govulncheck go1.27 environmental warning.
   - Files: `AGENTS.md`.

3. **README installation info finding cleared + stale badge fixed.**
   - Evidence: `readme-install-section` finding gone in cold-cache re-detection.
   - Read the rule source first (`go-structure-linter/internal/rules/readme_install_section_rule.go`): regex `^#{1,6}\s*(install|installation|getting started|setup|quickstart)\b` — "Quick start" (with a space) never matched, which is why the info fired although `go get` was already in the README.
   - Fix: added a dedicated `## Installation` section (go get, Go 1.27+ requirement, sole runtime dep) and moved the `go get` block out of Quick start; also fixed the `Go-1.26+` badge → `Go-1.27+`.
   - Files: `README.md` (committed in `ab2b2e7`).

4. **lychee `docs/feedback` 404 finding cleared at root cause — shared `lychee.toml`.**
   - Evidence: `buildflow -s lychee --format finding` → `"total": 0`; the papdashboard 404 and the transient go-error-family 504 are both gone.
   - Root cause: the CI Docs job excludes historical snapshots via explicit `--exclude-path` flags (`.github/workflows/ci.yml:190-192`), but BuildFlow's lychee step does not inherit those flags and the repo had no `lychee.toml`.
   - Fix: added `lychee.toml` with `exclude_path = ["docs/feedback", "docs/status", "docs/planning"]` (auto-discovered by BuildFlow, merges with the CI flags — same semantics); updated the AGENTS.md CI paragraph to name `lychee.toml` as the shared source.
   - Files: `lychee.toml` (new), `AGENTS.md`.

5. **`examples-directory` warning triaged and closed as a documented deliberate non-fix.**
   - Evidence: verified false positive in go-structure-linter's own status report (`docs/status/2026-10-08_23-39…: "Verified false positive: examples-directory vs idiomatic Go"`); upstream re-scoping is already tracked in that repo's TODO_LIST. The flagged files are in-package godoc `Example*` functions that must stay in their packages to render on pkg.go.dev; `example/` (runnable walkthrough) stays.
   - Recorded in AGENTS.md "Known Tool Findings" so future sessions don't "fix" it.

6. **Gate verified green end-to-end.**
   - Cold-cache full run: "BuildFlow passed with warnings 47/53, 24.7s, 40 findings" — no findings-gate ERROR block.
   - Fresh fast-mode rerun: `EXIT=0`.
   - `./scripts/check-stale-refs.sh`: clean after every doc edit.
   - Auto-commit daemon captured all changes; `git status` clean at session end.

---

## b) PARTIALLY DONE

1. **BuildFlow result-cache replay gotcha: diagnosed but not yet written into AGENTS.md.**
   - What works: I hit it live — after editing AGENTS.md, a plain `buildflow -s go-structure-linter` still replayed the cached "29 days old" finding; `BUILDFLOW_NO_RESULT_CACHE=1` showed fresh detection (age gone).
   - What remains: a one-bullet AGENTS.md entry ("after editing a file a `-s` step may replay cached findings; bypass with `BUILDFLOW_NO_RESULT_CACHE=1`") so the next session doesn't get misled. Not yet written.
   - Effort: S.

2. **Verification of the lychee CI merge behavior.**
   - What works: BuildFlow's local step honors `lychee.toml` (0 findings) and the TOML is valid (the step would have errored otherwise). Config+flags merge semantics are documented lychee behavior and the exclusions are identical, so the CI Docs job should be unaffected.
   - What remains: the real CI Docs job has not run since the change — I cannot trigger GitHub Actions locally. First push should be watched.
   - Blocker: none, just pending the next CI run.
   - Effort: S (watch + fix if the action rejects the merge).

3. **Doc freshness (FEATURES.md, TODO_LIST.md at 35 days).**
   - What works: README.md and AGENTS.md aged out of the report because they got genuine updates this session.
   - What remains: FEATURES.md and TODO_LIST.md still warn (840h threshold); I deliberately did NOT touch them just to reset mtime. Section (f) below is the harvest input; a `docs-health` HARVEST pass should update them with real content.
   - Effort: M.

4. **Preflight advisories — observed, deliberately not acted on (out of this repo's scope).**
   - `GOEXPERIMENT=jsonv2` set in shell but unused here (BuildFlow says harmless).
   - BuildFlow binary freshness: binary at `25b07c4`, repo HEAD moved `cd1c94e` → `2878d64` **during** my session (a concurrent session is active in the BuildFlow repo). Advisory only.
   - `lychee-private-links` preflight: no GITHUB_TOKEN locally; fleet authenticate-vs-exclude policy explicitly undecided (see question g1).

---

## c) NOT STARTED

All of the below was *observed* this session but intentionally not started (below the error gate, or out of scope):

1. **file-size debt**: `store_test.go` (516), `contract/contract.go` (434), `contract/contract_negative_test.go` (483) all exceed the 350-line warning threshold. Not started; splitting test/contract files has real design cost and was not gated.
2. **art-dupl findings (20)**: token-level duplication in `bench_test.go` and `example/main.go`. The repo has an established "annotate accepted duplication" pattern (commit `96d0d04`); triaging these 20 into harmful-vs-accepted was not started.
3. **Strict-timing contract pass at `TimingScale: 3` locally**: CI runs it weekly via `GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE`; I never ran it locally (not needed for the gate; local parity unverified).
4. **Local 3-minute fuzz runs**: CI runs all four fuzz targets; I did not run them locally this session (BuildFlow's step set did not include them in the 47 executed steps I inspected).
5. **A minimal `.buildflow.yml` for this repo**: none exists (BuildFlow auto-detects everything). Whether skip-steps/tool-options should be codified (e.g., `doc-files-age-check` noise policy) is an open decision, not started.
6. **v1.0 release-prep work implied by AGENTS.md** (MemoryStore removal target, `Delete` API deferred per ADR-004): untouched — no session mandate.

---

## d) TOTALLY FUCKED UP

**Nothing is broken.** The gate is green, tests were green in the full BuildFlow run (47/53 steps passed, 0 errors), and `git status` is clean. Radical-honesty candidates that stop short of "fucked up", listed anyway so they are visible:

1. **The first cold full-run exit code was not directly measured.** The background pipe lost `PIPESTATUS`, so I inferred the full-run verdict from the absence of the gate ERROR block and then measured `EXIT=0` on a fresh fast-mode rerun. Verdict is solid (two independent green runs), but the specific number "full-mode exit 0" is inferred, not captured. Severity: none (cosmetic verification gap). Mitigation: any future run confirms.
2. **The auto-commit daemon creates heuristic commit messages** (`chore: auto-commit N changed file(s)`), so this session's SECURITY/README/AGENTS/lychee fixes are spread across daemon commits without a descriptive message. Fleet-known behavior, not a regression — but if descriptive history matters for this repo, the fixes deserve one manual squashed narrative commit. Not done (no commit mandate).

---

## e) WHAT WE SHOULD IMPROVE

1. **Read the linter source before "fixing" a finding.** Both securitymd findings were self-inflicted phrasing/structure gaps that only the rule source revealed (`validate.go` substring list; `readme_install_section_rule.go` regex). Checking rule source first would have saved two detection cycles. Candidate lesson for `crush-config` `references/lessons.md`.
2. **Bypass the result cache when re-checking a single step after edits.** `-s <tool>` after editing a file can replay stale findings (`BUILDFLOW_NO_RESULT_CACHE=1` is the escape). I initially trusted a cached "29 days" reading; caught it, but the trap is undocumented in this repo. → bullet in AGENTS.md (see b1).
3. **Centralize linter-exclusion config instead of per-runner flags.** The CI-only `--exclude-path` flags vs BuildFlow's default scan was a split brain that produced a recurring phantom warning; `lychee.toml` as the single source fixed it. Pattern generalizes: any exclusion should live in the tool's config file, not in one runner's CLI args.
4. **AGENTS.md refreshes must be genuine drift-checks, and the 14-day gate guarantees they happen.** This refresh caught real drift (Go version, doc.go scope, missing migration-doc entry). The recurring cost is accepted; a mechanism to make it less reactive (see f44) would help.
5. **Daemon commits dilute history.** Multi-file doc fixes land as "auto-commit N file(s)". Where a session produces a coherent changeset, staging a descriptive commit before the daemon sweeps would keep history readable — but only when the user asks for commits (harness contract).

---

## f) Top 50 things we should get done next

Ranked by impact; effort S (<30min) / M (30min–2h) / L (>2h). **This section is the harvest input for `docs-health` → HARVEST into TODO_LIST.md/ROADMAP.md.** Items 1–15 are directly grounded in this session's observations; 16–50 are repo-health items noticed in passing.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | HARVEST this report: pull items below into TODO_LIST.md (short-term) and ROADMAP.md (ideas) | High | S | Documentation |
| 2 | Watch next CI run: confirm the Docs job (lychee-action) accepts `lychee.toml` merged with its explicit `--exclude-path` flags | High | S | Quality |
| 3 | Add AGENTS.md bullet: `-s <tool>` can replay cached findings after edits; use `BUILDFLOW_NO_RESULT_CACHE=1` | Medium | S | Documentation |
| 4 | Update FEATURES.md with real drift: doc.go SQLite section, migration guide, lychee.toml, Go 1.27 floor | Medium | S | Documentation |
| 5 | Decide fleet lychee policy: export GITHUB_TOKEN (authenticate) vs exclude `github.com/larsartmann/.*` (fleet-wide, BuildFlow preflight says undecided) | High | S | Quality |
| 6 | Triage the 20 art-dupl findings in `bench_test.go`/`example/main.go`: extract harmful dupes, annotate accepted ones (repo precedent `96d0d04`) | Medium | M | Cleanup |
| 7 | Split `contract/contract.go` (434 lines) if a natural seam exists (suite vs helpers vs README/docs block) | Medium | M | Cleanup |
| 8 | Split `contract/contract_negative_test.go` (483 lines) — likely by suite (main vs context) | Medium | M | Cleanup |
| 9 | Split `store_test.go` (516 lines) by strategy (unit vs concurrency vs edge cases) | Medium | M | Cleanup |
| 10 | Run the strict-timing contract pass locally (`GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE=3 go test ./contract/...`) to confirm local parity with the weekly CI job | Medium | S | Quality |
| 11 | Run all four fuzz targets 3m each locally once (`go test -fuzz=... -fuzztime=3m`) for parity with CI | Medium | M | Quality |
| 12 | Run `erraudit . --enforce-go-error-family` to re-confirm 0 violations (policy sanity, no error code touched this session) | Low | S | Quality |
| 13 | Annotate the `docs/feedback/new/…papdashboard…` snapshot noting the linked repo is gone (annotation, not rewrite — historical snapshot policy) | Low | S | Documentation |
| 14 | Sweep living docs for other links to deleted/renamed repos (papdashboard pattern) before they hit lychee | Medium | S | Documentation |
| 15 | Verify CODECOV_TOKEN secret is actually set (CI gates the upload on it; silent no-op if absent) | Medium | S | Quality |
| 16 | Decide whether this repo wants a minimal `.buildflow.yml` (codify skips/tool-options) or stays zero-config | Low | S | Quality |
| 17 | Consider a `doc-files-age-check` noise policy (7-day threshold vs realistic doc cadence) — config or accepted-warnings note | Low | S | Quality |
| 18 | Establish a cadence/mechanism so the AGENTS.md 14-day gate is met by genuine refreshes (calendar, CI reminder, or task-queue dispatch), never clock resets | Medium | S | Process |
| 19 | Rebuild/reinstall the BuildFlow binary (`nix build . && nix run .#reinstall`) once the concurrent BuildFlow session settles | Low | S | Cleanup |
| 20 | Scope `GOEXPERIMENT=jsonv2` to the projects that need it instead of exporting it globally in the login shell | Low | S | Cleanup |
| 21 | Check upstream go-structure-linter TODO item ("re-scope examples-directory"); nudge or implement if the repo is free | Low | S | Cleanup |
| 22 | Draft the v1.0 release checklist: MemoryStore removal, migration guide prominence, contract-suite promotion story | High | M | Feature |
| 23 | Revisit ADR-004 (`Delete` on `Store`): any consumer need surfaced since deferral? | Low | S | Feature |
| 24 | Keep ADR-002 YAGNI gate on `EventIdempotency`/`QueryIdempotency` (confirm still no consumers) | Low | S | Feature |
| 25 | Verify `go-error-family` bump (`4540c84`) has a CHANGELOG entry | Low | S | Documentation |
| 26 | Review current coverage number (CI uploads to Codecov; local number unknown) and record it in TODO_LIST as a baseline | Medium | S | Quality |
| 27 | Confirm middleware dispatch cancellation semantics: does the context-aware contract suite cover `middleware.NewCommand`, or only stores? Gap-check | Medium | M | Quality |
| 28 | Property-test generators: confirm empty keys / zero TTL are in the rapid generator space (fuzz covers them; property should too) | Low | S | Quality |
| 29 | Re-run memory benchmarks and record current bytes/key + %-reclaimed numbers somewhere durable (docs or TODO_LIST baseline) | Low | S | Quality |
| 30 | pkg.go.dev: after the next tag, verify the README Installation section and doc.go TOC render correctly | Low | S | Documentation |
| 31 | Confirm RELEASING.md's push-master-and-tag-together advice still matches the lychee CHANGELOG compare-link strategy (unchanged this session, but cheap to re-verify) | Low | S | Documentation |
| 32 | Add a stale-refs pattern if any new rot-prone phrase was introduced this session (none identified — verify once) | Low | S | Documentation |
| 33 | Consider `gofmt`/dprint check of `lychee.toml` in the repo's format loop (dprint ran; confirm TOML is in its scope) | Low | S | Quality |
| 34 | git history hygiene: if descriptive history matters, note a repo policy for pre-staging coherent changesets before the daemon sweeps | Low | S | Process |
| 35 | Fleet idea for BuildFlow: lychee step could detect CI `--exclude-path` flags vs missing `lychee.toml` (config-drift warning) — the exact split brain fixed here | Medium | M | Feature |
| 36 | Fleet idea for BuildFlow: surface `-s` result-cache replay in step output ("served from cache, N s old") to prevent false stale readings | Medium | M | Feature |
| 37 | Record the "read the rule source before fixing" lesson in `crush-config/references/lessons.md` (cross-project) | Medium | S | Process |
| 38 | Record the "exclusions belong in tool config, not one runner's CLI flags" lesson alongside it | Medium | S | Process |
| 39 | Verify the daemon committed `lychee.toml` with all three paths intact (spot-check `git show` — status clean but content worth one look) | Low | S | Quality |
| 40 | Check whether `internal/teststore/context.go` (flagged by govulncheck warnings) is exported-anywhere-safe as-is (module-internal, should be — confirm no accidental promotion) | Low | S | Quality |
| 41 | govulncheck go1.27 warnings: once the local binary is refreshed (f19), confirm they disappear; if not, investigate for real | Low | S | Quality |
| 42 | Confirm `.golangci.yml` exhaustruct_v5 config still validates after future golangci-lint bumps (the v2.14.0 bump `1bcbf23` was exactly such a fix) | Low | S | Quality |
| 43 | Add the strict-timing weekly run's result visibility: does anyone read it? If not, add a failure-notification path | Low | S | Process |
| 44 | Evaluate pinning `actions/*` SHAs review cadence (all actions are SHA-pinned; confirm a renewal reminder exists) | Low | S | Quality |
| 45 | README: consider linking `docs/migrating-from-memorystore.md` from the deprecation note in Quick start | Low | S | Documentation |
| 46 | AGENTS.md: consider a one-line pointer to the contract README's "Extending the suite" workflow for new-invariant PRs | Low | S | Documentation |
| 47 | Sweep `docs/` for any other CI-vs-local tool-config split brains (same pattern as lychee: gitleaks? codespell? configs) | Medium | M | Quality |
| 48 | Confirm example_test godoc examples still render (pkg.go.dev) after any future package reshuffle — add to release checklist | Low | S | Documentation |
| 49 | Consider benchmarking `CheckAndRecord` against a real SQL/SQLite backend once (doc.go claims it is "simpler AND faster" — unbenchmarked claim in this repo) | Medium | M | Quality |
| 50 | Post-release: verify the new CHANGELOG compare links pass lychee before tagging (RELEASING.md ordering) | Low | S | Documentation |

---

## g) Questions I cannot figure out myself

1. **Fleet lychee policy (blocks #5):** BuildFlow's preflight says "authenticate-vs-exclude policy is undecided" — for private `github.com/larsartmann/*` links, do you want local runs to authenticate via `GITHUB_TOKEN`, or should repos exclude the namespace in `lychee.toml`? I can't decide fleet policy, and this repo's `lychee.toml` is now the natural place to encode your answer.
2. **AGENTS.md freshness mechanism (blocks #18):** the 14-day gate will keep erroring between genuine refreshes. Is reactive per-session refresh (current behavior) acceptable as the standing policy, or do you want a proactive mechanism (scheduled task-queue dispatch / CI reminder) — and if so, which?
3. **Doc cadence vs 7-day threshold (blocks #4, #17):** FEATURES.md/TODO_LIST.md are 35 days stale with only warnings. Should I run a full `docs-health` BUILD/HARVEST pass over them now (genuine content update from this report), or do you deliberately let living docs age between feature sessions and want the warnings accepted-and-documented instead?

---

**Next step per protocol:** section (f) is the harvest input — run `docs-health` → HARVEST to land items into TODO_LIST.md/ROADMAP.md rather than entombing them here.

*Waiting for instructions.*
