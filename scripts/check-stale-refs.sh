#!/usr/bin/env bash
# check-stale-refs.sh — fail when LIVING docs contain references that are
# known to go stale.
#
# Living docs are the root-level *.md files plus docs/, excluding the
# historical snapshots (status reports, planning docs, feedback, release
# notes, reviews) and CHANGELOG.md. Historical documents record what was
# true at the time and are annotated, never rewritten; stale phrases in
# them are expected and allowed.
#
# Patterns (each one caused real doc drift before this script existed):
#   1. "reference implementation"     — the pre-deprecation endorsement of
#                                       MemoryStore; it is deprecated, not
#                                       recommended.
#   2. "single-process use cases"     — the old endorsement phrase from the
#                                       Store interface comment.
#   3. store.go:<line>                — file:line references rot on every
#                                       edit; reference symbols instead.
#   4. "not yet tagged"               — release-status drift; update the
#                                       version line when a tag is cut.
#   5. "staging in CHANGELOG"          — release-status drift; the version was
#                                       tagged but a doc still describes the
#                                       content as staging in [Unreleased].
#   6. "goes live with the next"       — placeholder note about a page that
#                                       materializes only with the next
#                                       release (e.g. a pkg.go.dev page).
#
# Usage: scripts/check-stale-refs.sh   (exit 0 = clean, 1 = stale refs found)

set -euo pipefail

cd "$(dirname "$0")/.."

status=0

check() {
	local pattern="$1" label="$2"
	local hits
	hits=$(grep -rnE --include='*.md' --exclude-dir=.git "$pattern" . |
		grep -vE '^\./(CHANGELOG\.md|docs/(status|planning|feedback|releases|reviews)/)' || true)
	if [ -n "$hits" ]; then
		echo "STALE: $label"
		echo "$hits"
		echo
		status=1
	fi
}

check 'reference implementation' \
	"MemoryStore described as 'reference implementation' (it is deprecated, not recommended)"

check 'single-process use cases' \
	"pre-deprecation endorsement phrase 'single-process use cases'"

check 'store\.go:[0-9]+' \
	"file:line reference to store.go (line numbers rot; reference the symbol)"

check 'not yet tagged' \
	"release-status drift: 'not yet tagged' (update the version line when the tag is cut)"

check 'staging in CHANGELOG' \
	"release-status drift: content described as staging in CHANGELOG [Unreleased] after the version was tagged"

check 'goes live with the next' \
	"release-status drift: placeholder note about a page that appears with the next release"

if [ "$status" -eq 0 ]; then
	echo "check-stale-refs: living docs clean"
fi

exit "$status"
