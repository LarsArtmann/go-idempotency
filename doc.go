// Package idempotency provides a deduplication store for command idempotency
// keys (and any other opaque at-most-once-processing keys).
//
// Delivery in a CQRS system is at-least-once: a client may submit a command,
// lose the acknowledgement, and retry. Without deduplication, the retried
// command executes twice and produces duplicate events and side effects.
//
// An idempotency store closes that gap. The client attaches a stable key to
// each logical command; the server records the key before processing and
// rejects retries whose key has already been recorded.
//
// # Quick Start
//
// The example below uses the deprecated [MemoryStore] to illustrate the API.
// In production, substitute your own [Store] implementation (see the
// "Implementing a Custom Backend" section below).
//
//	store := idempotency.NewMemoryStore(5 * time.Minute)
//	defer store.Close()
//
//	// Check-and-record in a single atomic step (preferred over Seen + Record).
//	err := store.CheckAndRecord(ctx, cmdKey, 10*time.Minute)
//	if errors.Is(err, idempotency.ErrDuplicate) {
//	    return err // already processed — drop the retry
//	}
//	if errors.Is(err, idempotency.ErrInvalidTTL) {
//	    return err // programmer error: ttl must be positive
//	}
//	if err != nil {
//	    return err // store failure — do not process
//	}
//
// # Design Philosophy: Interface-First, You Implement the Backend
//
// This library is an SDK, not a batteries-included framework. It deliberately
// ships only the [Store] interface, its error semantics, and a deprecated
// [MemoryStore] intended for development and testing only. There is no
// production backend by design.
//
// It intentionally does NOT provide production backends. There will be no
// Redis store, SQL store, or any other concrete backend added to this module.
// Each backend has its own driver, connection-pool semantics, deployment
// constraints, and operational tradeoffs — encoding those decisions here would
// bloat the dependency tree and force choices on consumers that they should
// make themselves.
//
// Instead, you implement the [Store] interface against whatever backend fits
// your system. The interface is small (three methods) and the comments on
// [Store.CheckAndRecord] name the atomic primitive each backend should use
// (Redis SET NX, SQL INSERT ... ON CONFLICT DO NOTHING).
//
// This module owns the storage contract only. A future middleware package
// (planned, not yet implemented) will provide CommandIdempotency,
// EventIdempotency, and QueryIdempotency helpers that wire a Store into CQRS
// dispatch pipelines. For custom integrations (transport hooks, manual checks),
// use the [Store] interface directly.
//
// # Implementing a Custom Backend
//
// The interface is three methods. Each maps to a single round-trip on a
// typical backend. The critical requirement is that [Store.CheckAndRecord] is
// atomic — use your backend's native check-and-set primitive, not a separate
// check followed by a record.
//
// Example — Redis adapter using github.com/redis/go-redis/v9 (illustrative:
// that client is intentionally NOT a dependency of this module):
//
//	type RedisStore struct {
//	    client *redis.Client
//	    prefix string // e.g. "idem:"
//	}
//
//	func (s *RedisStore) Seen(ctx context.Context, key string) (bool, error) {
//	    n, err := s.client.Exists(ctx, s.prefix+key).Result()
//	    if err != nil {
//	        return false, err
//	    }
//	    return n > 0, nil
//	}
//
//	func (s *RedisStore) Record(ctx context.Context, key string, ttl time.Duration) error {
//	    if ttl <= 0 {
//	        return idempotency.ErrInvalidTTL
//	    }
//	    // SET NX: if the key already exists, leave it (no-op, TTL not extended).
//	    return s.client.SetNX(ctx, s.prefix+key, "1", ttl).Err()
//	}
//
//	func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
//	    if ttl <= 0 {
//	        return idempotency.ErrInvalidTTL
//	    }
//	    ok, err := s.client.SetNX(ctx, s.prefix+key, "1", ttl).Result()
//	    if err != nil {
//	        return err
//	    }
//	    if !ok {
//	        return idempotency.ErrDuplicate
//	    }
//	    return nil
//	}
//
// The same pattern works for SQL (INSERT ... ON CONFLICT DO NOTHING), DynamoDB
// (PutItem with ConditionExpression), or any backend that supports an atomic
// conditional write. See the contract test suite (package contract) to verify
// your implementation against the same invariants as MemoryStore.
//
// # Recipe: Dedup + Response Replay (HTTP Idempotency)
//
// [Store.CheckAndRecord] answers "first attempt or duplicate", but an HTTP
// client retrying a payment does not want a 409 — it wants the original
// response replayed. The recipe: use the store for the atomic claim, and keep
// the finished response under a derived key in whatever KV storage your
// backend already gives you ("resp:" prefix below; keep the same TTL).
//
//	store := mybackend.NewStore(...) // any idempotency.Store (claims)
//	// responses: e.g. Redis SET resp:<key> <body> EX <ttl> / GET resp:<key>
//
//	func handle(ctx context.Context, key string, op func() (response, error)) (response, error) {
//	    err := store.CheckAndRecord(ctx, key, ttl)
//	    switch {
//	    case err == nil: // first attempt: we own the claim
//	        resp, opErr := op()
//	        if opErr == nil {
//	            saveResponse("resp:"+key, resp, ttl)
//	        }
//	        return resp, opErr
//	    case errors.Is(err, idempotency.ErrDuplicate):
//	        resp, ok := loadResponse("resp:" + key)
//	        if !ok {
//	            return response{}, errDuplicateNoResponse // 409: claimed, unfinished
//	        }
//	        return resp, nil // replay the original result
//	    default:
//	        return response{}, err // store failure: do NOT process
//	    }
//	}
//
// Correctness notes:
//
//   - Never execute op in the ErrDuplicate branch — the claim, not the
//     response, is what makes execution exactly-once.
//   - A crash between op succeeding and saveResponse leaves a claim without a
//     response: retriers get the 409-without-response path until the TTL
//     expires. Safe (no double execution), but the command is stuck. Storing
//     the response in the same transaction as the business effect closes the
//     gap; manual claim invalidation (a Store#Delete method is planned) is the
//     operational recovery.
//   - op returning an error intentionally keeps the claim: the command has
//     claimed its slot. If failed attempts should be retryable, model that in
//     your domain (e.g. save a failure marker as the response) instead of
//     deleting the claim behind the store's back.
package idempotency
