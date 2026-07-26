#!/usr/bin/env python3
"""Fail-closed verifier for a Firecracker host-binary build receipt."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import stat
import sys
from typing import Any

SCHEMA = "orquesta_firecracker_host_bundle_build.v1"
MAX_RECEIPT_BYTES = 16 * 1024
HEX = re.compile(r"^[0-9a-f]{64}$")
OBJECT_ID = re.compile(r"^[0-9a-f]{40}$|^[0-9a-f]{64}$")
VERSION = re.compile(r"^go[0-9A-Za-z._+-]+$")


class VerificationError(Exception):
    pass


def fail(code: str) -> None:
    raise VerificationError(code)


def exact_keys(value: Any, expected: set[str], label: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != expected:
        fail(f"{label}_schema")
    return value


def no_duplicate_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            fail("receipt_duplicate_key")
        result[key] = value
    return result


def digest_regular_file(
    path: str,
    *,
    max_bytes: int | None = None,
    capture: bool = False,
) -> tuple[str, int, int, bytes | None]:
    flags = os.O_RDONLY | os.O_CLOEXEC
    flags |= getattr(os, "O_NOFOLLOW", 0)
    try:
        descriptor = os.open(path, flags)
    except OSError:
        fail("file_open")
    try:
        before = os.fstat(descriptor)
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1:
            fail("file_metadata")
        if max_bytes is not None and before.st_size > max_bytes:
            fail("file_too_large")
        digest = hashlib.sha256()
        content = bytearray() if capture else None
        read_bytes = 0
        while True:
            block = os.read(descriptor, 1024 * 1024)
            if not block:
                break
            digest.update(block)
            if content is not None:
                content.extend(block)
            read_bytes += len(block)
            if max_bytes is not None and read_bytes > max_bytes:
                fail("file_too_large")
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
        if identity_after != identity_before or read_bytes != before.st_size:
            fail("file_changed")
        return (
            digest.hexdigest(),
            before.st_size,
            stat.S_IMODE(before.st_mode),
            bytes(content) if content is not None else None,
        )
    finally:
        os.close(descriptor)


def read_receipt(path: str, expected_sha256: str) -> dict[str, Any]:
    digest, size, mode, content = digest_regular_file(
        path,
        max_bytes=MAX_RECEIPT_BYTES,
        capture=True,
    )
    if digest != expected_sha256:
        fail("receipt_hash")
    if size == 0 or mode & 0o022:
        fail("receipt_metadata")
    try:
        if content is None:
            fail("receipt_content")
        raw = content.decode("utf-8")
        document = json.loads(raw, object_pairs_hook=no_duplicate_object)
    except (UnicodeDecodeError, json.JSONDecodeError):
        fail("receipt_json")
    return exact_keys(
        document,
        {
            "artifacts",
            "build",
            "builder",
            "environment",
            "schema_version",
            "source",
            "toolchain",
        },
        "receipt",
    )


def prefixed_sha(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value.startswith("sha256:"):
        fail(f"{label}_sha256")
    digest = value.removeprefix("sha256:")
    if not HEX.fullmatch(digest):
        fail(f"{label}_sha256")
    return digest


def validate_artifact(
    artifact: Any,
    *,
    label: str,
    expected_file: str,
    expected_package: str,
    path: str,
    expected_sha256: str,
) -> None:
    item = exact_keys(
        artifact,
        {"file", "mode", "package", "sha256", "size_bytes"},
        label,
    )
    if (
        item["file"] != expected_file
        or item["mode"] != "0755"
        or item["package"] != expected_package
        or isinstance(item["size_bytes"], bool)
        or not isinstance(item["size_bytes"], int)
        or item["size_bytes"] <= 0
    ):
        fail(f"{label}_contract")
    receipt_sha = prefixed_sha(item["sha256"], label)
    actual_sha, actual_size, actual_mode, _ = digest_regular_file(path)
    if (
        receipt_sha != expected_sha256
        or actual_sha != expected_sha256
        or actual_size != item["size_bytes"]
        or actual_mode != 0o755
    ):
        fail(f"{label}_identity")


def validate(args: argparse.Namespace) -> str:
    for label, value in (
        ("receipt", args.expected_receipt_sha256),
        ("source_archive", args.expected_source_archive_sha256),
        ("source_commit_object", args.expected_source_commit_object_sha256),
        ("toolchain_tree", args.expected_toolchain_tree_sha256),
        ("toolchain_go", args.expected_toolchain_go_sha256),
        ("builder", args.expected_builder_sha256),
        ("launcher", args.expected_launcher_sha256),
        ("supervisor", args.expected_supervisor_sha256),
    ):
        if not HEX.fullmatch(value):
            fail(f"expected_{label}_sha256")
    if not OBJECT_ID.fullmatch(args.expected_source_commit):
        fail("expected_source_commit")
    if not OBJECT_ID.fullmatch(args.expected_source_tree):
        fail("expected_source_tree")
    if not VERSION.fullmatch(args.expected_toolchain_version):
        fail("expected_toolchain_version")

    document = read_receipt(args.receipt, args.expected_receipt_sha256)
    if document["schema_version"] != SCHEMA:
        fail("receipt_schema_version")

    source = exact_keys(
        document["source"],
        {
            "archive_sha256",
            "commit",
            "commit_object_sha256",
            "export",
            "gitlinks",
            "tree_oid",
        },
        "source",
    )
    if (
        source["commit"] != args.expected_source_commit
        or source["tree_oid"] != args.expected_source_tree
        or source["export"] != "git_archive_exact_commit"
        or source["gitlinks"] is not False
        or prefixed_sha(source["archive_sha256"], "source_archive")
        != args.expected_source_archive_sha256
        or prefixed_sha(source["commit_object_sha256"], "source_commit_object")
        != args.expected_source_commit_object_sha256
    ):
        fail("source_identity")

    toolchain = exact_keys(
        document["toolchain"],
        {"go_binary_sha256", "platform", "tree_sha256", "version"},
        "toolchain",
    )
    if (
        toolchain["platform"] != "linux/amd64"
        or toolchain["version"] != args.expected_toolchain_version
        or prefixed_sha(toolchain["tree_sha256"], "toolchain_tree")
        != args.expected_toolchain_tree_sha256
        or prefixed_sha(toolchain["go_binary_sha256"], "toolchain_go")
        != args.expected_toolchain_go_sha256
    ):
        fail("toolchain_identity")

    build = exact_keys(
        document["build"],
        {
            "buildvcs",
            "cgo_enabled",
            "double_build",
            "goarch",
            "goos",
            "isolated_caches",
            "mod",
            "recipe",
            "source_date_epoch",
            "trimpath",
        },
        "build",
    )
    expected_build = {
        "buildvcs": False,
        "cgo_enabled": False,
        "double_build": True,
        "goarch": "amd64",
        "goos": "linux",
        "isolated_caches": True,
        "mod": "vendor",
        "recipe": "go build -mod=vendor -trimpath -buildvcs=false",
        "source_date_epoch": 0,
        "trimpath": True,
    }
    if build != expected_build:
        fail("build_contract")

    builder = exact_keys(document["builder"], {"file", "sha256"}, "builder")
    builder_sha, builder_size, builder_mode, _ = digest_regular_file(args.builder)
    if (
        builder["file"] != "build_firecracker_host_bundle.sh"
        or prefixed_sha(builder["sha256"], "builder") != args.expected_builder_sha256
        or builder_sha != args.expected_builder_sha256
        or builder_size <= 0
        or builder_mode & 0o022
    ):
        fail("builder_identity")

    environment = exact_keys(
        document["environment"],
        {
            "CGO_ENABLED",
            "GOCACHE",
            "GOENV",
            "GOMODCACHE",
            "GOOS",
            "GOARCH",
            "GOPATH",
            "GOPROXY",
            "GOSUMDB",
            "GOTOOLCHAIN",
            "LC_ALL",
            "SOURCE_DATE_EPOCH",
            "TMPDIR",
            "TZ",
        },
        "environment",
    )
    expected_environment = {
        "CGO_ENABLED": "0",
        "GOCACHE": "isolated_private",
        "GOENV": "off",
        "GOMODCACHE": "isolated_private",
        "GOOS": "linux",
        "GOARCH": "amd64",
        "GOPATH": "isolated_private",
        "GOPROXY": "off",
        "GOSUMDB": "off",
        "GOTOOLCHAIN": "local",
        "LC_ALL": "C",
        "SOURCE_DATE_EPOCH": "0",
        "TMPDIR": "isolated_private",
        "TZ": "UTC",
    }
    if environment != expected_environment:
        fail("environment_contract")

    artifacts = exact_keys(document["artifacts"], {"launcher", "supervisor"}, "artifacts")
    validate_artifact(
        artifacts["launcher"],
        label="launcher",
        expected_file="orquesta-firecracker-launcher",
        expected_package="./cmd/orquesta-firecracker-launcher",
        path=args.launcher,
        expected_sha256=args.expected_launcher_sha256,
    )
    validate_artifact(
        artifacts["supervisor"],
        label="supervisor",
        expected_file="orquesta-firecracker-attestor-e2e",
        expected_package="./cmd/orquesta-firecracker-attestor-e2e",
        path=args.supervisor,
        expected_sha256=args.expected_supervisor_sha256,
    )
    return hashlib.sha256(
        json.dumps(document, separators=(",", ":"), sort_keys=True).encode("utf-8")
    ).hexdigest()


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser()
    result.add_argument("--receipt", required=True)
    result.add_argument("--expected-receipt-sha256", required=True)
    result.add_argument("--expected-source-commit", required=True)
    result.add_argument("--expected-source-tree", required=True)
    result.add_argument("--expected-source-archive-sha256", required=True)
    result.add_argument("--expected-source-commit-object-sha256", required=True)
    result.add_argument("--expected-toolchain-version", required=True)
    result.add_argument("--expected-toolchain-tree-sha256", required=True)
    result.add_argument("--expected-toolchain-go-sha256", required=True)
    result.add_argument("--expected-builder-sha256", required=True)
    result.add_argument("--builder", required=True)
    result.add_argument("--launcher", required=True)
    result.add_argument("--expected-launcher-sha256", required=True)
    result.add_argument("--supervisor", required=True)
    result.add_argument("--expected-supervisor-sha256", required=True)
    return result


def main() -> int:
    try:
        args = parser().parse_args()
        canonical_digest = validate(args)
    except VerificationError as error:
        print(
            f"ORQUESTA_FIRECRACKER_HOST_RECEIPT_ERROR code={error}",
            file=sys.stderr,
        )
        return 1
    print(
        f"status=passed schema={SCHEMA} canonical_sha256={canonical_digest}",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
