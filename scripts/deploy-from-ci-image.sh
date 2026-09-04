#!/usr/bin/env bash
# Deploy HAUSV from CI-built GHCR image on csb1.
#
# This script runs LOCALLY on csb1 (no SSH, no git archive) and pulls the image
# that Blacksmith CI already built and pushed to GHCR. It is the pull+swap half
# of the full deploy.sh workflow, intended for triggering from GitHub Actions
# workflow_dispatch or manual csb1 execution.
#
# Usage:
#   scripts/deploy-from-ci-image.sh <version> <commit-sha> [--dry-run]
#
# Example:
#   scripts/deploy-from-ci-image.sh 0.98.10 a1b2c3d
#
# Prerequisites:
#   - Must run on csb1 with the same deploy.env that deploy.sh uses
#   - CI must have already pushed ghcr.io/inspr-at/hausv-org:release-<version>-<commit>
#   - The exact commit must have a green Blacksmith CI run
#
# Safety gates:
#   - Verifies the image exists in GHCR before pulling
#   - Preserves the previous image as a rollback target
#   - Validates health after container swap
#   - Fails before change if preconditions are not met
#
# Exit codes:
#   0  released, live version verified
#   3  nothing to release: this VERSION is already live, production untouched
#   1  refused or failed; the output names the rollback state

set -u

fail_before_change() {
    echo "release refused: $1" >&2
    echo "rollback: not required; production was not changed." >&2
    exit 1
}

# Every push to main triggers this path, and most pushes carry no VERSION bump.
# That is not a failed release, it is nothing to release: exit 3 lets the caller
# end neutrally instead of raising an alarm nobody can tell from a real one.
# Reached only after the live build was read and parsed, so a live check that
# cannot answer still refuses through fail_before_change above.
nothing_to_release() {
    echo "nothing to release: $1"
    echo "production keeps its current image."
    exit 3
}

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

