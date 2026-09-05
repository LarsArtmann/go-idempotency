# Architecture Decision Records

Decisions that shape this library's scope and API. Newer ADRs may carry a short owner-veto window; otherwise statuses are settled.

| ADR                                          | Decision                                                                                                                                                                                  | Status                                                  |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| [ADR-001](001-no-backends.md)                | No production backends: interface-first SDK; consumers implement `Store` against Redis/SQL/etc. and validate with the contract suite; `MemoryStore` is deprecated dev/test tooling.       | Accepted                                                |
| [ADR-002](002-middleware-module-boundary.md) | Middleware ships as a stdlib-only subpackage of this module (`middleware/`), HTTP-first; `EventIdempotency`/`QueryIdempotency` YAGNI-gated; a separate module waits for a demand trigger. | Accepted (provisional under owner blanket approval)     |
| [ADR-003](003-bounded-store-position.md)     | No `BoundedStore`: an LRU claim store silently sacrifices exactly-once; restart durability stays a documented position, not a half-guarantee.                                             | Accepted                                                |
| [ADR-004](004-store-interface-evolution.md)  | `Store` interface unchanged; `Delete` deferred (claims are TTL-windowed, not forever; recovery is an operational concern); `ErrStoreClosed`/`Stats`/`Reset` not added.                    | Accepted (`Delete` deferred pending owner confirmation) |
