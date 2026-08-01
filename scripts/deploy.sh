#!/usr/bin/env bash
# Fail-closed manual production release (HAUSV-403).
#
#   scripts/deploy.sh [--dry-run]
#
# CI proves the exact commit on Blacksmith but never publishes an image.
# Production is built from `git archive HEAD` on csb1 after every precondition
# below has passed. A schema-changing release first creates a quiesced,
# transactionally consistent recovery point outside the live mount.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash. Deliberately does not
# use `set -e`: every step checks its own status and reports a rollback path.

set -u

fail_before_change() {
    echo "release refused: $1" >&2
    echo "rollback: not required; production was not changed." >&2
    exit 1
}

# POSIX single-quote escaping. csb1's SSH login shell is fish, which decodes
# the '\'' idiom identically to sh — verified by round-tripping quotes,
# backslashes, dollars, backticks, globs and newlines through `fish -c`.
shell_quote() {
    printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

visible_build() {
    local match version commit
    match=$(printf '%s' "$1" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+ \([0-9a-f]{7}\)' | head -1)
    if [ -n "$match" ]; then
        version=${match%% *}
        commit=${match#*\(}
        commit=${commit%\)}
        printf '%s\t%s\n' "$version" "$commit"
    fi
}

valid_health_payload() {
    printf '%s' "$1" | grep -qE '"service"[[:space:]]*:[[:space:]]*"hausv-org"' \
        && printf '%s' "$1" | grep -qE '"status"[[:space:]]*:[[:space:]]*"ok"'
}

remote_sh_command() {
    # Hand every compound remote operation to POSIX sh explicitly so `-e` and
    # `-u` cannot be parsed (or ignored) by the login shell.
    printf '/bin/sh -eu -c %s' "$(shell_quote "$1")"
}

printable_remote_ssh() {
    # $1 ssh_port, $2 ssh_host, $3 script
    printf 'ssh -p %s %s %s' "$1" "$2" "$(shell_quote "$(remote_sh_command "$3")")"
}

print_rollback() {
    # $1 schema_changed, $2 previous_tag, $3 ssh_host, $4 ssh_port, $5 image,
    # $6 compose_dir, $7 service, $8 snapshot_dir
    local image_script image_command containment_script
    image_script="docker tag $2 $5; cd $6; docker compose up -d --force-recreate --no-deps $7"
    image_command=$(printable_remote_ssh "$4" "$3" "$image_script")
    if [ "$1" = 1 ]; then
        containment_script="cd $6; docker compose stop -t 30 $7"
        echo "containment command: $(printable_remote_ssh "$4" "$3" "$containment_script")"
        echo "data/schema restore required: do NOT run an image-only rollback against a possibly migrated database."
        echo "restore source: $8 (root-only, pre-deploy SQLite + blobs)."
        # shellcheck disable=SC1111 # German typographic quotes, intentional
        echo "restore procedure: hausv-org docs/csb1-deploy.md → “Schema rollback procedure”; verify an isolated copy with system Python before replacing live data."
        echo "image command after the matching data restore: $image_command"
    else
        echo "image rollback command: $image_command"
        echo "data/schema restore: not required; this release contains no migration change."
    fi
}

fail_after_change() {
    # $1 message, then the eight print_rollback arguments
    local message=$1
    shift
    echo "release FAILED: $message" >&2
    print_rollback "$@" >&2
    exit 1
}

dry_run=0
for arg in ${@+"$@"}; do
    case $arg in
        --dry-run) dry_run=1 ;;
        *)
            echo "usage: scripts/deploy.sh [--dry-run]" >&2
            exit 1
            ;;
    esac
done

