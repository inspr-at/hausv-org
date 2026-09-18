#!/usr/bin/env python3
"""Install deterministic token hashes for an isolated handover QA record.

Raw fixture tokens arrive on stdin and are never written to the database. The
product stores only URL-safe SHA-256 hashes, exactly like production creation.

HAUSV-757: talks to PostgreSQL via DATABASE_URL. Local throwaway databases
are reached with docker exec; CI uses the pinned client image on host network.
"""

from __future__ import annotations

import base64
import hashlib
import json
import os
import subprocess
import sys
from urllib.parse import unquote, urlparse

POSTGRES_IMAGE = (
    "postgres:17.6-alpine3.22@sha256:"
    "ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94"
)


def token_hash(token: str) -> str:
    digest = hashlib.sha256(
        ("handover-confirmation-v1:" + token.strip()).encode("utf-8")
    ).digest()
    return base64.urlsafe_b64encode(digest).decode("ascii").rstrip("=")


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def run_psql(sql: str) -> str:
    container = os.environ.get("HAUSV_EPHEMERAL_PG_CONTAINER", "").strip()
    if container:
        command = [
            "docker",
            "exec",
            "-i",
            container,
            "psql",
            "-U",
            "hausv",
            "-d",
            "hausv",
            "-v",
            "ON_ERROR_STOP=1",
            "-t",
            "-A",
            "-c",
            sql,
        ]
        env = os.environ
    else:
        dsn = os.environ.get("DATABASE_URL", "").strip()
        if not dsn:
            print("DATABASE_URL is required", file=sys.stderr)
            raise SystemExit(2)
        parsed = urlparse(dsn)
        password = unquote(parsed.password or "")
        command = [
            "docker",
            "run",
            "--rm",
            "--network",
            "host",
            "--env",
            f"PGPASSWORD={password}",
            POSTGRES_IMAGE,
            "psql",
            "--host",
            parsed.hostname or "127.0.0.1",
            "--port",
            str(parsed.port or 5432),
            "--username",
            unquote(parsed.username or "hausv"),
            "--dbname",
            (parsed.path or "/hausv").lstrip("/") or "hausv",
            "-v",
            "ON_ERROR_STOP=1",
            "-t",
            "-A",
            "-c",
            sql,
        ]
        env = os.environ
    result = subprocess.run(command, check=False, capture_output=True, text=True, env=env)
    if result.returncode != 0:
        print(result.stderr or result.stdout, file=sys.stderr)
        raise SystemExit(1)
    return result.stdout


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: set-handover-fixture-tokens.py <handover-id>", file=sys.stderr)
        return 2

    payload = json.load(sys.stdin)
    tokens = payload.get("tokens", [])
    if len(tokens) != 2 or any(not isinstance(token, str) or not token.strip() for token in tokens):
        print("exactly two non-empty fixture tokens are required", file=sys.stderr)
        return 2

    handover_id = sys.argv[1]
    raw = run_psql(
        "SELECT data FROM handovers WHERE tenant_slug='demo' AND id="
        + sql_literal(handover_id)
    ).strip()
    if not raw:
        print("handover fixture record not found", file=sys.stderr)
        return 1
    record = json.loads(raw)
    confirmations = record.get("confirmations", [])
    if len(confirmations) != 2:
        print("handover fixture must have two confirmations", file=sys.stderr)
        return 1
    for index, token in enumerate(tokens):
        confirmations[index]["token_hash"] = token_hash(token)
    encoded = json.dumps(record, ensure_ascii=False, separators=(",", ":"))
    updated = run_psql(
        "UPDATE handovers SET data="
        + sql_literal(encoded)
        + " WHERE tenant_slug='demo' AND id="
        + sql_literal(handover_id)
        + " RETURNING 1"
    ).strip()
    if updated != "1":
        print("handover fixture token update did not affect one row", file=sys.stderr)
        return 1

    print("1")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
