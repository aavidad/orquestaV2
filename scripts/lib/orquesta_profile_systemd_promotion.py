#!/usr/bin/env python3
"""Validaciones y backup sellado para la promoción systemd de un perfil."""

from __future__ import annotations

import argparse
import base64
import copy
import ctypes
import hashlib
import json
import os
import pathlib
import re
import sqlite3
import stat
import sys
import tempfile
import tomllib
import urllib.parse


HEX = re.compile(r"^[0-9a-f]{64}$")
REVISION = re.compile(r"^[0-9a-f]{40}$")
INVOCATION_ID = re.compile(r"^[0-9a-f]{32}$")
REACCREDIT_PROFILE = re.compile(r"^SqliteReaccredit[0-9a-f]{16}$")
BACKUP_SCHEMA = "orquesta_profile_systemd_promotion_backup.v1"
RECEIPT_SCHEMA = "orquesta_profile_systemd_promotion_receipt.v1"
UPGRADE_RECEIPT_SCHEMA = "orquesta_sqlite_upgrade_audit.v1"
FUNCTIONAL_AUDIT_TABLES = {
    "authorization_receipts",
    "command_invocations",
    "command_outcomes",
}
CORE_TABLES = ("goals", "executions", "work_items", "outbox", "effect_attempts")
V23_TABLES = (
    "intake_states",
    "intake_receipts",
    "intake_dossiers",
    "intake_dossier_generation_receipts",
    "intake_dossier_confirmations",
)
PROFILE_HELPERS = (
    "firecracker_launcher_probe.py",
    "pidfd_signal.py",
    "profile_maintenance_marker.py",
)
EVIDENCE_LABELS = (
    "user-manager-control-group",
    "binary-buildinfo",
    "repository-init",
    "repository-add",
    "repository-commit",
    "repository-revision",
    "preflight-unit",
    "cycle-1-start",
    "cycle-1-control-group",
    "cycle-1-status",
    "cycle-1-stop",
    "cycle-1-collect",
    "cycle-1-unit-collected",
    "cycle-1-profile-status",
    "cycle-2-start",
    "cycle-2-control-group",
    "cycle-2-status",
    "cycle-2-stop",
    "cycle-2-collect",
    "cycle-2-unit-collected",
    "cycle-2-profile-status",
    "final-unit",
)
RENAME_NOREPLACE = 1
AT_FDCWD = -100


class ContractError(Exception):
    """A machine-readable, non-sensitive contract failure."""


def strict_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            raise ContractError("duplicate_key")
        result[key] = value
    return result


def secure_read(
    path: str,
    *,
    maximum: int,
    expected_uid: int | None = None,
    allowed_modes: tuple[int, ...] = (0o400, 0o600),
) -> tuple[bytes, os.stat_result]:
    flags = os.O_RDONLY | os.O_CLOEXEC | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        before = os.fstat(descriptor)
        if (
            not stat.S_ISREG(before.st_mode)
            or stat.S_IMODE(before.st_mode) not in allowed_modes
            or before.st_nlink != 1
            or (expected_uid is not None and before.st_uid != expected_uid)
            or not 0 < before.st_size <= maximum
        ):
            raise ContractError("file_metadata")
        blocks = bytearray()
        while block := os.read(descriptor, min(1024 * 1024, maximum + 1)):
            blocks.extend(block)
            if len(blocks) > maximum:
                raise ContractError("file_size")
        after = os.fstat(descriptor)
        identity_before = (
            before.st_dev,
            before.st_ino,
            before.st_mode,
            before.st_nlink,
            before.st_size,
            before.st_mtime_ns,
            before.st_ctime_ns,
        )
        identity_after = (
            after.st_dev,
            after.st_ino,
            after.st_mode,
            after.st_nlink,
            after.st_size,
            after.st_mtime_ns,
            after.st_ctime_ns,
        )
        if identity_before != identity_after or len(blocks) != before.st_size:
            raise ContractError("file_changed")
        return bytes(blocks), before
    finally:
        os.close(descriptor)


def read_json(
    path: str, maximum: int, expected_uid: int | None = None
) -> dict[str, object]:
    raw, _ = secure_read(path, maximum=maximum, expected_uid=expected_uid)
    value = json.loads(raw, object_pairs_hook=strict_object)
    if not isinstance(value, dict):
        raise ContractError("json_root")
    return value


def read_toml(path: str) -> dict[str, object]:
    raw, _ = secure_read(path, maximum=4 * 1024 * 1024, expected_uid=os.getuid())
    value = tomllib.loads(raw.decode("utf-8"))
    if not isinstance(value, dict):
        raise ContractError("toml_root")
    return value


def nested(document: dict[str, object], *keys: str) -> object:
    value: object = document
    for key in keys:
        if not isinstance(value, dict) or key not in value:
            raise ContractError("config_key")
        value = value[key]
    return value


def secure_file_sha256(path: str, uid: int) -> tuple[str, os.stat_result]:
    flags = os.O_RDONLY | os.O_CLOEXEC | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        before = os.fstat(descriptor)
        if (
            not stat.S_ISREG(before.st_mode)
            or stat.S_IMODE(before.st_mode) != 0o600
            or before.st_uid != uid
            or before.st_nlink != 1
        ):
            raise ContractError("private_file")
        digest = hashlib.sha256()
        size = 0
        while block := os.read(descriptor, 1024 * 1024):
            digest.update(block)
            size += len(block)
        after = os.fstat(descriptor)
        identity_before = (
            before.st_dev,
            before.st_ino,
            before.st_mode,
            before.st_nlink,
            before.st_size,
            before.st_mtime_ns,
            before.st_ctime_ns,
        )
        identity_after = (
            after.st_dev,
            after.st_ino,
            after.st_mode,
            after.st_nlink,
            after.st_size,
            after.st_mtime_ns,
            after.st_ctime_ns,
        )
        if identity_before != identity_after or size != before.st_size:
            raise ContractError("file_changed")
        return digest.hexdigest(), before
    finally:
        os.close(descriptor)


def require_regular_private(path: str, uid: int) -> os.stat_result:
    metadata = os.lstat(path)
    if (
        not stat.S_ISREG(metadata.st_mode)
        or stat.S_IMODE(metadata.st_mode) != 0o600
        or metadata.st_uid != uid
        or metadata.st_nlink != 1
    ):
        raise ContractError("private_file")
    return metadata


def connect_read_only(path: str) -> sqlite3.Connection:
    uri = "file:" + urllib.parse.quote(path, safe="/") + "?mode=ro"
    connection = sqlite3.connect(uri, uri=True)
    connection.execute("PRAGMA query_only=ON")
    return connection


def sqlite_facts(
    connection: sqlite3.Connection,
) -> tuple[list[str], int, list[tuple[object, ...]], str, str]:
    quick = [row[0] for row in connection.execute("PRAGMA quick_check")]
    version = connection.execute("PRAGMA user_version").fetchone()[0]
    migrations = list(
        connection.execute(
            "SELECT version,name,checksum FROM schema_migrations ORDER BY version"
        )
    )
    migrations_sha = hashlib.sha256(
        json.dumps(migrations, separators=(",", ":"), ensure_ascii=True).encode("ascii")
    ).hexdigest()
    logical = hashlib.sha256()
    for statement in connection.iterdump():
        logical.update(statement.encode("utf-8"))
        logical.update(b"\n")
    return quick, version, migrations, migrations_sha, logical.hexdigest()


def quote_identifier(value: str) -> str:
    return '"' + value.replace('"', '""') + '"'


def encode_sqlite_value(value: object) -> bytes:
    if value is None:
        return b"n:"
    if isinstance(value, int):
        return b"i:" + str(value).encode("ascii")
    if isinstance(value, float):
        return b"f:" + value.hex().encode("ascii")
    if isinstance(value, str):
        return b"s:" + value.encode("utf-8")
    if isinstance(value, bytes):
        return b"b:" + base64.b64encode(value)
    raise ContractError("functional_value_type")


def functional_projection(
    connection: sqlite3.Connection,
) -> list[tuple[str, tuple[str, ...]]]:
    tables = [
        row[0]
        for row in connection.execute(
            "SELECT name FROM sqlite_schema "
            "WHERE type='table' AND name NOT LIKE 'sqlite_%' "
            "AND name <> 'schema_migrations' ORDER BY name"
        )
    ]
    tables = [table for table in tables if table not in FUNCTIONAL_AUDIT_TABLES]
    projection: list[tuple[str, tuple[str, ...]]] = []
    for table in tables:
        columns = tuple(
            row[1]
            for row in connection.execute(
                f"PRAGMA table_info({quote_identifier(table)})"
            )
        )
        if not columns:
            raise ContractError("functional_table_columns")
        projection.append((table, columns))
    return projection


