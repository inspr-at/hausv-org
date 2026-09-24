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
# shellcheck source=scripts/ephemeral-postgres.sh
. "$repo/scripts/ephemeral-postgres.sh" || exit 1

# Build. Note: after the cmd/ split this must target the main package's dir, so
# resolve it rather than assuming the repo root.
pkg=.
[ -d "$src/cmd/hausv-org" ] && pkg=./cmd/hausv-org

echo "── building $ref ($pkg)"
if ! ( cd "$src" && bash scripts/build.sh -o "$tmp/app" "$pkg" ); then
    echo "build failed" >&2
    exit 1
fi

# HV_QA_HA_PORT must be set BEFORE env.sh, because env.sh bakes it into
# HA_CONNECTORS_JSON and MAP_TILE_BASE_URL.
export HV_QA_HA_PORT=${HV_QA_HA_PORT:-$((port + 100))}

# shellcheck source=scripts/snapshot/env.sh
. "$repo/scripts/snapshot/env.sh"

capture=${HV_CAPTURE:-capture.mjs}
runtime=$src
if [ "$capture" = qa-annual-costs.mjs ]; then
    # Stored cost results require the complete demo accounting fixture.
    . "$repo/scripts/demo/env.sh" || exit 1
    runtime=$tmp
    ( cd "$runtime" && "$tmp/app" demo-seed -dir "$repo/scripts/demo/seed" -anchor none ) || exit 1
fi
if [ "$capture" = qa-legacy-routes.mjs ]; then
    # The existing fixture writer exits before launching Playwright. Month
    # details must contain real samples rather than silently testing a 404.
    node "$repo/scripts/snapshot/qa-settings-parking.mjs" --write-populated-fixture "$HV_DATA/parking.json" || exit 1
    runtime=$tmp
fi
if [ "$capture" = qa-inbox-suggest.mjs ]; then
    # The oracle owns the local provider's lifetime. A different default tenant
    # makes an unprefixed /app poll fail instead of silently hitting Demohaus.
    export DEFAULT_TENANT=haus-b
    export HV_QA_AI_PORT=${HV_QA_AI_PORT:-$((port + 200))}
    export AI_BASE_URL="http://127.0.0.1:$HV_QA_AI_PORT/v1"
    export AI_MODEL=qa-inbox-suggest
    export AI_API_KEY=qa-local-fixture
    export AI_TIMEOUT=40s
    export AI_MIN_CONFIDENCE=0.6
    export AI_PROVIDER_LABEL="QA lokal"
    # newApp loads .env.local from cwd; this oracle must never read a developer
    # environment file. Assets are embedded and fixture data paths are absolute.
    runtime=$tmp
fi

# env.sh points MAP_TILE_BASE_URL and HA_CONNECTORS_JSON at this fixture. Without
# it the sidebar map renders as a BROKEN IMAGE in every captured page — which
# looks exactly like a product bug and was once reported as one. qa-main-flows.sh
# has always started it; this harness never did.
if command -v node >/dev/null 2>&1; then
    node "$repo/scripts/snapshot/fake-ha.mjs" "$HV_QA_HA_PORT" >"$tmp/fake-ha.log" 2>&1 &
    ha_pid=$!
    disown %% 2>/dev/null || true
    for _ in $(seq 40); do
        curl -sf "http://127.0.0.1:$HV_QA_HA_PORT/map-tiles/0/0/0.png" -o /dev/null 2>/dev/null && break
        sleep 0.25
    done
else
    ha_pid=""
    echo "node missing; map tiles and HA fixtures will be unavailable" >&2
fi

echo "── booting on :$port"
# `exec` so $! is the app itself and the kill below reaches it.
( cd "$runtime" && exec "$tmp/app" ) >"$tmp/app.log" 2>&1 &
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
    [ -n "$ha_pid" ] && kill "$ha_pid" 2>/dev/null
    hausv_ephemeral_postgres_stop
    exit 1
fi

# Which script gets the booted app. Defaults to the snapshot capture; the
# responsive probe reuses this whole boot-with-seeded-fixtures dance rather than
# copying it and drifting from it.
echo "── running $capture"
node "$repo/scripts/snapshot/$capture" "http://localhost:$port" "$out"
rc=$?

kill "$pid" 2>/dev/null
[ -n "$ha_pid" ] && kill "$ha_pid" 2>/dev/null
hausv_ephemeral_postgres_stop
if [ "$ref" != WORKTREE ]; then
    git -C "$repo" worktree remove --force "$src" 2>/dev/null
fi
rm -rf "$tmp"
exit $rc
