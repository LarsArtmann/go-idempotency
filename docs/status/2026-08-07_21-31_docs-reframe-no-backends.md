# Status Report: 2026-08-07 Documentation Reframe — Interface-First SDK

> **Session goal:** Make it clear in all docs that go-idempotency SHOULD NOT provide actual backends — only an API/SDK interface that the end user implements.

---

## Executive Summary

Reframed the entire documentation surface to communicate a single, consistent message: **go-idempotency is an interface-first SDK. It provides the `Store` interface and `MemoryStore` (a reference implementation only). It will NOT ship production backends (Redis, SQL, etc.). Consumers implement the interface against their own backend.**

9 files were updated. Build, vet, race tests, and 60+ linters pass clean. However, the reframe introduced **stale line-number references** in FEATURES.md and DOMAIN_LANGUAGE.md that need fixing, and there are several consistency gaps that remain. _[2026-08-29: the stale references and gaps were fixed in `e8d545c`; per-item verdicts inline below.]_

---

## a) FULLY DONE

| #  | What                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Files                                                | Verified                  |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | ------------------------- |
| 1  | **`doc.go` design philosophy section added** — "Interface-First, You Implement the Backend" heading with clear rationale (no backends, you implement, interface is small, atomic primitives named)                                                                                                                                                                                                                                                                                     | `doc.go:29-51`                                       | `go build`, `go vet` pass |
| 2  | **`store.go` comments reframed** — `Store` interface doc now says "MemoryStore is provided as a reference implementation; implement this interface against your own backend for production." `CheckAndRecord` comment changed from "for a future Redis store it would be..." to "When implementing the interface against your own backend, use its native atomic primitive." `MemoryStore` context comment changed from "future network backends" to "custom backend implementations." | `store.go:43-46`, `store.go:60-64`, `store.go:75-77` | `go build`, `go vet` pass |
| 3  | **`README.md` design philosophy section added** — New section between "How it works" and "Quick start" with bold framing: "SDK, not a batteries-included framework." Ships interface + reference impl only. Will never add Redis/SQL. You implement.                                                                                                                                                                                                                                   | `README.md:41-50`                                    | Manual read               |
| 4  | **`README.md` Status & roadmap rewritten** — Removed "Redis and SQL are planned." Removed "v1.0 ships when a persistent backend exists." Now says "This library will not add production backends. That is by design." Versioning updated.                                                                                                                                                                                                                                              | `README.md:119-125`                                  | Manual read               |
| 5  | **`ROADMAP.md` Distributed Backends section replaced** — Old section listed Redis and SQL as planned with open questions. New section "Backend Implementations (Out of Scope by Design)" explains why + gives implementation guidance.                                                                                                                                                                                                                                                 | `ROADMAP.md:5-14`                                    | Manual read               |
| 6  | **`FEATURES.md` Redis/SQL moved to NOT PLANNED** — Removed from PLANNED table. New "NOT PLANNED (Out of Scope by Design)" section with per-feature rationale.                                                                                                                                                                                                                                                                                                                          | `FEATURES.md:34-41`                                  | Manual read               |
| 7  | **`TODO_LIST.md` contract test reworded** — Changed "reuse when Redis/SQL backends ship" to "consumers reuse it to verify their own backend implementations."                                                                                                                                                                                                                                                                                                                          | `TODO_LIST.md:10`                                    | Manual read               |
| 8  | **`docs/DOMAIN_LANGUAGE.md` definitions updated** — `Store` entry now says "core SDK contract that consumers implement against their own backend." `MemoryStore` entry now says "reference implementation" and notes no production backends shipped.                                                                                                                                                                                                                                   | `DOMAIN_LANGUAGE.md`                                 | Manual read               |
| 9  | **`AGENTS.md` updated** — MemoryStore called "reference implementation." New design decision: "No production backends — by design." New subsection: "Backend Implementations (Out of Scope)." Context comment reframed.                                                                                                                                                                                                                                                                | `AGENTS.md:22,33-34,40-42`                           | Manual read               |
| 10 | **`CHANGELOG.md` unreleased entry added** — Documents the reframe across all affected files with sub-bullets for the major structural changes.                                                                                                                                                                                                                                                                                                                                         | `CHANGELOG.md:8-15`                                  | Manual read               |
| 11 | **Verification gates passed** — `go test ./... -race -count=1` (8s, clean), `go vet ./...` (clean), `golangci-lint run ./...` (0 issues). Grep confirmed no living doc still says backends are "planned" or "will ship."                                                                                                                                                                                                                                                               | —                                                    | All pass                  |

