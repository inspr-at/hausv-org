#!/usr/bin/env bash
# Run HAUSV locally with deterministic fixture data and automatic rebuilds.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.
# env.sh exports the local fixture values before the binary starts, so they
# shadow any developer settings which the application later reads from .env.local.

set -u

repo=$(git rev-parse --show-toplevel) || exit 1
port=${HV_DEV_PORT:-${HV_PORT:-8098}}
ha_port=${HV_DEV_HA_PORT:-8102}
tmp=$(mktemp -d "${TMPDIR:-/tmp}/hausv-dev.XXXXXX") || exit 1
app="$tmp/hausv-org"
next_app="$tmp/hausv-org.next"
app_pid=""
templ_pid=""
ha_pid=""

cleanup() {
    trap - EXIT INT TERM
    if [ -n "$app_pid" ]; then
        kill "$app_pid" 2>/dev/null || true
        wait "$app_pid" 2>/dev/null || true
        app_pid=""
    fi
    if [ -n "$templ_pid" ]; then
        kill "$templ_pid" 2>/dev/null || true
        wait "$templ_pid" 2>/dev/null || true
        templ_pid=""
    fi
    if [ -n "$ha_pid" ]; then
        kill "$ha_pid" 2>/dev/null || true
        wait "$ha_pid" 2>/dev/null || true
        ha_pid=""
    fi
    command rm -rf -- "$tmp"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

stop_app() {
    if [ -n "$app_pid" ]; then
        kill "$app_pid" 2>/dev/null || true
        wait "$app_pid" 2>/dev/null || true
        app_pid=""
    fi
}

wait_for_app() {
    ready=0
    for _ in $(seq 60); do
        if curl -sf "http://127.0.0.1:$port/healthz" >/dev/null 2>&1; then
            ready=1
            break
        fi
        sleep 0.25
    done
    if [ "$ready" -eq 0 ]; then
        echo "Portal did not become healthy on :$port." >&2
        return 1
    fi
    return 0
}

build_and_start() {
    echo "── building"
    if ! ( builtin cd "$repo" && go build -o "$next_app" ./cmd/hausv-org ); then
        echo "Build failed; the previous portal keeps running." >&2
        command rm -f -- "$next_app"
        return 1
    fi

    stop_app
    mv "$next_app" "$app"
    echo "── starting fixture portal on http://localhost:$port"
    ( builtin cd "$repo" && exec "$app" ) &
    app_pid=$!
    if ! wait_for_app; then
        stop_app
        return 1
    fi
    echo "── ready; edit a .go or .templ file to rebuild (Ctrl-C stops and cleans up)"
    return 0
}

source_fingerprint() {
    # Include filenames as well as contents, so adds, removals, renames and
    # ordinary edits all cause a rebuild. cksum and find are available on the
    # stock macOS toolchain.
    find "$repo" -type f \( -name '*.go' -o -name '*.templ' \) -exec cksum {} + \
        | LC_ALL=C sort \
        | cksum
}

# Keep all data isolated from both the repository and a user's configuration.
export HV_PORT="$port"
export HV_QA_HA_PORT="$ha_port"
export HV_DATA="$tmp/data"
mkdir -p "$HV_DATA" || exit 1
if [ "${HV_DEV_FIXTURE:-}" = "demo" ]; then
    # shellcheck source=scripts/demo/env.sh
    . "$repo/scripts/demo/env.sh"
else
    # shellcheck source=scripts/snapshot/env.sh
    . "$repo/scripts/snapshot/env.sh"
fi

if command -v node >/dev/null 2>&1; then
    echo "── starting local Home Assistant fixture on :$ha_port"
    node "$repo/scripts/snapshot/fake-ha.mjs" "$ha_port" >"$tmp/fake-ha.log" 2>&1 &
    ha_pid=$!
else
    echo "Node is not installed; Home Assistant fixture requests will be unavailable." >&2
fi

if command -v templ >/dev/null 2>&1; then
    echo "── watching templ files"
    ( builtin cd "$repo" && exec templ generate --watch ) &
    templ_pid=$!
else
    echo "templ is not installed; .templ changes still rebuild, but generation is skipped." >&2
fi

build_and_start || true
last_fingerprint=$(source_fingerprint)
while :; do
    sleep 0.5
    next_fingerprint=$(source_fingerprint)
    if [ "$next_fingerprint" != "$last_fingerprint" ]; then
        # templ may write several generated files for one source edit. Let its
        # watch process finish that burst before compiling the application.
        sleep 0.5
        last_fingerprint=$(source_fingerprint)
        echo "── source change detected"
        build_and_start || true
    fi
done
