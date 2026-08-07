# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks. When an item becomes bounded and estimable, it graduates to [TODO_LIST.md](TODO_LIST.md).

## Backend Implementations (Out of Scope by Design)

This library provides the `Store` interface and `MemoryStore` (a reference implementation for development and single-process use). It **will not** ship production backends — Redis, SQL, or otherwise.

**Why:** Each backend carries its own driver dependency, connection-pool semantics, deployment constraints, and operational tradeoffs. Bundling them here would bloat the dependency tree and impose decisions that consumers should own. The interface is intentionally small (three methods) so that implementing your own backend is straightforward.

**Implementation guidance for your own backend:**
- Read the `Store` interface and the atomicity contract on `CheckAndRecord` in `store.go`.
- Use your backend's native atomic primitive: Redis `SET NX EX`, SQL `INSERT ... ON CONFLICT DO NOTHING`, etc.
- A `Store` interface contract test suite (planned, see [TODO_LIST.md](TODO_LIST.md)) will let you verify your implementation against the same invariants as `MemoryStore`.

## Ecosystem Integration

- **Middleware/dispatch layer** — the `doc.go` package docs reference `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the integration point for CQRS pipelines. First step is in [TODO_LIST.md](TODO_LIST.md); the broader vision includes transport-agnostic middleware that works with HTTP, gRPC, and message queue consumers.
- **Key generation utilities** — helpers for producing stable idempotency keys (UUID v7, content-hash-based, request-derived). Currently every consumer must roll their own key strategy.

## Observability

- **Metrics hooks** — instrument hit/miss/expiry/contention rates so operators can tune TTL and sweep intervals based on real traffic patterns.
- **Lock strategy optimization** — baseline benchmarks exist (`bench_test.go`); next step is evaluating sharded mutexes, `sync.Map`, and lock-free approaches against the current single-`sync.RWMutex` design under high contention.

## Versioning Strategy

- **v0.x** — API may change between minor versions. `MemoryStore` is stable but the `Store` interface may evolve as real-world use reveals missing methods (e.g., `Delete`, `Stats`, `Reset`).
- **v1.0** — when the interface stabilizes (proven by multiple independent backend implementations in the wild) and the middleware layer ships.
