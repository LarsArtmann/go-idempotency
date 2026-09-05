# ADR-002: Middleware Module Boundary

## Status

Accepted (2026-08-29; the owner's veto window closed 2026-09-05 with the
explicit go-ahead to execute the v0.3.0 release train)

## Context

`doc.go` advertises a middleware package with `CommandIdempotency`,
`EventIdempotency`, and `QueryIdempotency` helpers that wire a `Store` into
CQRS dispatch pipelines. It is the primary advertised integration point, and
the PapDashboard consumer evaluation (feedback B3) named its absence as one
of three reasons they could not adopt. Before implementing it, the boundary
must be decided:

- **(a) Subpackage of this module** — `github.com/larsartmann/go-idempotency/middleware`
- **(b) Sibling module** — `github.com/larsartmann/go-idempotency-middleware` with its own go.mod
- **(c) Per-transport modules** — `.../middleware-http`, `.../middleware-grpc`, etc.

The tension: transport adapters (net/http middleware, gRPC interceptors)
pull in transport dependencies. This module's identity is a near-zero
dependency surface (`go-error-family` is the only runtime dep).

## Decision

**(a) Subpackage, stdlib-only, HTTP-first — with a hard rule: the middleware
subpackage may import nothing outside the standard library plus this module.**

1. `middleware` lives in this module as a subpackage. Zero new runtime
   dependencies: the core is transport-agnostic (wrap any `func(ctx, key,
   operation) error`-shaped dispatcher), and the HTTP adapter uses only
   `net/http` and the `Idempotency-Key` header.
2. If a future adapter genuinely requires a transport dependency (gRPC
   protobuf types, etc.), that adapter moves to its own module at that point —
   the core middleware API does not change, only where the adapter lives.
3. `EventIdempotency` and `QueryIdempotency` wait until a consumer needs
   them (YAGNI); the package doc marks them planned, not implemented.

## Rationale

- Subpackage preserves one `go get` for the common case and keeps the
  middleware versioned with the `Store` contract it integrates with —
  during v0.x, interface evolution and middleware often move together, and a
  sibling module would force lockstep multi-repo releases for no benefit.
- Sibling/per-transport modules buy dependency isolation we do not yet need:
  with the stdlib-only rule, the core module's dependency graph is unchanged.
- Splitting later is cheap; merging modules later is not. The rule makes the
  likely split point explicit (transport deps) instead of vague.

## Consequences

- The root module ships a `middleware` subpackage; its API is v0.x and may
  evolve in minor bumps like the rest of the SDK.
- gRPC/other transport adapters are out of this module's scope until they
  exist as separate decisions.
- Reversal cost pre-1.0: moving a subpackage to a sibling module is a
  mechanical change (new go.mod, updated import path) and reasonable to make
  at the first transport dependency, without an API break to the middleware
  contract itself.
