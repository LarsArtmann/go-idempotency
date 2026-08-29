# ADR-001: No Production Backends

## Status

Accepted

## Context

go-idempotency provides a `Store` interface for idempotency key tracking. The initial documentation referenced Redis and SQL backends as "planned." As the library evolved, the question arose: should this module ship concrete backend implementations (Redis, SQL, DynamoDB, etc.), or should it remain interface-only with `MemoryStore` as a reference?

## Decision

**This module will not ship production backends.** It provides:

1. The `Store` interface (three methods: `Seen`, `Record`, `CheckAndRecord`)
2. `MemoryStore` — an in-memory implementation for development and testing (deprecated since v0.2.0; removal targeted for v1.0)
3. The `contract` package — a test suite for verifying any `Store` implementation

Consumers implement the `Store` interface against their own backend.

## Rationale

Each backend carries:

- **Driver dependencies** that bloat the dependency tree (Redis client, SQL drivers, protobuf for gRPC-based stores)
- **Connection-pool semantics** that vary by deployment (single-node vs cluster, pooling strategies, timeout policies)
- **Operational tradeoffs** that are the consumer's decision (eviction policies, persistence guarantees, failover behavior)
- **Deployment constraints** that differ per environment (managed Redis, self-hosted Postgres, serverless KV stores)

Bundling any of these would force choices on consumers that they should make themselves, and would make the library harder to maintain (each backend needs its own test infrastructure, version tracking, and release cadence).

The interface is intentionally small (three methods) so that implementing a backend is straightforward. Each method maps to a single atomic primitive on typical backends (Redis `SET NX`, SQL `INSERT ... ON CONFLICT DO NOTHING`). The `contract` package provides the same test invariants that `MemoryStore` passes, so consumers can verify their implementation with one function call.

## Alternatives Considered

### A. Ship Redis and SQL backends in this module

**Rejected.** Adds driver dependencies, connection management, and configuration surface area to a library whose core value is a three-method interface. Every consumer pays the dependency cost even if they use a different backend.

### B. Ship backends as subpackages (optional imports)

**Rejected.** Go modules don't support optional subpackage imports — all dependencies in `go.mod` are downloaded regardless of which packages are imported. Subpackages would still bloat the dependency tree.

### C. Ship backends in separate modules under the same org

**Deferred.** This is viable but not needed today. If demand emerges, separate modules (`go-idempotency-redis`, `go-idempotency-sql`) can be created without changing this module. The `Store` interface and `contract` package already provide everything needed.

## Supplement (2026-08-29): adapter example scope

Documentation carries adapter examples for Redis and SQL (PostgreSQL) only —
they cover the dominant deployments. Example requests for further backends
(DynamoDB, MongoDB, etc.) are declined per this ADR: each backend's atomic
conditional write plus the contract suite is the template consumers follow,
and every added example would hint that the backend is "supported" here.

## Consequences

- Consumers must implement the `Store` interface for their backend (typically 20-30 lines of code)
- The `contract` package must be maintained as the source of truth for `Store` invariants
- `MemoryStore` is deprecated (since v0.2.0) and will be removed in v1.0; it remains functional for development and testing. Supersedes the original stance of keeping MemoryStore around permanently.
- No driver dependencies will ever appear in this module's `go.mod`
