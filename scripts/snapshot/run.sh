#!/usr/bin/env bash
# Snapshot one build of the app.
#
#   run.sh <git-ref|WORKTREE> <out-dir> [port]
#
# Builds the given ref in a throwaway worktree (or the current tree if given
# WORKTREE), boots it with the deterministic snapshot env against a freshly
# seeded data dir, captures every page, then tears everything down.
#
# The data dir is seeded IDENTICALLY per run, so two builds see the same state.
# Without that, the pages differ for reasons that have nothing to do with code.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

ref=${1:-}
out=${2:-}
port=${3:-8099}
if [ -z "$ref" ] || [ -z "$out" ]; then
    echo "usage: run.sh <git-ref|WORKTREE> <out-dir> [port]" >&2
    exit 1
fi

repo=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d /tmp/hv-snap.XXXXXX) || exit 1

if [ "$ref" = WORKTREE ]; then
    src=$repo
else
    src=$tmp/src
    git -C "$repo" worktree add -q --detach "$src" "$ref" || exit 1
fi

export HV_PORT=$port
export HV_DATA=$tmp/data
mkdir -p "$HV_DATA"

# Build. Note: after the cmd/ split this must target the main package's dir, so
# resolve it rather than assuming the repo root.
pkg=.
[ -d "$src/cmd/hausv-org" ] && pkg=./cmd/hausv-org

echo "── building $ref ($pkg)"
if ! ( cd "$src" && go build -o "$tmp/app" "$pkg" ); then
    echo "build failed" >&2
    exit 1
fi

# shellcheck source=scripts/snapshot/env.sh
. "$repo/scripts/snapshot/env.sh"

echo "── booting on :$port"
# `exec` so $! is the app itself and the kill below reaches it.
( cd "$src" && exec "$tmp/app" ) >"$tmp/app.log" 2>&1 &
pid=$!
disown %% 2>/dev/null || true

# Wait for readiness rather than sleeping blindly.
ready=0
for _ in $(seq 60); do
    if curl -sf "http://localhost:$port/healthz" >/dev/null 2>&1; then
        ready=1
        break
    fi
    sleep 0.25
done
if [ "$ready" -eq 0 ]; then
    echo "app did not become healthy; log:" >&2
    cat "$tmp/app.log" >&2
    kill "$pid" 2>/dev/null
    exit 1
fi

echo "── capturing"
node "$repo/scripts/snapshot/capture.mjs" "http://localhost:$port" "$out"
rc=$?

kill "$pid" 2>/dev/null
if [ "$ref" != WORKTREE ]; then
    git -C "$repo" worktree remove --force "$src" 2>/dev/null
fi
rm -rf "$tmp"
exit $rc
