# Domain Language

Ubiquitous vocabulary for the go-idempotency library. These terms appear in code, doc comments, tests, and documentation. The "Where" column cites the defining location (file names are relative to the repository root; `store.go` means the root package).

## Bounded Contexts

The vocabulary splits across four bounded contexts. A term from one context rarely appears in another except at the seams (`middleware` consumes `Store`; `idemtest` drills both).

| Context                       | Vocabulary                                                                                              |
| ----------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Store** (root package)      | Key, TTL, Claim, Forever, Sweep, Lazy deletion, Seen/Record/CheckAndRecord, ErrDuplicate, ErrInvalidTTL |
| **Dispatch** (`middleware`)   | Command, Dispatch, Fail-closed, Idempotency-Key header, Response replay                                 |
| **Verification** (`contract`) | Invariant, Negative test, Sabotage, TimingScale, Cancellation suite                                     |
| **Kit** (`idemtest`)          | Drill, Executions counter, Blessed test store, Store spy, Retry storm, One winner                       |

## Core Concepts

| Term                        | Where                                | Definition                                                                                                                                                       |
| --------------------------- | ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Idempotency Key**         | `store.go` (`Store` doc), `doc.go`   | A stable string identifier attached to a logical operation so that retries can be detected and dropped. Typically a UUID or content hash provided by the client. |
| **At-least-once delivery**  | `doc.go` (package overview)          | A message delivery guarantee where a message may arrive more than once (e.g., due to network retries). This is the reality the library is designed for.          |
| **At-most-once processing** | `doc.go`, `middleware/middleware.go` | A processing guarantee where each logical operation executes at most once. This is what the library achieves on top of at-least-once delivery.                   |
| **Idempotency**             | `doc.go`                             | The property that processing the same operation multiple times has the same effect as processing it once. The library enforces this by rejecting duplicate keys. |

## Operations

| Term                   | Where                                         | Definition                                                                                                                                                                                                  |
| ---------------------- | --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Store**              | `store.go`                                    | The interface (`Store` in `store.go`) for tracking idempotency keys. Three methods: `Seen`, `Record`, `CheckAndRecord`. This is the core SDK contract that consumers implement against their own backend.   |
| **Seen**               | `store.go`, `contract/contract_invariants.go` | Reports whether a key is currently recorded and not expired. Performs lazy deletion of expired entries on read.                                                                                             |
| **Record**             | `store.go`, `contract/contract_invariants.go` | Marks a key as seen with a given TTL. No-op if the key is already recorded and not expired (`Forever` does not upgrade a live short claim; a short TTL does not downgrade a live permanent claim).          |
| **CheckAndRecord**     | `store.go`, `contract/contract_invariants.go` | The atomic primitive: checks whether a key is already seen, and if not, records it in a single locked operation. Returns `ErrDuplicate` if the key was already recorded. This is the preferred entry point. |
| **TTL (Time-to-Live)** | `store.go`, `doc.go`                          | The duration after which a recorded key expires and can be re-recorded. Specified per-key at record time. `Forever` is the one TTL that never expires.                                                      |
| **Sweep**              | `store.go` (`MemoryStore.sweep`)              | A background goroutine that periodically scans the map and removes expired entries. Configurable via `sweepInterval`; disabled when `sweepInterval == 0`.                                                   |
| **Lazy deletion**      | `store.go` (`MemoryStore.Seen`)               | Removal of expired entries on read (`Seen`), as opposed to background sweep. Ensures the map cannot grow unboundedly even when the sweep goroutine is disabled.                                             |
| **Pinned clock**       | `store.go` (`WithClock`), `fakeclock_test.go` | An injected clock (`func() time.Time`) that turns expiry into a deterministic state transition: tests advance it instead of sleeping. See `testing/synctest` bubbles for the option-free equivalent.        |

## Errors

| Term              | Where                                    | Definition                                                                                                                                                                     |
| ----------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **ErrDuplicate**  | `store.go`, `errors_test.go`             | The sentinel error returned by `CheckAndRecord` when the key was already recorded and not expired. Stable error code: `"idempotency.duplicate"`.                               |
| **ErrInvalidTTL** | `store.go`, `errors_test.go`             | The sentinel error returned by `Record` and `CheckAndRecord` when the TTL is not positive. `Forever` is positive and accepted. Stable error code: `"idempotency.invalid-ttl"`. |
| **Conflict**      | `go-error-family`, applied in `store.go` | The error family `ErrDuplicate` belongs to (via `go-error-family`). Maps to HTTP 409 and is non-retryable. Callers should return this error to the client, not retry.          |
| **Rejection**     | `go-error-family`, applied in `store.go` | The error family `ErrInvalidTTL` belongs to (via `go-error-family`). Maps to HTTP 400 and is non-retryable: the caller passed bad input.                                       |

## Design Concepts

