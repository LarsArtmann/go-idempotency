# Releasing

Checklist for cutting a release of `github.com/larsartmann/go-idempotency`.
Worked end-to-end for v0.2.0 (2026-08-29).

## Pre-flight

1. **Decide the version** per [ROADMAP.md](ROADMAP.md) Versioning Strategy.
   - Breaking changes to the `Store` interface or error semantics require a major-version module path bump (v2, v3, ...).
   - New additive API (new methods, new packages) → minor bump (v0.x → v0.x+1 while pre-1.0).
   - Docs, tests, tooling only → patch bump.
2. **Confirm the working tree is clean** (`git status`) and everything intended for the release is on `master`.

## Verification gates (all must be green)

> Local-env note: if the default Go caches point at an unwritable path, prefix
> the commands with
> `GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache`.

```bash
gofmt -l .                      # must print nothing
go vet ./...                    # must be silent
go test ./... -race -count=1    # must pass; -race is mandatory (concurrency is the product)
golangci-lint config verify     # config must validate
golangci-lint run ./...         # must report 0 issues
```

3. **`go mod tidy` diff check** — run tidy on a *copy* of the module, never on
   the live tree (protects uncommitted work):

```bash
rm -rf /tmp/tidycheck && mkdir -p /tmp/tidycheck
cp -r go.mod go.sum *.go contract /tmp/tidycheck/
cd /tmp/tidycheck && go mod tidy
diff ../go-idempotency/go.mod go.mod && diff ../go-idempotency/go.sum go.sum   # must be identical
```

4. **Godoc render read-through** — `go doc .`, `go doc . Store`,
   `go doc . MemoryStore`, `go doc . NewMemoryStore`, `go doc . ErrInvalidTTL`,
   `go doc github.com/larsartmann/go-idempotency/contract`.
   Deprecation notices, error semantics, and examples must render correctly.

## Finalize the changelog

5. Diff `CHANGELOG.md`'s `[Unreleased]` section against
   `git log vPREV..HEAD` — every user-visible change listed, nothing wrong.
   Internal planning/status docs may be omitted.
6. Rename `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`, add the compare link
   (`[Unreleased]: .../compare/vX.Y.Z...HEAD`, `[X.Y.Z]: .../compare/vPREV...vX.Y.Z`).
7. Commit: `chore(release): finalize vX.Y.Z changelog and release checklist`.

## Tag and publish

8. Create the annotated tag (message = release-notes summary):

```bash
git tag -a vX.Y.Z -m "go-idempotency vX.Y.Z: <one-line summary>"
git push origin master vX.Y.Z
```

9. Confirm the module proxy indexed the new version (may take a minute):

```bash
go list -m -versions github.com/larsartmann/go-idempotency
```

10. Create the GitHub Release from the release notes
    (`docs/releases/vX.Y.Z-notes.md`), mark it **Latest**:

```bash
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file docs/releases/vX.Y.Z-notes.md --latest
```

11. `gh run list` — CI must be green on the tagged commit.

## Post-release

12. Verify pkg.go.dev renders the new version: deprecation notices, contract
    package, godoc examples. Capture proof (screenshot or link) in the release
    notes or status log; file any rendering issues.
13. If new work surfaced during release: add it to [TODO_LIST.md](TODO_LIST.md)
    and stage a fresh `## [Unreleased]` section in `CHANGELOG.md`.