def functional_data_sha256(
    connection: sqlite3.Connection,
    projection: list[tuple[str, tuple[str, ...]]],
) -> str:
    digest = hashlib.sha256()
    for table, columns in projection:
        digest.update(b"table\0" + table.encode("utf-8") + b"\0")
        for column in columns:
            digest.update(b"column\0" + column.encode("utf-8") + b"\0")
        selected = ",".join(quote_identifier(column) for column in columns)
        ordering = ",".join(
            f"typeof({quote_identifier(column)}),quote({quote_identifier(column)})"
            for column in columns
        )
        query = f"SELECT {selected} FROM {quote_identifier(table)} ORDER BY {ordering}"
        try:
            rows = connection.execute(query)
            for row in rows:
                digest.update(b"row\0")
                for value in row:
                    encoded = encode_sqlite_value(value)
                    digest.update(str(len(encoded)).encode("ascii"))
                    digest.update(b":")
                    digest.update(encoded)
                    digest.update(b"\0")
        except sqlite3.Error as error:
            raise ContractError("functional_projection_changed") from error
    return digest.hexdigest()


def rename_noreplace(source: str, target: str) -> None:
    libc = ctypes.CDLL(None, use_errno=True)
    renameat2 = getattr(libc, "renameat2", None)
    if renameat2 is None:
        raise ContractError("renameat2_unavailable")
    renameat2.argtypes = [
        ctypes.c_int,
        ctypes.c_char_p,
        ctypes.c_int,
        ctypes.c_char_p,
        ctypes.c_uint,
    ]
    renameat2.restype = ctypes.c_int
    if (
        renameat2(
            AT_FDCWD,
            os.fsencode(source),
            AT_FDCWD,
            os.fsencode(target),
            RENAME_NOREPLACE,
        )
        != 0
    ):
        raise ContractError("atomic_publish_conflict")


def fsync_directory(path: str) -> None:
    descriptor = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def publish_private_record(path: str, content: bytes, directory: str) -> None:
    if os.path.lexists(path):
        observed, _ = secure_read(
            path,
            maximum=max(len(content), 1),
            expected_uid=os.getuid(),
            allowed_modes=(0o600,),
        )
        if observed != content:
            raise ContractError("record_conflict")
        return
    descriptor, temporary = tempfile.mkstemp(prefix=".promotion-record-", dir=directory)
    try:
        os.fchmod(descriptor, 0o600)
        os.write(descriptor, content)
        os.fsync(descriptor)
    finally:
        os.close(descriptor)
    try:
        rename_noreplace(temporary, path)
        temporary = ""
        fsync_directory(directory)
    finally:
        if temporary and os.path.exists(temporary):
            os.unlink(temporary)


def command_publish_record(args: argparse.Namespace) -> None:
    directory = pathlib.Path(args.directory)
    target = pathlib.Path(args.path)
    metadata = os.lstat(directory)
    if (
        not stat.S_ISDIR(metadata.st_mode)
        or stat.S_IMODE(metadata.st_mode) != 0o700
        or metadata.st_uid != os.getuid()
        or directory.resolve(strict=True) != directory
        or target.parent != directory
        or "\x00" in args.content
        or args.content.endswith("\n")
    ):
        raise ContractError("record_target")
    try:
        content = (args.content + "\n").encode("ascii")
    except UnicodeEncodeError as error:
        raise ContractError("record_ascii") from error
    if not 0 < len(content) <= 65536:
        raise ContractError("record_size")
    publish_private_record(str(target), content, str(directory))


def command_remove_record(args: argparse.Namespace) -> None:
    directory = pathlib.Path(args.directory)
    target = pathlib.Path(args.path)
    metadata = os.lstat(directory)
    if (
        not stat.S_ISDIR(metadata.st_mode)
        or stat.S_IMODE(metadata.st_mode) != 0o700
        or metadata.st_uid != os.getuid()
        or directory.resolve(strict=True) != directory
        or target.parent != directory
        or "\x00" in args.content
        or args.content.endswith("\n")
    ):
        raise ContractError("record_target")
    try:
        expected = (args.content + "\n").encode("ascii")
    except UnicodeEncodeError as error:
        raise ContractError("record_ascii") from error
    observed, _ = secure_read(
        str(target),
        maximum=max(len(expected), 1),
        expected_uid=os.getuid(),
        allowed_modes=(0o600,),
    )
    if observed != expected:
        raise ContractError("record_conflict")
    os.unlink(target)
    fsync_directory(str(directory))


def command_final_receipt(args: argparse.Namespace) -> None:
    raw, _ = secure_read(
        args.path,
        maximum=8192,
        expected_uid=args.owner_uid,
        allowed_modes=(0o600,),
    )
    try:
        text = raw.decode("ascii")
    except UnicodeDecodeError as error:
        raise ContractError("final_receipt_ascii") from error
    if not text.endswith("\n"):
        raise ContractError("final_receipt_framing")
    lines = text.splitlines()
    if len(lines) != 14:
        raise ContractError("final_receipt_framing")
    values: dict[str, str] = {}
    for line in lines:
        if "=" not in line:
            raise ContractError("final_receipt_line")
        key, value = line.split("=", 1)
        if key in values:
            raise ContractError("final_receipt_duplicate")
        values[key] = value
    expected_keys = {
        "schema",
        "contract_sha256",
        "result",
        "binary_sha256",
        "config_sha256",
        "backup_sha256",
        "sqlite_user_version",
        "firecracker_config_active",
        "firecracker_root_gate_ref",
        "firecracker_root_evidence_scope",
        "firecracker_application_attestation",
        "backup_restore_performed",
        "old_unit_outcome",
        "functional_data_sha256",
    }
    result = values.get("result")
    if result == "primary":
        expected_config = args.primary_config_sha
        expected_active = "true"
    elif result == "rollback":
        expected_config = args.rollback_config_sha
        expected_active = "false"
    else:
        raise ContractError("final_receipt_result")
    if (
        set(values) != expected_keys
        or values.get("schema") != RECEIPT_SCHEMA
        or values.get("contract_sha256") != args.contract
        or values.get("binary_sha256") != args.binary_sha
        or values.get("config_sha256") != expected_config
        or values.get("backup_sha256") != args.backup_sha
        or values.get("sqlite_user_version") != "19"
        or values.get("firecracker_config_active") != expected_active
        or values.get("firecracker_root_gate_ref") != args.root_gate_ref
        or values.get("firecracker_root_evidence_scope") != "operator_reference_only"
        or values.get("firecracker_application_attestation") != "pending"
        or values.get("backup_restore_performed") != "false"
        or values.get("old_unit_outcome") != args.old_unit_outcome
        or values.get("functional_data_sha256") != args.functional_data_sha
        or not HEX.fullmatch(values.get("functional_data_sha256", ""))
    ):
        raise ContractError("final_receipt_contract")
    print(result)


def command_configs(args: argparse.Namespace) -> None:
    primary = read_toml(args.primary)
    rollback = read_toml(args.rollback)
    primary_shared = copy.deepcopy(primary)
    rollback_shared = copy.deepcopy(rollback)
    for document in (primary_shared, rollback_shared):
        attestor = nested(document, "test_attestor")
        if not isinstance(attestor, dict):
            raise ContractError("config_attestor")
        for key in (
            "provider",
            "max_concurrent_runs",
            "microvm",
            "bubblewrap",
        ):
            attestor.pop(key, None)
    if primary_shared != rollback_shared:
        raise ContractError("config_shared_drift")
    shared = (
        ("state", "sqlite", "path"),
        ("runtime", "codex", "cgroup_root"),
        ("test_attestor", "resources", "cgroup_root"),
        ("config", "effective_path"),
        ("server", "listen"),
        ("identity", "local_token_path"),
        ("repository", "local", "seed_path"),
        ("repository", "local", "target_ref"),
        ("runtime", "codex", "account_profile"),
    )
    if any(nested(primary, *path) != nested(rollback, *path) for path in shared):
        raise ContractError("config_shared_drift")
    expected = (
        (nested(primary, "state", "sqlite", "path"), args.sqlite),
        (nested(primary, "runtime", "codex", "cgroup_root"), args.cgroup),
        (
            nested(primary, "test_attestor", "resources", "cgroup_root"),
            args.cgroup,
        ),
        (nested(primary, "repository", "local", "seed_path"), args.repository),
        (nested(primary, "repository", "local", "target_ref"), args.target_ref),
        (nested(primary, "runtime", "codex", "account_profile"), args.profile),
        (nested(primary, "test_attestor", "provider"), "microvm"),
        (
            nested(primary, "test_attestor", "max_concurrent_runs"),
            args.primary_concurrency,
        ),
        (
            nested(primary, "test_attestor", "microvm", "launcher_socket"),
            args.launcher_socket,
        ),
        (
            nested(
                primary,
                "test_attestor",
                "microvm",
                "expected_asset_digest",
            ),
            args.asset_digest,
        ),
        (nested(rollback, "test_attestor", "provider"), "bubblewrap"),
        (
            nested(rollback, "test_attestor", "max_concurrent_runs"),
            args.rollback_concurrency,
        ),
        (
            nested(rollback, "test_attestor", "bubblewrap", "command"),
            args.bubblewrap,
        ),
    )
    if any(observed != wanted for observed, wanted in expected):
        raise ContractError("config_contract")
    effective = nested(primary, "config", "effective_path")
    if (
        not isinstance(effective, str)
        or not pathlib.PurePosixPath(effective).is_absolute()
    ):
        raise ContractError("effective_path")
    print(effective)


