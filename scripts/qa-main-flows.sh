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
public_auth_log="$log_dir/public-auth.log"
home_setup_log="$log_dir/home-setup.log"
fake_smtp_log="$log_dir/fake-smtp.log"
handover_log="$log_dir/handover.log"
document_log="$log_dir/document.log"
settings_empty_log="$log_dir/settings-parking-empty.log"
settings_populated_log="$log_dir/settings-parking-populated.log"
structured_log="$log_dir/structured-log-check.log"
port=${HV_QA_PORT:-8121}
ha_port=${HV_QA_HA_PORT:-8122}
smtp_port=${HV_QA_SMTP_PORT:-8123}
smtp_api_port=${HV_QA_SMTP_API_PORT:-8124}
HAUSV_QA_TMP="$tmp"
HAUSV_QA_PID=""
HAUSV_QA_HA_PID=""
HAUSV_QA_SMTP_PID=""
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

stop_portal() {
    if [ -n "${HAUSV_QA_PID:-}" ]; then
        kill "$HAUSV_QA_PID" 2>/dev/null
        wait "$HAUSV_QA_PID" 2>/dev/null || true
        HAUSV_QA_PID=""
    fi
}

cleanup_main_flow_qa() {
    stop_portal
    if [ -n "${HAUSV_QA_HA_PID:-}" ]; then
        kill "$HAUSV_QA_HA_PID" 2>/dev/null
    fi
    if [ -n "${HAUSV_QA_SMTP_PID:-}" ]; then
        kill "$HAUSV_QA_SMTP_PID" 2>/dev/null
    fi
    if [ -n "${HAUSV_QA_TMP:-}" ] && [ -d "$HAUSV_QA_TMP" ]; then
        command rm -rf -- "$HAUSV_QA_TMP"
    fi
}
trap cleanup_main_flow_qa EXIT

start_portal() {
    echo "── starting isolated portal on :$port"
    # Append so the structured-log gate covers both the empty and populated
    # parking-store processes used by this one deterministic run.
    ( cd "$repo" && exec "$tmp/hausv-org" ) >>"$app_log" 2>&1 &
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
}

restart_portal() {
    echo "── restarting isolated portal to reset in-memory QA limits"
    stop_portal
    start_portal
}

if [ ! -d "$repo/scripts/snapshot/node_modules/playwright" ]; then
    echo "── installing pinned Playwright package (without browser download)"
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
        npm --prefix "$repo/scripts/snapshot" ci --ignore-scripts --no-audit --no-fund || exit 1
fi

export HV_PORT=$port
export HV_QA_HA_PORT=$ha_port
export HV_QA_SMTP_PORT=$smtp_port
export HV_QA_SMTP_API="http://127.0.0.1:$smtp_api_port"
export HV_DATA="$tmp/data"
mkdir -p "$HV_DATA"
# shellcheck source=scripts/snapshot/env.sh
. "$repo/scripts/snapshot/env.sh"

for checked_port in "$port" "$ha_port" "$smtp_port" "$smtp_api_port"; do
    if curl -sS --max-time 1 "http://localhost:$checked_port/" >/dev/null 2>&1; then
        echo "Port $checked_port ist bereits belegt. Mit HV_QA_PORT/HV_QA_HA_PORT freie Ports wählen." >&2
        exit 1
    fi
done

echo "── starting deterministic local mail fixture on :$smtp_port"
node "$repo/scripts/snapshot/fake-smtp.mjs" "$smtp_port" "$smtp_api_port" >"$fake_smtp_log" 2>&1 &
HAUSV_QA_SMTP_PID=$!
disown %% 2>/dev/null || true

smtp_ready=0
for _ in $(seq 40); do
    if curl -sf "http://127.0.0.1:$smtp_api_port/healthz" >/dev/null 2>&1; then
        smtp_ready=1
        break
    fi
    sleep 0.1
done
if [ "$smtp_ready" -eq 0 ]; then
    echo "Mail-Fixture wurde nicht bereit:" >&2
    command tail -n 40 "$fake_smtp_log" >&2
    exit 1
fi

echo "── starting deterministic read-only Home Assistant fixture on :$ha_port"
node "$repo/scripts/snapshot/fake-ha.mjs" "$ha_port" >"$fake_ha_log" 2>&1 &
HAUSV_QA_HA_PID=$!
# Aus der Job-Tabelle nehmen, sonst meldet bash beim Aufräumen "Terminated" auf
# stderr — im Protokoll eines Prüflaufs sieht das wie ein Fehler aus.
disown %% 2>/dev/null || true

ha_ready=0
for _ in $(seq 40); do
    if curl -sf "http://127.0.0.1:$ha_port/demo/api/states" >/dev/null 2>&1; then
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
( cd "$repo" && HV_GO="$go_bin" bash scripts/build.sh -o "$tmp/hausv-org" ./cmd/hausv-org ) >"$build_log" 2>&1
build_status=$?
if [ "$build_status" -ne 0 ]; then
    command tail -n 80 "$build_log" >&2
    exit "$build_status"
fi

: >"$app_log"
start_portal

node "$repo/scripts/snapshot/qa-support-view.mjs" "http://localhost:$port" >"$log_dir/support-view.log" 2>&1
support_status=$?
if [ "$support_status" -ne 0 ]; then
    cat "$log_dir/support-view.log"
node "$repo/scripts/snapshot/qa-unified-context.mjs" "http://localhost:$port" >"$log_dir/unified-context.log" 2>&1
context_status=$?
cat "$log_dir/unified-context.log"
if [ "$context_status" -ne 0 ]; then exit "$context_status"; fi >&2
    exit "$support_status"
