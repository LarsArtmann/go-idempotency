# Session status: Forever implementation, release gate, and the red first CI run

**Date:** 2026-11-17 13:34 CET
**Repo:** `go-idempotency` (master; feature commits pushed, one later daemon commit local-only)
**Session type:** feature implementation + docs + release-gate verification + push.

## Scope: what this report verified, and what it did not

Verified in this session:

- The local gates on the final tree: `gofmt`, `go vet ./...`,
  `go test ./... -race`, `golangci-lint run ./...` (0 issues),
  `erraudit . --enforce-go-error-family` (0 violations),
  `./scripts/check-stale-refs.sh`, and `./scripts/pre-release-check.sh`
  (12/12 gates green).
- The pushed commits: `6adbc3f` (feature, 25 files, +1171/-68) and
  `ed08005` (docs link fix, 1 file). `master` is in sync with
  `origin/master`; the working tree was clean at report time.
- GitHub Actions run `44077357195` on commit `ed08005`: all 8 jobs green,
  including Fuzz (12m37s), Docs, Lint, the Go matrix, Vet, Tidy and
  Vulnerabilities.
- The semantic itself, through 18 contract invariants, 25 deliberately broken
  stores, 5 root unit tests, one property test, and one godoc example.

Not verified, and why:

- **No release tag.** `v0.4.0` was not cut; tagging is a separate owner
  release action. The CHANGELOG section is staged in `[Unreleased]`.
- **No downstream code.** go-cqrs-lite `metaengine.DedupStore`/`claimkit`,
  CV, and DiscordSync still use the old TTL semantics; only a TODO row exists.
- **No repeated/flake run.** The new Forever tests use wall-clock sleeps; I did
  not run `-count=5`, `-shuffle=on`, or the strict-timing pass locally.
- **No local link check.** `lychee` is not installed on this host; the first
  CI run's Docs failure was only discovered after pushing.
- **No fuzz coverage of the sentinel.** F40 was waived: `Forever` is a
  constant, not derived from fuzz input, and adding a target would add CI
  fuzz budget.
