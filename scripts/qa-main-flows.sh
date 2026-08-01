#!/usr/bin/env bash
# Run the role-aware Playwright main-flow QA against an isolated local build.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.
# Deliberately does not use `set -e`: several steps capture an exit status and
# print the matching log tail instead of dying silently.

set -u

repo=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d /tmp/hausv-main-flow-qa.XXXXXX) || exit 1
artifact_dir=${HV_QA_ARTIFACT_DIR:-}
if [ -n "$artifact_dir" ]; then
    mkdir -p "$artifact_dir" || exit 1
    artifact_dir=$(python3 -c 'import os, sys; print(os.path.abspath(sys.argv[1]))' "$artifact_dir")
    export HV_QA_ARTIFACT_DIR="$artifact_dir"
fi
log_dir="$tmp"
if [ -n "$artifact_dir" ]; then
    log_dir="$artifact_dir"
fi
build_log="$log_dir/build.log"
fake_ha_log="$log_dir/fake-ha.log"
app_log="$log_dir/app.log"
playwright_log="$log_dir/playwright.log"
structured_log="$log_dir/structured-log-check.log"
port=${HV_QA_PORT:-8121}
ha_port=${HV_QA_HA_PORT:-8122}
HAUSV_QA_TMP="$tmp"
HAUSV_QA_PID=""
HAUSV_QA_HA_PID=""
go_bin=${HV_GO:-}
if [ -z "$go_bin" ]; then
    go_bin=$(command -v go || true)
fi
if [ -z "$go_bin" ] && [ -d /nix/store ]; then
    required_go=$(awk '$1 == "go" { print $2; exit }' "$repo/go.mod")
    for candidate in /nix/store/*-go-"$required_go"/bin/go; do
        if [ -x "$candidate" ]; then
            go_bin="$candidate"
            break
        fi
    done
fi
if [ -z "$go_bin" ] || [ ! -x "$go_bin" ]; then
    echo "Go aus go.mod wurde nicht gefunden. Optional HV_GO auf das Go-Binary setzen." >&2
    exit 1
fi

cleanup_main_flow_qa() {
    if [ -n "${HAUSV_QA_PID:-}" ]; then
        kill "$HAUSV_QA_PID" 2>/dev/null
    fi
    if [ -n "${HAUSV_QA_HA_PID:-}" ]; then
        kill "$HAUSV_QA_HA_PID" 2>/dev/null
    fi
    if [ -n "${HAUSV_QA_TMP:-}" ] && [ -d "$HAUSV_QA_TMP" ]; then
        command rm -rf -- "$HAUSV_QA_TMP"
    fi
}
trap cleanup_main_flow_qa EXIT

if [ ! -d "$repo/scripts/snapshot/node_modules/playwright" ]; then
    echo "── installing pinned Playwright package (without browser download)"
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
        npm --prefix "$repo/scripts/snapshot" ci --ignore-scripts --no-audit --no-fund || exit 1
fi

export HV_PORT=$port
export HV_QA_HA_PORT=$ha_port
export HV_DATA="$tmp/data"
mkdir -p "$HV_DATA"
# shellcheck source=scripts/snapshot/env.sh
. "$repo/scripts/snapshot/env.sh"

for checked_port in "$port" "$ha_port"; do
    if curl -sS --max-time 1 "http://localhost:$checked_port/" >/dev/null 2>&1; then
        echo "Port $checked_port ist bereits belegt. Mit HV_QA_PORT/HV_QA_HA_PORT freie Ports wählen." >&2
        exit 1
    fi
done

echo "── starting deterministic read-only Home Assistant fixture on :$ha_port"
node "$repo/scripts/snapshot/fake-ha.mjs" "$ha_port" >"$fake_ha_log" 2>&1 &
HAUSV_QA_HA_PID=$!
# Aus der Job-Tabelle nehmen, sonst meldet bash beim Aufräumen "Terminated" auf
# stderr — im Protokoll eines Prüflaufs sieht das wie ein Fehler aus.
disown %% 2>/dev/null || true

ha_ready=0
for _ in $(seq 40); do
    if curl -sf "http://127.0.0.1:$ha_port/jhw22/api/states" >/dev/null 2>&1; then
        ha_ready=1
        break
    fi
    sleep 0.1
done
if [ "$ha_ready" -eq 0 ]; then
    echo "Home-Assistant-Fixture wurde nicht bereit:" >&2
    command tail -n 40 "$fake_ha_log" >&2
    exit 1
fi

echo "── building current worktree"
( cd "$repo" && "$go_bin" build -o "$tmp/hausv-org" ./cmd/hausv-org ) >"$build_log" 2>&1
build_status=$?
if [ "$build_status" -ne 0 ]; then
    command tail -n 80 "$build_log" >&2
    exit "$build_status"
fi

echo "── starting isolated portal on :$port"
# `exec` so $! is the portal itself and the trap can kill it.
( cd "$repo" && exec "$tmp/hausv-org" ) >"$app_log" 2>&1 &
HAUSV_QA_PID=$!
disown %% 2>/dev/null || true

ready=0
for _ in $(seq 60); do
    if curl -sf "http://localhost:$port/healthz" >/dev/null 2>&1; then
        ready=1
        break
    fi
    sleep 0.25
done
if [ "$ready" -eq 0 ]; then
    echo "Portal wurde nicht bereit:" >&2
    command tail -n 80 "$app_log" >&2
    exit 1
fi

if [ "${HV_QA_CI_CORE:-}" = true ]; then
    echo "── running Playwright resident/admin CI core"
else
    echo "── running full Playwright role flows"
fi
node "$repo/scripts/snapshot/qa-main-flows.mjs" "http://localhost:$port" 2>&1 | tee "$playwright_log"
qa_status=${PIPESTATUS[0]}
if [ "$qa_status" -ne 0 ]; then
    if [ -n "$artifact_dir" ]; then
        echo "  Fehlerartefakte: $artifact_dir" >&2
    fi
    exit "$qa_status"
fi
if [ "${HV_QA_LANDING_ONLY:-}" != true ]; then
    if ! command grep -q '^fake map tile served /map-tiles/' "$fake_ha_log"; then
        echo "Lokale Karten-Fixture wurde nicht verwendet; Browser-QA darf keine öffentliche Kachelquelle benötigen." >&2
        exit 1
    fi
    echo "  ✓ Kartenkacheln vollständig aus lokaler PNG-Fixture"
fi
if [ "${HV_QA_LANDING_ONLY:-}" = true ]; then
    echo "  ✓ Startseiten-QA vollständig"
elif [ "${HV_QA_CI_CORE:-}" = true ]; then
    echo "  ✓ Bewohner/Admin-CI-Kernlauf vollständig"
else
    echo "  ✓ Rollen-QA vollständig"
fi

echo "── checking structured logs"
python3 "$repo/scripts/check-structured-logs.py" < "$app_log" 2>&1 | tee "$structured_log"
log_status=${PIPESTATUS[0]}
if [ "$log_status" -ne 0 ]; then
    if [ -n "$artifact_dir" ]; then
        echo "  Fehlerartefakte: $artifact_dir" >&2
    fi
    exit "$log_status"
fi