def require_exact_keys(
    value: object, expected: set[str], code: str
) -> dict[str, object]:
    if not isinstance(value, dict) or set(value) != expected:
        raise ContractError(code)
    return value


def require_string(value: object, code: str) -> str:
    if type(value) is not str or not value:
        raise ContractError(code)
    return value


def require_hex(value: object, code: str) -> str:
    text = require_string(value, code)
    if not HEX.fullmatch(text):
        raise ContractError(code)
    return text


def require_integer(value: object, code: str) -> int:
    if type(value) is not int or value < 0:
        raise ContractError(code)
    return value


def require_count_map(
    value: object, code: str, expected_keys: set[str] | None = None
) -> dict[str, int]:
    if not isinstance(value, dict) or (
        expected_keys is not None and set(value) != expected_keys
    ):
        raise ContractError(code)
    result: dict[str, int] = {}
    for key, count in value.items():
        if (
            type(key) is not str
            or not key
            or "\x00" in key
            or type(count) is not int
            or count < 0
        ):
            raise ContractError(code)
        result[key] = count
    return result


def stable_source_sha256(path: pathlib.Path, owner_uid: int) -> str:
    flags = os.O_RDONLY | os.O_CLOEXEC | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        before = os.fstat(descriptor)
        if (
            not stat.S_ISREG(before.st_mode)
            or before.st_uid not in {0, owner_uid}
            or stat.S_IMODE(before.st_mode) & 0o022
            or before.st_nlink != 1
        ):
            raise ContractError("upgrade_source_metadata")
        digest = hashlib.sha256()
        size = 0
        while block := os.read(descriptor, 1024 * 1024):
            digest.update(block)
            size += len(block)
        after = os.fstat(descriptor)
        if (before.st_dev, before.st_ino, before.st_mode, before.st_size) != (
            after.st_dev,
            after.st_ino,
            after.st_mode,
            after.st_size,
        ) or size != before.st_size:
            raise ContractError("upgrade_source_changed")
        return digest.hexdigest()
    finally:
        os.close(descriptor)


def private_output_bytes(
    path: pathlib.Path,
    owner_uid: int,
    maximum: int = 1024 * 1024,
    allowed_modes: tuple[int, ...] = (0o600,),
) -> bytes:
    flags = os.O_RDONLY | os.O_CLOEXEC | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        before = os.fstat(descriptor)
        if (
            not stat.S_ISREG(before.st_mode)
            or stat.S_IMODE(before.st_mode) not in allowed_modes
            or before.st_uid != owner_uid
            or before.st_nlink != 1
            or before.st_size > maximum
        ):
            raise ContractError("upgrade_output_file_metadata")
        content = bytearray()
        while block := os.read(descriptor, min(1024 * 1024, maximum + 1)):
            content.extend(block)
            if len(content) > maximum:
                raise ContractError("upgrade_output_file_size")
        after = os.fstat(descriptor)
        if (
            before.st_dev,
            before.st_ino,
            before.st_mode,
            before.st_size,
            before.st_mtime_ns,
            before.st_ctime_ns,
        ) != (
            after.st_dev,
            after.st_ino,
            after.st_mode,
            after.st_size,
            after.st_mtime_ns,
            after.st_ctime_ns,
        ) or len(content) != before.st_size:
            raise ContractError("upgrade_output_file_changed")
        return bytes(content)
    finally:
        os.close(descriptor)


def require_private_output_directory(
    path: pathlib.Path, owner_uid: int
) -> pathlib.Path:
    metadata = os.lstat(path)
    if (
        not stat.S_ISDIR(metadata.st_mode)
        or stat.S_IMODE(metadata.st_mode) != 0o700
        or metadata.st_uid != owner_uid
        or path.resolve(strict=True) != path
    ):
        raise ContractError("upgrade_output_directory")
    return path


def within(path: pathlib.Path, root: pathlib.Path) -> bool:
    try:
        path.relative_to(root)
    except ValueError:
        return False
    return True


def output_path(output: pathlib.Path, relative: object, owner_uid: int) -> pathlib.Path:
    text = require_string(relative, "upgrade_output_relative_path")
    pure = pathlib.PurePosixPath(text)
    if (
        pure.is_absolute()
        or text != pure.as_posix()
        or any(part in {"", ".", ".."} for part in pure.parts)
    ):
        raise ContractError("upgrade_output_relative_path")
    candidate = output.joinpath(*pure.parts)
    if candidate.resolve(strict=True) != candidate or not within(candidate, output):
        raise ContractError("upgrade_output_path_escape")
    current = candidate.parent
    while True:
        require_private_output_directory(current, owner_uid)
        if current == output:
            break
        current = current.parent
    return candidate


def current_migration_contract(
    repository: pathlib.Path, owner_uid: int
) -> tuple[dict[int, dict[str, str]], str]:
    root = repository / "internal/adapters/state/sqlite/migrations"
    expected: dict[int, dict[str, str]] = {}
    for version in range(1, 20):
        matches = sorted(root.glob(f"{version:03d}_*.sql"))
        if len(matches) != 1 or matches[0].is_symlink():
            raise ContractError("upgrade_migration_set")
        path = matches[0]
        digest = stable_source_sha256(path, owner_uid)
        expected[version] = {
            "name": path.name,
            "checksum": "sha256:" + digest,
            "sha256": digest,
        }
    for version, name in (
        (17, "017_intake.sql"),
        (18, "018_intake_dossiers.sql"),
        (19, "019_intake_dossier_confirmations.sql"),
    ):
        if expected[version]["name"] != name:
            raise ContractError("upgrade_migration_name")
    manifest = hashlib.sha256(
        json.dumps(expected, sort_keys=True, separators=(",", ":")).encode("utf-8")
    ).hexdigest()
    return expected, manifest


def require_migration_rows(
    value: object,
    expected: dict[int, dict[str, str]],
    final_version: int,
    code: str,
) -> list[dict[str, object]]:
    if not isinstance(value, list) or len(value) != final_version:
        raise ContractError(code)
    result: list[dict[str, object]] = []
    for index, item in enumerate(value, start=1):
        row = require_exact_keys(item, {"version", "name", "checksum"}, code)
        if (
            type(row["version"]) is not int
            or row["version"] != index
            or row["name"] != expected[index]["name"]
            or row["checksum"] != expected[index]["checksum"]
        ):
            raise ContractError(code)
        result.append(row)
    return result


def sqlite_schema_manifest(
    connection: sqlite3.Connection,
) -> list[dict[str, object]]:
    return [
        {
            "type": str(row[0]),
            "name": str(row[1]),
            "table": str(row[2]),
            "sql": None if row[3] is None else str(row[3]),
        }
        for row in connection.execute(
            "SELECT type,name,tbl_name,sql FROM sqlite_schema "
            "ORDER BY type,name,tbl_name,sql"
        )
    ]


def expected_v23_schema_manifest(
    repository: pathlib.Path,
) -> list[dict[str, object]]:
    connection = sqlite3.connect(":memory:")
    try:
        connection.execute("PRAGMA foreign_keys=OFF")
        for name in (
            "017_intake.sql",
            "018_intake_dossiers.sql",
            "019_intake_dossier_confirmations.sql",
        ):
            connection.executescript(
                (
                    repository / "internal/adapters/state/sqlite/migrations" / name
                ).read_text(encoding="utf-8")
            )
        return sqlite_schema_manifest(connection)
    finally:
        connection.close()


def validate_v23_schema(
    connection: sqlite3.Connection, repository: pathlib.Path
) -> None:
    expected = expected_v23_schema_manifest(repository)
    expected_names = {item["name"] for item in expected}
    observed = [
        item
        for item in sqlite_schema_manifest(connection)
        if item["name"] in expected_names or item["table"] in V23_TABLES
    ]
    if observed != expected or any(
        connection.execute(
            f"SELECT COUNT(*) FROM {quote_identifier(table)}"
        ).fetchone()[0]
        != 0
        for table in V23_TABLES
    ):
        raise ContractError("sqlite_v23_schema_changed")


def audit_encode_sqlite_value(value: object) -> dict[str, object]:
    if value is None:
        return {"type": "null", "value": None}
    if isinstance(value, bytes):
        return {"type": "blob", "value": value.hex()}
    if isinstance(value, int):
        return {"type": "integer", "value": value}
    if isinstance(value, float):
        return {"type": "real", "value": value.hex()}
    if isinstance(value, str):
        return {"type": "text", "value": value}
    raise ContractError("upgrade_sqlite_value")


