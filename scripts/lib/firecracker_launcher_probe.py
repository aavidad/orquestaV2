#!/usr/bin/env python3
"""Sonda local sin efectos para el UDS del launcher Firecracker."""

import argparse
import os
import socket
import stat
import struct
import sys


EXIT_CONFIG_INVALID = 40
EXIT_UNAVAILABLE = 41
EXIT_SOCKET_UNSAFE = 42
EXIT_IDENTITY_MISMATCH = 43


def fail(status: int) -> None:
    raise SystemExit(status)


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--socket", required=True)
    parser.add_argument("--trusted-uid", required=True, type=int)
    parser.add_argument("--trusted-gid", required=True, type=int)
    arguments = parser.parse_args()
    if arguments.trusted_uid < 0 or arguments.trusted_gid < 0:
        fail(EXIT_CONFIG_INVALID)
    return arguments


def validate_path(path: str, trusted_uid: int) -> os.stat_result:
    try:
        encoded_path = os.fsencode(path)
    except (TypeError, ValueError):
        fail(EXIT_CONFIG_INVALID)
    if (
        not path
        or not os.path.isabs(path)
        or os.path.normpath(path) != path
        or b"\0" in encoded_path
        or "\n" in path
        or "\r" in path
        or len(encoded_path) >= 108
    ):
        fail(EXIT_CONFIG_INVALID)
    if os.path.realpath(path) != path:
        fail(EXIT_SOCKET_UNSAFE)

    current = os.path.sep
    components = path.split(os.path.sep)[1:]
    for component in components[:-1]:
        current = os.path.join(current, component)
        try:
            metadata = os.lstat(current)
        except OSError:
            fail(EXIT_UNAVAILABLE)
        if (
            stat.S_ISLNK(metadata.st_mode)
            or not stat.S_ISDIR(metadata.st_mode)
            or metadata.st_uid not in (0, trusted_uid)
            or stat.S_IMODE(metadata.st_mode) & 0o022
        ):
            fail(EXIT_SOCKET_UNSAFE)

    try:
        metadata = os.lstat(path)
    except FileNotFoundError:
        fail(EXIT_UNAVAILABLE)
    except OSError:
        fail(EXIT_SOCKET_UNSAFE)
    if (
        stat.S_ISLNK(metadata.st_mode)
        or not stat.S_ISSOCK(metadata.st_mode)
        or metadata.st_uid != trusted_uid
        or stat.S_IMODE(metadata.st_mode) != 0o660
        or metadata.st_nlink != 1
    ):
        fail(EXIT_SOCKET_UNSAFE)
    return metadata


def probe(
    path: str,
    trusted_uid: int,
    trusted_gid: int,
    expected: os.stat_result,
) -> None:
    try:
        launcher = socket.socket(
            socket.AF_UNIX,
            socket.SOCK_SEQPACKET | socket.SOCK_CLOEXEC,
        )
    except OSError:
        fail(EXIT_UNAVAILABLE)
    try:
        launcher.settimeout(0.25)
        try:
            launcher.connect(path)
        except (FileNotFoundError, ConnectionError, PermissionError, TimeoutError, OSError):
            fail(EXIT_UNAVAILABLE)
        try:
            credentials = launcher.getsockopt(
                socket.SOL_SOCKET,
                socket.SO_PEERCRED,
                struct.calcsize("3i"),
            )
            peer_pid, peer_uid, peer_gid = struct.unpack("3i", credentials)
        except (OSError, struct.error):
            fail(EXIT_IDENTITY_MISMATCH)
        if (
            peer_pid <= 0
            or peer_uid != trusted_uid
            or peer_gid != trusted_gid
        ):
            fail(EXIT_IDENTITY_MISMATCH)
        try:
            observed = os.lstat(path)
        except OSError:
            fail(EXIT_SOCKET_UNSAFE)
        if (
            observed.st_dev != expected.st_dev
            or observed.st_ino != expected.st_ino
            or observed.st_uid != expected.st_uid
            or observed.st_gid != expected.st_gid
            or observed.st_mode != expected.st_mode
            or observed.st_nlink != expected.st_nlink
        ):
            fail(EXIT_SOCKET_UNSAFE)
    finally:
        launcher.close()


def main() -> None:
    if sys.platform != "linux":
        fail(EXIT_CONFIG_INVALID)
    arguments = parse_arguments()
    metadata = validate_path(arguments.socket, arguments.trusted_uid)
    probe(
        arguments.socket,
        arguments.trusted_uid,
        arguments.trusted_gid,
        metadata,
    )


if __name__ == "__main__":
    main()
