#!/usr/bin/env bash
# Ship the HAUSV demo bundle to a remote Docker host and (re)start it.
#
#   HAUSV_DEMO_SSH_HOST=mba@<ip> [HAUSV_DEMO_SSH_PORT=2222] [HAUSV_DEMO_SSH_KEY=~/.ssh/agm_deploy]
#   HAUSV_DEMO_BASE_URL=https://hausv.agm.ng
#   [HAUSV_DEMO_SECRETS_FILE=/run/agenix/agm1-hausv-demo-env]   # absolute path ON THE HOST
#   [HAUSV_DEMO_PYTHON=python3]   # Python executable ON THE HOST, for the release manifest
#   [HAUSV_DEMO_REMOTE_DIR=/srv/hausv-demo] [HAUSV_DEMO_PORT=8098] [HAUSV_DEMO_SEED_ANCHOR=today]
#   [HAUSV_DEMO_SUBNET=172.30.98.0/24] [HAUSV_DEMO_TRUSTED_PROXY_CIDRS=<gateway>/32]
#   deploy/demo/deploy-remote.sh [--seed] [--dry-run]
#
# Mirrors scripts/deploy.sh in spirit, not in ceremony: HEAD is shipped via
# `git archive` (a dirty tree is refused), the image is built on the host with
# the demo version baked in, the compose service is recreated, and the health
# endpoint is checked. Releases are kept in <dir>/releases/<sha>; <dir>/src
# points at the live one, so a rollback is `ln -sfn` plus `up -d`.
#
# Demo only: uses the already reserved VERSION in a separate demo channel.
# The release coordinator must check CI first; this script does not query CI.
# Existing data is retained; --seed explicitly recreates the fixture data.
set -euo pipefail

usage() { sed -n '2,12p' "$0" >&2; exit 2; }

seed=0
dry_run=0
for arg in "$@"; do
    case $arg in
        --seed) seed=1 ;;
        --dry-run) dry_run=1 ;;
        *) usage ;;
    esac
done

need() { [ -n "${!1:-}" ] || { echo "$1 is required" >&2; exit 2; }; }
need HAUSV_DEMO_SSH_HOST
need HAUSV_DEMO_BASE_URL
ssh_host=$HAUSV_DEMO_SSH_HOST
ssh_port=${HAUSV_DEMO_SSH_PORT:-22}
remote_dir=${HAUSV_DEMO_REMOTE_DIR:-/srv/hausv-demo}
port=${HAUSV_DEMO_PORT:-8098}
base_url=${HAUSV_DEMO_BASE_URL%/}
secrets_file=${HAUSV_DEMO_SECRETS_FILE:-$remote_dir/secrets.env}
python_bin=${HAUSV_DEMO_PYTHON:-python3}
case $python_bin in *[!a-zA-Z0-9/_+.-]*) echo "invalid remote Python executable" >&2; exit 2 ;; esac
seed_anchor=${HAUSV_DEMO_SEED_ANCHOR:-}
# The reverse proxy reaches the container from the compose network gateway,
# the first host of the pinned subnet. A /24 is assumed for the derivation;
# pass HAUSV_DEMO_TRUSTED_PROXY_CIDRS explicitly for anything else.
subnet=${HAUSV_DEMO_SUBNET:-172.30.98.0/24}
trusted_proxies=${HAUSV_DEMO_TRUSTED_PROXY_CIDRS:-${subnet%.*}.1/32}
project=hausv-demo
case $base_url in
    https://*) root_domain=${base_url#https://} ;;
    http://*) root_domain=${base_url#http://} ;;
    *) echo "HAUSV_DEMO_BASE_URL must start with http:// or https://" >&2; exit 2 ;;