encode_locked_body() {
    local raw encoded
    raw=$(printf '%s' "$1" | base64) || return 1
    encoded=${raw//$'\n'/}
    encoded=${encoded//$'\r'/}
    [ -n "$encoded" ] || return 1
    case $encoded in
        *[!A-Za-z0-9+/=]*) return 1 ;;
    esac
    printf '%s' "$encoded"
}

locked_remote_script() {
    local encoded
    encoded=$(encode_locked_body "$1") || return 1
    printf 'locked_script=$(%s /tmp/hausv-locked.XXXXXX); trap "rm -f $locked_script" EXIT HUP INT TERM; printf %%s %s | %s -d >"$locked_script"; %s -w 300 %s /bin/sh -eu "$locked_script"' \
        "$mktemp_bin" "$encoded" "$base64_bin" "$flock_bin" "$compose_lock"
}

activation_locked_remote_script() {
    local encoded
    encoded=$(encode_locked_body "$1") || return 1
    printf 'locked_script=$(%s /tmp/hausv-locked.XXXXXX) || exit 40; trap "rm -f $locked_script" EXIT HUP INT TERM; printf %%s %s | %s -d >"$locked_script" || exit 40; %s -E 41 -w 300 %s /bin/sh -eu "$locked_script"' \
        "$mktemp_bin" "$encoded" "$base64_bin" "$flock_bin" "$compose_lock"
}

print_rollback() {
    # $1 schema_changed, $2 previous_tag, $3 image, $4 compose_command, $5 service, $6 snapshot_dir
    local image_body
    image_body="docker tag $2 $3 && $4 up -d --force-recreate --no-deps $5"
    if [ "$1" = 1 ]; then
        echo "data/schema restore required: do NOT run an image-only rollback against a possibly migrated database." >&2
        echo "restore source: $6 (root-only, pre-deploy SQLite + blobs)." >&2
        echo "locked recovery shell: ssh to the runner host and run: $flock_bin -w 300 $compose_lock /bin/sh -eu" >&2
        echo "inside that same locked shell, containment command: $4 stop -t 30 $5" >&2
        echo "restore procedure: hausv-org docs/production-deploy.md → \"Schema rollback procedure\"; keep the locked shell open through verification, data replacement and recreation." >&2
        echo "inside that same locked shell after the matching data restore: $image_body" >&2
    else
        echo "image rollback command (must run under the project lock): $image_body"
        echo "data/schema restore: not required; this release contains no migration change."
    fi
}

fail_after_change() {
    # $1 message, $2 schema_changed, $3 previous_tag, $4 image, $5 compose_command, $6 service, $7 snapshot_dir
    local message=$1
    shift
    echo "release FAILED: $message" >&2
    print_rollback "$@" >&2
    exit 1
}

# Parse arguments
if [ $# -lt 2 ] || [ $# -gt 3 ]; then
    echo "usage: scripts/deploy-from-ci-image.sh <version> <commit-sha> [--dry-run]" >&2
    exit 1
fi

app_version="$1"
commit="$2"
dry_run=0
if [ $# -eq 3 ]; then
    case $3 in
        --dry-run) dry_run=1 ;;
        *)
            echo "usage: scripts/deploy-from-ci-image.sh <version> <commit-sha> [--dry-run]" >&2
            exit 1
            ;;
    esac
fi

printf '%s' "$app_version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$' \
    || fail_before_change "version must be a semantic version (e.g. 0.98.10)"

printf '%s' "$commit" | grep -qE '^[0-9a-f]{7}$' \
    || fail_before_change "commit must be a 7-character hex SHA (e.g. a1b2c3d)"

required_deploy_env() {
    local name=$1 value
    eval "value=\${$name-}"
    [ -n "$value" ] || fail_before_change "$name is required; load a private deployment environment file"
    printf '%s' "$value"
}

image=${HAUSV_DEPLOY_IMAGE:-ghcr.io/inspr-at/hausv-org:latest}
case $image in *:latest) ;; *) fail_before_change "HAUSV_DEPLOY_IMAGE must end in :latest" ;; esac
image_repo=${image%:latest}
compose_dir=$(required_deploy_env HAUSV_DEPLOY_COMPOSE_DIR)
compose_file=$(required_deploy_env HAUSV_DEPLOY_COMPOSE_FILE)
compose_project=${HAUSV_DEPLOY_COMPOSE_PROJECT:-hausv}
compose_lock=${HAUSV_DEPLOY_COMPOSE_LOCK:-/run/lock/hausv-compose.lock}
compose_lock_dir=${compose_lock%/*}
flock_bin=${HAUSV_DEPLOY_FLOCK_BIN:-/usr/bin/flock}
base64_bin=${HAUSV_DEPLOY_BASE64_BIN:-/usr/bin/base64}
mktemp_bin=${HAUSV_DEPLOY_MKTEMP_BIN:-/usr/bin/mktemp}
compose_command="docker compose --project-directory $compose_dir -p $compose_project -f $compose_file"
service=hausv-org
container=${HAUSV_DEPLOY_CONTAINER:-$service}
data_dir=""
snapshot_root=""
snapshot_dir=""
case $container in
    ""|*[!A-Za-z0-9_.-]*) fail_before_change "HAUSV_DEPLOY_CONTAINER must be a valid Docker container name" ;;
esac
live_url=$(required_deploy_env HAUSV_DEPLOY_LIVE_URL)
health_url=${HAUSV_DEPLOY_HEALTH_URL:-${live_url%/}/healthz}
verify_attempts=15
verify_sleep=2
snapshot_verify_attempts=50
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
if [ -n "${HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS+x}" ]; then
    printf '%s' "$HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS" | grep -qE '^[1-9][0-9]*$' \
        || fail_before_change "HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS must be a positive integer"
    snapshot_verify_attempts=$HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS
fi

for required in docker curl git; do
    command -v "$required" >/dev/null \
        || fail_before_change "required command '$required' is unavailable"
done

# This script must run from a git checkout to verify migrations.
repo=$(git rev-parse --show-toplevel 2>/dev/null) \
    || fail_before_change "not inside the HAUSV repository; the runner must check out hausv-org at the green SHA"
cd "$repo" || fail_before_change "cannot enter repository root"

live_page=$(curl -fsS --max-time 10 "$live_url") \
    || fail_before_change "cannot read the current live build"
live_fields=$(visible_build "$live_page")
live_version=$(printf '%s' "$live_fields" | cut -f1)
live_commit=$(printf '%s' "$live_fields" | cut -f2)
if [ -z "$live_version" ] || [ -z "$live_commit" ]; then
    fail_before_change "the current live version and commit are not visible"
fi
if [ "$live_version" = "$app_version" ]; then
    nothing_to_release "VERSION $app_version is already live; a release requires a VERSION bump"
fi

live_health=$(curl -fsS --max-time 10 "$health_url") \
    || fail_before_change "the current production health endpoint is unavailable"
valid_health_payload "$live_health" \
    || fail_before_change "the current production health payload is not healthy"

# Resolve live commit to full SHA for migration check. Fail closed if git
# cannot resolve the live commit: without it we cannot prove migrations
# did not change.
live_sha=$(git rev-parse "$live_commit^{commit}" 2>/dev/null) \
    || fail_before_change "live commit $live_commit is not present in local git history; the runner must check out hausv-org at the green SHA"
case $live_sha in
    "$live_commit"*) ;;
    *) fail_before_change "live commit $live_commit is ambiguous" ;;
esac

# Resolve HEAD for snapshot metadata.
head_sha=$(git rev-parse HEAD 2>/dev/null) \
    || fail_before_change "cannot resolve HEAD"

# Check whether migrations changed. When migrations changed, deployment requires
# a fresh consistent snapshot before container replacement, captured atomically
# under the project lock using the same helper as scripts/deploy.sh.
schema_changed=0
git diff --quiet "$live_sha" HEAD -- internal/db/migrations
schema_diff_status=$?
case $schema_diff_status in
    0) ;;
    1) schema_changed=1 ;;
    *) fail_before_change "cannot determine whether database migrations changed" ;;
esac

if [ "$schema_changed" -eq 1 ]; then
    data_dir=$(required_deploy_env HAUSV_DEPLOY_DATA_DIR)
    snapshot_root=$(required_deploy_env HAUSV_DEPLOY_SNAPSHOT_ROOT)
    snapshot_dir="$snapshot_root/$app_version-$commit"
    case $data_dir in
        /*) ;;
        *) fail_before_change "HAUSV_DEPLOY_DATA_DIR must be an absolute path" ;;
    esac
    case $data_dir in
        *[!A-Za-z0-9_./-]*) fail_before_change "HAUSV_DEPLOY_DATA_DIR contains unsupported path characters" ;;
    esac
    case $snapshot_root in
        /*) ;;
        *) fail_before_change "HAUSV_DEPLOY_SNAPSHOT_ROOT must be an absolute path" ;;
    esac
    case $snapshot_root in
        *[!A-Za-z0-9_./-]*) fail_before_change "HAUSV_DEPLOY_SNAPSHOT_ROOT contains unsupported path characters" ;;
    esac
fi

ghcr_token_file=${HAUSV_DEPLOY_GHCR_TOKEN_FILE:-/run/agenix/csb1-hausv-ghcr-pull}
ghcr_user=${HAUSV_DEPLOY_GHCR_USER:-x-access-token}

previous_tag="$image_repo:prev-$live_version-$live_commit"
release_tag="$image_repo:release-$app_version-$commit"

# Preflight verifies the currently healthy service. The script runs locally on
# the runner, so locked_remote_script evals the body instead of using SSH.
preflight_body="\
    locked_live_page=\"\$(curl -fsS --max-time 10 $live_url)\"; \
    locked_open=\"\$(printf \"\\050\")\"; \
    locked_close=\"\$(printf \"\\051\")\"; \
    locked_build_marker=\"$live_version \${locked_open}$live_commit\${locked_close}\"; \
    printf \"%s\" \"\$locked_live_page\" | grep -F \"\$locked_build_marker\" >/dev/null; \
    test \"\$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $container)\" = healthy; \
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $container)\"; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\"; \
    compose_project_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.project\"}}' $container)\"; \
    compose_service_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.service\"}}' $container)\"; \
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
    printf '%s\n' \"\$compose_services\" | grep -Fx $service >/dev/null;"
preflight_body="$preflight_body \
    printf 'running-image-id=%s\n' \"\$running_image_id\""
preflight_script=$(locked_remote_script "$preflight_body") \
    || fail_before_change "cannot encode the locked remote preflight"
preflight_proof=$(eval "$preflight_script") \
    || fail_before_change "local preflight failed"
expected_running_image_id=${preflight_proof#running-image-id=}
if [ "$preflight_proof" != "running-image-id=$expected_running_image_id" ] \
    || [ "${#expected_running_image_id}" -ne 71 ] \
    || ! printf '%s' "$expected_running_image_id" \
        | grep -qE '^sha256:[0-9a-f]{64}$'; then
    fail_before_change "local preflight returned invalid image identity proof"
fi

echo "release candidate: $app_version ($commit)"
echo "CI image: $release_tag"
echo "live now: $live_version ($live_commit), health ok"
if [ "$schema_changed" -eq 1 ]; then
    echo "schema change: yes — snapshot mandatory"
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

preserve_body="\
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $container)\"; \
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
preserve_script=$(locked_remote_script "$preserve_body") \
    || fail_before_change "cannot encode rollback image preservation"
eval "$preserve_script" \
    || fail_before_change "could not preserve the currently running image"
echo "preserved previous image: $previous_tag"

echo "pulling CI image from GHCR…"
if [ ! -r "$ghcr_token_file" ]; then
    fail_before_change "GHCR token file is not readable: $ghcr_token_file"
fi
( docker login ghcr.io -u "$ghcr_user" --password-stdin < "$ghcr_token_file" ) >/dev/null 2>&1 \
    || fail_before_change "GHCR login failed"
docker pull "$release_tag" \
    || fail_before_change "CI image pull failed; the live image and container were not changed"
docker image inspect "$release_tag" >/dev/null \
    || fail_before_change "CI image is not available after pull"
expected_release_image_id=$(docker image inspect --format '{{.Id}}' "$release_tag") \
    || fail_before_change "cannot resolve the pulled CI image identity"
if [ "${#expected_release_image_id}" -ne 71 ] \
    || ! printf '%s' "$expected_release_image_id" \
        | grep -qE '^sha256:[0-9a-f]{64}$'; then
    fail_before_change "pulled CI image returned invalid identity proof"
fi

if [ "$schema_changed" -eq 1 ]; then
    snapshot_name="$app_version-$commit"
    container_snapshot="/snapshots/$snapshot_name"
    source_mount=$(shell_quote "type=bind,src=$data_dir,dst=/source,readonly")
    snapshot_mount=$(shell_quote "type=bind,src=$snapshot_root,dst=/snapshots")
    snapshot_body="
recovery_required=0
recover_snapshot_service() {
    if [ \"\$recovery_required\" = 1 ]; then
        recovery_required=0
        $compose_command up -d --force-recreate --no-deps $service >/dev/null 2>&1 || true
    fi
}
trap recover_snapshot_service EXIT HUP INT TERM
test \"\$(docker inspect --format '{{.Image}}' $container)\" = $expected_running_image_id
test \"\$(docker image inspect --format '{{.Id}}' $image)\" = $expected_running_image_id
test \"\$(docker image inspect --format '{{.Id}}' $release_tag)\" = $expected_release_image_id
recovery_required=1
$compose_command stop -t 30 $service
docker run --rm \
    --user 0:0 \
    --network none \
    --read-only \
    --cap-drop ALL \
    --cap-add DAC_OVERRIDE \
    --security-opt no-new-privileges \
    --tmpfs /tmp:rw,noexec,nosuid,nodev,size=16m \
    --mount $source_mount \
    --mount $snapshot_mount \
    $expected_release_image_id predeploy-snapshot \
    --source /source \
    --snapshot $container_snapshot \
    --source-version $live_version \
    --source-commit $live_sha \
    --target-version $app_version \
    --target-commit $head_sha
$compose_command start $service
snapshot_health=none
snapshot_attempt=0
while [ \"\$snapshot_attempt\" -lt $snapshot_verify_attempts ]; do
    snapshot_health=\"\$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $container 2>/dev/null | tr -d '[:space:]')\"
    [ \"\$snapshot_health\" = healthy ] && break
    snapshot_attempt=\$((snapshot_attempt + 1))
    sleep $verify_sleep
done
test \"\$snapshot_health\" = healthy
recovery_required=0
trap - EXIT HUP INT TERM
printf 'snapshot-host-path=%s\nservice-health=healthy\n' $snapshot_dir"
    snapshot_script=$(locked_remote_script "$snapshot_body") \
        || fail_before_change "cannot encode the locked pre-deploy snapshot"
    if ! snapshot_proof=$(eval "$snapshot_script"); then
        current_health=none
        for _ in $(seq "$snapshot_verify_attempts"); do
            current_health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container" 2>/dev/null | tr -d '[:space:]')
            [ "$current_health" = healthy ] && break
            sleep "$verify_sleep"
        done
        if [ "$current_health" = healthy ]; then
            echo "release refused: fresh consistent pre-deploy snapshot failed." >&2
            echo "recovery verified: the current production service is healthy; no new image or schema was activated." >&2
        else
            echo "release FAILED: pre-deploy snapshot recovery did not return HAUSV to healthy state (container=$current_health)." >&2
            echo "locked mandatory recovery shell: $flock_bin -w 300 $compose_lock /bin/sh -eu" >&2
            echo "inside that locked shell, mandatory recovery command: $compose_command up -d --force-recreate --no-deps $service" >&2
        fi
        exit 1
    fi
    if ! { [[ $snapshot_proof == *"snapshot-path=$container_snapshot"* ]] \
        && [[ $snapshot_proof == *"snapshot-host-path=$snapshot_dir"* ]] \
        && [[ $snapshot_proof == *"snapshot-integrity=ok"* ]] \
        && [[ $snapshot_proof == *"service-health=healthy"* ]] \
        && printf '%s' "$snapshot_proof" | grep -qE 'snapshot-created=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z'; }; then
        echo "release refused: snapshot command returned incomplete proof." >&2
        echo "recovery verified: the current production service is healthy; no new image or schema was activated." >&2
        exit 1
    fi
    echo "pre-deploy recovery point: fresh, consistent and healthy ✓"
fi

echo "replacing the production container…"
activate_body="\
    running_image_id=\"\$(docker inspect --format '{{.Image}}' $container)\" || exit 42; \
    latest_image_id=\"\$(docker image inspect --format '{{.Id}}' $image)\" || exit 42; \
    previous_image_id=\"\$(docker image inspect --format '{{.Id}}' $previous_tag)\" || exit 42; \
    release_image_id=\"\$(docker image inspect --format '{{.Id}}' $release_tag)\" || exit 42; \
    compose_project_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.project\"}}' $container)\" || exit 42; \
    compose_service_label=\"\$(docker inspect --format '{{index .Config.Labels \"com.docker.compose.service\"}}' $container)\" || exit 42; \
    if [ -z \"\$running_image_id\" ] \
        || [ \"\$release_image_id\" != $expected_release_image_id ] \
        || [ \"\$running_image_id\" != $expected_running_image_id ] \
        || [ \"\$latest_image_id\" != $expected_running_image_id ] \
        || [ \"\$previous_image_id\" != $expected_running_image_id ] \
        || [ \"\$running_image_id\" != \"\$latest_image_id\" ] \
        || [ \"\$running_image_id\" != \"\$previous_image_id\" ] \
        || [ \"\$compose_project_label\" != $compose_project ] \
        || [ \"\$compose_service_label\" != $service ]; then \
        exit 42; \
    fi; \
    docker tag $expected_release_image_id $image || exit 43; \
    test \"\$(docker image inspect --format '{{.Id}}' $image)\" = \"\$release_image_id\" || exit 43; \
    $compose_command up -d --force-recreate --no-deps $service || exit 43"
activate_script=$(activation_locked_remote_script "$activate_body") \
    || fail_before_change "cannot encode the locked activation transaction"
eval "$activate_script"
activation_status=$?
case $activation_status in
    0) ;;
    40) fail_before_change "activation transport setup failed before lock and retag; production was not changed by this release" ;;
    41) fail_before_change "activation lock was unavailable before retag; production was not changed by this release" ;;
    42) fail_before_change "activation identity changed before retag; production was not changed by this release" ;;
    *) fail_after_change "container replacement failed" "$schema_changed" "$previous_tag" "$image" "$compose_command" "$service" "$snapshot_dir" ;;
esac

post_ok=0
deployed=""
container_health=""
for _ in $(seq "$verify_attempts"); do
    container_health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container" 2>/dev/null | tr -d '[:space:]')
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
    fail_after_change "post-deploy health/version verification failed (container=$container_health, visible='$deployed')" "$schema_changed" "$previous_tag" "$image" "$compose_command" "$service" "$snapshot_dir"
fi

critical_pattern='application initialization failed|sqlite unavailable|migration .* failed|import to sqlite failed|not atomic|expired (audit|energy) data purge failed|panic|fatal'
startlog_script="\
    started=\"\$(docker inspect --format '{{.State.StartedAt}}' $container)\"; \
    log_file=\"\$(mktemp)\"; \
    trap 'rm -f \"\$log_file\"' EXIT; \
    docker logs $container --since \"\$started\" >\"\$log_file\" 2>&1; \
    grep -F '\"msg\":\"listening\"' \"\$log_file\" >/dev/null; \
    if grep -qiE '$critical_pattern' \"\$log_file\"; then exit 1; fi"
eval "$startlog_script" \
    || fail_after_change "critical start-log check failed or the listening marker is missing" "$schema_changed" "$previous_tag" "$image" "$compose_command" "$service" "$snapshot_dir"

echo "live version: $deployed ✓"
echo "container health: $container_health ✓"
echo "public health: ok ✓"
echo "critical start logs: clean ✓"
print_rollback "$schema_changed" "$previous_tag" "$image" "$compose_command" "$service" "$snapshot_dir"
