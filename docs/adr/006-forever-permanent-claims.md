# ADR-006: Forever Claims (Permanent Idempotency Windows)

## Status

Accepted (2026-11-17): owner green-lit the `idempotency.Forever` semantic
from the 2026-09-16 ecosystem plan
(`cv/docs/planning/2026-09-16_18-52_forever-ttl-ecosystem-execution-plan.md`,
M1-M4).

## Context

The `Store` contract is TTL-windowed by design (`ttl <= 0` returns
`ErrInvalidTTL`). That is correct for request-facing idempotency: a permanent
claim on a crashed request would be a permanent lockout (ADR-004).

A different class of consumer needs the opposite guarantee:

- **Event/consumer dedupe.** A redelivered event must not fold twice, ever.
  CV shipped a hand-rolled forever store after a TTL expiry caused exactly
  that corruption (`CV/career-pipeline/eventstore/idempotency.go`).
- **Permanent domain invariants.** "An invoice may only be submitted once"
  (invoices' KSeF path) and a completed task's permanent dedup key
  (go-taskqueue) are not retry windows; a TTL would re-admit the duplicate
  after expiry.
- **Projection replay and outbox consumers.** The duplicate window is the
  lifetime of the store, not the lifetime of a request.

Without a library-owned sentinel, every consumer invents one, and the
semantics diverge: some keep keys forever, some use a 100-year TTL, some
silently treat `ttl == 0` as forever. The 2026-11-17 consumer audit corpus
identifies this as the most common reason consumers stay off this library.

## Decision

### 1. `Forever` is an explicit sentinel

```go
const Forever time.Duration = 1<<63 - 1 // math.MaxInt64
```

- `Forever` is a positive `time.Duration`, so it passes the existing
  `ttl <= 0` guard. `ttl == 0` remains `ErrInvalidTTL`: a forgotten TTL must
  stay loud.
- The sentinel is scoped to consumer/event dedupe and other permanent
  invariants. Request-facing `Idempotency-Key` windows stay TTL-windowed.
  The constant's doc, the README FAQ, and the package docs all state this.

### 2. In-process representation is zero time

`MemoryStore`, `internal/teststore`, and the example's `demoStore` store
`time.Time{}` (the zero value) to mean "never expires".

- Every liveness check is `exp.IsZero() || now.Before(exp)`.
- The sweep skips zero-time entries; otherwise the first tick would delete
  every permanent claim.
- `Record` never upgrades a live short-TTL claim to permanent and never
  downgrades a live permanent claim to short-TTL. The existing no-extension
  rule applies in both directions.

### 3. Adapters store the sentinel directly

Implementations must not compute `now.Add(Forever)`.

`time.Time.Add` is representable to roughly year 2318, but converting to
`UnixNano` overflows int64 in 2262. Backends that store epoch nanoseconds or
compute an absolute expiry from the TTL must special-case the sentinel:

| Backend | Encoding |
| --- | --- |
| Redis | `SET key 1` with no `EX`; `Seen` is `EXISTS`; `Record` is `SET NX`; `CheckAndRecord` is `SET NX` |
| PostgreSQL / MySQL | `expires_at IS NULL` means forever; live means `expires_at IS NULL OR expires_at > now()` |
| SQLite | `expires_at` NULL (or a fixed max sentinel stored directly); never render `now.Add(Forever)` into a TEXT expiry |
| DynamoDB | Omit the TTL attribute; TTL deletion is best-effort anyway, so `Seen` must still check the expiry |
| In-process | zero `time.Time` |

### 4. The contract suite pins the semantics

Five invariants:

- `Seen/ForeverSurvives`
- `Record/ForeverNeverExpires`
- `Record/ForeverNoopOnExistingKey` (short first, Forever second: no upgrade)
- `Record/ForeverNotDowngraded` (Forever first, short second: no downgrade)
- `CheckAndRecord/Forever` (first claim succeeds, second returns
  `ErrDuplicate`)

Each ships with a broken-Store scenario in the negative suite, per the
existing pairing rule.

## Alternatives rejected

1. **`ttl == 0` means forever.** Rejected: it silently converts a forgotten
   TTL into a permanent claim, the exact bug the current validation prevents.
   The 2026-09-16 plan lists this as a non-goal.
2. **A huge finite TTL (100 years).** Rejected: still a TTL, still re-admits
   the duplicate at the boundary, and it hides the intent. The CV ADR already
   rejected this workaround.
3. **A separate `PermanentStore` interface.** Rejected: a second interface
   would force every backend to implement two contracts and every caller to
   type-switch. A sentinel value keeps one three-method contract.
4. **`Delete`/release instead of forever.** Rejected for this use case:
   release is for poisoned request claims (ADR-004, still deferred), not for
   permanent event dedupe.
5. **Zero time in the contract's public surface.** Rejected: zero time is an
   implementation detail of in-process stores; adapters may use NULL or a
   max sentinel. The public surface is only the `Forever` Duration.

## Consequences

- `Store` keeps three methods and two sentinels; the addition is additive
  under SemVer pre-1.0 (next minor release).
- The contract suite grows from 13 to 18 invariants and from 19 to 24
  deliberately broken stores. Every count-bearing doc must move in the same
  change.
- Backend authors gain one documented rule: map `ttl == Forever` at the
  boundary, never arithmetic.
- Request-facing misuse is documented against, not prevented: the library
  cannot tell an event key from an HTTP request key. The scope split is a
  documentation contract.
- Downstream propagation is follow-up work: go-cqrs-lite's
  `metaengine.DedupStore` / `claimkit` and its map runtime must map the
  sentinel, and CV/DiscordSync converge on the library semantic.
