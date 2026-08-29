# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks. When an item becomes bounded and estimable, it graduates to [TODO_LIST.md](TODO_LIST.md).

## Backend Implementations (Out of Scope by Design)

This library provides the `Store` interface and `MemoryStore` (a **deprecated** in-memory store for development and testing only). It **will not** ship production backends — Redis, SQL, or otherwise.

**Why:** Each backend carries its own driver dependency, connection-pool semantics, deployment constraints, and operational tradeoffs. Bundling them here would bloat the dependency tree and impose decisions that consumers should own. The interface is intentionally small (three methods) so that implementing your own backend is straightforward.

**Implementation guidance for your own backend:**

- Read the `Store` interface and the atomicity contract on `CheckAndRecord` in `store.go`.
- Use your backend's native atomic primitive: Redis `SET NX EX`, SQL `INSERT ... ON CONFLICT DO NOTHING`, etc.
- Use the `contract` package (`contract.RunTests`) to verify your implementation against the same invariants as `MemoryStore`.

## Ecosystem Integration

- **Middleware/dispatch layer** — **Shipped 2026-08-29** for the command path: `middleware.NewCommand` (transport-agnostic at-most-once dispatch) and the `net/http` adapter honoring `Idempotency-Key` (stdlib-only per [ADR-002](docs/adr/002-middleware-module-boundary.md)). Remaining, demand-gated: `EventIdempotency`/`QueryIdempotency` (only when a consumer needs them) and a gRPC adapter (only as its own module, per ADR-002's split trigger — write the ADR supplement first).
- **Key generation utilities** — helpers for producing stable idempotency keys (UUID v7, content-hash-based, request-derived). Currently every consumer must roll their own key strategy.

## Observability

- **Metrics hooks** — instrument hit/miss/expiry/contention rates so operators can tune TTL and sweep intervals based on real traffic patterns.
- **Lock strategy optimization** — baseline benchmarks exist (`bench_test.go`); next step is evaluating sharded mutexes, `sync.Map`, and lock-free approaches against the current single-`sync.RWMutex` design under high contention.

## In-Process Store Evolution (from PapDashboard consumer evaluation)

Raw ideas surfaced by the 2026-08-18 consumer evaluation in `docs/feedback/new/2026-08-18_13-07_papdashboard-evaluation-dedup-only-stopped-adoption.md` (evaluated v0.1.2, did not adopt — see the report for the full reasoning):

- **Response-replay composition recipe** — the store answers "seen or duplicate" only; HTTP-style consumers (Stripe semantics) need a retried request to return the original response, not a 409. A documented dedup+replay pattern (e.g., a thin `ResponseCache` wrapped around `Store`, ~20 lines in `doc.go`) would close this gap without polluting the key-only `Store` contract. **Done 2026-08-29:** shipped as the "Recipe: Dedup + Response Replay (HTTP Idempotency)" section of `doc.go` (claim via `CheckAndRecord`, response under a derived key, replay on duplicate, crash-gap correctness notes).
- **Bounded in-process store — or a documented position.** Deprecating `MemoryStore` removed the only in-process option, so single-process apps roll their own (LRU + TTL). Either ship a `BoundedStore(maxEntries, ttl)` (swap the map for an LRU, validated by `contract.RunTests`) or document restart-durability as table stakes for idempotency and point single-process apps to what they should do instead. **Resolved 2026-08-29 (ADR-003):** documented position — restart durability is table stakes, and an LRU-capped claim store silently sacrifices exactly-once; no `BoundedStore` will ship.

## Versioning Strategy

- **v0.2.0 (released 2026-08-29)** — contract test suite, implementation examples (Redis adapter), fuzz tests, memory benchmarks, godoc examples, ADR-001. All docs updated to interface-first SDK framing. `MemoryStore` formally deprecated (dev/test only).
- **v0.3.0 (staging in CHANGELOG `[Unreleased]`)** — `middleware` package (command dispatch + HTTP adapter), `RunTestsStrict`/`TimingScale`, negative-test detection proofs for every contract invariant, runnable `example/`, migration guide, response-replay recipe, governance files. Cut per [RELEASING.md](RELEASING.md) once the owner's ADR veto window closes.
- **v0.x** — API may change between minor versions. `MemoryStore` is deprecated; the `Store` interface evolves only per [ADR-004](docs/adr/004-store-interface-evolution.md) (`Delete` deferred pending demonstrated need; `Stats`/`Reset`/return-shape changes rejected or deferred). The `middleware` subpackage ships stdlib-only per [ADR-002](docs/adr/002-middleware-module-boundary.md).
- **v1.0** — `MemoryStore` removed and the interface stabilizes (proven by multiple independent backend implementations in the wild). The middleware layer already ships as a subpackage; further transports land as their own modules per ADR-002.
