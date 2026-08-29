# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks. When an item becomes bounded and estimable, it graduates to [TODO_LIST.md](TODO_LIST.md).

## Backend Implementations (Out of Scope by Design)

This library provides the `Store` interface and `MemoryStore` (a **deprecated** reference implementation for development and testing only). It **will not** ship production backends — Redis, SQL, or otherwise.

**Why:** Each backend carries its own driver dependency, connection-pool semantics, deployment constraints, and operational tradeoffs. Bundling them here would bloat the dependency tree and impose decisions that consumers should own. The interface is intentionally small (three methods) so that implementing your own backend is straightforward.

**Implementation guidance for your own backend:**

- Read the `Store` interface and the atomicity contract on `CheckAndRecord` in `store.go`.
- Use your backend's native atomic primitive: Redis `SET NX EX`, SQL `INSERT ... ON CONFLICT DO NOTHING`, etc.
- Use the `contract` package (`contract.RunTests`) to verify your implementation against the same invariants as `MemoryStore`.

## Ecosystem Integration

- **Middleware/dispatch layer** — the `doc.go` package docs reference `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the integration point for CQRS pipelines. First step is in [TODO_LIST.md](TODO_LIST.md); the broader vision includes transport-agnostic middleware that works with HTTP, gRPC, and message queue consumers.
- **Key generation utilities** — helpers for producing stable idempotency keys (UUID v7, content-hash-based, request-derived). Currently every consumer must roll their own key strategy.

## Observability

- **Metrics hooks** — instrument hit/miss/expiry/contention rates so operators can tune TTL and sweep intervals based on real traffic patterns.
- **Lock strategy optimization** — baseline benchmarks exist (`bench_test.go`); next step is evaluating sharded mutexes, `sync.Map`, and lock-free approaches against the current single-`sync.RWMutex` design under high contention.

## In-Process Store Evolution (from PapDashboard consumer evaluation)

Raw ideas surfaced by the 2026-08-18 consumer evaluation in `docs/feedback/new/2026-08-18_13-07_papdashboard-evaluation-dedup-only-stopped-adoption.md` (evaluated v0.1.2, did not adopt — see the report for the full reasoning):

- **Response-replay composition recipe** — the store answers "seen or duplicate" only; HTTP-style consumers (Stripe semantics) need a retried request to return the original response, not a 409. A documented dedup+replay pattern (e.g., a thin `ResponseCache` wrapped around `Store`, ~20 lines in `doc.go`) would close this gap without polluting the key-only `Store` contract.
- **Bounded in-process store — or a documented position.** Deprecating `MemoryStore` removed the only in-process option, so single-process apps roll their own (LRU + TTL). Either ship a `BoundedStore(maxEntries, ttl)` (swap the map for an LRU, validated by `contract.RunTests`) or document restart-durability as table stakes for idempotency and point single-process apps to what they should do instead.

## Versioning Strategy

- **v0.2.0 (planned; content complete on `master`, not yet tagged)** — contract test suite, implementation examples (Redis adapter), fuzz tests, memory benchmarks, godoc examples, ADR-001. All docs updated to interface-first SDK framing. `MemoryStore` formally deprecated (dev/test only).
- **v0.x** — API may change between minor versions. `MemoryStore` is deprecated; the `Store` interface may evolve as real-world use reveals missing methods (e.g., `Delete`, `Stats`, `Reset`).
- **v1.0** — `MemoryStore` removed; the interface stabilizes (proven by multiple independent backend implementations in the wild) and the middleware layer ships.
