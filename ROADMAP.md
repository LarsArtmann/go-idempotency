# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks. When an item becomes bounded and estimable, it graduates to [TODO_LIST.md](TODO_LIST.md).

## Distributed Backends

The `Store` interface (`store.go:30-48`) is designed for multiple backends. `MemoryStore` is the first; the interface comments name the intended atomic primitives for future implementations.

- **Redis store** — distributed idempotency using `SET NX EX` for atomic check-and-record across multiple service instances. Open questions: key namespacing/prefix strategy, cluster-mode behavior, connection pooling, TTL precision vs Redis eviction policy.
- **SQL store** — persistent idempotency using `INSERT ... ON CONFLICT DO NOTHING`. Open questions: which database drivers to support, schema/migration strategy, cleanup of expired rows (scheduled job vs lazy delete vs both).

## Ecosystem Integration

- **Middleware/dispatch layer** — the `doc.go` package docs reference `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the integration point for CQRS pipelines. First step is in [TODO_LIST.md](TODO_LIST.md); the broader vision includes transport-agnostic middleware that works with HTTP, gRPC, and message queue consumers.
- **Key generation utilities** — helpers for producing stable idempotency keys (UUID v7, content-hash-based, request-derived). Currently every consumer must roll their own key strategy.

## Observability

- **Metrics hooks** — instrument hit/miss/expiry/contention rates so operators can tune TTL and sweep intervals based on real traffic patterns.
- **Benchmark-driven optimization** — after baseline benchmarks exist (see [TODO_LIST.md](TODO_LIST.md)), evaluate lock strategies: sharded mutexes, `sync.Map`, lock-free approaches. The current single-`sync.RWMutex` design is correct but may bottleneck under high contention.

## Versioning Strategy

- **v0.x** — API may change between minor versions. `MemoryStore` is stable but the `Store` interface may evolve as backend implementations reveal missing methods (e.g., `Delete`, `Stats`, `Reset`).
- **v1.0** — when the interface stabilizes and at least one persistent backend (Redis or SQL) ships.
