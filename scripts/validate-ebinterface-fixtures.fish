#!/usr/bin/env fish
# Reproducible operator check for HAUSV's synthetic ebInterface fixtures.
# Deliberately accepts no file arguments: real invoices must never be uploaded
# to the public validator by this script.

if test (count $argv) -ne 0
    echo "usage: scripts/validate-ebinterface-fixtures.fish" >&2
    echo "Only the synthetic repository fixtures are submitted." >&2
    exit 2
end

set -l repo (git rev-parse --show-toplevel); or exit 1
set -l validator https://labs.ebinterface.at/
set -l scratch (mktemp -d); or exit 1

function cleanup_ebinterface_validator --on-event fish_exit
    if test -n "$scratch" -a -d "$scratch"
        command rm -r -- "$scratch"
    end
end

for spec in \
    "5.0|$repo/internal/integrations/testdata/ebinterface-5p0.xml" \
    "6.0|$repo/internal/integrations/testdata/ebinterface-6p0.xml"
    set -l fields (string split '|' -- $spec)
    set -l profile_version $fields[1]
    set -l fixture $fields[2]
    set -l cookie "$scratch/cookies-$profile_version.txt"

    set -l page (curl -fsSL -c "$cookie" "$validator" | string collect); or begin
        echo "ebInterface $profile_version: validator page unavailable" >&2
        exit 1
    end
    set -l action (string match -rg '<form id="id1"[^>]*action="([^"]+)"' -- "$page" | head -1)
    test -n "$action"; or begin
        echo "ebInterface $profile_version: validator form not found" >&2
        exit 1
    end
    set action (string replace -a '&amp;' '&' -- "$action")
    set action (string replace -r '^\./' '' -- "$action")

    set -l response (curl -fsSL -b "$cookie" -c "$cookie" \
        -F "fileInput=@$fixture;type=application/xml" \
        "$validator$action&submitButtonSchemaOnly=x" | string collect); or begin
        echo "ebInterface $profile_version: validator request failed" >&2
        exit 1
    end
    if not string match -q "*Diese Datei ist gültig gemäß ebInterface Standard*ebInterface $profile_version*" -- "$response"
        echo "ebInterface $profile_version: official schema validation failed" >&2
        exit 1
    end
    echo "ebInterface $profile_version: official schema validation passed"
end
