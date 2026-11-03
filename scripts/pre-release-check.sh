#!/usr/bin/env bash
# pre-release-check.sh — one failure-loud command encoding the entire release
# gate for go-idempotency (pareto plan 2026-10-19, task T1; pattern ported
# from go-etag's validated release protocol). Run before every tag.
#
# CI already runs gofmt/vet/lint/tidy/fuzz continuously; this bundle adds the
# release-only gates that CI cannot check on an untagged HEAD:
#   - tag↔CHANGELOG parity (the v0.3.1 gap this script exists to prevent)
#   - anchored go-directive floor (anti-silent-relaxation defense)
#   - tidy-diff-on-a-copy, go mod verify, erraudit, stale-refs
#
# Gate list: gofmt, go vet, go build, go test -race, golangci-lint
# (config verify + run), tidy-diff-on-copy, go mod verify, erraudit
# (--enforce-go-error-family), stale-refs, anchored go directive,
# tag↔CHANGELOG parity, clean tree (warn-only).
#
# Usage: scripts/pre-release-check.sh   (exit 0 = all gates green)

set -euo pipefail

cd "$(dirname "$0")/.."

# The login shell may export cache paths pointing at a nonexistent
# /mnt/buildcache. A release gate must be environment-proof, so pin writable
# caches unconditionally (the same prefix AGENTS.md documents).
export GOCACHE=/tmp/go-build-cache-idem
export GOMODCACHE=/tmp/go-mod-cache-idem
export GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache-idem

# The toolchain floor is a deliberate decision recorded in ADR-005. Changing
# it silently (auto-commit, dependency sweep) is the go-etag 969d077
# daemon-relaxation incident class; this anchor makes any drift a loud
# failure. To bump the floor consciously: update this constant AND ADR-005
# in the same commit.
EXPECTED_GO_DIRECTIVE="go 1.27"

step_no=0
failures=0
failed_gates=()

gate() {
	local name="$1"
	shift
	step_no=$((step_no + 1))
	printf '\n[%02d] %s\n' "$step_no" "$name"
	if "$@"; then
		echo "  PASS"
	else
		echo "  FAIL"
		failures=$((failures + 1))
		failed_gates+=("$name")
	fi
}

summary() {
	echo
	if [ "$failures" -eq 0 ]; then
		echo "pre-release-check: ALL ${step_no} GATES GREEN"
	else
		echo "pre-release-check: ${failures}/${step_no} GATES FAILED:"
		local g
		for g in "${failed_gates[@]}"; do
			echo "  - $g"
		done
		echo "Fix every failed gate before tagging (see RELEASING.md)."
	fi
}
trap summary EXIT

gate_gofmt() {
	local unformatted
	unformatted="$(gofmt -l .)"
	if [ -n "$unformatted" ]; then
		echo "  files not gofmt-formatted:"
		echo "$unformatted"
		return 1
	fi
}

gate_vet() {
	go vet ./...
}

gate_build() {
	go build ./...
}

gate_test() {
	go test ./... -race -count=1
}

gate_lint() {
	golangci-lint config verify
	golangci-lint run ./...
}

gate_tidy() {
	# go mod tidy on a copy, never the live tree (RELEASING.md §3): protects
	# uncommitted work and still catches a drifted go.mod/go.sum.
	local tmp=/tmp/pre-release-tidycheck
	rm -rf "$tmp"
	mkdir -p "$tmp"
	cp go.mod go.sum "$tmp"/
	cp ./*.go "$tmp"/
	cp -r contract internal middleware example "$tmp"/
	(cd "$tmp" && go mod tidy >/dev/null 2>&1)
	diff go.mod "$tmp/go.mod" && diff go.sum "$tmp/go.sum"
}

gate_modverify() {
	go mod verify
}

gate_erraudit() {
	erraudit . --enforce-go-error-family
}

gate_stalerefs() {
	./scripts/check-stale-refs.sh
}

gate_godirective() {
	local line
	line="$(grep -E '^go [0-9]' go.mod || true)"
	if [ "$line" != "$EXPECTED_GO_DIRECTIVE" ]; then
		echo "  go.mod says '$line', want exactly '$EXPECTED_GO_DIRECTIVE'."
		echo "  The floor is deliberate (ADR-005). To change it, update this"
		echo "  script's EXPECTED_GO_DIRECTIVE and ADR-005 in the same commit."
		return 1
	fi
}

gate_parity() {
	# The v0.3.1 release shipped without a CHANGELOG section because the
	# checklist was run manually and a step was skipped. This gate makes the
	# omission mechanically impossible: the latest tag must have a section,
	# and the [Unreleased] compare link must target it.
	local latest version
	latest="$(git tag -l 'v*' | sort -V | tail -1)"
	if [ -z "$latest" ]; then
		echo "  no version tags found; nothing to check parity against"
		return 1
	fi
	version="${latest#v}"
	if ! grep -q "^## \[${version}\] " CHANGELOG.md; then
		echo "  latest tag ${latest} has no '## [${version}]' section in CHANGELOG.md."
		echo "  Cut the section before tagging (RELEASING.md steps 5-6)."
		return 1
	fi
	if ! grep -q "^\[Unreleased\]: .*compare/${latest}\.\.\.HEAD$" CHANGELOG.md; then
		echo "  [Unreleased] compare link does not target ${latest}...HEAD."
		return 1
	fi
	echo "  tag ${latest} has a CHANGELOG section; [Unreleased] targets it"
}

gate_cleantree() {
	# Warn-only by design: a dirty tree does not fail the gate, but releasing
	# from one is how unintended work rides along in a tag.
	local dirty
	dirty="$(git status --porcelain)"
	if [ -n "$dirty" ]; then
		echo "  WARNING: working tree not clean; release from a clean tree:"
		echo "$dirty"
	fi
	return 0
}

echo "pre-release-check: go-idempotency release gate at $(git rev-parse --short HEAD 2>/dev/null || echo unborn)"

gate "gofmt: gofmt -l ." gate_gofmt
gate "vet: go vet ./..." gate_vet
gate "build: go build ./..." gate_build
gate "test: go test ./... -race -count=1" gate_test
gate "lint: golangci-lint config verify + run" gate_lint
gate "tidy: go mod tidy diff (on a copy)" gate_tidy
gate "modverify: go mod verify" gate_modverify
gate "erraudit: --enforce-go-error-family" gate_erraudit
gate "stalerefs: check-stale-refs.sh" gate_stalerefs
gate "godirective: anchored ${EXPECTED_GO_DIRECTIVE}" gate_godirective
gate "parity: tag↔CHANGELOG" gate_parity
gate "cleantree: git status (warn-only)" gate_cleantree

if [ "$failures" -gt 0 ]; then
	exit 1
fi
