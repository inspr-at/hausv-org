#!/usr/bin/env fish
# Run the role-aware Playwright main-flow QA against an isolated local build.

set -l repo (git rev-parse --show-toplevel)
set -l tmp (mktemp -d /tmp/hausv-main-flow-qa.XXXXXX)
set -l port $HV_QA_PORT
test -n "$port"; or set port 8121
set -g HAUSV_QA_TMP "$tmp"
set -g HAUSV_QA_PID ""
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
set -gx HV_DATA "$tmp/data"
mkdir -p "$HV_DATA"
source "$repo/scripts/snapshot/env.fish"

if curl -sS --max-time 1 "http://localhost:$port/" >/dev/null 2>&1
    echo "Port $port ist bereits belegt. Mit HV_QA_PORT einen freien Port wählen." >&2
    exit 1
end

echo "── building current worktree"
env -C "$repo" "$go_bin" build -o "$tmp/hausv-org" ./cmd/hausv-org; or exit 1

echo "── starting isolated portal on :$port"
env -C "$repo" "$tmp/hausv-org" >"$tmp/app.log" 2>&1 &
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
    command tail -n 80 "$tmp/app.log" >&2
    exit 1
end

echo "── running Playwright role flows"
node "$repo/scripts/snapshot/qa-main-flows.mjs" "http://localhost:$port"; or exit 1
echo "  ✓ Rollen-QA vollständig"

echo "── checking structured logs"
python3 "$repo/scripts/check-structured-logs.py" < "$tmp/app.log"; or exit 1
