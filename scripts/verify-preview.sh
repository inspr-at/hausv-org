#!/usr/bin/env bash
# Does the deployed preview slot actually work — can you log in, and does every
# authenticated route render templ?
#
#   ssh csb1 'bash -s' < scripts/verify-preview.sh [base-url] [tenant]
#
# Runs ON the preview host, because the mail sink is only reachable from inside
# the app container's network namespace (compose uses network_mode: service:app),
# so the magic link cannot be collected from anywhere else.
#
# Written after a deploy that was healthy, answered on the tailnet, carried
# TEMPL_PORTAL_ENABLED=true in its env file — and could not be logged into by
# anyone, because the SMTP fixture was never started (NIX-371). Three pieces of
# evidence all pointing the right way, none of which meant the slot was usable.
#
# Never prints the magic link: it is a working credential for the slot.

set -u

BASE=${1:-http://100.64.0.4:8099}
TENANT=${2:-demo}
USER_EMAIL=${VERIFY_EMAIL:-admin@example.com}

ROUTES="/app /app/announcements /app/events /app/anliegen /app/anliegen/board
/app/dokumente /app/abstimmungen /app/uebergaben /app/kontakte /app/hilfe
/app/parking /app/zuhause/onboarding /app/settings /app/settings/profile
/app/settings/notifications /app/settings/building /app/settings/users /app/audit"

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

jar=$(mktemp) || exit 1
trap 'rm -f "$jar"' EXIT

code=$(curl -s -m 10 -o /dev/null -w '%{http_code}' "$BASE/healthz") || fail "cannot reach $BASE"
[ "$code" = 200 ] || fail "healthz returned $code"

curl -s -m 10 -c "$jar" -b "$jar" -o /dev/null "$BASE/$TENANT/" || fail "landing page unreachable"

# The app rejects a cross-origin POST, which is why a bare curl gets 403 here.
curl -s -m 10 -c "$jar" -b "$jar" -o /dev/null \
  -H "Origin: $BASE" -H "Referer: $BASE/$TENANT/" \
  -X POST "$BASE/$TENANT/auth/request" --data-urlencode "email=$USER_EMAIL" \
  || fail "auth request failed"

sink=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -- '-fixture-smtp-' | head -1)
[ -n "$sink" ] || fail "no fixture-smtp container is running, so the magic link goes nowhere and nobody can log in (NIX-371)"

link=""
for _ in 1 2 3 4 5 6 7 8 9 10; do
    link=$(docker exec "$sink" node -e '
      fetch("http://127.0.0.1:8124/messages").then(r => r.json()).then(list => {
        const body = list.map(m => m.body).join("\n");
        const urls = body.match(/https?:\/\/[^\s<>"]+/g) || [];
        const hit = urls.reverse().find(u => u.includes("/auth/"));
        process.stdout.write(hit || "");
      }).catch(() => process.stdout.write(""))' 2>/dev/null)
    [ -n "$link" ] && break
    sleep 2
done
[ -n "$link" ] || fail "no magic link arrived in the mail sink within 20s"

curl -sL -s -m 10 -c "$jar" -b "$jar" -o /dev/null "$link" || fail "magic link did not establish a session"

bad=0
total=0
for route in $ROUTES; do
    total=$((total + 1))
    body=$(curl -s -m 10 -b "$jar" "$BASE/$TENANT$route" 2>/dev/null)
    marker=$(printf '%s' "$body" | grep -o 'data-templ-[a-z-]*' | head -1)
    auth=$(printf '%s' "$body" | grep -c 'data-authenticated-app')
    if [ -z "$marker" ]; then
        printf '  %-30s LEGACY or not authenticated\n' "$route"
        bad=$((bad + 1))
    elif [ "$auth" -eq 0 ]; then
        printf '  %-30s %s but no data-authenticated-app\n' "$route" "$marker"
        bad=$((bad + 1))
    else
        printf '  %-30s %s\n' "$route" "$marker"
    fi
done

echo
if [ "$bad" -ne 0 ]; then
    fail "$bad of $total routes did not render templ with an authenticated body"
fi
echo "PASS — logged in and all $total routes render templ"
