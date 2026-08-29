# ADR-004: Store Interface Evolution

## Status

Accepted (2026-08-29): per-item decisions below; `Delete` deferred pending
owner confirmation

## Context

The `Store` interface has three methods and two sentinels. Several additions
have been requested or considered. Each carries the same cost: a new method
on `Store` must be implemented by every backend, and the contract suite must
pin its semantics. Deciding now (for v0.3.0 planning) prevents ad hoc growth.

The owner additionally raised a domain objection to claim invalidation:
"idempotency is supposed to be forever" — deleting a claim reopens duplicate
processing for that key, and a `Delete` on the interface invites request-path
misuse even if documented otherwise. That objection is real input, not noise,
and it demotes `Delete` from "obvious accept" to "needs a demonstrated need".

## Decisions

### `Delete(ctx, key) error` — DEFERRED (revisit on demonstrated need)

The use case is operational: in the claim-then-process model, a crash after
`CheckAndRecord` wins leaves a live claim that swallows retries until TTL
expiry, and `Delete` is the manual recovery. But:

- The owner's objection stands: an invalidation method on the core interface
  legitimizes reopening claims from ordinary code, and every backend author
  would carry that surface forever.
- The gap has cheaper mitigations that do not change the interface: tune TTL
  per key, store a failure marker (response-replay recipe), or use a
  backend-specific invalidation in ops tooling where it belongs.
- Trigger to revisit: a consumer (or we) hits repeated poisoned-claim
  incidents that TTL tuning cannot absorb. Then `Delete` lands in a minor
  release with: idempotent semantics (nil on missing), contract subtests
  (delete existing / missing / expired / race with `CheckAndRecord`), and
  docs scoped to operational recovery — never request paths.

### `Stats() (...)` — REJECTED for now

Observability without a production deployment that needs it. Metrics hooks
and introspection are ROADMAP ideas; they pay off when there are production
deployments to observe. Also, any stats shape hardens backend capabilities
into the interface (counts? bytes? hit rate?) long before evidence exists.

### `Reset() error` — REJECTED

A test-isolation concern. Backends that want bulk clearing should expose it
on their concrete type (or reuse their backend's native flush), not in the
interface every production backend must implement. The contract suite gets
fresh stores via the factory; it does not need `Reset`.

### `ErrStoreClosed` — DEFERRED until a backend needs it

Post-`Close` behavior is currently documented per implementation
(`MemoryStore` remains usable; `Close` only stops the sweeper). A
closed-store sentinel only becomes necessary when a backend genuinely cannot
serve after `Close` (pooled connections torn down). Codifying it now would
force a semantics choice without an implementation to validate it against.

### `CheckAndRecord` return shape `(claimed bool, err error)` — REJECTED

The current shape — nil, `ErrDuplicate`, or `ErrInvalidTTL`, all checked via
`errors.Is` — is settled, wraps cleanly, and maps to HTTP statuses through
`go-error-family`. A bool would force every caller to branch twice and
duplicates `errors.Is`'s job. Changing it now would be a breaking change
with no benefit.

## SemVer framing

Nothing in this ADR lands in v0.3.0; the interface is unchanged. If
`Delete` is later accepted, it is additive (minor bump while pre-1.0), ships
with contract + negative-test updates and README invariant-table rows in the
same PR, and is documented as operational recovery, not request-path API.
