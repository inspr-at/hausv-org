#!/usr/bin/env bash
# Every supported local and CI build verifies the offline source closure first.
set -euo pipefail
repo=$(cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
"${HV_GO:-go}" run ./cmd/verify-release
release_version=$(cat VERSION)
release_commit=$(git rev-parse --short=7 HEAD 2>/dev/null || printf dev)
exec "${HV_GO:-go}" build -ldflags="-X github.com/inspr-at/hausv-org/internal/version.Version=$release_version -X github.com/inspr-at/hausv-org/internal/version.Commit=$release_commit" "$@"
