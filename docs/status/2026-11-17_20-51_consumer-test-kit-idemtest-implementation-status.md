# Consumer Test Kit (`idemtest`) — Implementation Status

**Date:** 2026-11-17 20:51
**Scope:** This session only — the owner-commissioned "domain driven test kit like go-sse provides", from design to shipped-and-verified. No unrelated research.

---

## What was commissioned

> "We should ship a domain driven test kit for apps and libraries that leverage us like /home/lars/projects/go-sse provides!"

## What was delivered (FULLY DONE, verified green)

### 1. Research & grounding

- Loaded the `how-to-golang` skill (mandatory Go trigger).
- Studied go-sse's `ssetest` pattern end to end (README, doc.go, collect.go): consumer-facing one-liners, typed assertions, `testing.TB` everywhere, test-only module/packaging rationale.
- Read this repo's surface before designing: `store.go`, `middleware/*`, `contract/*`, `internal/teststore`, root `doc.go`, README, `.golangci.yml` (full linter set), CHANGELOG, ADR-005 (single-module rule), ADR index.
- Dispatched a subagent to extract `docs/reviews/2026-11-17_consumer-audit-synthesis.html`; its findings shaped the design: the blessed-test-store gap (three fleets hand-rolled bridges), the poll-don't-sleep recommendation for timing drills, "copyable store" ask, and middleware/'s zero-fleet-importer status.

### 2. Design decision — ADR-007 (`docs/adr/007-consumer-test-kit.md`, accepted)

- `idemtest` is a **package of the single module** (stdlib + root + `middleware` only; neither ADR-005 revisit trigger fires).
- Test stores are **infrastructure, not backends** (httptest precedent) — ADR-001's no-backends stance unchanged.
- Drills take plain dispatch funcs so raw `CheckAndRecord` integrations need no `middleware` import.
- Sabotage-proof convention extends from `contract/` to the kit.
- ADR index row added.

### 3. The package (`idemtest/`, 7 files, all under the 350-line cap)

| File            | Surface                                                                                                                                                                                                                                         |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `doc.go`        | ssetest-style package doc: the three problems it solves, quick starts, composition with `contract`                                                                                                                                              |
| `store.go`      | `NewStore()` — blessed test double delegating to `internal/teststore` (one implementation, no split brain); `Reset()` for phase isolation                                                                                                       |
| `failing.go`    | `Fail(next, err)` + `Failing` — failure injection; `Err = nil` mid-test models the store healing                                                                                                                                                |
| `recorder.go`   | `NewRecorder(next)` spy, `Call{Op,Key,TTL,Err}`, `Calls()` copy, `RequireClaimed`                                                                                                                                                               |
| `executions.go` | `Executions` counter: `Fn`/`Wrap` (command adapters), `Counting` (HTTP adapter), `RequireExecutions`                                                                                                                                            |
| `drill.go`      | `Dispatch` (type **alias** — see lessons), `RequireExactlyOnce` (retry storm), `RequireOneWinner` (barrier race / TOCTOU detector), `RequireFailClosed`, `RequireAllowsAfterExpiry` (bounded poll, 2ms step / 100ms grace — never a bare sleep) |
| `http.go`       | `Send` (recorder-based, in-process, no sockets), `SendConcurrently` (barrier storm), `WithMethod`/`WithBody`/`WithHeader`, `RequireStatus`, `RequireDuplicate` (409), `RequireMissingKeyRejected` (400), `RequireStoreUnavailable` (503)        |

### 4. Tests — 43 tests, 87.9% statement coverage

