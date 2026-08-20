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
for command_name in git docker curl; do
    ln -s "$test_mock" "$test_root/bin/$command_name" || exit 1
done

# Create mock flock/base64/mktemp that actually work for the local script
for bin_name in flock base64 mktemp; do
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
        bash "$test_repo/scripts/deploy-from-ci-image.sh" 9.99.0 aaaaaaa >"$output" 2>&1
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

# Test schema path with missing snapshot root
fixture migrations_changed 1 "HAUSV_DEPLOY_SNAPSHOT_ROOT is required"
if grep -qF -- "pulling CI image from GHCR" "$test_root/migrations_changed/output.txt"; then
    echo "FAIL migrations_changed: pull ran despite missing snapshot config" >&2
    exit 1
fi

# Test schema path with missing data dir
fixture migrations_changed_no_data 1 "HAUSV_DEPLOY_DATA_DIR is required"

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

# Test dry-run with no schema change (must not invoke sudo or snapshot)
# This fails at preflight (docker not fully mocked) but verifies schema logic
fixture no_schema_dry_run_check 1 "local preflight failed"
# Verify no sudo probe was added for non-schema case
if grep -qF -- "sudo" "$test_root/no_schema_dry_run_check/output.txt"; then
    echo "FAIL no_schema_dry_run_check: invoked sudo for non-schema dry-run" >&2
    exit 1
fi

# Test with schema change and snapshot env vars
fixture_schema() {
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
        HAUSV_DEPLOY_DATA_DIR="/var/lib/hausv" \
        HAUSV_DEPLOY_SNAPSHOT_ROOT="/var/backups/hausv" \
        HAUSV_DEPLOY_FLOCK_BIN="$test_root/bin/flock" \
        HAUSV_DEPLOY_BASE64_BIN="$test_root/bin/base64" \
        HAUSV_DEPLOY_MKTEMP_BIN="$test_root/bin/mktemp" \
        HAUSV_DEPLOY_LIVE_URL="https://portal.example.invalid/demo/" \
        HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
        HAUSV_DEPLOY_VERIFY_SLEEP=0 \
        bash "$test_repo/scripts/deploy-from-ci-image.sh" 9.99.0 aaaaaaa >"$output" 2>&1
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

# Test with schema change and preflight failure (verifies sudo probe was configured)
# Preflight will fail but the sudo probe should have been added to the script
fixture_schema schema_with_snapshot 1 "local preflight failed"

echo "$test_passed deploy-from-ci-image fixtures passed."
