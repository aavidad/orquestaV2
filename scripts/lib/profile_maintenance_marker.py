#!/usr/bin/env python3
"""Lee un marker de mantenimiento privado sin seguir enlaces."""

import os
import re
import stat
import sys


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit(2)
    path = sys.argv[1]
    try:
        expected_uid = int(sys.argv[2])
    except ValueError:
        raise SystemExit(2)

    flags = os.O_RDONLY | os.O_CLOEXEC
    if not hasattr(os, "O_NOFOLLOW"):
        raise SystemExit(1)
    flags |= os.O_NOFOLLOW
    try:
        descriptor = os.open(path, flags)
    except OSError:
        raise SystemExit(1)
    try:
        metadata = os.fstat(descriptor)
        if (
            not stat.S_ISREG(metadata.st_mode)
            or metadata.st_uid != expected_uid
            or stat.S_IMODE(metadata.st_mode) != 0o600
            or metadata.st_nlink != 1
            or metadata.st_size != 81
        ):
            raise SystemExit(1)
        content = os.read(descriptor, 82)
        if os.read(descriptor, 1) != b"":
            raise SystemExit(1)
    finally:
        os.close(descriptor)

    match = re.fullmatch(b"maintenance_ref=([0-9a-f]{64})\\n", content)
    if match is None:
        raise SystemExit(1)
    print(match.group(1).decode("ascii"))


if __name__ == "__main__":
    main()
