# v0.3.0 Release Execution Status

> **Snapshot:** 2026-09-05 ~14:15 CEST · **Branch:** `master` · **Scope:** the owner-gated remainder of the archived v0.3.0 pareto plan, executed after the owner's explicit go-ahead ("execute the TODO list") closed the ADR veto window the same day the soak week elapsed.

## Executive summary

**v0.3.0 is released and verified end-to-end.** The middleware package, the proven-detection contract suite, the runnable example, the recipes, and the governance docs are on pkg.go.dev; the module proxy serves the tag; the GitHub Release is live; CI is green on the tagged commit. The TODO list is harvested down to two trigger-gated feature items, one blocked-on-owner infrastructure item, and the optional announcement.

## What was executed (in order)

1. **ADR veto window closed → statuses settled** (`e78f173`). ADR-002 and ADR-004 drop their provisional markers; the ADR index shows every status settled; both 2026-08-29 planning snapshots (v0.2.0 hardening plan, v0.3.0 pareto plan with its closing execution record) archived to `docs/planning/archived/`. The lychee docs job now excludes `docs/status` and `docs/planning` with the same historical-snapshot policy as check-stale-refs, so links inside snapshots to archived paths cannot fail CI.
2. **Fuzz budget raised 30 s → 3 min per target** (`523107c`). Gate honored: the 2026-08-29 soak (4×15 min, ≈860M execs) was clean, the soak week elapsed 2026-09-05, and all four targets were re-verified locally (90 s each: FuzzCheckAndRecord 28.4M, FuzzConcurrentMixed 22.8M, FuzzDispatch 21.6M, FuzzRecord 20.8M execs — ≈93.6M total, zero findings) before raising. `FuzzConcurrentMixed` joined the CI rotation it was missing.
3. **Changelog finalized + release notes staged** (`e543db3`). `[Unreleased]` diffed against `git log v0.2.0..HEAD`; gaps closed (ADR-index/docs batch, property tests + error-family API locks, middleware godoc example, CI job expansion, a Changed section for the checkout bump and fuzz raise); section renamed `[0.3.0] - 2026-09-05` with fresh `[Unreleased]` placeholders and compare links; `docs/releases/v0.3.0-notes.md` written in the v0.2.0 format.
4. **RELEASING.md tidy-check recipe fixed** (`dd48290`, auto-committed). The recipe predates `internal/teststore` and failed on a healthy module; it now copies every package directory.
5. **All verification gates green before tagging**: `gofmt -l` silent, `go vet` silent, `go test ./... -race -count=1` pass, `golangci-lint config verify` + `run` 0 issues, tidy-diff identical, `go run ./example` walks the three-attempt demo, godoc read-through renders, stale-refs clean.
6. **Tag + publish.** Annotated tag `v0.3.0` on `dd48290`, pushed together with master per RELEASING.md step 8 — load-bearing, because the lychee docs job checks the CHANGELOG compare links that 404 until the tag exists.
7. **Post-release sync** (`93982fc`). README middleware link flipped back to its pkg.go.dev page; ROADMAP marks v0.3.0 released; `doc.go`'s stale "middleware planned, not yet implemented" paragraph (exposed by the pkg.go.dev render) corrected to describe the shipped package; check-stale-refs learned the two new release-status phrases ("staging in CHANGELOG", "goes live with the next"); TODO_LIST harvested.
8. **Dependabot refresh merged**: PR #7 (setup-go 5.6.0 → 7.0.0 — clears the Node 20 deprecation warnings), PR #8 (golangci-lint-action 8.0.0 → 9.3.0), PR #5 (upload-artifact 4.6.2 → 7.0.1). PR #6 (checkout) had been merged 2026-09-04.

## Proof, not assumption

| Claim | Evidence |
| --- | --- |
| CI green on the tagged commit | Run [33970380542](https://github.com/LarsArtmann/go-idempotency/actions/runs/33970380542): success, 10/10 jobs, on `dd48290` = `v0.3.0` |
| Proxy indexed the tag at the right commit | `proxy.golang.org/.../@v/v0.3.0.info` → `{"Version":"v0.3.0","Hash":"dd48290410cba9d95c10769d01df85bf284df862","Ref":"refs/tags/v0.3.0"}` |
| Consumers can `go get` it | Clean-module `go get github.com/larsartmann/go-idempotency@v0.3.0` resolves, pulling only `go-error-family v0.10.0` |
| pkg.go.dev renders the release | [v0.3.0 module page](https://pkg.go.dev/github.com/larsartmann/go-idempotency@v0.3.0): deprecation notices, godoc examples, `contract`/`example`/`middleware` directories; [middleware page](https://pkg.go.dev/github.com/larsartmann/go-idempotency/middleware) renders `Command`/`NewCommand`/`HTTP`/`HeaderKey` + `ExampleNewCommand` |
| GitHub Release live | [v0.3.0 Release](https://github.com/LarsArtmann/go-idempotency/releases/tag/v0.3.0), notes from `docs/releases/v0.3.0-notes.md`, marked Latest (matching the v0.2.0 precedent: v0.x releases are not flagged prerelease) |
| Fuzz clean before the budget raise | Local 90 s × 4 targets ≈ 93.6M execs, zero findings, no `testdata/` additions |

## Remaining open

- **`CODECOV_TOKEN`** — still unset (owner-only); the coverage floor stays blocked on it.
- **Announce v0.3.0** — owner preference (channel/timing); the middleware package plus the detection proofs are the story.
- **Trigger-gated features** — `EventIdempotency`/`QueryIdempotency` (consumer demand) and `Delete` (demonstrated poisoned-claim need), per ADR-002/ADR-004.

## Notes for the next release

- Push master and the tag in ONE push (lychee checks the CHANGELOG compare links).
- Do not link a not-yet-released pkg.go.dev page from living docs (the stale-refs pattern `goes live with the next` now guards the placeholder wording).
- The auto-commit daemon may commit release-prep files before you do; check `git log origin/master..master` before writing commit messages.
