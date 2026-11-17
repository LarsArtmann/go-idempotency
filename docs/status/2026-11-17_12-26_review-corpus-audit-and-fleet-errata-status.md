# Session status: fleet review corpus audit and errata

**Date:** 2026-11-17 12:26 CET
**Repo:** `go-idempotency` (cwd) with cross-repo writes into `profileagent`
**Session type:** research, verification, report writing. No production code changed.

## Scope: what this report verified, and what it did not

Verified in this session:

- The `Store` vs `metaengine.DedupStore` layering, against source:
  `go-cqrs-lite/metaengine/v4/dedup_store.go`, `go-cqrs-lite/middleware/v4/idempotency.go`,
  `go-cqrs-lite/docs/adr/0158-*.md`, `go-idempotency/store.go`.
- All 25 `*/docs/review/libs/go-idempotency.md` consumer audits under
  `/home/lars/projects` (read in full).
- All 20 `profileagent/docs/review/libs/*.md` reports plus `index.html`
  (read in full).
- 18 of 20 "latest version" fields, the CV pins for every library the
  profileagent corpus names, selected evidence citations, and the
  cmdguard/contract/module censuses, via `git tag/describe/show`, `rg`,
  `find`, and `meta get`.
- The deliverables named below exist on disk and parse.

Not verified, and why:

- Not every claim in every report was re-checked. Deep verification covered
  versions, pins, premise claims, counts, and evidence citations; prose
  judgments ("would be a split brain", "the strongest signal") were assessed,
  not proven.
- The `games` and duplicate-checkout directories in the import census were not
  individually validated as distinct active projects. `ci-siblings/` and
  `DiscordSync`/`discordsync` were excluded as mirrors.
- The two HTML artifacts were checked for structure (Python parser, no
  unclosed tags, no leftover placeholders) and the profileagent index was
  reviewed from source, but neither was browser-rendered in this session.
- No claim here about GitHub CI, published pkg.go.dev state, or remote tags;
  everything is from local checkouts.
- Nothing was verified about the Go 1.27 toolchain behavior; no build, test,
  vet, or lint was run because no Go source changed.

## Artifacts produced

| Artifact                                                               | Location                                                                                        | Status                                                                          |
| ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Fleet consumer-audit synthesis (HTML, 8 findings, 25-row corpus table) | `go-idempotency/docs/reviews/2026-11-17_consumer-audit-synthesis.html`                          | Written, structurally validated, not browser-rendered; auto-committed `fdbf9b1` |
| Profileagent review-corpus errata (Markdown, 13 findings)              | `profileagent/docs/feedback/new/2026-11-17_12-06_review-corpus-errata-and-verification-gaps.md` | Written, not linted or link-checked; auto-committed `a7eb469`                   |
| This status report                                                     | `go-idempotency/docs/status/2026-11-17_12-26_review-corpus-audit-and-fleet-errata-status.md`    | Written                                                                         |

## a) FULLY DONE

1. **Answered the layering question with primary sources.** go-idempotency and
   `metaengine.DedupStore` do not compete: go-idempotency owns the `Store`
   contract plus middleware; `DedupStore` is an engine capability. The
   dependency runs go-cqrs-lite -> go-idempotency; `DedupStore` is not a
   `Store` (collection, injected `now`, `(bool, error)`) and needs an adapter.
2. **Read all 25 go-idempotency consumer audits** and extracted the recurring
   signals: permanent-claim gap (CV, taskqueue, invoices), no production
   backend, response replay, contract-suite timing, middleware non-adoption.
3. **Produced the synthesis report** with: 8 prioritized findings, an action
   matrix, a 25-row per-repo stance table, methodology, and verification
   commands.
4. **Read all 20 profileagent library reports plus the index**, and produced a
   13-finding errata report with per-report corrections.
5. **Verified the hard facts** that drive both reports: ADR-0158's
   "go-idempotency core is UNAFFECTED" line; `DedupStore` shape; cqrs-htmx
   v4.18.1 tag; go-appkit root v0.12.0; 18/20 version fields; every CV pin the
   corpus names; `atomicwrite.WriteWithPerm`; credentials-store production use
   in CV; go-codec transitive presence; go-health `samber/do/v2`; `meta`
   importance 74 for go-daemon; go-output's "16 formats".
