#!/usr/bin/env python3
"""Reacreditación aislada y reproducible de SQLite V16 a V19."""

from __future__ import annotations

import argparse
from collections import Counter
import datetime as dt
import hashlib
import json
import os
import re
import secrets
import select
import signal
import shutil
import socket
import sqlite3
import stat
import subprocess
import sys
import time
import tomllib
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable, Sequence


SCHEMA = "orquesta_sqlite_upgrade_audit.v1"
PROGRAM = "orquesta_sqlite_v23_reaccredit"
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
REVISION_RE = re.compile(r"^[0-9a-f]{40}$")
PROFILE_RE = re.compile(r"^SqliteReaccredit[0-9a-f]{16}$")
EXCLUDED_FUNCTIONAL_TABLES = frozenset(
    {
        "schema_migrations",
        "command_invocations",
        "command_outcomes",
        "authorization_receipts",
    }
)
AUDIT_TABLES = (
    "command_invocations",
    "command_outcomes",
    "authorization_receipts",
)
CORE_TABLES = ("goals", "executions", "work_items", "outbox", "effect_attempts")
MARKERS = {
    "@@ORQUESTA_REACCREDIT_RUNTIME_ROOT@@": "runtime_root",
    "@@ORQUESTA_REACCREDIT_ACCOUNT_ROOT@@": "account_root",
    "@@ORQUESTA_REACCREDIT_PROFILE@@": "profile",
    "@@ORQUESTA_REACCREDIT_LISTEN@@": "listen",
    "@@ORQUESTA_REACCREDIT_CGROUP_ROOT@@": "cgroup_root",
    "@@ORQUESTA_REACCREDIT_REPOSITORY_SEED@@": "repository_seed",
    "@@ORQUESTA_REACCREDIT_BUBBLEWRAP@@": "bubblewrap",
}
MIGRATION_NAMES = {
    17: "017_intake.sql",
    18: "018_intake_dossiers.sql",
    19: "019_intake_dossier_confirmations.sql",
}


class HarnessError(RuntimeError):
    def __init__(self, code: str, detail: str = "") -> None:
        super().__init__(code)
        self.code = code
        self.detail = detail[:2048]


def fail(code: str, detail: str = "") -> None:
    raise HarnessError(code, detail)


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    descriptor = os.open(path, os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW)
    try:
        with os.fdopen(descriptor, "rb", closefd=False) as handle:
            while chunk := handle.read(1024 * 1024):
                digest.update(chunk)
    finally:
        os.close(descriptor)
    return digest.hexdigest()


def canonical_existing(path_text: str, kind: str) -> Path:
    path = Path(path_text)
    if not path.is_absolute() or "\n" in path_text or "\r" in path_text:
        fail(f"{kind}_path_invalid")
    try:
        metadata = path.lstat()
        resolved = path.resolve(strict=True)
    except OSError as error:
        fail(f"{kind}_unavailable", str(error))
    if stat.S_ISLNK(metadata.st_mode) or resolved != path:
        fail(f"{kind}_symlink_or_noncanonical")
    return path


def require_directory(path_text: str, kind: str) -> Path:
    path = canonical_existing(path_text, kind)
    metadata = path.stat(follow_symlinks=False)
    if not stat.S_ISDIR(metadata.st_mode):
        fail(f"{kind}_not_directory")
    if metadata.st_mode & 0o002:
        fail(f"{kind}_world_writable")
    return path


def require_source_file(
    path_text: str,
    expected_sha: str,
    kind: str,
    *,
    executable: bool = False,
    private: bool = False,
) -> Path:
    if not SHA_RE.fullmatch(expected_sha):
        fail(f"{kind}_expected_sha256_invalid")
    path = canonical_existing(path_text, kind)
    metadata = path.stat(follow_symlinks=False)
    if not stat.S_ISREG(metadata.st_mode):
        fail(f"{kind}_not_regular")
    if metadata.st_uid not in {0, os.geteuid()}:
        fail(f"{kind}_owner_invalid")
    mode = stat.S_IMODE(metadata.st_mode)
    if executable and not mode & stat.S_IXUSR:
        fail(f"{kind}_not_executable")
    if mode & 0o002:
        fail(f"{kind}_world_writable")
    if private and (metadata.st_uid != os.geteuid() or mode not in {0o400, 0o600}):
        fail(f"{kind}_not_private")
    if private and metadata.st_nlink != 1:
        fail(f"{kind}_link_count_invalid")
    observed = sha256_file(path)
    if observed != expected_sha:
        fail(f"{kind}_hash_mismatch", f"expected={expected_sha} observed={observed}")
    return path


def is_within(path: Path, root: Path) -> bool:
    try:
        path.relative_to(root)
    except ValueError:
        return False
    return True


def reject_live_path(path: Path, live_root: Path, kind: str) -> None:
    lowered = [part.casefold() for part in path.parts]
    if is_within(path, live_root) or "codex12" in lowered:
        fail(f"{kind}_points_to_live_codex12")


def ensure_output_target(path_text: str, live_root: Path) -> Path:
    path = Path(path_text)
    if not path.is_absolute() or "\n" in path_text or "\r" in path_text:
        fail("output_dir_path_invalid")
    if path.exists() or path.is_symlink():
        fail("output_dir_already_exists")
    parent = require_directory(str(path.parent), "output_parent")
    if path.parent.resolve(strict=True) != parent:
        fail("output_parent_noncanonical")
    if path.name in {"", ".", ".."}:
        fail("output_dir_path_invalid")
    reject_live_path(path, live_root, "output_dir")
    return path


def write_private(path: Path, content: bytes) -> None:
    descriptor = os.open(
        path,
        os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC | os.O_NOFOLLOW,
        0o600,
    )
    try:
        view = memoryview(content)
        while view:
            written = os.write(descriptor, view)
            view = view[written:]
        os.fsync(descriptor)
    finally:
        os.close(descriptor)
    os.chmod(path, 0o600, follow_symlinks=False)


def copy_private(source: Path, target: Path, mode: int) -> None:
    descriptor_in = os.open(
        source, os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW
    )
    descriptor_out = os.open(
        target,
        os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC | os.O_NOFOLLOW,
        mode,
    )
    try:
        while chunk := os.read(descriptor_in, 1024 * 1024):
            view = memoryview(chunk)
            while view:
                written = os.write(descriptor_out, view)
                view = view[written:]
        os.fsync(descriptor_out)
    finally:
        os.close(descriptor_in)
        os.close(descriptor_out)
    os.chmod(target, mode, follow_symlinks=False)


@dataclass(frozen=True)
class GuardedFile:
    label: str
    path: Path
    sha256: str


class IntegrityGuard:
    def __init__(self, files: Iterable[GuardedFile]) -> None:
        self.files = tuple(files)

    def verify(self) -> None:
        for item in self.files:
            try:
                observed = sha256_file(item.path)
            except OSError as error:
                fail(f"{item.label}_unreadable", str(error))
            if observed != item.sha256:
                fail(
                    f"{item.label}_drift",
                    f"expected={item.sha256} observed={observed}",
                )


def choose_loopback() -> str:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
    return f"127.0.0.1:{port}"


def nested(value: dict[str, Any], *parts: str) -> Any:
    current: Any = value
    for part in parts:
        if not isinstance(current, dict) or part not in current:
            fail("materialized_config_missing_key", ".".join(parts))
        current = current[part]
    return current


def materialize_config(
    template: Path,
    target: Path,
    runtime_root: Path,
    account_root: Path,
    profile: str,
    listen: str,
    cgroup_root: Path,
    repository_seed: Path,
    bubblewrap: Path,
    forbidden_live_root: Path,
) -> tuple[str, dict[str, Path]]:
    try:
        text = template.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as error:
        fail("config_template_read_failed", str(error))
    values = {
        "runtime_root": str(runtime_root),
        "account_root": str(account_root),
        "profile": profile,
        "listen": listen,
        "cgroup_root": str(cgroup_root),
        "repository_seed": str(repository_seed),
        "bubblewrap": str(bubblewrap),
    }
    for marker, name in MARKERS.items():
        if marker not in text:
            fail("config_template_marker_missing", marker)
        text = text.replace(marker, values[name])
    if "@@ORQUESTA_REACCREDIT_" in text:
        fail("config_template_unknown_marker")
    encoded = text.encode("utf-8")
    write_private(target, encoded)
    try:
        config = tomllib.loads(text)
    except tomllib.TOMLDecodeError as error:
        fail("materialized_config_invalid_toml", str(error))
    state_path = runtime_root / "state" / "orquesta.sqlite"
    exact = {
        ("server", "listen"): listen,
        ("state", "sqlite", "path"): str(state_path),
        ("runtime", "codex", "account_home_root"): str(account_root),
        ("runtime", "codex", "account_profile"): profile,
        ("runtime", "codex", "cgroup_root"): str(cgroup_root),
        ("repository", "local", "seed_path"): str(repository_seed),
        ("test_attestor", "provider"): "bubblewrap",
        ("test_attestor", "bubblewrap", "command"): str(bubblewrap),
        ("test_attestor", "resources", "cgroup_root"): str(cgroup_root),
    }
    for key, expected in exact.items():
        if nested(config, *key) != expected:
            fail(
                "materialized_config_binding_invalid",
                f"{'.'.join(key)} expected={expected!r}",
            )
    isolated_path_keys = (
        ("state", "sqlite", "path"),
        ("artifact", "filesystem", "root"),
        ("credentials", "local", "path"),
        ("runtime", "codex", "work_root"),
        ("runtime", "codex", "cache_root"),
        ("workspace", "local", "root"),
        ("identity", "local_token_path"),
        ("config", "effective_path"),
    )
    isolated_paths: dict[str, Path] = {}
    for key in isolated_path_keys:
        raw = nested(config, *key)
        name = ".".join(key)
        if not isinstance(raw, str) or not raw:
            fail("materialized_config_path_invalid", name)
        path = Path(raw)
        if not path.is_absolute() or path != Path(os.path.normpath(path)):
            fail("materialized_config_path_invalid", name)
        if not is_within(path, runtime_root):
            fail("materialized_config_path_outside_runtime", name)
        reject_live_path(path, forbidden_live_root, "materialized_config")
        isolated_paths[name] = path
    command_path_keys = (
        ("runtime", "codex", "command"),
        ("runtime", "codex", "go_toolchain_root"),
        ("test_attestor", "go", "toolchain_root"),
    )
    for key in command_path_keys:
        raw = nested(config, *key)
        name = ".".join(key)
        if not isinstance(raw, str):
            fail("materialized_config_path_invalid", name)
        if not raw:
            continue
        path = Path(raw)
        if not path.is_absolute() or path != Path(os.path.normpath(path)):
            fail("materialized_config_path_invalid", name)
        reject_live_path(path, forbidden_live_root, "materialized_config")
    microvm = config.get("test_attestor", {}).get("microvm", {})
    launcher = microvm.get("launcher_socket", "") if isinstance(microvm, dict) else ""
    if launcher:
        launcher_path = Path(launcher)
        if not launcher_path.is_absolute() or not is_within(
            launcher_path, runtime_root
        ):
            fail("materialized_config_launcher_not_isolated")
        if launcher_path.exists() or launcher_path.is_symlink():
            fail("materialized_config_launcher_exists")
    serialized = json.dumps(config, sort_keys=True, default=str).casefold()
    if "codex12" in serialized:
        fail("materialized_config_references_live_codex12")
    return sha256_bytes(encoded), isolated_paths


