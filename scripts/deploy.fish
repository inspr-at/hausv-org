#!/usr/bin/env fish
# Fail-closed manual production release (HAUSV-403).
#
#   scripts/deploy.fish [--dry-run]
#
# CI proves the exact commit on Blacksmith but never publishes an image.
# Production is built from `git archive HEAD` on csb1 after every precondition
# below has passed. A schema-changing release first creates a quiesced,
# transactionally consistent recovery point outside the live mount.

function fail_before_change --argument-names message
    echo "release refused: $message" >&2
    echo "rollback: not required; production was not changed." >&2
    exit 1
end

function visible_build --argument-names page
    set -l captures (string match -rg '([0-9]+\.[0-9]+\.[0-9]+) \(([0-9a-f]{7})\)' -- $page | head -2)
    if test (count $captures) -eq 2
        printf '%s\t%s\n' $captures[1] $captures[2]
    end
end

function valid_health_payload --argument-names payload
    string match -qr '"service"[[:space:]]*:[[:space:]]*"hausv-org"' -- $payload
    and string match -qr '"status"[[:space:]]*:[[:space:]]*"ok"' -- $payload
end

function remote_sh_command --argument-names script
    # csb1's SSH login shell is fish. Hand every compound remote operation to
    # POSIX sh explicitly so `-e` and `-u` cannot be parsed (or ignored) by the
    # login shell. The outer escaping is deliberately fish-compatible because
    # that is the shell which receives SSH's command string.
    printf '/bin/sh -eu -c %s' (string escape --style=script -- "$script")
end

function printable_remote_ssh --argument-names ssh_port ssh_host script
    set -l remote_command (remote_sh_command "$script")
    printf 'ssh -p %s %s %s' $ssh_port $ssh_host (string escape --style=script -- "$remote_command")
end

function print_rollback --argument-names schema_changed previous_tag ssh_host ssh_port image compose_dir service snapshot_dir
    set -l image_script "docker tag $previous_tag $image; cd $compose_dir; docker compose up -d --force-recreate --no-deps $service"
    set -l image_command (printable_remote_ssh $ssh_port $ssh_host "$image_script")
    if test "$schema_changed" = 1
        set -l containment_script "cd $compose_dir; docker compose stop -t 30 $service"
        echo "containment command: "(printable_remote_ssh $ssh_port $ssh_host "$containment_script")
        echo "data/schema restore required: do NOT run an image-only rollback against a possibly migrated database."
        echo "restore source: $snapshot_dir (root-only, pre-deploy SQLite + blobs)."
        echo "restore procedure: hausv-org docs/csb1-deploy.md → “Schema rollback procedure”; verify an isolated copy with system Python before replacing live data."
        echo "image command after the matching data restore: $image_command"
    else
        echo "image rollback command: $image_command"
        echo "data/schema restore: not required; this release contains no migration change."
    end
end

function fail_after_change --argument-names message schema_changed previous_tag ssh_host ssh_port image compose_dir service snapshot_dir
    echo "release FAILED: $message" >&2
    print_rollback $schema_changed $previous_tag $ssh_host $ssh_port $image $compose_dir $service $snapshot_dir >&2
    exit 1
end

set -l dry_run 0
for arg in $argv
    switch $arg
        case --dry-run
            set dry_run 1
        case '*'
            echo "usage: scripts/deploy.fish [--dry-run]" >&2
            exit 1
    end
end

set -l ssh_host mba@cs1.barta.cm
set -l ssh_port 2222
set -l github_repo inspr-at/hausv-org
set -l workflow CI
set -l image ghcr.io/markus-barta/hausv-org:latest
set -l compose_dir /home/mba/Code/nixcfg/hosts/csb1/docker
set -l service hausv-org
set -l build_dir /tmp/hausv-build
set -l data_dir /var/lib/csb1-docker/hausv-org
set -l snapshot_root /var/lib/csb1-docker/hausv-org-predeploy
set -l live_url https://jhw22.hausv.org/
set -l health_url https://jhw22.hausv.org/healthz
set -l verify_attempts 15
set -l verify_sleep 2
if set -q HAUSV_DEPLOY_VERIFY_ATTEMPTS
    string match -qr '^[1-9][0-9]*$' -- $HAUSV_DEPLOY_VERIFY_ATTEMPTS
    or fail_before_change "HAUSV_DEPLOY_VERIFY_ATTEMPTS must be a positive integer"
    set verify_attempts $HAUSV_DEPLOY_VERIFY_ATTEMPTS
