# Status Report — MemoryStore Deprecation Self-Critique

**Date:** 2026-08-07 22:30
**Scope:** This session only — the deprecation of `MemoryStore` (triggered by owner decision "deprecate it"). Covers what was done, what was missed, what is broken, and what comes next.
**Verdict:** 🟡 **PARTIALLY DONE.** The code-level `// Deprecated:` annotations landed cleanly and lint/build/test are green, but the deprecation is **inconsistent across docs** — I missed 4 files, including the `Store` interface doc comment itself, which still _steers users toward_ the deprecated type. A deprecation that is not uniformly applied is worse than none: it teaches readers the wrong default.

> **Format note:** The `status-report` skill canonicalizes HTML output. The user explicitly requested `.md` — the override wins per skill rules, and is flagged here so the divergence is visible.

---

## a) FULLY DONE

| #  | Item                                                                                  | Evidence                                                                     |
| -- | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| 1  | `// Deprecated:` doc comment on `MemoryStore` struct                                  | `store.go:78-84` — renders in `go doc`, IDEs, pkg.go.dev, staticcheck SA1019 |
| 2  | `// Deprecated:` doc comment on `NewMemoryStore` constructor                          | `store.go:90-99`                                                             |
| 3  | `doc.go` Design Philosophy reworded + Quick Start note                                | `doc.go:31-35`, `doc.go:12-16`                                               |
| 4  | `README.md` updated in 5 sections (design, quick start, features, status, versioning) | `README.md:46,71,131,166,168`                                                |
| 5  | `FEATURES.md` — new `## DEPRECATED` section + NOT PLANNED text                        | `FEATURES.md`                                                                |
| 6  | `ROADMAP.md` — versioning: v0.2.0 deprecates, v1.0 removes                            | `ROADMAP.md:28-30`                                                           |
| 7  | `CHANGELOG.md` — `### Deprecated` entry under `[Unreleased]`                          | `CHANGELOG.md`                                                               |
| 8  | `docs/adr/001-no-backends.md` — corrected "not deprecated" → "deprecated v0.2.0"      | `docs/adr/001-no-backends.md:16,52`                                          |
| 9  | `docs/DOMAIN_LANGUAGE.md` — MemoryStore entry marked deprecated                       | `docs/DOMAIN_LANGUAGE.md:38`                                                 |
| 10 | `TODO_LIST.md` — deprecation logged as done                                           | `TODO_LIST.md`                                                               |
| 11 | Build, vet, lint (0 issues), `-race` tests all pass                                   | verified via CLI                                                             |
| 12 | godoc renders the Deprecated notice correctly                                         | `go doc . MemoryStore` confirmed                                             |

---

## b) PARTIALLY DONE

### P1. 🔴 Deprecation is inconsistent across the codebase (THE core failure of this session)

I applied `// Deprecated:` and "deprecated" language to **10 files**, but **4 files were missed** and still describe `MemoryStore` as the default path. A reader hitting any of these learns the wrong thing:

| File:line                                     | Current text (stale)                                                                                                                    | Problem                                                                                                                                             |
| --------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`store.go:43-44`**                          | `// [MemoryStore] is provided as a reference implementation for development and single-process use cases;`                              | **Worst miss.** This is the `Store` interface doc — the canonical entry point. It actively recommends the deprecated type with no deprecation note. |
| **`CONTRIBUTING.md:7`**                       | `It provides the Store interface and MemoryStore (a reference implementation).`                                                         | Contributor-facing doc still frames MemoryStore as a co-equal deliverable. I never touched CONTRIBUTING.md this session.                            |
| **`AGENTS.md:22`**                            | `**MemoryStore** (store.go) — the reference implementation.`                                                                            | The Architecture section. I updated the "Key Design Decisions" bullets but missed the primary description above them.                               |
| **`example_test.go` / `contract_test.go:11`** | `ExampleMemoryStore` + `ExampleStore` both construct via `NewMemoryStore`; comment says "the reference implementation satisfies every…" | godoc examples actively demonstrate the deprecated API as the happy path. No `// Deprecated` acknowledgment, no "for production see…" pointer.      |

**Why this matters:** Deprecation only works if _every_ surface agrees. The `go doc` output now shows `Deprecated:` on the type — but the prose three lines up (in the interface comment) and the runnable examples still say "this is what you use." Mixed signals = no deprecation.

### P2. 🟡 Soft deprecation — no enforcement mechanism

- No staticcheck/`depguard`/`forbidigo` rule in `.golangci.yml` to flag _new_ uses of `NewMemoryStore`/`MemoryStore` outside `_test.go`.
- CI does not fail on deprecated-API usage.
- No build tag or `//go:build` mechanism to stage removal.

### P3. 🟡 The contract package depends on the deprecated type

