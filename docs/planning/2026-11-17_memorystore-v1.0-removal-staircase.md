# MemoryStore v1.0 Removal Staircase

Created 2026-11-17 (plan T7, from TODO_LIST). Modeled on go-etag's validated
v1.0.0 deletion plan. `MemoryStore` is deprecated since v0.2.0 with removal
targeted for v1.0; this is the concrete staircase so the removal is a
checklist, not an investigation.

## Fleet audit — who still uses it (2026-11-17)

`rg -l 'MemoryStore|NewMemoryStore' ~/projects --glob '*.go'` (excluding this
repo). Hits are *references*, not necessarily production dependencies — each
row must be re-verified per project before removal:

| Project        | Files referencing MemoryStore        | First impression                              |
| -------------- | ------------------------------------ | --------------------------------------------- |
| go-compliance  | `reporter.go`, `reporter_test.go`    | Production import — migrate first             |
| Standup-Killer | `domain/cqrs/*` (4 files, has `idempotency_sqlite.go`) | Mid-migration: SQLite backend exists, MemoryStore remnants remain |
| branching-flow | `pkg/ifacecomplete/analyzer_test.go` | Test-only reference                           |
| go-datastar    | `store.go`, `store_test.go`, `docspec_test.go` | Own `MemoryStore` type? Verify it is not go-idempotency's before counting |

## Preconditions (all must hold before the v1.0 tag)

1. **Zero external references**: re-run the fleet audit above; every hit is
   migrated or verified as an unrelated same-named type. Record per-project
   verdicts in this file.
2. **Replacement paths are live**: `idemtest.NewStore` (test double) and the
   migration guide (`docs/migrating-from-memorystore.md`) both point
   consumers at a blessed alternative — done (v0.5.0 kit, guide current).
3. **Deprecation notice is complete**: `MemoryStore` and `NewMemoryStore`
   carry `// Deprecated:` doc comments, README/FEATURES list them as
   deprecated, `docs/DOMAIN_LANGUAGE.md` lists the term as deprecated —
   done since v0.2.0, re-verify at removal time.
4. **One full release cycle with the deprecation as loud as possible**
   (godoc renders strikethrough; staticcheck flags usage) — satisfied many
   times over by v0.6.0.
5. **Contract suite self-test no longer depends on it**: `contract_test.go`
   (root) runs `contract.RunTests` against `MemoryStore` — before removal,
   re-point that self-test at `internal/teststore` (the kit's blessed
   backend) so the suite's certification story survives the deletion.
   Same for the godoc `ExampleMemoryStore` (delete with the type).

## Staircase

1. **Announce** — CHANGELOG note in the release before the last v0.x: "the
   next major version removes MemoryStore" (the existing Deprecated comments
   say this; make it a standalone release-notes line).
2. **Migrate the fleet** — fix go-compliance and Standup-Killer remnants
   (the only production imports found); test-only references are mechanical.
3. **Re-point internal users** — `contract_test.go` self-test →
   `internal/teststore`; delete `ExampleMemoryStore` in the same change;
   update `doc.go` Quick Start to lead with a custom-backend example.
4. **v1.0.0 tag** — module path stays `github.com/larsartmann/go-idempotency`
   (no v2 path bump is triggered by *deleting* a deprecated type that every
   consumer already migrated; the major version signals the breaking
   deletion itself). Delete `MemoryStore`, `NewMemoryStore`,
   `MemoryStoreOption`, `WithClock`, `isLive`'s MemoryStore-specific notes,
   and the `store_*_test.go` files that exercise only MemoryStore.
   `WithClock` dies WITH the store: it exists to pin MemoryStore's clock and
   has no other consumer.
5. **Post-removal sweep** — `rg -n 'MemoryStore'` over the repo must return
   only historical documents (CHANGELOG, status reports, ADRs); the
   migration guide flips to past tense or is deleted.

## Verification

Each step lands with `./scripts/pre-release-check.sh` green; step 4 is the
only breaking step and is the v1.0.0 tag itself.