---

## b) PARTIALLY DONE

| #     | What                                                          | Why partial                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | Impact                                                                                     |
| ----- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| ~~1~~ | ~~**Consistency sweep of FEATURES.md**~~ done at `e8d545c`    | ~~The evidence column (line references to `store.go`) is now **stale** — my edits to the `Store` interface comment shifted all line numbers downward by ~3-17 lines. Example: FEATURES.md says `store.go:30-48` for the Store interface, but it is now at `store.go:47-66`. Says `store.go:56-61` for MemoryStore, actually `store.go:78-83`. Says `store.go:118-129` for CheckAndRecord, actually `store.go:147-163`. These were already approximately stale before my edits, but I made them worse.~~ | ~~Medium — evidence column is inaccurate, undermines the "verified against code" promise~~ |
| ~~2~~ | ~~**FEATURES.md MemoryStore description**~~ done at `e8d545c` | ~~Line 10 still says "in-memory `Store` **implementation**" — should say "**reference** implementation" to match the new framing used everywhere else (line 36, README, doc.go, AGENTS.md, DOMAIN_LANGUAGE.md all say "reference implementation").~~                                                                                                                                                                                                                                                    | ~~Low — minor inconsistency within the same file~~                                         |
| ~~3~~ | ~~**DOMAIN_LANGUAGE.md line reference**~~ done at `e8d545c`   | ~~Says `store.go:30-48` for the Store interface — now stale (actual: `store.go:47-66`).~~                                                                                                                                                                                                                                                                                                                                                                                                               | ~~Low — one stale reference~~                                                              |

---

## c) NOT STARTED

| #     | What                                                                     | Why it matters                                                                                                                                                                                                                                                                                  |
| ----- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~ | ~~**Fix stale line-number references**~~ done at `e8d545c`               | ~~FEATURES.md evidence column and DOMAIN_LANGUAGE.md both reference `store.go` line numbers that no longer match. Should update all references to match current line numbers, or switch to a more durable reference style (symbol names, section anchors).~~                                    |
| ~~2~~ | ~~**Add a custom backend implementation example**~~ done at `43b236a`    | ~~Every doc now says "you implement the Store interface" but none shows what that looks like. A minimal example (e.g., a Redis `SET NX` adapter, 15-20 lines) in `doc.go` or README would make the promise actionable instead of abstract. This is the single highest-value addition missing.~~ |
| ~~3~~ | ~~**CONTRIBUTING.md update**~~ done at `43b236a`                         | ~~Not touched. Doesn't mention backends explicitly, but could add a note that backend implementations are out of scope and won't be accepted as PRs (manages contributor expectations).~~                                                                                                       |
| ~~4~~ | ~~**Store interface contract test suite**~~ done at `46aa38d`, `9db0f6e` | ~~Already in TODO_LIST.md but not started. This is now MORE important than before — it's the primary tool consumers use to validate their own backends. The docs repeatedly reference it as the verification path.~~                                                                            |

---

## d) TOTALLY FUCKED UP

**Nothing is totally fucked up.** No behavioral or API changes were made. All changes are documentation-only. Build, vet, tests, and lint all pass clean. The stale line references (section b) are the worst issue and are easily fixable.

---

## e) WHAT WE SHOULD IMPROVE (Self-Critique)