def sqlite_connect_readonly(path: Path) -> sqlite3.Connection:
    uri = "file:" + str(path) + "?mode=ro"
    connection = sqlite3.connect(uri, uri=True)
    connection.execute("PRAGMA query_only = ON")
    return connection


def encode_sqlite_value(value: Any) -> dict[str, Any]:
    if value is None:
        return {"type": "null", "value": None}
    if isinstance(value, bytes):
        return {"type": "blob", "value": value.hex()}
    if isinstance(value, bool):
        return {"type": "integer", "value": int(value)}
    if isinstance(value, int):
        return {"type": "integer", "value": value}
    if isinstance(value, float):
        return {"type": "real", "value": value.hex()}
    if isinstance(value, str):
        return {"type": "text", "value": value}
    fail("sqlite_value_type_unsupported", type(value).__name__)


def quote_identifier(value: str) -> str:
    return '"' + value.replace('"', '""') + '"'


def table_names(connection: sqlite3.Connection) -> list[str]:
    rows = connection.execute(
        """
        SELECT name FROM sqlite_schema
        WHERE type='table' AND name NOT LIKE 'sqlite_%'
        ORDER BY name
        """
    ).fetchall()
    return [str(row[0]) for row in rows]


def sqlite_schema_manifest(connection: sqlite3.Connection) -> list[dict[str, Any]]:
    rows = connection.execute(
        """
        SELECT type,name,tbl_name,sql
        FROM sqlite_schema
        ORDER BY type,name,tbl_name,sql
        """
    ).fetchall()
    return [
        {
            "type": str(row[0]),
            "name": str(row[1]),
            "table": str(row[2]),
            "sql": None if row[3] is None else str(row[3]),
        }
        for row in rows
    ]


def table_columns(connection: sqlite3.Connection, table: str) -> list[str]:
    rows = connection.execute(
        f"PRAGMA table_info({quote_identifier(table)})"
    ).fetchall()
    columns = [str(row[1]) for row in rows]
    if not columns:
        fail("sqlite_table_columns_missing", table)
    return columns


def functional_manifest(
    connection: sqlite3.Connection,
) -> list[dict[str, Any]]:
    return [
        {"table": table, "columns": table_columns(connection, table)}
        for table in table_names(connection)
        if table not in EXCLUDED_FUNCTIONAL_TABLES
    ]


def functional_digest(
    connection: sqlite3.Connection, manifest: Sequence[dict[str, Any]]
) -> str:
    available = set(table_names(connection))
    digest = hashlib.sha256()
    for entry in manifest:
        table = entry["table"]
        columns = list(entry["columns"])
        if table not in available:
            fail("functional_table_missing", table)
        current_columns = set(table_columns(connection, table))
        if not set(columns).issubset(current_columns):
            fail("functional_column_missing", table)
        header = json.dumps(
            {"table": table, "columns": columns},
            ensure_ascii=False,
            separators=(",", ":"),
            sort_keys=True,
        ).encode("utf-8")
        digest.update(len(header).to_bytes(8, "big"))
        digest.update(header)
        selected = ",".join(quote_identifier(column) for column in columns)
        ordering = ",".join(
            f"{quote_identifier(column)} IS NULL,{quote_identifier(column)}"
            for column in columns
        )
        query = (
            f"SELECT {selected} FROM {quote_identifier(table)} "
            f"ORDER BY {ordering}"
        )
        for row in connection.execute(query):
            payload = json.dumps(
                [encode_sqlite_value(value) for value in row],
                ensure_ascii=False,
                separators=(",", ":"),
                sort_keys=True,
                allow_nan=False,
            ).encode("utf-8")
            digest.update(len(payload).to_bytes(8, "big"))
            digest.update(payload)
    return digest.hexdigest()


def table_row_fingerprints(
    connection: sqlite3.Connection, table: str
) -> list[str]:
    columns = table_columns(connection, table)
    selected = ",".join(quote_identifier(column) for column in columns)
    result: list[str] = []
    for row in connection.execute(
        f"SELECT {selected} FROM {quote_identifier(table)}"
    ):
        encoded = json.dumps(
            [encode_sqlite_value(value) for value in row],
            ensure_ascii=False,
            separators=(",", ":"),
            sort_keys=True,
            allow_nan=False,
        ).encode("utf-8")
        result.append(sha256_bytes(encoded))
    return sorted(result)


def migration_rows(connection: sqlite3.Connection) -> list[dict[str, Any]]:
    try:
        rows = connection.execute(
            """
            SELECT version,name,checksum FROM schema_migrations
            ORDER BY version,name,checksum
            """
        ).fetchall()
    except sqlite3.Error as error:
        fail("schema_migrations_unreadable", str(error))
    return [
        {"version": int(row[0]), "name": str(row[1]), "checksum": str(row[2])}
        for row in rows
    ]


def validate_migrations(
    rows: Sequence[dict[str, Any]],
    expected: dict[int, dict[str, str]],
    final_version: int,
) -> None:
    if len(rows) != final_version:
        fail(
            "schema_migration_count_invalid",
            f"expected={final_version} observed={len(rows)}",
        )
    seen: set[int] = set()
    for row in rows:
        version = int(row["version"])
        if version in seen or version not in expected:
            fail("schema_migration_version_invalid", str(version))
        seen.add(version)
        if (
            row["name"] != expected[version]["name"]
            or row["checksum"] != expected[version]["checksum"]
        ):
            fail("schema_migration_identity_invalid", str(version))
    if seen != set(range(1, final_version + 1)):
        fail("schema_migration_sequence_invalid")


def sqlite_snapshot(
    path: Path,
    expected_migrations: dict[int, dict[str, str]],
    baseline_manifest: Sequence[dict[str, Any]] | None = None,
) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    try:
        connection = sqlite_connect_readonly(path)
    except sqlite3.Error as error:
        fail("sqlite_open_failed", str(error))
    try:
        quick_rows = connection.execute("PRAGMA quick_check").fetchall()
        quick = [str(row[0]) for row in quick_rows]
        foreign_keys = connection.execute("PRAGMA foreign_key_check").fetchall()
        user_version = int(connection.execute("PRAGMA user_version").fetchone()[0])
        tables = table_names(connection)
        for required in (*CORE_TABLES, *AUDIT_TABLES, "schema_migrations"):
            if required not in tables:
                fail("required_table_missing", required)
        counts = {
            table: int(
                connection.execute(
                    f"SELECT COUNT(*) FROM {quote_identifier(table)}"
                ).fetchone()[0]
            )
            for table in tables
        }
        migrations = migration_rows(connection)
        manifest = (
            list(baseline_manifest)
            if baseline_manifest is not None
            else functional_manifest(connection)
        )
        digest = functional_digest(connection, manifest)
        status_refs = [
            str(row[0])
            for row in connection.execute(
                """
                SELECT ref FROM command_invocations
                WHERE command_id='orquesta.system.status'
                ORDER BY ref
                """
            ).fetchall()
        ]
        audit_row_fingerprints = {
            table: table_row_fingerprints(connection, table)
            for table in AUDIT_TABLES
        }
        schema_manifest = sqlite_schema_manifest(connection)
    except sqlite3.Error as error:
        fail("sqlite_snapshot_failed", str(error))
    finally:
        connection.close()
    if quick != ["ok"]:
        fail("sqlite_quick_check_failed", repr(quick))
    if foreign_keys:
        fail("sqlite_foreign_key_check_failed", repr(foreign_keys[:10]))
    if user_version not in {16, 19}:
        fail("sqlite_user_version_invalid", str(user_version))
    validate_migrations(migrations, expected_migrations, user_version)
    migration_counts = {
        str(version): sum(
            1 for row in migrations if int(row["version"]) == version
        )
        for version in (17, 18, 19)
    }
    return (
        {
            "user_version": user_version,
            "quick_check": quick[0],
            "foreign_key_violations": 0,
            "functional_digest": digest,
            "table_counts": counts,
            "core_counts": {table: counts[table] for table in CORE_TABLES},
            "audit_counts": {table: counts[table] for table in AUDIT_TABLES},
            "audit_row_fingerprints": audit_row_fingerprints,
            "schema_manifest": schema_manifest,
            "status_invocation_refs": status_refs,
            "migration_counts": migration_counts,
            "migrations": migrations,
        },
        manifest,
    )


def load_expected_migrations(
    repository_root: Path,
) -> tuple[dict[int, dict[str, str]], list[GuardedFile], str]:
    root = require_directory(
        str(repository_root / "internal/adapters/state/sqlite/migrations"),
        "migration_root",
    )
    expected: dict[int, dict[str, str]] = {}
    guarded: list[GuardedFile] = []
    for candidate in sorted(root.glob("[0-9][0-9][0-9]_*.sql")):
        path = canonical_existing(str(candidate), "migration_file")
        if not path.is_file():
            fail("migration_file_invalid", str(path))
        match = re.match(r"^([0-9]{3})_", path.name)
        if match is None:
            continue
        version = int(match.group(1))
        if version > 19:
            continue
        checksum = sha256_file(path)
        expected[version] = {
            "name": path.name,
            "checksum": "sha256:" + checksum,
            "sha256": checksum,
        }
        guarded.append(GuardedFile(f"migration_{version}", path, checksum))
    if set(expected) != set(range(1, 20)):
        fail("migration_manifest_incomplete")
    for version, name in MIGRATION_NAMES.items():
        if expected[version]["name"] != name:
            fail("migration_name_unexpected", f"{version}:{expected[version]['name']}")
    canonical = json.dumps(
        expected, sort_keys=True, separators=(",", ":")
    ).encode("utf-8")
    return expected, guarded, sha256_bytes(canonical)


def build_expected_v19_schema_manifest(
    backup: Path,
    target: Path,
    migration_files: dict[int, GuardedFile],
    expected_migrations: dict[int, dict[str, str]],
) -> list[dict[str, Any]]:
    copy_private(backup, target, 0o600)
    try:
        connection = sqlite3.connect(target)
        try:
            current = int(
                connection.execute("PRAGMA user_version").fetchone()[0]
            )
            if current != 16:
                fail("expected_schema_source_not_v16", str(current))
            connection.execute("PRAGMA foreign_keys=OFF")
            for version in (17, 18, 19):
                migration = migration_files[version]
                try:
                    sql = migration.path.read_text(encoding="utf-8")
                except (OSError, UnicodeError) as error:
                    fail("expected_schema_migration_read_failed", str(error))
                connection.executescript(sql)
                connection.execute(
                    """
                    INSERT INTO schema_migrations(version,name,checksum)
                    VALUES(?,?,?)
                    """,
                    (
                        version,
                        expected_migrations[version]["name"],
                        expected_migrations[version]["checksum"],
                    ),
                )
                connection.execute(f"PRAGMA user_version={version}")
                connection.commit()
            return sqlite_schema_manifest(connection)
        except sqlite3.Error as error:
            fail("expected_schema_build_failed", str(error))
        finally:
            connection.close()
    finally:
        for candidate in (
            target,
            Path(str(target) + "-journal"),
            Path(str(target) + "-wal"),
            Path(str(target) + "-shm"),
        ):
            try:
                candidate.unlink()
            except FileNotFoundError:
                pass
            except OSError as error:
                fail("expected_schema_cleanup_failed", str(error))


