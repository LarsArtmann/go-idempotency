# ADR-005: Single Module and Anchored Toolchain Floor

## Status

Accepted (2026-11-03): the owner approved executing the 2026-10-19 pareto
plan, which specified this ADR's content; the provisional owner-veto window
closed with that approval. Records two related structural decisions: the
module stays single, and the `go` directive stays anchored.

## Context

The go-etag cross-repo learning review (docs/status/
2026-10-19_06-14_go-etag-cross-repo-learning-review.md) raised two
structural questions:

1. Should this module adopt go-etag's five-module layout (root tombstone
   plus `entitytag/`, `server/`, `client/`, `metrics/` with a `go.work` and
   version-qualified replaces)? go-etag split because independent consumers
   import its subpackages at different release cadences. The review's
   verdict for go-idempotency was "wrong fit", pending the fleet evidence
   below.
2. Is the `go 1.27` directive safe from silent relaxation? go-etag suffered
   exactly that class (commit `969d077`: an automated commit changed the go
   directive; CI pins the toolchain via `go-version-file`, so nothing
   failed). This module had the same exposure: the floor was set in v0.3.1
   with no mechanical anchor.

Fleet evidence (2026-10-19 sweep of all 10 known consumers):

- Root-only: DiscordSync, Zlota44, cqrs-htmx, file-and-image-renamer
- Root + contract: KeyHolderAI, bank-sync
- Indirect-only (via go-cqrs-lite): Cyberdom, accountability-system,
  go-localsync, crm
- **Zero consumers import `middleware/`.**

Dependency evidence: the `contract` package's only imports are the standard
library and this module's root package (verified via `go list -f`), so
consumers running the suite pull nothing beyond the root's single runtime
dependency (`go-error-family`); `rapid` is needed only by this module's own
tests.

## Decision

1. **Single module.** go-idempotency stays one Go module. Triggers to
   revisit, either of which is sufficient:
   - the contract suite gains a heavy test dependency consumers would
     otherwise not pull (then `contract/` becomes its own module), or
   - a consumer demonstrates a concrete need for a middleware-only import
     path.
2. **The toolchain floor stays `go 1.27`, anchored.** Changing it is a
   conscious act that must update `scripts/pre-release-check.sh`'s
   `EXPECTED_GO_DIRECTIVE` and this ADR in the same commit.

## Consequences

- Consumers get root, `middleware`, and `contract` in one module.
  `middleware` is stdlib-only per ADR-002, so the module is already minimal;
  a split would add version-train and `go.work` machinery while buying
  nothing any of the 10 fleet consumers asked for.
- The floor has a documented adoption cost: consumers pinned one Go release
  behind cannot upgrade. wise-go pins `go 1.26.0` on purpose (nixpkgs
  toolchain line; a 1.27 bump there is recurring sweep damage), and issue
  #13 records the full rejection analysis. The cost is accepted: the floor
  tracks the latest stable toolchain so the module does not accumulate
  legacy workarounds, and the wise-go case is a non-adoption by design
  (Wise owns the dedup contract server-side), not a lost consumer.
- An automated commit or dependency sweep that silently changes the go
  directive now fails the release gate loudly instead of passing CI.
