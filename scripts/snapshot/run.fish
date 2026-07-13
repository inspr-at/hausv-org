#!/usr/bin/env fish
# Snapshot one build of the app.
#
#   run.fish <git-ref|WORKTREE> <out-dir> [port]
#
# Builds the given ref in a throwaway worktree (or the current tree if given
# WORKTREE), boots it with the deterministic snapshot env against a freshly
# seeded data dir, captures every page, then tears everything down.
#
# The data dir is seeded IDENTICALLY per run, so two builds see the same state.
# Without that, the pages differ for reasons that have nothing to do with code.

set -l ref $argv[1]
set -l out $argv[2]
set -l port $argv[3]
test -n "$port"; or set port 8099
test -n "$ref" -a -n "$out"; or begin
    echo "usage: run.fish <git-ref|WORKTREE> <out-dir> [port]" >&2
    exit 1
end

set -l repo (git rev-parse --show-toplevel)
set -l tmp (mktemp -d /tmp/hv-snap.XXXXXX)
set -l src

if test "$ref" = WORKTREE
    set src $repo
else
    set src $tmp/src
    git -C $repo worktree add -q --detach $src $ref; or exit 1
end

set -gx HV_PORT $port
set -gx HV_DATA $tmp/data
mkdir -p $HV_DATA

# Build. Note: after the cmd/ split this must target the main package's dir, so
# resolve it rather than assuming the repo root.
set -l pkg .
test -d $src/cmd/hausv-org; and set pkg ./cmd/hausv-org

echo "── building $ref ($pkg)"
env -C $src go build -o $tmp/app $pkg; or begin
    echo "build failed" >&2
    exit 1
end

source $repo/scripts/snapshot/env.fish

echo "── booting on :$port"
env -C $src $tmp/app >$tmp/app.log 2>&1 &
set -l pid $last_pid

# Wait for readiness rather than sleeping blindly.
set -l ready 0
for i in (seq 60)
    if curl -sf "http://localhost:$port/healthz" >/dev/null 2>&1
        set ready 1
        break
    end
    sleep 0.25
end
if test $ready -eq 0
    echo "app did not become healthy; log:" >&2
    cat $tmp/app.log >&2
    kill $pid 2>/dev/null
    exit 1
end

echo "── capturing"
node $repo/scripts/snapshot/capture.mjs "http://localhost:$port" $out
set -l rc $status

kill $pid 2>/dev/null
if test "$ref" != WORKTREE
    git -C $repo worktree remove --force $src 2>/dev/null
end
rm -rf $tmp
exit $rc