def audit_functional_digest(
    connection: sqlite3.Connection, source_tables: list[str]
) -> str:
    available = {
        str(row[0])
        for row in connection.execute(
            "SELECT name FROM sqlite_schema "
            "WHERE type='table' AND name NOT LIKE 'sqlite_%'"
        )
    }
    digest = hashlib.sha256()
    for table in source_tables:
        if table not in available:
            raise ContractError("upgrade_functional_table_missing")
        columns = [
            str(row[1])
            for row in connection.execute(
                f"PRAGMA table_info({quote_identifier(table)})"
            )
        ]
        if not columns:
            raise ContractError("upgrade_functional_columns")
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
        for row in connection.execute(
            f"SELECT {selected} FROM {quote_identifier(table)} ORDER BY {ordering}"
        ):
            payload = json.dumps(
                [audit_encode_sqlite_value(item) for item in row],
                ensure_ascii=False,
                separators=(",", ":"),
                sort_keys=True,
                allow_nan=False,
            ).encode("utf-8")
            digest.update(len(payload).to_bytes(8, "big"))
            digest.update(payload)
    return digest.hexdigest()


def validate_materialized_audit_config(
    path: pathlib.Path,
    digest: str,
    output: pathlib.Path,
    forbidden: pathlib.Path,
    harness: dict[str, object],
    result_database: pathlib.Path,
    bubblewrap: str,
    live_profile: str,
) -> None:
    content = private_output_bytes(path, os.getuid(), maximum=4 * 1024 * 1024)
    if hashlib.sha256(content).hexdigest() != digest:
        raise ContractError("upgrade_materialized_config_hash")
    config = tomllib.loads(content.decode("utf-8"))
    profile = require_string(harness["profile"], "upgrade_harness_profile")
    runtime_root = output / "runtime" / profile
    exact = (
        (nested(config, "server", "listen"), harness["listen"]),
        (nested(config, "state", "sqlite", "path"), str(result_database)),
        (
            nested(config, "runtime", "codex", "account_home_root"),
            str(output / "accounts"),
        ),
        (nested(config, "runtime", "codex", "account_profile"), profile),
        (
            nested(config, "runtime", "codex", "cgroup_root"),
            harness["delegated_cgroup_root"],
        ),
        (
            nested(config, "repository", "local", "seed_path"),
            str(output / "projection/repository"),
        ),
        (nested(config, "test_attestor", "provider"), "bubblewrap"),
        (
            nested(config, "test_attestor", "bubblewrap", "command"),
            bubblewrap,
        ),
        (
            nested(config, "test_attestor", "resources", "cgroup_root"),
            harness["delegated_cgroup_root"],
        ),
    )
    if any(observed != expected for observed, expected in exact):
        raise ContractError("upgrade_materialized_config_contract")
    isolated = (
        ("state", "sqlite", "path"),
        ("artifact", "filesystem", "root"),
        ("credentials", "local", "path"),
        ("runtime", "codex", "work_root"),
        ("runtime", "codex", "cache_root"),
        ("workspace", "local", "root"),
        ("identity", "local_token_path"),
        ("config", "effective_path"),
    )
    credential_paths: list[pathlib.Path] = []
    for keys in isolated:
        raw = require_string(nested(config, *keys), "upgrade_materialized_config_path")
        current = pathlib.Path(raw)
        if (
            not current.is_absolute()
            or pathlib.Path(os.path.normpath(current)) != current
            or not within(current, runtime_root)
            or within(current, forbidden)
        ):
            raise ContractError("upgrade_materialized_config_isolation")
        if keys in (
            ("credentials", "local", "path"),
            ("identity", "local_token_path"),
        ):
            credential_paths.append(current)
    if any(os.path.lexists(item) for item in credential_paths):
        raise ContractError("upgrade_credential_projection_present")
    if live_profile.casefold() in json.dumps(config, sort_keys=True).casefold():
        raise ContractError("upgrade_live_profile_reference")


def validate_upgrade_evidence(
    value: object,
    output: pathlib.Path,
    owner_uid: int,
    revision: str,
    harness: dict[str, object],
) -> None:
    if not isinstance(value, list) or len(value) != len(EVIDENCE_LABELS):
        raise ContractError("upgrade_evidence_count")
    contents: dict[str, str] = {}
    seen_paths: set[pathlib.Path] = set()
    for index, (item, expected_label) in enumerate(
        zip(value, EVIDENCE_LABELS), start=1
    ):
        evidence = require_exact_keys(
            item,
            {"label", "argv", "returncode", "stdout", "stderr"},
            "upgrade_evidence_shape",
        )
        if evidence["label"] != expected_label:
            raise ContractError("upgrade_evidence_order")
        argv = evidence["argv"]
        if (
            not isinstance(argv, list)
            or not argv
            or any(type(argument) is not str or not argument for argument in argv)
        ):
            raise ContractError("upgrade_evidence_argv")
        expected_return = 3 if expected_label.endswith("profile-status") else 0
        if (
            type(evidence["returncode"]) is not int
            or evidence["returncode"] != expected_return
        ):
            raise ContractError("upgrade_evidence_returncode")
        for stream_name in ("stdout", "stderr"):
            stream = require_exact_keys(
                evidence[stream_name],
                {"path", "sha256", "bytes"},
                "upgrade_evidence_stream",
            )
            expected_relative = f"evidence/{index:02d}-{expected_label}.{stream_name}"
            if stream["path"] != expected_relative:
                raise ContractError("upgrade_evidence_path")
            path = output_path(output, stream["path"], owner_uid)
            if path in seen_paths:
                raise ContractError("upgrade_evidence_path_reused")
            seen_paths.add(path)
            content = private_output_bytes(path, owner_uid)
            if (
                require_integer(stream["bytes"], "upgrade_evidence_bytes")
                != len(content)
                or require_hex(stream["sha256"], "upgrade_evidence_hash")
                != hashlib.sha256(content).hexdigest()
            ):
                raise ContractError("upgrade_evidence_content")
            if stream_name == "stderr" and content:
                raise ContractError("upgrade_evidence_stderr")
            if stream_name == "stdout":
                try:
                    contents[expected_label] = content.decode("utf-8")
                except UnicodeDecodeError as error:
                    raise ContractError("upgrade_evidence_utf8") from error
    buildinfo = contents["binary-buildinfo"]
    if (
        f"\tbuild\tvcs.revision={revision}\n" not in buildinfo
        or "\tbuild\tvcs.modified=false\n" not in buildinfo
    ):
        raise ContractError("upgrade_evidence_buildinfo")
    if (
        contents["repository-revision"].strip()
        != harness["projected_repository_revision"]
    ):
        raise ContractError("upgrade_evidence_repository")
    user_manager_group = contents["user-manager-control-group"].strip()
    if (
        not user_manager_group.startswith("/")
        or user_manager_group in {"", "/"}
        or not str(harness["delegated_cgroup_root"]).endswith(
            user_manager_group + "/app.slice/" + str(harness["unit"])
        )
    ):
        raise ContractError("upgrade_evidence_user_manager_cgroup")
    for label in (
        "preflight-unit",
        "cycle-1-unit-collected",
        "cycle-2-unit-collected",
        "final-unit",
    ):
        if contents[label].strip() != "not-found":
            raise ContractError("upgrade_evidence_unit_state")
    invocation_ids: list[str] = []
    for cycle in (1, 2):
        observed: list[str] = []
        for action in ("start", "status"):
            text = contents[f"cycle-{cycle}-{action}"]
            expected_action = "start" if action == "start" else "check"
            match = re.search(r"(?:^| )invocation_id=([0-9a-f]{32})(?:\s|$)", text)
            if f"status=running action={expected_action}" not in text or match is None:
                raise ContractError("upgrade_evidence_invocation")
            if action == "start":
                for pid_name in ("main_pid", "daemon_pid"):
                    if (
                        re.search(rf"(?:^| ){pid_name}=([1-9][0-9]*)(?:\s|$)", text)
                        is None
                    ):
                        raise ContractError("upgrade_evidence_runtime_pid")
            observed.append(match.group(1))
        if observed[0] != observed[1]:
            raise ContractError("upgrade_evidence_invocation_changed")
        invocation_ids.append(observed[0])
        control_group = contents[f"cycle-{cycle}-control-group"].strip()
        if (
            not control_group.startswith("/")
            or "/../" in control_group
            or not str(harness["unit_control_group"]).endswith(
                control_group + "/orquesta-control"
            )
        ):
            raise ContractError("upgrade_evidence_control_group")
        if (
            f"status=stopped profile={harness['profile']}"
            not in contents[f"cycle-{cycle}-profile-status"]
        ):
            raise ContractError("upgrade_evidence_profile_status")
    if (
        invocation_ids[0] != harness["_first_invocation_id"]
        or invocation_ids[1] != harness["_restart_invocation_id"]
        or invocation_ids[0] == invocation_ids[1]
    ):
        raise ContractError("upgrade_evidence_cycle_identity")