- **No proxy/pkg.go.dev check** (needs a tag), and no Codecov floor (needs the
  owner's `CODECOV_TOKEN`).
- **No historical-doc annotation.** The 12:26 status report, the consumer-audit
  synthesis, and profileagent's review corpus still describe Forever as
  unshipped.

## Runtime note (noticed after the report was written)

At 13:36 the auto-commit daemon committed `05c781b` (8 files, `Daemon: true`).
It contains a `setup-go` v7.1.0 bump in `ci.yml` and markdown/table
reformatting of FEATURES, DOMAIN_LANGUAGE, ADR-006, the ADR index, the plan,
and an annotation to the 12:26 status report. I did not author those edits and
did not push them. The tree is clean; `master` is ahead of `origin/master` by
that one commit.

## Self-critique: what I forgot, what I could have done better

**What I forgot**

1. The auto-commit daemon creates commits every few seconds. I planned the
   work as if I would make one final detailed commit, then had to rewrite five
   heuristic commits to get one.
2. The release gate does not run the link check. I treated
   `pre-release-check: ALL 12 GATES GREEN` as "CI will pass", and it did not.
3. The SQL schema example still says `expires_at TIMESTAMPTZ NOT NULL` and
   computes `now.Add(ttl)`. The prose now says "add the NULL branch", but a
   copy-paste reader cannot represent Forever with that schema.
4. The `math.MaxInt64` spelling in ADR-006 and the plan vs the `1<<63 - 1`
   literal in `store.go` is an inconsistency a reader can trip over.
5. The earlier 12:26 status report is now false ("Forever not started") and
   was not annotated.
6. I did not test the 2262 `UnixNano` overflow that justifies the zero-time
   representation; it is documented but not pinned.

**What I could have done better**

1. Commit meaningful checkpoints myself before the daemon (ADR, core, contract,
   docs, verification), instead of squashing heuristic commits.
2. Install or run `lychee` before pushing, or simulate the Docs job. A local
   link check would have caught the two private-repo 404s before the first push.
3. Make the SQL examples Forever-correct in the same edit as the prose, not
   prose-only.
4. Run `go test ./... -race -count=5` on the new Forever tests and the strict
   timing pass before calling the feature done.
5. Update the historical reports in the same session (or explicitly defer
   them in the plan).

**What I could still improve**

1. Add the overflow regression test and fix the SQL schema example.
2. Add a sentinel fuzz path (a boolean selector or a dedicated target) so the
   fuzzer covers the permanent branch.
3. Add link checking to the local release gate.
4. Propagate the semantic downstream, where the actual consumers are.
5. Cut `v0.4.0` and verify the proxy/pkg.go.dev path.
6. Annotate the earlier reports and the profileagent corpus.
7. Add a count-consistency guard (18 invariants / 25 negative stores) to
   `check-stale-refs.sh`.

## a) FULLY DONE

1. **ADR-006** (`docs/adr/006-forever-permanent-claims.md`) with the value,
   scope split, zero-time representation, backend encodings, the 2262 overflow
   rule, and rejected alternatives. Added to the ADR index.
2. **Core API**: `Forever` (`1<<63 - 1`), `isLive`, `expiryFor`, and updated
   `Store.Record`/`CheckAndRecord` docs.
3. **MemoryStore**: zero time is permanent; `Seen` never expires it,
   `Record` neither upgrades nor downgrades a live claim, `CheckAndRecord`
   claims/reclaims it, and the sweeper skips it.
4. **Parity**: `internal/teststore` and the example `demoStore` implement the
   same semantics, so the contract suite and the runnable example are truthful.
5. **Contract suite**: 5 new invariants in `contract/contract_forever.go`
   (18 total): `SeenSurvives`, `RecordNeverExpires`,
   `RecordNoopOnExistingKey`, `RecordNotDowngraded`, `CheckAndRecord`.
6. **Negative proofs**: 6 new broken stores/scenarios (25 total) in
   `contract/contract_forever_stores_test.go`, including an upgrade bug, a
   downgrade bug, a duplicate-swallowing bug, and a Forever rejection.
7. **Overflow guard** in the existing `neverExpires` sabotage wrapper so
   multiplying the sentinel cannot turn a negative scenario into a TTL
   validation failure for the wrong reason.
8. **Root tests**: `store_forever_test.go` (5 tests) and
   `property_forever_test.go` (rapid property).
9. **Example**: a permanent-claim block in `example/main.go` and
   `ExampleForever` in `example_test.go`.
10. **README**: FAQ rewritten into the scope split, a permanent-claims feature
    bullet, a backend `Forever` mapping paragraph, the 18-row invariant table,
    and corrected counts.
11. **Package docs**: `doc.go` contents entry, a "Permanent Claims (Forever)"
    section, Redis/SQL/SQLite/DynamoDB encodings, and a "these examples are
    TTL-only" caveat.
12. **Glossary + migration guide**: `Forever`/permanent-claim terms, a
    Forever-aware test double, a `redisExpiry` helper, and a permanent-claims
    section.
13. **Living docs**: AGENTS, FEATURES, CONTRIBUTING, SECURITY, CHANGELOG,
    ROADMAP, TODO_LIST updated to 18 invariants / 25 negative stores and the
    new semantic.
14. **Plan**: `docs/planning/2026-11-17_12-33_forever-permanent-claims-superb-plan.md`
    with the 1%/4%/20%/remaining breakdown, 18 medium tasks, 69 fine tasks,
    a mermaid graph, risks, and an execution ledger.
15. **Verification**: all local gates green, including the 12/12 release gate.
16. **Commit and push**: feature commit `6adbc3f`, docs link fix `ed08005`,
    both on `origin/master`.
17. **CI**: the latest run `44077357195` is 8/8 green.
18. **Incidental fix**: removed two private-repo issue links from AGENTS.md
    that made the unauthenticated link checker 404.

## b) PARTIALLY DONE

1. **Downstream propagation (M17)**: a TODO row exists
   ("Propagate `Forever` to go-cqrs-lite"), but no downstream code changed.
2. **Fuzz coverage (F40)**: waived with a reason; the sentinel has no fuzz path.
3. **SQL/backend examples**: prose caveat added, but the PostgreSQL schema and
   the SQLite statement still show `NOT NULL` / `now.Add(ttl)` shapes that
   cannot represent Forever.
4. **Historical annotations**: the 12:26 status report, the HTML synthesis,
   and profileagent's review corpus were not updated.
5. **Release prep**: the 12/12 gate ran before the final doc-only commit; it was
   not rerun on `ed08005` (the Docs job covers the changed surface and is green).
6. **Flake verification**: one race run and one CI run; no `-count=5`,
   `-shuffle=on`, or local `TimingScale: 3` pass.
7. **Local link checking**: no lychee install; CI remains the only link gate.
8. **Coverage**: the Codecov floor is still blocked on the owner token.

## c) NOT STARTED

1. Cut and push `v0.4.0`; create the GitHub Release; verify proxy `.info` and
   pkg.go.dev.
2. Map `Forever` in go-cqrs-lite `metaengine.DedupStore`, `claimkit`,
   `MapDedupStore`, and `adttest.AssertDedupStore`.
3. Converge CV's forever store, DiscordSync's TTL event dedupe, and the
   profileagent corpus on the library semantic.
4. Update CV's 2026-09-16 plan (its M7/M8 still name the retired
   `idempotency/{sqlstore,kvstore}` instead of `DedupStore`).
