#!/usr/bin/env python3
"""Fail-closed inventory, backup and drain adapter for Orquesta-owned processes."""

from __future__ import annotations

import argparse
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import secrets
import shutil
import stat
import subprocess
import sys
import time
import urllib.parse


OWNER_SCHEMA = "orquesta_codex_app_server_tmux_owner_generation.v0"
OWNER_REF = "orquesta-codex-goal-app-server-tmux-v0"
TMUX_FORMAT = "#{session_name}\t#{session_id}\t#{session_created}\t#{pane_id}\t#{pane_pid}\t#{pane_start_command}\t#{socket_path}"
BACKUP_TOKENS = (
    "checkpoint", "orquesta_goal_result", "agent_ack", "goal.log", "plan_state",
    "outbox", "director_decision", "owner.json", "server.pid", "base_url.txt",
)


def redact_cmdline(parts: list[str]) -> list[str]:
    redacted, hide_next = [], False
    for part in parts:
        lowered = part.lower()
        sensitive = any(token in lowered for token in ("token", "password", "passwd", "secret", "api_key", "api-key", "authorization"))
        if hide_next:
            redacted.append("<redacted>")
            hide_next = False
        elif sensitive and "=" in part:
            redacted.append(part.split("=", 1)[0] + "=<redacted>")
        elif sensitive:
            redacted.append(part)
            hide_next = True
        else:
            redacted.append(part)
    return redacted


def now_id(prefix: str) -> str:
    return f"{prefix}-{time.time_ns()}-{secrets.token_hex(8)}"


def atomic_json(path: Path, value: object) -> None:
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    temp = path.with_name(path.name + f".tmp.{os.getpid()}.{secrets.token_hex(4)}")
    with open(temp, "x", encoding="utf-8") as handle:
        os.chmod(temp, 0o600)
        json.dump(value, handle, sort_keys=True)
        handle.write("\n")
        handle.flush()
        os.fsync(handle.fileno())
    os.replace(temp, path)


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with open(path, "rb", buffering=0) as handle:
        while chunk := handle.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def stat_fields(proc_root: Path, pid: int) -> dict[str, str]:
    text = (proc_root / str(pid) / "stat").read_text(encoding="ascii").strip()
    end = text.rfind(")")
    if end < 0:
        raise ValueError("stat_comm_terminator_missing")
    fields = text[end + 2 :].split()
    if len(fields) <= 19:
        raise ValueError("stat_fields_missing")
    return {
        "ppid": fields[1], "pgid": fields[2], "process_session_id": fields[3],
        "start_ref": fields[19], "state": fields[0],
    }


def readlink_required(path: Path) -> str:
    return os.readlink(path)


def canonical_private_root(raw: str, *, create: bool = False) -> Path:
    path = Path(raw)
    if not path.is_absolute() or path == Path("/"):
        raise ValueError(f"unsafe_root:{raw}")
    current = Path("/")
    for part in path.parts[1:]:
        current /= part
        if os.path.lexists(current) and stat.S_ISLNK(os.lstat(current).st_mode):
            raise ValueError(f"symlink_root_component:{current}")
    if create:
        existed = path.exists()
        path.mkdir(mode=0o700, parents=True, exist_ok=True)
        if not existed:
            os.chmod(path, 0o700)
    current = Path("/")
    for part in path.parts[1:]:
        current /= part
        info = os.lstat(current)
        if stat.S_ISLNK(info.st_mode):
            raise ValueError(f"symlink_root_component:{current}")
    resolved = path.resolve(strict=True)
    if resolved != path:
        raise ValueError(f"noncanonical_root:{path}")
    info = path.stat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o022:
        raise ValueError(f"root_not_private_or_owned:{path}")
    return path


def under(path: str, root: Path) -> bool:
    if not path:
        return False
    try:
        return Path(path).resolve(strict=False).is_relative_to(root)
    except (OSError, ValueError):
        return False


def roots_overlap(left: Path, right: Path) -> bool:
    return left == right or left.is_relative_to(right) or right.is_relative_to(left)