### Process mistakes this session

1. **I didn't re-verify line-number references after editing `store.go`.** I added 3 lines to the `Store` interface comment, which shifted every subsequent line number. FEATURES.md and DOMAIN_LANGUAGE.md both cite specific line numbers as "evidence." I should have either (a) re-run the line references after the edit, or (b) noticed during the FEATURES.md edit that the evidence column existed and would be invalidated. This is the same class of error as the AGENTS.md cross-cutting lesson: "always `go build` after deleting a package, before editing dependents" — except here it's "always check line references after adding lines to a file others cite."

2. **I should have checked FEATURES.md for framing consistency.** I edited FEATURES.md to move Redis/SQL to NOT PLANNED, but didn't update the MemoryStore description on line 10 from "implementation" to "reference implementation." Half the file was updated, half wasn't. Incomplete edit within a single file.

3. **No implementation example was provided.** The user said "make it clear that the end user implements." I made the _claim_ clear in every doc, but didn't provide the _evidence_ — a code example showing how simple it is to implement the interface. The message would land much harder with a 15-line Redis example. This is a "show, don't tell" gap.

### Broader improvements (pre-existing, not caused by this session)

4. **FEATURES.md evidence column uses brittle line numbers.** These break on every edit to `store.go`. Should consider switching to symbol-name-based references (e.g., "`Store` interface in `store.go`") or accepting that they'll be approximate. The file has been edited 3 times since v0.1.0 and the line numbers were already drifting before this session.

5. **The "Store interface contract test suite" (TODO_LIST.md) is now the critical-path item.** The docs frame it as the tool consumers use to verify their backends. But it doesn't exist yet. The documentation is making a promise ("implement the interface, verify with contract tests") that the codebase can't support yet. This creates a gap between marketing and reality.