@dataclass
class CommandEvidence:
    label: str
    argv: list[str]
    returncode: int
    stdout_path: Path
    stderr_path: Path
    stdout: str
    stderr: str

    def receipt(self, root: Path) -> dict[str, Any]:
        return {
            "label": self.label,
            "argv": self.argv,
            "returncode": self.returncode,
            "stdout": {
                "path": str(self.stdout_path.relative_to(root)),
                "sha256": sha256_file(self.stdout_path),
                "bytes": self.stdout_path.stat().st_size,
            },
            "stderr": {
                "path": str(self.stderr_path.relative_to(root)),
                "sha256": sha256_file(self.stderr_path),
                "bytes": self.stderr_path.stat().st_size,
            },
        }


@dataclass
class CrashWatchdog:
    pid: int
    command_fd: int
    acknowledgement_fd: int
    armed: bool = True

    def send(self, message: str) -> None:
        if not self.armed:
            fail("crash_watchdog_not_armed")
        payload = (message + "\n").encode("ascii")
        try:
            written = os.write(self.command_fd, payload)
        except OSError as error:
            fail("crash_watchdog_channel_failed", str(error))
        if written != len(payload):
            fail("crash_watchdog_channel_short_write")


def process_starttime(pid: int) -> int:
    try:
        content = (Path("/proc") / str(pid) / "stat").read_text(
            encoding="ascii"
        )
    except (OSError, UnicodeError) as error:
        fail("process_starttime_unreadable", f"pid={pid}:{error}")
    separator = content.rfind(")")
    fields = content[separator + 2 :].split() if separator >= 0 else []
    if len(fields) < 20 or not fields[19].isdigit():
        fail("process_starttime_invalid", str(pid))
    return int(fields[19])


def process_group_live_members(pgid: int) -> list[int]:
    members: list[int] = []
    for candidate in Path("/proc").iterdir():
        if not candidate.name.isdigit():
            continue
        try:
            content = (candidate / "stat").read_text(encoding="ascii")
        except (OSError, UnicodeError):
            continue
        separator = content.rfind(")")
        fields = content[separator + 2 :].split() if separator >= 0 else []
        if (
            len(fields) >= 3
            and fields[2].lstrip("-").isdigit()
            and int(fields[2]) == pgid
            and fields[0] != "Z"
        ):
            members.append(int(candidate.name))
    return sorted(members)


class Runner:
    def __init__(
        self, root: Path, guard: IntegrityGuard, timeout: int
    ) -> None:
        self.root = root
        self.evidence_root = root / "evidence"
        self.guard = guard
        self.timeout = timeout
        self.items: list[CommandEvidence] = []
        self.watchdog: CrashWatchdog | None = None

    def run(
        self,
        label: str,
        argv: Sequence[str],
        *,
        allowed: frozenset[int] = frozenset({0}),
    ) -> CommandEvidence:
        self.guard.verify()
        environment = {"PATH": "/usr/bin:/bin", "LC_ALL": "C"}
        for name in ("DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR"):
            if name in os.environ:
                environment[name] = os.environ[name]
        process: subprocess.Popen[bytes] | None = None
        try:
            process = subprocess.Popen(
                list(argv),
                stdin=subprocess.DEVNULL,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                env=environment,
                start_new_session=True,
            )
            if self.watchdog is not None:
                self.watchdog.send(
                    f"PGID {process.pid} {process_starttime(process.pid)}"
                )
            stdout_full, stderr_full = process.communicate(timeout=self.timeout)
            if self.watchdog is not None:
                self.watchdog.send(f"CLEAR {process.pid}")
        except subprocess.TimeoutExpired as error:
            if process is not None:
                terminate_process_group(process)
                if self.watchdog is not None:
                    self.watchdog.send(f"CLEAR {process.pid}")
            fail("command_timeout", f"{label}:{error.timeout}")
        except BaseException:
            if process is not None:
                terminate_process_group(process)
                if self.watchdog is not None:
                    self.watchdog.send(f"CLEAR {process.pid}")
            raise
        stdout = stdout_full[:1024 * 1024]
        stderr = stderr_full[:1024 * 1024]
        index = len(self.items) + 1
        stdout_path = self.evidence_root / f"{index:02d}-{label}.stdout"
        stderr_path = self.evidence_root / f"{index:02d}-{label}.stderr"
        write_private(stdout_path, stdout)
        write_private(stderr_path, stderr)
        item = CommandEvidence(
            label=label,
            argv=list(argv),
            returncode=process.returncode,
            stdout_path=stdout_path,
            stderr_path=stderr_path,
            stdout=stdout.decode("utf-8", errors="replace"),
            stderr=stderr.decode("utf-8", errors="replace"),
        )
        self.items.append(item)
        self.guard.verify()
        if process.returncode not in allowed:
            fail(
                "command_failed",
                f"{label}:exit={process.returncode}:stderr={item.stderr[:512]!r}",
            )
        return item


def terminate_process_group(process: subprocess.Popen[bytes]) -> None:
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        pass
    try:
        os.killpg(process.pid, 0)
    except ProcessLookupError:
        return
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        return
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        pass


def terminate_external_process_group(pgid: int, expected_starttime: int) -> None:
    if pgid <= 1:
        return
    try:
        observed_starttime = process_starttime(pgid)
    except HarnessError:
        observed_starttime = None
    if (
        observed_starttime is not None
        and observed_starttime != expected_starttime
    ):
        fail("active_command_pid_reused", str(pgid))
    try:
        os.killpg(pgid, signal.SIGTERM)
    except ProcessLookupError:
        return
    deadline = time.monotonic() + 2
    while time.monotonic() < deadline:
        if not process_group_live_members(pgid):
            return
        time.sleep(0.05)
    try:
        os.killpg(pgid, signal.SIGKILL)
    except ProcessLookupError:
        return
    deadline = time.monotonic() + 2
    while time.monotonic() < deadline:
        if not process_group_live_members(pgid):
            return
        time.sleep(0.05)
    fail(
        "active_command_group_remaining",
        repr(process_group_live_members(pgid)),
    )


def parse_invocation(output: str, expected_action: str) -> str:
    if f"status=running action={expected_action} " not in output:
        fail("adapter_output_invalid", output[:512])
    match = re.search(r"(?:^| )invocation_id=([0-9a-f]{32})(?:\s|$)", output)
    if match is None:
        fail("adapter_invocation_id_missing", output[:512])
    return match.group(1)


def parse_runtime_pids(output: str) -> tuple[int, int]:
    values: dict[str, int] = {}
    for name in ("main_pid", "daemon_pid"):
        match = re.search(rf"(?:^| ){name}=([1-9][0-9]*)(?:\s|$)", output)
        if match is None:
            fail("adapter_runtime_pid_missing", name)
        values[name] = int(match.group(1))
    return values["main_pid"], values["daemon_pid"]


def read_process_cgroup(proc_root: Path, pid: int) -> str:
    path = proc_root / str(pid) / "cgroup"
    try:
        content = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as error:
        fail("runtime_process_cgroup_unreadable", f"pid={pid}:{error}")
    lines = content.splitlines()
    if len(lines) != 1 or not lines[0].startswith("0::/"):
        fail("runtime_process_cgroup_invalid", f"pid={pid}")
    return lines[0].removeprefix("0::")


def parse_go_buildinfo(output: str) -> tuple[str, bool]:
    fields: dict[str, str] = {}
    for line in output.splitlines():
        stripped = line.strip()
        if not stripped.startswith("build\t"):
            continue
        assignment = stripped.removeprefix("build\t")
        key, separator, value = assignment.partition("=")
        if not separator or key in fields:
            fail("binary_buildinfo_invalid", stripped)
        fields[key] = value
    revision = fields.get("vcs.revision", "")
    modified = fields.get("vcs.modified", "")
    if not REVISION_RE.fullmatch(revision) or modified not in {"true", "false"}:
        fail("binary_buildinfo_vcs_missing")
    return revision, modified == "true"


def direct_unit_state(
    runner: Runner, systemctl: Path, unit: str, label: str
) -> str:
    result = runner.run(
        label,
        [
            str(systemctl),
            "--user",
            "show",
            unit,
            "--property=LoadState",
            "--value",
        ],
    )
    state = result.stdout.strip()
    if not state or "\n" in state:
        fail("unit_load_state_invalid", repr(state))
    return state


def direct_control_group(
    runner: Runner, systemctl: Path, unit: str, label: str
) -> str:
    result = runner.run(
        label,
        [
            str(systemctl),
            "--user",
            "show",
            unit,
            "--property=ControlGroup",
            "--value",
        ],
    )
    value = result.stdout.strip()
    if (
        not value.startswith("/")
        or "\n" in value
        or "/../" in value
        or value.endswith("/..")
    ):
        fail("unit_control_group_invalid", repr(value))
    return value


def direct_user_manager_control_group(
    runner: Runner, systemctl: Path
) -> str:
    result = runner.run(
        "user-manager-control-group",
        [
            str(systemctl),
            "--user",
            "show",
            "--property=ControlGroup",
            "--value",
        ],
    )
    value = result.stdout.strip()
    if (
        not value.startswith("/")
        or value == "/"
        or "\n" in value
        or "/../" in value
        or value.endswith("/..")
        or value.endswith("/")
    ):
        fail("user_manager_control_group_invalid", repr(value))
    return value


def audit_delta_valid(
    source: dict[str, Any], observed: dict[str, Any], cycle: int
) -> bool:
    for table in AUDIT_TABLES:
        if (
            observed["audit_counts"][table]
            != source["audit_counts"][table] + cycle
        ):
            return False
        source_rows = Counter(source["audit_row_fingerprints"][table])
        observed_rows = Counter(observed["audit_row_fingerprints"][table])
        if any(
            observed_rows[fingerprint] < count
            for fingerprint, count in source_rows.items()
        ):
            return False
    if len(observed["status_invocation_refs"]) != len(
        source["status_invocation_refs"]
    ) + cycle:
        return False
    return True


def functional_counts_equal(
    source: dict[str, Any], observed: dict[str, Any]
) -> bool:
    for table, count in source["table_counts"].items():
        if table in EXCLUDED_FUNCTIONAL_TABLES:
            continue
        if observed["table_counts"].get(table) != count:
            return False
    baseline_tables = set(source["table_counts"])
    for table, count in observed["table_counts"].items():
        if (
            table not in baseline_tables
            and table not in EXCLUDED_FUNCTIONAL_TABLES
            and count != 0
        ):
            return False
    return True


