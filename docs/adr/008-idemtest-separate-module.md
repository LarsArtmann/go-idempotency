# ADR-008: `idemtest` Ships as a Separate Nested Module

## Status

Accepted (2026-11-17): executed the same day the kit landed (ADR-007),
mirroring go-sse's `ssetest`, which is itself a nested module
(`github.com/LarsArtmann/go-sse/ssetest` requiring its parent). The kit's
first release is `idemtest/v0.1.0`, tagged together with parent `v0.5.0`.

## Context

ADR-007 shipped the kit as a package of the single module, per ADR-005.
Two considerations pushed the kit — and only the kit — out of that shape:

- The LarsArtmann library-policy companion rule — "a project that uses
  go-idempotency must adopt `idemtest`" — is only machine-checkable when
  the kit appears as a distinct `require` line in a consumer's go.mod. A
  same-module subpackage is invisible to that detection: a require of the
  root can hide the kit's absence.
- The kit's production files import `testing` (every drill takes
  `testing.TB`). A module boundary makes the test-only posture structural:
  no consumer can accidentally compile the kit into a binary.

The split explicitly does NOT fire either of ADR-005's recorded revisit
triggers (no heavy `contract` dependency; no middleware-only import need);
it is a scoped exception for the test kit, whose detection semantics a
subpackage cannot provide.

## Decision

1. `idemtest/` is its own Go module: `idemtest/go.mod` declares
   `github.com/larsartmann/go-idempotency/idemtest` and requires the root
   module at `v0.4.0` — the newest tag whose root/`middleware`/
   `internal/teststore` surface the kit builds against unchanged (the kit
   adds no root API).
2. The root module stays single (ADR-005 unchanged for the root,
   `middleware`, and `contract`): no `go.work`, no version-qualified
   replaces, and the parent imports the kit nowhere, so root consumers are
   untouched.
3. The kit is released via `idemtest/v*` tags cut at the same commits as
   parent tags; `idemtest/v0.1.0` ships together with `v0.5.0`.
4. CI and `scripts/pre-release-check.sh` run
   build/test/vet/lint/tidy/modverify/govulncheck inside the nested module
   too — `./...` from the root cannot reach a nested module's packages, so
   without explicit steps the kit would ship untested. Dependabot tracks
   `/idemtest` as its own gomod ecosystem.
5. `internal/teststore` stays importable by the kit: Go's `internal` rule
   is lexical on import-path prefixes (anything under
   `github.com/larsartmann/go-idempotency/...` may import it), independent
   of module boundaries. External consumers still cannot.

## Consequences

- Consumers add the kit with
  `go get github.com/larsartmann/go-idempotency/idemtest`; the import path
  is the same string ADR-007 documented — only the module boundary moved.
  MVS lifts the root to whatever version the consumer also requires
  (v0.4.0 floor today).
- Two version trains now exist: cutting a release means one parent tag and,
  when the kit changed, one `idemtest/v*` tag at the same commit, pushed
  together (RELEASING.md). The nested tag does not trigger `release.yml`
  (`v*` pattern); its GitHub Release is cut manually as non-Latest.
- Requiring root `v0.4.0` rather than the to-be-cut `v0.5.0` is deliberate:
  a nested go.mod cannot require a version that does not exist at its own
  tag time.
- The kit's godoc renders at
  `pkg.go.dev/github.com/larsartmann/go-idempotency/idemtest` once
  `idemtest/v0.1.0` is indexed (separate page from the root module's).
