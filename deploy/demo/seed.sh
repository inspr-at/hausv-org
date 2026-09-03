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

# HAUSV_DEMO_SEED_ANCHOR: today | none | YYYY-MM-DD (default: the seed's own demo day).
anchor_args=()
if [ -n "${HAUSV_DEMO_SEED_ANCHOR:-}" ]; then anchor_args=(-anchor "$HAUSV_DEMO_SEED_ANCHOR"); fi
compose exec -T hausv-demo /hausv-org demo-seed -dir /seed -reset "${anchor_args[@]}"
compose exec -T hausv-demo /hausv-org demo-seed -dir /seed -stats
# The unit register is a JSON store the app loads once at boot (UNIT_DATA_PATH);
# the CLI rewrote the file, so the running app must reload it. The portal's own
# demo reset seeds in-process and does not need this.
compose restart hausv-demo
for i in $(seq 1 30); do
    if compose exec -T hausv-demo /hausv-org healthcheck >/dev/null 2>&1; then break; fi
    [ "$i" = 30 ] && { echo "hausv-demo did not become healthy after the restart" >&2; exit 1; }
    sleep 2
done
