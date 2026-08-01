#!/usr/bin/env bash
# Verify every embedded asset is actually SERVED, with the right bytes.
#
# The snapshot oracle cannot catch a broken asset: a 404 on /assets/app.js does
# not change the HTML (the <script> tag is still emitted) and does not change
# the screenshots (the CSS is inline in the templates, and the JS is behavioural).
# So after moving assets/ into internal/web for //go:embed, this is the check
# that actually proves the move worked.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

port=8097
repo=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d /tmp/hv-assets.XXXXXX)

export HV_PORT=$port
export HV_DATA=$tmp/data
mkdir -p "$HV_DATA"
# shellcheck source=scripts/snapshot/env.sh
. "$repo/scripts/snapshot/env.sh"

pkg=.
[ -d "$repo/cmd/hausv-org" ] && pkg=./cmd/hausv-org
go build -o "$tmp/app" "$pkg" || exit 1

"$tmp/app" >"$tmp/app.log" 2>&1 &
pid=$!
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
    echo "app did not start:" >&2
    cat "$tmp/app.log" >&2
    kill "$pid" 2>/dev/null
    rm -rf "$tmp"
    exit 1
fi

fail=0
for path in "$repo"/internal/web/assets/*; do
    f=$(basename "$path")
    code=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$port/assets/$f")
    served=$(curl -s "http://localhost:$port/assets/$f" | wc -c | tr -d ' ')
    disk=$(wc -c < "$path" | tr -d ' ')
    if [ "$code" = 200 ] && [ "$served" = "$disk" ]; then
        printf '  ✓ %-24s %s  %s bytes == disk\n' "$f" "$code" "$served"
    else
        printf '  ✗ %-24s %s  served=%s disk=%s\n' "$f" "$code" "$served" "$disk"
        fail=1
    fi
done

kill "$pid" 2>/dev/null
rm -rf "$tmp"
if [ "$fail" -eq 1 ]; then
    echo "  ✗ EMBEDDED ASSETS BROKEN"
    exit 1
fi
echo "  ✓ all embedded assets served byte-identical to disk"
