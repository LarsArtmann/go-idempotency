# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Release


## Features

- [ ] **Extend middleware package** — `CommandIdempotency` (command wrapper) and the `net/http` `Idempotency-Key` adapter shipped 2026-08-29 per [ADR-002](docs/adr/002-middleware-module-boundary.md). Remaining: `EventIdempotency`/`QueryIdempotency` — implement only when a consumer needs them (YAGNI). **Impact: Low · Effort: M (when triggered)**
- [ ] **Revisit `Delete(ctx, key)` on `Store`** — manual claim invalidation for operational recovery of poisoned claims. **Deferred per [ADR-004](docs/adr/004-store-interface-evolution.md)**: owner raised the domain concern that claim invalidation must not become request-path API; revisit on a demonstrated poisoned-claim need that TTL tuning cannot absorb. **Impact: Medium · Effort: S (when triggered)**

## Testing

## Infrastructure

