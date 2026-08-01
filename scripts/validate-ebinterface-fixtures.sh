#!/usr/bin/env bash
# Reproducible operator check for HAUSV's synthetic ebInterface fixtures.
# Deliberately accepts no file arguments: real invoices must never be uploaded
# to the public validator by this script.
#
# Targets bash 3.2 so it runs on a stock macOS /bin/bash as well as on CI.

set -euo pipefail

if [ "$#" -ne 0 ]; then
    echo "usage: scripts/validate-ebinterface-fixtures.sh" >&2
    echo "Only the synthetic repository fixtures are submitted." >&2
    exit 2
fi

repo=$(git rev-parse --show-toplevel) || exit 1
validator=https://labs.ebinterface.at/
scratch=$(mktemp -d) || exit 1

cleanup_ebinterface_validator() {
    if [ -n "${scratch:-}" ] && [ -d "$scratch" ]; then
        command rm -r -- "$scratch"
    fi
}
trap cleanup_ebinterface_validator EXIT

for spec in \
    "5.0|$repo/internal/integrations/testdata/ebinterface-5p0.xml" \
    "6.0|$repo/internal/integrations/testdata/ebinterface-6p0.xml"; do
    profile_version=${spec%%|*}
    fixture=${spec#*|}
    cookie="$scratch/cookies-$profile_version.txt"

    if ! page=$(curl -fsSL -c "$cookie" "$validator"); then
        echo "ebInterface $profile_version: validator page unavailable" >&2
        exit 1
    fi
    # The form tag sits on a single line, so a line-based extraction matches
    # what the fish version's regex did.
    action=$(printf '%s\n' "$page" \
        | grep -o '<form id="id1"[^>]*action="[^"]*"' \
        | head -1 \
        | sed 's/.*action="\([^"]*\)"$/\1/')
    if [ -z "$action" ]; then
        echo "ebInterface $profile_version: validator form not found" >&2
        exit 1
    fi
    action=${action//&amp;/&}
    action=${action#./}

    if ! response=$(curl -fsSL -b "$cookie" -c "$cookie" \
        -F "fileInput=@$fixture;type=application/xml" \
        "$validator$action&submitButtonSchemaOnly=x"); then
        echo "ebInterface $profile_version: validator request failed" >&2
        exit 1
    fi
    if [[ $response != *"Diese Datei ist gültig gemäß ebInterface Standard"*"ebInterface $profile_version"* ]]; then
        echo "ebInterface $profile_version: official schema validation failed" >&2
        exit 1
    fi
    echo "ebInterface $profile_version: official schema validation passed"
done
