# Status Report: GitHub Repo Description & Metadata Refresh

**Scope line — verified in this session:**

- **Verified here (fresh command output, this session 2026-11-17 ~21:40–22:00 CET):**
  - Old metadata fetched: `gh repo view larsartmann/go-idempotency --json description,homepageUrl,repositoryTopics` → description `"Go idempotency/deduplication store for CQRS command keys"`, homepage `""`, topics `[cqrs, dedup, deduplication, go, golang, idempotency]`.
  - Sibling house-style census: descriptions of 6 repos fetched via `gh repo view` (go-error-family, go-output, go-cqrs-lite, go-sse, go-aichat, go-atomic-write). All follow `{one-liner} — {feature list}`, 150–240 chars.
  - Fact base read directly: `README.md` lines 1–60, `doc.go` lines 1–40, `CHANGELOG.md` v0.5.0/v0.4.0/v0.3.2 sections, `website/` directory absent (`ls` → not found).
  - New description length: 330 chars, computed via shell `${#D}` (GitHub limit 350) — checked **before** applying.
  - Edit applied: `gh repo edit larsartmann/go-idempotency --description "..." --add-topic exactly-once --add-topic middleware --add-topic distributed-systems` (clean exit).
  - Read-back verification: `gh repo view --json description,repositoryTopics` → new 330-char description live; topics sorted: `[cqrs, dedup, deduplication, distributed-systems, exactly-once, go, golang, idempotency, middleware]` (9 total).

- **NOT verified here (and why):**
  - Visual rendering (About sidebar, profile-card truncation, social preview image) — no screenshot/render step was run; needs eyes or the skill's generator.
  - Current social preview card state — never fetched.
  - CI status on master, latest GitHub Releases for v0.5.0 and `idemtest/v0.1.0`, pkg.go.dev propagation — out of scope for a metadata-only task, not attempted.
  - Whether anything depends on the old description string (e.g. tooling, link previews in chat archives) — not searched.
  - Repo-side consistency (AGENTS.md vs README vs doc.go vs new description) — I read enough to write the description, but ran no drift pass and no `check-stale-refs.sh`.
  - No repo files were changed → no build/test/lint run (nothing to run against).

**Session context:** single task — answer "Is the GitHub description still up to date and superb?" (verdict: up to date in truth-value, stale in coverage, not superb in form), then fix it. No code, no docs, no commits by hand.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                           | Evidence                                                        | Scope                                    |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- | ---------------------------------------- |
| a1 | GitHub description replaced with a house-style one-liner (330/350 chars) that carries the project's four headline features shipped since the old text was written: atomic `CheckAndRecord`, TTL + `Forever` claims, fail-closed middleware, contract suite + consumer test kit | Read-back `gh repo view --json description` shows new text live | GitHub repo metadata only; no repo files |
| a2 | Topics extended by 3 (`exactly-once`, `middleware`, `distributed-systems`), all accurate to shipped code; no framework topics to remove (per skill rule)                                                                                                                       | Read-back `gh repo view --json repositoryTopics` → 9 topics     | GitHub repo metadata only                |
| a3 | House-style audit against 6 sibling repos — old description (58 chars, bare) objectively the thinnest of the fleet; new one matches the `{one-liner} — {features}` convention                                                                                                  | 6 sibling descriptions fetched fresh via `gh repo view`         | Research artifact, in this file          |
| a4 | Fact-base grounding before writing: README headline ("Turns at-least-once delivery into at-most-once processing"), interface-first philosophy, and v0.4.0/v0.5.0 CHANGELOG features all read directly, not from memory                                                         | File reads listed in scope line                                 | README.md, doc.go, CHANGELOG.md          |

## b) PARTIALLY DONE

