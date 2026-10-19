# Status Report — go-etag Cross-Repo Learning Review

**Date:** 2026-10-19 06:14 (Monday)
**Session type:** Analysis / cross-repo research — read-only, no code changes
**Scope:** Single-session review of `../go-etag` (structure, modularization, testing-for-downstream patterns) against `go-idempotency`, per user question: *"What can we learn from ../go-etag/? The go-modularization structure? Having a DDD based testing system for downstream users?"*
**Author:** Crush session (glm-5.3)
**Format note:** written as `.md` at explicit user demand (status-report skill default is HTML dashboard — override flagged in session output).

---

## Executive Summary

The session delivered a complete comparative analysis with a clear verdict: **do NOT adopt go-etag's five-module structure** (wrong fit, documented reasons + a future trigger that would change the answer), **the DDD-testing question is direction-reversed** (go-idempotency's `contract/` suite is ahead of anything go-etag has), and **five concrete patterns are worth stealing** (release gate script, release workflow, benchmark baseline discipline, DOMAIN_LANGUAGE upgrades, spec-citation rigor). Nothing was implemented — the session ended at the recommendation boundary awaiting user go-ahead. Three claims in the delivered analysis were labeled "verify" instead of being verified; closing them is the top follow-up work.

---

## a) FULLY DONE

Verifiable by the sources read and the tool outputs in this session's transcript.

| # | What | Evidence |
|---|------|----------|
| 1 | Loaded `go-modularize` skill and applied its decision framework (anti-signal scoring, composability-payoff test) to the split question | Skill read in full; framework cited in verdict |
| 2 | go-etag structural survey: 5-module layout (root tombstone, `entitytag/`, `server/`, `client/`, `metrics/`), `go.work` with version-qualified replaces, version-train + tag-staircase mechanics | `go-etag/go.work`, `go-etag/go.mod`, full `go-etag/AGENTS.md` read |
| 3 | go-etag testing-pattern survey: native-t.Run BDD (`etag_bdd_test.go`), RFC-citing spec suite (`client/spec_test.go`), AST export-parity tests (`server/entity_parity_test.go`), real-wire integration tests, shim-parity suite | File heads + AGENTS.md Testing Conventions read |
| 4 | go-etag `docs/DOMAIN_LANGUAGE.md` analyzed: Where-citation column, Bounded contexts section, Deprecated terms section — the three features go-idempotency's glossary lacks | Full file read (50 lines); compared to go-idempotency's 51-line flat-table version |
| 5 | go-idempotency dependency audit: `go-error-family` = production (part of the Store contract itself), `rapid` = test-only; **proved `contract/` imports neither** (rg over `contract/*.go`: zero matches) | `go.mod` + rg output in session |
| 6 | go-idempotency repo-inventory: `scripts/` contains only `check-stale-refs.sh`; `.github/workflows/` contains only `ci.yml` (no release workflow); `docs/adr/` has 4 ADRs; `docs/DOMAIN_LANGUAGE.md` exists | `ls` outputs in session |
| 7 | Consumer-fleet import survey (partial — see b/1): DiscordSync + cqrs-htmx import root only; KeyHolderAI + bank-sync import root + `contract`; 10+ consumers found pinning v0.3.x | go.mod grep across `~/projects` + per-project import greps |
| 8 | Delivered final comparative verdict with rationale, future modularization trigger, and ranked steal-list; offered to implement top-2 items | Session output (chat) |

---

## b) PARTIALLY DONE

| # | What works now | What remains open | Blocker | Effort |
|---|----------------|-------------------|---------|--------|
| 1 | Consumer import survey covered 4 consumers fully (DiscordSync, cqrs-htmx, KeyHolderAI, bank-sync) | 6+ more consumers unswept (Zlota44, Cyberdom, crm, go-localsync, accountability-system, file-and-image-renamer); **middleware importers never counted** — the "nobody needs a middleware module" claim rests on a sample | None — one command | S |
| 2 | Release-gate comparison delivered as recommendation ("verify RELEASING.md covers annotated-tag/proxy-hash/clean-room gates") | `RELEASING.md` was **never opened** — the recommendation ships unverified | None | S |
| 3 | Real-wire middleware test recommended | Claim "middleware/http_test.go is probably recorder-based" was **not checked against the file** | None | S |
| 4 | middleware→root dependency asserted (blocks the "split buys nothing" argument) | Inferred from AGENTS.md architecture notes; `middleware.go`/`http.go` imports **not read this session** | None | S |
| 5 | Modularization rationale summarized | go-etag's proposal doc `docs/modularization/2026-09-23_modularization-proposal.html` (records "metrics = tombstone purity, not composability") relied on via AGENTS.md summary, **not read directly** | None | S |

---

## c) NOT STARTED

All recommended in the delivered analysis; zero work begun — the session deliberately stopped at the recommendation boundary ("Want me to implement any of these?").

