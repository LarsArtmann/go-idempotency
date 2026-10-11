# Security Policy

## Supported versions

| Version                      | Supported                                                          |
| ---------------------------- | ------------------------------------------------------------------ |
| latest release on pkg.go.dev | yes                                                                |
| older releases               | no — upgrade via `go get -u github.com/larsartmann/go-idempotency` |

Go's module system means consumers pin an exact version; fixes land in a new
tag, never retroactively.

## Reporting a vulnerability

**Do not open a public issue for a security vulnerability.**

1. Use GitHub's private vulnerability reporting:
   **Security → Report a vulnerability** on
   [LarsArtmann/go-idempotency](https://github.com/LarsArtmann/go-idempotency/security/advisories/new).
2. Include: affected version(s), a minimal reproduction, and your assessment
   of impact.

We respond within 7 days with an acknowledgement and follow up with a
fix-or-decision within 30 days for anything confirmed. Credit is given in
the release notes unless you prefer otherwise.

## Security Practices

- **Minimal attack surface.** The library performs no network I/O, no file
  I/O, and spawns no subprocesses; keys live in whatever storage the consumer
  implements. The only runtime dependency is `go-error-family`.
- **Continuous verification.** CI runs `govulncheck` on every push, fuzzes
  all four fuzz targets for 3 minutes each, and gates every change behind
  the race detector and the 13-invariant contract suite.
- **Fail-closed dispatch.** The middleware executes a command only when
  `CheckAndRecord` confirms the key is new; a store failure never degrades
  into "not a duplicate", because executing twice is the unrecoverable
  outcome.
- **Input validation at the boundary.** Non-positive TTLs are rejected with
  `ErrInvalidTTL` instead of silently recording an already-past expiry that
  would weaken the exactly-once guarantee.
- **Hardened releases.** Releases are built by CI from pinned commits; tags
  are annotated and pushed from `master` only after every gate is green.

## Scope notes

This library is a small, dependency-light SDK: the only runtime dependency is
`github.com/larsartmann/go-error-family` (error classification). There are no
network listeners, no file I/O, no subprocesses, and no production backends —
keys live in whatever storage the consumer implements. The realistic risk
surface is therefore:

- behavior around key/TTL validation that could weaken the exactly-once
  guarantee (a correctness bug, reported here as a security issue when it
  silently breaks dedup),
- a vulnerability in the single runtime dependency (covered by the
  `govulncheck` CI job),
- a compromised release tag (releases are built by CI from pinned commits;
  tags are annotated and pushed from `master` after all gates are green).
