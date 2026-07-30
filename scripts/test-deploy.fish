#!/usr/bin/env fish
# Deterministic fail-closed release fixtures. Never contacts production.

set -g deploy_fixture_repo (builtin cd (dirname (status filename))/..; and pwd)
set -g deploy_fixture_mock $deploy_fixture_repo/scripts/testdata/deploy/mock-command
set -g deploy_fixture_root (mktemp -d -t hausv-deploy-fixture.XXXXXX)
or exit 1

function cleanup --on-event fish_exit
    if string match -qr '^/.*hausv-deploy-fixture\.' -- $deploy_fixture_root
        command rm -rf -- $deploy_fixture_root
    end
end

mkdir -p $deploy_fixture_root/bin
or exit 1
for command_name in git gh curl ssh sleep
    ln -s $deploy_fixture_mock $deploy_fixture_root/bin/$command_name
    or exit 1
end

set -g deploy_fixture_passed 0

function contains_fail_fast_remote_command --argument-names output
    set -l escaped_commands (string match -rg \
        '(?:mandatory recovery command|image rollback command): ssh -p [^[:space:]]+ [^[:space:]]+ (.+)' \
        -- "$output")
    for escaped_command in $escaped_commands
        set -l decoded_command (string unescape --style=script -- "$escaped_command")
        if string match -q '/bin/sh -eu -c *' -- "$decoded_command"
            return 0
        end
    end
    return 1
end

function fixture --argument-names case_name expected_status expected_text mode
    set -l state $deploy_fixture_root/$case_name
    mkdir -p $state
    set -l output $state/output.txt
    set -l fixture_path (string join : $deploy_fixture_root/bin $PATH)
    set -l args
    if test "$mode" = dry-run
        set args --dry-run
    end

    env \
        PATH=$fixture_path \
        DEPLOY_FIXTURE_CASE=$case_name \
        DEPLOY_FIXTURE_STATE=$state \
        DEPLOY_FIXTURE_REPO=$deploy_fixture_repo \
        HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
        HAUSV_DEPLOY_VERIFY_SLEEP=0 \
        fish --no-config $deploy_fixture_repo/scripts/deploy.fish $args >$output 2>&1
    set -l actual_status $status

    if test $actual_status -ne $expected_status
        echo "FAIL $case_name: exit $actual_status, expected $expected_status" >&2
        command cat $output >&2
        exit 1
    end
    if not string match -q "*$expected_text*" -- (string collect <$output)
        echo "FAIL $case_name: missing '$expected_text'" >&2
        command cat $output >&2
        exit 1
    end
    set deploy_fixture_passed (math $deploy_fixture_passed + 1)
    echo "ok $case_name"
end

fixture no_schema 0 "all fail-closed preconditions passed" dry-run
set -l dry_log (string collect <$deploy_fixture_root/no_schema/commands.log)
for forbidden in "docker build" "docker tag" "--source /var/lib/csb1-docker/hausv-org"
    if string match -q "*$forbidden*" -- $dry_log
        echo "FAIL no_schema: dry-run invoked mutating command '$forbidden'" >&2
        exit 1
    end
end

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
set -l retry_log (string collect <$deploy_fixture_root/retry_after_activation_fail/commands.log)
for forbidden in "docker build" "docker tag"
    if string match -q "*$forbidden*" -- $retry_log
        echo "FAIL retry_after_activation_fail: retry invoked '$forbidden' despite image identity mismatch" >&2
        exit 1
    end
end
fixture rollback_tag_conflict 1 "remote preflight failed" dry-run
fixture schema_snapshot_fail 1 "fresh consistent pre-deploy snapshot failed" release
set -l snapshot_failure_log (string collect <$deploy_fixture_root/schema_snapshot_fail/commands.log)
if string match -q "*docker build*" -- $snapshot_failure_log
    echo "FAIL schema_snapshot_fail: build ran without a recovery point" >&2
    exit 1
end
fixture schema_snapshot_recovery_fail 1 "mandatory recovery command:" release
set -l recovery_failure_output (string collect <$deploy_fixture_root/schema_snapshot_recovery_fail/output.txt)
if not contains_fail_fast_remote_command "$recovery_failure_output"
    echo "FAIL schema_snapshot_recovery_fail: recovery command is not fail-fast" >&2
    exit 1
end

fixture preserve_fail 1 "could not preserve the currently running image" release
fixture build_fail 1 "release image build failed" release
fixture activation_fail 1 "image rollback command:" release
set -l activation_failure_output (string collect <$deploy_fixture_root/activation_fail/output.txt)
if not contains_fail_fast_remote_command "$activation_failure_output"
    echo "FAIL activation_fail: rollback command is not fail-fast" >&2
    exit 1
end
fixture post_health_fail 1 "image rollback command:" release
fixture post_version_mismatch 1 "post-deploy health/version verification failed" release
fixture schema_startlog_fail 1 "data/schema restore required" release
set -l schema_failure_output (string collect <$deploy_fixture_root/schema_startlog_fail/output.txt)
set -l fixture_app_version (string trim (command cat $deploy_fixture_repo/VERSION))
if not string match -q "*/hausv-org-predeploy/$fixture_app_version-aaaaaaa*" -- $schema_failure_output
    echo "FAIL schema_startlog_fail: exact versioned recovery path missing" >&2
    exit 1
end

fixture success 0 "critical start logs: clean" release
set -l success_output (string collect <$deploy_fixture_root/success/output.txt)
if not contains_fail_fast_remote_command "$success_output"
    echo "FAIL success: printed rollback command is not fail-fast" >&2
    exit 1
end
fixture schema_success 0 "pre-deploy recovery point: fresh, consistent and healthy" release

python3 $deploy_fixture_repo/scripts/test-predeploy-snapshot.py
or exit 1

echo "$deploy_fixture_passed deploy fixtures and the snapshot fixture passed."
