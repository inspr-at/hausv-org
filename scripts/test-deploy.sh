#!/usr/bin/env bash
# Deterministic fail-closed release fixtures. Never contacts production.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

deploy_fixture_repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
deploy_fixture_mock=$deploy_fixture_repo/scripts/testdata/deploy/mock-command
deploy_fixture_root=$(mktemp -d -t hausv-deploy-fixture.XXXXXX) || exit 1

cleanup() {
    case $deploy_fixture_root in
        /*hausv-deploy-fixture.*) command rm -rf -- "$deploy_fixture_root" ;;
    esac
}
trap cleanup EXIT

mkdir -p "$deploy_fixture_root/bin" || exit 1
for command_name in git gh curl ssh sleep; do
    ln -s "$deploy_fixture_mock" "$deploy_fixture_root/bin/$command_name" || exit 1
done

deploy_fixture_passed=0

contains_fail_fast_remote_command() {
    local output=$1 line escaped decoded
    while IFS= read -r line; do
        case $line in
            *"mandatory recovery command: ssh -p "*) ;;
            *"image rollback command: ssh -p "*) ;;
            *) continue ;;
        esac
        escaped=${line#*: ssh -p }
        escaped=${escaped#* } # drop the port
        escaped=${escaped#* } # drop the host
        # The value was produced by our own shell_quote; eval is the exact
        # inverse and the input is not attacker-controlled in a fixture run.
        decoded=$(eval "printf '%s' $escaped" 2>/dev/null) || continue
        case $decoded in
            "/bin/sh -eu -c "*) return 0 ;;
        esac
    done <<EOF
$output
EOF
    return 1
}

fixture() {
    local case_name=$1 expected_status=$2 expected_text=$3 mode=$4
    local state=$deploy_fixture_root/$case_name
    mkdir -p "$state"
    local output=$state/output.txt
    local fixture_path=$deploy_fixture_root/bin:$PATH
    local actual_status

    if [ "$mode" = dry-run ]; then
        env \
            PATH="$fixture_path" \
            DEPLOY_FIXTURE_CASE="$case_name" \
            DEPLOY_FIXTURE_STATE="$state" \
            DEPLOY_FIXTURE_REPO="$deploy_fixture_repo" \
            HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
            HAUSV_DEPLOY_VERIFY_SLEEP=0 \
            bash "$deploy_fixture_repo/scripts/deploy.sh" --dry-run >"$output" 2>&1
    else
        env \
            PATH="$fixture_path" \
            DEPLOY_FIXTURE_CASE="$case_name" \
            DEPLOY_FIXTURE_STATE="$state" \
            DEPLOY_FIXTURE_REPO="$deploy_fixture_repo" \
            HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
            HAUSV_DEPLOY_VERIFY_SLEEP=0 \
            bash "$deploy_fixture_repo/scripts/deploy.sh" >"$output" 2>&1
    fi
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
    deploy_fixture_passed=$((deploy_fixture_passed + 1))
    echo "ok $case_name"
}

fixture no_schema 0 "all fail-closed preconditions passed" dry-run
for forbidden in "docker build" "docker tag" "--source /var/lib/csb1-docker/hausv-org"; do
    if grep -qF -- "$forbidden" "$deploy_fixture_root/no_schema/commands.log"; then
        echo "FAIL no_schema: dry-run invoked mutating command '$forbidden'" >&2
        exit 1
    fi
done

fixture dirty 1 "working tree is dirty" dry-run
fixture fetch_fail 1 "cannot refresh origin/main" dry-run
fixture origin_mismatch 1 "is not exactly origin/main" dry-run
fixture ci_query_fail 1 "cannot query the required GitHub Actions workflow" dry-run
fixture ci_missing 1 "no successful completed Blacksmith CI" dry-run
fixture ci_wrong_head 1 "successful CI result does not belong to exact HEAD" dry-run
fixture non_blacksmith 1 "every required CI job must run on Blacksmith" dry-run
fixture reusable_job 1 "uses a reusable workflow" dry-run
fixture live_unreadable 1 "current live version and commit are not visible" dry-run
fixture live_health_fail 1 "current production health endpoint is unavailable" dry-run
fixture same_version_different_commit 1 "every production deployment requires a VERSION bump" dry-run
fixture remote_preflight_fail 1 "remote preflight failed" dry-run
fixture remote_early_failure_masked 1 "remote preflight failed" dry-run
fixture retry_after_activation_fail 1 "remote preflight failed" dry-run
for forbidden in "docker build" "docker tag"; do
    if grep -qF -- "$forbidden" "$deploy_fixture_root/retry_after_activation_fail/commands.log"; then
        echo "FAIL retry_after_activation_fail: retry invoked '$forbidden' despite image identity mismatch" >&2
        exit 1
    fi
done
fixture rollback_tag_conflict 1 "remote preflight failed" dry-run
fixture schema_snapshot_fail 1 "fresh consistent pre-deploy snapshot failed" release
if grep -qF -- "docker build" "$deploy_fixture_root/schema_snapshot_fail/commands.log"; then
    echo "FAIL schema_snapshot_fail: build ran without a recovery point" >&2
    exit 1
fi
fixture schema_snapshot_recovery_fail 1 "mandatory recovery command:" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/schema_snapshot_recovery_fail/output.txt")"; then
    echo "FAIL schema_snapshot_recovery_fail: recovery command is not fail-fast" >&2
    exit 1
fi

fixture preserve_fail 1 "could not preserve the currently running image" release
fixture build_fail 1 "release image build failed" release
fixture activation_fail 1 "image rollback command:" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/activation_fail/output.txt")"; then
    echo "FAIL activation_fail: rollback command is not fail-fast" >&2
    exit 1
fi
fixture post_health_fail 1 "image rollback command:" release
fixture post_version_mismatch 1 "post-deploy health/version verification failed" release
fixture schema_startlog_fail 1 "data/schema restore required" release
fixture_app_version=$(tr -d '[:space:]' < "$deploy_fixture_repo/VERSION")
if ! grep -qF -- "/hausv-org-predeploy/$fixture_app_version-aaaaaaa" \
    "$deploy_fixture_root/schema_startlog_fail/output.txt"; then
    echo "FAIL schema_startlog_fail: exact versioned recovery path missing" >&2
    exit 1
fi

fixture success 0 "critical start logs: clean" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/success/output.txt")"; then
    echo "FAIL success: printed rollback command is not fail-fast" >&2
    exit 1
fi
fixture schema_success 0 "pre-deploy recovery point: fresh, consistent and healthy" release

python3 "$deploy_fixture_repo/scripts/test-predeploy-snapshot.py" || exit 1

echo "$deploy_fixture_passed deploy fixtures and the snapshot fixture passed."
