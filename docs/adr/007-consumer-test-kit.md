# ADR-007: Consumer Test Kit (`idemtest/`)

## Status

Accepted (2026-11-17): the owner commissioned the kit directly ("ship a domain
driven test kit for apps and libraries that leverage us, like go-sse
provides"), which is the demand-side evidence ADR-002/ADR-005 style decisions
ask for.

## Context

go-sse ships [`ssetest`](https://github.com/LarsArtmann/go-sse/tree/main/ssetest)
— a test kit whose one-liners spin up real infrastructure, drive a consumer's
actual wiring, and hand back typed, assertable results. It exists because
every consumer otherwise hand-rolls the same scaffolding.

go-idempotency consumers are in exactly that position today:

- The 2026-11-17 consumer audit synthesis (docs/reviews) found three fleets
  hand-writing store bridges and test doubles, and named the HTTP adapter as
  plausible-but-unwired in three more.
- `MemoryStore` is deprecated (ADR-001) and `internal/teststore` is
  unimportable, so a consumer testing their integration has no blessed store
  and must first write ~60 lines of one.
- The failure-path and concurrency drills every consumer needs — fail-closed
  on store errors, exactly-one-winner under a retry storm, duplicate → 409 —
  exist only as private test code inside this repository
  (`middleware/failingStore`, the barrier-race harnesses, the httptest
  drivers). Every consumer rewrites them.

## Decision

1. **Ship `idemtest` as a package of the single module** (path
   `github.com/larsartmann/go-idempotency/idemtest`). It is not a new module:
   its imports are the standard library, the root package, and `middleware`
   (itself stdlib-only per ADR-002), so neither ADR-005 revisit trigger
   (heavy test dependency; middleware-only import path) fires.

2. **Contents, all named in the domain language** (claims, duplicates, retry
   storms, executions — not raw store calls):

   - `NewStore()` — a blessed test `Store` that delegates to
     `internal/teststore` and is continuously certified by
     `contract.RunTests` in this repository's CI. It is the answer to
     "MemoryStore is deprecated but I need a store in my tests."
   - `Fail(next, err)` — failure injection for fail-closed drills.
   - `NewRecorder(next)` — a spy recording every `Store` call, so wiring bugs
     (wrong key derivation, wrong TTL) are caught as assertions, not silences.
   - `Executions` — a side-effect counter with `Fn`/`Wrap` adapters for
     `middleware.NewCommand` and `Counting` for `http.Handler`.
   - Command drills over any `func(ctx, key) error` dispatch (which
     `middleware.Command` satisfies structurally): `RequireExactlyOnce`,
     `RequireOneWinner` (barrier race), `RequireFailClosed`,
     `RequireAllowsAfterExpiry` (bounded poll, never a bare sleep — the
     shape the consumer audit recommended for timing-based tests).
   - HTTP drills: `Send`/`SendConcurrently` drive a wrapped handler and the
     `Require*` status assertions name the adapter's contract (duplicate,
     missing key, store unavailable).

3. **A test store is not a backend.** ADR-001 forbids production backends; a
   test double in a test-only package follows the `httptest.Server` precedent
   — infrastructure for tests, never a deployment choice. Like `contract`,
   the package is imported from consumers' `_test.go` files only, so it never
   enters a production import path. This is not a `MemoryStore`
   resurrection: the root package keeps no in-memory store, and the kit's
   store gains no production surface (no sweep tuning, no lifecycle beyond
   `Reset` for phase isolation).

4. **The kit's assertions are sabotage-proven.** Extending the repository
   convention that every contract invariant ships with a broken-store
   negative scenario, every `Require*` helper ships a negative test proving
   it fails against the violation class it names (a dispatch that skips the
   store, a fail-open pipeline, a TOCTOU `Seen`+`Record` split).

## Consequences

- Consumers get ssetest-style one-liners for the two integration surfaces
  (command dispatch, HTTP adapter) and a store for tests without writing one;
  the audit's "copyable reference implementation" ask is answered by an
  importable, CI-certified one instead.
- The kit's drills accept plain `func(context.Context, string) error` rather
  than `middleware.Command`, so raw `CheckAndRecord` call sites (the
  go-cqrs-lite bridge shape) drill through the same helpers without
  importing `middleware`.
- HTTP drills run handlers through `httptest.ResponseRecorder` in-process —
  no sockets, no ports — because idempotency responses are single-shot
  (unlike SSE, which needs a live streaming server).
- The kit does not re-export the contract suite; backend implementers keep
  importing `contract` directly, and the kit's README points there.
- Surface discipline: `idemtest` gains no store implementation of its own
  (delegation to `internal/teststore` keeps one implementation), and no
  timing-injection clock — the polls are bounded waits with tiny
  consumer-chosen TTLs, the same trade the contract suite already makes.
