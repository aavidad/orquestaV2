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
BACKUP_SCHEMA = "orquesta_profile_systemd_promotion_backup.v1"
RECEIPT_SCHEMA = "orquesta_profile_systemd_promotion_receipt.v1"
FUNCTIONAL_AUDIT_TABLES = {
    "authorization_receipts",
    "command_invocations",
    "command_outcomes",
}
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
    raw, _ = secure_read(
        path, maximum=4 * 1024 * 1024, expected_uid=os.getuid()
    )
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


def secure_file_sha256(
    path: str, uid: int
) -> tuple[str, os.stat_result]:
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
        json.dumps(
            migrations, separators=(",", ":"), ensure_ascii=True
        ).encode("ascii")
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
            f"typeof({quote_identifier(column)}),"
            f"quote({quote_identifier(column)})"
            for column in columns
        )
        query = (
            f"SELECT {selected} FROM {quote_identifier(table)} "
            f"ORDER BY {ordering}"
        )
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
        or values.get("firecracker_root_evidence_scope")
        != "operator_reference_only"
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
    candidate = document.get("candidate", {})
    first = document.get("first_start", {})
    restart = document.get("restart", {})
    closure = document.get("closure", {})
    result_db = document.get("result_database", {})
    source = document.get("source", {})
    if not all(
        isinstance(item, dict)
        for item in (candidate, first, restart, closure, result_db, source)
    ):
        raise ContractError("upgrade_receipt_sections")
    checks = (
        document.get("schema_version") == "orquesta_sqlite_upgrade_audit.v0",
        document.get("result") == "pass",
        candidate.get("binary_path") == args.binary,
        candidate.get("binary_sha256") == args.binary_sha,
        candidate.get("repository_revision") == args.revision,
        candidate.get("binary_vcs_modified") is False,
        candidate.get("profile_script_sha256") == args.profile_sha,
        candidate.get("systemd_adapter_sha256") == args.adapter_sha,
        candidate.get("systemctl_sha256") == args.systemctl_sha,
        candidate.get("systemd_run_sha256") == args.systemd_run_sha,
        source.get("user_version") == 16,
        first.get("source_user_version") == 16,
        first.get("result_user_version") == 19,
        first.get("schema_migrations_added") == [17, 18, 19],
        first.get("system_status") == "passed",
        restart.get("result_user_version") == 19,
        restart.get("schema_migrations_duplicated") is False,
        restart.get("system_status") == "passed",
        closure.get("candidate_processes") == 0,
        closure.get("database_open_processes") == 0,
        closure.get("live_profile_touched") is False,
        closure.get("unit_load_state") == "not-found",
        result_db.get("quick_check") == "ok",
        result_db.get("foreign_key_check_rows") == 0,
        result_db.get("user_version") == 19,
    )
    if not all(checks):
        raise ContractError("upgrade_receipt_contract")


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
        if not args.skip_functional:
            projection = functional_projection(backup)
            if functional_data_sha256(
                backup, projection
            ) != functional_data_sha256(connection, projection):
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
    document = read_json(
        args.path, 16 * 1024 * 1024, expected_uid=os.getuid()
    )
    entries = document.get("entries")
    if (
        document.get("document_type") != "orquesta.effective_config"
        or not isinstance(entries, list)
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
        values.get("test_attestor.microvm.launcher_socket")
        != args.launcher_socket
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
        or not re.fullmatch(
            r"(0|[1-9][0-9]*)", values.get("source_user_version", "")
        )
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
        "profile-sha",
        "adapter-sha",
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
