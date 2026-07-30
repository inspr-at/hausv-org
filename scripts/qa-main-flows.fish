#!/usr/bin/env fish
# Run the role-aware Playwright main-flow QA against an isolated local build.

set -l repo (git rev-parse --show-toplevel)
set -l tmp (mktemp -d /tmp/hausv-main-flow-qa.XXXXXX)
set -l artifact_dir "$HV_QA_ARTIFACT_DIR"
if test -n "$artifact_dir"
    mkdir -p "$artifact_dir"; or exit 1
    set artifact_dir (python3 -c 'import os, sys; print(os.path.abspath(sys.argv[1]))' "$artifact_dir")
    set -gx HV_QA_ARTIFACT_DIR "$artifact_dir"
end
set -l log_dir "$tmp"
if test -n "$artifact_dir"
    set log_dir "$artifact_dir"
end
set -l build_log "$log_dir/build.log"
set -l fake_ha_log "$log_dir/fake-ha.log"
set -l app_log "$log_dir/app.log"
set -l playwright_log "$log_dir/playwright.log"
set -l structured_log "$log_dir/structured-log-check.log"
set -l port $HV_QA_PORT
test -n "$port"; or set port 8121
set -l ha_port $HV_QA_HA_PORT
test -n "$ha_port"; or set ha_port 8122
set -g HAUSV_QA_TMP "$tmp"
set -g HAUSV_QA_PID ""
set -g HAUSV_QA_HA_PID ""
set -l go_bin $HV_GO
if test -z "$go_bin"
    set go_bin (command -s go)
end
if test -z "$go_bin" -a -d /nix/store
    set -l required_go (string match -rg '^go\s+(.+)$' < "$repo/go.mod")
    for candidate in /nix/store/*-go-$required_go/bin/go
        if test -x "$candidate"
            set go_bin "$candidate"
            break
        end
    end
end
if test -z "$go_bin" -o ! -x "$go_bin"
    echo "Go aus go.mod wurde nicht gefunden. Optional HV_GO auf das Go-Binary setzen." >&2
    exit 1
end

function cleanup_main_flow_qa --on-event fish_exit
    if test -n "$HAUSV_QA_PID"
        kill $HAUSV_QA_PID 2>/dev/null
    end
    if test -n "$HAUSV_QA_HA_PID"
        kill $HAUSV_QA_HA_PID 2>/dev/null
    end
    if test -n "$HAUSV_QA_TMP" -a -d "$HAUSV_QA_TMP"
        command rm -rf -- "$HAUSV_QA_TMP"
    end
end

if not test -d "$repo/scripts/snapshot/node_modules/playwright"
    echo "── installing pinned Playwright package (without browser download)"
    set -lx PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD 1
    npm --prefix "$repo/scripts/snapshot" ci --ignore-scripts --no-audit --no-fund; or exit 1
end

set -gx HV_PORT $port
set -gx HV_QA_HA_PORT $ha_port
set -gx HV_DATA "$tmp/data"
mkdir -p "$HV_DATA"
source "$repo/scripts/snapshot/env.fish"

for checked_port in $port $ha_port
    if curl -sS --max-time 1 "http://localhost:$checked_port/" >/dev/null 2>&1
        echo "Port $checked_port ist bereits belegt. Mit HV_QA_PORT/HV_QA_HA_PORT freie Ports wählen." >&2
        exit 1
    end
end

echo "── starting deterministic read-only Home Assistant fixture on :$ha_port"
node "$repo/scripts/snapshot/fake-ha.mjs" "$ha_port" >"$fake_ha_log" 2>&1 &
set -g HAUSV_QA_HA_PID $last_pid

set -l ha_ready 0
for i in (seq 40)
    if curl -sf "http://127.0.0.1:$ha_port/jhw22/api/states" >/dev/null 2>&1
        set ha_ready 1
        break
    end
    sleep 0.1
end
if test $ha_ready -eq 0
    echo "Home-Assistant-Fixture wurde nicht bereit:" >&2
    command tail -n 40 "$fake_ha_log" >&2
    exit 1
end

echo "── building current worktree"
env -C "$repo" "$go_bin" build -o "$tmp/hausv-org" ./cmd/hausv-org >"$build_log" 2>&1
set -l build_status $status
if test $build_status -ne 0
    command tail -n 80 "$build_log" >&2
    exit $build_status
end

echo "── starting isolated portal on :$port"
env -C "$repo" "$tmp/hausv-org" >"$app_log" 2>&1 &
set -g HAUSV_QA_PID $last_pid

set -l ready 0
for i in (seq 60)
    if curl -sf "http://localhost:$port/healthz" >/dev/null 2>&1
        set ready 1
        break
    end
    sleep 0.25
end
if test $ready -eq 0
    echo "Portal wurde nicht bereit:" >&2
    command tail -n 80 "$app_log" >&2
    exit 1
end

if test "$HV_QA_CI_CORE" = true
    echo "── running Playwright resident/admin CI core"
else
    echo "── running full Playwright role flows"
end
node "$repo/scripts/snapshot/qa-main-flows.mjs" "http://localhost:$port" 2>&1 | tee "$playwright_log"
set -l qa_status $pipestatus[1]
if test $qa_status -ne 0
    if test -n "$artifact_dir"
        echo "  Fehlerartefakte: $artifact_dir" >&2
    end
    exit $qa_status
end
if test "$HV_QA_LANDING_ONLY" != true
    if not command grep -q '^fake map tile served /map-tiles/' "$fake_ha_log"
        echo "Lokale Karten-Fixture wurde nicht verwendet; Browser-QA darf keine öffentliche Kachelquelle benötigen." >&2
        exit 1
    end
    echo "  ✓ Kartenkacheln vollständig aus lokaler PNG-Fixture"
end
if test "$HV_QA_LANDING_ONLY" = true
    echo "  ✓ Startseiten-QA vollständig"
else if test "$HV_QA_CI_CORE" = true
    echo "  ✓ Bewohner/Admin-CI-Kernlauf vollständig"
else
    echo "  ✓ Rollen-QA vollständig"
end

echo "── checking structured logs"
python3 "$repo/scripts/check-structured-logs.py" < "$app_log" 2>&1 | tee "$structured_log"
set -l log_status $pipestatus[1]
if test $log_status -ne 0
    if test -n "$artifact_dir"
        echo "  Fehlerartefakte: $artifact_dir" >&2
    end
    exit $log_status
end
