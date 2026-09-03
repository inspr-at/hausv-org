#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

docker compose down --volumes --remove-orphans
docker compose up -d
"$script_dir/seed.sh"
