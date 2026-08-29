## Summary

What does this PR change, and why? One or two sentences a new contributor can
understand without opening the diff.

## Checklist

- [ ] `go test ./... -race -count=1` passes locally
- [ ] `golangci-lint run ./...` reports 0 issues
- [ ] `gofmt -l .` prints nothing
- [ ] `go mod tidy` produces no diff
- [ ] `./scripts/check-stale-refs.sh` passes (living docs clean)
- [ ] Tests added for new behavior (contract subtests for `Store` semantics)
- [ ] Docs updated where the promise changed (README, godoc, FEATURES.md)
- [ ] CHANGELOG.md has an entry under `[Unreleased]`
- [ ] No new runtime dependencies (dev/test-only needs a strong justification)

## Contract-suite impact

Does this change `Store` semantics or add invariants? If yes:

- [ ] New/changed subtests in `contract/contract.go`
- [ ] README invariant table updated
- [ ] Internal self-test (`contract/contract_test.go`) still green
- [ ] Negative tests still fail for their broken Stores, pass for correct ones
