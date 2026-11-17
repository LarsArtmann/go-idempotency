# Releasing

Checklist for cutting a release of `github.com/larsartmann/go-idempotency`.
Worked end-to-end for v0.2.0 (2026-08-29).

## Pre-flight

1. **Decide the version** per [ROADMAP.md](ROADMAP.md) Versioning Strategy.
   - Breaking changes to the `Store` interface or error semantics require a major-version module path bump (v2, v3, ...).
   - New additive API (new methods, new packages) → minor bump (v0.x → v0.x+1 while pre-1.0).
   - Docs, tests, tooling only → patch bump.
   - The nested `idemtest` module (ADR-008) has its OWN version train (`idemtest/v0.x.y`): bump it whenever the kit changed since the last `idemtest/v*` tag, independently of the parent bump.
2. **Confirm the working tree is clean** (`git status`) and everything intended for the release is on `master`.

## Verification gates (all must be green)

3. **Run the one-command release gate** — `./scripts/pre-release-check.sh`
   (exit 0 = all gates green). The script pins writable cache paths itself,
   so the `/mnt/buildcache` login-shell gotcha cannot bite. It encodes:
   gofmt, `go vet`, `go build`, `go test ./... -race -count=1`, golangci-lint
   (config verify + run), `go mod tidy` diff on a copy (never the live
   tree), `go mod verify`, erraudit (`--enforce-go-error-family`),
   stale-refs, the anchored `go 1.27` directive in BOTH go.mod files
   (ADR-005), tag↔CHANGELOG parity, and a warn-only clean-tree check. The
   vet/build/test/lint/tidy/modverify gates run inside the nested `idemtest`
   module too (ADR-008): `./...` from the root cannot reach it. The parity
   gate is the structural fix for the v0.3.1 changelog gap and understands
   both release states: cut-but-untagged (the newest section has no tag yet)
   and tagged. A staged release therefore reads as PASS, not as a failure to
   ignore.

4. **Godoc render read-through** — `go doc .`, `go doc . Store`,
   `go doc . MemoryStore`, `go doc . NewMemoryStore`, `go doc . ErrInvalidTTL`,
   `go doc github.com/larsartmann/go-idempotency/contract`, and (nested
   module, own pkg.go.dev page) `go doc github.com/larsartmann/go-idempotency/idemtest`.
   Deprecation notices, error semantics, and examples must render correctly.

## Finalize the changelog

5. Diff `CHANGELOG.md`'s `[Unreleased]` section against
   `git log vPREV..HEAD` — every user-visible change listed, nothing wrong.
   Internal planning/status docs may be omitted.
6. Rename `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`, add the compare link
   (`[Unreleased]: .../compare/vX.Y.Z...HEAD`, `[X.Y.Z]: .../compare/vPREV...vX.Y.Z`).
7. Commit: `chore(release): finalize vX.Y.Z changelog and release checklist`.

## Tag and publish

8. Create the annotated tags (message = release-notes summary). When the
   kit changed, cut the nested tag at the SAME commit and push everything in
   **one** push:

```bash
git tag -a vX.Y.Z -m "go-idempotency vX.Y.Z: <one-line summary>"
git tag -a idemtest/vA.B.C -m "go-idempotency idemtest/vA.B.C: <one-line summary>"  # only when idemtest changed
git push origin master vX.Y.Z idemtest/vA.B.C
```

    Push the branch and every release tag in **one** push. `CHANGELOG.md`'s
    `compare/vPREV...vX.Y.Z` link 404s until the tag exists, so a branch-only
    push turns the Docs job red on the release commit (the link check runs
    against `HEAD`, where the tag is not yet reachable). If that happens —
    for example an auto-commit daemon pushed master first — push the tag and
    re-run the failed job; never delete the compare link to quiet the check:

    ```bash
    gh run rerun <run-id> --failed
    ```

9. Confirm the module proxy indexed the new versions (may take a minute;
   the nested module is a separate proxy entry under its own path):

```bash
go list -m -versions github.com/larsartmann/go-idempotency
go list -m -versions github.com/larsartmann/go-idempotency/idemtest
```

    Then verify the proxy serves them and a clean-room consumer can resolve
    each (no local cache hits; fresh module in a temp dir):

```bash
curl -s "https://proxy.golang.org/github.com/larsartmann/go-idempotency/@v/vX.Y.Z.info"   # must return Version + Time
curl -s "https://proxy.golang.org/github.com/larsartmann/go-idempotency/idemtest/@v/vA.B.C.info"
tmp=$(mktemp -d) && cd "$tmp" && go mod init cleanroom && go get github.com/larsartmann/go-idempotency@vX.Y.Z github.com/larsartmann/go-idempotency/idemtest@vA.B.C
grep go-idempotency go.mod   # must pin exactly vX.Y.Z and idemtest vA.B.C
```

10. The GitHub Release is created automatically: the tag push triggers
    [`.github/workflows/release.yml`](.github/workflows/release.yml), which
    extracts the CHANGELOG section for the tag (failing loudly when it is
    absent — the same parity rule the pre-release gate checks) and publishes
    the Release as **Latest**. `idemtest/v*` tags do NOT match the `v*`
    trigger — cut that Release manually as NON-Latest so the parent stays
    Latest. Manual fallback if the workflow cannot run:

```bash
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file docs/releases/vX.Y.Z-notes.md --latest
```

11. `gh run list` — CI **and** the Release workflow must be green on the
    tagged commit. Untested-by-default workflows earn trust only on their
    first live fire: watch this run end-to-end for the first release after
    2026-11-03. The Docs job is the one expected to be red before the tag
    (see step 8) — re-run it after the push and confirm it turns green.

## Post-release

12. Verify pkg.go.dev renders the new version: deprecation notices, contract
    package, godoc examples. Capture proof (screenshot or link) in the release
    notes or status log; file any rendering issues.
13. If new work surfaced during release: add it to [TODO_LIST.md](TODO_LIST.md)
    and stage a fresh `## [Unreleased]` section in `CHANGELOG.md`.
