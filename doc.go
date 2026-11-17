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
// # Contents
//
//   - Quick Start — the three calls and the two error sentinels
//   - Permanent Claims (Forever) — never-expiring claims and backend encodings
//   - Design Philosophy: Interface-First, You Implement the Backend
//   - Implementing a Custom Backend — the Redis adapter example
//   - Recipe: Dedup + Response Replay (HTTP Idempotency)
//   - Client-side vs receiver-side posture
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
// # Permanent Claims (Forever)
//
// Pass [Forever] to [Store.Record] or [Store.CheckAndRecord] when the
// duplicate window is the lifetime of the system: event and consumer dedupe,
// projection replay, or a permanent domain invariant (for example, "an
// invoice may only be submitted once"). TTL-windowed claims remain the default
// for request-facing idempotency keys; a permanent claim on a crashed request
// becomes a permanent lockout (ADR-004, ADR-006).
//
//	// A redelivered event must never fold twice.
//	err := store.CheckAndRecord(ctx, "event:"+eventID, idempotency.Forever)
//	if errors.Is(err, idempotency.ErrDuplicate) {
//	    return nil // already applied
//	}
//	if err != nil {
//	    return err // store failure: do not process
//	}
//
// Do not compute now.Add(Forever). time.Time can represent that instant
// (about year 2318), but UnixNano overflows in 2262. Map the sentinel at the
// backend boundary instead:
//
//   - Redis: SET key 1 (no EX); Seen is EXISTS; Record is SET NX;
//     CheckAndRecord is SET NX.
//   - PostgreSQL/MySQL: expires_at IS NULL means forever; a claim is live when
//     expires_at IS NULL OR expires_at > now().
//   - SQLite: store NULL for forever; never render now.Add(Forever) into the
//     TEXT expiry column.
//   - DynamoDB: omit the TTL attribute; Seen must still check the expiry
//     because TTL deletion is best-effort.
//
// ttl == 0 stays ErrInvalidTTL. Forever is the explicit opt-in, so a forgotten
// TTL cannot silently become a permanent claim. The in-process stores use the
// zero time.Time as the permanent representation and their sweeper never
// removes it.
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
// One first-party implementation exists outside this module: go-cqrs-lite's
// metaengine.DedupStore engine capability (github.com/larsartmann/go-cqrs-lite),
// implemented by every go-cqrs-lite storage engine and pinned by cross-engine
// conformance tests. Consumers of that stack do not hand-write a backend.
//
// This module owns the storage contract; the middleware subpackage
// (github.com/larsartmann/go-idempotency/middleware) wires a Store into CQRS
// dispatch pipelines: middleware.NewCommand wraps a command function for
// at-most-once dispatch and the net/http adapter honors the Idempotency-Key
// header (stdlib-only per ADR-002). EventIdempotency and QueryIdempotency
// helpers wait for a consumer that needs them. For custom integrations
// (transport hooks, manual checks), use the [Store] interface directly.
//
// # Implementing a Custom Backend
//
// The interface is three methods. Each maps to a single round-trip on a
// typical backend. The critical requirement is that [Store.CheckAndRecord] is
// atomic — use your backend's native check-and-set primitive, not a separate
// check followed by a record.
//
// Both examples below handle [Forever] explicitly: Redis omits the expiry and
// SQL stores NULL, so no adapter ever computes now.Add(Forever).
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
//	// redisExpiry maps Forever to 0 seconds, which go-redis sends as a SET
//	// with no EX argument: the key never expires.
//	func redisExpiry(ttl time.Duration) time.Duration {
//	    if ttl == idempotency.Forever {
//	        return 0
//	    }
//	    return ttl
//	}
//
//	func (s *RedisStore) Record(ctx context.Context, key string, ttl time.Duration) error {
//	    if ttl <= 0 {
//	        return idempotency.ErrInvalidTTL
//	    }
//	    // SET NX: if the key already exists, leave it (no-op: the TTL is not
//	    // extended and a permanent claim is never downgraded).
//	    return s.client.SetNX(ctx, s.prefix+key, "1", redisExpiry(ttl)).Err()
//	}
//
//	func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
//	    if ttl <= 0 {
//	        return idempotency.ErrInvalidTTL
//	    }
//	    ok, err := s.client.SetNX(ctx, s.prefix+key, "1", redisExpiry(ttl)).Result()
//	    if err != nil {
//	        return err
//	    }
//	    if !ok {
//	        return idempotency.ErrDuplicate
//	    }
//	    return nil
//	}
//
// The same pattern works for SQL (a conditional INSERT ... ON CONFLICT DO
// UPDATE), DynamoDB (PutItem with ConditionExpression), or any backend that
// supports an atomic conditional write. See the contract test suite (package
// contract) to verify your implementation against the same invariants as
// MemoryStore.
//
// Example — SQL adapter (PostgreSQL; database/sql is stdlib, but drivers are
// intentionally NOT dependencies of this module). Schema:
//
//	CREATE TABLE idempotency_keys (
//	    key        TEXT PRIMARY KEY,
//	    expires_at TIMESTAMPTZ  -- NULL means the claim is permanent (Forever)
//	);
//
// The expiry is computed by the application clock in this example; filter on
// expires_at (or add a scheduled purge) to reclaim expired rows, treating NULL
// as live forever:
//
//	type SQLStore struct {
//	    db *sql.DB
//	}
//
//	// expiryArg maps the Forever sentinel to SQL NULL (never expires) and
//	// rejects a non-positive TTL, so no adapter ever computes
//	// now.Add(Forever).
//	func expiryArg(ttl time.Duration) (*time.Time, error) {
//	    if ttl <= 0 {
//	        return nil, idempotency.ErrInvalidTTL
//	    }
//	    if ttl == idempotency.Forever {
//	        return nil, nil // NULL: permanent claim
//	    }
//	    exp := time.Now().Add(ttl)
//	    return &exp, nil
//	}
//
//	func (s *SQLStore) Seen(ctx context.Context, key string) (bool, error) {
//	    var live bool
//	    err := s.db.QueryRowContext(ctx,
//	        `SELECT EXISTS (SELECT 1 FROM idempotency_keys WHERE key = $1
//	         AND (expires_at IS NULL OR expires_at > now()))`,
//	        key,
//	    ).Scan(&live)
//
//	    return live, err
//	}
//
//	func (s *SQLStore) Record(ctx context.Context, key string, ttl time.Duration) error {
//	    expiresAt, err := expiryArg(ttl)
//	    if err != nil {
//	        return err
//	    }
//	    // Set the expiry only when the row is absent or already expired. A
//	    // live row is left untouched (Record never extends, upgrades, or
//	    // downgrades a claim); the NULL guard makes that explicit for a
//	    // permanent claim, whose own predicate would evaluate to NULL.
//	    _, err = s.db.ExecContext(ctx,
//	        `INSERT INTO idempotency_keys (key, expires_at) VALUES ($1, $2)
//	         ON CONFLICT (key) DO UPDATE SET expires_at = EXCLUDED.expires_at
//	         WHERE idempotency_keys.expires_at IS NOT NULL
//	           AND idempotency_keys.expires_at <= now()`,
//	        key, expiresAt,
//	    )
//
//	    return err
//	}
//
//	func (s *SQLStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
//	    expiresAt, err := expiryArg(ttl)
//	    if err != nil {
//	        return err
//	    }
//
//	    // One atomic statement does the whole job. The INSERT claims a new
//	    // key; the ON CONFLICT branch takes over only a row that has expired.
//	    // A live row (future expires_at) or a permanent claim (NULL) leaves
//	    // RowsAffected at 0, which is the duplicate. Concurrent claims serialize
//	    // on the row lock and re-evaluate the WHERE clause after the winner
//	    // commits, so exactly one of them sees RowsAffected 1.
//	    res, err := s.db.ExecContext(ctx,
//	        `INSERT INTO idempotency_keys (key, expires_at) VALUES ($1, $2)
//	         ON CONFLICT (key) DO UPDATE SET expires_at = EXCLUDED.expires_at
//	         WHERE idempotency_keys.expires_at IS NOT NULL
//	           AND idempotency_keys.expires_at <= now()`,
//	        key, expiresAt,
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    claimed, err := res.RowsAffected()
//	    if err != nil {
//	        return err
//	    }
//	    if claimed == 0 {
//	        return idempotency.ErrDuplicate
//	    }
//
//	    return nil
//	}
//
//	func (s *SQLStore) Close() error { return s.db.Close() }
//
// Record and CheckAndRecord run the same statement. Only the return differs:
// Record swallows the duplicate (a no-op), CheckAndRecord reports it.
//
// # SQL adapter notes — SQLite specifics (battle-tested in production)
//
// The PostgreSQL shape above ports directly to SQLite (placeholders become
// "?"), with four pitfalls that only surface in production use. The first
// three were found the hard way in a production consumer (KeyHolderAI's
// command-dedup store, 2026-10-03):
//
//  1. ":memory:" is PER-CONNECTION: every pooled *sql.DB connection gets its
//     own private, empty database. Use "file:<name>?mode=memory&cache=shared"
//     so the pool shares one schema — and SetMaxOpenConns(1) in tests.
//
//  2. A file-backed pool needs WAL + busy_timeout on the DSN, or the lazy
//     purge racing a concurrent CheckAndRecord fails instantly with
//     SQLITE_BUSY (a 5xx on a legitimate command):
//
//     dsn += "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
//
//     DSN pragmas apply to every pooled connection; a PRAGMA via db.Exec
//     configures only one.
//
//  3. CheckAndRecord needs NO explicit transaction: the PRIMARY KEY makes a
//     single INSERT the atomic claim, and RowsAffected() detects the duplicate
//     in one statement. Reuse the PostgreSQL statement above with "?" in place
//     of "$n", binding the expiry from expiryArg (NULL for Forever) and the
//     current instant rendered as in (4). A plain ON CONFLICT(key) DO NOTHING
//     is wrong here: it refuses the takeover of an expired row too, so that
//     claim could never be reclaimed and the key would stay blocked forever.
//
//  4. SQLite has no TIMESTAMPTZ: an expiry stored as TEXT must use a
//     fixed-width fraction. time.RFC3339Nano trims trailing zeros, so
//     lexicographic TEXT comparison is not chronological order:
//     "2026-10-11T05:00:00Z" sorts AFTER "2026-10-11T05:00:00.5Z"
//     (byte 'Z' > '.'), which misorders expired-claim checks by any
//     sub-second amount, and claim TTLs are routinely milliseconds.
//     Render expiries with an always-9-digit fraction instead:
//
//     expiresAt := t.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
//
//     A permanent claim stores NULL instead of a rendered instant, and the
//     live predicate becomes expires_at IS NULL OR expires_at > ?.
//
//     Fixed width makes lexicographic order chronological order: no schema
//     change, no driver change (found in sperrmuell-direct's SQLite store,
//     2026-10-11; issue #12).
//
// A lazy purge (DELETE ... WHERE expires_at IS NOT NULL AND expires_at <= now)
// on the hot path keeps the table bounded; NULL rows are permanent claims and
// are never purged. It is the query that collides with concurrent claims, which
// is why (2) matters.
//
// Example scope: Redis and SQL adapters are documented because they cover the
// dominant deployments. DynamoDB, MongoDB, and other backend example requests
// are declined (ADR-001) — the pattern above plus your backend's conditional
// write primitive is the template.
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
//
// # Client-side vs receiver-side posture
//
// Everything in this package guards the RECEIVER: the side that would
// otherwise process the same command twice. When the API you CALL already
// deduplicates server-side and replays the resource on retry (Wise and
// Stripe both do), a local key-only guard adds no safety and actively
// degrades the retry: it turns "here is the transfer you already created"
// into a 409. Client-side idempotency against such an API is exactly the
// replay recipe above (stable caller-held key plus the stored response), or
// nothing at all; a bare CheckAndRecord belongs on the receiving end of a
// request. This is the posture recorded by the wise-go adoption review
// (2026-11): non-adoption, because Wise owns the dedup contract server-side
// and a retry deserves the resource, not a conflict.
package idempotency