- `idemtest_test.go` — unit: store claims/reset, Fail fail+pass-through, Recorder order/errors/copy, Executions Fn/Wrap/Counting, RequireClaimed.
- `drill_test.go` — happy paths incl. a raw `CheckAndRecord` closure through `RequireOneWinner` and `Failing` mid-test recovery.
- `http_test.go` — first/duplicate/missing-key/unavailable/independent-keys, request options, 50-request concurrent storm.
- `negative_test.go` — **16 sabotage proofs** via an in-process `captureTB`/`expectDrillFailure` harness: store bypass, mid-storm error, TOCTOU split (with realistic 2ms gap), fail-open, sentinel misreport, stuck (Forever) claim, wrong key/TTL claims, and every HTTP status assertion. Every `Require*` is proven to fail against the violation class it names.
- `contract_test.go` — `contract.RunTests` certifies `NewStore()` continuously in CI.
- `example_test.go` — 4 godoc Examples (TB-taking drills can't be Examples; doc.go quick start carries those — recorded in AGENTS.md).

### 5. Documentation

- README: new "Testing your integration" section (store + 4 drills + HTTP walkthrough + spy), Features bullet, Documentation link (relative — pkg.go.dev link would 404 until the next tag).
- FEATURES.md: two FULLY_FUNCTIONAL rows (kit + HTTP drills) with evidence.
- CHANGELOG `[Unreleased]`: two Added entries.
- DOMAIN_LANGUAGE.md: new "Testing" table (test kit, dispatch, retry storm, one winner, executions, fail-closed drill, blessed test store).
- AGENTS.md: architecture bullet (incl. the type-alias rationale), test-suite inventory, sabotage convention extension — genuine refresh, satisfies the 14-day age gate honestly.
- Root `doc.go`: paragraph pointing at the kit.

### 6. Gates — all green at close

`go build` / `go test -race ./...` (all 6 packages) / `go vet` / `golangci-lint run ./...` (0 issues) / `gofmt -l` clean / `erraudit . --enforce-go-error-family` (0 violations) / `check-stale-refs.sh` clean / `dprint check` on touched markdown / `go mod tidy` diff no-op / no new dependencies.

---

## Mistakes made this session (all caught by gates, none shipped)

1. **`Dispatch` declared as a named type, not an alias** — two named func types are not mutually assignable in Go, so `middleware.Command` would not have passed to the drills. The compiler caught it; fixed as `type Dispatch = func(...)`. Documented in the declaration and AGENTS.md so it cannot regress.
2. **makezero fix regression** — changing `make([]string, len)` to `make([]string, 0, len)` in `formatCalls` left the indexed assignment `parts[i] = ...`, panicking at runtime. My own tests caught it one run later; fixed to append.
3. Undefined helper references (`joinComma`, `newRecordingWriter`) — instant compile catches.
4. An over-clever anonymous-interface parameter in `http_test.go` — replaced with plain `idempotency.Store`.
5. Stale-refs tripwire fired on the ADR quoting the audit's "reference implementation" phrase — rephrased.
6. Two lychee traps caught before CI could hit them: a pkg.go.dev URL for the not-yet-tagged package (→ relative link) and go-sse `tree/main` vs actual branch `master`.
7. Process noise: ghost LSP diagnostics (the exact `/mnt/buildcache`-class staleness AGENTS.md warns about) cost several verification round-trips before I trusted only the CLI.

---

## PARTIALLY DONE / gaps I noticed at close

- **Root `doc.go` Contents list** does not enumerate the kit (the pointer paragraph exists, but the TOC bullet list was not extended with a "Testing" heading).
- **Audit finding 4 (contract suite wall-clock sleeps)** — the kit implements the poll shape, but the `contract/` suite itself still sleeps; I scoped it out and did not check whether TODO_LIST already carries it.

## NOT STARTED (deliberate or missed — see next-work list)

- TODO_LIST.md / ROADMAP.md were never read this session (a protocol miss against the project-discovery checklist; nothing was harvested or cross-checked).
- CONTRIBUTING.md was never read (package conventions may say more than AGENTS.md).
- `example/` walkthrough does not demonstrate the kit.
- `docs/migrating-from-memorystore.md` does not point test-migration readers at `idemtest.NewStore`.
- `scripts/pre-release-check.sh` was not run as one command (all its relevant components were run manually; `go mod verify` and the parity gates were not).
- No fuzz/property/bench targets for the kit (root and middleware have them; I never consciously decided and recorded the YAGNI).
- `go-structure-linter` not run locally against the new package (known-findings section suggests it only surfaces via BuildFlow).
- middleware's private `failingStore` scaffolding not migrated onto `idemtest.Fail`.

## TOTALLY FUCKED UP

Nothing user-visible. Two design/compile mistakes (named-type assignability, the formatCalls regression) were made and caught by the gate chain within the session. No reverted user work, no broken main, no gate left red.

---

## What we should improve (consolidated)

1. **`RequireAllowsAfterExpiry` cannot detect a too-short claim** — if the store freed the key instantly, the first poll retry succeeds and the drill passes. A stronger drill asserts a duplicate _inside_ the window before polling for the reclaim.
2. **No TimingScale escape hatch on the expiry drill** — fixed 100ms grace; a pathological CI runner could flake. The contract suite solved this with `Options.TimingScale`.
3. **`SendConcurrently` ignores `RequestOption`s** — inconsistent with `Send`; a body-carrying concurrent storm is currently impossible.
4. **Negative harness asserts only the failed flag** — `captureTB.msg` is captured but never asserted; messages could rot while the flag stays green.
5. **`Failing` has no per-method failure selection** — YAGNI'd silently; deserves an explicit recorded decision.
6. **`Counter`-style naming consistency** — `Executions` vs `Recorder` vs `Failing` is fine, but `Counting` (HTTP) returns a bare handler; a `(handler, exec)` pair shape was rejected for Go taste — worth revisiting if consumers stumble.

## Next work (ranked, up to 50 — realistically 24 solid items)

**Kit hardening**

1. Strengthen `RequireAllowsAfterExpiry` (assert duplicate within window first).
2. Add a timing-scale option (or env knob) to the expiry drill for slow runners.
3. Accept `RequestOption`s in `SendConcurrently`.
4. Assert failure-message fragments in the sabotage harness (at least per drill class).
5. Decide + record: fuzz target for drills? property tests? benchmarks? (currently none).
6. Decide + record: per-method `Failing` selection, `Slow` store (latency injection) for timeout drills.
7. Cover the remaining 12% of statements (mostly guard paths) or document why not.

**Docs & cross-links**
8. Extend root `doc.go` Contents with a Testing entry.
9. Point `docs/migrating-from-memorystore.md` (test half of the migration) at `idemtest.NewStore`.
10. Add a kit demo to `example/` (one `RequireExactlyOnce` + one HTTP storm in the walkthrough).
11. Read + harvest TODO_LIST.md and ROADMAP.md against this session (protocol miss).
12. Read CONTRIBUTING.md; reconcile any conventions I unknowingly diverged from.
13. After the next release tag: swap the README relative link to the pkg.go.dev page (the lychee-safe window).

**Repo-internal cleanup**
14. Migrate `middleware/middleware_test.go`'s private `failingStore` and its hand-rolled barrier harness onto `idemtest` doubles (decide dependency direction first — question below).
15. Audit finding 4: poll-instead-of-sleep inside `contract/` itself (kit already models the shape).
16. Audit finding 3 (record-on-success/`Delete`): owner-level revisit of ADR-004; kit would then need a release drill.
17. Audit finding 7: make the response-replay recipe runnable (`example/replay`); kit `Send` gives it a test surface for free.
18. Run `go-structure-linter` locally on `idemtest/`; record any new known-findings.
19. Run `scripts/pre-release-check.sh` once as a single command (validates the gate chain end-to-end incl. `go mod verify`).

**Release & adoption**
20. Decide v0.5.0 timing (the kit is the headline; CHANGELOG section already drafted).
21. Clean-room consumer smoke: in a scratch module, `go get` this repo @master and run one drill — verifies the kit imports cleanly from outside.
22. Announce the kit to the three fleets that hand-rolled bridges (bank-sync/webphone-class adapters, per the audit) — the kit deletes their scaffolding.
23. Propagate to the go-sse playbook: this repo now has both patterns (contract suite + consumer kit); the pairing is reusable across LarsArtmann libs.

## Questions for the owner (cannot be self-answered)

1. **Release timing:** cut v0.5.0 with the kit now, or batch it with open audit findings (poll-based contract suite, replay example) first?
2. **Dependency direction:** may `middleware`'s own tests import `idemtest` (dedup of `failingStore`/barrier harness), or should the package's self-tests stay independent of the kit it is tested by?
3. **Kit scope:** do you want latency injection (`Slow`) and per-method failure selection now, or strictly YAGNI until a fleet consumer asks?
