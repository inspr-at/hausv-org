#!/usr/bin/env bash
# Test fixtures for scripts/deploy-from-ci-image.sh migration refuse.
#
# This tests the fail-closed migration gate that was added to prevent
# schema-changing releases from the Mac-less CI deployment path.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

test_repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
test_mock=$test_repo/scripts/testdata/deploy-ci-image/mock-command
test_root=$(mktemp -d -t hausv-deploy-ci-image-test.XXXXXX) || exit 1

cleanup() {
    case $test_root in
        /*hausv-deploy-ci-image-test.*) command rm -rf -- "$test_root" ;;
    esac
}
trap cleanup EXIT

mkdir -p "$test_root/bin" || exit 1
for command_name in git docker curl flock; do
    ln -s "$test_mock" "$test_root/bin/$command_name" || exit 1
done

# Create base64/mktemp links that actually work for the local script. flock is
# mocked above so this fixture remains deterministic on macOS, which lacks it.
for bin_name in base64 mktemp; do
    if command -v "$bin_name" >/dev/null 2>&1; then
        ln -s "$(command -v "$bin_name")" "$test_root/bin/$bin_name" || exit 1
    fi
done

test_passed=0

fixture() {
    local case_name=$1 expected_status=$2 expected_text=$3
    local state=$test_root/$case_name
    mkdir -p "$state"
    local output=$state/output.txt
    local fixture_path=$test_root/bin:$PATH
    local actual_status

    env \
        PATH="$fixture_path" \
        TEST_FIXTURE_CASE="$case_name" \
        TEST_FIXTURE_STATE="$state" \
        TEST_FIXTURE_REPO="$test_repo" \
        HAUSV_DEPLOY_COMPOSE_DIR="/srv/hausv/compose" \
        HAUSV_DEPLOY_COMPOSE_FILE="/srv/hausv/compose/docker-compose.yml" \
        HAUSV_DEPLOY_COMPOSE_PROJECT=hausv \
        HAUSV_DEPLOY_CONTAINER=hausv-demo \
        HAUSV_DEPLOY_COMPOSE_LOCK="$state/compose.lock" \
        HAUSV_DEPLOY_FLOCK_BIN="$test_root/bin/flock" \
        HAUSV_DEPLOY_BASE64_BIN="$test_root/bin/base64" \
        HAUSV_DEPLOY_MKTEMP_BIN="$test_root/bin/mktemp" \
        HAUSV_DEPLOY_LIVE_URL="https://portal.example.invalid/demo/" \
        HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
        HAUSV_DEPLOY_VERIFY_SLEEP=0 \
        HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS=1 \
        bash "$test_repo/scripts/deploy-from-ci-image.sh" 9.99.0 aaaaaaa >"$output" 2>&1
    actual_status=$?

    if [ "$actual_status" -ne "$expected_status" ]; then
        echo "FAIL $case_name: exit $actual_status, expected $expected_status" >&2
        command cat "$output" >&2
        [ ! -f "$state/commands.log" ] || command cat "$state/commands.log" >&2
        exit 1
    fi
    if ! grep -qF -- "$expected_text" "$output"; then
        echo "FAIL $case_name: missing '$expected_text'" >&2
        command cat "$output" >&2
        exit 1
    fi
    test_passed=$((test_passed + 1))
    echo "ok $case_name"
}

# Test schema path with missing snapshot root
fixture migrations_changed 1 "HAUSV_DEPLOY_SNAPSHOT_ROOT is required"
if grep -qF -- "pulling CI image from GHCR" "$test_root/migrations_changed/output.txt"; then
    echo "FAIL migrations_changed: pull ran despite missing snapshot config" >&2
    exit 1
fi

# Test schema path with missing data dir
fixture migrations_changed_no_data 1 "HAUSV_DEPLOY_DATA_DIR is required"

# A push without a VERSION bump is nothing to release, not a failed release:
# exit 3, no pull, no swap. Every push to main reaches this case.
fixture version_already_live 3 "nothing to release: VERSION 9.99.0 is already live"
if grep -qF -- "pulling CI image from GHCR" "$test_root/version_already_live/output.txt"; then
    echo "FAIL version_already_live: pulled an image although there was nothing to release" >&2
    exit 1
fi
if grep -qF -- "release refused" "$test_root/version_already_live/output.txt"; then
    echo "FAIL version_already_live: reported a refusal instead of nothing to release" >&2
    exit 1
fi

# Test git repo missing
fixture not_in_repo 1 "not inside the HAUSV repository"
if grep -qF -- "schema change" "$test_root/not_in_repo/output.txt"; then
    echo "FAIL not_in_repo: reached migration check without git repo" >&2
    exit 1
fi

# Test live commit not in git history
fixture live_commit_missing 1 "live commit"
if ! grep -qF -- "not present in local git history" "$test_root/live_commit_missing/output.txt"; then
    echo "FAIL live_commit_missing: missing fail-closed message" >&2
    exit 1
fi

# Test git diff failure
fixture git_diff_fail 1 "cannot determine whether database migrations changed"

# Test success when no migrations changed
fixture no_migrations 1 "local preflight failed"
# This test case fails at preflight because we don't mock the full docker
# environment, but it verifies the migration check passed.
if ! grep -qF -- "git	diff	--quiet" "$test_root/no_migrations/commands.log"; then
    echo "FAIL no_migrations: migration check did not run" >&2
    exit 1
fi

# Fixtures for the new snapshot support: dry-run behavior and full path validation

# Test with schema change and snapshot env vars
fixture_schema() {
    local case_name=$1 expected_status=$2 expected_text=$3
    local mode=${4:-dry-run}
    local snapshot_attempts=${5:-1}
    local state=$test_root/$case_name
    mkdir -p "$state"
    local output=$state/output.txt
    local fixture_path=$test_root/bin:$PATH
    local actual_status
    local deploy_args=(9.99.0 aaaaaaa)
    if [ "$mode" != release ]; then
        deploy_args+=(--dry-run)
    fi
    mkdir -p "$state/compose"
    : > "$state/compose/compose.yml"
    : > "$state/ghcr-token"

    env \
        PATH="$fixture_path" \
        TEST_FIXTURE_CASE="$case_name" \
        TEST_FIXTURE_STATE="$state" \
        TEST_FIXTURE_REPO="$test_repo" \
        HAUSV_DEPLOY_COMPOSE_DIR="$state/compose" \
        HAUSV_DEPLOY_COMPOSE_FILE="$state/compose/compose.yml" \
        HAUSV_DEPLOY_COMPOSE_PROJECT=hausv \
        HAUSV_DEPLOY_CONTAINER=hausv-demo \
        HAUSV_DEPLOY_COMPOSE_LOCK="$state/compose.lock" \
        HAUSV_DEPLOY_DATA_DIR="/var/lib/hausv" \
        HAUSV_DEPLOY_SNAPSHOT_ROOT="/var/backups/hausv" \
        HAUSV_DEPLOY_FLOCK_BIN="$test_root/bin/flock" \
        HAUSV_DEPLOY_BASE64_BIN="$test_root/bin/base64" \
        HAUSV_DEPLOY_MKTEMP_BIN="$test_root/bin/mktemp" \
        HAUSV_DEPLOY_GHCR_TOKEN_FILE="$state/ghcr-token" \
        HAUSV_DEPLOY_LIVE_URL="https://portal.example.invalid/demo/" \
        HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
        HAUSV_DEPLOY_VERIFY_SLEEP=0 \
        HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS="$snapshot_attempts" \
        bash "$test_repo/scripts/deploy-from-ci-image.sh" "${deploy_args[@]}" >"$output" 2>&1
    actual_status=$?

    if [ "$actual_status" -ne "$expected_status" ]; then
        echo "FAIL $case_name: exit $actual_status, expected $expected_status" >&2
        command cat "$output" >&2
        exit 1
    fi
    if ! grep -qF -- "$expected_text" "$output"; then
        echo "FAIL $case_name: missing '$expected_text'" >&2
        command cat "$output" >&2
        exit 1
    fi
    test_passed=$((test_passed + 1))
    echo "ok $case_name"
}

# A schema-changing dry run reaches every read-only precondition without sudo.
fixture_schema schema_with_snapshot 0 "all fail-closed preconditions passed"
if grep -qF -- "sudo" "$test_root/schema_with_snapshot/commands.log"; then
    echo "FAIL schema_with_snapshot: unattended schema preflight still invoked sudo" >&2
    exit 1
fi

# The non-dry schema fixture reaches the candidate-image snapshot command,
# verifies its Docker confinement, restarts the old service, and only then
# activates and verifies the release.
fixture_schema schema_release_success 0 "live version: 9.99.0 (aaaaaaa)" release 3
if ! grep -qF -- $'docker\trun\t--rm\t--user\t0:0\t--network\tnone\t--read-only' \
    "$test_root/schema_release_success/commands.log"; then
    echo "FAIL schema_release_success: restricted snapshot container did not run" >&2
    exit 1
fi
if ! grep -qF -- $'\tsha256:2222222222222222222222222222222222222222222222222222222222222222\tpredeploy-snapshot\t' \
    "$test_root/schema_release_success/commands.log"; then
    echo "FAIL schema_release_success: snapshot did not use the pinned release image ID" >&2
    exit 1
fi
snapshot_stop_line=$(grep -n $'docker\tcompose\t.*\tstop\t-t\t30\thausv-org$' \
    "$test_root/schema_release_success/commands.log" | cut -d: -f1)
snapshot_run_line=$(grep -n $'^docker\trun\t' \
    "$test_root/schema_release_success/commands.log" | cut -d: -f1)
snapshot_start_line=$(grep -n $'docker\tcompose\t.*\tstart\thausv-org$' \
    "$test_root/schema_release_success/commands.log" | cut -d: -f1)
if [ -z "$snapshot_stop_line" ] || [ -z "$snapshot_run_line" ] || [ -z "$snapshot_start_line" ] \
    || [ "$snapshot_stop_line" -ge "$snapshot_run_line" ] \
    || [ "$snapshot_run_line" -ge "$snapshot_start_line" ]; then
    echo "FAIL schema_release_success: snapshot stop/run/start ordering is not atomic" >&2
    exit 1
fi
if grep -qE -- $'--cap-add\t(CHOWN|FOWNER|DAC_READ_SEARCH)' \
    "$test_root/schema_release_success/commands.log"; then
    echo "FAIL schema_release_success: snapshot container has unnecessary capabilities" >&2
    exit 1
fi
if grep -qF -- "sudo" "$test_root/schema_release_success/commands.log"; then
    echo "FAIL schema_release_success: schema release still invoked sudo" >&2
    exit 1
fi

# A failed snapshot container must recreate the old service under the same
# lock and refuse before the release tag can become latest.
fixture_schema schema_snapshot_container_fail 1 \
    "recovery verified: the current production service is healthy" release
if ! grep -qE -- $'^docker\tcompose\t.*\tup\t-d\t--force-recreate\t--no-deps\thausv-org$' \
    "$test_root/schema_snapshot_container_fail/commands.log"; then
    echo "FAIL schema_snapshot_container_fail: old service was not recovered" >&2
    exit 1
fi
if grep -qE -- $'^docker\ttag\tsha256:2222222222222222222222222222222222222222222222222222222222222222\t.*:latest$' \
    "$test_root/schema_snapshot_container_fail/commands.log"; then
    echo "FAIL schema_snapshot_container_fail: failed snapshot activated the release image" >&2
    exit 1
fi

# HAUSV-634: the host lost the :latest tag. The live release's own tag still
# names the running image, so the preflight accepts it, the preserve step puts
# :latest back under the lock, and the release goes through.
fixture_schema latest_missing 0 "live version: 9.99.0 (aaaaaaa)" release 3
if ! grep -qF -- "is missing on the host" "$test_root/latest_missing/output.txt"; then
    echo "FAIL latest_missing: the missing tag was not reported" >&2
    exit 1
fi
restore_line=$(grep -n $'^docker\ttag\tsha256:1111111111111111111111111111111111111111111111111111111111111111\t.*:latest$' \
    "$test_root/latest_missing/commands.log" | head -1 | cut -d: -f1)
activate_line=$(grep -n $'^docker\ttag\tsha256:2222222222222222222222222222222222222222222222222222222222222222\t.*:latest$' \
    "$test_root/latest_missing/commands.log" | head -1 | cut -d: -f1)
if [ -z "$restore_line" ] || [ -z "$activate_line" ] || [ "$restore_line" -ge "$activate_line" ]; then
    echo "FAIL latest_missing: :latest was not restored onto the running image before activation" >&2
    exit 1
fi

# Without :latest AND without the live release's tag nothing proves the
# running image; the release is refused before any change.
fixture_schema latest_missing_no_release 1 "local preflight failed" release
if grep -qF -- "pulling CI image from GHCR" "$test_root/latest_missing_no_release/output.txt"; then
    echo "FAIL latest_missing_no_release: pulled an image although the running image was unproven" >&2
    exit 1
fi

echo "$test_passed deploy-from-ci-image fixtures passed."
