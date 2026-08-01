#!/usr/bin/env python3
"""Install deterministic token hashes for an isolated handover QA record.

Raw fixture tokens arrive on stdin and are never written to the database. The
product stores only URL-safe SHA-256 hashes, exactly like production creation.
"""

from __future__ import annotations

import base64
import hashlib
import json
import sqlite3
import sys


def token_hash(token: str) -> str:
    digest = hashlib.sha256(
        ("handover-confirmation-v1:" + token.strip()).encode("utf-8")
    ).digest()
    return base64.urlsafe_b64encode(digest).decode("ascii").rstrip("=")


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: set-handover-fixture-tokens.py <db-path> <handover-id>", file=sys.stderr)
        return 2

    payload = json.load(sys.stdin)
    tokens = payload.get("tokens", [])
    if len(tokens) != 2 or any(not isinstance(token, str) or not token.strip() for token in tokens):
        print("exactly two non-empty fixture tokens are required", file=sys.stderr)
        return 2

    connection = sqlite3.connect(sys.argv[1], timeout=5)
    try:
        row = connection.execute(
            "SELECT data FROM handovers WHERE tenant_slug=? AND id=?",
            ("jhw22", sys.argv[2]),
        ).fetchone()
        if row is None:
            print("handover fixture record not found", file=sys.stderr)
            return 1
        record = json.loads(row[0])
        confirmations = record.get("confirmations", [])
        if len(confirmations) != 2:
            print("handover fixture must have two confirmations", file=sys.stderr)
            return 1
        for index, token in enumerate(tokens):
            confirmations[index]["token_hash"] = token_hash(token)
        encoded = json.dumps(record, ensure_ascii=False, separators=(",", ":"))
        cursor = connection.execute(
            "UPDATE handovers SET data=? WHERE tenant_slug=? AND id=?",
            (encoded, "jhw22", sys.argv[2]),
        )
        connection.commit()
        if cursor.rowcount != 1:
            print("handover fixture token update did not affect one row", file=sys.stderr)
            return 1
    finally:
        connection.close()

    print("1")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