end
if set -q HAUSV_DEPLOY_VERIFY_SLEEP
    string match -qr '^[0-9]+$' -- $HAUSV_DEPLOY_VERIFY_SLEEP
    or fail_before_change "HAUSV_DEPLOY_VERIFY_SLEEP must be a non-negative integer"
    set verify_sleep $HAUSV_DEPLOY_VERIFY_SLEEP
end

for command in git gh curl ssh
    type -q $command
    or fail_before_change "required command '$command' is unavailable"
end

set -l repo (git rev-parse --show-toplevel 2>/dev/null)
or fail_before_change "not inside the HAUSV repository"
builtin cd $repo
or fail_before_change "cannot enter repository root"

# The archive contains only HEAD. A dirty tree would make the operator believe
# uncommitted work was shipped even though it cannot be in the image.
set -l dirty (git status --porcelain)
or fail_before_change "cannot inspect the working tree"
if test (count $dirty) -gt 0
    echo "dirty paths:" >&2
    git status --short >&2
    fail_before_change "working tree is dirty"
end

# Fetch, do not trust a stale remote-tracking ref.
git fetch --quiet origin main
or fail_before_change "cannot refresh origin/main"
set -l head_sha (git rev-parse HEAD 2>/dev/null)
or fail_before_change "cannot resolve HEAD"
set -l origin_sha (git rev-parse refs/remotes/origin/main 2>/dev/null)
or fail_before_change "cannot resolve origin/main"
if test "$head_sha" != "$origin_sha"
    fail_before_change "HEAD $head_sha is not exactly origin/main $origin_sha"
end
set -l commit (git rev-parse --short=7 HEAD 2>/dev/null)
or fail_before_change "cannot resolve the visible commit"

set -l app_version (string trim (cat VERSION))
string match -qr '^[0-9]+\.[0-9]+\.[0-9]+$' -- $app_version
or fail_before_change "VERSION must be a non-empty semantic version"
set -l snapshot_dir "$snapshot_root/$app_version-$commit"

# The required workflow at this exact commit must itself use Blacksmith for
# every job. Parse the jobs block instead of merely collecting runs-on lines:
# otherwise a reusable-workflow job without runs-on would silently escape the
# gate. Reusable jobs are rejected; their runner contract is outside this file.
set -l workflow_source (git show HEAD:.github/workflows/ci.yml 2>/dev/null)
or fail_before_change "cannot read .github/workflows/ci.yml from HEAD"
set -l in_jobs 0
set -l job_count 0
set -l current_job ""
set -l current_has_runner 0
for workflow_line in $workflow_source
    if test $in_jobs -eq 0
        if string match -qr '^jobs:[[:space:]]*$' -- $workflow_line
            set in_jobs 1
        end
        continue
    end

    if string match -qr '^[^[:space:]]' -- $workflow_line
        break
    end

    if string match -qr '^  [A-Za-z0-9_-]+:[[:space:]]*$' -- $workflow_line
        if test -n "$current_job" -a $current_has_runner -ne 1
            fail_before_change "CI job '$current_job' has no explicit Blacksmith runner"
        end
        set current_job (string replace -r '^  ([A-Za-z0-9_-]+):[[:space:]]*$' '$1' -- $workflow_line)
        set current_has_runner 0
        set job_count (math $job_count + 1)
        continue
    end

    if test -n "$current_job"
        if string match -qr '^    uses:[[:space:]]*' -- $workflow_line
            fail_before_change "CI job '$current_job' uses a reusable workflow whose Blacksmith runners cannot be proven"
        end
        if string match -qr '^    runs-on:[[:space:]]*' -- $workflow_line
            string match -qr '^    runs-on:[[:space:]]*blacksmith-[A-Za-z0-9_-]+[[:space:]]*(#.*)?$' -- $workflow_line
            or fail_before_change "every required CI job must run on Blacksmith"
            set current_has_runner 1
        end
    end
end
if test $job_count -eq 0
    fail_before_change "CI workflow contains no jobs"
end
if test -n "$current_job" -a $current_has_runner -ne 1
    fail_before_change "CI job '$current_job' has no explicit Blacksmith runner"
end

set -l ci_lines (gh run list \
    --repo $github_repo \
    --workflow $workflow \
    --branch main \
    --commit $head_sha \
    --event push \
    --limit 10 \
    --json databaseId,headSha,headBranch,status,conclusion,url \
    --jq '.[] | select(.headBranch == "main" and .status == "completed" and .conclusion == "success") | [.databaseId,.headSha,.url] | @tsv' 2>/dev/null)
or fail_before_change "cannot query the required GitHub Actions workflow"
if test (count $ci_lines) -eq 0
    fail_before_change "no successful completed Blacksmith CI push run exists for exact HEAD $head_sha"
