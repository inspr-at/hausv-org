#!/usr/bin/env bash
set -euo pipefail

base_url=${HAUSV_DEMO_BASE_URL:-"http://localhost:${HAUSV_DEMO_PORT:-8098}"}
vera_email=vera.verwalter@musterstadt.example
secrets_file=${HAUSV_DEMO_SECRETS_FILE:-"$(dirname "$0")/secrets.env"}
if [ -f "$secrets_file" ]; then set -a; . "$secrets_file"; set +a; fi
cookie_jar=$(mktemp "${TMPDIR:-/tmp}/hausv-demo-verify.XXXXXX")
trap 'rm -f "$cookie_jar"' EXIT

pass=0
fail=0

check() {
    local label=$1
    shift
    if "$@"; then
        printf 'PASS  %s\n' "$label"
        pass=$((pass + 1))
    else
        printf 'FAIL  %s\n' "$label" >&2
        fail=$((fail + 1))
    fi
}

healthcheck() {
    curl --fail --silent --show-error "$base_url/healthz" | grep -F '"status":"ok"' >/dev/null
}

login_vera() {
    local page href target
    page=$(curl --fail --silent --show-error --cookie-jar "$cookie_jar" \
        --header "Origin: $base_url" --data-urlencode "email=$vera_email" \
        --data-urlencode "access_code=${DEMO_LOGIN_ACCESS_CODE:-}" "$base_url/auth/request")
    href=$(printf '%s' "$page" | sed -n 's/.*<a class="dev-link" href="\([^"]*\)".*/\1/p')
    [ -n "$href" ]
    href=$(printf '%s' "$href" | sed 's/&amp;/\&/g')
    case $href in
        http://*|https://*) target=$href ;;
        /*) target=$base_url$href ;;
        *) return 1 ;;
    esac
    curl --fail --silent --show-error --location --cookie "$cookie_jar" --cookie-jar "$cookie_jar" "$target" >/dev/null
}

# Consume the complete response: grep -q can close the pipe early and turn a
# successful match into SIGPIPE (141) under pipefail on larger portal pages.
verwaltung_page() {
    local page
    page=$(curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/app/verwaltung")
    printf '%s' "$page" | grep -F 'Hausverwaltung Musterstadt GmbH' >/dev/null
    printf '%s' "$page" | grep -F 'Janusbergweg 123' >/dev/null
}

inbox_count() {
    # The Posteingang summarises its queue as "N offen · …"; a seeded demo
    # must report at least one open item.
    local page text
    page=$(curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/app/verwaltung/posteingang")
    text=$(printf '%s' "$page" | tr '\n' ' ' | sed 's/<[^>]*>/ /g')
    printf '%s' "$text" | grep -E '(^|[^0-9])[1-9][0-9]* offen' >/dev/null
}

check 'healthz' healthcheck
check 'Vera dev-login' login_vera
check 'Verwaltung: Organisation und Haus' verwaltung_page
check 'Posteingang: offene Anzahl' inbox_count

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