ssh_host=mba@cs1.barta.cm
ssh_port=2222
github_repo=inspr-at/hausv-org
workflow=CI
image=ghcr.io/markus-barta/hausv-org:latest
compose_dir=/home/mba/Code/nixcfg/hosts/csb1/docker
service=hausv-org
build_dir=/tmp/hausv-build
data_dir=/var/lib/csb1-docker/hausv-org
snapshot_root=/var/lib/csb1-docker/hausv-org-predeploy
live_url=https://jhw22.hausv.org/
health_url=https://jhw22.hausv.org/healthz
verify_attempts=15
verify_sleep=2
if [ -n "${HAUSV_DEPLOY_VERIFY_ATTEMPTS+x}" ]; then
    printf '%s' "$HAUSV_DEPLOY_VERIFY_ATTEMPTS" | grep -qE '^[1-9][0-9]*$' \
        || fail_before_change "HAUSV_DEPLOY_VERIFY_ATTEMPTS must be a positive integer"
    verify_attempts=$HAUSV_DEPLOY_VERIFY_ATTEMPTS
fi
if [ -n "${HAUSV_DEPLOY_VERIFY_SLEEP+x}" ]; then
    printf '%s' "$HAUSV_DEPLOY_VERIFY_SLEEP" | grep -qE '^[0-9]+$' \
        || fail_before_change "HAUSV_DEPLOY_VERIFY_SLEEP must be a non-negative integer"
    verify_sleep=$HAUSV_DEPLOY_VERIFY_SLEEP
fi

for required in git gh curl ssh; do
    command -v "$required" >/dev/null \
        || fail_before_change "required command '$required' is unavailable"
done

repo=$(git rev-parse --show-toplevel 2>/dev/null) \
    || fail_before_change "not inside the HAUSV repository"
cd "$repo" || fail_before_change "cannot enter repository root"

# The archive contains only HEAD. A dirty tree would make the operator believe
# uncommitted work was shipped even though it cannot be in the image.
dirty=$(git status --porcelain) \
    || fail_before_change "cannot inspect the working tree"
if [ -n "$dirty" ]; then
    echo "dirty paths:" >&2
    git status --short >&2
    fail_before_change "working tree is dirty"
fi

# Fetch, do not trust a stale remote-tracking ref.
git fetch --quiet origin main \
    || fail_before_change "cannot refresh origin/main"
head_sha=$(git rev-parse HEAD 2>/dev/null) \
    || fail_before_change "cannot resolve HEAD"
origin_sha=$(git rev-parse refs/remotes/origin/main 2>/dev/null) \
    || fail_before_change "cannot resolve origin/main"
if [ "$head_sha" != "$origin_sha" ]; then
    fail_before_change "HEAD $head_sha is not exactly origin/main $origin_sha"
fi
commit=$(git rev-parse --short=7 HEAD 2>/dev/null) \
    || fail_before_change "cannot resolve the visible commit"

app_version=$(tr -d '[:space:]' < VERSION)
printf '%s' "$app_version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$' \
    || fail_before_change "VERSION must be a non-empty semantic version"
snapshot_dir="$snapshot_root/$app_version-$commit"

# The required workflow at this exact commit must itself use Blacksmith for
# every job. Parse the jobs block instead of merely collecting runs-on lines:
# otherwise a reusable-workflow job without runs-on would silently escape the
# gate. Reusable jobs are rejected; their runner contract is outside this file.
workflow_source=$(git show HEAD:.github/workflows/ci.yml 2>/dev/null) \
    || fail_before_change "cannot read .github/workflows/ci.yml from HEAD"