def scan_processes(
    proc_root: Path, binary: Path, database: Path
) -> dict[str, list[int]]:
    binary_stat = binary.stat(follow_symlinks=False)
    database_stat = database.stat(follow_symlinks=False)
    binary_pids: list[int] = []
    database_pids: list[int] = []
    for entry in proc_root.iterdir():
        if not entry.name.isdigit() or not entry.is_dir():
            continue
        try:
            owner = entry.stat(follow_symlinks=False).st_uid
        except (FileNotFoundError, PermissionError):
            continue
        if owner != os.geteuid():
            continue
        pid = int(entry.name)
        try:
            executable = (entry / "exe").stat()
            if (
                executable.st_dev == binary_stat.st_dev
                and executable.st_ino == binary_stat.st_ino
            ):
                binary_pids.append(pid)
        except (FileNotFoundError, PermissionError):
            pass
        descriptor_root = entry / "fd"
        try:
            descriptors = list(descriptor_root.iterdir())
        except FileNotFoundError:
            continue
        except PermissionError as error:
            fail("process_scan_incomplete", f"pid={pid}:{error}")
        for descriptor in descriptors:
            try:
                opened = descriptor.stat()
            except (FileNotFoundError, PermissionError):
                continue
            if (
                opened.st_dev == database_stat.st_dev
                and opened.st_ino == database_stat.st_ino
            ):
                database_pids.append(pid)
                break
    return {
        "candidate_pids": sorted(set(binary_pids)),
        "database_open_pids": sorted(set(database_pids)),
    }


def remove_credential_projections(
    root: Path,
    projected_auth: Path,
    isolated_config_paths: dict[str, Path],
    synthetic_auth_sha: str,
) -> None:
    targets = {
        projected_auth,
        isolated_config_paths["credentials.local.path"],
        isolated_config_paths["identity.local_token_path"],
    }
    for target in targets:
        if not is_within(target, root):
            fail("credential_projection_outside_private_root")
        if not target.exists() and not target.is_symlink():
            continue
        metadata = target.lstat()
        if (
            stat.S_ISLNK(metadata.st_mode)
            or not stat.S_ISREG(metadata.st_mode)
            or metadata.st_uid != os.geteuid()
        ):
            fail("credential_projection_type_invalid", str(target))
        target.unlink()
    for current, _, files in os.walk(root, followlinks=False):
        for name in files:
            candidate = Path(current) / name
            if name == "auth.json":
                fail("credential_projection_remaining", str(candidate))
            try:
                if sha256_file(candidate) == synthetic_auth_sha:
                    fail("credential_projection_remaining", str(candidate))
            except OSError as error:
                fail("credential_projection_scan_failed", str(error))


def validate_private_tree(root: Path, executable_paths: set[Path]) -> None:
    for current_root, directories, files in os.walk(root, followlinks=False):
        directory = Path(current_root)
        metadata = directory.lstat()
        if (
            stat.S_ISLNK(metadata.st_mode)
            or metadata.st_uid != os.geteuid()
            or stat.S_IMODE(metadata.st_mode) != 0o700
        ):
            fail("private_directory_contract_failed", str(directory))
        for name in directories:
            child = directory / name
            if child.is_symlink():
                fail("private_tree_symlink", str(child))
        for name in files:
            child = directory / name
            metadata = child.lstat()
            if stat.S_ISLNK(metadata.st_mode) or not stat.S_ISREG(metadata.st_mode):
                fail("private_file_type_invalid", str(child))
            expected = 0o500 if child in executable_paths else 0o600
            if (
                metadata.st_uid != os.geteuid()
                or stat.S_IMODE(metadata.st_mode) != expected
                or metadata.st_nlink != 1
            ):
                fail(
                    "private_file_contract_failed",
                    f"{child}:mode={stat.S_IMODE(metadata.st_mode):o}",
                )


def write_receipt(path: Path, value: dict[str, Any]) -> str:
    encoded = (
        json.dumps(
            value,
            ensure_ascii=False,
            sort_keys=True,
            indent=2,
            allow_nan=False,
        )
        + "\n"
    ).encode("utf-8")
    write_private(path, encoded)
    return sha256_bytes(encoded)


def publish_pending_receipt(pending: Path, path: Path) -> None:
    linked = False
    try:
        os.link(pending, path, follow_symlinks=False)
        linked = True
        pending.unlink()
        metadata = path.lstat()
        if (
            not stat.S_ISREG(metadata.st_mode)
            or metadata.st_nlink != 1
            or stat.S_IMODE(metadata.st_mode) != 0o600
        ):
            fail("terminal_receipt_contract_failed")
        descriptor = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
    except BaseException:
        for candidate in (path if linked else None, pending):
            if candidate is None:
                continue
            try:
                candidate.unlink()
            except FileNotFoundError:
                pass
            except OSError:
                pass
        try:
            descriptor = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
            try:
                os.fsync(descriptor)
            finally:
                os.close(descriptor)
        except OSError:
            pass
        raise


def write_terminal_receipt(path: Path, value: dict[str, Any]) -> str:
    pending = path.with_name("." + path.name + ".pending")
    digest = write_receipt(pending, value)
    publish_pending_receipt(pending, path)
    return digest


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(
        prog=PROGRAM,
        description=(
            "Reacredita SQLite V16 -> V19 sobre copia privada. La plantilla "
            "TOML debe contener todos los marcadores ORQUESTA_REACCREDIT."
        ),
        epilog="Marcadores: " + ", ".join(sorted(MARKERS)),
    )
    result.add_argument("--backup", required=True)
    result.add_argument("--expected-backup-sha256", required=True)
    result.add_argument("--binary", required=True)
    result.add_argument("--expected-binary-sha256", required=True)
    result.add_argument("--expected-revision", required=True)
    result.add_argument("--repository-root", required=True)
    result.add_argument("--profile-script", required=True)
    result.add_argument("--expected-profile-script-sha256", required=True)
    result.add_argument("--adapter", required=True)
    result.add_argument("--expected-adapter-sha256", required=True)
    result.add_argument("--config-template", required=True)
    result.add_argument("--expected-config-template-sha256", required=True)
    result.add_argument("--output-dir", required=True)
    result.add_argument("--forbidden-live-root", required=True)
    result.add_argument("--exec-path", required=True)
    result.add_argument("--go", required=True)
    result.add_argument("--expected-go-sha256", required=True)
    result.add_argument("--git", required=True)
    result.add_argument("--expected-git-sha256", required=True)
    result.add_argument("--bubblewrap", required=True)
    result.add_argument("--expected-bubblewrap-sha256", required=True)
    result.add_argument("--systemctl", required=True)
    result.add_argument("--expected-systemctl-sha256", required=True)
    result.add_argument("--systemd-run", required=True)
    result.add_argument("--expected-systemd-run-sha256", required=True)
    result.add_argument("--proc-root", required=True)
    result.add_argument("--cgroup-mount", required=True)
    result.add_argument("--readiness-timeout", type=int, default=30)
    result.add_argument("--collection-timeout", type=int, default=10)
    result.add_argument("--command-timeout", type=int, default=90)
    return result


def adapter_arguments(
    adapter: Path,
    operation: str,
    *,
    unit: str,
    profile: str,
    repository_root: Path,
    profile_script: Path,
    profile_script_sha: str,
    runtime_base: Path,
    source_home: Path,
    binary: Path,
    binary_sha: str,
    config: Path,
    config_sha: str,
    exec_path: str,
    systemctl: Path,
    systemctl_sha: str,
    systemd_run: Path,
    systemd_run_sha: str,
    proc_root: Path,
    cgroup_mount: Path,
    readiness_timeout: int,
    collection_timeout: int,
) -> list[str]:
    prefix = [str(adapter)]
    if operation != "check":
        prefix.extend(["--apply", operation])
    return prefix + [
        "--unit",
        unit,
        "--profile",
        profile,
        "--repository-root",
        str(repository_root),
        "--profile-script",
        str(profile_script),
        "--expected-profile-script-sha256",
        profile_script_sha,
        "--runtime-base",
        str(runtime_base),
        "--source-codex-home",
        str(source_home),
        "--binary",
        str(binary),
        "--expected-binary-sha256",
        binary_sha,
        "--config",
        str(config),
        "--expected-config-sha256",
        config_sha,
        "--exec-path",
        exec_path,
        "--systemctl",
        str(systemctl),
        "--expected-systemctl-sha256",
        systemctl_sha,
        "--systemd-run",
        str(systemd_run),
        "--expected-systemd-run-sha256",
        systemd_run_sha,
        "--proc-root",
        str(proc_root),
        "--cgroup-mount",
        str(cgroup_mount),
        "--readiness-timeout",
        str(readiness_timeout),
        "--collection-timeout",
        str(collection_timeout),
    ]


def cleanup_command(
    label: str,
    argv: Sequence[str],
    guard: IntegrityGuard,
    timeout: int,
    watchdog: CrashWatchdog | None = None,
) -> dict[str, Any]:
    attempt: dict[str, Any] = {
        "label": label,
        "argv": list(argv),
        "returncode": None,
        "completed": False,
    }
    process: subprocess.Popen[bytes] | None = None
    try:
        guard.verify()
        environment = {"PATH": "/usr/bin:/bin", "LC_ALL": "C"}
        for name in ("DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR"):
            if name in os.environ:
                environment[name] = os.environ[name]
        process = subprocess.Popen(
            list(argv),
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=environment,
            start_new_session=True,
        )
        if watchdog is not None:
            watchdog.send(
                f"PGID {process.pid} {process_starttime(process.pid)}"
            )
        stdout, stderr = process.communicate(timeout=timeout)
        if watchdog is not None:
            watchdog.send(f"CLEAR {process.pid}")
        attempt["returncode"] = process.returncode
        attempt["completed"] = True
        attempt["stdout"] = stdout[:4096].decode(
            "utf-8", errors="replace"
        )
        attempt["stderr"] = stderr[:4096].decode(
            "utf-8", errors="replace"
        )
    except subprocess.TimeoutExpired:
        if process is not None:
            terminate_process_group(process)
            if watchdog is not None:
                watchdog.send(f"CLEAR {process.pid}")
        attempt["error"] = "timeout"
    except BaseException as error:
        if process is not None:
            terminate_process_group(process)
            if watchdog is not None:
                watchdog.send(f"CLEAR {process.pid}")
        attempt["error"] = type(error).__name__
    return attempt


def exact_identity_paths(runtime_root: Path) -> tuple[Path, ...]:
    return tuple(
        runtime_root / relative
        for relative in (
            "run/server.pid",
            "run/server.start_ref",
            "run/server.binary_id",
            "run/server.binary_sha256",
            "run/server.config_sha256",
        )
    )


def cleanup_census(
    proc_root: Path,
    binary: Path,
    database: Path,
    runtime_root: Path,
    output_root: Path,
    synthetic_auth_sha: str,
    unit_load_state: str,
) -> dict[str, Any]:
    census: dict[str, Any] = {
        "unit_load_state": unit_load_state,
        "candidate_pids": [],
        "database_open_pids": [],
        "identity_files": [],
        "sqlite_ancillary_files": [],
        "credential_projections": [],
        "scan_errors": [],
    }
    try:
        census.update(scan_processes(proc_root, binary, database))
    except (HarnessError, OSError) as error:
        census["scan_errors"].append(f"processes:{type(error).__name__}")
    census["identity_files"] = [
        str(path.relative_to(output_root))
        for path in exact_identity_paths(runtime_root)
        if path.exists() or path.is_symlink()
    ]
    census["sqlite_ancillary_files"] = [
        str(path.relative_to(output_root))
        for path in (
            Path(str(database) + "-wal"),
            Path(str(database) + "-shm"),
            Path(str(database) + "-journal"),
        )
        if path.exists() or path.is_symlink()
    ]
    try:
        for current, _, files in os.walk(output_root, followlinks=False):
            for name in files:
                candidate = Path(current) / name
                if name == "auth.json":
                    census["credential_projections"].append(
                        str(candidate.relative_to(output_root))
                    )
                    continue
                try:
                    if sha256_file(candidate) == synthetic_auth_sha:
                        census["credential_projections"].append(
                            str(candidate.relative_to(output_root))
                        )
                except OSError as error:
                    census["scan_errors"].append(
                        f"credentials:{type(error).__name__}"
                    )
    except OSError as error:
        census["scan_errors"].append(
            f"credentials_walk:{type(error).__name__}"
        )
    return census


