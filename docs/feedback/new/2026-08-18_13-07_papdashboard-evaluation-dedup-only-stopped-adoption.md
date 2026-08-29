# Consumer Feedback — PapDashboard Evaluation: "Dedup Only" Stopped Adoption

**Date:** 2026-08-18 13:07
**Submitter:** Lars (via agent session, on behalf of the PapDashboard codebase)
**Consumer:** [PapDashboard](https://github.com/larsartmann/papdashboard) — event-sourced notification hub (Go, CQRS + Event Sourcing, SQLite, single-process)
**Adoption status:** ⚠️ **Evaluated, not adopted directly.** Present in our `go.mod` only as an _indirect_ dependency (`papdashboard → cqrs-htmx/v4 → go-cqrs-lite/idempotency/v4 → go-idempotency v0.1.2`). We shipped a homegrown ~60-line store instead (`internal/api/idempotency.go`). Details below — this document explains exactly why, and what would change the calculus.

---

## 1. Where we encountered the library

PapDashboard exposes `POST /api/ingest` for external applications to push notifications/questions/alerts. Real integrators retry on transient failures, so we needed idempotency keys on the ingest path (tracked in our ROADMAP as a pre-trust requirement). The natural candidate was already in our module graph transitively — we just had to import it directly. We looked, and didn't.

## 2. What we found genuinely good (why this was a hard "no", not an easy one)

- **`CheckAndRecord` as the atomic primitive.** The TOCTOU guidance is excellent — the README's "✅ atomic / ❌ racy" side-by-side is the best explanation of the race I've seen in a Go dedup library. Our homegrown store sidesteps the race only because the underlying LRU is internally synchronized.
- **`ErrInvalidTTL` rejecting non-positive TTLs at the boundary.** We copied this stance. A silent correctness hole from `ttl <= 0` is exactly the kind of bug that ships to production and never fires until it does.
- **Error taxonomy via go-error-family.** `ErrDuplicate` as a Conflict maps cleanly to HTTP 409 and composes with our existing error-mapping middleware. No impedance mismatch at all.
- **The `contract/` suite.** "Implement `Store`, prove it with `RunTests`" is the right interface-first shape, and rare.

## 3. Why we did not adopt it — the three blockers

### B1. The store records opaque keys; our use case requires replaying the original _response_

This is the decisive one. HTTP idempotency-key semantics (Stripe-style) demand that a retried request **returns the original response**, not a 409:

```
1. POST /api/ingest (Idempotency-Key: K) → 201 {id: "ntf_123", version: 1}
2. ack lost; client retries with K
3. GET-equivalent: server must answer 201 {id: "ntf_123", version: 1}  ← replay, not reject
```

`Store` can only answer "seen" or "duplicate" — there is nowhere to put the payload. Our ingest handler caches `(status, id, version)` per key and replays them (`internal/api/handler_impl.go:231-249`). With go-idempotency as-is, a retried ingest would surface `ErrDuplicate`/409, and the client would be left with **no way to learn which entity its key created** — it would have to list-and-guess. For a _creation_ endpoint that is a broken UX: the retry is not a conflict, it is the same request.

We could not fix this by composition either: `CheckAndRecord` gives no hook to attach "the key won → record this response alongside it", and `Seen`+`Record` split re-introduces exactly the TOCTOU race the library (rightly) forbids.

### B2. Nothing shippable for a single-process production app

Our production topology is one process + SQLite. That rules out the intended path ("implement `Store` against Redis/SQL") — there is no Redis, and a SQL-backed store buys nothing over our event store. Which leaves `MemoryStore`, which is:

- **Deprecated** for production use — correctly, since it loses keys on restart. After a crash, every in-flight client retry re-executes. We'd inherit that hole silently.
- **Unbounded** (TTL-only, no size cap). An attacker or a chatty integrator can grow the map until OOM. Our replacement is LRU-capped at 1,000 entries _plus_ TTL (`hashicorp/golang-lru/v2/expirable`).

So the honest comparison was "deprecated, unbounded" vs "60 lines with the two knobs we need." The 60 lines won. Not because the library is worse — because the segment "single-process app wants bounded, in-process dedup" has no product here anymore now that `MemoryStore` is deprecated.

### B3. The integration point we'd actually use doesn't exist yet

`doc.go` promises a middleware package (`CommandIdempotency`, `EventIdempotency`, `QueryIdempotency`). That is the layer where PapDashboard would naturally consume this library — at the command-dispatch boundary, not in the HTTP handler. AGENTS.md flags it as planned-not-implemented. We evaluated what exists.

## 4. What would change the calculus

Prioritized, with the reasoning:

1. **Decide and document where response-replay lives.** Our strong suggestion: _not_ in `Store` — keep it key-only and let a thin `ResponseCache[K, V]` wrap it (the atomicity of `CheckAndRecord` is the hard part; caching the payload next to the key is trivial). Even a doc-level composition recipe ("dedup + replay" pattern, 20 lines in `doc.go`) would have let us adopt the library and build only the replay layer. Right now the docs leave a reader in our position assuming the library "doesn't do idempotency keys properly" — it does dedup properly and is silent on replay. → Routed to ROADMAP.md (response-replay composition recipe), 2026-08-29.
2. **An LRU-bounded in-process store as a supported (non-deprecated) option.** Deprecating `MemoryStore` for restart-durability reasons is right, but it also removed the only bounded option. A `BoundedStore(maxEntries, ttl)` — swap the map for an LRU, keep the same contract tests — serves single-process production apps honestly, with the restart caveat documented rather than deprecated. Alternative we'd accept equally: a documented position that restart-durability is table stakes for idempotency (defensible!) and a pointer to what such apps should do instead. → Routed to ROADMAP.md (In-Process Store Evolution), 2026-08-29.
3. **Ship the middleware package.** Once command-boundary middleware exists, PapDashboard would consume it directly at the dispatcher layer and the homegrown HTTP store likely shrinks or disappears. → Already tracked in TODO_LIST.md (blocked on the module-boundary decision), confirmed 2026-08-29.

## 5. Silver lining / current relationship

The library still influences us transitively: `go-cqrs-lite/idempotency/v4` (which wraps it) is in our graph via `cqrs-htmx/v4`, and this evaluation directly shaped our homegrown store's design (TTL validation stance, atomicity discipline). We're a design consumer even where we're not an import consumer.

## 6. Facts for triage

| Question                   | Answer                                                                        |
| -------------------------- | ----------------------------------------------------------------------------- |
| Evaluated version          | v0.1.2 (in module graph) against `main` docs as of 2026-08-18                 |
| Direct imports in our code | 0                                                                             |
| Replacement                | `internal/api/idempotency.go` (~60 lines, LRU 1000 + TTL, response replay)    |
| Blocking gap               | Response replay (B1), then no shippable in-process backend (B2)               |
| Would re-evaluate on       | Replay-pattern docs or `ResponseCache`; bounded store; middleware package     |
| Consumed via               | `cqrs-htmx/v4 → go-cqrs-lite/idempotency/v4 → go-idempotency` (indirect only) |