6. **Corrected three of my own near-misses before publication**: the
   "skeleton" field refers to profileagent, not the library; go-datastar
   `static/v0.8.0` does exist; `NewSystem` is profileagent's planned wrapper.
7. **Recorded the fleet adoption facts**: 13 project checkouts import
   go-idempotency directly, 8 run `contract/`, 0 import `middleware/`.
8. **Wrote both reports to the requested locations** and confirmed them on
   disk.

## b) PARTIALLY DONE

1. **Evidence coverage.** Deep verification covered the decision-critical
   claims; a full claim-by-claim audit of 20 reports would have been much
   larger. The errata says what it verified and what it did not.
2. **The synthesis report's action matrix.** All 8 actions are identified and
   evidenced, but 0 are implemented. Seven are ready; one is owner-gated.
3. **The errata's 13 findings.** The findings are written; the profileagent
   reports, `index.html`, `STACK.md`, and `TODO_LIST.md` are unchanged.
4. **Census definitions.** Direct importers and contract users are clean.
   The "13 direct importers" figure still includes `games` and one duplicate
   checkout family, so the exact project count has a +/-1 margin.
5. **HTML quality.** Structure and placeholders were checked mechanically.
   No browser render, no link check, no print check.
6. **Markdown deliverable hygiene.** The feedback report has not been run
   through a markdown linter or `lychee`; links are inline code spans, but the
   command blocks were not executed as a script.
7. **Cross-repo consistency.** The errata names the propagation into
   `profileagent/STACK.md:61,89`; the propagation is still live.

## c) NOT STARTED

1. Fix the false "superseded by DedupStore" premise in
   `profileagent/docs/review/libs/go-idempotency.md`.
2. Fix the same premise in `profileagent/STACK.md:61,89`.
3. Fix the profileagent index scorecard partition (6 + 13 + 1).
4. Fix cqrs-htmx `v4.18.0 -> v4.18.1`.
5. Fix go-appkit's root version (`v0.12.0`, `preset/v0.1.1` is a submodule).
6. Cite or correct cmdguard's "27 fleet consumers".
7. Reconcile sentinel counts (18 vs 17 vs 30+) and define the counting rule.
8. Define the module-count convention in the index and regenerate the counts.
9. Add verification-status blocks and expected outputs to the corpus.
10. Re-render `profileagent/docs/review/libs/index.html` after fixes.
11. Harvest both reports into `go-idempotency/TODO_LIST.md` and
    `profileagent/TODO_LIST.md`.
12. The go-idempotency docs fixes: DedupStore adapter recipe, README reword,
    `doc.go` and migration-guide updates.
13. The go-idempotency code fix: contract-suite polling instead of sleeps.
14. The record-on-success limitation docs and ADR-004 trigger update.
15. The `middleware.Command` keep/delete decision and ADR-005 refresh.
16. Release `v0.3.3` (five commits are unreleased).
17. A runnable response-replay example.
18. The "what this is / is not" README block and `NewMemoryStore` doc fix.
19. The `Forever` owner decision.
20. Neither deliverable has been harvested into a TODO_LIST. Both were
    auto-committed by the daemon (`fdbf9b1` in go-idempotency, `a7eb469` in
    profileagent); no manual commit was made, per the harness rule.

## d) TOTALLY FUCKED UP!

Honest list of this session's mistakes, including those caught before
publication. These are process failures, not shipped defects.

1. **Almost misreported "Repo state: skeleton".** I first read it as a claim
   about go-idempotency. It describes profileagent. Caught by reading the
   sibling report and the index hero.
2. **Almost declared go-datastar `static/v0.8.0` nonexistent.** I used
   `git tag --sort=-v:refname | head -1`, which returned `v0.7.0` because of
   tag-name ordering. `git describe --tags` returned `static/v0.8.0`. The
   audit was right; my first check was wrong.
