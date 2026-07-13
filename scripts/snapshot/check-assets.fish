#!/usr/bin/env fish
# Verify every embedded asset is actually SERVED, with the right bytes.
#
# The snapshot oracle cannot catch a broken asset: a 404 on /assets/app.js does
# not change the HTML (the <script> tag is still emitted) and does not change
# the screenshots (the CSS is inline in the templates, and the JS is behavioural).
# So after moving assets/ into internal/web for //go:embed, this is the check
# that actually proves the move worked.

set -l port 8097
set -l repo (git rev-parse --show-toplevel)
set -l tmp (mktemp -d /tmp/hv-assets.XXXXXX)

set -gx HV_PORT $port
set -gx HV_DATA $tmp/data
mkdir -p $HV_DATA
source $repo/scripts/snapshot/env.fish

set -l pkg .
test -d $repo/cmd/hausv-org; and set pkg ./cmd/hausv-org
go build -o $tmp/app $pkg; or exit 1

$tmp/app >$tmp/app.log 2>&1 &
set -l pid $last_pid

set -l ready 0
for i in (seq 60)
    if curl -sf "http://localhost:$port/healthz" >/dev/null 2>&1
        set ready 1
        break
    end
    sleep 0.25
end
if test $ready -eq 0
    echo "app did not start:" >&2
    cat $tmp/app.log >&2
    kill $pid 2>/dev/null
    rm -rf $tmp
    exit 1
end

set -l fail 0
for f in (ls $repo/internal/web/assets)
    set -l code (curl -s -o /dev/null -w '%{http_code}' "http://localhost:$port/assets/$f")
    set -l served (curl -s "http://localhost:$port/assets/$f" | wc -c | string trim)
    set -l disk (wc -c < $repo/internal/web/assets/$f | string trim)
    if test "$code" = 200 -a "$served" = "$disk"
        printf '  ✓ %-24s %s  %s bytes == disk\n' $f $code $served
    else
        printf '  ✗ %-24s %s  served=%s disk=%s\n' $f $code $served $disk
        set fail 1
    end
end

kill $pid 2>/dev/null
rm -rf $tmp
if test $fail -eq 1
    echo "  ✗ EMBEDDED ASSETS BROKEN"
    exit 1
end
echo "  ✓ all embedded assets served byte-identical to disk"
