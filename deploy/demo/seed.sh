#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

docker compose exec hausv-demo /hausv-org demo-seed -dir /seed -reset
docker compose exec hausv-demo /hausv-org demo-seed -dir /seed -stats
