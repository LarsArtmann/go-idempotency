# ADR-003: Bounded In-Process Store vs Documented Restart-Durability Position

## Status

Accepted (2026-08-29): option (b) — documented position; no `BoundedStore`

## Context

The PapDashboard consumer evaluation (feedback B2) rejected the library
partly because nothing shippable remains for single-process production use:
`MemoryStore` is deprecated (correctly — it loses keys on restart and grows
without bound), and their replacement was ~60 lines using
`hashicorp/golang-lru/v2/expirable`. The question: should this module ship a
`BoundedStore(maxEntries, ttl)` (LRU + TTL, validated by `contract.RunTests`)
to win that segment back, or document a position?

## Decision

**(b) Document the position; do not ship `BoundedStore`.** The position,
documented in README/doc.go:

1. **Restart durability is table stakes for production idempotency.** A store
   that forgets on restart re-executes every command retried after the crash.
   Any single-process production deployment therefore needs its claims in
   something durable (SQLite, bbolt, a table in the existing database) — at
   which point the consumer is implementing `Store` against a real backend
   anyway, which is this module's whole model.
2. **An LRU is a correctness lie for idempotency claims.** Bounded eviction
   means the store can forget a live claim under load — and a forgotten claim
   is a silently re-executed command. An LRU-capped idempotency store is a
   heuristic, not an exactly-once guarantee; shipping one from this module
   would put the library's name on the failure mode PapDashboard was right to
   reject.
3. Consumers who genuinely want bounded in-process behavior (dev, load
   testing, best-effort dedup) can write it in ~60 lines — and validate it
   with `contract.RunTests` like any other backend. The migration guide and
   the runnable `example/` show the shape.

## Considered and rejected: (a) ship `BoundedStore`

- It contradicts ADR-001's spirit: it IS a backend (storage policy choices —
  eviction, sizing, memory accounting), just an in-memory one.
- Eviction semantics cannot be both "bounded" and "exactly-once"; documenting
  that tradeoff inside a shipped component invites misuse of a type wearing
  this module's guarantee-laden name.
- Real demand pointed the other way: PapDashboard's actual production need
  was durability (SQLite), not bounding.

## Consequences

- The "single-process production" segment is served by documentation only:
  the position above, the migration guide, and the response-replay recipe.
- ROADMAP's "In-Process Store Evolution" idea is resolved as documented
  position; no v0.3.0 code work.
- If a future consumer demonstrates a durable single-process need that is
  truly generic (e.g., a maintained SQLite adapter), that is a new decision —
  and per ADR-001 it starts as its own module, not code here.