in_jobs=0
job_count=0
current_job=""
current_has_runner=0
while IFS= read -r workflow_line; do
    if [ "$in_jobs" -eq 0 ]; then
        if printf '%s' "$workflow_line" | grep -qE '^jobs:[[:space:]]*$'; then
            in_jobs=1
        fi
        continue
    fi

    if printf '%s' "$workflow_line" | grep -qE '^[^[:space:]]'; then
        break
    fi

    if printf '%s' "$workflow_line" | grep -qE '^  [A-Za-z0-9_-]+:[[:space:]]*$'; then
        if [ -n "$current_job" ] && [ "$current_has_runner" -ne 1 ]; then
            fail_before_change "CI job '$current_job' has no explicit Blacksmith runner"
        fi
        current_job=$(printf '%s' "$workflow_line" | sed -E 's/^  ([A-Za-z0-9_-]+):[[:space:]]*$/\1/')
        current_has_runner=0
        job_count=$((job_count + 1))
        continue
    fi

    if [ -n "$current_job" ]; then
        if printf '%s' "$workflow_line" | grep -qE '^    uses:[[:space:]]*'; then
            fail_before_change "CI job '$current_job' uses a reusable workflow whose Blacksmith runners cannot be proven"
        fi
        if printf '%s' "$workflow_line" | grep -qE '^    runs-on:[[:space:]]*'; then
            printf '%s' "$workflow_line" | grep -qE '^    runs-on:[[:space:]]*blacksmith-[A-Za-z0-9_-]+[[:space:]]*(#.*)?$' \
                || fail_before_change "every required CI job must run on Blacksmith"
            current_has_runner=1
        fi
    fi
done <<EOF
$workflow_source
EOF
if [ "$job_count" -eq 0 ]; then
    fail_before_change "CI workflow contains no jobs"
fi
if [ -n "$current_job" ] && [ "$current_has_runner" -ne 1 ]; then
    fail_before_change "CI job '$current_job' has no explicit Blacksmith runner"
fi

ci_lines=$(gh run list \
    --repo "$github_repo" \
    --workflow "$workflow" \
    --branch main \
    --commit "$head_sha" \
    --event push \
    --limit 10 \
    --json databaseId,headSha,headBranch,status,conclusion,url \
    --jq '.[] | select(.headBranch == "main" and .status == "completed" and .conclusion == "success") | [.databaseId,.headSha,.url] | @tsv' 2>/dev/null) \
    || fail_before_change "cannot query the required GitHub Actions workflow"
if [ -z "$ci_lines" ]; then
    fail_before_change "no successful completed Blacksmith CI push run exists for exact HEAD $head_sha"
fi
ci_first_line=$(printf '%s\n' "$ci_lines" | head -1)
ci_run=$(printf '%s' "$ci_first_line" | cut -f1)
ci_sha=$(printf '%s' "$ci_first_line" | cut -f2)
ci_url=$(printf '%s' "$ci_first_line" | cut -f3)
if [ -z "$ci_url" ] || [ "$ci_sha" != "$head_sha" ]; then
    fail_before_change "the successful CI result does not belong to exact HEAD $head_sha"
fi

live_page=$(curl -fsS --max-time 10 "$live_url") \
    || fail_before_change "cannot read the current live build"
live_fields=$(visible_build "$live_page")
live_version=$(printf '%s' "$live_fields" | cut -f1)
live_commit=$(printf '%s' "$live_fields" | cut -f2)
if [ -z "$live_version" ] || [ -z "$live_commit" ]; then
    fail_before_change "the current live version and commit are not visible"
fi
if [ "$live_version" = "$app_version" ]; then
    fail_before_change "VERSION $app_version is already live; every production deployment requires a VERSION bump"
fi

live_health=$(curl -fsS --max-time 10 "$health_url") \
    || fail_before_change "the current production health endpoint is unavailable"
valid_health_payload "$live_health" \
    || fail_before_change "the current production health payload is not healthy"

live_sha=$(git rev-parse "$live_commit^{commit}" 2>/dev/null) \
    || fail_before_change "live commit $live_commit is not present in local git history"
case $live_sha in
    "$live_commit"*) ;;
    *) fail_before_change "live commit $live_commit is ambiguous" ;;
esac

schema_changed=0
git diff --quiet "$live_sha" HEAD -- internal/db/migrations
schema_diff_status=$?
case $schema_diff_status in
    0) ;;
    1) schema_changed=1 ;;
    *) fail_before_change "cannot determine whether database migrations changed" ;;
esac

previous_tag="ghcr.io/markus-barta/hausv-org:prev-$live_version-$live_commit"

