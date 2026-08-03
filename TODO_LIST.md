# TODO List

Short-term actionable work. Open items only — completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Bugs

- [ ] **`Record` ignores TTL expiry** — `MemoryStore.Record` (`store.go:103-112`) checks map presence only, not expiry. An expired-but-not-yet-swept key is treated as "already recorded" and `Record` becomes a no-op, failing to record with the new TTL. Fix: check `time.Now().Before(exp)` before deciding it's a duplicate, same pattern as `CheckAndRecord` (`store.go:118-129`). Add a test for `Record` after expiry without intervening `Seen`.

## Infrastructure

- [ ] **Add `.golangci.yml`** — `CONTRIBUTING.md` references `golangci-lint run ./...` but no config exists. The code uses `//nolint:exhaustruct` (`store.go:70`), implying the `exhaustruct` linter is expected. Without a config, `golangci-lint` uses defaults that don't include `exhaustruct`.
- [ ] **Add benchmarks** — this is a concurrency library where lock contention and allocation matter. No `Benchmark*` functions exist. Add `go test -bench` coverage for `CheckAndRecord` under contention (serial vs parallel goroutine counts).

## Features

- [ ] **Context support in `MemoryStore`** — all three methods ignore `ctx` (params named `_`). At minimum, `Close()` should be able to interrupt the sweep goroutine's `ticker.C` wait, and future store backends (Redis, SQL) will need context for cancellation/timeout.
- [ ] **Implement middleware package** — `doc.go:26-31` references `CommandIdempotency`, `EventIdempotency`, and `QueryIdempotency` as the way to wire the store into CQRS dispatch pipelines, but no code exists. This is the primary advertised integration point and currently the library is storage-only.