end
set -l ci_fields (string split \t -- $ci_lines[1])
if test (count $ci_fields) -lt 3 -o "$ci_fields[2]" != "$head_sha"
    fail_before_change "the successful CI result does not belong to exact HEAD $head_sha"
end
set -l ci_run $ci_fields[1]
set -l ci_url $ci_fields[3]

set -l live_page_lines (curl -fsS --max-time 10 $live_url)
or fail_before_change "cannot read the current live build"
set -l live_page (string join \n -- $live_page_lines | string collect)
set -l live_fields (string split \t -- (visible_build $live_page))
if test (count $live_fields) -ne 2
    fail_before_change "the current live version and commit are not visible"
end
set -l live_version $live_fields[1]
set -l live_commit $live_fields[2]
if test "$live_version" = "$app_version"
    fail_before_change "VERSION $app_version is already live; every production deployment requires a VERSION bump"
end

set -l live_health (curl -fsS --max-time 10 $health_url)
or fail_before_change "the current production health endpoint is unavailable"
valid_health_payload (string join \n -- $live_health)
or fail_before_change "the current production health payload is not healthy"

set -l live_sha (git rev-parse "$live_commit^{commit}" 2>/dev/null)
or fail_before_change "live commit $live_commit is not present in local git history"
string match -qr "^$live_commit" -- $live_sha
or fail_before_change "live commit $live_commit is ambiguous"

set -l schema_changed 0
git diff --quiet $live_sha HEAD -- internal/db/migrations
set -l schema_diff_status $status
switch $schema_diff_status
    case 0
    case 1
        set schema_changed 1
    case '*'
        fail_before_change "cannot determine whether database migrations changed"
end

set -l previous_tag "ghcr.io/markus-barta/hausv-org:prev-$live_version-$live_commit"

# Read-only remote preflight. It verifies the currently healthy service, image,
# compose entry and (for schema changes) non-interactive root capability. The
# running container must be the exact image currently named by :latest. An
# existing rollback tag is accepted only when it already names that same image;
# this makes retries after a partial activation fail closed. Dry run stops here
# and never creates a snapshot or changes a container.
set -l sudo_probe ""
if test $schema_changed -eq 1
    set sudo_probe "sudo -n /run/current-system/sw/bin/python3 -c 'import sqlite3' >/dev/null;"
end
set -l preflight_script "\
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
ssh -p $ssh_port $ssh_host (remote_sh_command "$preflight_script")
or fail_before_change "remote preflight failed"

echo "release candidate: $app_version ($commit)"
echo "origin/main: $head_sha ✓"
echo "Blacksmith CI: run $ci_run for exact HEAD ✓"
echo "CI evidence: $ci_url"
echo "live now: $live_version ($live_commit), health ok"
if test $schema_changed -eq 1
    echo "schema change: yes — a fresh consistent snapshot is mandatory before container replacement"
else
    echo "schema change: no"
end

if test $dry_run -eq 1
    echo "[dry-run] all fail-closed preconditions passed; production was not changed."
    if test $schema_changed -eq 1
        echo "[dry-run] actual release will publish $snapshot_dir atomically before starting the new image."
    end
    exit 0
end

if test $schema_changed -eq 1
    set -l snapshot_script "\
        sudo -n /run/current-system/sw/bin/python3 - \
        --source $data_dir \
        --snapshot $snapshot_dir \
        --compose-dir $compose_dir \
        --service $service \
        --source-version $live_version \
        --source-commit $live_sha \
        --target-version $app_version \
        --target-commit $head_sha"
    set -l snapshot_proof (ssh -p $ssh_port $ssh_host \
        (remote_sh_command "$snapshot_script") \
        < $repo/scripts/create-predeploy-snapshot.py)
    or begin
        set -l current_health (ssh -p $ssh_port $ssh_host \
            "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service" 2>/dev/null | string trim)
        if test "$current_health" = healthy
            echo "release refused: fresh consistent pre-deploy snapshot failed." >&2
            echo "recovery verified: the current production service is healthy; no new image or schema was activated." >&2
        else
            echo "release FAILED: pre-deploy snapshot recovery did not return HAUSV to healthy state (container=$current_health)." >&2
            set -l recovery_script "cd $compose_dir; docker compose up -d --force-recreate --no-deps $service"
            echo "mandatory recovery command: "(printable_remote_ssh $ssh_port $ssh_host "$recovery_script") >&2
            echo "after recovery, require Docker health=healthy before any retry; do not activate the new image." >&2
        end
        exit 1
    end
    string match -q "snapshot-path=$snapshot_dir" -- $snapshot_proof
    and string match -q "snapshot-integrity=ok" -- $snapshot_proof
    and string match -q "service-health=healthy" -- $snapshot_proof
    and string match -qr '^snapshot-created=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' -- $snapshot_proof
    or begin
        echo "release refused: snapshot command returned incomplete proof." >&2
        echo "recovery verified by the helper: the current production service is healthy; no new image or schema was activated." >&2
        exit 1
    end
    echo "pre-deploy recovery point: fresh, consistent and healthy ✓"
