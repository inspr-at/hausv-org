#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"
# Compose derives the project name from this directory ("demo") unless told
# otherwise; a deployment that used -p must pass the same name here.
export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-${HAUSV_DEMO_PROJECT:-demo}}
# `docker compose` (plugin) or the standalone v2 binary, whichever the host has.
if docker compose version >/dev/null 2>&1; then
    compose() { docker compose "$@"; }
else
    compose() { docker-compose "$@"; }
fi

compose down --volumes --remove-orphans
compose up -d
"$script_dir/seed.sh"
