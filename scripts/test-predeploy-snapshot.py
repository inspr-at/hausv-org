#!/usr/bin/env python3
"""Deterministic local fixture test for the pre-deploy snapshot helper."""

from __future__ import annotations

from contextlib import redirect_stdout
import importlib.util
import io
import json
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
from unittest import mock

sys.dont_write_bytecode = True

ROOT = Path(__file__).resolve().parent.parent
HELPER = ROOT / "scripts" / "create-predeploy-snapshot.py"


def load_helper():
    spec = importlib.util.spec_from_file_location("predeploy_snapshot", HELPER)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load snapshot helper")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def run_snapshot(
    module,
    source: Path,
    snapshot: Path,
    compose: Path,
    compose_file: Path,
    target: str,
    *,
    compose_project: str = "hausv",
    postgres_container: str = "none",
    dump_error: bool = False,
    backup_error: bool = False,
    stop_error: bool = False,
    recovery_error: bool = False,
    expected_error: str = "",
):
    argv = [
        str(HELPER),
        "--source",
        str(source),
        "--snapshot",
        str(snapshot),
        "--compose-dir",
        str(compose),
        "--compose-file",
        str(compose_file),
        "--compose-project",
        compose_project,
        "--service",
        "hausv-org",
        "--postgres-container",
        postgres_container,
        "--postgres-user",
        "postgres",
        "--postgres-db",
        "hausv",
        "--source-version",
        "0.57.0",
        "--source-commit",
        "b" * 40,
        "--target-version",
        target,
        "--target-commit",
        "a" * 40,
    ]
    commands: list[tuple[str, ...]] = []
    real_sqlite_backup = module.sqlite_backup
    compose_prefix = (
        "docker",
        "compose",
        "--project-directory",
        str(compose),
        "-p",
        compose_project,
        "-f",
        str(compose_file),
    )

    def fake_run(*args: str, **_kwargs) -> str:
        if _kwargs.get("cwd") is not None:
            raise AssertionError(f"compose mutation relied on cwd: {args}")
        commands.append(args)
        if args[: len(compose_prefix)] != compose_prefix:
            raise AssertionError(f"compose mutation is not canonical: {args}")
        mutation = args[len(compose_prefix) :]
        if stop_error and mutation[:1] == ("stop",):
            raise subprocess.CalledProcessError(1, args)
        if recovery_error and mutation[:1] == ("up",):
            raise subprocess.CalledProcessError(1, args)
        return ""

    def fake_sqlite_backup(source_path: Path, destination_path: Path) -> None:
        if backup_error:
            raise RuntimeError("fixture backup failure")
        real_sqlite_backup(source_path, destination_path)

    dumps: list[tuple[str, str, str, Path]] = []

    def fake_postgres_dump(
        container: str, user: str, database: str, destination: Path
    ) -> tuple[int, str, int]:
        dumps.append((container, user, database, destination))
        if dump_error:
            raise RuntimeError("fixture dump failure")
        archive = b"PGDMP" + bytes((index * 7 + 3) % 256 for index in range(5, 4096))
        destination.write_bytes(archive)
        destination.chmod(0o600)
        return len(archive), "c" * 64, 3

    output = io.StringIO()
    with (
        mock.patch.object(sys, "argv", argv),
        mock.patch.object(module.os, "geteuid", return_value=0),
        mock.patch.object(module, "container_health", return_value="healthy"),
        mock.patch.object(module, "wait_healthy", return_value=True),
        mock.patch.object(module, "run", side_effect=fake_run),
        mock.patch.object(module, "sqlite_backup", side_effect=fake_sqlite_backup),
        mock.patch.object(module, "postgres_dump", side_effect=fake_postgres_dump),
        redirect_stdout(output),
    ):
        try:
            result = module.main()
        except Exception as exc:
            if not expected_error or expected_error not in str(exc):
                raise
            return output.getvalue(), commands + [("dump",) + dump for dump in dumps]
    if expected_error:
        raise AssertionError(f"snapshot helper did not raise {expected_error!r}")
    if result != 0:
        raise AssertionError(f"snapshot helper returned {result}")
    return output.getvalue(), commands + [("dump",) + dump for dump in dumps]


