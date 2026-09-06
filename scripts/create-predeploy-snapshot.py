#!/usr/bin/env python3
"""Create one atomic, root-only HAUSV pre-schema recovery point.

The caller streams this committed file to hausv and runs it with the system
Python as root while holding the host's compose lock for the helper's complete
lifetime. Compose mutations deliberately do not reacquire that non-reentrant
lock. The helper never prints database rows, filenames from user content, or
secret/config values. The application is stopped while SQLite and blob files
are captured as one recovery point — and, when the host runs PostgreSQL, a
custom-format pg_dump of the production database taken in that same window
(HAUSV-562), so the recovery point never describes a database the files do
not match.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import subprocess
import sys
import time


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--source", required=True)
    parser.add_argument("--snapshot", required=True)
    parser.add_argument("--compose-dir", required=True)
    parser.add_argument("--compose-file", required=True)
    parser.add_argument("--compose-project", required=True)
    parser.add_argument("--service", required=True)
    parser.add_argument("--postgres-container", required=True)
    parser.add_argument("--postgres-user", default="postgres")
    parser.add_argument("--postgres-db", default="hausv")
    parser.add_argument("--source-version", required=True)
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--target-version", required=True)
    parser.add_argument("--target-commit", required=True)
    return parser.parse_args()


def checked_directory(raw: str, *, forbidden: set[Path], label: str) -> Path:
    path = Path(raw)
    if (
        not path.is_absolute()
        or path in forbidden
        or len(path.parts) < 4
        or ".." in path.parts
    ):
        raise RuntimeError(f"unsafe {label} path")
    return path


def checked_file(raw: str, *, label: str) -> Path:
    path = Path(raw)
    if not path.is_absolute() or len(path.parts) < 4 or ".." in path.parts:
        raise RuntimeError(f"unsafe {label} path")
    if not path.is_file():
        raise RuntimeError(f"required {label} is unavailable")
    return path


def checked_compose_project(raw: str) -> str:
    if re.fullmatch(r"[a-z0-9][a-z0-9_-]*", raw) is None:
        raise RuntimeError("unsafe compose project")
    return raw


def checked_postgres_container(raw: str) -> str:
    """The database container name, or "" when the host runs SQLite only."""
    if raw == "none":
        return ""
    if re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]*", raw) is None:
        raise RuntimeError("unsafe PostgreSQL container name")
    return raw


def checked_postgres_identifier(raw: str, *, label: str) -> str:
    if re.fullmatch(r"[A-Za-z0-9_]+", raw) is None:
        raise RuntimeError(f"unsafe PostgreSQL {label}")
    return raw


POSTGRES_DUMP_NAME = "postgres.pgdump"
POSTGRES_DUMP_MAGIC = b"PGDMP"


def postgres_dump(
    container: str, user: str, database: str, destination: Path
) -> tuple[int, str, int]:
    """Dump the production database into the staging directory.

    pg_dump runs inside the database container over its local socket, so no
    credential leaves the host configuration. The archive is verified by its
    header and by listing it with pg_restore; the count of TABLE DATA entries
    is the restorability witness printed as proof. Returns (bytes, sha256,
    tables).
    """
    with destination.open("wb") as archive:
        subprocess.run(
            [
                "docker",
                "exec",
                container,
                "pg_dump",
                "-U",
                user,
                "--format=custom",
                database,
            ],
            check=True,
            stdout=archive,
            stderr=subprocess.DEVNULL,
        )
    os.chmod(destination, 0o600)
    digest = hashlib.sha256()
    size = 0
    with destination.open("rb") as archive:
        header = archive.read(len(POSTGRES_DUMP_MAGIC))
        if header != POSTGRES_DUMP_MAGIC:
            raise RuntimeError("PostgreSQL dump is not a custom-format pg_dump archive")
        digest.update(header)
        size += len(header)
        while True:
            chunk = archive.read(1 << 20)
            if not chunk:
                break
            digest.update(chunk)
            size += len(chunk)
    with destination.open("rb") as archive:
        listing = subprocess.run(
            ["docker", "exec", "-i", container, "pg_restore", "--list"],
            check=True,
            stdin=archive,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
        ).stdout
    tables = sum(1 for line in listing.splitlines() if "TABLE DATA" in line)
    if tables < 1:
        raise RuntimeError("PostgreSQL dump lists no table data")
    return size, digest.hexdigest(), tables


def run(*args: str, capture: bool = False) -> str:
    result = subprocess.run(
        args,
        check=True,
        text=True,
        stdout=subprocess.PIPE if capture else subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return result.stdout.strip() if capture else ""


def run_compose_mutation(
    compose_dir: Path,
    compose_project: str,
    compose_file: Path,
    *mutation: str,
) -> None:
    run(
        "docker",
        "compose",
        "--project-directory",
        str(compose_dir),
        "-p",
        compose_project,
        "-f",
        str(compose_file),
        *mutation,
    )


def container_health(service: str) -> str:
    return run(
        "docker",
        "inspect",
        "--format",
        "{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}",
        service,
        capture=True,
    )


def wait_healthy(service: str, attempts: int = 18) -> bool:
    for _ in range(attempts):
        try:
            if container_health(service) == "healthy":
                return True
        except subprocess.CalledProcessError:
            pass
        time.sleep(5)
    return False


def sqlite_backup(source: Path, destination: Path) -> None:
    source_uri = f"{source.resolve().as_uri()}?mode=ro"
    with sqlite3.connect(source_uri, uri=True) as source_db:
        with sqlite3.connect(destination) as destination_db:
            source_db.backup(destination_db)
            result = destination_db.execute("PRAGMA integrity_check").fetchone()
            if result is None or result[0] != "ok":
                raise RuntimeError("snapshot integrity check failed")


def main() -> int:
    args = parse_args()
    source = checked_directory(
        args.source, forbidden={Path("/")}, label="source directory"
    )
    snapshot = checked_directory(
        args.snapshot, forbidden={Path("/"), source}, label="snapshot"
    )
    compose_dir = checked_directory(
        args.compose_dir, forbidden={Path("/")}, label="compose directory"
    )
    compose_file = checked_file(args.compose_file, label="compose file")
    compose_project = checked_compose_project(args.compose_project)
    postgres_container = checked_postgres_container(args.postgres_container)
    postgres_user = checked_postgres_identifier(args.postgres_user, label="role")
    postgres_db = checked_postgres_identifier(args.postgres_db, label="database")
    if snapshot.is_relative_to(source) or source.is_relative_to(snapshot):
        raise RuntimeError("source and snapshot paths must be separate")
    staging = snapshot.with_name(f".{snapshot.name}.staging")
    database = source / "hausv.db"
    if not source.is_dir() or not database.is_file():
        raise RuntimeError("required HAUSV source is unavailable")
    if not compose_dir.is_dir():
        raise RuntimeError("required compose directory is unavailable")
    if os.geteuid() != 0:
        raise RuntimeError("snapshot helper requires root")
    if snapshot.exists():
        raise RuntimeError(
            "snapshot path already exists; the existing recovery point was preserved"
        )
    if container_health(args.service) != "healthy":
        raise RuntimeError("HAUSV must be healthy before snapshot")
    snapshot.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(snapshot.parent, 0o700)

    needs_service_recovery = False
    published = False
    created = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
    try:
        # A stop command can partially take effect before returning non-zero.
        # Arm recovery before invoking it so every attempted quiesce has the
        # same automatic fail-closed recovery path.
        needs_service_recovery = True
        run_compose_mutation(
            compose_dir,
            compose_project,
            compose_file,
            "stop",
            "-t",
            "30",
            args.service,
        )

        shutil.rmtree(staging, ignore_errors=True)
        staging.mkdir(mode=0o700, parents=False)
        shutil.copytree(source, staging, dirs_exist_ok=True, symlinks=True)
        for suffix in ("", "-wal", "-shm"):
            (staging / f"hausv.db{suffix}").unlink(missing_ok=True)
        sqlite_backup(database, staging / "hausv.db")

        metadata = {
            "created_utc": created.isoformat().replace("+00:00", "Z"),
            "source_version": args.source_version,
            "source_commit": args.source_commit,
            "target_version": args.target_version,
            "target_commit": args.target_commit,
            "sqlite_integrity": "ok",
            "scope": "pre-schema SQLite and blob recovery point",
        }
        postgres_proof: dict[str, object] = {}
        if postgres_container:
            dump_bytes, dump_sha256, dump_tables = postgres_dump(
                postgres_container,
                postgres_user,
                postgres_db,
                staging / POSTGRES_DUMP_NAME,
            )
            metadata["scope"] = "pre-schema SQLite, blob and PostgreSQL recovery point"
            metadata["postgres_dump"] = POSTGRES_DUMP_NAME
            metadata["postgres_dump_bytes"] = dump_bytes
            metadata["postgres_dump_sha256"] = dump_sha256
            postgres_proof = {
                "postgres-dump": snapshot / POSTGRES_DUMP_NAME,
                "postgres-dump-bytes": dump_bytes,
                "postgres-dump-sha256": dump_sha256,
                "postgres-dump-tables": dump_tables,
            }
        (staging / "PREDEPLOY-SNAPSHOT.json").write_text(
            json.dumps(metadata, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
        os.chmod(staging, 0o700)
        os.sync()

        # The target is versioned by release and is deliberately write-once.
        # rename() refuses a non-empty existing directory; the explicit check
        # above gives a clearer error before the service is stopped.
        os.rename(staging, snapshot)
        published = True

        run_compose_mutation(
            compose_dir,
            compose_project,
            compose_file,
            "start",
            args.service,
        )
        if not wait_healthy(args.service):
            raise RuntimeError("HAUSV did not become healthy after snapshot")
        needs_service_recovery = False
    finally:
        recovery_cause: Exception | None = None
        if needs_service_recovery:
            try:
                run_compose_mutation(
                    compose_dir,
                    compose_project,
                    compose_file,
                    "up",
                    "-d",
                    "--force-recreate",
                    "--no-deps",
                    args.service,
                )
                if not wait_healthy(args.service):
                    raise RuntimeError(
                        "HAUSV recovery command completed but service stayed unhealthy"
                    )
            except Exception as exc:
                recovery_cause = exc
        if not published:
            shutil.rmtree(staging, ignore_errors=True)
        if recovery_cause is not None:
            raise RuntimeError(
                "HAUSV recovery failed; production service may be unavailable"
            ) from recovery_cause

    print(f"snapshot-path={snapshot}")
    print(f"snapshot-created={metadata['created_utc']}")
    print("snapshot-integrity=ok")
    for key, value in postgres_proof.items():
        print(f"{key}={value}")
    print("service-health=healthy")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"pre-deploy snapshot failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