esac
root_domain=${root_domain%%/*}
root_domain=${root_domain%%:*}

ssh_args=(-p "$ssh_port")
if [ -n "${HAUSV_DEMO_SSH_KEY:-}" ]; then ssh_args+=(-i "$HAUSV_DEMO_SSH_KEY"); fi

repo=$(git rev-parse --show-toplevel)
cd "$repo"
if [ -n "$(git status --porcelain)" ]; then
    echo "refusing to deploy: working tree is dirty (the archive contains only HEAD)" >&2
    exit 1
fi
sha=$(git rev-parse --short=12 HEAD)
full_sha=$(git rev-parse HEAD)
version="$(tr -d '[:space:]' < VERSION)"
release_dir="$remote_dir/releases/$sha"

echo "release  $version ($sha)"
echo "host     $ssh_host:$ssh_port -> $release_dir"
echo "url      $base_url  (loopback :$port, secrets $secrets_file)"
echo "proxy    subnet $subnet, TRUSTED_PROXY_CIDRS=$trusted_proxies"
if [ "$dry_run" = 1 ]; then
    echo "dry run: nothing shipped"
    exit 0
fi

# The deploy user's login shell may be fish (NixOS hosts), so every remote
# command runs under bash explicitly: step 1 as a bash -c string (stdin
# carries the archive), step 2 as a script on stdin.
case $release_dir in *"'"*) echo "release dir must not contain a single quote" >&2; exit 2 ;; esac

# 1. Ship HEAD. The release directory is immutable once extracted.
git archive --format=tar HEAD | ssh "${ssh_args[@]}" "$ssh_host" \
    "bash -c 'set -e; mkdir -p \"$release_dir\"; tar -x -C \"$release_dir\"'"

# 2. Configure, build, start, seed, verify — one remote shell so the
#    compose invocation is identical for every step.
remote=$(cat <<REMOTE
set -euo pipefail
cd '$release_dir/deploy/demo'
if docker compose version >/dev/null 2>&1; then compose() { docker compose "\$@"; }; else compose() { docker-compose "\$@"; }; fi
[ -r '$secrets_file' ] || { echo "secrets file $secrets_file is missing or unreadable for \$(id -un)" >&2; exit 1; }
command -v '$python_bin' >/dev/null || { echo 'remote Python executable is unavailable; set HAUSV_DEMO_PYTHON'; exit 1; }
sed -e 's#^BASE_URL=.*#BASE_URL=$base_url#' -e 's#^ROOT_DOMAIN=.*#ROOT_DOMAIN=$root_domain#' -e 's#^TRUSTED_PROXY_CIDRS=.*#TRUSTED_PROXY_CIDRS=$trusted_proxies#' demo.env.example > demo.env
grep -q '^TRUSTED_PROXY_CIDRS=$trusted_proxies\$' demo.env || { echo 'demo.env.example lacks a TRUSTED_PROXY_CIDRS line' >&2; exit 1; }
export COMPOSE_PROJECT_NAME='$project' HAUSV_DEMO_VERSION='$version' HAUSV_DEMO_COMMIT='$sha' HAUSV_DEMO_PORT='$port' HAUSV_DEMO_SUBNET='$subnet'
export HAUSV_DEMO_SECRETS_FILE='$secrets_file' HAUSV_DEMO_IMAGE='hausv-demo:$sha'
mkdir -p '$remote_dir/release-records'
[ ! -e '$remote_dir/release-records/$version.json' ] || { echo 'release coordinate already frozen; reuse its exact image or reserve a new version'; exit 1; }
compose -p '$project' build
image_digest=\$(docker image ls --no-trunc --quiet 'hausv-demo:$sha')
'$python_bin' ../../scripts/release-manifest.py --channel demo --commit '$full_sha' --image 'hausv-demo:$sha' --image-digest "\$image_digest" --output '$remote_dir/release-records/$version.json'
ln -sfn '$release_dir' '$remote_dir/src'
compose -p '$project' up -d --remove-orphans
for i in \$(seq 1 30); do
    if curl -fsS "http://127.0.0.1:$port/healthz" 2>/dev/null | grep -q '"status":"ok"'; then break; fi
    [ "\$i" = 30 ] && { echo "healthz never answered ok" >&2; compose -p '$project' logs --tail 40; exit 1; }
    sleep 2
done
if [ '$seed' = 1 ]; then
    HAUSV_DEMO_SEED_ANCHOR='$seed_anchor' ./seed.sh
fi
echo "healthz: \$(curl -fsS http://127.0.0.1:$port/healthz)"
echo "live: $release_dir ($version)"
REMOTE
)
printf '%s\n' "$remote" | ssh "${ssh_args[@]}" "$ssh_host" bash -s

echo
echo "verify from here:"
echo "  HAUSV_DEMO_BASE_URL=$base_url DEMO_LOGIN_ACCESS_CODE=<code> deploy/demo/verify.sh"
echo "rollback on the host:"
echo "  ln -sfn $remote_dir/releases/<previous sha> $remote_dir/src && cd $remote_dir/src/deploy/demo && HAUSV_DEMO_SECRETS_FILE=$secrets_file HAUSV_DEMO_IMAGE=hausv-demo:<previous sha> docker-compose -p $project up -d"
echo "reseed on the host:"
echo "  cd $remote_dir/src/deploy/demo && COMPOSE_PROJECT_NAME=$project HAUSV_DEMO_SECRETS_FILE=$secrets_file HAUSV_DEMO_SEED_ANCHOR=today ./seed.sh"