# Read-only remote preflight. It verifies the currently healthy service, image,
# compose entry and (for schema changes) non-interactive root capability. The
# running container must be the exact image currently named by :latest. An
# existing rollback tag is accepted only when it already names that same image;
# this makes retries after a partial activation fail closed. Dry run stops here
# and never creates a snapshot or changes a container.
sudo_probe=""
if [ "$schema_changed" -eq 1 ]; then
    sudo_probe="sudo -n /run/current-system/sw/bin/python3 -c 'import sqlite3' >/dev/null;"
fi
preflight_script="\
    test \"\$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service)\" = healthy; \
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $service)\"; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\"; \
    test -n \"\$running_image_id\"; \
    test \"\$running_image_id\" = \"\$latest_image_id\"; \
    if docker image inspect $previous_tag >/dev/null 2>&1; then \
        previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\"; \
        test \"\$previous_image_id\" = \"\$running_image_id\"; \
    fi; \
    test -d $compose_dir; \
    cd $compose_dir; \
    docker compose config --services | grep -Fx $service >/dev/null; \
    $sudo_probe"
ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$preflight_script")" \
    || fail_before_change "remote preflight failed"

echo "release candidate: $app_version ($commit)"
echo "origin/main: $head_sha ✓"
echo "Blacksmith CI: run $ci_run for exact HEAD ✓"
echo "CI evidence: $ci_url"
echo "live now: $live_version ($live_commit), health ok"
if [ "$schema_changed" -eq 1 ]; then
    echo "schema change: yes — a fresh consistent snapshot is mandatory before container replacement"
else
    echo "schema change: no"
fi

if [ "$dry_run" -eq 1 ]; then
    echo "[dry-run] all fail-closed preconditions passed; production was not changed."
    if [ "$schema_changed" -eq 1 ]; then
        echo "[dry-run] actual release will publish $snapshot_dir atomically before starting the new image."
    fi
    exit 0
fi

if [ "$schema_changed" -eq 1 ]; then
    snapshot_script="\
        sudo -n /run/current-system/sw/bin/python3 - \
        --source $data_dir \
        --snapshot $snapshot_dir \
        --compose-dir $compose_dir \
        --service $service \
        --source-version $live_version \
        --source-commit $live_sha \
        --target-version $app_version \
        --target-commit $head_sha"
    if ! snapshot_proof=$(ssh -p "$ssh_port" "$ssh_host" \
        "$(remote_sh_command "$snapshot_script")" \
        < "$repo/scripts/create-predeploy-snapshot.py"); then
        current_health=$(ssh -p "$ssh_port" "$ssh_host" \
            "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service" 2>/dev/null | tr -d '[:space:]')
        if [ "$current_health" = healthy ]; then
            echo "release refused: fresh consistent pre-deploy snapshot failed." >&2
            echo "recovery verified: the current production service is healthy; no new image or schema was activated." >&2
        else
            echo "release FAILED: pre-deploy snapshot recovery did not return HAUSV to healthy state (container=$current_health)." >&2
            recovery_script="cd $compose_dir; docker compose up -d --force-recreate --no-deps $service"
            echo "mandatory recovery command: $(printable_remote_ssh "$ssh_port" "$ssh_host" "$recovery_script")" >&2
            echo "after recovery, require Docker health=healthy before any retry; do not activate the new image." >&2
        fi
        exit 1
    fi
    if ! { [[ $snapshot_proof == *"snapshot-path=$snapshot_dir"* ]] \
        && [[ $snapshot_proof == *"snapshot-integrity=ok"* ]] \
        && [[ $snapshot_proof == *"service-health=healthy"* ]] \
        && printf '%s' "$snapshot_proof" | grep -qE 'snapshot-created=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z'; }; then
        echo "release refused: snapshot command returned incomplete proof." >&2
        echo "recovery verified by the helper: the current production service is healthy; no new image or schema was activated." >&2
        exit 1
    fi
    echo "pre-deploy recovery point: fresh, consistent and healthy ✓"
fi

