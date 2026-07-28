#!/usr/bin/env python3
"""Privacy-safe validation for hausv.org JSON logs.

Reads log lines from stdin, prints counts only, and never echoes log values.
"""

from __future__ import annotations

import collections
import json
import re
import sys
from typing import Any


EMAIL = re.compile(r"(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b")
SENSITIVE_QUERY = re.compile(
    r"(?i)(?:[?&]|\\u0026)(?:token|secret|password|code|client_secret)="
)
RAW_TOKEN_ROUTE = re.compile(r"^/(?:calendar|handover)/[^/{][^/]*")
SENSITIVE_KEYS = {
    "body",
    "chat_id",
    "client_secret",
    "password",
    "passwd",
    "payload",
    "private_key",
    "request_body",
    "secret",
    "token",
}
REQUEST_FIELDS = {
    "bytes",
    "duration_ms",
    "method",
    "request_id",
    "route",
    "status",
    "tenant",
}
VALID_LEVELS = {"DEBUG", "INFO", "WARN", "ERROR"}


def walk(value: Any):
    if isinstance(value, dict):
        for key, item in value.items():
            yield str(key), item
            yield from walk(item)
    elif isinstance(value, list):
        for item in value:
            yield from walk(item)


def main() -> int:
    counts = collections.Counter()
    levels = collections.Counter()
    messages = collections.Counter()
    failures = collections.Counter()

    for raw in sys.stdin:
        if not raw.strip():
            continue
        counts["lines"] += 1
        try:
            event = json.loads(raw)
        except json.JSONDecodeError:
            failures["invalid_json"] += 1
            continue
        if not isinstance(event, dict):
            failures["non_object_json"] += 1
            continue

        counts["valid_json"] += 1
        level = str(event.get("level", ""))
        levels[level] += 1
        if level not in VALID_LEVELS:
            failures["invalid_severity"] += 1

        message = str(event.get("msg", ""))
        messages[message] += 1
        if message == "request":
            counts["requests"] += 1
            if not REQUEST_FIELDS.issubset(event):
                failures["request_fields_missing"] += 1
            if "path" in event or "host" in event:
                failures["legacy_request_fields"] += 1

        encoded = json.dumps(event, ensure_ascii=False)
        if EMAIL.search(encoded):
            failures["plaintext_email"] += 1
        if SENSITIVE_QUERY.search(encoded):
            failures["sensitive_query"] += 1

        for key, value in walk(event):
            normalized = key.strip().lower()
            if normalized in SENSITIVE_KEYS or any(
                marker in normalized
                for marker in ("password", "private_key", "client_secret", "request_body")
            ):
                failures["sensitive_field"] += 1
            if normalized in {"path", "route"} and RAW_TOKEN_ROUTE.match(str(value)):
                failures["raw_token_route"] += 1

    print(
        "JSON-Logprüfung: "
        f"{counts['valid_json']}/{counts['lines']} gültig, "
        f"{counts['requests']} Requests, "
        f"Level INFO={levels['INFO']} WARN={levels['WARN']} ERROR={levels['ERROR']}."
    )
    if messages:
        print(f"Nachrichtentypen: {len(messages)}.")

    if not counts["lines"]:
        print("FEHLER: keine Logzeilen erhalten.", file=sys.stderr)
        return 1
    if failures:
        summary = ", ".join(f"{key}={value}" for key, value in sorted(failures.items()))
        print(f"FEHLER: Datenschutz-/Strukturabweichungen: {summary}.", file=sys.stderr)
        return 1

    print("✓ Struktur, Severity und Datenschutz-Stichprobe ohne Abweichung.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