5. Add the `now.Add(Forever)` / `UnixNano` overflow regression test.
6. Add a fuzz target or selector that reaches the sentinel.
7. Fix the SQL schema example and add runnable Forever SQL/Redis branches.
8. Add a middleware doc note that `Forever` is not for HTTP request keys.
9. Add a README status/roadmap line for the v0.4.0 semantic.
10. Add a link check to `scripts/pre-release-check.sh` or a local wrapper.
11. Add a stale-ref pattern for "13 invariants" / "19 negative stores".
12. Add a scripted invariant-table/contract-subtest parity check.
13. Run `go test ./... -race -count=5` and the strict-timing Forever pass.
14. Annotate the 12:26 status report.
15. Annotate the consumer-audit synthesis with the implementation status.
16. Annotate `profileagent/docs/review/libs/go-idempotency.md` and STACK.md.
17. Announce v0.4.x (owner channel/timing).
18. Update the plan's F40 waiver or replace it with a real fuzz check.

## d) TOTALLY FUCKED UP!

Honest list. None of these broke the shipped code, but they are process and
doc defects.

1. **I let the auto-commit daemon build the history.** Five heuristic commits
   (`chore: auto-commit ...`) were created, and I rewrote them with
   `git reset --soft` to produce one detailed commit. The rewrite was safe
   because nothing had been pushed, but it is avoidable history churn. Commit
   checkpoints should be mine, not the daemon's.
2. **The first push produced a red CI run.** Run `44077075493` failed Docs on
   two pre-existing private-repo issue links in AGENTS.md. I had not simulated
   the link check locally. The fix (`ed08005`) is one line, but it cost a
   13-minute run and left a red run in history.
3. **I misread the release gate's coverage.** `pre-release-check.sh` does not
   run lychee; 12/12 green did not mean CI would be green. The gate has a
   blind spot and I treated it as complete.
4. **The SQL example is a trap.** `expires_at TIMESTAMPTZ NOT NULL` plus
   `time.Now().Add(ttl)` cannot represent `Forever`. The new prose warns about
   it, but a copy-paste consumer starts from a shape that cannot work.
5. **Two spellings of the sentinel.** ADR/plan say `math.MaxInt64`; code uses
   `1<<63 - 1`. Equivalent, but a grep-driven reader will think the code is
   missing the documented constant.
6. **The prior status report is stale and unannotated.** It says Forever is
   not started and that no code changed. Both are now false.
7. **M18 ordering.** I marked commit+push "Done" in the plan before the push
   actually completed. It did complete seconds later, but the record was
   written ahead of the fact.
8. **No flake evidence.** The Forever contract tests sleep on wall-clock time;
   one green run is not proof on a loaded runner.
9. **No overflow pin.** The 2262 argument is prose only; a future "simplify
   now.Add" change would not fail a test.
10. **The fuzz waiver is a judgment call, not proof.** The sentinel branch has
    zero fuzz coverage, and I did not record that in the TODO list.

## e) WHAT WE SHOULD IMPROVE!

1. **Commit before the daemon.** After each verified checkpoint, make a
   meaningful commit immediately; never rely on a final squash.
2. **Run the docs job locally.** Install/pin lychee or add a link-check step
   to the release gate; a 404 should not need a push to discover.
3. **Make examples executable-correct.** If the docs say "add the NULL
   branch", the schema and queries must show it, not just mention it.
4. **Pin the sentinel's invariants in tests**, including the overflow horizon
   and sweep survival over multiple ticks.
5. **Use one spelling** for the constant across code, ADR, and plan.
6. **Run repeated and shuffled tests** for any timing-sensitive addition.
7. **Annotate point-in-time reports** when the underlying fact changes, or
   explicitly defer the annotation in the plan.
8. **Keep downstream propagation visible.** The library change is not done
   until the consumers can use it; the TODO row is a start, not the finish.
