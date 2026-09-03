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
    curl --fail --silent --show-error "$base_url/healthz" | grep -q '"status":"ok"'
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

verwaltung_page() {
    local page
    page=$(curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/app/verwaltung")
    printf '%s' "$page" | grep -q 'Hausverwaltung Musterstadt GmbH'
    printf '%s' "$page" | grep -q 'Janischhofweg 22'
}

inbox_count() {
    # The sidebar carries the open count as a badge on the Posteingang item;
    # a seeded demo must show at least one open item there.
    local page compact item
    page=$(curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/app/verwaltung/posteingang")
    compact=$(printf '%s' "$page" | tr '\n' ' ')
    item=${compact#*'href="/app/verwaltung/posteingang"'}
    [ "$item" != "$compact" ] || return 1
    item=${item%%'</a>'*}
    printf '%s' "$item" | grep -Eq 'class="nav-badge">[1-9][0-9]*<'
}

check 'healthz' healthcheck
check 'Vera dev-login' login_vera
check 'Verwaltung: Organisation und Haus' verwaltung_page
check 'Posteingang: offene Anzahl' inbox_count

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