3. **Almost accused go-cqrs-lite of a fabricated `NewSystem` API.** The
   symbol is profileagent's planned wrapper (`profileagent/AGENTS.md`), not
   an upstream constructor. Downgraded to an ambiguity finding.
4. **Wrote "8 months of fleet evidence"** in the HTML hero; the audit window
   is about 3 months. Fixed before the report was final.
5. **Claimed "18 of 20 version fields" while listing 17.** Fixed by adding
   go-cqrs-lite's family-tag framing to the verified list.
6. **Wrote "roughly 70 non-example modules"** for go-cqrs-lite; the actual
   non-example count is 92. Corrected to a measured number.
7. **Churned on the cmdguard count** (41 -> 40 -> 37) because "consumer" was
   never defined. The errata now gives the exact command and exclusions, but
   the underlying report's definition remains unknown.
8. **First HTML edit attempt failed** because I had not viewed the copied
   file. One wasted tool call.
9. **Used `rg -rn` on reflex** and was correctly blocked by the hook. No file
   damage; one wasted call.
10. **Validated HTML with a naive parser** that reported false positives for
    `<meta>`. The real check (browser render) is still missing.
11. **Left the wrong premise live.** The errata identifies it, but
    `profileagent/STACK.md:61,89` still asserts it, so the next reader can
    inherit it.