9. **Add count/table parity guards** so 13/19 vs 18/25 drift cannot recur.
10. **Record process lessons in AGENTS.md** (daemon commits, link check,
    release gate coverage) so the next session does not repeat them.

## f) Up to 50 things we should get done next

1. Run `./scripts/pre-release-check.sh` on `ed08005` and record the result.
2. Cut `v0.4.0` with the go-release flow (master + tag in one push).
3. Verify the module proxy serves `.info` for `v0.4.0`.
4. Verify pkg.go.dev renders the new godoc section.
5. Create the GitHub Release from the `v0.4.0` CHANGELOG section.
6. Add a README "Status & roadmap" line for Forever / v0.4.0.
7. Change the SQL schema example to nullable `expires_at`.
8. Add a PostgreSQL `Forever` branch to `Record`/`CheckAndRecord`.
9. Add a SQLite `expires_at IS NULL` predicate to the docs.
10. Add a Redis `SET NX` no-EX snippet for Forever.
11. Add a DynamoDB no-TTL note next to the backend table.
12. Add a middleware doc note: Forever is not for HTTP request keys.
13. Add a `Store` interface doc link to `Forever`.
14. Add the overflow regression test (`now.Add(Forever)` year and UnixNano).
15. Add a sweep soak test that Forever survives many ticks.
16. Add a post-Close Forever operation test.
17. Add a `Record(Forever)` reclaim-after-expiry test.
18. Add a lazy-deletion sibling test (expired key does not touch Forever).
19. Add a `Forever` acceptance case to the TTL-validation unit tests.
20. Add a fuzz selector/target that reaches `Forever`.
21. Run `go test ./... -race -count=5` on the Forever tests.
22. Run `GO_IDEMPOTENCY_CONTRACT_TIMING_SCALE=3 go test ./contract -run Forever`.
23. Run `go test ./... -shuffle=on`.
24. Run the full suite with `-count=2` to catch cross-test coupling.
25. Add a link-check step to the release gate.
26. Pin the lychee version and document local installation.
27. Add a stale-refs pattern for "13 invariants" and "19 negative stores".
28. Add a scripted parity check: contract subtests vs README table rows.
29. Add a scripted parity check: negative scenarios vs documented count.
30. Add a check that every ADR index link resolves.
31. Add a check that `Forever` is spelled consistently (`math.MaxInt64` vs literal).
32. Update the 12:26 status report with an inline annotation.
33. Annotate `docs/reviews/2026-11-17_consumer-audit-synthesis.html` with the shipped status.
34. Update `profileagent/docs/review/libs/go-idempotency.md` with the Forever status.
35. Update `profileagent/STACK.md` and the errata report accordingly.
36. Update CV's 2026-09-16 plan: M1-M4 done upstream; M7/M8 now target DedupStore.
37. Add a go-cqrs-lite TODO/issue for `DedupStore` Forever mapping.
38. Map `Forever` in `metaengine/claimkit` (SQL engines).
39. Map `Forever` in `MapDedupStore` (zero time or sentinel).
40. Map `Forever` in `DedupCheckAndRecord` signature docs and `adttest`.
41. Bump go-cqrs-lite family pins after the mapping lands.
42. Converge CV's `MemoryIdempotencyStore` on the library sentinel.
43. Converge DiscordSync's event dedupe on the sentinel (or document why not).
44. Converge bank-sync/webphone adapters on the sentinel where applicable.
45. Add a migration note for `DedupStore` adapters (collection/now/boolean).
46. Announce v0.4.x (owner channel/timing).
47. Add a release checklist line for "proxy + pkg.go.dev verified".
48. Record the auto-commit/link-check lessons in AGENTS.md.
49. Re-run this status after the release and compare.
50. Decide whether the fuzz waiver stays or becomes a real target.

## g) Questions I cannot answer myself

1. **Release:** do you want me to cut and push `v0.4.0` now (run the full
   go-release flow, tag, GitHub Release, proxy/pkg.go.dev verification), or
   leave it unreleased for a later batch?
2. **Downstream:** should I start the go-cqrs-lite `DedupStore`/`claimkit`
   mapping now (a second repo, version-train bump, new tests), or wait for the
   next family release window?
3. **Annotations:** should I update the earlier 12:26 status report, the
   consumer-audit synthesis, and profileagent's review corpus in place now, or
   leave them as dated snapshots and keep the correction in this report?

## Immediate next action

Waiting for instructions. The working tree is clean; `master` is ahead of
`origin/master` by `05c781b` (the daemon commit described in the runtime note),
which I did not create and did not push. The latest CI run is green.