def cleanup_verified(census: dict[str, Any]) -> bool:
    return (
        census["unit_load_state"] == "not-found"
        and not census["candidate_pids"]
        and not census["database_open_pids"]
        and not census["identity_files"]
        and not census["sqlite_ancillary_files"]
        and not census["credential_projections"]
        and not census["scan_errors"]
        and not census.get("cleanup_errors", [])
    )


def perform_failure_cleanup(
    *,
    cleanup_args: dict[str, Any],
    adapter_guard: IntegrityGuard,
    systemctl_guard: IntegrityGuard,
    timeout: int,
    proc_root: Path,
    binary: Path,
    database: Path,
    runtime_root: Path,
    output_root: Path,
    credential_context: tuple[Path, dict[str, Path], str],
    watchdog: CrashWatchdog | None = None,
) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    adapter_path = cleanup_args["adapter"]
    common_cleanup = {
        key: value for key, value in cleanup_args.items() if key != "adapter"
    }
    attempts: list[dict[str, Any]] = []
    for operation, label in (
        ("stop-profile", "failure-cleanup-stop"),
        ("collect", "failure-cleanup-collect"),
    ):
        attempts.append(
            cleanup_command(
                label,
                adapter_arguments(adapter_path, operation, **common_cleanup),
                adapter_guard,
                timeout,
                watchdog,
            )
        )
    systemctl = common_cleanup["systemctl"]
    unit = common_cleanup["unit"]

    def observe_unit(label: str) -> str:
        attempt = cleanup_command(
            label,
            [
                str(systemctl),
                "--user",
                "show",
                unit,
                "--property=LoadState",
                "--value",
            ],
            systemctl_guard,
            timeout,
            watchdog,
        )
        attempts.append(attempt)
        if attempt.get("returncode") != 0:
            return "unknown"
        state = str(attempt.get("stdout", "")).strip()
        return state if state and "\n" not in state else "unknown"

    unit_state = observe_unit("failure-cleanup-unit-before-fallback")
    if unit_state != "not-found":
        for label, action in (
            ("failure-cleanup-systemctl-stop", "stop"),
            ("failure-cleanup-systemctl-reset-failed", "reset-failed"),
        ):
            attempts.append(
                cleanup_command(
                    label,
                    [str(systemctl), "--user", action, unit],
                    systemctl_guard,
                    timeout,
                    watchdog,
                )
            )
        attempts.append(
            cleanup_command(
                "failure-cleanup-collect-retry",
                adapter_arguments(adapter_path, "collect", **common_cleanup),
                adapter_guard,
                timeout,
                watchdog,
            )
        )
        deadline = time.monotonic() + min(timeout, 5)
        while True:
            unit_state = observe_unit("failure-cleanup-unit-after-fallback")
            if unit_state == "not-found" or time.monotonic() >= deadline:
                break
            time.sleep(0.1)

    preliminary = cleanup_census(
        proc_root,
        binary,
        database,
        runtime_root,
        output_root,
        credential_context[2],
        unit_state,
    )
    if (
        unit_state == "not-found"
        and not preliminary["candidate_pids"]
        and not preliminary["database_open_pids"]
    ):
        for identity in exact_identity_paths(runtime_root):
            try:
                identity.unlink()
            except FileNotFoundError:
                pass
            except OSError:
                preliminary["scan_errors"].append("identity_cleanup:OSError")
        if database.exists() and not database.is_symlink():
            try:
                connection = sqlite3.connect(database)
                try:
                    connection.execute("PRAGMA wal_checkpoint(TRUNCATE)")
                finally:
                    connection.close()
            except sqlite3.Error:
                preliminary["scan_errors"].append("sqlite_cleanup:Error")
    try:
        remove_credential_projections(output_root, *credential_context)
    except HarnessError:
        preliminary["scan_errors"].append("credential_cleanup:HarnessError")
    final_state = observe_unit("failure-cleanup-unit-final")
    census = cleanup_census(
        proc_root,
        binary,
        database,
        runtime_root,
        output_root,
        credential_context[2],
        final_state,
    )
    census["cleanup_errors"] = preliminary["scan_errors"]
    return attempts, census


def valid_terminal_receipt(path: Path) -> dict[str, Any] | None:
    try:
        metadata = path.lstat()
        if (
            not stat.S_ISREG(metadata.st_mode)
            or metadata.st_nlink != 1
            or stat.S_IMODE(metadata.st_mode) != 0o600
        ):
            return None
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return None
    if (
        not isinstance(value, dict)
        or value.get("schema_version") != SCHEMA
        or value.get("result") not in {"pass", "fail"}
    ):
        return None
    return value