class Drain:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.test_mode = os.environ.get("DRAIN_TEST_MODE") == "1"
        self.proc_root = Path(os.environ.get("ORQUESTA_DRAIN_PROC_ROOT", "/proc"))
        self.runtime_root = canonical_private_root(os.environ.get("ORQUESTA_DRAIN_RUNTIME_ROOT", "/srv/orquesta-self/runtime"))
        self.state_root = canonical_private_root(os.environ.get("ORQUESTA_DRAIN_STATE_ROOT", "/srv/orquesta-self/claude-director-20260705"))
        self.protected_root = canonical_private_root(os.environ.get("ORQUESTA_DRAIN_PROTECTED_APP_ROOT", "/srv/orquesta-self/uso-app"))
        backup_parent = canonical_private_root(os.environ.get("ORQUESTA_DRAIN_BACKUP_ROOT", "/srv/orquesta-self/backups"), create=True)
        backup_raw = os.environ.get("ORQUESTA_DRAIN_BACKUP_DIR")
        self.backup_id = now_id("backup-ref-orquesta-drain")
        self.backup_dir = Path(backup_raw) if backup_raw else backup_parent / self.backup_id
        self.receipt_path = Path(os.environ.get("ORQUESTA_DRAIN_RECEIPT", str(self.state_root / "state/orquesta_server_drain_receipt_v1.json")))
        self.base_url_file = Path(os.environ.get("ORQUESTA_DRAIN_BASE_URL_FILE", str(self.state_root / "runtime/base_url.txt")))
        self.wait_seconds = int(os.environ.get("ORQUESTA_DRAIN_WAIT_SECONDS", "8"))
        self.tmux = os.environ.get("ORQUESTA_DRAIN_TMUX_COMMAND", "tmux")
        self.curl = os.environ.get("ORQUESTA_DRAIN_CURL_COMMAND", "curl")
        self.pidfd_helper = os.environ.get("ORQUESTA_DRAIN_PIDFD_HELPER", str(Path(__file__).with_name("pidfd_signal.py")))
        self.actions: list[dict[str, object]] = []
        self.action_errors: list[str] = []
        self.receipt_id = now_id("receipt-ref-orquesta-drain")
        self.lock_handle = None
        self.validate_configuration(backup_parent)

    def validate_configuration(self, backup_parent: Path) -> None:
        if self.wait_seconds < 0 or self.wait_seconds > 300:
            raise ValueError("wait_seconds_invalid")
        if roots_overlap(self.runtime_root, self.state_root):
            raise ValueError("runtime_state_overlap")
        for root in (self.runtime_root, self.state_root, backup_parent):
            if roots_overlap(root, self.protected_root):
                raise ValueError("protected_root_overlap")
        if roots_overlap(self.backup_dir.resolve(strict=False), self.runtime_root) or roots_overlap(self.backup_dir.resolve(strict=False), self.state_root) or roots_overlap(self.backup_dir.resolve(strict=False), self.protected_root):
            raise ValueError("backup_root_overlap")
        if not self.backup_dir.is_absolute() or self.backup_dir.parent != backup_parent or self.backup_dir.exists() or self.backup_dir.is_symlink():
            raise ValueError("backup_dir_not_new_child_of_backup_root")
        if self.receipt_path.resolve(strict=False) == Path("/") or not self.receipt_path.is_absolute():
            raise ValueError("receipt_path_invalid")
        canonical_private_root(str(self.receipt_path.parent), create=True)
        if self.receipt_path.is_symlink():
            raise ValueError("receipt_path_symlink")
        if self.receipt_path.exists():
            receipt_info = os.lstat(self.receipt_path)
            if not stat.S_ISREG(receipt_info.st_mode) or receipt_info.st_uid != os.getuid() or receipt_info.st_mode & 0o022:
                raise ValueError("receipt_path_identity_invalid")
        if not self.base_url_file.resolve(strict=False).is_relative_to(self.state_root):
            raise ValueError("base_url_file_outside_state_root")
        overrides = any(name in os.environ for name in (
            "ORQUESTA_DRAIN_TMUX_COMMAND", "ORQUESTA_DRAIN_CURL_COMMAND",
            "ORQUESTA_DRAIN_PIDFD_HELPER", "ORQUESTA_DRAIN_TEST_HOLD_LOCK_SECONDS",
            "ORQUESTA_DRAIN_TEST_SYNTHETIC_SOCKET_FILES",
        ))
        synthetic = self.proc_root.resolve(strict=False) != Path("/proc")
        if self.test_mode:
            if not synthetic:
                raise ValueError("test_mode_requires_synthetic_proc_root")
            canonical_private_root(str(self.proc_root))
        elif synthetic or overrides:
            raise ValueError("test_override_forbidden_in_real_mode")
        for command in (self.tmux, self.curl, self.pidfd_helper):
            resolved = shutil.which(command) if "/" not in command else command
            if not resolved or not os.access(resolved, os.X_OK):
                raise ValueError(f"required_tool_missing:{command}")

    def socket_materialized(self, path: str | Path) -> bool:
        info = os.lstat(path)
        if stat.S_ISSOCK(info.st_mode) and not stat.S_ISLNK(info.st_mode):
            return True
        return bool(
            self.test_mode
            and os.environ.get("ORQUESTA_DRAIN_TEST_SYNTHETIC_SOCKET_FILES") == "1"
            and stat.S_ISREG(info.st_mode)
            and not stat.S_ISLNK(info.st_mode)
            and info.st_uid == os.getuid()
            and stat.S_IMODE(info.st_mode) == 0o600
        )

    def acquire_lock(self) -> None:
        lock_dir = self.state_root / "state"
        canonical_private_root(str(lock_dir), create=True)
        path = lock_dir / "orquesta_server_drain.lock"
        fd = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
        handle = os.fdopen(fd, "r+")
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
            handle.close()
            raise RuntimeError("drain_lock_identity_invalid")
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            handle.close()
            raise RuntimeError("drain_lock_busy")
        handle.seek(0)
        handle.truncate()
        handle.write(self.receipt_id + "\n")
        handle.flush()
        self.lock_handle = handle
        hold = float(os.environ.get("ORQUESTA_DRAIN_TEST_HOLD_LOCK_SECONDS", "0"))
        if hold:
            time.sleep(hold)

    def action(self, phase: str, action: str, outcome: str, identity: dict | None = None, detail: str = "") -> None:
        self.actions.append({
            "phase": phase, "action": action, "outcome": outcome,
            "identity": identity or {}, "detail": detail, "at_unix_ns": time.time_ns(),
        })
        if outcome in ("failed", "error", "identity_changed", "still_present"):
            self.action_errors.append(f"{phase}:{action}:{outcome}")

    def command(self, argv: list[str]) -> subprocess.CompletedProcess[str]:
        return subprocess.run(argv, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)

    def tmux_snapshot(self, exact_socket: str = "") -> tuple[list[dict[str, str]], list[str]]:
        argv = [self.tmux]
        if exact_socket:
            argv += ["-S", exact_socket]
        argv += ["list-panes", "-a", "-F", TMUX_FORMAT]
        result = self.command(argv)
        if result.returncode != 0:
            return [], [f"tmux_list_failed:rc={result.returncode}:{result.stderr.strip()[:200]}"]
        panes, errors = [], []
        for index, raw in enumerate(result.stdout.splitlines(), 1):
            values = raw.split("\t")
            if len(values) != 7 or not all(values[pos].strip() for pos in (0, 1, 2, 3, 4, 6)):
                errors.append(f"tmux_row_invalid:{index}")
                continue
            panes.append(dict(zip(("session_name", "session_id", "session_created", "pane_id", "pane_pid", "pane_start_command", "tmux_socket"), values)))
        return panes, errors

    def scan_markers(self) -> tuple[list[dict], list[str]]:
        markers, errors = [], []

        def walk_error(error: OSError) -> None:
            errors.append(f"marker_walk_error:{type(error).__name__}:{error.filename}")

        for base, dirs, files in os.walk(self.runtime_root, topdown=True, followlinks=False, onerror=walk_error):
            kept = []
            for name in dirs:
                path = Path(base) / name
                try:
                    if path.is_symlink():
                        errors.append(f"marker_symlink_dir:{path}")
                    else:
                        kept.append(name)
                except OSError as error:
                    errors.append(f"marker_dir_stat_error:{path}:{type(error).__name__}")
            dirs[:] = kept
            for name in files:
                if not name.endswith(".owner.json"):
                    continue
                path = Path(base) / name
                try:
                    info = os.lstat(path)
                    if not stat.S_ISREG(info.st_mode) or stat.S_ISLNK(info.st_mode) or info.st_uid != os.getuid() or info.st_size > 65536:
                        raise ValueError("marker_identity_invalid")
                    marker = json.loads(path.read_text(encoding="utf-8"))
                    required = (
                        "schema_version", "owner_ref", "session_name", "socket_path",
                        "app_server_pid", "app_server_start_ref", "app_server_process_group_id", "socket_owner_pid",
                        "socket_owner_start_ref", "tmux_session_id", "tmux_session_created",
                        "tmux_pane_pid", "tmux_pane_start_ref",
                    )
                    if marker.get("schema_version") != OWNER_SCHEMA or marker.get("owner_ref") != OWNER_REF or any(marker.get(key) in (None, "", 0) for key in required):
                        raise ValueError("marker_schema_or_fields_invalid")
                    socket_path = Path(str(marker["socket_path"]))
                    if not socket_path.is_absolute() or not socket_path.resolve(strict=False).is_relative_to(self.runtime_root) or path != Path(str(socket_path) + ".owner.json"):
                        raise ValueError("marker_socket_path_invalid")
                    marker["marker_path"] = str(path)
                    markers.append(marker)
                except (OSError, ValueError, json.JSONDecodeError, TypeError) as error:
                    errors.append(f"marker_invalid:{path}:{type(error).__name__}:{error}")
        return markers, errors

    def inventory(self) -> dict:
        errors: list[str] = []
        processes: dict[str, dict] = {}
        try:
            entries = sorted((item for item in os.listdir(self.proc_root) if item.isdigit()), key=int)
        except OSError as error:
            entries = []
            errors.append(f"proc_list_failed:{type(error).__name__}:{error}")
        for raw_pid in entries:
            pid = int(raw_pid)
            base = self.proc_root / raw_pid
            try:
                fields = stat_fields(self.proc_root, pid)
                cmd = [item.decode("utf-8", "replace") for item in (base / "cmdline").read_bytes().split(b"\0") if item]
                processes[raw_pid] = {
                    "pid": raw_pid, **fields, "cmdline": redact_cmdline(cmd),
                    "exe": readlink_required(base / "exe"), "cwd": readlink_required(base / "cwd"),
                    "classification": "ignored", "classification_reason": "not_managed",
                    "role": "", "marker_refs": [], "tmux": None, "listening_ports": [],
                }
            except (OSError, ValueError, IndexError) as error:
                errors.append(f"proc_read_failed:{raw_pid}:{type(error).__name__}:{error}")

        panes, tmux_errors = self.tmux_snapshot()
        errors.extend(tmux_errors)
        pane_by_pid = {pane["pane_pid"]: pane for pane in panes}
        for pid, process in processes.items():
            if pid in pane_by_pid:
                pane = dict(pane_by_pid[pid])
                pane["pane_start_ref"] = process["start_ref"]
                process["tmux"] = pane

        markers, marker_errors = self.scan_markers()
        errors.extend(marker_errors)
        managed_seed: set[str] = set()
        marker_pids: set[str] = set()
        for marker in markers:
            identities = (
                ("app_server_pid", "app_server_start_ref"),
                ("socket_owner_pid", "socket_owner_start_ref"),
                ("tmux_pane_pid", "tmux_pane_start_ref"),
            )
            marker_ok = True
            for pid_key, start_key in identities:
                pid = str(marker[pid_key])
                marker_pids.add(pid)
                process = processes.get(pid)
                if not process or process["start_ref"] != str(marker[start_key]):
                    errors.append(f"marker_process_identity_mismatch:{marker['marker_path']}:{pid_key}")
                    marker_ok = False
                else:
                    process["marker_refs"].append(marker["marker_path"])
            app_process = processes.get(str(marker["app_server_pid"]))
            if app_process and app_process["pgid"] != str(marker["app_server_process_group_id"]):
                errors.append(f"marker_process_group_mismatch:{marker['marker_path']}")
                marker_ok = False
            pane_pid = str(marker["tmux_pane_pid"])
            socket_pid = str(marker["socket_owner_pid"])
            cursor, seen = socket_pid, set()
            while cursor in processes and cursor not in seen and cursor != pane_pid:
                seen.add(cursor)
                cursor = processes[cursor]["ppid"]
            if cursor != pane_pid:
                errors.append(f"marker_socket_owner_not_pane_descendant:{marker['marker_path']}")
                marker_ok = False
            pane = next((item for item in panes if item["session_id"] == str(marker["tmux_session_id"])), None)
            if not pane or any(str(pane[key]) != str(marker[marker_key]) for key, marker_key in (
                ("session_name", "session_name"), ("session_created", "tmux_session_created"),
                ("pane_pid", "tmux_pane_pid"),
            )):
                errors.append(f"marker_tmux_identity_mismatch:{marker['marker_path']}")
                marker_ok = False
            if pane and not pane.get("tmux_socket"):
                errors.append(f"marker_tmux_socket_missing:{marker['marker_path']}")
                marker_ok = False
            if pane and pane.get("tmux_socket"):
                try:
                    tmux_socket_info = os.lstat(pane["tmux_socket"])
                    if not self.socket_materialized(pane["tmux_socket"]):
                        raise ValueError("not_socket")
                except (OSError, ValueError) as error:
                    errors.append(f"marker_tmux_socket_not_materialized:{pane['tmux_socket']}:{type(error).__name__}")
                    marker_ok = False
            socket_path = Path(str(marker["socket_path"]))
            try:
                if not self.socket_materialized(socket_path):
                    raise ValueError("not_socket")
            except (OSError, ValueError) as error:
                errors.append(f"marker_socket_not_materialized:{socket_path}:{type(error).__name__}")
                marker_ok = False
            if marker_ok:
                managed_seed.update(str(marker[key]) for key in ("app_server_pid", "socket_owner_pid", "tmux_pane_pid"))

        server_pid_file = self.state_root / "server.pid"
        server_pid = ""
        if server_pid_file.exists() or server_pid_file.is_symlink():
            try:
                info = os.lstat(server_pid_file)
                if not stat.S_ISREG(info.st_mode) or stat.S_ISLNK(info.st_mode) or info.st_uid != os.getuid():
                    raise ValueError("server_pid_identity_invalid")
                server_pid = server_pid_file.read_text(encoding="ascii").strip()
                if not server_pid.isdigit() or server_pid not in processes:
                    raise ValueError("server_pid_not_live")
                server = processes[server_pid]
                if not under(server["exe"], self.runtime_root) or not Path(server["exe"]).name.startswith("orquesta-server"):
                    raise ValueError("server_exe_not_managed")
                managed_seed.add(server_pid)
                server["role"] = "orquesta_server"
            except (OSError, ValueError) as error:
                errors.append(f"server_pid_invalid:{type(error).__name__}:{error}")

        protected_seed = {pid for pid, process in processes.items() if under(process["exe"], self.protected_root) or under(process["cwd"], self.protected_root)}
        protected = set(protected_seed)
        changed = True
        while changed:
            changed = False
            for pid, process in processes.items():
                if process["ppid"] in protected and pid not in protected:
                    protected.add(pid)
                    changed = True

        for pid in managed_seed:
            if pid in protected:
                errors.append(f"managed_identity_overlaps_protected:{pid}")
                continue
            process = processes.get(pid)
            if process:
                process["classification"] = "drainable"
                process["classification_reason"] = "managed_identity_complete"
                if not process["role"]:
                    process["role"] = "codex_app_server"
        changed = True
        while changed:
            changed = False
            for pid, process in processes.items():
                parent = processes.get(process["ppid"])
                if process["classification"] == "ignored" and parent and parent["classification"] == "drainable" and pid not in protected:
                    process["classification"] = "drainable"
                    process["classification_reason"] = "descendant_of_managed_identity"
                    process["role"] = "managed_descendant"
                    changed = True
        for pid in protected:
            if pid in processes:
                processes[pid]["classification"] = "protected"
                processes[pid]["classification_reason"] = "protected_uso_app" if pid in protected_seed else "protected_uso_app_descendant"

        for pid, process in processes.items():
            exe_name = Path(process["exe"]).name
            app_shape = bool(process["cmdline"] and Path(process["cmdline"][0]).name == "codex" and "app-server" in process["cmdline"][1:])
            tmux_shape = bool(process["tmux"] and process["tmux"]["session_name"].startswith("orquesta-goal-"))
            if process["classification"] == "ignored" and (exe_name.startswith("orquesta-server") or app_shape or tmux_shape or pid in marker_pids):
                process["classification"] = "ambiguous"
                process["classification_reason"] = "orquesta_identity_incomplete"

        if server_pid and server_pid in processes and processes[server_pid]["classification"] == "drainable":
            try:
                ports = self.listening_ports(server_pid)
                processes[server_pid]["listening_ports"] = ports
            except (OSError, ValueError) as error:
                errors.append(f"server_socket_inventory_failed:{type(error).__name__}:{error}")

        relevant = sorted((item for item in processes.values() if item["classification"] != "ignored"), key=lambda item: int(item["pid"]))
        sessions = []
        for marker in markers:
            pane = next((item for item in panes if item["session_id"] == str(marker["tmux_session_id"])), None)
            sessions.append({
                "session_name": marker.get("session_name", ""), "session_id": marker.get("tmux_session_id", ""),
                "session_created": marker.get("tmux_session_created", ""), "pane_pid": str(marker.get("tmux_pane_pid", "")),
                "pane_start_ref": marker.get("tmux_pane_start_ref", ""), "tmux_socket": pane.get("tmux_socket", "") if pane else "",
                "classification": "drainable" if pane and not errors and str(marker.get("tmux_pane_pid")) in managed_seed else "ambiguous",
                "marker_path": marker.get("marker_path", ""),
            })
        return {
            "schema_version": "orquesta_server_drain_inventory.v2", "captured_at_unix_ns": time.time_ns(),
            "processes": relevant,
            "drainable": [item for item in relevant if item["classification"] == "drainable"],
            "protected": [item for item in relevant if item["classification"] == "protected"],
            "ambiguous": [item for item in relevant if item["classification"] == "ambiguous"],
            "tmux_sessions": sessions, "owner_markers": markers, "inventory_errors": errors,
        }

    def listening_ports(self, pid: str) -> list[int]:
        fd_dir = self.proc_root / pid / "fd"
        inodes = set()
        for name in os.listdir(fd_dir):
            target = os.readlink(fd_dir / name)
            if target.startswith("socket:[") and target.endswith("]"):
                inodes.add(target[8:-1])
        ports = set()
        for name in ("tcp", "tcp6"):
            path = self.proc_root / "net" / name
            for raw in path.read_text(encoding="ascii").splitlines()[1:]:
                fields = raw.split()
                if len(fields) < 10 or fields[3] != "0A" or fields[9] not in inodes:
                    continue
                ports.add(int(fields[1].split(":")[1], 16))
        return sorted(ports)

    def validate_base_url(self, before: dict) -> tuple[str, list[str]]:
        servers = [item for item in before["drainable"] if item.get("role") == "orquesta_server"]
        if not servers:
            if self.base_url_file.exists() or self.base_url_file.is_symlink():
                return "", ["base_url_without_managed_server"]
            return "", []
        if len(servers) != 1:
            return "", ["managed_server_count_not_one"]
        try:
            info = os.lstat(self.base_url_file)
            if not stat.S_ISREG(info.st_mode) or stat.S_ISLNK(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o022:
                raise ValueError("base_url_file_identity_invalid")
            raw = self.base_url_file.read_text(encoding="utf-8").strip()
            parsed = urllib.parse.urlsplit(raw)
            if parsed.scheme != "http" or parsed.username is not None or parsed.password is not None or parsed.query or parsed.fragment or parsed.path not in ("", "/"):
                raise ValueError("base_url_structure_invalid")
            host = parsed.hostname
            if not host or not ipaddress.ip_address(host).is_loopback:
                raise ValueError("base_url_host_not_loopback_ip")
            port = parsed.port
            if port is None or not 1 <= port <= 65535 or port not in servers[0].get("listening_ports", []):
                raise ValueError("base_url_port_not_owned_by_server")
            return f"http://{host if ':' not in host else '[' + host + ']'}:{port}", []
        except (OSError, ValueError) as error:
            return "", [f"base_url_invalid:{type(error).__name__}:{error}"]

    def backup(self) -> dict:
        copied, errors = [], []
        try:
            self.backup_dir.mkdir(mode=0o700, parents=False, exist_ok=False)
            files_root = self.backup_dir / "files"
            files_root.mkdir(mode=0o700)
        except OSError as error:
            return {"schema_version": "orquesta_server_drain_backup.v2", "backup_id": self.backup_id, "backup_dir": str(self.backup_dir), "status": "failed", "copied": [], "errors": [f"backup_create_failed:{type(error).__name__}:{error}"], "source_preserved": False}

        def walk_error(error: OSError) -> None:
            errors.append(f"backup_walk_error:{type(error).__name__}:{error.filename}")

        for label, root in (("state", self.state_root), ("runtime", self.runtime_root)):
            for base, dirs, files in os.walk(root, topdown=True, followlinks=False, onerror=walk_error):
                kept = []
                for name in dirs:
                    path = Path(base) / name
                    try:
                        if path.is_symlink():
                            if any(token in str(path.relative_to(root)).lower() for token in BACKUP_TOKENS):
                                errors.append(f"backup_external_symlink:{path}")
                        else:
                            kept.append(name)
                    except OSError as error:
                        errors.append(f"backup_dir_error:{path}:{type(error).__name__}")
                dirs[:] = kept
                for name in files:
                    source = Path(base) / name
                    relative = source.relative_to(root)
                    if not any(token in str(relative).lower() for token in BACKUP_TOKENS):
                        continue
                    try:
                        before = os.lstat(source)
                        if stat.S_ISLNK(before.st_mode):
                            raise ValueError("external_symlink")
                        if not stat.S_ISREG(before.st_mode):
                            raise ValueError("reference_not_materialized_regular_file")
                        destination = files_root / label / relative
                        destination.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
                        digest = hashlib.sha256()
                        with open(source, "rb", buffering=0) as src, open(destination, "xb", buffering=0) as dst:
                            os.chmod(destination, 0o600)
                            while chunk := src.read(1024 * 1024):
                                digest.update(chunk)
                                dst.write(chunk)
                            os.fsync(dst.fileno())
                        after = os.lstat(source)
                        destination_hash = file_sha256(destination)
                        if (before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns) != (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns) or destination_hash != digest.hexdigest():
                            raise ValueError("source_changed_or_hash_mismatch")
                        copied.append({"source": str(source), "backup_relative": str(destination.relative_to(self.backup_dir)), "size": before.st_size, "sha256": digest.hexdigest(), "source_preserved": True})
                    except (OSError, ValueError) as error:
                        errors.append(f"backup_copy_failed:{source}:{type(error).__name__}:{error}")
        manifest = {
            "schema_version": "orquesta_server_drain_backup.v2", "backup_id": self.backup_id,
            "backup_dir": str(self.backup_dir), "created_at_unix_ns": time.time_ns(),
            "status": "complete" if not errors else "failed", "copied": copied,
            "errors": errors, "source_preserved": not errors and all(item["source_preserved"] for item in copied),
        }
        try:
            atomic_json(self.backup_dir / "manifest.json", manifest)
        except OSError as error:
            manifest["status"] = "failed"
            manifest["source_preserved"] = False
            manifest["errors"].append(f"backup_manifest_failed:{type(error).__name__}:{error}")
        self.action("backup", "materialize_artifacts", "completed" if manifest["status"] == "complete" else "failed", detail=self.backup_id)
        return manifest

    def identity_current(self, identity: dict) -> bool:
        try:
            observed = stat_fields(self.proc_root, int(identity["pid"]))
            return observed["start_ref"] == str(identity["start_ref"])
        except (OSError, ValueError, KeyError):
            return False

    def wait_gone(self, identity: dict, phase: str) -> bool:
        deadline = time.monotonic() + self.wait_seconds
        while self.identity_current(identity) and time.monotonic() < deadline:
            time.sleep(0.1 if self.test_mode else 0.5)
        gone = not self.identity_current(identity)
        if gone:
            outcome = "gone"
        elif phase == "wait_after_sigkill":
            outcome = "still_present"
        else:
            outcome = "pending_escalation"
        self.action(phase, "wait_identity", outcome, identity)
        return gone

    def send_signal(self, identity: dict, signal_name: str, phase: str) -> bool:
        if not self.identity_current(identity):
            self.action(phase, f"signal_{signal_name}", "identity_changed", identity, "no_signal")
            return False
        if self.args.dry_run:
            self.action(phase, f"signal_{signal_name}", "dry_run", identity)
            return True
        result = self.command([self.pidfd_helper, "--pid", str(identity["pid"]), "--start-ref", str(identity["start_ref"]), "--signal", signal_name])
        try:
            payload = json.loads(result.stdout)
        except json.JSONDecodeError:
            payload = {}
        if result.returncode == 0 and payload.get("outcome") == "sent":
            self.action(phase, f"signal_{signal_name}", "sent", identity)
            return True
        self.action(phase, f"signal_{signal_name}", "failed", identity, f"rc={result.returncode} outcome={payload.get('outcome', 'invalid')} error={payload.get('error', '')}")
        return False

    def request_http(self, base_url: str, server_identity: dict | None) -> None:
        if not base_url or not server_identity:
            self.action("http_shutdown", "post_server_shutdown", "skipped", detail="managed_server_absent")
            return
        request_id = now_id("request-ref-orquesta-drain")
        payload = {
            "request_id": request_id, "idempotency_key": request_id,
            "requested_by": "orquesta-director", "cleanup_goal_backends": True,
        }
        if self.args.dry_run:
            self.action("http_shutdown", "post_server_shutdown", "dry_run", server_identity, request_id)
            return
        payload_path = self.state_root / "state" / f".{request_id}.json"
        atomic_json(payload_path, payload)
        try:
            result = self.command([self.curl, "-fsS", "-m", "20", "-X", "POST", base_url + "/api/v0/server/shutdown", "-H", "Content-Type: application/json", "--data-binary", "@" + str(payload_path)])
        finally:
            payload_path.unlink(missing_ok=True)
        if result.returncode == 0:
            self.action("http_shutdown", "post_server_shutdown", "accepted", server_identity, request_id)
        else:
            self.action("http_shutdown", "post_server_shutdown", "failed", server_identity, f"rc={result.returncode}")

    def tmux_current(self, session: dict) -> bool:
        panes, errors = self.tmux_snapshot(str(session["tmux_socket"]))
        if errors:
            self.action("tmux_revalidate", "list_exact_socket", "failed", session, ";".join(errors))
            return False
        pane = next((item for item in panes if item["session_id"] == session["session_id"]), None)
        if not pane or any(str(pane[key]) != str(session[expected]) for key, expected in (
            ("session_created", "session_created"), ("pane_pid", "pane_pid"), ("tmux_socket", "tmux_socket"),
        )):
            return False
        try:
            return stat_fields(self.proc_root, int(session["pane_pid"]))["start_ref"] == str(session["pane_start_ref"])
        except (OSError, ValueError):
            return False

    def kill_tmux(self, session: dict) -> None:
        if not self.tmux_current(session):
            self.action("tmux_exact", "kill_session", "identity_changed", session, "no_kill_session")
            return
        if self.args.dry_run:
            self.action("tmux_exact", "kill_session", "dry_run", session)
            return
        result = self.command([self.tmux, "-S", str(session["tmux_socket"]), "kill-session", "-t", str(session["session_id"])])
        if result.returncode == 0:
            self.action("tmux_exact", "kill_session", "sent", session, "immutable_session_id")
        else:
            self.action("tmux_exact", "kill_session", "failed", session, f"rc={result.returncode}")

    def write_receipt(self, before: dict, after: dict, backup: dict, decision: str, status: str, exit_code: int) -> None:
        protected_key = lambda item: (item.get("pid"), item.get("start_ref"))
        before_protected = {protected_key(item) for item in before.get("protected", [])}
        after_protected = {protected_key(item) for item in after.get("protected", [])}
        protected_pids = {item[0] for item in before_protected}
        protected_signals = any(str(action.get("action", "")).startswith("signal_") and action.get("identity", {}).get("pid") in protected_pids for action in self.actions)
        receipt = {
            "schema_version": "orquesta_server_drain_receipt.v2", "receipt_id": self.receipt_id,
            "dry_run": self.args.dry_run, "decision": decision, "drain_status": status,
            "exit_code": exit_code, "backup_prepared_before_stop": backup.get("status") == "complete",
            "backup": backup, "before": before, "after": after, "inventory_errors": before.get("inventory_errors", []) + after.get("inventory_errors", []),
            "actions": self.actions, "action_errors": self.action_errors,
            "uso_app_protected": {
                "root": str(self.protected_root), "intact": before_protected.issubset(after_protected),
                "signals_sent": protected_signals, "before_identities": before.get("protected", []),
                "after_identities": after.get("protected", []),
            },
            "created_at_unix_ns": time.time_ns(),
        }
        atomic_json(self.receipt_path, receipt)

    def run(self) -> int:
        self.acquire_lock()
        before = self.inventory()
        base_url, url_errors = self.validate_base_url(before)
        before["inventory_errors"].extend(url_errors)
        backup = self.backup()
        refused = bool(before["inventory_errors"] or before["ambiguous"] or backup.get("status") != "complete" or not backup.get("source_preserved"))
        if refused:
            decision = "refused_precondition"
            self.action("decision", "authorize_mutation", "refused", detail="inventory_or_backup_precondition_failed")
        elif self.args.dry_run:
            decision = "dry_run"
        else:
            decision = "authorized"

        identities = before["drainable"]
        server = next((item for item in identities if item.get("role") == "orquesta_server"), None)
        if not refused:
            self.request_http(base_url, server)
            if not self.args.dry_run:
                for identity in identities:
                    self.wait_gone(identity, "wait_after_http")
            for identity in identities:
                if self.identity_current(identity):
                    self.send_signal(identity, "SIGTERM", "sigterm")
            if not self.args.dry_run:
                for identity in identities:
                    if self.identity_current(identity):
                        self.wait_gone(identity, "wait_after_sigterm")
            for session in before["tmux_sessions"]:
                if session["classification"] == "drainable":
                    self.kill_tmux(session)
            if not self.args.dry_run:
                for session in before["tmux_sessions"]:
                    if session["classification"] == "drainable":
                        deadline = time.monotonic() + self.wait_seconds
                        while self.tmux_current(session) and time.monotonic() < deadline:
                            time.sleep(0.1 if self.test_mode else 0.5)
                        self.action("wait_after_tmux", "wait_session", "gone" if not self.tmux_current(session) else "pending_escalation", session)
                for identity in identities:
                    if self.identity_current(identity):
                        self.send_signal(identity, "SIGKILL", "sigkill_final")
                        self.wait_gone(identity, "wait_after_sigkill")

        after = self.inventory()
        protected_before = {(item["pid"], item["start_ref"]) for item in before["protected"]}
        protected_after = {(item["pid"], item["start_ref"]) for item in after["protected"]}
        protected_intact = protected_before.issubset(protected_after)
        if refused:
            status, exit_code = "refused", 3
        elif self.args.dry_run:
            status, exit_code = "dry_run", 0
        elif after["inventory_errors"] or after["drainable"] or after["ambiguous"] or self.action_errors or not protected_intact:
            status, exit_code = "residual", 4
        else:
            status, exit_code = "clean", 0
        self.write_receipt(before, after, backup, decision, status, exit_code)
        stream = sys.stdout if exit_code == 0 else sys.stderr
        outcome = "ok" if exit_code == 0 else "not_ok"
        print(f"orquesta_server_drain={outcome} drain_status={status} receipt={self.receipt_path} backup={self.backup_dir}", file=stream)
        return exit_code


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--drain", action="store_true")
    parser.add_argument("--confirm-drain", default="")
    args = parser.parse_args()
    if args.drain:
        if args.confirm_drain != "orquesta-server-drain" or args.dry_run:
            parser.error("drain real requiere --drain --confirm-drain orquesta-server-drain")
        args.dry_run = False
    else:
        if args.confirm_drain:
            parser.error("--confirm-drain sin --drain")
        args.dry_run = True
    return args


def main() -> int:
    try:
        return Drain(parse_args()).run()
    except (OSError, ValueError, RuntimeError) as error:
        print(f"orquesta_server_drain=not_ok drain_status=refused preflight_error={type(error).__name__}:{error}", file=sys.stderr)
        return 73 if str(error) == "drain_lock_busy" else 2


if __name__ == "__main__":
    raise SystemExit(main())
