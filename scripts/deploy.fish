#!/usr/bin/env fish
# Deploy the committed tree to production (HAUSV-176).
#
#   deploy.fish [--dry-run] [--allow-same-version]
#
# Ships HEAD to csb1: pipes a git archive over SSH, builds the distroless image
# there with VERSION as APP_VERSION, then recreates the compose service. Prints
# a rollback command at the end.
#
# Why git archive and not tar: it ships exactly what is committed, and it cannot
# pick up macOS AppleDouble "._*" sidecars — those once landed in the build
# context and were embedded as junk SQL migrations.
#
# The compose service references `image:`, not `build:`, so the image only has
# to exist locally on the host under that tag. Nothing is pushed to ghcr, and CI
# never pushes either: deployment is entirely manual.

set -l dry_run 0
set -l allow_same 0
for arg in $argv
    switch $arg
        case --dry-run
            set dry_run 1
        case --allow-same-version
            set allow_same 1
        case '*'
            echo "usage: deploy.fish [--dry-run] [--allow-same-version]" >&2
            exit 1
    end
end

set -l ssh_host mba@cs1.barta.cm
set -l ssh_port 2222
set -l image ghcr.io/markus-barta/hausv-org:latest
set -l compose_dir /home/mba/Code/nixcfg/hosts/csb1/docker
set -l service hausv-org
set -l build_dir /tmp/hausv-build
set -l live_url https://jhw22.hausv.org/

set -l repo (git rev-parse --show-toplevel); or exit 1
# `builtin cd`: a user-defined cd function (e.g. one wrapping z) is not available
# in a non-interactive shell and would abort the script.
builtin cd $repo; or exit 1

# Deploy only what is committed: the image is built from `git archive HEAD`, so
# uncommitted work would silently NOT ship.
if test (count (git status --porcelain)) -gt 0
    echo "refusing to deploy: working tree is dirty (the image is built from HEAD)" >&2
    git status --short >&2
    exit 1
end

set -l app_version (string trim (cat VERSION))
set -l commit (git rev-parse --short HEAD)
test -n "$app_version"; or begin
    echo "refusing to deploy: VERSION is empty" >&2
    exit 1
end

# AGENTS.md: never deploy a changed product under an unchanged visible version.
set -l page (curl -sf --max-time 10 $live_url | string collect)
set -l live (string match -rg '([0-9]+\.[0-9]+\.[0-9]+) \([0-9a-f]{7}\)' -- $page | head -1)
if test -n "$live"
    echo "live now: $live — deploying: $app_version ($commit)"
    if test "$live" = "$app_version" -a $allow_same -eq 0
        echo "refusing to deploy: VERSION $app_version is already live; bump VERSION first (or pass --allow-same-version)" >&2
        exit 1
    end
else
    echo "could not read the live version (continuing) — deploying: $app_version ($commit)"
end

if test $dry_run -eq 1
    echo "[dry-run] would deploy $app_version ($commit) to $ssh_host:$ssh_port -> $image"
    exit 0
end

# Rollback point: the deploy overwrites :latest in place, so tag what is running
# before replacing it.
if test -n "$live"
    ssh -p $ssh_port $ssh_host "docker tag $image ghcr.io/markus-barta/hausv-org:prev-$live" >/dev/null
    and echo "tagged rollback image: prev-$live"
end

echo "building and deploying…"
git archive --format=tar HEAD | ssh -p $ssh_port $ssh_host "
    rm -rf $build_dir && mkdir -p $build_dir && cd $build_dir && tar -x \
    && docker build --build-arg APP_VERSION=$app_version --build-arg GIT_COMMIT=$commit -t $image . \
    && cd $compose_dir && docker compose up -d --no-deps $service \
    && rm -rf $build_dir"
or begin
    echo "deploy FAILED — the previous container is still running" >&2
    exit 1
end

echo ""
echo "verifying…"
ssh -p $ssh_port $ssh_host "docker ps --filter name=$service --format 'status: {{.Status}}'"
# Boot problems show up here: a store falling back to JSON, or a failed import.
ssh -p $ssh_port $ssh_host "docker logs $service --since 2m 2>&1 | grep -iE 'sqlite unavailable|import to sqlite failed|not atomic|panic|fatal' | head -5"

# The container needs a moment to pass its healthcheck and start serving; without
# waiting, this reads an empty page and reports a false failure.
set -l deployed ""
for attempt in (seq 15)
    set -l after (curl -sf --max-time 10 $live_url | string collect)
    set deployed (string match -rg '([0-9]+\.[0-9]+\.[0-9]+ \([0-9a-f]{7}\))' -- $after | head -1)
    if test "$deployed" = "$app_version ($commit)"
        break
    end
    sleep 2
end
if test "$deployed" = "$app_version ($commit)"
    echo "live version: $deployed ✓"
else
    echo "WARNING: live version reads '$deployed', expected '$app_version ($commit)' after 30s" >&2
end

echo ""
if test -n "$live"
    echo "rollback: ssh -p $ssh_port $ssh_host \"docker tag ghcr.io/markus-barta/hausv-org:prev-$live $image; cd $compose_dir; docker compose up -d --no-deps $service\""
end