def process_reference_counts(
    proc_root: pathlib.Path,
    binary: pathlib.Path,
    database: pathlib.Path,
) -> tuple[int, int]:
    binary_stat = binary.stat(follow_symlinks=False)
    database_stat = database.stat(follow_symlinks=False)
    binary_refs: set[int] = set()
    database_refs: set[int] = set()
    for entry in proc_root.iterdir():
        if not entry.name.isdigit() or not entry.is_dir():
            continue
        pid = int(entry.name)
        try:
            executable = os.stat(entry / "exe")
        except OSError:
            executable = None
        if executable is not None and (
            executable.st_dev,
            executable.st_ino,
        ) == (binary_stat.st_dev, binary_stat.st_ino):
            binary_refs.add(pid)
        descriptors = entry / "fd"
        try:
            descriptor_entries = list(descriptors.iterdir())
        except OSError:
            descriptor_entries = []
        for descriptor in descriptor_entries:
            try:
                opened = os.stat(descriptor)
            except OSError:
                continue
            if (opened.st_dev, opened.st_ino) == (
                database_stat.st_dev,
                database_stat.st_ino,
            ):
                database_refs.add(pid)
                break
    return len(binary_refs), len(database_refs)


def command_upgrade_receipt(args: argparse.Namespace) -> None:
    raw, _ = secure_read(
        args.path,
        maximum=65536,
        expected_uid=args.owner_uid,
        allowed_modes=(0o400, 0o600),
    )
    if hashlib.sha256(raw).hexdigest() != args.sha:
        raise ContractError("upgrade_receipt_hash")
    document = json.loads(raw, object_pairs_hook=strict_object)
    root = require_exact_keys(
        document,
        {
            "schema_version",
            "result",
            "candidate",
            "source",
            "first_start",
            "restart",
            "result_database",
            "closure",
            "harness",
            "evidence",
        },
        "upgrade_receipt_root",
    )
    if root["schema_version"] != UPGRADE_RECEIPT_SCHEMA or root["result"] != "pass":
        raise ContractError("upgrade_receipt_version")
    candidate = require_exact_keys(
        root["candidate"],
        {
            "binary_path",
            "binary_sha256",
            "repository_revision",
            "binary_vcs_modified",
            "profile_script_sha256",
            "systemd_adapter_sha256",
            "profile_helper_sha256",
            "config_template_sha256",
            "materialized_config_sha256",
            "migration_manifest_sha256",
            "go_sha256",
            "git_sha256",
            "bubblewrap_sha256",
            "systemctl_sha256",
            "systemd_run_sha256",
            "harness_wrapper_sha256",
            "harness_helper_sha256",
        },
        "upgrade_candidate_shape",
    )
    source = require_exact_keys(
        root["source"],
        {
            "database_sha256",
            "user_version",
            "quick_check",
            "foreign_key_check_rows",
            "functional_data_sha256",
            "table_counts",
            "core_counts",
            "schema_migrations",
        },
        "upgrade_source_shape",
    )
    first = require_exact_keys(
        root["first_start"],
        {
            "source_user_version",
            "result_user_version",
            "schema_migrations_added",
            "system_status",
            "invocation_id",
            "quick_check",
            "foreign_key_check_rows",
            "functional_data_sha256",
            "table_counts",
            "core_counts",
            "audit_counts",
        },
        "upgrade_first_shape",
    )
    restart = require_exact_keys(
        root["restart"],
        {
            "source_user_version",
            "result_user_version",
            "schema_migrations_duplicated",
            "system_status",
            "invocation_id",
            "quick_check",
            "foreign_key_check_rows",
            "functional_data_sha256",
            "table_counts",
            "core_counts",
            "audit_counts",
        },
        "upgrade_restart_shape",
    )
    result_db = require_exact_keys(
        root["result_database"],
        {
            "path",
            "sha256",
            "quick_check",
            "foreign_key_check_rows",
            "user_version",
            "functional_data_sha256",
            "table_counts",
            "core_counts",
            "schema_migrations",
        },
        "upgrade_result_database_shape",
    )
    closure = require_exact_keys(
        root["closure"],
        {
            "candidate_processes",
            "database_open_processes",
            "live_profile_touched",
            "unit_load_state",
            "profile_status",
            "identity_files",
            "sqlite_ancillary_files",
            "credential_projections",
        },
        "upgrade_closure_shape",
    )
    harness = require_exact_keys(
        root["harness"],
        {
            "created_at",
            "output_dir",
            "forbidden_live_root",
            "profile",
            "unit",
            "listen",
            "delegated_cgroup_root",
            "unit_control_group",
            "projected_repository_revision",
            "directory_mode",
            "data_file_mode",
            "executable_projection_mode",
            "checks",
        },
        "upgrade_harness_shape",
    )
    checks = require_exact_keys(
        harness["checks"],
        {
            "input_hashes_stable",
            "source_backup_unchanged",
            "user_version_16_19_19",
            "migrations_17_18_19_once",
            "quick_check_and_foreign_keys",
            "functional_digest_and_counts_equal",
            "schema_manifest_exact",
            "one_status_audit_per_cycle",
            "invocation_ids_distinct",
            "config_paths_isolated",
            "credentials_removed",
            "zero_residual_resources",
        },
        "upgrade_harness_checks_shape",
    )
    if any(type(value) is not bool or value is not True for value in checks.values()):
        raise ContractError("upgrade_harness_checks")

    repository = pathlib.Path(args.repository)
    forbidden = pathlib.Path(
        require_string(harness["forbidden_live_root"], "upgrade_forbidden_root")
    )
    runtime_base = pathlib.Path(args.runtime_base)
    output = pathlib.Path(require_string(harness["output_dir"], "upgrade_output_dir"))
    if (
        not output.is_absolute()
        or not forbidden.is_absolute()
        or forbidden != runtime_base
        or forbidden.resolve(strict=True) != forbidden
        or runtime_base.resolve(strict=True) != runtime_base
        or within(output, forbidden)
    ):
        raise ContractError("upgrade_output_isolation")
    require_private_output_directory(output, args.owner_uid)
    receipt_path = pathlib.Path(args.path)
    if receipt_path.parent != output or receipt_path.name != "receipt.json":
        raise ContractError("upgrade_receipt_location")
    profile = require_string(harness["profile"], "upgrade_harness_profile")
    unit = require_string(harness["unit"], "upgrade_harness_unit")
    if (
        not REACCREDIT_PROFILE.fullmatch(profile)
        or unit != f"orquesta-v23-{profile}.service"
        or not re.fullmatch(
            r"127\.0\.0\.1:([1-9][0-9]{0,4})",
            require_string(harness["listen"], "upgrade_harness_listen"),
        )
        or int(str(harness["listen"]).rsplit(":", 1)[1]) > 65535
        or not REVISION.fullmatch(
            require_string(
                harness["projected_repository_revision"],
                "upgrade_harness_revision",
            )
        )
        or not re.fullmatch(
            r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z",
            require_string(harness["created_at"], "upgrade_harness_time"),
        )
        or harness["directory_mode"] != "0700"
        or harness["data_file_mode"] != "0600"
        or harness["executable_projection_mode"] != "0500"
    ):
        raise ContractError("upgrade_harness_contract")
    control_group = pathlib.Path(
        require_string(harness["unit_control_group"], "upgrade_harness_control_group")
    )
    delegated_group = pathlib.Path(
        require_string(
            harness["delegated_cgroup_root"],
            "upgrade_harness_delegated_cgroup",
        )
    )
    if (
        not control_group.is_absolute()
        or pathlib.Path(os.path.normpath(control_group)) != control_group
        or not delegated_group.is_absolute()
        or pathlib.Path(os.path.normpath(delegated_group)) != delegated_group
        or delegated_group.parts[-2:] != ("app.slice", unit)
        or control_group != delegated_group / "orquesta-control"
    ):
        raise ContractError("upgrade_harness_control_group")

    expected_migrations, migration_manifest = current_migration_contract(
        repository, args.owner_uid
    )
    helper_hashes = require_exact_keys(
        candidate["profile_helper_sha256"],
        set(PROFILE_HELPERS),
        "upgrade_profile_helpers_shape",
    )
    for helper in PROFILE_HELPERS:
        expected = stable_source_sha256(
            repository / "scripts/lib" / helper, args.owner_uid
        )
        if helper_hashes[helper] != expected:
            raise ContractError("upgrade_profile_helper_hash")
    subject_checks = (
        candidate["binary_path"] == args.binary,
        candidate["binary_sha256"] == args.binary_sha,
        candidate["repository_revision"] == args.revision,
        candidate["binary_vcs_modified"] is False,
        candidate["profile_script_sha256"] == args.profile_sha,
        candidate["systemd_adapter_sha256"] == args.adapter_sha,
        candidate["config_template_sha256"] == args.config_template_sha,
        candidate["migration_manifest_sha256"] == migration_manifest,
        candidate["go_sha256"] == args.go_sha,
        candidate["git_sha256"] == args.git_sha,
        candidate["bubblewrap_sha256"] == args.bubblewrap_sha,
        candidate["systemctl_sha256"] == args.systemctl_sha,
        candidate["systemd_run_sha256"] == args.systemd_run_sha,
        candidate["harness_wrapper_sha256"]
        == stable_source_sha256(
            repository / "scripts/orquesta_sqlite_v23_reaccredit.sh",
            args.owner_uid,
        ),
        candidate["harness_helper_sha256"]
        == stable_source_sha256(
            repository / "scripts/lib/orquesta_sqlite_v23_reaccredit.py",
            args.owner_uid,
        ),
    )
    if not all(subject_checks):
        raise ContractError("upgrade_candidate_contract")
    for key in (
        "binary_sha256",
        "profile_script_sha256",
        "systemd_adapter_sha256",
        "config_template_sha256",
        "materialized_config_sha256",
        "migration_manifest_sha256",
        "go_sha256",
        "git_sha256",
        "bubblewrap_sha256",
        "systemctl_sha256",
        "systemd_run_sha256",
        "harness_wrapper_sha256",
        "harness_helper_sha256",
    ):
        require_hex(candidate[key], "upgrade_candidate_hash")
    if not REVISION.fullmatch(
        require_string(candidate["repository_revision"], "upgrade_revision")
    ):
        raise ContractError("upgrade_revision")

    projected = output / "projection"
    projected_hashes = (
        (projected / "subject/orquesta", args.binary_sha),
        (
            projected / "repository/scripts/orquesta_profile_server.sh",
            args.profile_sha,
        ),
        (
            projected / "repository/scripts/orquesta_profile_systemd_user.sh",
            args.adapter_sha,
        ),
    )
    for path, expected_sha in projected_hashes:
        if (
            output_path(output, str(path.relative_to(output)), args.owner_uid) != path
            or stat.S_IMODE(path.stat(follow_symlinks=False).st_mode) != 0o500
            or hashlib.sha256(
                private_output_bytes(
                    path,
                    args.owner_uid,
                    64 * 1024 * 1024,
                    allowed_modes=(0o500,),
                )
            ).hexdigest()
            != expected_sha
        ):
            raise ContractError("upgrade_projected_subject")
    for helper in PROFILE_HELPERS:
        path = projected / "repository/scripts/lib" / helper
        if (
            output_path(output, str(path.relative_to(output)), args.owner_uid) != path
            or stat.S_IMODE(path.stat(follow_symlinks=False).st_mode) != 0o500
            or hashlib.sha256(
                private_output_bytes(path, args.owner_uid, allowed_modes=(0o500,))
            ).hexdigest()
            != helper_hashes[helper]
        ):
            raise ContractError("upgrade_projected_helper")
    for version, migration in expected_migrations.items():
        path = (
            projected
            / "repository/internal/adapters/state/sqlite/migrations"
            / migration["name"]
        )
        if (
            output_path(output, str(path.relative_to(output)), args.owner_uid) != path
            or hashlib.sha256(private_output_bytes(path, args.owner_uid)).hexdigest()
            != migration["sha256"]
        ):
            raise ContractError("upgrade_projected_migration")

    require_hex(source["database_sha256"], "upgrade_source_database_hash")
    source_counts = require_count_map(
        source["table_counts"], "upgrade_source_table_counts"
    )
    source_core = require_count_map(
        source["core_counts"], "upgrade_source_core_counts", set(CORE_TABLES)
    )
    if source_core != {table: source_counts.get(table) for table in CORE_TABLES}:
        raise ContractError("upgrade_source_core_counts")
    require_migration_rows(
        source["schema_migrations"],
        expected_migrations,
        16,
        "upgrade_source_migrations",
    )
    first_counts = require_count_map(
        first["table_counts"], "upgrade_first_table_counts"
    )
    restart_counts = require_count_map(
        restart["table_counts"], "upgrade_restart_table_counts"
    )
    result_counts = require_count_map(
        result_db["table_counts"], "upgrade_result_table_counts"
    )
    first_core = require_count_map(
        first["core_counts"], "upgrade_first_core_counts", set(CORE_TABLES)
    )
    restart_core = require_count_map(
        restart["core_counts"], "upgrade_restart_core_counts", set(CORE_TABLES)
    )
    result_core = require_count_map(
        result_db["core_counts"], "upgrade_result_core_counts", set(CORE_TABLES)
    )
    first_audit = require_count_map(
        first["audit_counts"],
        "upgrade_first_audit_counts",
        FUNCTIONAL_AUDIT_TABLES,
    )
    restart_audit = require_count_map(
        restart["audit_counts"],
        "upgrade_restart_audit_counts",
        FUNCTIONAL_AUDIT_TABLES,
    )
    if (
        require_integer(source["user_version"], "upgrade_source_version") != 16
        or require_string(source["quick_check"], "upgrade_source_quick") != "ok"
        or require_integer(
            source["foreign_key_check_rows"], "upgrade_source_foreign_keys"
        )
        != 0
        or require_integer(first["source_user_version"], "upgrade_first_source_version")
        != 16
        or require_integer(first["result_user_version"], "upgrade_first_result_version")
        != 19
        or first["schema_migrations_added"] != [17, 18, 19]
        or require_string(first["system_status"], "upgrade_first_status") != "passed"
        or require_integer(
            restart["source_user_version"], "upgrade_restart_source_version"
        )
        != 19
        or require_integer(
            restart["result_user_version"], "upgrade_restart_result_version"
        )
        != 19
        or restart["schema_migrations_duplicated"] is not False
        or require_string(restart["system_status"], "upgrade_restart_status")
        != "passed"
        or require_string(result_db["quick_check"], "upgrade_result_quick") != "ok"
        or require_integer(
            result_db["foreign_key_check_rows"], "upgrade_result_foreign_keys"
        )
        != 0
        or require_integer(result_db["user_version"], "upgrade_result_version") != 19
        or require_string(first["quick_check"], "upgrade_first_quick") != "ok"
        or require_integer(
            first["foreign_key_check_rows"], "upgrade_first_foreign_keys"
        )
        != 0
        or require_string(restart["quick_check"], "upgrade_restart_quick") != "ok"
        or require_integer(
            restart["foreign_key_check_rows"], "upgrade_restart_foreign_keys"
        )
        != 0
    ):
        raise ContractError("upgrade_sqlite_versions")
    first_invocation = require_string(
        first["invocation_id"], "upgrade_first_invocation"
    )
    restart_invocation = require_string(
        restart["invocation_id"], "upgrade_restart_invocation"
    )
    if (
        not INVOCATION_ID.fullmatch(first_invocation)
        or not INVOCATION_ID.fullmatch(restart_invocation)
        or first_invocation == restart_invocation
    ):
        raise ContractError("upgrade_invocations")
    functional_digest = require_hex(
        source["functional_data_sha256"], "upgrade_functional_digest"
    )
    if any(
        require_hex(item["functional_data_sha256"], "upgrade_functional_digest")
        != functional_digest
        for item in (first, restart, result_db)
    ):
        raise ContractError("upgrade_functional_digest")
    if (
        source_core != first_core
        or source_core != restart_core
        or source_core != result_core
        or result_counts != restart_counts
        or set(first_counts) != set(restart_counts)
        or source_counts.get("schema_migrations") != 16
        or first_counts.get("schema_migrations") != 19
        or restart_counts.get("schema_migrations") != 19
        or any(first_counts.get(table) != 0 for table in V23_TABLES)
        or any(restart_counts.get(table) != 0 for table in V23_TABLES)
    ):
        raise ContractError("upgrade_count_contract")
    for table, count in source_counts.items():
        if table in FUNCTIONAL_AUDIT_TABLES or table == "schema_migrations":
            continue
        if first_counts.get(table) != count or restart_counts.get(table) != count:
            raise ContractError("upgrade_functional_count_changed")
    for table in set(first_counts) - set(source_counts):
        if (
            table not in FUNCTIONAL_AUDIT_TABLES
            and first_counts[table] != restart_counts[table]
        ):
            raise ContractError("upgrade_new_table_count_changed")
    for table in FUNCTIONAL_AUDIT_TABLES:
        source_count = source_counts.get(table)
        if (
            source_count is None
            or first_audit[table] != source_count + 1
            or restart_audit[table] != source_count + 2
            or first_counts.get(table) != first_audit[table]
            or restart_counts.get(table) != restart_audit[table]
        ):
            raise ContractError("upgrade_audit_delta")
    require_migration_rows(
        result_db["schema_migrations"],
        expected_migrations,
        19,
        "upgrade_result_migrations",
    )

    result_path = output_path(output, result_db["path"], args.owner_uid)
    expected_result_path = output / "runtime" / profile / "state/orquesta.sqlite"
    if result_path != expected_result_path:
        raise ContractError("upgrade_result_database_path")
    result_sha, _ = secure_file_sha256(str(result_path), args.owner_uid)
    if require_hex(result_db["sha256"], "upgrade_result_database_hash") != result_sha:
        raise ContractError("upgrade_result_database_hash")
    connection = connect_read_only(str(result_path))
    try:
        quick = [str(row[0]) for row in connection.execute("PRAGMA quick_check")]
        foreign = list(connection.execute("PRAGMA foreign_key_check"))
        version = connection.execute("PRAGMA user_version").fetchone()[0]
        tables = [
            str(row[0])
            for row in connection.execute(
                "SELECT name FROM sqlite_schema "
                "WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
            )
        ]
        actual_counts = {
            table: connection.execute(
                f"SELECT COUNT(*) FROM {quote_identifier(table)}"
            ).fetchone()[0]
            for table in tables
        }
        actual_migrations = [
            {"version": row[0], "name": row[1], "checksum": row[2]}
            for row in connection.execute(
                "SELECT version,name,checksum FROM schema_migrations "
                "ORDER BY version,name,checksum"
            )
        ]
        source_functional_tables = sorted(
            set(source_counts) - FUNCTIONAL_AUDIT_TABLES - {"schema_migrations"}
        )
        actual_functional = audit_functional_digest(
            connection, source_functional_tables
        )
        validate_v23_schema(connection, repository)
    finally:
        connection.close()
    if (
        quick != ["ok"]
        or foreign
        or version != 19
        or actual_counts != result_counts
        or actual_migrations != result_db["schema_migrations"]
        or {table: actual_counts.get(table) for table in CORE_TABLES} != result_core
        or actual_functional != functional_digest
    ):
        raise ContractError("upgrade_result_database_contract")

    config_path = output_path(output, "config.toml", args.owner_uid)
    validate_materialized_audit_config(
        config_path,
        require_hex(
            candidate["materialized_config_sha256"],
            "upgrade_materialized_config_hash",
        ),
        output,
        forbidden,
        harness,
        result_path,
        args.bubblewrap,
        args.profile,
    )
    harness["_first_invocation_id"] = first_invocation
    harness["_restart_invocation_id"] = restart_invocation
    validate_upgrade_evidence(
        root["evidence"], output, args.owner_uid, args.revision, harness
    )
    del harness["_first_invocation_id"]
    del harness["_restart_invocation_id"]

    closure_expected = {
        "candidate_processes": 0,
        "database_open_processes": 0,
        "live_profile_touched": False,
        "unit_load_state": "not-found",
        "profile_status": "stopped",
        "identity_files": 0,
        "sqlite_ancillary_files": 0,
        "credential_projections": 0,
    }
    if (
        closure != closure_expected
        or any(
            type(closure[key]) is not int
            for key in (
                "candidate_processes",
                "database_open_processes",
                "identity_files",
                "sqlite_ancillary_files",
                "credential_projections",
            )
        )
        or closure["live_profile_touched"] is not False
        or type(closure["unit_load_state"]) is not str
        or type(closure["profile_status"]) is not str
    ):
        raise ContractError("upgrade_closure_contract")
    projected_binary = projected / "subject/orquesta"
    binary_processes, database_processes = process_reference_counts(
        pathlib.Path(args.proc_root), projected_binary, result_path
    )
    if binary_processes or database_processes:
        raise ContractError("upgrade_closure_processes")
    for name in (
        "server.pid",
        "server.start_ref",
        "server.binary_id",
        "server.binary_sha256",
        "server.config_sha256",
    ):
        if os.path.lexists(output / "runtime" / profile / "run" / name):
            raise ContractError("upgrade_identity_residue")
    for suffix in ("-wal", "-shm", "-journal"):
        if os.path.lexists(str(result_path) + suffix):
            raise ContractError("upgrade_sqlite_residue")


