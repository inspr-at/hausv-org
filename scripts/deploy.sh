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

locked_remote_script() {
    # Hold the csb1 project lock across the complete transition, not merely
    # one compose subprocess. This keeps retag + recreate and
    # stop + snapshot + restart atomic against declarative reconcile jobs.
    printf '%s -w 300 %s /bin/sh -eu -c %s' \
        "$flock_bin" "$compose_lock" "$(shell_quote "$1")"
}

activation_locked_remote_script() {
    # Activation needs a unique lock-conflict status. Validation and every
    # post-retag operation use different statuses below, so the caller never
    # recommends a rollback for a transition that did not acquire the lock.
    printf '%s -E 41 -w 300 %s /bin/sh -eu -c %s' \
        "$flock_bin" "$compose_lock" "$(shell_quote "$1")"
}

printable_remote_ssh() {
    # $1 ssh_port, $2 ssh_host, $3 script
    printf 'ssh -p %s %s %s' "$1" "$2" "$(shell_quote "$(remote_sh_command "$3")")"
}

printable_locked_recovery_ssh() {
    # Schema recovery is an attended, multi-step operation. The interactive
    # shell keeps the same project lock from containment through data restore
    # and image recreation, so reconcile cannot restart the old image midway.
    printf 'ssh -tt -p %s %s %s' "$1" "$2" \
        "$(shell_quote "$flock_bin -w 300 $compose_lock /bin/sh -eu")"
}