fi
cat "$log_dir/support-view.log"
node "$repo/scripts/snapshot/qa-unified-context.mjs" "http://localhost:$port" >"$log_dir/unified-context.log" 2>&1
context_status=$?
cat "$log_dir/unified-context.log"
if [ "$context_status" -ne 0 ]; then exit "$context_status"; fi

if [ "${HV_QA_ENERGY_ONLY:-}" = true ]; then
    echo "── running focused Playwright energy and breakpoint flows"
elif [ "${HV_QA_CI_CORE:-}" = true ]; then
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
    # Every specialist uses the same persistent fake dataset, but an
    # independent process. Production-like auth throttling therefore remains
    # enabled without one harness consuming the next harness's allowance.
    restart_portal
    echo "── running public landing/auth/map flows"
    node "$repo/scripts/snapshot/qa-public-auth.mjs" "http://localhost:$port" 2>&1 | tee "$public_auth_log"
    public_auth_status=${PIPESTATUS[0]}
    if [ "$public_auth_status" -ne 0 ]; then
        if [ -n "$artifact_dir" ]; then
            echo "  Fehlerartefakte: $artifact_dir" >&2
        fi
        exit "$public_auth_status"
    fi
fi

if [ "${HV_QA_LANDING_ONLY:-}" != true ]; then
    restart_portal
    handover_artifacts=""
    if [ -n "$artifact_dir" ]; then
        handover_artifacts="$artifact_dir/handover"
        mkdir -p "$handover_artifacts" || exit 1
    fi
    echo "── running complete handover lifecycle"
    HV_QA_ARTIFACT_DIR="$handover_artifacts" \
        node "$repo/scripts/snapshot/qa-handover-flow.mjs" "http://localhost:$port" 2>&1 | tee "$handover_log"
    handover_status=${PIPESTATUS[0]}
    if [ "$handover_status" -ne 0 ]; then
        if [ -n "$artifact_dir" ]; then
            echo "  Fehlerartefakte: $handover_artifacts" >&2
        fi
        exit "$handover_status"
    fi

    restart_portal
    document_artifacts=""
    if [ -n "$artifact_dir" ]; then
        document_artifacts="$artifact_dir/document"
        mkdir -p "$document_artifacts" || exit 1
    fi
    echo "── running complete document/e-invoice lifecycle"
    HV_QA_ARTIFACT_DIR="$document_artifacts" \
        node "$repo/scripts/snapshot/qa-document-flow.mjs" "http://localhost:$port" 2>&1 | tee "$document_log"
    document_status=${PIPESTATUS[0]}
    if [ "$document_status" -ne 0 ]; then
        if [ -n "$artifact_dir" ]; then
            echo "  Fehlerartefakte: $document_artifacts" >&2
        fi
        exit "$document_status"
    fi

    restart_portal
    echo "── running settings/parking empty-state lifecycle"
    HV_QA_PARKING_STATE=empty \
        node "$repo/scripts/snapshot/qa-settings-parking.mjs" "http://localhost:$port" 2>&1 | tee "$settings_empty_log"
    settings_empty_status=${PIPESTATUS[0]}
    if [ "$settings_empty_status" -ne 0 ]; then
        if [ -n "$artifact_dir" ]; then
            echo "  Fehlerartefakte: $artifact_dir/settings-parking-empty" >&2
        fi
        exit "$settings_empty_status"
    fi

    # ParkingStore is intentionally loaded once. Restart the same isolated
    # binary after writing the rolling two-month fixture so both an honest
    # empty state and a populated month lifecycle are proven in one gate.
    echo "── restarting portal with populated parking fixture"
    stop_portal
    node "$repo/scripts/snapshot/qa-settings-parking.mjs" \
        --write-populated-fixture "$PARKING_DATA_PATH" 2>&1 | tee -a "$settings_populated_log"
    settings_fixture_status=${PIPESTATUS[0]}
    if [ "$settings_fixture_status" -ne 0 ]; then
        exit "$settings_fixture_status"
    fi
    start_portal

    echo "── running settings/parking populated lifecycle"
    HV_QA_PARKING_STATE=populated \
        node "$repo/scripts/snapshot/qa-settings-parking.mjs" "http://localhost:$port" 2>&1 | tee -a "$settings_populated_log"
    settings_populated_status=${PIPESTATUS[0]}
    if [ "$settings_populated_status" -ne 0 ]; then
        if [ -n "$artifact_dir" ]; then
            echo "  Fehlerartefakte: $artifact_dir/settings-parking-populated" >&2
        fi
        exit "$settings_populated_status"
    fi
fi

# The existing role flows intentionally use the local login shortcut. Enable
# SMTP only for this final lifecycle so it must consume a real captured mail.
export SMTP_HOST=127.0.0.1
export SMTP_PORT=$smtp_port
restart_portal
home_setup_state="$tmp/home-setup-state.json"
echo "── running HAUSV Home setup from reservation through connector"
node "$repo/scripts/snapshot/qa-home-setup.mjs" "http://localhost:$port" create "$home_setup_state" 2>&1 | tee "$home_setup_log"
home_setup_status=${PIPESTATUS[0]}
if [ "$home_setup_status" -ne 0 ]; then
    exit "$home_setup_status"
fi
restart_portal
node "$repo/scripts/snapshot/qa-home-setup.mjs" "http://localhost:$port" verify "$home_setup_state" 2>&1 | tee -a "$home_setup_log"
home_setup_status=${PIPESTATUS[0]}
if [ "$home_setup_status" -ne 0 ]; then
    exit "$home_setup_status"
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
    echo "  ✓ Bewohner/Admin-CI-Kern plus Fachlebenszyklen vollständig"
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