def command_sqlite_before(args: argparse.Namespace) -> None:
    connection = connect_read_only(args.path)
    try:
        quick, version, _, _, _ = sqlite_facts(connection)
        if quick != ["ok"] or version != 16:
            raise ContractError("sqlite_before")
    finally:
        connection.close()


def expected_v23_migrations(repository: str) -> list[tuple[int, str, str]]:
    result = []
    root = (
        pathlib.Path(repository)
        / "internal"
        / "adapters"
        / "state"
        / "sqlite"
        / "migrations"
    )
    for version, name in (
        (17, "017_intake.sql"),
        (18, "018_intake_dossiers.sql"),
        (19, "019_intake_dossier_confirmations.sql"),
    ):
        digest = hashlib.sha256((root / name).read_bytes()).hexdigest()
        result.append((version, name, "sha256:" + digest))
    return result


def command_sqlite_after(args: argparse.Namespace) -> None:
    connection = connect_read_only(args.path)
    backup = connect_read_only(args.backup)
    try:
        quick, version, _, _, _ = sqlite_facts(connection)
        observed = list(
            connection.execute(
                "SELECT version,name,checksum FROM schema_migrations "
                "WHERE version BETWEEN 17 AND 19 ORDER BY version"
            )
        )
        counts = [
            connection.execute(
                "SELECT COUNT(*) FROM schema_migrations WHERE version=?",
                (migration,),
            ).fetchone()[0]
            for migration in (17, 18, 19)
        ]
        if (
            quick != ["ok"]
            or version != 19
            or list(connection.execute("PRAGMA foreign_key_check"))
            or observed != expected_v23_migrations(args.repository)
            or counts != [1, 1, 1]
        ):
            raise ContractError("sqlite_after")
        validate_v23_schema(connection, pathlib.Path(args.repository))
        if not args.skip_functional:
            projection = functional_projection(backup)
            if functional_data_sha256(backup, projection) != functional_data_sha256(
                connection, projection
            ):
                raise ContractError("sqlite_functional_data_changed")
    finally:
        backup.close()
        connection.close()