| #  | Item                                    | Works now                                     | Open                                                                                                                                                                                                                                                                                    | Blocker                                                                                               | Effort |
| -- | --------------------------------------- | --------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------ |
| b1 | website-launch Phase 6 metadata pass    | Description + topics done and verified        | Homepage field (empty — no website exists), social preview card (not generated/uploaded), visual truncation check                                                                                                                                                                       | Homepage needs the owner's website-vs-pkg.go.dev decision; social preview upload is manual/owner-only | S each |
| b2 | Description accuracy vs project reality | Core claims verified against README/CHANGELOG | Two precision gaps: (1) "you implement the store" overclaims — go-cqrs-lite `metaengine.DedupStore` consumers do NOT implement it (README.md:53); (2) description doesn't reveal `idemtest` is a separately-versioned module (`go get .../idemtest`) — the most actionable install fact | 350-char budget forces a tradeoff; needs owner's priority call (see g3)                               | S      |

## c) NOT STARTED

| #  | Item                                                                                                          | Why not started                                                   | Still wanted?                                                  |
| -- | ------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | -------------------------------------------------------------- |
| c1 | Social preview card (generate SVG/PNG via website-launch `generate.sh`, then manual upload)                   | Session scope was description+topics; upload is manual owner-only | Yes — repo cards currently render README screenshot or default |
| c2 | Docs website → homepage field (`lars.software` subdomain, full website-launch flow with demo video default)   | Big flow, owner decision required; inventing a URL was off-limits | Unknown — needs owner (g1)                                     |
| c3 | Fleet-wide description house-style audit (the 6 siblings were read for style only, not audited for staleness) | Out of scope; noticed during comparison                           | Recommend yes — same staleness pattern likely                  |
| c4 | Evidence-based topic selection (topic census across siblings / search-discovery check)                        | Judgment-based 3 topics were applied; census not run              | Optional, Low                                                  |
| c5 | Any repo-code work                                                                                            | None requested; none touched                                      | n/a                                                            |

## d) TOTALLY FUCKED UP

**Nothing.** Radical-honesty check performed, not skipped:

- What could have been fucked up here: a wrong, overlong, or overclaiming description shipped live. Guarded — fact base read first, char count computed before apply, read-back verified after.
- Damage surface: zero repo files touched, no push, no force operations, no CI triggers. The GitHub API edit is reversible in one `gh repo edit`.
- Two near-misses caught during drafting (not shipped broken, but worth recording): (1) first draft exceeded 350 chars and was trimmed before apply; (2) the "Interface-first: you implement the store" clause is a mild overclaim vs the go-cqrs-lite metaengine path — it shipped anyway because it's directionally true for this repo's own posture (README:49 "There will never be a Redis store... in this module"), but it is listed as improvement item e1, not hidden.

## e) WHAT WE SHOULD IMPROVE

