#!/usr/bin/env bash
# Does the templ rendering still offer everything the legacy rendering offered?
#
#   templ-parity.sh <git-ref|WORKTREE> [out-dir]
#
# Captures the SAME code twice — once with TEMPL_PORTAL_ENABLED off, once on —
# and diffs the interactive contract of every page against itself.
#
# This is the oracle Phase 3 was missing. The byte diff only ever runs with the
# switch off, so it never renders a single templ component; the reachability
# check only compares hrefs. Between them they certified a phase that had lost
# post-logout bfcache protection, delete confirmations, the mobile context
# switch and a whole settings script.
#
# Must run from the main checkout: a linked worktree has no node_modules, and the
# captures then die inside node in a way that reads exactly like agent failure.

set -u

ref=${1:-}
out=${2:-}
if [ -z "$ref" ]; then
    echo "usage: templ-parity.sh <git-ref|WORKTREE> [out-dir]" >&2
    exit 1
fi

repo=$(git rev-parse --show-toplevel)
if [ -z "$out" ]; then
    out=$(mktemp -d /tmp/hv-parity.XXXXXX) || exit 1
fi
mkdir -p "$out"

echo "══ legacy rendering (TEMPL_PORTAL_ENABLED unset)"
TEMPL_PORTAL_ENABLED= "$repo/scripts/snapshot/run.sh" "$ref" "$out/legacy" 8099 || exit 1

echo
echo "══ templ rendering (TEMPL_PORTAL_ENABLED=1)"
TEMPL_PORTAL_ENABLED=1 "$repo/scripts/snapshot/run.sh" "$ref" "$out/templ" 8199 || exit 1

echo
echo "══ contract diff"
node "$repo/scripts/snapshot/contract-diff.mjs" "$out/legacy" "$out/templ"
rc=$?
echo
echo "captures kept in $out"
exit $rc