6. **CONTRIBUTING.md doesn't mention the no-backends policy.** A contributor reading CONTRIBUTING.md and ROADMAP.md (old version) might have submitted a Redis store PR. Now ROADMAP.md is clear, but CONTRIBUTING.md still doesn't set the expectation. Should add a "Scope" or "What won't be accepted" note.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate fixes (this session's debt)

1. ~~**Fix all stale `store.go` line references in FEATURES.md** — update to current line numbers~~ done at `e8d545c`
2. ~~**Fix stale `store.go:30-48` reference in DOMAIN_LANGUAGE.md** — update to current~~ done at `e8d545c`
3. ~~**Update FEATURES.md line 10** — "implementation" → "reference implementation"~~ done at `e8d545c`
4. ~~**Verify no other files cite stale `store.go` line numbers** — grep for `store.go:` across all `.md` files~~ done at `e8d545c`

### High-value additions

5. ~~**Add a custom Store implementation example** — minimal Redis `SET NX` adapter (~15-20 lines) in `doc.go` or README, showing how simple implementing the interface is~~ done at `43b236a`
6. ~~**Add Store interface contract test suite** — table-driven tests any `Store` impl must pass (already in TODO_LIST.md, now critical-path for the SDK framing)~~ done at `46aa38d`, `9db0f6e`
7. ~~**Add CONTRIBUTING.md scope note** — "Backend implementations (Redis, SQL, etc.) are out of scope and will not be accepted as PRs. Implement the Store interface in your own project."~~ done at `43b236a`

### Interface evolution (from TODO_LIST.md + ROADMAP.md)

8. **Add `Delete` method to `Store` interface** — manual key invalidation for ops/testing
9. **Add `Stats` method to `Store` interface** — hit/miss/expiry counts for observability
10. **Consider `Reset` method** — clear all entries (useful for testing)
11. **Evaluate whether `Store` should return `Store` (self) for fluent chaining or stay simple**
12. **Consider a `StoreCloser` interface** (`io.Closer`) — `MemoryStore` has `Close()` but `Store` interface doesn't require it. Should cleanup be part of the contract?

### Middleware layer (from ROADMAP.md — the next big module addition)

13. **Design middleware package API** — `CommandIdempotency`, `EventIdempotency`, `QueryIdempotency`
14. **Implement `CommandIdempotency` middleware** — the most common use case, first to ship
15. **Implement `EventIdempotency` middleware** — for event handlers with at-least-once delivery
16. **Implement `QueryIdempotency` middleware** — for cacheable query dedup
17. **Design transport-agnostic middleware interface** — works with HTTP, gRPC, message queues
18. **HTTP middleware adapter** — extract idempotency key from `Idempotency-Key` header
19. **gRPC interceptor adapter** — extract key from metadata
20. **Message queue consumer adapter** — extract key from message attributes

### Key generation utilities (from ROADMAP.md)

21. **UUID v7 key generator** — time-ordered, sortable, collision-resistant
22. **Content-hash key generator** — deterministic keys from request body hash
23. **Request-derived key generator** — composite keys from user ID + operation + parameters
24. **Key validation utility** — length, character set, format checks

### Testing improvements (from TODO_LIST.md + self-identified)

25. ~~**Add fuzz tests** — `FuzzCheckAndRecord`, `FuzzRecord` with arbitrary keys, TTLs, concurrency~~ done at `9db0f6e`
26. **Add concurrent fuzz test** — randomized goroutine counts + interleavings
27. **Benchmark sweep goroutine overhead** — measure cost of background sweep vs disabled
28. ~~**Benchmark memory usage** — `testing.B` with `runtime.ReadMemStats` for large key counts~~ done at `46aa38d`
29. **Test context cancellation on custom backends** — contract test should verify ctx is honored
30. ~~**Add race test for Close + concurrent operations** — verify no panic on use-after-close~~ done at `9db0f6e`

### Observability (from ROADMAP.md)

31. **Add metrics hooks to MemoryStore** — hit/miss/expiry/contention counters
32. **Add structured logging** — optional `*slog.Logger` for sweep, expiry, operations
33. ~~**Add tracing support** — OpenTelemetry spans for store operations~~ **Won't implement — rejected in planning doc R1 (OpenTelemetry).**
34. ~~**Evaluate sharded mutex design** — benchmark vs single `sync.RWMutex` under contention~~ **Won't implement — rejected in planning doc R4 (sharded mutex).**
35. ~~**Evaluate `sync.Map`** — benchmark for read-heavy workloads~~ **Won't implement — rejected in planning doc R2 (sync.Map).**
36. ~~**Evaluate lock-free approaches** — CAS-based map for extreme contention~~ **Won't implement — rejected in planning doc R3 (lock-free).**

### Documentation improvements

37. ~~**Add "Implementing your own backend" guide** — dedicated doc page with patterns, pitfalls, testing~~ done at `43b236a`
38. ~~**Add architecture decision record (ADR)** — why no backends, why interface-first~~ done at `9db0f6e`
39. **Add example repo link** — reference implementation of a Redis backend in a separate repo
40. ~~**Add godoc examples** — `Example()` functions that appear on pkg.go.dev~~ done at `9db0f6e`
41. ~~**Rewrite README Features section** — lead with "Store interface" as the product, not MemoryStore features~~ done at `43b236a`
42. ~~**Add comparison table** — vs other idempotency libraries (stripe/go-idempotency, etc.)~~ **Won't implement — rejected in planning doc R5 (comparison table).**

### CI/CD and release

43. ~~**Add `Store` contract test to CI** — runs against MemoryStore on every push~~ done at `9db0f6e`
44. ~~**Set up GoReleaser** — automated tagged releases with changelog extraction~~ **Won't implement — rejected in planning doc R6 (GoReleaser).**
45. ~~**Add code coverage reporting** — `go test -cover` + Codecov/badge~~ done at `9db0f6e`
46. ~~**Add `gosec` to CI** — security scanning (gosec is in golangci-lint but not as standalone gate)~~ done (gosec enabled in .golangci.yml)
47. ~~**Add dependency scanning** — Dependabot or Renovate for `go.mod`~~ done at `9db0f6e`
48. **Tag v0.2.0** — after middleware package or contract test suite lands
49. ~~**Plan v1.0 criteria** — interface stability, multiple independent backend implementations in the wild, middleware layer shipped~~ done (ROADMAP v1.0 criteria documented)

### Polish

50. ~~**Standardize em-dash usage** — AGENTS.md `store.go:61` uses `*what`.`(missing opening backtick). Search for similar typos across docs.~~ done at`e8d545c`

---

## g) Questions I CANNOT Figure Out Myself

### 1. Should `MemoryStore` be deprecated/removed eventually, or kept permanently?

The reframe positions `MemoryStore` as a "reference implementation for development and single-process use cases." But some interface-first SDKs (like `database/sql` without a driver) provide NO default implementation — they force you to bring your own. If the goal is to push consumers toward implementing their own backend, keeping `MemoryStore` might undermine that message (it's too easy to just use `MemoryStore` and never implement the interface). Should `MemoryStore` stay as a permanent citizen, or should there be a migration path toward deprecating it in favor of consumer-owned implementations? This is a product/philosophy decision I cannot make. _Answered by the owner on 2026-08-07: "deprecate it" — executed in `5848f38` and `67fa850`; removal targeted for v1.0._

### 2. Should the middleware package live in this module or a separate module?

`doc.go` references a "future middleware package" with `CommandIdempotency`, `EventIdempotency`, `QueryIdempotency`. The no-backends philosophy is rooted in avoiding dependency bloat. But middleware might need transport dependencies (HTTP, gRPC). Should middleware be:

- (a) A subpackage in this same module (`github.com/larsartmann/go-idempotency/middleware`)?
- (b) A separate module (`github.com/larsartmann/go-idempotency-middleware`)?
- (c) Multiple transport-specific modules?

This determines the module boundary architecture and I can't decide it without knowing your dependency philosophy and how much you want to keep the core module dependency-free.

### 3. Should there be an `ErrStoreClosed` sentinel error?

Currently, operations on `MemoryStore` after `Close()` silently succeed (they still work, only the sweep goroutine stops). There's no error to signal "this store is closed." With the SDK framing where consumers implement their own backends (which might have real connection-close semantics), should the `Store` interface define a sentinel `ErrStoreClosed` that implementations return after shutdown? Or is close-handling intentionally left as an implementation detail? This affects the interface contract and I can't determine the right answer without knowing how you expect consumers to handle backend lifecycle.

---

## Files Changed This Session

| File                      | Type                     | Change                                                                        |
| ------------------------- | ------------------------ | ----------------------------------------------------------------------------- |
| `doc.go`                  | Go source (comment only) | Added "Design Philosophy" section                                             |
| `store.go`                | Go source (comment only) | Reframed 3 comments: Store interface, CheckAndRecord, MemoryStore context     |
| `README.md`               | Docs                     | Added "Design philosophy" section; rewrote "Status & roadmap"                 |
| `ROADMAP.md`              | Docs                     | Replaced "Distributed Backends" with "Backend Implementations (Out of Scope)" |
| `FEATURES.md`             | Docs                     | Moved Redis/SQL to "NOT PLANNED"; left stale line refs (debt)                 |
| `TODO_LIST.md`            | Docs                     | Reworded contract test description                                            |
| `docs/DOMAIN_LANGUAGE.md` | Docs                     | Updated Store + MemoryStore definitions; left stale line ref (debt)           |
| `AGENTS.md`               | Docs                     | Added no-backends design decision + out-of-scope section                      |
| `CHANGELOG.md`            | Docs                     | Added `[Unreleased]` entry                                                    |

**No `.go` behavioral changes. No `go.mod` changes. No test changes. Documentation-only session.**
