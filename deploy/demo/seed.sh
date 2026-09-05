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
# The reset also cleared the mail ledger; a restarted mailbox holds its seed
# mails unread again, so they flow into the Posteingang on the next poll.
compose restart hausv-demo-mailbox >/dev/null 2>&1 || true
