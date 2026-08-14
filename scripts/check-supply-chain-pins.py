#!/usr/bin/env python3
"""Fail when a CI or build input can move without the repository changing.

The release gate binds a deployment to one exact commit. That guarantee is
only worth as much as the inputs behind it: a floating `actions/checkout@v6`
or a `FROM golang:1.26.6-alpine` without digest lets the very same commit
build from different bytes tomorrow.

This check is deliberately dumb and textual. It runs in CI and locally, needs
no network, and reports every finding at once instead of stopping at the first.

Update path for the pins it enforces: docs/supply-chain-pins.md
"""

from __future__ import annotations

import pathlib
import re
import sys

REPO = pathlib.Path(__file__).resolve().parents[1]

# A full 40-character commit sha; short shas are ambiguous and rejected.
ACTION_REF = re.compile(r"uses:\s*(?P<ref>[\w.-]+/[\w.-]+(?:/[\w.-]+)*)@(?P<version>\S+)")
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
FROM_LINE = re.compile(r"^\s*FROM\s+(?P<image>\S+)", re.MULTILINE)
NODE_VERSION = re.compile(r"node-version:\s*(?P<version>[^\s#]+)")
EXACT_NODE = re.compile(r"^\d+\.\d+\.\d+$")


def workflow_files() -> list[pathlib.Path]:
    return sorted((REPO / ".github" / "workflows").glob("*.yml"))


def check_actions() -> list[str]:
    problems = []
    for path in workflow_files():
        for number, line in enumerate(path.read_text().splitlines(), start=1):
            match = ACTION_REF.search(line)
            if not match:
                continue
            # Local composite actions (./.github/...) carry no external risk.
            if match.group("ref").startswith("."):
                continue
            version = match.group("version")
            if not FULL_SHA.match(version):
                problems.append(
                    f"{path.relative_to(REPO)}:{number}: action {match.group('ref')} "
                    f"is pinned to '{version}', which can move. Use the full commit sha "
                    f"with a trailing '# <tag>' comment."
                )
    return problems


def check_node() -> list[str]:
    problems = []
    for path in workflow_files():
        for number, line in enumerate(path.read_text().splitlines(), start=1):
            match = NODE_VERSION.search(line)
            if match and not EXACT_NODE.match(match.group("version")):
                problems.append(
                    f"{path.relative_to(REPO)}:{number}: node-version "
                    f"'{match.group('version')}' is not an exact patch version."
                )
    return problems


def check_images() -> list[str]:
    problems = []
    for path in sorted(REPO.glob("Dockerfile*")) + sorted(REPO.glob("*/Dockerfile*")):
        for number, line in enumerate(path.read_text().splitlines(), start=1):
            match = FROM_LINE.match(line)
            if not match:
                continue
            image = match.group("image")
            # A stage alias (FROM build AS x) references an earlier stage.
            if "/" not in image and ":" not in image:
                continue
            if "@sha256:" not in image:
                problems.append(
                    f"{path.relative_to(REPO)}:{number}: base image '{image}' has no "
                    f"sha256 digest. Keep the readable tag and append the digest."
                )
    return problems


def main() -> int:
    problems = check_actions() + check_node() + check_images()
    if not problems:
        print("supply-chain pins: ok — actions, node and base images are immutable")
        return 0
    print("supply-chain pins: FAILED", file=sys.stderr)
    for problem in problems:
        print(f"  {problem}", file=sys.stderr)
    print(
        "\nSee docs/supply-chain-pins.md for how to update a pin deliberately.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