def remove_terminal_artifacts(output_root: Path) -> None:
    for name in (
        "receipt.json",
        ".receipt.json.pending",
        "failure.json",
        ".failure.json.pending",
    ):
        try:
            (output_root / name).unlink()
        except FileNotFoundError:
            pass
    descriptor = os.open(output_root, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def read_watchdog_ack(
    watchdog: CrashWatchdog, expected: bytes, timeout: int
) -> None:
    readable, _, _ = select.select(
        [watchdog.acknowledgement_fd], [], [], timeout
    )
    if not readable:
        fail("crash_watchdog_ack_timeout", expected.decode("ascii").strip())
    try:
        observed = os.read(watchdog.acknowledgement_fd, 128)
    except OSError as error:
        fail("crash_watchdog_ack_failed", str(error))
    if observed != expected:
        fail(
            "crash_watchdog_ack_invalid",
            f"expected={expected!r}:observed={observed!r}",
        )


def crash_watchdog_child(
    *,
    command_fd: int,
    acknowledgement_fd: int,
    parent_pid: int,
    cleanup_args: dict[str, Any],
    adapter_guard: IntegrityGuard,
    systemctl_guard: IntegrityGuard,
    timeout: int,
    proc_root: Path,
    binary: Path,
    database: Path,
    runtime_root: Path,
    output_root: Path,
    credential_context: tuple[Path, dict[str, Path], str],
) -> None:
    os.setsid()
    for current in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
        signal.signal(current, signal.SIG_DFL)
    null_fd = os.open("/dev/null", os.O_RDWR)
    try:
        for descriptor in (0, 1, 2):
            os.dup2(null_fd, descriptor)
    finally:
        if null_fd > 2:
            os.close(null_fd)
    os.write(acknowledgement_fd, b"READY\n")
    active_group: tuple[int, int] | None = None
    buffer = b""
    committed = False
    while True:
        chunk = os.read(command_fd, 4096)
        if not chunk:
            break
        buffer += chunk
        while b"\n" in buffer:
            raw, buffer = buffer.split(b"\n", 1)
            command = raw.decode("ascii", errors="strict")
            if command.startswith("PGID "):
                parts = command.split()
                if (
                    len(parts) != 3
                    or not parts[1].isdigit()
                    or not parts[2].isdigit()
                ):
                    raise RuntimeError("watchdog pgid format invalid")
                value = int(parts[1])
                starttime = int(parts[2])
                if value <= 1 or starttime <= 0:
                    raise RuntimeError("watchdog pgid invalid")
                active_group = (value, starttime)
            elif command.startswith("CLEAR "):
                value = int(command.removeprefix("CLEAR "))
                if active_group is not None and active_group[0] == value:
                    active_group = None
            elif command.startswith("PUBLISH_PASS "):
                expected_sha = command.removeprefix("PUBLISH_PASS ")
                pending = output_root / ".receipt.json.pending"
                receipt_path = output_root / "receipt.json"
                if (
                    not SHA_RE.fullmatch(expected_sha)
                    or sha256_file(pending) != expected_sha
                ):
                    raise RuntimeError("watchdog pass digest invalid")
                try:
                    pending_value = json.loads(
                        pending.read_text(encoding="utf-8")
                    )
                except (OSError, UnicodeError, json.JSONDecodeError) as error:
                    raise RuntimeError(
                        "watchdog pass receipt invalid"
                    ) from error
                if (
                    not isinstance(pending_value, dict)
                    or pending_value.get("schema_version") != SCHEMA
                    or pending_value.get("result") != "pass"
                ):
                    raise RuntimeError("watchdog pass contract invalid")
                publish_pending_receipt(pending, receipt_path)
                committed = True
                os.write(acknowledgement_fd, b"PUBLISHED\n")
                break
            elif command == "COMMIT":
                committed = True
                os.write(acknowledgement_fd, b"COMMITTED\n")
                break
            else:
                raise RuntimeError("watchdog command invalid")
        if committed:
            break
    if committed:
        return
    watchdog_errors: list[str] = []
    if active_group is not None:
        try:
            terminate_external_process_group(*active_group)
        except (HarnessError, OSError) as error:
            watchdog_errors.append(
                f"active_group:{type(error).__name__}:{error}"
            )
    try:
        attempts, census = perform_failure_cleanup(
            cleanup_args=cleanup_args,
            adapter_guard=adapter_guard,
            systemctl_guard=systemctl_guard,
            timeout=timeout,
            proc_root=proc_root,
            binary=binary,
            database=database,
            runtime_root=runtime_root,
            output_root=output_root,
            credential_context=credential_context,
        )
    except (HarnessError, KeyError, OSError) as error:
        attempts = []
        census = cleanup_census(
            proc_root,
            binary,
            database,
            runtime_root,
            output_root,
            credential_context[2],
            "unknown",
        )
        watchdog_errors.append(
            f"cleanup:{type(error).__name__}:{error}"
        )
    census.setdefault("cleanup_errors", []).extend(watchdog_errors)
    failed = valid_terminal_receipt(output_root / "failure.json")
    if failed is not None:
        return
    remove_terminal_artifacts(output_root)
    write_terminal_receipt(
        output_root / "failure.json",
        {
            "schema_version": SCHEMA,
            "result": "fail",
            "created_at": utc_now(),
            "reason_code": "parent_process_lost",
            "detail": f"parent_pid={parent_pid}",
            "cleanup_attempts": attempts,
            "cleanup_verified": cleanup_verified(census),
            "residues": census,
            "watchdog": {
                "pid": os.getpid(),
                "parent_pid": parent_pid,
                "active_pgid_terminated": (
                    None if active_group is None else active_group[0]
                ),
            },
        },
    )


def arm_crash_watchdog(
    *,
    cleanup_args: dict[str, Any],
    adapter_guard: IntegrityGuard,
    systemctl_guard: IntegrityGuard,
    timeout: int,
    proc_root: Path,
    binary: Path,
    database: Path,
    runtime_root: Path,
    output_root: Path,
    credential_context: tuple[Path, dict[str, Path], str],
) -> CrashWatchdog:
    command_read, command_write = os.pipe2(os.O_CLOEXEC)
    acknowledgement_read, acknowledgement_write = os.pipe2(os.O_CLOEXEC)
    parent_pid = os.getpid()
    try:
        pid = os.fork()
    except OSError:
        for descriptor in (
            command_read,
            command_write,
            acknowledgement_read,
            acknowledgement_write,
        ):
            os.close(descriptor)
        raise
    if pid == 0:
        os.close(command_write)
        os.close(acknowledgement_read)
        retained = {command_read, acknowledgement_write}
        try:
            inherited = [
                int(candidate.name)
                for candidate in Path("/proc/self/fd").iterdir()
                if candidate.name.isdigit()
            ]
        except OSError:
            inherited = []
        for descriptor in inherited:
            if descriptor > 2 and descriptor not in retained:
                try:
                    os.close(descriptor)
                except OSError:
                    pass
        try:
            crash_watchdog_child(
                command_fd=command_read,
                acknowledgement_fd=acknowledgement_write,
                parent_pid=parent_pid,
                cleanup_args=cleanup_args,
                adapter_guard=adapter_guard,
                systemctl_guard=systemctl_guard,
                timeout=timeout,
                proc_root=proc_root,
                binary=binary,
                database=database,
                runtime_root=runtime_root,
                output_root=output_root,
                credential_context=credential_context,
            )
            os._exit(0)
        except BaseException:
            os._exit(125)
    os.close(command_read)
    os.close(acknowledgement_write)
    watchdog = CrashWatchdog(pid, command_write, acknowledgement_read)
    try:
        read_watchdog_ack(watchdog, b"READY\n", min(timeout, 5))
    except BaseException:
        os.close(command_write)
        os.close(acknowledgement_read)
        try:
            os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        os.waitpid(pid, 0)
        raise
    return watchdog


def publish_pass_with_watchdog(
    watchdog: CrashWatchdog,
    receipt_path: Path,
    receipt: dict[str, Any],
    timeout: int,
) -> str:
    pending = receipt_path.with_name("." + receipt_path.name + ".pending")
    digest = write_receipt(pending, receipt)
    watchdog.send(f"PUBLISH_PASS {digest}")
    read_watchdog_ack(watchdog, b"PUBLISHED\n", min(timeout, 5))
    os.close(watchdog.command_fd)
    os.close(watchdog.acknowledgement_fd)
    _, status = os.waitpid(watchdog.pid, 0)
    watchdog.armed = False
    if status != 0:
        fail("crash_watchdog_publish_failed", str(status))
    if sha256_file(receipt_path) != digest:
        fail("terminal_receipt_hash_mismatch")
    return digest


def commit_crash_watchdog(watchdog: CrashWatchdog, timeout: int) -> None:
    if not watchdog.armed:
        return
    watchdog.send("COMMIT")
    read_watchdog_ack(watchdog, b"COMMITTED\n", min(timeout, 5))
    os.close(watchdog.command_fd)
    os.close(watchdog.acknowledgement_fd)
    _, status = os.waitpid(watchdog.pid, 0)
    watchdog.armed = False
    if status != 0:
        fail("crash_watchdog_exit_failed", str(status))


def trigger_crash_watchdog(watchdog: CrashWatchdog) -> None:
    if not watchdog.armed:
        return
    os.close(watchdog.command_fd)
    os.close(watchdog.acknowledgement_fd)
    _, status = os.waitpid(watchdog.pid, 0)
    watchdog.armed = False
    if status != 0:
        fail("crash_watchdog_cleanup_failed", str(status))


def main(arguments: Sequence[str]) -> int:
    args = parser().parse_args(arguments)
    os.umask(0o077)
    handled_signals = (signal.SIGINT, signal.SIGTERM, signal.SIGHUP)
    previous_signal_handlers = {
        current: signal.getsignal(current) for current in handled_signals
    }
    terminal_published = False

    def interrupt_handler(signum: int, _frame: Any) -> None:
        fail("interrupted_by_signal", signal.Signals(signum).name)

    for current in handled_signals:
        signal.signal(current, interrupt_handler)
    output_root: Path | None = None
    runner: Runner | None = None
    cleanup_args: dict[str, Any] | None = None
    cleanup_guard: IntegrityGuard | None = None
    cleanup_systemctl_guard: IntegrityGuard | None = None
    cleanup_resource_context: (
        tuple[Path, Path, Path, Path] | None
    ) = None
    crash_watchdog: CrashWatchdog | None = None
    credential_cleanup_context: (
        tuple[Path, dict[str, Path], str] | None
    ) = None
    try:
        if not REVISION_RE.fullmatch(args.expected_revision):
            fail("expected_revision_invalid")
        for name in (
            "readiness_timeout",
            "collection_timeout",
            "command_timeout",
        ):
            value = getattr(args, name)
            if value < 1 or value > 300:
                fail(f"{name}_invalid")
        live_root = require_directory(args.forbidden_live_root, "forbidden_live_root")
        repository_root = require_directory(args.repository_root, "repository_root")
        backup = require_source_file(
            args.backup,
            args.expected_backup_sha256,
            "backup",
            private=True,
        )
        binary_source = require_source_file(
            args.binary,
            args.expected_binary_sha256,
            "binary",
            executable=True,
        )
        profile_source = require_source_file(
            args.profile_script,
            args.expected_profile_script_sha256,
            "profile_script",
            executable=True,
        )
        adapter_source = require_source_file(
            args.adapter,
            args.expected_adapter_sha256,
            "adapter",
            executable=True,
        )
        template = require_source_file(
            args.config_template,
            args.expected_config_template_sha256,
            "config_template",
            private=True,
        )
        go_command = require_source_file(
            args.go,
            args.expected_go_sha256,
            "go",
            executable=True,
        )
        git_command = require_source_file(
            args.git,
            args.expected_git_sha256,
            "git",
            executable=True,
        )
        bubblewrap = require_source_file(
            args.bubblewrap,
            args.expected_bubblewrap_sha256,
            "bubblewrap",
            executable=True,
        )
        systemctl = require_source_file(
            args.systemctl,
            args.expected_systemctl_sha256,
            "systemctl",
            executable=True,
        )
        systemd_run = require_source_file(
            args.systemd_run,
            args.expected_systemd_run_sha256,
            "systemd_run",
            executable=True,
        )
        proc_root = require_directory(args.proc_root, "proc_root")
        cgroup_mount = require_directory(args.cgroup_mount, "cgroup_mount")
        expected_profile_path = repository_root / "scripts/orquesta_profile_server.sh"
        expected_adapter_path = (
            repository_root / "scripts/orquesta_profile_systemd_user.sh"
        )
        if profile_source != expected_profile_path:
            fail("profile_script_not_current_repository_path")
        if adapter_source != expected_adapter_path:
            fail("adapter_not_current_repository_path")
        for kind, path in (
            ("backup", backup),
            ("binary", binary_source),
            ("repository_root", repository_root),
            ("profile_script", profile_source),
            ("adapter", adapter_source),
            ("config_template", template),
            ("go", go_command),
            ("git", git_command),
            ("bubblewrap", bubblewrap),
        ):
            reject_live_path(path, live_root, kind)
        output_root = ensure_output_target(args.output_dir, live_root)
        expected_migrations, migration_guards, migration_manifest_sha = (
            load_expected_migrations(repository_root)
        )
        harness_helper = canonical_existing(str(Path(__file__)), "harness_helper")
        harness_wrapper = canonical_existing(
            str(Path(__file__).parent.parent / "orquesta_sqlite_v23_reaccredit.sh"),
            "harness_wrapper",
        )
        harness_helper_sha = sha256_file(harness_helper)
        harness_wrapper_sha = sha256_file(harness_wrapper)
        helper_sources = (
            repository_root / "scripts/lib/pidfd_signal.py",
            repository_root / "scripts/lib/firecracker_launcher_probe.py",
            repository_root / "scripts/lib/profile_maintenance_marker.py",
        )
        helper_guards: list[GuardedFile] = []
        for helper in helper_sources:
            checked = canonical_existing(str(helper), "profile_helper")
            checksum = sha256_file(checked)
            require_source_file(
                str(checked),
                checksum,
                f"profile_helper_{checked.stem}",
                executable=True,
            )
            helper_guards.append(
                GuardedFile(f"profile_helper_{checked.stem}", checked, checksum)
            )
        source_guard = IntegrityGuard(
            [
                GuardedFile("backup", backup, args.expected_backup_sha256),
                GuardedFile("binary_source", binary_source, args.expected_binary_sha256),
                GuardedFile(
                    "profile_script_source",
                    profile_source,
                    args.expected_profile_script_sha256,
                ),
                GuardedFile(
                    "adapter_source", adapter_source, args.expected_adapter_sha256
                ),
                GuardedFile(
                    "config_template",
                    template,
                    args.expected_config_template_sha256,
                ),
                GuardedFile("go", go_command, args.expected_go_sha256),
                GuardedFile("git", git_command, args.expected_git_sha256),
                GuardedFile(
                    "bubblewrap", bubblewrap, args.expected_bubblewrap_sha256
                ),
                GuardedFile("systemctl", systemctl, args.expected_systemctl_sha256),
                GuardedFile("systemd_run", systemd_run, args.expected_systemd_run_sha256),
                *migration_guards,
                *helper_guards,
                GuardedFile(
                    "harness_helper", harness_helper, harness_helper_sha
                ),
                GuardedFile(
                    "harness_wrapper", harness_wrapper, harness_wrapper_sha
                ),
            ]
        )
        source_guard.verify()
        source_snapshot, manifest = sqlite_snapshot(
            backup, expected_migrations
        )
        if source_snapshot["user_version"] != 16:
            fail(
                "backup_user_version_not_16",
                str(source_snapshot["user_version"]),
            )
        if any(source_snapshot["migration_counts"].values()):
            fail("backup_contains_v17_v19_migrations")
        source_open = scan_processes(proc_root, binary_source, backup)
        if source_open["database_open_pids"]:
            fail(
                "backup_has_open_processes",
                repr(source_open["database_open_pids"]),
            )
        for suffix in ("-wal", "-shm", "-journal"):
            ancillary = Path(str(backup) + suffix)
            if ancillary.exists() or ancillary.is_symlink():
                fail("backup_ancillary_file_present", str(ancillary))

        output_root.mkdir(mode=0o700)
        os.chmod(output_root, 0o700)
        evidence_root = output_root / "evidence"
        runtime_base = output_root / "runtime"
        account_root = output_root / "accounts"
        projection_root = output_root / "projection"
        projected_repo = projection_root / "repository"
        projected_scripts = projected_repo / "scripts"
        projected_lib = projected_scripts / "lib"
        projected_migrations = (
            projected_repo / "internal/adapters/state/sqlite/migrations"
        )
        binary_root = projection_root / "subject"
        for directory in (
            evidence_root,
            runtime_base,
            account_root,
            projection_root,
            projected_repo,
            projected_scripts,
            projected_lib,
            projected_repo / "internal",
            projected_repo / "internal/adapters",
            projected_repo / "internal/adapters/state",
            projected_repo / "internal/adapters/state/sqlite",
            projected_migrations,
            binary_root,
        ):
            directory.mkdir(mode=0o700, exist_ok=True)
            os.chmod(directory, 0o700)
        migration_files = {
            version: next(
                item
                for item in migration_guards
                if item.path.name == expected_migrations[version]["name"]
            )
            for version in (17, 18, 19)
        }
        expected_schema_manifest = build_expected_v19_schema_manifest(
            backup,
            evidence_root / ".expected-v19-schema.sqlite",
            migration_files,
            expected_migrations,
        )
        runner = Runner(output_root, source_guard, args.command_timeout)
        user_manager_control_group = direct_user_manager_control_group(
            runner, systemctl
        )

        profile = "SqliteReaccredit" + secrets.token_hex(8)
        if not PROFILE_RE.fullmatch(profile):
            fail("generated_profile_invalid")
        unit = f"orquesta-v23-{profile}.service"
        listen = choose_loopback()
        runtime_root = runtime_base / profile
        source_home = account_root / profile
        delegated_cgroup_root = (
            cgroup_mount
            / user_manager_control_group.removeprefix("/")
            / "app.slice"
            / unit
        )
        unit_control_group = delegated_cgroup_root / "orquesta-control"
        source_home.mkdir(mode=0o700)
        os.chmod(source_home, 0o700)
        projected_auth = source_home / "auth.json"
        synthetic_auth = b'{"orquesta_reaccredit_fixture":true}\n'
        write_private(projected_auth, synthetic_auth)
        synthetic_auth_sha = sha256_bytes(synthetic_auth)

        profile = str(profile)
        projected_profile = projected_scripts / "orquesta_profile_server.sh"
        projected_adapter = projected_scripts / "orquesta_profile_systemd_user.sh"
        projected_binary = binary_root / "orquesta"
        copy_private(profile_source, projected_profile, 0o500)
        copy_private(adapter_source, projected_adapter, 0o500)
        copy_private(binary_source, projected_binary, 0o500)
        projected_helpers: list[Path] = []
        for helper in helper_sources:
            projected = projected_lib / helper.name
            copy_private(helper, projected, 0o500)
            projected_helpers.append(projected)
        for version in range(1, 20):
            source = (
                repository_root
                / "internal/adapters/state/sqlite/migrations"
                / expected_migrations[version]["name"]
            )
            copy_private(
                source,
                projected_migrations / source.name,
                0o600,
            )

        config = output_root / "config.toml"
        config_sha, isolated_config_paths = materialize_config(
            template,
            config,
            runtime_root,
            account_root,
            profile,
            listen,
            delegated_cgroup_root,
            projected_repo,
            bubblewrap,
            live_root,
        )
        credential_cleanup_context = (
            projected_auth,
            isolated_config_paths,
            synthetic_auth_sha,
        )
        database = runtime_root / "state" / "orquesta.sqlite"
        database.parent.mkdir(mode=0o700, parents=True)
        os.chmod(database.parent, 0o700)
        source_guard.verify()
        copy_private(backup, database, 0o600)
        if sha256_file(database) != args.expected_backup_sha256:
            fail("private_database_copy_hash_mismatch")

        projected_guard_files = [
            GuardedFile("binary_projected", projected_binary, args.expected_binary_sha256),
            GuardedFile(
                "profile_script_projected",
                projected_profile,
                args.expected_profile_script_sha256,
            ),
            GuardedFile(
                "adapter_projected",
                projected_adapter,
                args.expected_adapter_sha256,
            ),
            GuardedFile("config_materialized", config, config_sha),
            GuardedFile("synthetic_auth_projected", projected_auth, synthetic_auth_sha),
        ]
        for helper, projected in zip(helper_sources, projected_helpers):
            projected_guard_files.append(
                GuardedFile(
                    f"helper_{helper.name}",
                    projected,
                    sha256_file(helper),
                )
            )
        guard = IntegrityGuard([*source_guard.files, *projected_guard_files])
        cleanup_guard = IntegrityGuard(
            [
                *[
                    item
                    for item in projected_guard_files
                    if item.label != "synthetic_auth_projected"
                ],
                GuardedFile(
                    "systemctl", systemctl, args.expected_systemctl_sha256
                ),
                GuardedFile(
                    "systemd_run",
                    systemd_run,
                    args.expected_systemd_run_sha256,
                ),
            ]
        )
        cleanup_systemctl_guard = IntegrityGuard(
            [
                GuardedFile(
                    "systemctl", systemctl, args.expected_systemctl_sha256
                )
            ]
        )
        cleanup_resource_context = (
            proc_root,
            projected_binary,
            database,
            runtime_root,
        )
        guard.verify()
        runner.guard = guard
        buildinfo = runner.run(
            "binary-buildinfo",
            [str(go_command), "version", "-m", str(projected_binary)],
        )
        observed_revision, binary_vcs_modified = parse_go_buildinfo(
            buildinfo.stdout
        )
        if observed_revision != args.expected_revision or binary_vcs_modified:
            fail(
                "binary_revision_mismatch",
                f"revision={observed_revision}:modified={binary_vcs_modified}",
            )
        resolved_git = shutil.which("git", path=args.exec_path)
        if (
            resolved_git is None
            or Path(resolved_git).resolve(strict=True) != git_command
        ):
            fail("exec_path_git_identity_mismatch")
        runner.run(
            "repository-init",
            [
                str(git_command),
                "-C",
                str(projected_repo),
                "init",
                "--quiet",
                "--initial-branch=main",
            ],
        )
        runner.run(
            "repository-add",
            [
                str(git_command),
                "-C",
                str(projected_repo),
                "add",
                "--",
                "scripts",
                "internal",
            ],
        )
        runner.run(
            "repository-commit",
            [
                str(git_command),
                "-C",
                str(projected_repo),
                "-c",
                "user.name=Orquesta Reaccredit",
                "-c",
                "user.email=reaccredit@invalid",
                "-c",
                "commit.gpgSign=false",
                "-c",
                "core.hooksPath=/dev/null",
                "commit",
                "--quiet",
                "-m",
                "fixture privada de reacreditación",
            ],
        )
        repository_revision = runner.run(
            "repository-revision",
            [
                str(git_command),
                "-C",
                str(projected_repo),
                "rev-parse",
                "HEAD",
            ],
        ).stdout.strip()
        if not REVISION_RE.fullmatch(repository_revision):
            fail("projected_repository_revision_invalid")
        for current, directories, files in os.walk(
            projected_repo / ".git", followlinks=False
        ):
            os.chmod(current, 0o700)
            for name in directories:
                os.chmod(Path(current) / name, 0o700, follow_symlinks=False)
            for name in files:
                os.chmod(Path(current) / name, 0o600, follow_symlinks=False)
        if direct_unit_state(runner, systemctl, unit, "preflight-unit") != "not-found":
            fail("generated_unit_already_exists")
        common = {
            "unit": unit,
            "profile": profile,
            "repository_root": projected_repo,
            "profile_script": projected_profile,
            "profile_script_sha": args.expected_profile_script_sha256,
            "runtime_base": runtime_base,
            "source_home": source_home,
            "binary": projected_binary,
            "binary_sha": args.expected_binary_sha256,
            "config": config,
            "config_sha": config_sha,
            "exec_path": args.exec_path,
            "systemctl": systemctl,
            "systemctl_sha": args.expected_systemctl_sha256,
            "systemd_run": systemd_run,
            "systemd_run_sha": args.expected_systemd_run_sha256,
            "proc_root": proc_root,
            "cgroup_mount": cgroup_mount,
            "readiness_timeout": args.readiness_timeout,
            "collection_timeout": args.collection_timeout,
        }
        cleanup_args = {
            "adapter": projected_adapter,
            **common,
        }
        crash_watchdog = arm_crash_watchdog(
            cleanup_args=cleanup_args,
            adapter_guard=cleanup_guard,
            systemctl_guard=cleanup_systemctl_guard,
            timeout=args.command_timeout,
            proc_root=proc_root,
            binary=projected_binary,
            database=database,
            runtime_root=runtime_root,
            output_root=output_root,
            credential_context=credential_cleanup_context,
        )
        runner.watchdog = crash_watchdog
        cycles: list[dict[str, Any]] = []
        snapshots: list[dict[str, Any]] = []
        for cycle in (1, 2):
            start = runner.run(
                f"cycle-{cycle}-start",
                adapter_arguments(projected_adapter, "start", **common),
            )
            start_invocation = parse_invocation(start.stdout, "start")
            main_pid, daemon_pid = parse_runtime_pids(start.stdout)
            observed_control_group = direct_control_group(
                runner,
                systemctl,
                unit,
                f"cycle-{cycle}-control-group",
            )
            observed_cgroup_root = (
                cgroup_mount
                / observed_control_group.removeprefix("/")
            )
            if observed_cgroup_root != delegated_cgroup_root:
                fail(
                    "unit_control_group_config_mismatch",
                    (
                        f"expected={delegated_cgroup_root}:"
                        f"observed={observed_cgroup_root}"
                    ),
                )
            expected_process_cgroup = (
                observed_control_group + "/orquesta-control"
            )
            for role, pid in (
                ("main", main_pid),
                ("daemon", daemon_pid),
            ):
                if read_process_cgroup(proc_root, pid) != expected_process_cgroup:
                    fail(
                        "runtime_process_cgroup_mismatch",
                        f"cycle={cycle}:role={role}:pid={pid}",
                    )
            status = runner.run(
                f"cycle-{cycle}-status",
                adapter_arguments(projected_adapter, "check", **common),
            )
            status_invocation = parse_invocation(status.stdout, "check")
            if status_invocation != start_invocation:
                fail("adapter_invocation_changed_within_cycle")
            runner.run(
                f"cycle-{cycle}-stop",
                adapter_arguments(projected_adapter, "stop-profile", **common),
            )
            runner.run(
                f"cycle-{cycle}-collect",
                adapter_arguments(projected_adapter, "collect", **common),
            )
            unit_state = direct_unit_state(
                runner, systemctl, unit, f"cycle-{cycle}-unit-collected"
            )
            if unit_state != "not-found":
                fail("unit_not_collected", f"cycle={cycle}:state={unit_state}")
            profile_status = runner.run(
                f"cycle-{cycle}-profile-status",
                [
                    str(projected_profile),
                    "status",
                    "--profile",
                    profile,
                    "--runtime-base",
                    str(runtime_base),
                ],
                allowed=frozenset({3}),
            )
            if (
                f"status=stopped profile={profile}" not in profile_status.stdout
                or profile_status.stderr
            ):
                fail("profile_not_stopped", f"cycle={cycle}")
            snapshot, _ = sqlite_snapshot(
                database, expected_migrations, manifest
            )
            if snapshot["user_version"] != 19:
                fail("migration_did_not_reach_v19", f"cycle={cycle}")
            if snapshot["migration_counts"] != {"17": 1, "18": 1, "19": 1}:
                fail("v17_v19_migration_count_invalid", f"cycle={cycle}")
            if snapshot["functional_digest"] != source_snapshot["functional_digest"]:
                fail("functional_digest_changed", f"cycle={cycle}")
            if not functional_counts_equal(source_snapshot, snapshot):
                fail("functional_counts_changed", f"cycle={cycle}")
            if not audit_delta_valid(source_snapshot, snapshot, cycle):
                fail("status_audit_delta_invalid", f"cycle={cycle}")
            if snapshot["schema_manifest"] != expected_schema_manifest:
                fail("sqlite_schema_manifest_changed", f"cycle={cycle}")
            cycles.append(
                {
                    "cycle": cycle,
                    "profile": profile,
                    "unit": unit,
                    "invocation_id": start_invocation,
                    "unit_after_collect": unit_state,
                }
            )
            snapshots.append(snapshot)

        guard.verify()
        if sha256_file(backup) != args.expected_backup_sha256:
            fail("backup_mutated")
        residues = scan_processes(proc_root, projected_binary, database)
        residues["unit_load_state"] = direct_unit_state(
            runner, systemctl, unit, "final-unit"
        )
        residues["profile_status"] = "stopped"
        residues["identity_files"] = [
            str(path.relative_to(output_root))
            for path in (
                runtime_root / "run/server.pid",
                runtime_root / "run/server.start_ref",
                runtime_root / "run/server.binary_id",
                runtime_root / "run/server.binary_sha256",
                runtime_root / "run/server.config_sha256",
            )
            if path.exists() or path.is_symlink()
        ]
        residues["sqlite_ancillary_files"] = [
            str(path.relative_to(output_root))
            for path in (
                Path(str(database) + "-wal"),
                Path(str(database) + "-shm"),
                Path(str(database) + "-journal"),
            )
            if path.exists() or path.is_symlink()
        ]
        if (
            residues["candidate_pids"]
            or residues["database_open_pids"]
            or residues["identity_files"]
            or residues["sqlite_ancillary_files"]
            or residues["unit_load_state"] != "not-found"
        ):
            fail("residual_resources_present", repr(residues))
        if cycles[0]["invocation_id"] == cycles[1]["invocation_id"]:
            fail("systemd_invocation_reused")
        remove_credential_projections(
            output_root,
            projected_auth,
            isolated_config_paths,
            synthetic_auth_sha,
        )

        executable_paths = {
            projected_binary,
            projected_profile,
            projected_adapter,
            *projected_helpers,
        }
        validate_private_tree(output_root, executable_paths)
        receipt_path = output_root / "receipt.json"
        result_database_sha = sha256_file(database)
        receipt: dict[str, Any] = {
            "schema_version": SCHEMA,
            "result": "pass",
            "candidate": {
                "binary_path": str(binary_source),
                "binary_sha256": args.expected_binary_sha256,
                "repository_revision": observed_revision,
                "binary_vcs_modified": binary_vcs_modified,
                "profile_script_sha256": args.expected_profile_script_sha256,
                "systemd_adapter_sha256": args.expected_adapter_sha256,
                "profile_helper_sha256": {
                    item.path.name: item.sha256 for item in helper_guards
                },
                "config_template_sha256": args.expected_config_template_sha256,
                "materialized_config_sha256": config_sha,
                "migration_manifest_sha256": migration_manifest_sha,
                "go_sha256": args.expected_go_sha256,
                "git_sha256": args.expected_git_sha256,
                "bubblewrap_sha256": args.expected_bubblewrap_sha256,
                "systemctl_sha256": args.expected_systemctl_sha256,
                "systemd_run_sha256": args.expected_systemd_run_sha256,
                "harness_wrapper_sha256": harness_wrapper_sha,
                "harness_helper_sha256": harness_helper_sha,
            },
            "source": {
                "database_sha256": args.expected_backup_sha256,
                "user_version": source_snapshot["user_version"],
                "quick_check": source_snapshot["quick_check"],
                "foreign_key_check_rows": source_snapshot[
                    "foreign_key_violations"
                ],
                "functional_data_sha256": source_snapshot["functional_digest"],
                "table_counts": source_snapshot["table_counts"],
                "core_counts": source_snapshot["core_counts"],
                "schema_migrations": source_snapshot["migrations"],
            },
            "first_start": {
                "source_user_version": source_snapshot["user_version"],
                "result_user_version": snapshots[0]["user_version"],
                "schema_migrations_added": [17, 18, 19],
                "system_status": "passed",
                "invocation_id": cycles[0]["invocation_id"],
                "quick_check": snapshots[0]["quick_check"],
                "foreign_key_check_rows": snapshots[0][
                    "foreign_key_violations"
                ],
                "functional_data_sha256": snapshots[0]["functional_digest"],
                "table_counts": snapshots[0]["table_counts"],
                "core_counts": snapshots[0]["core_counts"],
                "audit_counts": snapshots[0]["audit_counts"],
            },
            "restart": {
                "source_user_version": snapshots[0]["user_version"],
                "result_user_version": snapshots[1]["user_version"],
                "schema_migrations_duplicated": False,
                "system_status": "passed",
                "invocation_id": cycles[1]["invocation_id"],
                "quick_check": snapshots[1]["quick_check"],
                "foreign_key_check_rows": snapshots[1][
                    "foreign_key_violations"
                ],
                "functional_data_sha256": snapshots[1]["functional_digest"],
                "table_counts": snapshots[1]["table_counts"],
                "core_counts": snapshots[1]["core_counts"],
                "audit_counts": snapshots[1]["audit_counts"],
            },
            "result_database": {
                "path": str(database.relative_to(output_root)),
                "sha256": result_database_sha,
                "quick_check": snapshots[1]["quick_check"],
                "foreign_key_check_rows": snapshots[1][
                    "foreign_key_violations"
                ],
                "user_version": snapshots[1]["user_version"],
                "functional_data_sha256": snapshots[1]["functional_digest"],
                "table_counts": snapshots[1]["table_counts"],
                "core_counts": snapshots[1]["core_counts"],
                "schema_migrations": snapshots[1]["migrations"],
            },
            "closure": {
                "candidate_processes": len(residues["candidate_pids"]),
                "database_open_processes": len(
                    residues["database_open_pids"]
                ),
                "live_profile_touched": False,
                "unit_load_state": residues["unit_load_state"],
                "profile_status": residues["profile_status"],
                "identity_files": len(residues["identity_files"]),
                "sqlite_ancillary_files": len(
                    residues["sqlite_ancillary_files"]
                ),
                "credential_projections": 0,
            },
            "harness": {
                "created_at": utc_now(),
                "output_dir": str(output_root),
                "forbidden_live_root": str(live_root),
                "profile": profile,
                "unit": unit,
                "listen": listen,
                "delegated_cgroup_root": str(delegated_cgroup_root),
                "unit_control_group": str(unit_control_group),
                "projected_repository_revision": repository_revision,
                "directory_mode": "0700",
                "data_file_mode": "0600",
                "executable_projection_mode": "0500",
                "checks": {
                    "input_hashes_stable": True,
                    "source_backup_unchanged": True,
                    "user_version_16_19_19": True,
                    "migrations_17_18_19_once": True,
                    "quick_check_and_foreign_keys": True,
                    "functional_digest_and_counts_equal": True,
                    "schema_manifest_exact": True,
                    "one_status_audit_per_cycle": True,
                    "invocation_ids_distinct": True,
                    "config_paths_isolated": True,
                    "credentials_removed": True,
                    "zero_residual_resources": True,
                },
            },
            "evidence": [item.receipt(output_root) for item in runner.items],
        }
        for current in handled_signals:
            signal.signal(current, signal.SIG_IGN)
        receipt_sha = publish_pass_with_watchdog(
            crash_watchdog,
            receipt_path,
            receipt,
            args.command_timeout,
        )
        terminal_published = True
        try:
            print(
                f"{PROGRAM}: status=accredited receipt={receipt_path} "
                f"receipt_sha256={receipt_sha}"
            )
        except OSError:
            pass
        return 0
    except (HarnessError, KeyboardInterrupt, OSError) as caught:
        for current in handled_signals:
            signal.signal(current, signal.SIG_IGN)
        if isinstance(caught, HarnessError):
            error = caught
        elif isinstance(caught, KeyboardInterrupt):
            error = HarnessError("interrupted_by_signal", "SIGINT")
        else:
            error = HarnessError(
                "unexpected_os_error",
                f"{type(caught).__name__}:{caught}",
            )
        cleanup_attempts: list[dict[str, Any]] = []
        cleanup_residues: dict[str, Any] = {
            "unit_load_state": "unknown",
            "candidate_pids": [],
            "database_open_pids": [],
            "identity_files": [],
            "sqlite_ancillary_files": [],
            "credential_projections": [],
            "scan_errors": ["cleanup_context_unavailable"],
            "cleanup_errors": [],
        }
        if output_root is not None and output_root.exists():
            if not terminal_published and (output_root / "receipt.json").exists():
                try:
                    remove_terminal_artifacts(output_root)
                except OSError:
                    pass
            if (
                cleanup_args is not None
                and cleanup_guard is not None
                and cleanup_systemctl_guard is not None
                and cleanup_resource_context is not None
                and credential_cleanup_context is not None
            ):
                try:
                    cleanup_attempts, cleanup_residues = (
                        perform_failure_cleanup(
                            cleanup_args=cleanup_args,
                            adapter_guard=cleanup_guard,
                            systemctl_guard=cleanup_systemctl_guard,
                            timeout=args.command_timeout,
                            proc_root=cleanup_resource_context[0],
                            binary=cleanup_resource_context[1],
                            database=cleanup_resource_context[2],
                            runtime_root=cleanup_resource_context[3],
                            output_root=output_root,
                            credential_context=credential_cleanup_context,
                            watchdog=crash_watchdog,
                        )
                    )
                except (HarnessError, KeyError, OSError) as cleanup_error:
                    cleanup_residues["cleanup_errors"].append(
                        type(cleanup_error).__name__
                    )
            elif credential_cleanup_context is not None:
                try:
                    remove_credential_projections(
                        output_root, *credential_cleanup_context
                    )
                except HarnessError:
                    pass
            failure_path = output_root / "failure.json"
            if not failure_path.exists():
                try:
                    write_terminal_receipt(
                        failure_path,
                        {
                            "schema_version": SCHEMA,
                            "result": "fail",
                            "created_at": utc_now(),
                            "reason_code": error.code,
                            "detail": error.detail,
                            "cleanup_attempts": cleanup_attempts,
                            "cleanup_verified": cleanup_verified(
                                cleanup_residues
                            ),
                            "residues": cleanup_residues,
                        },
                    )
                    terminal_published = True
                except (OSError, HarnessError):
                    pass
            if (
                terminal_published
                and crash_watchdog is not None
                and crash_watchdog.armed
            ):
                try:
                    commit_crash_watchdog(
                        crash_watchdog, args.command_timeout
                    )
                except HarnessError:
                    terminal_published = False
        print(
            f"{PROGRAM}: status=error reason_code={error.code}"
            + (f" detail={json.dumps(error.detail)}" if error.detail else ""),
            file=sys.stderr,
        )
        return 1
    finally:
        if crash_watchdog is not None and crash_watchdog.armed:
            trigger_crash_watchdog(crash_watchdog)
        if not terminal_published:
            for current, previous in previous_signal_handlers.items():
                signal.signal(current, previous)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