`contract_test.go` exercises the contract suite against `MemoryStore(0)`. If `MemoryStore` is the thing being deprecated, the contract suite's _own_ conformance witness is the deprecated code. This is a real coupling question (see Questions).

### P4. 🟡 Removal target is asserted but not tracked

ROADMAP says "v1.0 removal" but there is no issue/TODO item with an owner or a gating checklist (e.g., "ship ≥1 backend example in a separate repo first").

---

## c) NOT STARTED (carried over + new)

| #  | Item                                                                   | Source                        |
| -- | ---------------------------------------------------------------------- | ----------------------------- |
| 1  | Fix the 4 stale-file misses above                                      | This session (P1)             |
| 2  | `contract/contract_test.go` self-test of the suite (fixes 0% coverage) | Prior critique                |
| 3  | SQL adapter example in `doc.go` (only Redis exists)                    | Prior critique                |
| 4  | Enrich fuzz seed corpus (empty strings, unicode, `math.MaxInt64`)      | Prior critique                |
| 5  | Codecov/Coveralls badge in README                                      | Prior critique                |
| 6  | Comment on `BenchmarkMemoryUsage_AfterSweep` re: low reclaim %         | Prior critique                |
| 7  | `ErrStoreClosed` sentinel                                              | Open question                 |
| 8  | Middleware package (`CommandIdempotency` etc.)                         | Blocked — module boundary     |
| 9  | `Delete` / `Stats` on `Store` interface                                | Blocked — interface evolution |
| 10 | Standalone migration guide doc                                         | This session                  |

---

## d) TOTALLY FUCKED UP

Nothing is _broken_ (build/lint/test green), but two things are **embarrassing for a senior pass**:

1. **I deprecated a type without grepping its name afterward.** A single `grep -rn 'reference implementation\|single.process\|NewMemoryStore' --include='*.go' --include='*.md'` — which I ran _after_ being asked to self-critique — immediately surfaced the 4 misses. I should have run that grep as the **last step** of the deprecation, not the first step of the post-mortem. This is the exact "deprecate X" checklist item I skipped.

2. **The `Store` interface doc comment is the one place that must never lag.** It is the most-read doc in the library. Leaving it pointing at the deprecated type while the type itself says "Deprecated" is a direct contradiction _inside the same file, ~30 lines apart_. That is the definition of a split brain.

> **LSP noise, not a real problem:** the editor shows 20 `golangci_lint_ls` warnings on `contract/contract.go` (gocognit 74, mnd magic numbers, wsl_v5). `golangci-lint run ./...` from CLI returns **0 issues**. These are **stale LSP cache** from the prior session's pre-split version. Not real — but I did not clear the cache or restart the LSP, so the next session will see the same ghost warnings.

---

## e) WHAT WE SHOULD IMPROVE

1. **Make "grep the symbol after editing it" a mandatory close-out step.** Deprecation, rename, or removal = run a full-text search for every alias before declaring done.
2. **Treat the `Store` interface doc as the single source of truth.** When the status of a type changes, update the interface comment _first_, then propagate outward.
3. **Add a deprecation lint gate** so future drift is caught by CI, not by a human post-mortem.
4. **Decouple the contract suite from `MemoryStore`** — ship a tiny internal test-only store so the suite is self-contained and doesn't validate itself against the code it's deprecating.
5. **Write the migration guide before, not after, announcing the deprecation.** Inline `// Deprecated:` is necessary but not sufficient; a worked example of "here is the 20-line Redis adapter you drop in" closes the loop.
6. **Track removals with a gating checklist**, not a prose "v1.0" line.

---

## f) Next things to get done (50)

Sorted by impact × effort. **P0 = fix the split-brain now.**

