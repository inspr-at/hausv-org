#!/usr/bin/env bash
# Deterministic fail-closed release fixtures. Never contacts production.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -u

deploy_fixture_repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
deploy_fixture_mock=$deploy_fixture_repo/scripts/testdata/deploy/mock-command
deploy_fixture_root=$(mktemp -d -t hausv-deploy-fixture.XXXXXX) || exit 1
deploy_fixture_lock_prefix="/run/current-system/sw/bin/flock -w 300 /run/lock/compose-hausv.lock /bin/sh -eu"
deploy_fixture_base64_bin="/run/current-system/sw/bin/base64"
deploy_fixture_mktemp_bin="/run/current-system/sw/bin/mktemp"
deploy_fixture_compose_command="docker compose --project-directory /srv/hausv/compose -p hausv -f /srv/hausv/compose/docker-compose.yml"

cleanup() {
    case $deploy_fixture_root in
        /*hausv-deploy-fixture.*) command rm -rf -- "$deploy_fixture_root" ;;
    esac
}
trap cleanup EXIT

mkdir -p "$deploy_fixture_root/bin" || exit 1
for command_name in git gh curl ssh sleep base64; do
    ln -s "$deploy_fixture_mock" "$deploy_fixture_root/bin/$command_name" || exit 1
done

deploy_fixture_passed=0

contains_fail_fast_remote_command() {
    local output=$1 line escaped decoded inner transport encoded body
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
        inner=${decoded#"/bin/sh -eu -c "}
        [ "$inner" != "$decoded" ] || continue
        transport=$(eval "printf '%s' $inner" 2>/dev/null) || continue
        case $transport in
            "locked_script=\$("*"$deploy_fixture_mktemp_bin /tmp/hausv-locked.XXXXXX"*"; printf %s "*" | $deploy_fixture_base64_bin -d >\"\$locked_script\"; $deploy_fixture_lock_prefix \"\$locked_script\"") ;;
            *) continue ;;
        esac
        encoded=${transport#*"; printf %s "}
        encoded=${encoded%%" | "*}
        body=$(printf '%s' "$encoded" | base64 -d 2>/dev/null) || continue
        case $line in
            *"image rollback command: ssh -p "*)
                case $body in
                    *"docker tag"*" && $deploy_fixture_compose_command up"*) return 0 ;;
                esac
                ;;
            *"mandatory recovery command: ssh -p "*)
                case $body in
                    *"$deploy_fixture_compose_command up"*) return 0 ;;
                esac
                ;;
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
            HAUSV_DEPLOY_SSH_HOST="deployer@production.example.invalid" \
            HAUSV_DEPLOY_SSH_PORT=2222 \
            HAUSV_DEPLOY_COMPOSE_DIR="/srv/hausv/compose" \
            HAUSV_DEPLOY_COMPOSE_FILE="/srv/hausv/compose/docker-compose.yml" \
            HAUSV_DEPLOY_COMPOSE_PROJECT=hausv \
            HAUSV_DEPLOY_CONTAINER=hausv-demo \
            HAUSV_DEPLOY_COMPOSE_LOCK="/run/lock/compose-hausv.lock" \
            HAUSV_DEPLOY_FLOCK_BIN="/run/current-system/sw/bin/flock" \
            HAUSV_DEPLOY_BASE64_BIN="$deploy_fixture_base64_bin" \
            HAUSV_DEPLOY_MKTEMP_BIN="$deploy_fixture_mktemp_bin" \
            HAUSV_DEPLOY_DATA_DIR="/var/lib/hausv" \
            HAUSV_DEPLOY_SNAPSHOT_ROOT="/var/backups/hausv-predeploy" \
            HAUSV_DEPLOY_POSTGRES_CONTAINER="${DEPLOY_FIXTURE_POSTGRES-hausv-postgres}" \
            HAUSV_DEPLOY_POSTGRES_USER=hausv_backup \
            HAUSV_DEPLOY_LIVE_URL="https://portal.example.invalid/demo/" \
            HAUSV_DEPLOY_VERIFY_ATTEMPTS=1 \
            HAUSV_DEPLOY_VERIFY_SLEEP=0 \
            bash "$deploy_fixture_repo/scripts/deploy.sh" --dry-run >"$output" 2>&1
    else
        env \
            PATH="$fixture_path" \
            DEPLOY_FIXTURE_CASE="$case_name" \
            DEPLOY_FIXTURE_STATE="$state" \
            DEPLOY_FIXTURE_REPO="$deploy_fixture_repo" \
            HAUSV_DEPLOY_SSH_HOST="deployer@production.example.invalid" \
            HAUSV_DEPLOY_SSH_PORT=2222 \
            HAUSV_DEPLOY_COMPOSE_DIR="/srv/hausv/compose" \
            HAUSV_DEPLOY_COMPOSE_FILE="/srv/hausv/compose/docker-compose.yml" \
            HAUSV_DEPLOY_COMPOSE_PROJECT=hausv \
            HAUSV_DEPLOY_CONTAINER=hausv-demo \
            HAUSV_DEPLOY_COMPOSE_LOCK="/run/lock/compose-hausv.lock" \
            HAUSV_DEPLOY_FLOCK_BIN="/run/current-system/sw/bin/flock" \
            HAUSV_DEPLOY_BASE64_BIN="$deploy_fixture_base64_bin" \
            HAUSV_DEPLOY_MKTEMP_BIN="$deploy_fixture_mktemp_bin" \
            HAUSV_DEPLOY_DATA_DIR="/var/lib/hausv" \
            HAUSV_DEPLOY_SNAPSHOT_ROOT="/var/backups/hausv-predeploy" \
            HAUSV_DEPLOY_POSTGRES_CONTAINER="${DEPLOY_FIXTURE_POSTGRES-hausv-postgres}" \
            HAUSV_DEPLOY_POSTGRES_USER=hausv_backup \
            HAUSV_DEPLOY_LIVE_URL="https://portal.example.invalid/demo/" \
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
if ! grep -qF -- "$deploy_fixture_lock_prefix" \
    "$deploy_fixture_root/no_schema/commands.log" \
    || ! grep -qF -- "$deploy_fixture_base64_bin -d" \
    "$deploy_fixture_root/no_schema/commands.log"; then
    echo "FAIL no_schema: preflight did not use the encoded locked transport" >&2
    exit 1
fi
for forbidden in "git archive" "docker tag" "--source /var/lib/hausv"; do
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
for forbidden in "git archive" "docker tag"; do
    if grep -qF -- "$forbidden" "$deploy_fixture_root/retry_after_activation_fail/commands.log"; then
        echo "FAIL retry_after_activation_fail: retry invoked '$forbidden' despite image identity mismatch" >&2
        exit 1
    fi
done
fixture rollback_tag_conflict 1 "remote preflight failed" dry-run
fixture lock_timeout 1 "remote preflight failed" dry-run
fixture missing_rendered_compose 1 "remote preflight failed" dry-run
fixture wrong_compose_identity 1 "remote preflight failed" dry-run
fixture preflight_visible_drift 1 "remote preflight failed" dry-run
fixture encoder_fail 1 "cannot encode the locked remote preflight" dry-run
fixture schema_snapshot_identity_drift 1 "fresh consistent pre-deploy snapshot failed" release
if grep -qF -- "git archive" "$deploy_fixture_root/schema_snapshot_identity_drift/commands.log"; then
    echo "FAIL schema_snapshot_identity_drift: pull ran after snapshot identity changed" >&2
    exit 1
fi
fixture schema_snapshot_fail 1 "fresh consistent pre-deploy snapshot failed" release
if grep -qF -- "git archive" "$deploy_fixture_root/schema_snapshot_fail/commands.log"; then
    echo "FAIL schema_snapshot_fail: pull ran without a recovery point" >&2
    exit 1
fi
fixture schema_snapshot_recovery_fail 1 "mandatory recovery command:" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/schema_snapshot_recovery_fail/output.txt")"; then
    echo "FAIL schema_snapshot_recovery_fail: recovery command is not fail-fast" >&2
    exit 1
fi
fixture schema_snapshot_recovery_encoder_fail 1 "locked mandatory recovery shell:" release
if grep -qF -- "mandatory recovery command: ssh" \
    "$deploy_fixture_root/schema_snapshot_recovery_encoder_fail/output.txt" \
    || ! grep -qF -- "ssh -tt -p 2222 deployer@production.example.invalid" \
    "$deploy_fixture_root/schema_snapshot_recovery_encoder_fail/output.txt" \
    || ! grep -qF -- "inside that locked shell, mandatory recovery command: $deploy_fixture_compose_command up" \
    "$deploy_fixture_root/schema_snapshot_recovery_encoder_fail/output.txt"; then
    echo "FAIL schema_snapshot_recovery_encoder_fail: attended recovery fallback is incomplete" >&2
    exit 1
fi

fixture preserve_fail 1 "could not preserve the currently running image" release
fixture preserve_identity_drift 1 "could not preserve the currently running image" release
if grep -qF -- "git archive" "$deploy_fixture_root/preserve_identity_drift/commands.log"; then
    echo "FAIL preserve_identity_drift: pull ran after preflight image identity changed" >&2
    exit 1
fi
fixture ghcr_token_missing 1 "CI image pull failed" release
if ! grep -qF -- "GHCR token file is not readable" "$deploy_fixture_root/ghcr_token_missing/output.txt"; then
    echo "FAIL ghcr_token_missing: token file error not reported in stderr" >&2
    exit 1
fi
fixture ghcr_login_fail 1 "CI image pull failed" release
fixture build_fail 1 "CI image pull failed" release
fixture activation_fail 1 "image rollback command:" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/activation_fail/output.txt")"; then
    echo "FAIL activation_fail: rollback command is not fail-fast" >&2
    exit 1
fi
fixture rollback_encoder_fail 1 "locked image rollback shell:" release
if grep -qF -- "image rollback command: ssh" \
    "$deploy_fixture_root/rollback_encoder_fail/output.txt" \
    || ! grep -qF -- "ssh -tt -p 2222 deployer@production.example.invalid" \
    "$deploy_fixture_root/rollback_encoder_fail/output.txt" \
    || ! grep -qF -- "inside that locked shell, image rollback command: docker tag" \
    "$deploy_fixture_root/rollback_encoder_fail/output.txt" \
    || ! grep -qF -- " && $deploy_fixture_compose_command up" \
    "$deploy_fixture_root/rollback_encoder_fail/output.txt"; then
    echo "FAIL rollback_encoder_fail: attended rollback fallback is incomplete" >&2
    exit 1
fi
fixture activation_mktemp_fail 1 "activation transport setup failed before lock and retag" release
if grep -qF -- "image rollback command:" \
    "$deploy_fixture_root/activation_mktemp_fail/output.txt"; then
    echo "FAIL activation_mktemp_fail: pre-mutation staging failure suggested a rollback" >&2
    exit 1
fi
fixture activation_decode_fail 1 "activation transport setup failed before lock and retag" release
if grep -qF -- "image rollback command:" \
    "$deploy_fixture_root/activation_decode_fail/output.txt"; then
    echo "FAIL activation_decode_fail: pre-mutation decode failure suggested a rollback" >&2
    exit 1
fi
fixture activation_lock_timeout 1 "activation lock was unavailable before retag" release
if grep -qF -- "image rollback command:" \
    "$deploy_fixture_root/activation_lock_timeout/output.txt"; then
    echo "FAIL activation_lock_timeout: pre-mutation lock conflict suggested a rollback" >&2
    exit 1
fi
fixture activation_identity_drift 1 "activation identity changed before retag" release
if grep -qF -- "image rollback command:" \
    "$deploy_fixture_root/activation_identity_drift/output.txt"; then
    echo "FAIL activation_identity_drift: pre-mutation drift suggested a rollback" >&2
    exit 1
fi
fixture activation_post_tag_raw_42 1 "image rollback command:" release
if ! contains_fail_fast_remote_command "$(cat "$deploy_fixture_root/activation_post_tag_raw_42/output.txt")"; then
    echo "FAIL activation_post_tag_raw_42: remapped post-tag failure omitted rollback" >&2
    exit 1
fi
fixture post_health_fail 1 "image rollback command:" release
fixture post_version_mismatch 1 "post-deploy health/version verification failed" release
fixture schema_startlog_fail 1 "data/schema restore required" release
fixture_app_version=$(tr -d '[:space:]' < "$deploy_fixture_repo/VERSION")
if ! grep -qF -- "/var/backups/hausv-predeploy/$fixture_app_version-aaaaaaa" \
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
# HAUSV-562: the attended path dumps PostgreSQL into the same recovery point
# and tells the operator how to restore it.
if ! grep -qF -- "recovery point scope: SQLite + blobs + PostgreSQL dump" \
    "$deploy_fixture_root/schema_success/output.txt" \
    || ! grep -qF -- "PostgreSQL restore (service stopped): docker exec -i hausv-postgres pg_restore -U postgres --clean --if-exists --exit-on-error -d hausv < /var/backups/hausv-predeploy/$fixture_app_version-aaaaaaa/postgres.pgdump" \
    "$deploy_fixture_root/schema_success/output.txt"; then
    echo "FAIL schema_success: PostgreSQL recovery scope or restore guidance missing" >&2
    exit 1
fi
if grep -qF -- "pg_restore -U hausv_backup" "$deploy_fixture_root/schema_success/output.txt"; then
    echo "FAIL schema_success: rollback guidance names the read-only dump role for the restore (HAUSV-638)" >&2
    exit 1
fi
# The database container is a required setting for a schema release; a host
# that runs SQLite only says so with "none" and takes no dump.
DEPLOY_FIXTURE_POSTGRES="" fixture schema_postgres_unset 1 "HAUSV_DEPLOY_POSTGRES_CONTAINER is required" dry-run
if grep -qF -- "ssh" "$deploy_fixture_root/schema_postgres_unset/commands.log"; then
    echo "FAIL schema_postgres_unset: contacted the host without a stated recovery scope" >&2
    exit 1
fi
DEPLOY_FIXTURE_POSTGRES=none fixture schema_sqlite_only 0 "recovery point: SQLite + blobs" dry-run
if grep -qF -- "PostgreSQL" "$deploy_fixture_root/schema_sqlite_only/output.txt"; then
    echo "FAIL schema_sqlite_only: a SQLite-only host was promised a PostgreSQL dump" >&2
    exit 1
fi

python3 "$deploy_fixture_repo/scripts/test-predeploy-snapshot.py" || exit 1

echo "$deploy_fixture_passed deploy fixtures and the snapshot fixture passed."
