#!/usr/bin/env python3
"""Signal one already-open Linux process object after start-ref validation."""

from __future__ import annotations

import argparse
import json
import os
import signal
import sys


def process_start_ref(pid: int) -> str:
    with open(f"/proc/{pid}/stat", encoding="ascii") as handle:
        text = handle.read().strip()
    end = text.rfind(")")
    if end < 0:
        raise ValueError("proc_stat_comm_terminator_missing")
    fields = text[end + 2 :].split()
    if len(fields) <= 19:
        raise ValueError("proc_stat_start_ref_missing")
    return fields[19]


def emit(outcome: str, pid: int, expected: str, observed: str = "", error: str = "") -> None:
    print(
        json.dumps(
            {
                "schema_version": "orquesta_pidfd_signal_result.v0",
                "outcome": outcome,
                "pid": pid,
                "expected_start_ref": expected,
                "observed_start_ref": observed,
                "error": error,
            },
            sort_keys=True,
        )
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pid", required=True, type=int)
    parser.add_argument("--start-ref", required=True)
    parser.add_argument("--signal", required=True, choices=("SIGTERM", "SIGKILL"))
    args = parser.parse_args()

    if args.pid <= 0 or not args.start_ref.strip():
        emit("invalid_identity", args.pid, args.start_ref, error="pid_or_start_ref_invalid")
        return 2
    if not hasattr(os, "pidfd_open") or not hasattr(signal, "pidfd_send_signal"):
        emit("unsupported", args.pid, args.start_ref, error="python_pidfd_api_unavailable")
        return 3

    pidfd = -1
    try:
        # Opening first pins the process object. The subsequent start-ref read
        # rejects a recycled numeric PID, and pidfd_send_signal addresses that
        # same pinned kernel object rather than performing another PID lookup.
        pidfd = os.pidfd_open(args.pid, 0)
        observed = process_start_ref(args.pid)
        if observed != args.start_ref:
            emit("identity_mismatch", args.pid, args.start_ref, observed)
            return 4
        signal.pidfd_send_signal(pidfd, getattr(signal, args.signal), None, 0)
        emit("sent", args.pid, args.start_ref, observed)
        return 0
    except ProcessLookupError as error:
        emit("process_gone", args.pid, args.start_ref, error=type(error).__name__)
        return 5
    except (OSError, ValueError) as error:
        emit("error", args.pid, args.start_ref, error=f"{type(error).__name__}:{error}")
        return 6
    finally:
        if pidfd >= 0:
            os.close(pidfd)


if __name__ == "__main__":
    raise SystemExit(main())
