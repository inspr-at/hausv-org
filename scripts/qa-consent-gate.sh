#!/usr/bin/env bash
# Boot an isolated instance with the Google Ads tag configured and run the
# consent-gate browser suite against it (HAUSV-742). Google's hosts are
# stubbed inside the browser, so nothing leaves the machine.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.
set -u
repo=$(git rev-parse --show-toplevel) || exit 1
tmp=$(mktemp -d "${TMPDIR:-/tmp}/hausv-consent-qa.XXXXXX") || exit 1
port=${HV_CONSENT_QA_PORT:-8131}
app_log="$tmp/app.log"
pid=""
cleanup() {
    if [ -n "$pid" ]; then
        kill "$pid" 2>/dev/null
        wait "$pid" 2>/dev/null || true
    fi
    command rm -rf -- "$tmp"
}
trap cleanup EXIT INT TERM

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
if [ ! -d "$repo/scripts/snapshot/node_modules/playwright" ]; then
    echo "── installing pinned Playwright package (without browser download)"
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
        npm --prefix "$repo/scripts/snapshot" ci --ignore-scripts --no-audit --no-fund || exit 1
fi

echo "── building isolated binary"
(cd "$repo" && HV_GO="$go_bin" bash scripts/build.sh -o "$tmp/hausv-org" ./cmd/hausv-org) || exit 1

echo "── starting isolated portal with the Ads tag on :$port"
(
    cd "$repo" && \
    ADDR=":$port" \
    DB_PATH="$tmp/boot.db" \
    PARKING_DATA_PATH="$tmp/parking.json" \
    BASE_URL="http://localhost:$port" \
    ROOT_DOMAIN="localhost" \
    TRUSTED_PROXY_CIDRS="127.0.0.1/32" \
    SESSION_KEY="abababababababababababababababababababababababababababababababab" \
    SMTP_HOST="" \
    OIDC_ISSUER="" \
    DEMO_LOGIN_ENABLED="true" \
    DEMO_LOGIN_ACCESS_CODE="musterstadt-2026" \
    GOOGLE_ADS_TAG_ID="AW-000000000" \
    GOOGLE_ADS_LEAD_CONVERSION="AW-000000000/qa-consent-gate" \
    exec "$tmp/hausv-org"
) >"$app_log" 2>&1 &
pid=$!
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
    command tail -n 60 "$app_log" >&2
    exit 1
fi

(cd "$repo/scripts/snapshot" && node qa-consent-gate.mjs "http://localhost:$port")
status=$?
if [ "$status" -ne 0 ]; then
    echo "── portal log tail" >&2
    command tail -n 40 "$app_log" >&2
fi
exit "$status"