def command_functional_digest(args: argparse.Namespace) -> None:
    connection = connect_read_only(args.path)
    projection_source = connect_read_only(args.projection_from)
    try:
        projection = functional_projection(projection_source)
        source_digest = functional_data_sha256(projection_source, projection)
        observed_digest = functional_data_sha256(connection, projection)
        if source_digest != observed_digest:
            raise ContractError("sqlite_functional_data_changed")
        print(observed_digest)
    finally:
        projection_source.close()
        connection.close()


def command_effective(args: argparse.Namespace) -> None:
    document = read_json(args.path, 16 * 1024 * 1024, expected_uid=os.getuid())
    entries = document.get("entries")
    if document.get("document_type") != "orquesta.effective_config" or not isinstance(
        entries, list
    ):
        raise ContractError("effective_root")
    values: dict[str, object] = {}
    for entry in entries:
        if not isinstance(entry, dict) or not isinstance(entry.get("key"), str):
            raise ContractError("effective_entry")
        key = entry["key"]
        if key in values:
            raise ContractError("effective_duplicate")
        values[key] = entry.get("value")
    expected = {
        "state.sqlite.path": args.sqlite,
        "runtime.codex.cgroup_root": args.cgroup,
        "test_attestor.resources.cgroup_root": args.cgroup,
        "test_attestor.provider": args.provider,
        "test_attestor.max_concurrent_runs": args.concurrency,
    }
    if any(values.get(key) != value for key, value in expected.items()):
        raise ContractError("effective_contract")
    if args.provider == "microvm" and (
        values.get("test_attestor.microvm.launcher_socket") != args.launcher_socket
        or values.get("test_attestor.microvm.expected_asset_digest")
        != args.asset_digest
    ):
        raise ContractError("effective_microvm")


def backup_content(
    contract: str,
    source_sha: str,
    backup_sha: str,
    backup_size: int,
    version: int,
    migrations_sha: str,
) -> bytes:
    return (
        "\n".join(
            (
                f"schema={BACKUP_SCHEMA}",
                f"contract_sha256={contract}",
                f"source_sha256={source_sha}",
                f"backup_sha256={backup_sha}",
                f"backup_size={backup_size}",
                f"source_user_version={version}",
                f"schema_migrations_sha256={migrations_sha}",
            )
        )
        + "\n"
    ).encode("ascii")