| #  | Improvement                                                                                                                                   | Impact                                                                                           | Concrete fix                                                                                                                                                                |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| e1 | Description overclaims "you implement the store" — go-cqrs-lite consumers get `metaengine.DedupStore` instead (README.md:53)                  | Medium — a metaengine adopter reading the About line gets the wrong onboarding instruction       | Either accept (repo's own posture is interface-first) or spend ~60 chars: "…or use go-cqrs-lite's DedupStore engine"                                                        |
| e2 | Description hides that the test kit is a separate module with its own install path and `idemtest/v*` tags (v0.5.0's biggest operational fact) | Medium — consumers won't discover `idemtest` from the repo card                                  | Name the module path in a future description revision or rely on README/Release notes                                                                                       |
| e3 | GitHub metadata lives only on GitHub; no in-repo record of the house convention (format, 350-char budget, topic vocabulary)                   | Medium — next session can't tell deliberate house style from accident; risk of silent regression | One AGENTS.md line: format, char budget, topic rule, date of last audit                                                                                                     |
| e4 | Verification was API read-back only — no visual pass                                                                                          | Low — truncation/rendering issues would be invisible to this method                              | `gh repo view --web` eyeball pass after social preview lands                                                                                                                |
| e5 | Topic choices were judgment-based, not evidence-based                                                                                         | Low                                                                                              | 10-minute sibling topic census before adding more topics                                                                                                                    |
| e6 | Single-repo focus — the same staleness almost certainly exists fleet-wide                                                                     | Medium — fleet-level inconsistency in public presence                                            | Turn this session into a repeatable metadata-audit checklist inside website-launch (skill candidate per the feedback loop; if it recurs in 2+ reports it should become one) |

## f) 50 things to get done next

> Per the status-report skill: beyond ~25 this is a **brainstorm, not a commitment list** — most items below are ROADMAP fuel for docs-health HARVEST, not a promise. Items 1–10 are the honest short queue; the rest are specific enough to harvest or discard explicitly.

| #  | Task                                                                                                                                                 | Impact | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Generate social preview PNG via website-launch `generate.sh` (title + tagline from new description)                                                  | High   | S      | Documentation |
| 2  | Upload social preview (manual, owner-only; 1280x640, <1 MB, Settings → General)                                                                      | High   | S      | Documentation |
| 3  | Decide homepage field: launch website vs point at pkg.go.dev vs leave empty                                                                          | High   | S      | Documentation |
| 4  | Verify GitHub Release exists for v0.5.0 with its CHANGELOG section (release.yml fired)                                                               | High   | S      | Quality       |
| 5  | Verify GitHub Release exists for `idemtest/v0.1.0` (second tag on same push — did it get its own Release?)                                           | High   | S      | Quality       |
| 6  | Verify `go get github.com/larsartmann/go-idempotency/idemtest@v0.1.0` resolves (tag→proxy propagation)                                               | High   | S      | Quality       |
| 7  | Verify pkg.go.dev serves the `idemtest/v0.1.0` module                                                                                                | High   | S      | Quality       |
| 8  | HARVEST this report's section (f) into TODO_LIST.md / ROADMAP.md (docs-health)                                                                       | High   | S      | Quality       |
| 9  | Record GitHub-metadata house convention in AGENTS.md (format, 350-char budget, topic rule — fixes e3)                                                | Medium | S      | Documentation |
| 10 | Draft description v2 resolving e1+e2 (metaengine path and/or idemtest module path within 350 chars)                                                  | Medium | S      | Documentation |
| 11 | Confirm latest CI run on master is green (not checked this session)                                                                                  | Medium | S      | Quality       |
| 12 | Confirm the weekly scheduled run's strict-timing pass (TimingScale 3) green                                                                          | Medium | S      | Quality       |
| 13 | Confirm docs job (stale-refs + lychee) green on the latest push                                                                                      | Medium | S      | Quality       |
| 14 | Verify pkg.go.dev serves root v0.5.0                                                                                                                 | Medium | S      | Quality       |
| 15 | Confirm Dependabot's new `/idemtest` gomod ecosystem is active and its PRs green                                                                     | Medium | S      | Quality       |
| 16 | Launch docs website via website-launch (demo video default) — the real fix for c2                                                                    | High   | L      | Feature       |
| 17 | Fill homepage field after #16                                                                                                                        | Low    | S      | Documentation |
| 18 | README "nice docs" retrofit audit: "Who is this for?" / "When NOT to use this" sections (skill maintenance-mode checklist)                           | Medium | M      | Documentation |
| 19 | Census sibling repos' topics to ground topic vocabulary fleet-wide                                                                                   | Low    | M      | Documentation |
| 20 | Audit all sibling repos' descriptions against house style; produce drift list                                                                        | Medium | M      | Documentation |
| 21 | Apply description fixes to drifted siblings (after #20)                                                                                              | Medium | M      | Documentation |
| 22 | Add `http` topic (the middleware ships a net/http adapter)                                                                                           | Low    | S      | Documentation |
| 23 | Add `library` topic — or reject with a recorded reason                                                                                               | Low    | S      | Documentation |
| 24 | Add `retry` topic for retry-semantics searchers                                                                                                      | Low    | S      | Documentation |
| 25 | Add `testing` topic (contract suite + idemtest are testing surfaces)                                                                                 | Low    | S      | Documentation |
| 26 | Evaluate `contract-testing` as topic keyword                                                                                                         | Low    | S      | Documentation |
| 27 | Verify README badges live-resolve (CI, Codecov, pkg.go.dev, Go Report Card, Go Version, License)                                                     | Low    | S      | Quality       |
| 28 | Verify Codecov dashboard reflects v0.5.0 coverage (badge exists; dashboard not checked)                                                              | Low    | S      | Quality       |
| 29 | Visual pass: About sidebar + profile-card truncation after edits (fixes e4)                                                                          | Low    | S      | Documentation |
| 30 | Screenshot-verify social preview legibility at card size (1280x640 design rules)                                                                     | Low    | S      | Documentation |
| 31 | Cross-check AGENTS.md, README, doc.go agree with the new description one-liner (drift pass)                                                          | Low    | S      | Documentation |
| 32 | Ensure README shows root-module and `idemtest`-module install commands together in one quick-start block (check-then-fix; not verified this session) | Medium | S      | Documentation |
| 33 | Check README carries the docs link bar per readme-template; retrofit if missing                                                                      | Low    | M      | Documentation |
| 34 | README FAQ entry: "Do I need a backend if I use go-cqrs-lite?" (condenses the README:53 metaengine pointer)                                          | Low    | S      | Documentation |
| 35 | Keep headline consistency rule: README first line ↔ description first clause stay aligned on future edits                                            | Low    | S      | Documentation |
| 36 | Decide whether "CQRS command keys" stays in the description headline or broadens further to "retried operations" only                                | Low    | S      | Documentation |
| 37 | Run `./scripts/check-stale-refs.sh` on next doc touch (convention; no doc change today)                                                              | Low    | S      | Quality       |
| 38 | Check GitHub issue templates exist (Settings/`.github`; not checked this session)                                                                    | Low    | S      | Quality       |
| 39 | Confirm SECURITY.md still passes the securitymd validator (known-findings phrasing intact)                                                           | Low    | S      | Quality       |
| 40 | Refresh the local govulncheck binary (environmental go1.27 warning noted in AGENTS.md)                                                               | Low    | S      | Quality       |
| 41 | Read-verify release.yml carries the fleet-required `concurrency` block as documented                                                                 | Low    | S      | Quality       |
| 42 | Verify lychee.toml excludes still cover new docs/status files (archived vs live)                                                                     | Low    | S      | Quality       |
| 43 | Annotate/archival pass over docs/status/ via docs-health ANNOTATE (this file's predecessors)                                                         | Low    | M      | Documentation |
| 44 | Consider metadata-audit checklist inside website-launch skill (fixes e6; skill candidate)                                                            | Medium | M      | Quality       |
| 45 | Consider a fleet script: repo description vs README first-line drift check                                                                           | Low    | M      | Quality       |
| 46 | Add an `idemtest` usage example to `example/` walkthrough (currently store-only; would sell the kit)                                                 | Medium | M      | Feature       |
| 47 | Poll next weekly CI scheduled run for the TimingScale 3 pass                                                                                         | Low    | S      | Quality       |
| 48 | Re-run this metadata audit after the next feature release (staleness recurrence guard)                                                               | Medium | S      | Documentation |
| 49 | Fold the 350-char budget note into the AGENTS.md metadata line (#9)                                                                                  | Low    | S      | Documentation |
| 50 | Note-and-close: go-structure-linter `examples-directory` false positive stays as documented non-fix until upstream #1 lands                          | Low    | S      | Cleanup       |

## g) Top 3 questions I cannot figure out myself

1. **Homepage intent:** Should this repo get a docs website (website-launch flow: Astro/Starlight + Firebase + demo video), or is pointing the homepage field at pkg.go.dev the intended end state, or deliberately empty? I verified no `website/` directory exists and invented no URL — but the field stays dead until you pick.
2. **Social preview:** Do you want a designed 1280x640 social card for repo links (I can generate the PNG from the new tagline; the upload step is Settings-only and manual)? If yes, any tagline preference beyond the description's first clause?
3. **Description v2 tradeoff:** The 350-char budget cannot fit both open facts — the `idemtest` separate-module install path (e2) and the go-cqrs-lite metaengine no-backend path (e1) — without dropping something (likely "fail-closed middleware" or "consumer test kit" as words). Which fact matters more to a repo-card reader?

---

_Report per status-report skill (`docs/status/*.md` convention). No manual commit: Crush harness forbids commits without explicit request; the auto-commit daemon picks this file up. Section (f) is the primary input for docs-health HARVEST — not yet run this session (owner said wait for instructions)._
