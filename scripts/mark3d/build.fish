#!/usr/bin/env fish
# Build the rotating 3D landing mark into internal/web/assets/hausv-mark-3d.js.
#
#   build.fish [--vendor]
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

set -l here (dirname (status --current-filename))
set -l repo (git -C $here rev-parse --show-toplevel)
set -l out $repo/internal/web/assets/hausv-mark-3d.js

# `builtin cd`, because an interactive fish config may shadow cd with zoxide,
# which does not exist inside a script.
builtin cd $here; or exit 1

if not test -d node_modules
    echo "==> installing build deps"
    npm install --no-audit --no-fund; or exit 1
end

if contains -- --vendor $argv
    echo "==> refetching Canvas UI AsciiObject"
    node vendor.mjs; or exit 1
end

echo "==> bundling"
npx esbuild src/hausvMark3d.ts \
    --bundle \
    --format=esm \
    --minify \
    --target=es2020 \
    --outfile=$out \
    --log-level=warning; or exit 1

set -l raw (stat -f %z $out)
set -l gz (gzip -c $out | wc -c | string trim)
printf "==> %s\n    %s KB raw, %s KB gzipped\n" $out (math "round($raw / 1024)") (math "round($gz / 1024)")
