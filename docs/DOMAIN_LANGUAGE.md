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
| **Record**             | Marks a key as seen with a given TTL. No-op if the key is already recorded and not expired.                                                                                                                 |
| **CheckAndRecord**     | The atomic primitive: checks whether a key is already seen, and if not, records it in a single locked operation. Returns `ErrDuplicate` if the key was already recorded. This is the preferred entry point. |
| **TTL (Time-to-Live)** | The duration after which a recorded key expires and can be re-recorded. Specified per-key at record time.                                                                                                   |
| **Sweep**              | A background goroutine that periodically scans the map and removes expired entries. Configurable via `sweepInterval`; disabled when `sweepInterval == 0`.                                                   |
| **Lazy deletion**      | Removal of expired entries on read (`Seen`), as opposed to background sweep. Ensures the map cannot grow unboundedly even when the sweep goroutine is disabled.                                             |

## Errors

| Term             | Definition                                                                                                                                                            |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ErrDuplicate** | The sentinel error returned by `CheckAndRecord` when the key was already recorded and not expired. Stable error code: `"idempotency.duplicate"`.                      |
| **Conflict**     | The error family `ErrDuplicate` belongs to (via `go-error-family`). Maps to HTTP 409 and is non-retryable. Callers should return this error to the client, not retry. |

## Design Concepts

| Term            | Definition                                                                                                                                                                                                                                                                                                                                |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TOCTOU race** | Time-of-check-to-time-of-use race. If `Seen` and `Record` are called separately, two concurrent requests can both pass the `Seen` check before either records. `CheckAndRecord` prevents this by doing both under a single lock.                                                                                                          |
| **MemoryStore** | The in-memory `Store` reference implementation: `map[string]time.Time` guarded by `sync.RWMutex`. **Deprecated** — intended for development and testing only; removal targeted for v1.0. This library intentionally ships no production backends (Redis, SQL, etc.); consumers implement the `Store` interface against their own backend. |
| **Close**       | Stops the background sweep goroutine. Idempotent via `sync.Once`. Operations still function after Close; only the sweeper is stopped.                                                                                                                                                                                                     |