def main() -> int:
    module = load_helper()
    with tempfile.TemporaryDirectory(prefix="hausv-predeploy-fixture-") as raw:
        base = Path(raw)
        source = base / "live" / "hausv-org"
        compose = base / "config" / "docker"
        compose_file = base / "rendered" / "hausv" / "docker-compose.yml"
        snapshot_root = base / "recovery" / "hausv-org-predeploy"
        source.mkdir(parents=True)
        compose.mkdir(parents=True)
        compose_file.parent.mkdir(parents=True)
        compose_file.write_text("services: {}\n", encoding="utf-8")
        (source / "uploads").mkdir()
        (source / "uploads" / "fixture.txt").write_text(
            "blob fixture\n", encoding="utf-8"
        )
        with sqlite3.connect(source / "hausv.db") as database:
            database.execute("CREATE TABLE fixture (value TEXT NOT NULL)")
            database.execute("INSERT INTO fixture VALUES ('consistent')")
            database.commit()

        first = snapshot_root / "0.58.0-aaaaaaa"
        output, commands = run_snapshot(
            module,
            source,
            first,
            compose,
            compose_file,
            "0.58.0",
        )
        expected = {
            f"snapshot-path={first}",
            "snapshot-created=",  # checked by prefix below
            "snapshot-integrity=ok",
            "service-health=healthy",
        }
        lines = set(output.strip().splitlines())
        if not expected.difference({"snapshot-created="}).issubset(lines):
            raise AssertionError(f"incomplete proof: {lines}")
        if not any(line.startswith("snapshot-created=") for line in lines):
            raise AssertionError("snapshot timestamp proof missing")
        if commands[0][-4:] != ("stop", "-t", "30", "hausv-org"):
            raise AssertionError(f"service was not stopped first: {commands}")
        if commands[-1][-2:] != ("start", "hausv-org"):
            raise AssertionError(f"service was not restarted: {commands}")
        if any(command[0] == "dump" for command in commands):
            raise AssertionError("a SQLite-only host took a PostgreSQL dump")
        if (first / "postgres.pgdump").exists():
            raise AssertionError("a SQLite-only recovery point carries a PostgreSQL dump")
        if (first.stat().st_mode & 0o777) != 0o700:
            raise AssertionError("snapshot is not root-only")
        if (snapshot_root.stat().st_mode & 0o777) != 0o700:
            raise AssertionError("snapshot root is not root-only")
        with sqlite3.connect(first / "hausv.db") as database:
            integrity = database.execute("PRAGMA integrity_check").fetchone()
            value = database.execute("SELECT value FROM fixture").fetchone()
        if integrity != ("ok",) or value != ("consistent",):
            raise AssertionError("SQLite recovery point is inconsistent")
        if (first / "uploads" / "fixture.txt").read_text(encoding="utf-8") != (
            "blob fixture\n"
        ):
            raise AssertionError("blob recovery point is incomplete")

        # The release-keyed recovery path is write-once. A retry must not turn
        # a pre-migration snapshot into a copy of later/migrated live data.
        with sqlite3.connect(source / "hausv.db") as database:
            database.execute("UPDATE fixture SET value = 'changed after first snapshot'")
            database.commit()
        _, retry_commands = run_snapshot(
            module,
            source,
            first,
            compose,
            compose_file,
            "0.58.0",
            expected_error="already exists",
        )
        if retry_commands:
            raise AssertionError(
                f"same-path retry touched the service: {retry_commands}"
            )
        with sqlite3.connect(first / "hausv.db") as database:
            preserved = database.execute("SELECT value FROM fixture").fetchone()
        if preserved != ("consistent",):
            raise AssertionError("same-path retry overwrote the recovery point")

        second = snapshot_root / "0.59.0-aaaaaaa"
        run_snapshot(
            module,
            source,
            second,
            compose,
            compose_file,
            "0.59.0",
        )
        if not first.is_dir() or not second.is_dir():
            raise AssertionError("a later release overwrote an older recovery point")

        failed = snapshot_root / "0.60.0-aaaaaaa"
        _, failed_commands = run_snapshot(
            module,
            source,
            failed,
            compose,
            compose_file,
            "0.60.0",
            backup_error=True,
            recovery_error=True,
            expected_error="production service may be unavailable",
        )
        if not any(
            command[-5:]
            == ("up", "-d", "--force-recreate", "--no-deps", "hausv-org")
            for command in failed_commands
        ):
            raise AssertionError("failed snapshot did not attempt service recovery")
        if failed.exists() or failed.with_name(f".{failed.name}.staging").exists():
            raise AssertionError("failed snapshot left a publishable recovery directory")

        partial_stop = snapshot_root / "0.61.0-aaaaaaa"
        _, partial_stop_commands = run_snapshot(
            module,
            source,
            partial_stop,
            compose,
            compose_file,
            "0.61.0",
            stop_error=True,
            expected_error="docker",
        )
        if not any(
            command[-5:]
            == ("up", "-d", "--force-recreate", "--no-deps", "hausv-org")
            for command in partial_stop_commands
        ):
            raise AssertionError(
                "failed stop did not enter the automatic service recovery path"
            )

        invalid_project = snapshot_root / "0.62.0-aaaaaaa"
        _, invalid_project_commands = run_snapshot(
            module,
            source,
            invalid_project,
            compose,
            compose_file,
            "0.62.0",
            compose_project="hausv; docker compose down",
            expected_error="unsafe compose project",
        )
        if invalid_project_commands:
            raise AssertionError("invalid compose project touched the service")

        missing_compose_file = snapshot_root / "0.63.0-aaaaaaa"
        _, missing_file_commands = run_snapshot(
            module,
            source,
            missing_compose_file,
            compose,
            base / "rendered" / "hausv" / "missing-compose.yml",
            "0.63.0",
            expected_error="required compose file is unavailable",
        )
        if missing_file_commands:
            raise AssertionError("missing rendered compose file touched the service")

        missing_compose_dir = snapshot_root / "0.64.0-aaaaaaa"
        _, missing_dir_commands = run_snapshot(
            module,
            source,
            missing_compose_dir,
            base / "missing" / "compose" / "directory",
            compose_file,
            "0.64.0",
            expected_error="required compose directory is unavailable",
        )
        if missing_dir_commands:
            raise AssertionError("missing compose project directory touched the service")

        # HAUSV-562: production runs PostgreSQL. The dump is taken while the
        # service is stopped, lands next to hausv.db in the same write-once
        # recovery point, and is described by the proof lines and metadata.
        with_postgres = snapshot_root / "1.3.4-aaaaaaa"
        output, pg_commands = run_snapshot(
            module,
            source,
            with_postgres,
            compose,
            compose_file,
            "1.3.4",
            postgres_container="hausv-postgres",
        )
        pg_lines = set(output.strip().splitlines())
        for proof in (
            f"postgres-dump={with_postgres / 'postgres.pgdump'}",
            "postgres-dump-bytes=4096",
            "postgres-dump-sha256=" + "c" * 64,
            "postgres-dump-tables=3",
            "service-health=healthy",
        ):
            if proof not in pg_lines:
                raise AssertionError(f"PostgreSQL proof missing {proof!r}: {pg_lines}")
        dump_calls = [command for command in pg_commands if command[0] == "dump"]
        expected_dump = (
            "dump",
            "hausv-postgres",
            "postgres",
            "hausv",
            with_postgres.with_name(".1.3.4-aaaaaaa.staging") / "postgres.pgdump",
        )
        if dump_calls != [expected_dump]:
            raise AssertionError(f"PostgreSQL dump was not taken into staging: {dump_calls}")
        archive = (with_postgres / "postgres.pgdump").read_bytes()
        if not archive.startswith(b"PGDMP") or len(archive) != 4096:
            raise AssertionError("PostgreSQL dump was not published with the recovery point")
        if ((with_postgres / "postgres.pgdump").stat().st_mode & 0o777) != 0o600:
            raise AssertionError("PostgreSQL dump is not root-only")
        pg_metadata = json.loads(
            (with_postgres / "PREDEPLOY-SNAPSHOT.json").read_text(encoding="utf-8")
        )
        if (
            pg_metadata.get("postgres_dump") != "postgres.pgdump"
            or pg_metadata.get("postgres_dump_bytes") != 4096
            or pg_metadata.get("postgres_dump_sha256") != "c" * 64
            or pg_metadata.get("scope") != "pre-schema SQLite, blob and PostgreSQL recovery point"
        ):
            raise AssertionError(f"metadata does not describe the PostgreSQL dump: {pg_metadata}")

        # A failed dump publishes nothing and recovers the service.
        failed_dump = snapshot_root / "1.3.5-aaaaaaa"
        _, failed_dump_commands = run_snapshot(
            module,
            source,
            failed_dump,
            compose,
            compose_file,
            "1.3.5",
            postgres_container="hausv-postgres",
            dump_error=True,
            expected_error="fixture dump failure",
        )
        if not any(
            command[-5:] == ("up", "-d", "--force-recreate", "--no-deps", "hausv-org")
            for command in failed_dump_commands
        ):
            raise AssertionError("failed PostgreSQL dump did not recover the service")
        if failed_dump.exists() or failed_dump.with_name(f".{failed_dump.name}.staging").exists():
            raise AssertionError("failed PostgreSQL dump left a publishable recovery directory")

        unsafe_container = snapshot_root / "1.3.6-aaaaaaa"
        _, unsafe_commands = run_snapshot(
            module,
            source,
            unsafe_container,
            compose,
            compose_file,
            "1.3.6",
            postgres_container="hausv-postgres; docker rm -f hausv-org",
            expected_error="unsafe PostgreSQL container name",
        )
        if unsafe_commands:
            raise AssertionError("unsafe PostgreSQL container name touched the service")

    print("pre-deploy snapshot fixture: ok")
    return 0


raise SystemExit(main())