print_rollback() {
    # $1 schema_changed, $2 previous_tag, $3 ssh_host, $4 ssh_port, $5 image,
    # $6 compose_command, $7 service, $8 snapshot_dir
    local image_body image_script image_command containment_body
    image_body="docker tag $2 $5; $6 up -d --force-recreate --no-deps $7"
    if [ "$1" = 1 ]; then
        containment_body="$6 stop -t 30 $7"
        echo "data/schema restore required: do NOT run an image-only rollback against a possibly migrated database."
        echo "restore source: $8 (root-only, pre-deploy SQLite + blobs)."
        echo "locked schema recovery shell: $(printable_locked_recovery_ssh "$4" "$3")"
        echo "inside that same locked shell, containment command: $containment_body"
        # shellcheck disable=SC1111 # German typographic quotes, intentional
        echo "restore procedure: hausv-org docs/csb1-deploy.md → “Schema rollback procedure”; keep the locked shell open through verification, data replacement and recreation."
        echo "inside that same locked shell after the matching data restore: $image_body"
    else
        image_script=$(locked_remote_script "$image_body")
        image_command=$(printable_remote_ssh "$4" "$3" "$image_script")
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
compose_file=/etc/compose/csb1/docker-compose.yml
compose_project=csb1
compose_lock=/run/lock/compose-csb1.lock
compose_lock_dir=/run/lock
flock_bin=/run/current-system/sw/bin/flock
compose_command="docker compose --project-directory $compose_dir -p $compose_project -f $compose_file"
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
preflight_body="\
    locked_live_page=\"\$(curl -fsS --max-time 10 $live_url)\"; \
    locked_open=\"\$(printf \"\\050\")\"; \
    locked_close=\"\$(printf \"\\051\")\"; \
    locked_build_marker=\"$live_version \${locked_open}$live_commit\${locked_close}\"; \
    printf \"%s\" \"\$locked_live_page\" | grep -F \"\$locked_build_marker\" >/dev/null; \
    test \"\$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service)\" = healthy; \
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $service)\"; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\"; \
    compose_project_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.project\"}}' $service)\"; \
    compose_service_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.service\"}}' $service)\"; \
    test -n \"\$running_image_id\"; \
    test \"\$running_image_id\" = \"\$latest_image_id\"; \
    test \"\$compose_project_label\" = $compose_project; \
    test \"\$compose_service_label\" = $service; \
    if docker image inspect $previous_tag >/dev/null 2>&1; then \
        previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\"; \
        test \"\$previous_image_id\" = \"\$running_image_id\"; \
    fi; \
    test -x $flock_bin; \
    test -d $compose_lock_dir; \
    test -d $compose_dir; \
    test -f $compose_file; \
    compose_services=\"\$($compose_command config --services)\"; \
    printf '%s\n' \"\$compose_services\" | grep -Fx $service >/dev/null; \
    $sudo_probe"
preflight_body="$preflight_body \
    printf 'running-image-id=%s\n' \"\$running_image_id\""
preflight_script=$(locked_remote_script "$preflight_body")
preflight_proof=$(ssh -p "$ssh_port" "$ssh_host" \
    "$(remote_sh_command "$preflight_script")") \
    || fail_before_change "remote preflight failed"
expected_running_image_id=${preflight_proof#running-image-id=}
if [ "$preflight_proof" != "running-image-id=$expected_running_image_id" ] \
    || [ "${#expected_running_image_id}" -ne 71 ] \
    || ! printf '%s' "$expected_running_image_id" \
        | grep -qE '^sha256:[0-9a-f]{64}$'; then
    fail_before_change "remote preflight returned invalid image identity proof"
fi

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
    snapshot_body="\
        test \"\$(docker inspect --format '{{.Image}}' $service)\" = $expected_running_image_id; \
        test \"\$(docker image inspect --format '{{.Id}}' $image)\" = $expected_running_image_id; \
        sudo -n /run/current-system/sw/bin/python3 - \
        --source $data_dir \
        --snapshot $snapshot_dir \
        --compose-dir $compose_dir \
        --compose-file $compose_file \
        --compose-project $compose_project \
        --service $service \
        --source-version $live_version \
        --source-commit $live_sha \
        --target-version $app_version \
        --target-commit $head_sha"
    snapshot_script=$(locked_remote_script "$snapshot_body")
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
            recovery_body="$compose_command up -d --force-recreate --no-deps $service"
            recovery_script=$(locked_remote_script "$recovery_body")
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

preserve_body="\
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $service)\"; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\"; \
    test -n \"\$running_image_id\"; \
    test \"\$running_image_id\" = $expected_running_image_id; \
    test \"\$latest_image_id\" = $expected_running_image_id; \
    test \"\$running_image_id\" = \"\$latest_image_id\"; \
    if docker image inspect $previous_tag >/dev/null 2>&1; then \
        previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\"; \
        test \"\$previous_image_id\" = \"\$running_image_id\"; \
    else \
        docker tag \"\$running_image_id\" $previous_tag; \
    fi; \
    test \"\$(docker image inspect --format '{{.Id}}' $previous_tag)\" = \"\$running_image_id\""
preserve_script=$(locked_remote_script "$preserve_body")
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
activate_body="\
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $service)\" || exit 42; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\" || exit 42; \
    previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\" || exit 42; \
    release_image_id=\"\$(docker image inspect --format '{{.Id}}' $release_tag)\" || exit 42; \
    compose_project_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.project\"}}' $service)\" || exit 42; \
    compose_service_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.service\"}}' $service)\" || exit 42; \
    if [ -z \"\$running_image_id\" ] \
        || [ -z \"\$release_image_id\" ] \
        || [ \"\$running_image_id\" != $expected_running_image_id ] \
        || [ \"\$latest_image_id\" != $expected_running_image_id ] \
        || [ \"\$previous_image_id\" != $expected_running_image_id ] \
        || [ \"\$running_image_id\" != \"\$latest_image_id\" ] \
        || [ \"\$running_image_id\" != \"\$previous_image_id\" ] \
        || [ \"\$compose_project_label\" != $compose_project ] \
        || [ \"\$compose_service_label\" != $service ]; then \
        exit 42; \
    fi; \
    docker tag $release_tag $image || exit 43; \
    test \"\$(docker image inspect --format '{{.Id}}' $image)\" = \"\$release_image_id\" || exit 43; \
    $compose_command up -d --force-recreate --no-deps $service || exit 43"
activate_script=$(activation_locked_remote_script "$activate_body")
ssh -p "$ssh_port" "$ssh_host" "$(remote_sh_command "$activate_script")"
activation_status=$?
case $activation_status in
    0) ;;
    41) fail_before_change "activation lock was unavailable before retag; production was not changed by this release" ;;
    42) fail_before_change "activation identity changed before retag; production was not changed by this release" ;;
    *) fail_after_change "container replacement failed" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_command" "$service" "$snapshot_dir" ;;
esac

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
    fail_after_change "post-deploy health/version verification failed (container=$container_health, visible='$deployed')" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_command" "$service" "$snapshot_dir"
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
    || fail_after_change "critical start-log check failed or the listening marker is missing" "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_command" "$service" "$snapshot_dir"

echo "live version: $deployed ✓"
echo "container health: $container_health ✓"
echo "public health: ok ✓"
echo "critical start logs: clean ✓"
print_rollback "$schema_changed" "$previous_tag" "$ssh_host" "$ssh_port" "$image" "$compose_command" "$service" "$snapshot_dir"