| # | Planned | Not started because | Still wanted? |
|---|---------|--------------------|---------------|
| 1 | Port go-etag's `scripts/pre-release-check.sh` (one-command local release gate: build→vet→race→lint→fmt-diff→erraudit→mod-verify) | Awaiting user go-ahead | Yes — best effort/payoff |
| 2 | Port go-etag's `release.yml` (auto GitHub Release from CHANGELOG section, fail-loud if not cut) | Same | Yes |
| 3 | Benchmark baseline convention (`reports/bench/<date>_<name>.txt`, `-count=6 -benchmem`, before/after perf changes) | Same | Yes |
| 4 | DOMAIN_LANGUAGE.md upgrades: Where-citation column, Bounded contexts, Deprecated terms | Same | Yes |
| 5 | Spec-citation pass: every contract invariant cites its authority (TOCTOU, claim semantics, ADR) | Same | Medium value |
| 6 | Real-wire `httptest.Server` integration test for the middleware HTTP adapter | Same | Medium value |
| 7 | ADR-005: record the modularization rejection + future trigger as a decision | Same | Yes — otherwise this session's core verdict lives only in chat |
| 8 | go-directive ↔ CI-pin parity check (go-etag's `969d077` daemon-relaxation incident class; go-idempotency `go.mod` currently `go 1.27`, unanchored) | Noticed during session, not yet scoped | Yes |
| 9 | HARVEST of this report's section (f) into `TODO_LIST.md` (docs-health) | User instructed: report, then wait | Yes |

---

## d) TOTALLY FUCKED UP

Nothing data-losing — the session was read-only and nothing shipped. Radical honesty about process failures:

| # | What went wrong | Severity | Root cause | Mitigation |
|---|-----------------|----------|-----------|------------|
| 1 | **First consumer-sweep command failed** — mangled rg flags in a shell loop printed rg's help text instead of results; one full tool roundtrip wasted | Low | Sloppy argument quoting (`--glob` placement after path) | Recovered immediately with `grep --include`; should have used the Grep tool first |
| 2 | **Three unverified claims landed in the final recommendation table** (RELEASING.md coverage, recorder-based middleware tests, middleware import graph) — labeled "verify" instead of verified | Medium — credibility risk; a user acting on them inherits unexamined assumptions | Stopped researching one step short of source truth; deferral dressed as diligence | Items b/2–b/4; rule going forward: no "probably" in a final recommendation — verify or stamp UNVERIFIED |
| 3 | **Session output persisted nowhere** until this report — the modularization-rejection rationale, the future trigger, and the fleet-survey data existed only in chat | Medium — a session crash would have erased the session's entire value | Treated delivery-to-chat as completion | This report + proposed ADR-005 |
| 4 | **Fleet-level claims from a 4-consumer sample** ("nobody imports middleware") stated with more confidence than the evidence supports | Medium | Sampling without scoping the claim to the sample | Full sweep (task f/4) or re-scope wording |

---

## e) WHAT WE SHOULD IMPROVE

Process and design improvements observed in this session's run:

1. **Verify-before-recommend.** Every claim in a final recommendation table should be source-verified in the same session. "Check RELEASING.md" is a TODO for me, not a deliverable for the user. Impact: recommendation trustworthiness. Fix: b/2–b/5 close in under 15 minutes total — they should have been closed before answering.
2. **Scope claims to evidence.** Sample-of-4 conclusions must be worded as samples ("of the 4 consumers surveyed") or expanded to the full population. Impact: prevents wrong architectural decisions built on overreach.
3. **Persist analysis at formation time.** Cross-repo verdicts (especially negative decisions with future triggers) belong in an ADR the moment they're formed, not after the user asks for a status report. Fix: ADR-005 now (task f/6).
4. **Command hygiene.** Quote tool args carefully inside loops; prefer the Grep/Glob tools over ad-hoc bash rg/grep for first attempts. Impact: fewer wasted roundtrips.
5. **Surface interpretation risk explicitly.** "DDD based testing system" was interpreted (as: contract suite + domain-language grounding) rather than confirmed; the entire second half of the answer hinged on that reading. A one-line hypothesis statement up top would have made disagreement cheap. (Also raised as question g/2.)
6. **Read the primary source over the summary.** The modularization-proposal HTML was summarized in go-etag's AGENTS.md and I stopped there. AGENTS.md summaries are excellent but are one editorial pass away from the decision record. Impact: low here, but the habit matters at decision weight.

---

## f) Next Tasks (ranked; feeds docs-health HARVEST)

Sorted by impact, then effort. Impact: C=Critical, H=High, M=Medium, L=Low. Effort: S<30min, M=30min–2h, L>2h.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Close b/2: read `RELEASING.md`, diff its gates against go-etag's validated release protocol (annotated tag after green CI on exact commit, proxy `.info` hash == commit, clean-room `go get`, CHANGELOG-cut-before-tag), record deltas | H | S | Documentation |
| 2 | Close b/1: full consumer-fleet import sweep (all 10+: root / middleware / contract usage matrix) — turns the sampled claims into fleet facts | H | S | Research |
| 3 | Close b/3–b/4: read `middleware/http_test.go` + `middleware.go` + `http.go`; confirm real-wire coverage and the middleware→root dependency edge | M | S | Quality |
| 4 | Port `scripts/pre-release-check.sh` from go-etag: one gate for build, vet, race, lint, fmt-diff, erraudit (`--enforce-go-error-family`), tidy-diff, stale-refs, GOCACHE env handling | H | M | Quality |
| 5 | Add `release.yml`: auto GitHub Release extracting the CHANGELOG section, failing loudly when not cut; adapt go-etag's 53-line workflow | H | M | Feature |
| 6 | Write ADR-005: modularization rejected for go-idempotency — rationale (single domain, dep profiles don't diverge, consumers all need root, contract has no extra deps) + future trigger (contract gaining a heavy test dep ⇒ extract `contract/` module) | H | S | Documentation |
| 7 | Add go-directive ↔ CI-pin parity check to the release gate (anchored exact-match, go-etag's anti-daemon-relaxation defense; current `go 1.27` is unanchored) | M | S | Quality |
| 8 | Pre-record the MemoryStore v1.0-removal mini-staircase (go-etag's v1.0.0 root-deletion plan is the template): deprecation audit → removal commit → major-version plan | H | M | Documentation |
| 9 | Consumer fleet audit for `MemoryStore` usage before v1.0 (`rg MemoryStore ~/projects` over go.mod consumers) | H | S | Research |
| 10 | DOMAIN_LANGUAGE.md: add Where-citation column to every term (go-etag pattern) | M | S | Documentation |
| 11 | DOMAIN_LANGUAGE.md: add Bounded contexts section (Claim = middleware vocabulary; Seen/Record/CheckAndRecord = Store vocabulary) | M | S | Documentation |
| 12 | DOMAIN_LANGUAGE.md: add Deprecated terms section (MemoryStore) | L | S | Documentation |
| 13 | Benchmark baseline convention: capture `reports/bench/<date>_<name>.txt` with `-count=6 -benchmem` before/after perf-relevant changes; document in AGENTS.md | M | S | Quality |
| 14 | Spec-citation pass over `contract/contract.go`: each of the 13 invariants cites its authority (TOCTOU, claim semantics, ADR-00x) in the test name/comment | M | M | Quality |
| 15 | Real-wire integration test for the middleware HTTP adapter: `httptest.Server` round-trip pinning header canonicalization + fail-closed behavior over TCP | M | M | Quality |
| 16 | Check CHANGELOG for current released version; compare against fleet pins (v0.3.x found in survey) — decide whether an ecosystem sweep (go-ecosystem-upgrade skill) is due | M | S | Research |
| 17 | HARVEST this section into `TODO_LIST.md` (docs-health HARVEST mode) | M | S | Documentation |
| 18 | Close b/5: read go-etag's `docs/modularization/2026-09-23_modularization-proposal.html` primary source; confirm the AGENTS.md summary added no drift | L | S | Research |
| 19 | Dedupe check: grep `docs/status/` archive for prior go-etag / modularization mentions so this report's findings merge rather than duplicate | L | S | Documentation |
| 20 | Verify whether the GOTOOLCHAIN=local gotcha (go-etag's env trap) also applies here; extend the AGENTS.md env-gotcha block if so | L | S | Documentation |
| 21 | Consider recording the "contract-suite never swallows store errors" rule and the 13-invariant map as a consumer-facing doc (docs/invariants.md) — the go-etag `rfc9111-conformance.md` pattern applied to our own spec | M | M | Documentation |
| 22 | If release.yml is adopted: watch its first live run end-to-end (go-etag's own rule — untested-by-default workflows earn trust only on first fire) | M | S | Feature |

22 items — every one concrete and session-derived. Not padded toward the 50 cap; the remainder of that budget belongs to ROADMAP fuel if the user wants a brainstorm.

---

## g) Questions (cannot be answered from the repo)

1. **Was `paste_1.txt` intentional?** It attaches a full planning-workflow template (comprehensive plan → 100-min/30-min → 12-min task breakdowns → mermaid/d2 execution graph at `docs/planning/` → git commit & push). It conflicts with this message's "report, then wait, don't research unrelated stuff." Should I run that planning workflow over these findings (plan → breakdown → graph → commit/push), or was it a leftover paste?
2. **What did you mean by "DDD based testing system for downstream users"?** I read it as the `contract/` suite + domain-language grounding — my answer was "we already lead; steal spec-citation." If you meant something specific (e.g., a Ginkgo/Gomega BDD suite consumers run against their backend, or a consumer-facing conformance document), the recommendation changes materially.
3. **Is a go-idempotency release planned soon?** It decides whether the `release.yml` + `pre-release-check.sh` ports (tasks 4–5) are urgent-now or backlog — and whether the go-directive anchoring (task 7) should ride the same commit.

---

*Point-in-time snapshot. Do not rewrite; annotate via docs-health ANNOTATE mode when superseded. Section (f) is HARVEST input for TODO_LIST.md / ROADMAP.md.*