preserve_script="\
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $service)\"; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\"; \
    test -n \"\$running_image_id\"; \
    test \"\$running_image_id\" = \"\$latest_image_id\"; \
    if docker image inspect $previous_tag >/dev/null 2>&1; then \
        previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\"; \
        test \"\$previous_image_id\" = \"\$running_image_id\"; \
    else \
        docker tag \"\$running_image_id\" $previous_tag; \
    fi; \
    test \"\$(docker image inspect --format '{{.Id}}' $previous_tag)\" = \"\$running_image_id\""
ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$preserve_script")" \
    || fail_before_change "could not preserve the currently running image"
echo "preserved previous image: $previous_tag"

release_tag="ghcr.io/markus-barta/hausv-org:release-$app_version-$commit"
echo "building exact HEAD…"
export COPYFILE_DISABLE=1
build_script="\
    test \"$build_dir\" = /tmp/hausv-build; \
    rm -rf \"$build_dir\"; \
    mkdir -p $build_dir; \
    trap 'rm -rf \"$build_dir\"' EXIT; \
    cd $build_dir; \
    tar -x; \
    docker build --build-arg APP_VERSION=$app_version --build-arg GIT_COMMIT=$commit -t $release_tag .; \
    docker image inspect $release_tag >/dev/null"
git archive --format=tar HEAD \
    | ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$build_script")"
for command_status in "${PIPESTATUS[@]}"; do
    if [ "$command_status" -ne 0 ]; then
        fail_before_change "release image build failed; the live image and container were not changed"
    fi
done

echo "replacing the production container…"
activate_script="\
    docker tag $release_tag $image; \
    cd $compose_dir; \
    docker compose up -d --force-recreate --no-deps $service"
ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$activate_script")" \
    || fail_after_change "container replacement failed" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_dir" "$service" "$snapshot_dir"

post_ok=0
deployed=""
container_health=""
for _ in $(seq "$verify_attempts"); do
    container_health=$(ssh -p "$ssh_port" "$ssh_host" "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service" 2>/dev/null | tr -d '[:space:]')
    health_lines=$(curl -fsS --max-time 10 "$health_url" 2>/dev/null)
    page_text=$(curl -fsS --max-time 10 "$live_url" 2>/dev/null)
    deployed_fields=$(visible_build "$page_text")
    deployed_version=$(printf '%s' "$deployed_fields" | cut -f1)
    deployed_commit=$(printf '%s' "$deployed_fields" | cut -f2)
    if [ -n "$deployed_version" ] && [ -n "$deployed_commit" ]; then
        deployed="$deployed_version ($deployed_commit)"
    fi
    if [ "$container_health" = healthy ] \
        && valid_health_payload "$health_lines" \
        && [ "$deployed" = "$app_version ($commit)" ]; then
        post_ok=1
        break
    fi
    sleep "$verify_sleep"
done
if [ "$post_ok" -ne 1 ]; then
    fail_after_change "post-deploy health/version verification failed (container=$container_health, visible='$deployed')" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_dir" "$service" "$snapshot_dir"
fi

# Inspect only the current container's start window. Logs are evaluated on the
# host and never printed, avoiding accidental disclosure of operational data.
critical_pattern='application initialization failed|sqlite unavailable|migration .* failed|import to sqlite failed|not atomic|expired (audit|energy) data purge failed|panic|fatal'
startlog_script="\
    started=\"\$(docker inspect --format '{{.State.StartedAt}}' $service)\"; \
    log_file=\"\$(mktemp)\"; \
    trap 'rm -f \"\$log_file\"' EXIT; \
    docker logs $service --since \"\$started\" >\"\$log_file\" 2>&1; \
    grep -F '\"msg\":\"listening\"' \"\$log_file\" >/dev/null; \
    if grep -qiE '$critical_pattern' \"\$log_file\"; then exit 1; fi"
ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$startlog_script")" \
    || fail_after_change "critical start-log check failed or the listening marker is missing" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_dir" "$service" "$snapshot_dir"

echo "live version: $deployed ✓"
echo "container health: $container_health ✓"
echo "public health: ok ✓"
echo "critical start logs: clean ✓"
print_rollback "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_dir" "$service" "$snapshot_dir"