def command_backup(args: argparse.Namespace) -> None:
    uid = os.getuid()
    source_before = require_regular_private(args.source, uid)
    target_exists = os.path.lexists(args.target)
    receipt_exists = os.path.lexists(args.receipt)
    if receipt_exists and not target_exists:
        raise ContractError("backup_pair_incomplete")
    fence = sqlite3.connect(args.source, timeout=0, isolation_level=None)
    fence.execute("PRAGMA busy_timeout=1000")
    try:
        fence.execute("BEGIN IMMEDIATE")
    except sqlite3.Error:
        fence.close()
        raise
    source_db = connect_read_only(args.source)
    temporary = ""
    descriptor = -1
    try:
        source_facts = sqlite_facts(source_db)
        if source_facts[0] != ["ok"]:
            raise ContractError("source_quick_check")
        if target_exists:
            target_metadata = require_regular_private(args.target, uid)
            target_db = connect_read_only(args.target)
            try:
                if sqlite_facts(target_db) != source_facts:
                    raise ContractError("existing_backup_drift")
            finally:
                target_db.close()
        else:
            descriptor, temporary = tempfile.mkstemp(
                prefix=".promotion-backup-", dir=args.directory
            )
            os.fchmod(descriptor, 0o600)
            os.close(descriptor)
            descriptor = -1
            destination = sqlite3.connect(temporary)
            try:
                source_db.backup(destination)
                destination.commit()
            finally:
                destination.close()
            target_db = connect_read_only(temporary)
            try:
                if sqlite_facts(target_db) != source_facts:
                    raise ContractError("new_backup_drift")
            finally:
                target_db.close()
            require_regular_private(temporary, uid)
            with open(temporary, "rb", buffering=0) as handle:
                os.fsync(handle.fileno())
            rename_noreplace(temporary, args.target)
            temporary = ""
            fsync_directory(args.directory)
            target_metadata = require_regular_private(args.target, uid)
        source_after = os.lstat(args.source)
        before_identity = (
            source_before.st_dev,
            source_before.st_ino,
            source_before.st_size,
            source_before.st_mtime_ns,
            source_before.st_ctime_ns,
        )
        after_identity = (
            source_after.st_dev,
            source_after.st_ino,
            source_after.st_size,
            source_after.st_mtime_ns,
            source_after.st_ctime_ns,
        )
        if before_identity != after_identity:
            raise ContractError("source_changed")
        source_sha, source_hashed = secure_file_sha256(args.source, uid)
        target_sha, target_hashed = secure_file_sha256(args.target, uid)
        source_final = os.lstat(args.source)
        source_hashed_identity = (
            source_hashed.st_dev,
            source_hashed.st_ino,
            source_hashed.st_mode,
            source_hashed.st_nlink,
            source_hashed.st_size,
            source_hashed.st_mtime_ns,
            source_hashed.st_ctime_ns,
        )
        source_final_identity = (
            source_final.st_dev,
            source_final.st_ino,
            source_final.st_mode,
            source_final.st_nlink,
            source_final.st_size,
            source_final.st_mtime_ns,
            source_final.st_ctime_ns,
        )
        if (
            source_hashed_identity != source_final_identity
            or target_hashed.st_size != target_metadata.st_size
        ):
            raise ContractError("hash_identity_changed")
        content = backup_content(
            args.contract,
            source_sha,
            target_sha,
            target_metadata.st_size,
            source_facts[1],
            source_facts[3],
        )
        if receipt_exists:
            observed, _ = secure_read(
                args.receipt,
                maximum=4096,
                expected_uid=uid,
                allowed_modes=(0o600,),
            )
            if observed != content:
                raise ContractError("existing_backup_receipt_drift")
        else:
            publish_private_record(args.receipt, content, args.directory)
        print(target_sha)
    finally:
        source_db.close()
        fence.execute("ROLLBACK")
        fence.close()
        if descriptor >= 0:
            os.close(descriptor)
        if temporary and os.path.exists(temporary):
            os.unlink(temporary)


def command_verify_backup(args: argparse.Namespace) -> None:
    uid = os.getuid()
    target_metadata = require_regular_private(args.target, uid)
    raw, _ = secure_read(
        args.receipt,
        maximum=4096,
        expected_uid=uid,
        allowed_modes=(0o600,),
    )
    try:
        text = raw.decode("ascii")
    except UnicodeDecodeError as error:
        raise ContractError("backup_receipt_ascii") from error
    if not text.endswith("\n"):
        raise ContractError("backup_receipt_framing")
    lines = text.splitlines()
    if len(lines) != 7:
        raise ContractError("backup_receipt_framing")
    values = {}
    for line in lines:
        if "=" not in line:
            raise ContractError("backup_receipt_line")
        key, value = line.split("=", 1)
        if key in values:
            raise ContractError("backup_receipt_duplicate")
        values[key] = value
    expected_keys = {
        "schema",
        "contract_sha256",
        "source_sha256",
        "backup_sha256",
        "backup_size",
        "source_user_version",
        "schema_migrations_sha256",
    }
    target_sha, hashed_metadata = secure_file_sha256(args.target, uid)
    if (
        set(values) != expected_keys
        or values.get("schema") != BACKUP_SCHEMA
        or values.get("contract_sha256") != args.contract
        or not HEX.fullmatch(values.get("source_sha256", ""))
        or not HEX.fullmatch(values.get("backup_sha256", ""))
        or not HEX.fullmatch(values.get("schema_migrations_sha256", ""))
        or not re.fullmatch(r"[1-9][0-9]*", values.get("backup_size", ""))
        or not re.fullmatch(r"(0|[1-9][0-9]*)", values.get("source_user_version", ""))
        or int(values["backup_size"]) != target_metadata.st_size
        or hashed_metadata.st_size != target_metadata.st_size
        or target_sha != values["backup_sha256"]
    ):
        raise ContractError("backup_receipt_contract")
    print(values["backup_sha256"])


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(add_help=False)
    commands = root.add_subparsers(dest="command", required=True)

    configs = commands.add_parser("configs", add_help=False)
    for name in (
        "primary",
        "rollback",
        "sqlite",
        "cgroup",
        "launcher-socket",
        "asset-digest",
        "bubblewrap",
        "repository",
        "target-ref",
        "profile",
    ):
        configs.add_argument("--" + name, required=True)
    configs.add_argument("--primary-concurrency", required=True, type=int)
    configs.add_argument("--rollback-concurrency", required=True, type=int)
    configs.set_defaults(handler=command_configs)

    upgrade = commands.add_parser("upgrade-receipt", add_help=False)
    for name in (
        "path",
        "sha",
        "binary",
        "binary-sha",
        "revision",
        "repository",
        "runtime-base",
        "profile",
        "proc-root",
        "profile-sha",
        "adapter-sha",
        "config-template-sha",
        "go-sha",
        "git-sha",
        "bubblewrap",
        "bubblewrap-sha",
        "systemctl-sha",
        "systemd-run-sha",
    ):
        upgrade.add_argument("--" + name, required=True)
    upgrade.add_argument("--owner-uid", required=True, type=int)
    upgrade.set_defaults(handler=command_upgrade_receipt)

    before = commands.add_parser("sqlite-before", add_help=False)
    before.add_argument("--path", required=True)
    before.set_defaults(handler=command_sqlite_before)

    after = commands.add_parser("sqlite-after", add_help=False)
    after.add_argument("--path", required=True)
    after.add_argument("--backup", required=True)
    after.add_argument("--repository", required=True)
    after.add_argument("--skip-functional", action="store_true")
    after.set_defaults(handler=command_sqlite_after)

    functional = commands.add_parser("functional-digest", add_help=False)
    functional.add_argument("--path", required=True)
    functional.add_argument("--projection-from", required=True)
    functional.set_defaults(handler=command_functional_digest)

    effective = commands.add_parser("effective", add_help=False)
    for name in (
        "path",
        "provider",
        "sqlite",
        "cgroup",
        "launcher-socket",
        "asset-digest",
    ):
        effective.add_argument("--" + name, required=True)
    effective.add_argument("--concurrency", required=True, type=int)
    effective.set_defaults(handler=command_effective)

    backup = commands.add_parser("backup", add_help=False)
    for name in ("source", "target", "directory", "receipt", "contract"):
        backup.add_argument("--" + name, required=True)
    backup.set_defaults(handler=command_backup)

    verify = commands.add_parser("verify-backup", add_help=False)
    for name in ("target", "receipt", "contract"):
        verify.add_argument("--" + name, required=True)
    verify.set_defaults(handler=command_verify_backup)

    publish = commands.add_parser("publish-record", add_help=False)
    for name in ("path", "directory", "content"):
        publish.add_argument("--" + name, required=True)
    publish.set_defaults(handler=command_publish_record)

    remove = commands.add_parser("remove-record", add_help=False)
    for name in ("path", "directory", "content"):
        remove.add_argument("--" + name, required=True)
    remove.set_defaults(handler=command_remove_record)

    final = commands.add_parser("final-receipt", add_help=False)
    for name in (
        "path",
        "contract",
        "binary-sha",
        "primary-config-sha",
        "rollback-config-sha",
        "backup-sha",
        "root-gate-ref",
        "old-unit-outcome",
    ):
        final.add_argument("--" + name, required=True)
    final.add_argument("--functional-data-sha", required=True)
    final.add_argument("--owner-uid", required=True, type=int)
    final.set_defaults(handler=command_final_receipt)
    return root


def main() -> None:
    arguments = parser().parse_args()
    try:
        arguments.handler(arguments)
    except (
        ContractError,
        json.JSONDecodeError,
        OSError,
        sqlite3.Error,
        tomllib.TOMLDecodeError,
        UnicodeError,
        ValueError,
    ) as error:
        code = error.args[0] if isinstance(error, ContractError) else "invalid"
        print(f"promotion_helper: status=error reason_code={code}", file=sys.stderr)
        raise SystemExit(1) from None


if __name__ == "__main__":
    main()