12. **Reports now risk being entombed.** Both findings live in timestamped
    files; neither is in a TODO_LIST. The prior self-review's standing rule
    ("a plan older than two weeks with zero tasks executed gets executed or
    archived") applies to this work too.

## e) WHAT WE SHOULD IMPROVE!

1. **One claim, one command that prints the value.** No more `sort -V` or
   `head -1` on mixed tag families; no "verify" blocks that check a skill or
   a README phrase.
2. **State the definition next to every count.** Modules, consumers, pin
   kinds, and sentinels each need a one-line rule.
3. **Separate the decision premise from the citation.** "DedupStore is the
   default" is not the same claim as "the Store contract is superseded".
4. **Distinguish direct, indirect, and test-scoped pins** in every table.
5. **Apply non-owner-gated fixes in the same session** as the finding, or
   write the TODO row immediately. A report alone changes nothing.
6. **Harvest reports into living docs.** Both deliverables should land in
   TODO_LIST with owners, not just in `docs/status` and `docs/feedback`.
7. **Render-check HTML deliverables** in a browser before calling them done.
8. **Re-render the index after any living-doc edit**, and stamp each row with
   a last-verified date.
9. **Cross-check sibling documents before declaring an inconsistency.** Two
   of my three near-misses were context, not error.
10. **Write status reports in the repo whose work dominates the session**, or
    explicitly say "cross-repo". This session wrote reports into two repos
    and this status into a third context.

## f) Up to 50 things we should get done next

Sorted by impact, not by ease. Items 1-10 are the "do these and the corpus
stops lying" set.

1. Rewrite `profileagent/docs/review/libs/go-idempotency.md` stance with the
   true reason (no consumer of the contract today; DedupStore is an engine
   capability, not a replacement).
2. Fix `profileagent/STACK.md:61` ("Superseded by metaengine.DedupStore").
3. Fix `profileagent/STACK.md:89` (same claim in the rejected list).
4. Fix the index scorecard partition: 6 adopt/carry, 13 reject/defer,
   1 transitive; move the owner question out of the numbers.
5. Fix cqrs-htmx `v4.18.0 -> v4.18.1` in report and index.
6. Fix go-appkit's version field to root `v0.12.0` plus submodule tags.
7. Cite or correct cmdguard's 27-consumer claim.
8. Reconcile sentinel counts (portalapply 17; cloudbrowser 12; index 30+).
9. Define the module-count convention and regenerate every count.
10. Fix the circular and wrong verification commands (go-idempotency,
    cqrs-htmx, go-codec, go-appkit, go-output).
11. Re-render `profileagent/docs/review/libs/index.html`.
12. Add a last-verified timestamp per index row.
13. Add a verification-status block to each profileagent report.
14. Add a "known errata" banner to the index pointing at the feedback file.
15. Harvest the 13 errata findings into `profileagent/TODO_LIST.md`.
16. Harvest the 8 go-idempotency actions into `go-idempotency/TODO_LIST.md`.
17. Add a DedupStore-to-Store adapter recipe to `go-idempotency/doc.go`.
18. Add the same recipe to `docs/migrating-from-memorystore.md`.
19. Reword `go-idempotency/README.md:53` ("you do not need to write a
    backend" -> "use DedupStore directly, or wrap it with the adapter").
20. Correct `doc.go:60-62` with the same distinction.
21. Replace the contract suite's four `time.Sleep` calls with bounded polling.
22. Document `TimingScale` and the polling behavior in the README.
23. Document the record-on-success limitation and the `Seen` + `Record` trap.
24. Extend ADR-004's revisit triggers with webphone and PapDashboard evidence.
25. Decide `middleware.Command`'s fate (keep with a non-CQRS note, or delete at
    v1.0) and refresh ADR-005's 0/10 -> 0/13.
26. Add the "what this is / is not" block to the README and `doc.go`.
27. Fix `NewMemoryStore(sweepInterval)` documentation (sweep interval vs
    per-call TTL).
28. Cut `v0.3.3` with the test-split and CI fix; run
    `scripts/pre-release-check.sh` first.
29. Decide `Forever`: owner green light (M1-M4) or a REJECTED ADR.
30. If Forever ships: `const Forever`, MemoryStore zero-time, four contract
    invariants, negative stores, docs.
31. If Forever is rejected: record the REJECTED ADR and tell consumers to use
    domain permanence.
32. Add a runnable `example/replay` (or extend `example/main.go`).
33. Link the replay example from the README pitfalls section.
34. Add a fleet-adoption evidence section (13 direct, 8 contract, 0
    middleware) to `FEATURES.md` or the README.
35. Re-run `scripts/check-stale-refs.sh` after any go-idempotency doc edit.
36. Run `erraudit . --enforce-go-error-family` after any error-code touch.
37. Run `go test ./... -race` and `golangci-lint run ./...` before any release.
38. Browser-render the synthesis HTML and check anchors, tables, and print.
39. Link-check the new feedback report with lychee.
40. Run a markdown linter on `docs/feedback/new/`.
41. Validate the feedback report's command block end to end.
42. Add a CI/script check that compares report versions against local tags.
43. Add a CI/script check that pins the module-count convention.
44. Fix the go-cqrs-lite report's `NewSystem` cell wording
    ("profileagent's planned wrapper").
45. Fix the "Repo state: skeleton" header wording across the corpus.
46. Split "CV pin" into direct / indirect / test-scoped.
47. Replace the "~80 LOC" watcher estimate with a cited line span.
48. Re-verify the 13-project importer census and remove `games` if it is not
    an active adopter.
49. Add the review-corpus errata to the profileagent index as an owned,
    dated row rather than a silent feedback file.
50. Re-run this audit in two weeks and compare, per the standing plan-rot rule.

## g) Questions I cannot answer myself

1. **Forever:** do you green-light the 2026-09-16 ecosystem plan's M1-M4
   (`idempotency.Forever` as `time.Duration(math.MaxInt64)`, scope limited to
   consumer/event dedupe), or should I record a REJECTED ADR and close the
   question?
2. **Execution:** should I now implement the seven non-owner-gated
   go-idempotency items (adapter recipe, README/doc wording, contract
   polling, record-on-success docs, middleware decision, replay example,
   v0.3.3 release), or wait for explicit go-ahead?
3. **Profileagent corpus:** should I edit the reports, `index.html`, and
   `STACK.md` in place and annotate the old claims, or keep the feedback file
   as the only record until someone owns the fixes?

## Immediate next action

Waiting for instructions. The synthesis HTML (`fdbf9b1`) and the errata
report (`a7eb469`) were picked up by the auto-commit daemon; this status
report is the only untracked file.
