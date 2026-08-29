# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Release


## Features

- [ ] **Implement middleware package** — `CommandIdempotency` wiring a `Store` into CQRS dispatch pipelines; stdlib-only subpackage per [ADR-002](docs/adr/002-middleware-module-boundary.md). `EventIdempotency`/`QueryIdempotency` wait for a consumer that needs them (YAGNI). **Impact: High · Effort: L**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: owner raised the domain concern that claim invalidation must not become request-path API; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**

## Testing

## Infrastructure