**P0 — Close the consistency gap (this session's debt)**

1. Fix `store.go:43-44` Store interface doc to point to deprecation, not "reference implementation."
2. Fix `CONTRIBUTING.md:7` to mark MemoryStore deprecated.
3. Fix `AGENTS.md:22` Architecture bullet to mark deprecated.
4. Rewrite `example_test.go`: keep `ExampleMemoryStore` but add `// Deprecated:` note + a `// See the "Implementing a Custom Backend" section in the package docs.` pointer; ensure `ExampleStore` notes the store is illustrative.
5. Update `contract_test.go:11` comment language ("reference implementation" → "deprecated in-process store").
6. Re-run the deprecation grep as the final verification step — zero stale "reference implementation"/"single-process use cases" outside intentional historical contexts.
7. Sweep `CHANGELOG.md:26` historical wording if it reads as current truth (judgment call).

**P1 — Make the deprecation enforceable** 8. Add `depguard`/`forbidigo` rule in `.golangci.yml`: block `NewMemoryStore`/`MemoryStore{}` outside `*_test.go`. 9. Add a staticcheck `SA1019` gate to CI (fail on new deprecated usage in non-test code). 10. Add a `// Deprecated:` migration checklist comment block in `store.go` near the type. 11. Write `docs/migrating-from-memorystore.md` — worked Redis + SQL adapter, `contract.RunTests` wiring.

**P2 — Contract suite hardening** 12. Create `contract/internal_test_store` — a minimal in-process Store the suite validates _itself_ against (fixes 0% coverage on `contract/`). 13. Add `contract/contract_test.go` that runs `RunTests` against the internal test store. 14. Add a _negative_ contract test: feed a deliberately broken Store and assert `RunTests` _fails_ (proves the suite can catch bugs). 15. Enrich fuzz seed corpus (`fuzz_test.go`): `""`, unicode, `math.MaxInt64`, negative TTL, very long keys.

**P3 — Documentation & examples** 16. Add SQL adapter example to `doc.go` alongside Redis (INSERT ... ON CONFLICT DO NOTHING). 17. Add DynamoDB PutItem adapter example. 18. Add a MongoDB adapter example. 19. Wire a Codecov/Coveralls badge into README (CI already uploads the artifact). 20. Add comment to `BenchmarkMemoryUsage_AfterSweep` explaining low %-reclaimed (Go maps don't shrink post-delete). 21. Add `go test -fuzz=.` and `go test -bench=.` to CONTRIBUTING.md dev setup. 22. Render the contract invariant list in README/CONTRIBUTING (what `RunTests` actually checks).

**P4 — Interface evolution (blocked on owner)** 23. Decide `Delete` method on `Store` (manual key invalidation). 24. Decide `Stats` method (hit/miss/expiry observability). 25. Decide `Reset` / `Clear` method. 26. Decide `ErrStoreClosed` sentinel vs. returning plain error post-Close. 27. Decide method return-shape for `CheckAndRecord` (does it return a stored result/etag?).

**P5 — Middleware layer (blocked on module boundary)** 28. Decide: middleware in this module, a sibling module, or a separate `go-idempotency-middleware` module. 29. Implement `CommandIdempotency` dispatcher wrapper. 30. Implement `EventIdempotency` handler. 31. Implement `QueryIdempotency` (cacheable, idempotent read). 32. Key generation utilities (UUID v7, content-hash, request-derived).

**P6 — Observability & perf** 33. Metrics hooks (hit/miss/expiry/contention). 34. Sharded-mutex benchmark vs single `sync.RWMutex`. 35. `sync.Map` evaluation. 36. Lock-free CheckAndRecord prototype. 37. Allocation-free hot path for `Seen`.

**P7 — Release hygiene** 38. Cut v0.2.0 tag once deprecation is consistent + enforceable. 39. Verify `go doc ./...` output end-to-end before tag. 40. Verify pkg.go.dev renders deprecation after tag publish. 41. Add `CHANGELOG.md` review to release checklist. 42. Add a `SECURITY.md` (currently absent). 43. Add issue/PR templates (`.github/`).

**P8 — Quality of life** 44. Add `make`/flake-free task doc (document the exact `go test`/`lint` commands in one place). 45. Add a `doc.go` table of contents for long package doc. 46. Add cross-links from `Store` methods to the contract invariant they satisfy. 47. Add a "backend feature matrix" (Redis/SQL/DynamoDB → which atomic primitive, which limitations). 48. Add a CONTRIBUTING "Adding a contract invariant" guide. 49. Add a `CODE_OF_CONDUCT.md`. 50. Add retry/backoff guidance for transient store errors (`errorfamily.IsRetryable`).

---

## g) Questions I CANNOT figure out myself (3)

1. **Contract-suite coupling.** The contract package validates _itself_ against `MemoryStore` (`contract_test.go`). Now that `MemoryStore` is deprecated, should the suite ship its own tiny internal test-only Store (decoupling it from the deprecated code), or is it fine to keep using `MemoryStore` as the conformance witness until v1.0 removal? This is a real design tradeoff (self-contained suite vs. testing against the very thing we're killing) — I won't pick for you.

2. **Deprecation enforcement aggressiveness.** Should CI _fail_ on any new use of `NewMemoryStore`/`MemoryStore` outside `_test.go` (via `depguard`/`forbidigo` + staticcheck `SA1019`)? Too soft and the deprecation is theater; too hard and you break consumers mid-migration. Where is the line?

3. **Removal mechanism.** ROADMAP says "v1.0 removal." Is the gating condition "≥1 reference backend published as a separate module," "middleware layer ships," or a calendar version? Without a concrete gate, "v1.0 removal" is a wish, not a plan.

---

**Bottom line:** The code change is correct and green. The _discipline_ of the deprecation — applying it everywhere before declaring done — failed. Fix P0 items 1-6 before anything else.
