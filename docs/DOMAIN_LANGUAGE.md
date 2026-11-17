# Domain Language

Ubiquitous vocabulary for the go-idempotency library. These terms appear in code, doc comments, tests, and documentation.

## Core Concepts

| Term                        | Definition                                                                                                                                                       |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Idempotency Key**         | A stable string identifier attached to a logical operation so that retries can be detected and dropped. Typically a UUID or content hash provided by the client. |
| **At-least-once delivery**  | A message delivery guarantee where a message may arrive more than once (e.g., due to network retries). This is the reality the library is designed for.          |
| **At-most-once processing** | A processing guarantee where each logical operation executes at most once. This is what the library achieves on top of at-least-once delivery.                   |
| **Idempotency**             | The property that processing the same operation multiple times has the same effect as processing it once. The library enforces this by rejecting duplicate keys. |

## Operations

| Term                   | Definition                                                                                                                                                                                                  |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Store**              | The interface (`Store` in `store.go`) for tracking idempotency keys. Three methods: `Seen`, `Record`, `CheckAndRecord`. This is the core SDK contract that consumers implement against their own backend.   |
| **Seen**               | Reports whether a key is currently recorded and not expired. Performs lazy deletion of expired entries on read.                                                                                             |
| **Record**             | Marks a key as seen with a given TTL. No-op if the key is already recorded and not expired (`Forever` does not upgrade a live short claim; a short TTL does not downgrade a live permanent claim).          |
| **CheckAndRecord**     | The atomic primitive: checks whether a key is already seen, and if not, records it in a single locked operation. Returns `ErrDuplicate` if the key was already recorded. This is the preferred entry point. |
| **TTL (Time-to-Live)** | The duration after which a recorded key expires and can be re-recorded. Specified per-key at record time. `Forever` is the one TTL that never expires.                                                      |
| **Sweep**              | A background goroutine that periodically scans the map and removes expired entries. Configurable via `sweepInterval`; disabled when `sweepInterval == 0`.                                                   |
| **Lazy deletion**      | Removal of expired entries on read (`Seen`), as opposed to background sweep. Ensures the map cannot grow unboundedly even when the sweep goroutine is disabled.                                             |

## Errors

| Term              | Definition                                                                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **ErrDuplicate**  | The sentinel error returned by `CheckAndRecord` when the key was already recorded and not expired. Stable error code: `"idempotency.duplicate"`.                               |
| **ErrInvalidTTL** | The sentinel error returned by `Record` and `CheckAndRecord` when the TTL is not positive. `Forever` is positive and accepted. Stable error code: `"idempotency.invalid-ttl"`. |
| **Conflict**      | The error family `ErrDuplicate` belongs to (via `go-error-family`). Maps to HTTP 409 and is non-retryable. Callers should return this error to the client, not retry.          |
| **Rejection**     | The error family `ErrInvalidTTL` belongs to (via `go-error-family`). Maps to HTTP 400 and is non-retryable: the caller passed bad input.                                       |

## Design Concepts

| Term            | Definition                                                                                                                                                                                                                                                                                                                                 |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **TOCTOU race** | Time-of-check-to-time-of-use race. If `Seen` and `Record` are called separately, two concurrent requests can both pass the `Seen` check before either records. `CheckAndRecord` prevents this by doing both under a single lock.                                                                                                           |
| **MemoryStore** | The deprecated in-memory `Store` implementation: `map[string]time.Time` guarded by `sync.RWMutex`. **Deprecated** — intended for development and testing only; removal targeted for v1.0. This library intentionally ships no production backends (Redis, SQL, etc.); consumers implement the `Store` interface against their own backend. |
| **Close**       | Stops the background sweep goroutine. Idempotent via `sync.Once`. Operations still function after Close; only the sweeper is stopped.                                                                                                                                                                                                      |

## Middleware & Claims

| Term                    | Definition                                                                                                                                                                                                                                                                                                                       |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Claim**               | The atomic reservation a successful `CheckAndRecord` places on a key: from that moment until the TTL expires (or forever, with `Forever`), other callers with the same key receive `ErrDuplicate`.                                                                                                                               |
| **Permanent claim**     | A claim recorded with `Forever` (`math.MaxInt64`). It never expires, so it is only correct where re-processing a duplicate corrupts state: event/consumer dedupe, projection replay, permanent domain invariants. Request-facing claims stay TTL-windowed.                                                                       |
| **Poisoned claim**      | A claim whose side effect never completed (e.g. a crash between winning `CheckAndRecord` and finishing the work). Retries get `ErrDuplicate` until the TTL expires; the TTL bounds the damage. A poisoned permanent claim has no TTL escape, which is why `Forever` is scoped to event/consumer dedupe. See ADR-004 and ADR-006. |
| **TTL window**          | The lifetime of a TTL-windowed claim. Sized to cover the retry horizon (client backoff, redelivery, outbox replay) — not the request timeout. A permanent claim has no window.                                                                                                                                                   |
| **Command idempotency** | `middleware.NewCommand`: wraps a command function for at-most-once dispatch — first call claims and executes, duplicates are rejected without executing, store failures fail closed.                                                                                                                                             |
| **Response replay**     | Storing the original response and returning it to retriers instead of a bare `409`; the recipe lives in the root package documentation.                                                                                                                                                                                          |

## Testing

| Term                      | Definition                                                                                                                                                                                                                                      |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Test kit (`idemtest`)** | The consumer-facing test package (ADR-007): drills and doubles for testing an integration above the store. Counterpart of `contract`, which verifies the store itself. Imported from `_test.go` files only — never in a production import path. |
| **Dispatch**              | The shape `func(ctx, key string) error`: a keyed, at-most-once entry point. A `middleware.Command` satisfies it without conversion, and so does a closure over `CheckAndRecord`. The drills exercise the caller's wiring, never a rebuilt one.  |
| **Retry storm**           | A burst of dispatches for one key — sequential (the client lost the ack and retried) or concurrent (retries racing the first delivery). The drills require exactly one execution across the storm.                                              |
| **One winner**            | The outcome of a concurrent retry storm under a correct pipeline: exactly one dispatch executes, every other observes `ErrDuplicate`. The one-winner drill is the TOCTOU detector.                                                              |
| **Executions**            | The count of times a side effect actually ran — the observable half of at-most-once processing. The store reports which attempts were rejected; the counter reports what ran.                                                                   |
| **Fail-closed drill**     | The exercise proving a pipeline never executes while the store fails: the dispatch must error (non-sentinel) and the side effect must not run, because unknown duplicate state is not permission to process.                                    |
| **Blessed test store**    | `idemtest.NewStore`: the contract-certified in-memory double consumers use in tests instead of a hand-rolled map or the deprecated `MemoryStore`. Continuously certified by `contract.RunTests` in the library's CI.                            |