end

set -l preserve_script "\
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
ssh -p $ssh_port $ssh_host (remote_sh_command "$preserve_script")
or fail_before_change "could not preserve the currently running image"
echo "preserved previous image: $previous_tag"

set -l release_tag "ghcr.io/markus-barta/hausv-org:release-$app_version-$commit"
echo "building exact HEAD…"
set -lx COPYFILE_DISABLE 1
set -l build_script "\
    test \"$build_dir\" = /tmp/hausv-build; \
    rm -rf \"$build_dir\"; \
    mkdir -p $build_dir; \
    trap 'rm -rf \"$build_dir\"' EXIT; \
    cd $build_dir; \
    tar -x; \
    docker build --build-arg APP_VERSION=$app_version --build-arg GIT_COMMIT=$commit -t $release_tag .; \
    docker image inspect $release_tag >/dev/null"
git archive --format=tar HEAD | ssh -p $ssh_port $ssh_host (remote_sh_command "$build_script")
set -l pipeline_status $pipestatus
for command_status in $pipeline_status
    if test $command_status -ne 0
        fail_before_change "release image build failed; the live image and container were not changed"
    end
end

echo "replacing the production container…"
set -l activate_script "\
    docker tag $release_tag $image; \
    cd $compose_dir; \
    docker compose up -d --force-recreate --no-deps $service"
ssh -p $ssh_port $ssh_host (remote_sh_command "$activate_script")
or fail_after_change "container replacement failed" $schema_changed $previous_tag $ssh_host $ssh_port $image $compose_dir $service $snapshot_dir

set -l post_ok 0
set -l deployed ""
set -l container_health ""
for attempt in (seq $verify_attempts)
    set container_health (ssh -p $ssh_port $ssh_host "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $service" 2>/dev/null | string trim)
    set -l health_lines (curl -fsS --max-time 10 $health_url 2>/dev/null)
    set -l page_lines (curl -fsS --max-time 10 $live_url 2>/dev/null)
    set -l page_text (string join \n -- $page_lines | string collect)
    set -l deployed_fields (string split \t -- (visible_build $page_text))
    if test (count $deployed_fields) -eq 2
        set deployed "$deployed_fields[1] ($deployed_fields[2])"
    end
    if test "$container_health" = healthy
        and valid_health_payload (string join \n -- $health_lines)
        and test "$deployed" = "$app_version ($commit)"
        set post_ok 1
        break
    end
    sleep $verify_sleep
end
if test $post_ok -ne 1
    fail_after_change "post-deploy health/version verification failed (container=$container_health, visible='$deployed')" $schema_changed $previous_tag $ssh_host $ssh_port $image $compose_dir $service $snapshot_dir
end

# Inspect only the current container's start window. Logs are evaluated on the
# host and never printed, avoiding accidental disclosure of operational data.
set -l critical_pattern 'application initialization failed|sqlite unavailable|migration .* failed|import to sqlite failed|not atomic|expired (audit|energy) data purge failed|panic|fatal'
set -l startlog_script "\
    started=\"\$(docker inspect --format '{{.State.StartedAt}}' $service)\"; \
    log_file=\"\$(mktemp)\"; \
    trap 'rm -f \"\$log_file\"' EXIT; \
    docker logs $service --since \"\$started\" >\"\$log_file\" 2>&1; \
    grep -F '\"msg\":\"listening\"' \"\$log_file\" >/dev/null; \
    if grep -qiE '$critical_pattern' \"\$log_file\"; then exit 1; fi"
ssh -p $ssh_port $ssh_host (remote_sh_command "$startlog_script")
or fail_after_change "critical start-log check failed or the listening marker is missing" $schema_changed $previous_tag $ssh_host $ssh_port $image $compose_dir $service $snapshot_dir

echo "live version: $deployed ✓"
echo "container health: $container_health ✓"
echo "public health: ok ✓"
echo "critical start logs: clean ✓"
print_rollback $schema_changed $previous_tag $ssh_host $ssh_port $image $compose_dir $service $snapshot_dir