| Term            | Where                             | Definition                                                                                                                                                                                                                                                                                                                                 |
| --------------- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **TOCTOU race** | `store.go` (`CheckAndRecord` doc) | Time-of-check-to-time-of-use race. If `Seen` and `Record` are called separately, two concurrent requests can both pass the `Seen` check before either records. `CheckAndRecord` prevents this by doing both under a single lock.                                                                                                           |
| **MemoryStore** | `store.go`                        | The deprecated in-memory `Store` implementation: `map[string]time.Time` guarded by `sync.RWMutex`. **Deprecated** — intended for development and testing only; removal targeted for v1.0. This library intentionally ships no production backends (Redis, SQL, etc.); consumers implement the `Store` interface against their own backend. |
| **Close**       | `store.go` (`MemoryStore.Close`)  | Stops the background sweep goroutine. Idempotent via `sync.Once`. Operations still function after Close; only the sweeper is stopped.                                                                                                                                                                                                      |

## Middleware & Claims

| Term                    | Where                                             | Definition                                                                                                                                                                                                                                                                                              |
| ----------------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Claim**               | `doc.go`, `contract/contract_invariants.go`       | The atomic reservation a successful `CheckAndRecord` places on a key: from that moment until the TTL expires (or forever, with `Forever`), other callers with the same key receive `ErrDuplicate`.                                                                                                      |
| **Permanent claim**     | `store.go` (`Forever`), `doc.go`, ADR-006         | A claim recorded with `Forever` (`math.MaxInt64`). It never expires, so it is only correct where re-processing a duplicate corrupts state: event/consumer dedupe, projection replay, permanent domain invariants. Request-facing claims stay TTL-windowed.                                              |
| **Poisoned claim**      | `doc.go`, ADR-004, ADR-006                        | A claim whose side effect never completed (e.g. a crash between winning `CheckAndRecord` and finishing the work). Retries get `ErrDuplicate` until the TTL expires; the TTL bounds the damage. A poisoned permanent claim has no TTL escape, which is why `Forever` is scoped to event/consumer dedupe. |
| **TTL window**          | `doc.go`, `idemtest` (`RequireAllowsAfterExpiry`) | The lifetime of a TTL-windowed claim. Sized to cover the retry horizon (client backoff, redelivery, outbox replay) — not the request timeout. A permanent claim has no window.                                                                                                                          |
| **Command idempotency** | `middleware/middleware.go` (`NewCommand`)         | `middleware.NewCommand`: wraps a command function for at-most-once dispatch — first call claims and executes, duplicates are rejected without executing, store failures fail closed.                                                                                                                    |
| **Response replay**     | `doc.go` ("Recipe: Dedup + Response Replay")      | Storing the original response and returning it to retriers instead of a bare `409`; the recipe lives in the root package documentation.                                                                                                                                                                 |

## Testing

| Term                      | Where                                                      | Definition                                                                                                                                                                                                                                      |
| ------------------------- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Test kit (`idemtest`)** | `idemtest/`, ADR-007                                       | The consumer-facing test package (ADR-007): drills and doubles for testing an integration above the store. Counterpart of `contract`, which verifies the store itself. Imported from `_test.go` files only — never in a production import path. |
| **Dispatch**              | `idemtest` (`Dispatch` type alias)                         | The shape `func(ctx, key string) error`: a keyed, at-most-once entry point. A `middleware.Command` satisfies it without conversion, and so does a closure over `CheckAndRecord`. The drills exercise the caller's wiring, never a rebuilt one.  |
| **Retry storm**           | `idemtest` (`RequireExactlyOnce`)                          | A burst of dispatches for one key — sequential (the client lost the ack and retried) or concurrent (retries racing the first delivery). The drills require exactly one execution across the storm.                                              |
| **One winner**            | `idemtest` (`RequireOneWinner`)                            | The outcome of a concurrent retry storm under a correct pipeline: exactly one dispatch executes, every other observes `ErrDuplicate`. The one-winner drill is the TOCTOU detector.                                                              |
| **Executions**            | `idemtest` (`Executions`, `Counting`, `RequireExecutions`) | The count of times a side effect actually ran — the observable half of at-most-once processing. The store reports which attempts were rejected; the counter reports what ran.                                                                   |
| **Fail-closed drill**     | `idemtest` (`RequireFailClosed`)                           | The exercise proving a pipeline never executes while the store fails: the dispatch must error (non-sentinel) and the side effect must not run, because unknown duplicate state is not permission to process.                                    |
| **Blessed test store**    | `idemtest` (`NewStore`), `internal/teststore`              | `idemtest.NewStore`: the contract-certified in-memory double consumers use in tests instead of a hand-rolled map or the deprecated `MemoryStore`. Continuously certified by `contract.RunTests` in the library's CI.                            |
| **Sabotage proof**        | `contract/*_negative_test.go`, `idemtest/negative_test.go` | A negative test proving an assertion fails against the violation class it names. Every new contract invariant and every new `idemtest.Require*` must ship one.                                                                                  |

## Deprecated Terms

| Term                                  | Status                                                                                                  | Use instead                                                                   |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| **MemoryStore (as production store)** | Deprecated since v0.2.0; removed in a future major version. Legitimate only in dev/test sentences.      | "your `Store` implementation" (production), `idemtest.NewStore` (test double) |
| **Sleep-based expiry test**           | Deprecated as a practice since v0.6.0 (`WithClock`) — any real-time sleep guarding an expiry assertion. | Pinned clock (`WithClock`) or a `testing/synctest` bubble                     |
