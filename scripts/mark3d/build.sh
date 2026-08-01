#!/usr/bin/env bash
# Build the rotating 3D landing mark into internal/web/assets/hausv-mark-3d.js.
#
#   build.sh [--vendor]
#
# --vendor refetches the Canvas UI AsciiObject component from its registry and
# re-strips the React wrapper before bundling. Without it, the vendored copy in
# src/ is used as-is, so builds stay reproducible offline.
#
# The bundle is committed: internal/web/assets is //go:embed-ed into the binary,
# and the repo otherwise has no frontend build step. Rerun this only when the
# renderer or three.js needs updating.
#
# Output is ESM, NOT IIFE. three.js reads import.meta.url at module scope to
# resolve its Draco decoder; bundled as IIFE that becomes `new URL(path,
# undefined)`, which throws "Invalid URL" before any of our code runs.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)
out=$repo/internal/web/assets/hausv-mark-3d.js

cd "$here" || exit 1

if [ ! -d node_modules ]; then
    echo "==> installing build deps"
    npm install --no-audit --no-fund || exit 1
fi

case " ${*:-} " in
    *" --vendor "*)
        echo "==> refetching Canvas UI AsciiObject"
        node vendor.mjs || exit 1
        ;;
esac

echo "==> bundling"
npx esbuild src/hausvMark3d.ts \
    --bundle \
    --format=esm \
    --minify \
    --target=es2020 \
    --outfile="$out" \
    --log-level=warning || exit 1

# wc rather than `stat -f %z`: the BSD form is macOS-only and would break the
# moment this runs on a Linux runner.
raw=$(wc -c < "$out" | tr -d ' ')
gz=$(gzip -c "$out" | wc -c | tr -d ' ')
printf "==> %s\n    %s KB raw, %s KB gzipped\n" "$out" "$(( (raw + 512) / 1024 ))" "$(( (gz + 512) / 1024 ))"
