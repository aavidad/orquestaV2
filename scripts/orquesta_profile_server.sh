#!/usr/bin/env bash
# Arranca un daemon Orquesta ligado a un único perfil persistente.
#
# El daemon de bootstrap conserva un HOME privado vacío. Su runtime usa el
# perfil persistente configurado y admite una sola ejecución concurrente.

set -euo pipefail
umask 077

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
readonly PIDFD_SIGNAL="$SCRIPT_DIR/lib/pidfd_signal.py"
readonly FIRECRACKER_LAUNCHER_PROBE="$SCRIPT_DIR/lib/firecracker_launcher_probe.py"
readonly PROFILE_MAINTENANCE_MARKER_READER="$SCRIPT_DIR/lib/profile_maintenance_marker.py"

fail() {
  printf 'orquesta_profile_server: status=error reason_code=%s\n' "$1" >&2
  exit "${2:-1}"
}

fail_action() {
  printf 'orquesta_profile_server: status=error reason_code=%s action=%s\n' \
    "$1" "$2" >&2
  exit "${3:-1}"
}

usage() {
  cat <<'EOF'
uso:
  orquesta_profile_server.sh start --profile NOMBRE --runtime-base RUTA \
    --source-codex-home RUTA --binary RUTA --config RUTA --exec-path PATH \
    [--maintenance-ref SHA256]
  orquesta_profile_server.sh status --profile NOMBRE --runtime-base RUTA \
    [--maintenance-ref SHA256]
  orquesta_profile_server.sh stop --profile NOMBRE --runtime-base RUTA \
    [--maintenance-ref SHA256]

Cada daemon queda ligado a un único perfil persistente y admite una sola
ejecución concurrente.
EOF
}

require_value() {
  [ "$#" -ge 2 ] && [ -n "$2" ] || fail "argument_value_missing" 2
}

readonly ACTION="${1:-}"
[ -n "$ACTION" ] || {
  usage >&2
  exit 2
}
shift

profile=""
runtime_base=""
source_codex_home=""
binary=""
config_source=""
exec_path=""
maintenance_ref=""

while [ "$#" -gt 0 ]; do
  option="$1"
  case "$option" in
    --profile)
      require_value "$@"
      [ -z "$profile" ] || fail "argument_repeated" 2
      profile="$2"
      shift 2
      ;;
    --runtime-base)
      require_value "$@"
      [ -z "$runtime_base" ] || fail "argument_repeated" 2
      runtime_base="$2"
      shift 2
      ;;
    --source-codex-home)
      require_value "$@"
      [ -z "$source_codex_home" ] || fail "argument_repeated" 2
      source_codex_home="$2"
      shift 2
      ;;
    --binary)
      require_value "$@"
      [ -z "$binary" ] || fail "argument_repeated" 2
      binary="$2"
      shift 2
      ;;
    --config)
      require_value "$@"
      [ -z "$config_source" ] || fail "argument_repeated" 2
      config_source="$2"
      shift 2
      ;;
    --exec-path)
      require_value "$@"
      [ -z "$exec_path" ] || fail "argument_repeated" 2
      exec_path="$2"
      shift 2
      ;;
    --maintenance-ref)
      require_value "$@"
      [ -z "$maintenance_ref" ] || fail "argument_repeated" 2
      maintenance_ref="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "argument_unknown" 2
      ;;
  esac
done

case "$ACTION" in
  start)
    [ -n "$profile" ] &&
      [ -n "$runtime_base" ] &&
      [ -n "$source_codex_home" ] &&
      [ -n "$binary" ] &&
      [ -n "$config_source" ] &&
      [ -n "$exec_path" ] || fail "start_arguments_incomplete" 2
    ;;
  status|stop)
    [ -n "$profile" ] && [ -n "$runtime_base" ] ||
      fail "control_arguments_incomplete" 2
    [ -z "$source_codex_home" ] &&
      [ -z "$binary" ] &&
      [ -z "$config_source" ] &&
      [ -z "$exec_path" ] || fail "control_arguments_excess" 2
    ;;
  *)
    fail "action_invalid" 2
    ;;
esac

[[ "$profile" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$ ]] ||
  fail "profile_invalid" 2
[ -z "$maintenance_ref" ] ||
  [[ "$maintenance_ref" =~ ^[0-9a-f]{64}$ ]] ||
  fail "maintenance_ref_invalid" 2

command -v uname >/dev/null 2>&1 || fail "platform_probe_unavailable"
[ "$(uname -s 2>/dev/null)" = "Linux" ] || fail "platform_unsupported"

readonly CURRENT_UID="$(id -u)"

canonical_existing() {
  candidate="$1"
  [ "${candidate#/}" != "$candidate" ] || return 1
  [ ! -L "$candidate" ] || return 1
  canonical="$(realpath -e -- "$candidate" 2>/dev/null)" || return 1
  [ "$canonical" = "$candidate" ]
}

owner_is_current() {
  [ "$(stat -Lc '%u' -- "$1" 2>/dev/null)" = "$CURRENT_UID" ]
}

mode_is_exact() {
  [ "$(stat -Lc '%a' -- "$1" 2>/dev/null)" = "$2" ]
}

private_regular_file() {
  candidate="$1"
  canonical_existing "$candidate" &&
    [ -f "$candidate" ] &&
    owner_is_current "$candidate" &&
    case "$(stat -Lc '%a' -- "$candidate" 2>/dev/null)" in
      400|600) true ;;
      *) false ;;
    esac
}

private_directory() {
  candidate="$1"
  canonical_existing "$candidate" &&
    [ -d "$candidate" ] &&
    owner_is_current "$candidate" &&
    mode_is_exact "$candidate" 700
}

secure_account_ancestor() {
  candidate="$1"
  canonical_existing "$candidate" &&
    [ -d "$candidate" ] &&
    case "$(stat -Lc '%u' -- "$candidate" 2>/dev/null)" in
      0|"$CURRENT_UID") true ;;
      *) false ;;
    esac &&
    [ $((8#$(stat -Lc '%a' -- "$candidate") & 8#022)) -eq 0 ]
}

validate_account_home_ancestors() {
  current="$(dirname "$1")"
  while :; do
    secure_account_ancestor "$current" || return 1
    parent="$(dirname "$current")"
    [ "$parent" != "$current" ] || return 0
    current="$parent"
  done
}

private_executable() {
  candidate="$1"
  canonical_existing "$candidate" &&
    [ -f "$candidate" ] &&
    [ -x "$candidate" ] &&
    [ $((8#$(stat -Lc '%a' -- "$candidate") & 8#022)) -eq 0 ]
}

run_microvm_launcher_probe() {
  launcher_socket="$1"
  private_executable "$FIRECRACKER_LAUNCHER_PROBE" || return 44
  "$FIRECRACKER_LAUNCHER_PROBE" \
    --socket "$launcher_socket" \
    --trusted-uid 0 \
    --trusted-gid 0 >/dev/null 2>&1
}

fail_microvm_launcher_probe() {
  case "$1" in
    0) ;;
    40)
      fail_action \
        "test_attestor_microvm_launcher_config_invalid" \
        "repair_microvm_launcher_socket_configuration"
      ;;
    41)
      fail_action \
        "test_attestor_microvm_launcher_unavailable" \
        "start_or_repair_microvm_launcher"
      ;;
    42)
      fail_action \
        "test_attestor_microvm_launcher_socket_unsafe" \
        "repair_microvm_launcher_socket"
      ;;
    43)
      fail_action \
        "test_attestor_microvm_launcher_identity_mismatch" \
        "restore_trusted_microvm_launcher"
      ;;
    *)
      fail_action \
        "test_attestor_microvm_launcher_probe_failed" \
        "inspect_microvm_launcher"
      ;;
  esac
}

probe_microvm_launcher_socket() {
  if run_microvm_launcher_probe "$1"; then
    return
  else
    probe_status="$?"
  fi
  fail_microvm_launcher_probe "$probe_status"
}

probe_persisted_microvm_launcher() {
  private_regular_file "$CONFIG_TARGET" || fail "daemon_contract_invalid"
  expected_config_sha="$(read_identity_file "$CONFIG_SHA_FILE")" ||
    fail "daemon_contract_invalid"
  [ "$(sha256sum -- "$CONFIG_TARGET" | awk '{print $1}')" = \
    "$expected_config_sha" ] || fail "daemon_contract_invalid"
  persisted_attestor="$(
    python3 - "$CONFIG_TARGET" <<'PY'
import sys
import tomllib

with open(sys.argv[1], "rb") as handle:
    document = tomllib.load(handle)
attestor = document.get("test_attestor", {})
provider = attestor.get("provider", "disabled") if isinstance(attestor, dict) else ""
microvm = attestor.get("microvm", {}) if isinstance(attestor, dict) else {}
socket_path = microvm.get("launcher_socket", "") if isinstance(microvm, dict) else ""
if not isinstance(provider, str) or not isinstance(socket_path, str):
    raise SystemExit(1)
if provider != "microvm":
    socket_path = "-"
print(provider)
print(socket_path)
PY
  )" 2>/dev/null || fail "daemon_contract_invalid"
  mapfile -t persisted_attestor_lines <<<"$persisted_attestor"
  [ "${#persisted_attestor_lines[@]}" -eq 2 ] ||
    fail "daemon_contract_invalid"
  if [ "${persisted_attestor_lines[0]}" = "microvm" ]; then
    probe_microvm_launcher_socket "${persisted_attestor_lines[1]}"
  fi
}

private_directory "$runtime_base" || fail "runtime_base_not_private"
readonly RUNTIME_ROOT="$runtime_base/$profile"

prepare_runtime_root() {
  if [ ! -e "$RUNTIME_ROOT" ]; then
    mkdir -- "$RUNTIME_ROOT" || fail "runtime_root_create_failed"
    chmod 700 -- "$RUNTIME_ROOT" || fail "runtime_root_permissions_failed"
  fi
  private_directory "$RUNTIME_ROOT" || fail "runtime_root_not_private"
  for directory in \
    "$RUNTIME_ROOT/run" \
    "$RUNTIME_ROOT/logs" \
    "$RUNTIME_ROOT/config" \
    "$RUNTIME_ROOT/daemon-home"; do
    if [ ! -e "$directory" ]; then
      mkdir -- "$directory" || fail "runtime_directory_create_failed"
      chmod 700 -- "$directory" || fail "runtime_directory_not_private"
    fi
    private_directory "$directory" || fail "runtime_directory_not_private"
  done
}

readonly PID_FILE="$RUNTIME_ROOT/run/server.pid"
readonly START_REF_FILE="$RUNTIME_ROOT/run/server.start_ref"
readonly BINARY_ID_FILE="$RUNTIME_ROOT/run/server.binary_id"
readonly BINARY_SHA_FILE="$RUNTIME_ROOT/run/server.binary_sha256"
readonly CONFIG_SHA_FILE="$RUNTIME_ROOT/run/server.config_sha256"
readonly START_FAILED_FILE="$RUNTIME_ROOT/run/server.start_failed"
readonly SOURCE_BINDING_FILE="$RUNTIME_ROOT/run/source_code_home.sha256"
readonly CONFIG_TARGET="$RUNTIME_ROOT/config/orquesta.toml"
readonly LOG_OUT="$RUNTIME_ROOT/logs/server.stdout.log"
readonly LOG_ERR="$RUNTIME_ROOT/logs/server.stderr.log"
readonly DAEMON_HOME="$RUNTIME_ROOT/daemon-home"
readonly GLOBAL_LOCK_ROOT="$runtime_base/.profile-locks"
readonly GLOBAL_CONTROL_LOCK="$GLOBAL_LOCK_ROOT/$profile.control.lock"
readonly GLOBAL_LEASE_LOCK="$GLOBAL_LOCK_ROOT/$profile.lease.lock"
readonly GLOBAL_MAINTENANCE_MARKER="$GLOBAL_LOCK_ROOT/$profile.maintenance"

numeric_pid() {
  [[ "$1" =~ ^[1-9][0-9]*$ ]]
}

process_start_ref() {
  python3 - "$1" <<'PY'
import sys

with open(f"/proc/{int(sys.argv[1])}/stat", encoding="ascii") as handle:
    text = handle.read().strip()
end = text.rfind(")")
if end < 0:
    raise SystemExit(1)
fields = text[end + 2:].split()
if len(fields) <= 19:
    raise SystemExit(1)
print(fields[19])
PY
}

pid_alive() {
  pid="$1"
  numeric_pid "$pid" || return 1
  kill -0 "$pid" 2>/dev/null || return 1
  state="$(python3 - "$pid" <<'PY' 2>/dev/null || true
import sys

with open(f"/proc/{int(sys.argv[1])}/stat", encoding="ascii") as handle:
    text = handle.read().strip()
end = text.rfind(")")
fields = text[end + 2:].split()
print(fields[0] if fields else "")
PY
)"
  [ "$state" != "Z" ]
}

read_identity_file() {
  path="$1"
  private_regular_file "$path" || return 1
  value="$(<"$path")"
  [ -n "$value" ] || return 1
  printf '%s' "$value"
}

daemon_identity_matches() {
  pid="$1"
  expected_start_ref="$(read_identity_file "$START_REF_FILE")" || return 1
  expected_binary_id="$(read_identity_file "$BINARY_ID_FILE")" || return 1
  observed_start_ref="$(process_start_ref "$pid" 2>/dev/null)" || return 1
  observed_binary_id="$(stat -Lc '%d:%i' -- "/proc/$pid/exe" 2>/dev/null)" ||
    return 1
  [ "$observed_start_ref" = "$expected_start_ref" ] &&
    [ "$observed_binary_id" = "$expected_binary_id" ]
}

daemon_state() {
  [ -e "$PID_FILE" ] || {
    printf 'stopped'
    return
  }
  pid="$(read_identity_file "$PID_FILE")" || {
    printf 'invalid'
    return
  }
  if ! pid_alive "$pid"; then
    printf 'stopped'
    return
  fi
  if daemon_identity_matches "$pid"; then
    printf 'running'
    return
  fi
  printf 'foreign'
}

safe_write_value() {
  target="$1"
  value="$2"
  temporary="$(mktemp "$RUNTIME_ROOT/run/.identity.XXXXXX")"
  chmod 600 -- "$temporary"
  printf '%s\n' "$value" >"$temporary"
  mv -f -- "$temporary" "$target"
}

remove_stale_identity() {
  rm -f -- \
    "$PID_FILE" \
    "$START_REF_FILE" \
    "$BINARY_ID_FILE" \
    "$BINARY_SHA_FILE" \
    "$CONFIG_SHA_FILE" \
    "$START_FAILED_FILE"
}

prepare_global_profile_locks() {
  command -v flock >/dev/null 2>&1 || fail "profile_lock_unavailable"
  if [ ! -e "$GLOBAL_LOCK_ROOT" ]; then
    if [ "$ACTION" = "status" ]; then
      [ ! -e "$RUNTIME_ROOT" ] || fail "profile_lock_root_missing"
      return
    fi
    if mkdir -m 700 -- "$GLOBAL_LOCK_ROOT" 2>/dev/null; then
      :
    elif [ ! -d "$GLOBAL_LOCK_ROOT" ]; then
      fail "profile_lock_root_create_failed"
    fi
  fi
  private_directory "$GLOBAL_LOCK_ROOT" || fail "profile_lock_root_not_private"
  for lock_file in "$GLOBAL_CONTROL_LOCK" "$GLOBAL_LEASE_LOCK"; do
    if [ ! -e "$lock_file" ]; then
      [ "$ACTION" != "status" ] || fail "profile_lock_missing"
      (umask 077; : >"$lock_file") || fail "profile_lock_create_failed"
    fi
    private_regular_file "$lock_file" || fail "profile_lock_not_private"
  done
  exec 8<>"$GLOBAL_CONTROL_LOCK"
  flock -w 10 8 || fail "profile_control_busy"
}

read_maintenance_marker() {
  private_executable "$PROFILE_MAINTENANCE_MARKER_READER" || return 1
  "$PROFILE_MAINTENANCE_MARKER_READER" \
    "$GLOBAL_MAINTENANCE_MARKER" \
    "$CURRENT_UID"
}

authorize_profile_mutation() {
  if [ ! -e "$GLOBAL_MAINTENANCE_MARKER" ] &&
    [ ! -L "$GLOBAL_MAINTENANCE_MARKER" ]; then
    [ -z "$maintenance_ref" ] || fail "maintenance_marker_missing"
    return
  fi
  observed_maintenance_ref="$(read_maintenance_marker 2>/dev/null)" ||
    fail "maintenance_marker_invalid"
  [ -n "$maintenance_ref" ] || fail "profile_maintenance_active"
  [ "$maintenance_ref" = "$observed_maintenance_ref" ] ||
    fail "maintenance_ref_mismatch"
}

prepare_global_profile_locks

if [ "$ACTION" != "status" ]; then
  authorize_profile_mutation
fi

case "$ACTION" in
  status)
    [ -e "$RUNTIME_ROOT" ] || {
      printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
      exit 3
    }
    private_directory "$RUNTIME_ROOT" || fail "runtime_root_not_private"
    state="$(daemon_state)"
    case "$state" in
      running)
        [ ! -e "$START_FAILED_FILE" ] ||
          fail "daemon_previous_start_not_ready"
        probe_persisted_microvm_launcher
        pid="$(read_identity_file "$PID_FILE")"
        printf 'orquesta_profile_server: status=running profile=%s pid=%s\n' \
          "$profile" "$pid"
        exit 0
        ;;
      stopped)
        printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
        exit 3
        ;;
      *)
        fail "daemon_identity_mismatch"
        ;;
    esac
    ;;
  stop)
    [ -e "$RUNTIME_ROOT" ] || {
      printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
      exit 0
    }
    private_directory "$RUNTIME_ROOT" || fail "runtime_root_not_private"
    state="$(daemon_state)"
    case "$state" in
      stopped)
        remove_stale_identity
        printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
        exit 0
        ;;
      foreign|invalid)
        fail "daemon_identity_mismatch"
        ;;
    esac
    pid="$(read_identity_file "$PID_FILE")"
    start_ref="$(read_identity_file "$START_REF_FILE")"
    [ -x "$PIDFD_SIGNAL" ] || fail "pidfd_helper_unavailable"
    if ! "$PIDFD_SIGNAL" \
      --pid "$pid" \
      --start-ref "$start_ref" \
      --signal SIGTERM >/dev/null; then
      fail "cooperative_stop_not_sent"
    fi
    for _ in $(seq 1 300); do
      if ! pid_alive "$pid"; then
        remove_stale_identity
        printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
        exit 0
      fi
      sleep 0.1
    done
    fail "cooperative_stop_timeout"
    ;;
esac

prepare_runtime_root

private_directory "$source_codex_home" ||
  fail_action "source_code_home_not_private" "chmod_0700_source_code_home"
private_directory "$(dirname "$source_codex_home")" ||
  fail_action "account_home_root_not_private" "chmod_0700_account_home_root"
validate_account_home_ancestors "$(dirname "$source_codex_home")" ||
  fail_action "account_home_ancestor_insecure" "secure_account_home_ancestors"
readonly SOURCE_AUTH="$source_codex_home/auth.json"
private_regular_file "$SOURCE_AUTH" ||
  fail_action "source_auth_not_private" "chmod_0600_source_auth"
[ "$(stat -Lc '%h' -- "$SOURCE_AUTH")" = "1" ] ||
  fail "source_auth_link_count_invalid"

private_executable "$binary" || fail "binary_invalid"
private_regular_file "$config_source" || fail "orquesta_config_not_private"
[ "$(stat -Lc '%s' -- "$config_source")" -le 4194304 ] ||
  fail "orquesta_config_too_large"

IFS=':' read -r -a exec_path_entries <<<"$exec_path"
[ "${#exec_path_entries[@]}" -gt 0 ] || fail "exec_path_invalid"
for directory in "${exec_path_entries[@]}"; do
  canonical_existing "$directory" &&
    [ -d "$directory" ] &&
    [ $((8#$(stat -Lc '%a' -- "$directory") & 8#022)) -eq 0 ] ||
    fail "exec_path_invalid"
done

readonly REQUESTED_BINARY_SHA="$(sha256sum -- "$binary" | awk '{print $1}')"
readonly REQUESTED_CONFIG_SHA="$(sha256sum -- "$config_source" | awk '{print $1}')"

config_temporary="$(mktemp "$RUNTIME_ROOT/config/.orquesta.XXXXXX")"
install -m 600 -- "$config_source" "$config_temporary"
[ "$(sha256sum -- "$config_temporary" | awk '{print $1}')" = \
  "$REQUESTED_CONFIG_SHA" ] &&
  [ "$(sha256sum -- "$config_source" | awk '{print $1}')" = \
  "$REQUESTED_CONFIG_SHA" ] || fail "orquesta_config_projection_changed"

validation_output="$(
  python3 - \
    "$config_temporary" \
    "$RUNTIME_ROOT" \
    "$CURRENT_UID" \
    "$source_codex_home" \
    "$profile" <<'PY'
import ipaddress
import os
import stat
import sys
import tomllib

config_path, runtime_root, current_uid_text, source_home, profile = sys.argv[1:]
current_uid = int(current_uid_text)

with open(config_path, "rb") as handle:
    value = tomllib.load(handle)

def nested(*parts):
    current = value
    for part in parts:
        if not isinstance(current, dict) or part not in current:
            raise ValueError("missing")
        current = current[part]
    return current

listen = nested("server", "listen")
if not isinstance(listen, str) or not listen or "\n" in listen or "\r" in listen:
    raise ValueError("listen")
if listen.startswith("["):
    close = listen.find("]")
    if close < 0 or close + 2 > len(listen) or listen[close + 1] != ":":
        raise ValueError("listen")
    host, port_text = listen[1:close], listen[close + 2:]
else:
    host, separator, port_text = listen.rpartition(":")
    if not separator:
        raise ValueError("listen")
address = ipaddress.ip_address(host)
port = int(port_text)
if not address.is_loopback or not 1 <= port <= 65535:
    raise ValueError("listen")

root = os.path.normpath(runtime_root)
if not os.path.isabs(root) or os.path.realpath(root) != root:
    raise ValueError("root")

path_keys = (
    ("state", "sqlite", "path"),
    ("artifact", "filesystem", "root"),
    ("credentials", "local", "path"),
    ("runtime", "codex", "work_root"),
    ("runtime", "codex", "cache_root"),
    ("workspace", "local", "root"),
    ("identity", "local_token_path"),
    ("config", "effective_path"),
)
directory_keys = {
    ("artifact", "filesystem", "root"),
    ("runtime", "codex", "work_root"),
    ("runtime", "codex", "cache_root"),
    ("workspace", "local", "root"),
}

def check_no_symlink(path):
    current = os.path.sep
    for component in os.path.normpath(path).split(os.path.sep)[1:]:
        current = os.path.join(current, component)
        try:
            metadata = os.lstat(current)
        except FileNotFoundError:
            continue
        if stat.S_ISLNK(metadata.st_mode):
            raise ValueError("symlink")

def ensure_private_directory(path):
    relative = os.path.relpath(path, root)
    current = root
    for component in relative.split(os.path.sep):
        if component in ("", "."):
            continue
        current = os.path.join(current, component)
        try:
            metadata = os.lstat(current)
        except FileNotFoundError:
            os.mkdir(current, 0o700)
            metadata = os.lstat(current)
        if (
            not stat.S_ISDIR(metadata.st_mode)
            or metadata.st_uid != current_uid
            or stat.S_IMODE(metadata.st_mode) != 0o700
        ):
            raise ValueError("directory")

for key in path_keys:
    path = nested(*key)
    if (
        not isinstance(path, str)
        or not os.path.isabs(path)
        or os.path.normpath(path) != path
        or os.path.commonpath((root, path)) != root
        or "\n" in path
        or "\r" in path
    ):
        raise ValueError("path")
    check_no_symlink(path)
    target_directory = path if key in directory_keys else os.path.dirname(path)
    ensure_private_directory(target_directory)

allowlist = nested("runtime", "codex", "env_allowlist")
if (
    not isinstance(allowlist, list)
    or len(allowlist) != 3
    or set(allowlist) != {"PATH", "HOME", "CODEX_HOME"}
):
    raise ValueError("allowlist")

go_toolchain_root = nested("runtime", "codex", "go_toolchain_root")
if not isinstance(go_toolchain_root, str):
    raise ValueError("go_toolchain")
if go_toolchain_root:
    if (
        not os.path.isabs(go_toolchain_root)
        or os.path.normpath(go_toolchain_root) != go_toolchain_root
        or os.path.realpath(go_toolchain_root) != go_toolchain_root
        or "\n" in go_toolchain_root
        or "\r" in go_toolchain_root
    ):
        raise ValueError("go_toolchain")
    current = go_toolchain_root
    while True:
        metadata = os.lstat(current)
        if (
            not stat.S_ISDIR(metadata.st_mode)
            or metadata.st_uid != 0
            or metadata.st_gid != 0
            or stat.S_IMODE(metadata.st_mode) & 0o022
        ):
            raise ValueError("go_toolchain")
        parent = os.path.dirname(current)
        if parent == current:
            break
        current = parent
    pending = [go_toolchain_root]
    while pending:
        current = pending.pop()
        metadata = os.lstat(current)
        if (
            stat.S_ISLNK(metadata.st_mode)
            or metadata.st_uid != 0
            or metadata.st_gid != 0
            or stat.S_IMODE(metadata.st_mode) & 0o022
        ):
            raise ValueError("go_toolchain")
        if stat.S_ISDIR(metadata.st_mode):
            with os.scandir(current) as entries:
                pending.extend(entry.path for entry in entries)
        elif not stat.S_ISREG(metadata.st_mode):
            raise ValueError("go_toolchain")
    go_executable = os.path.join(go_toolchain_root, "bin", "go")
    metadata = os.lstat(go_executable)
    if (
        not stat.S_ISREG(metadata.st_mode)
        or metadata.st_uid != 0
        or metadata.st_gid != 0
        or not metadata.st_mode & 0o111
        or metadata.st_mode & 0o022
    ):
        raise ValueError("go_toolchain")

account_home_root = nested("runtime", "codex", "account_home_root")
account_profile = nested("runtime", "codex", "account_profile")
account_auth_max_document_bytes = nested(
    "runtime", "codex", "account_auth_max_document_bytes"
)
max_concurrent = nested("runtime", "codex", "max_concurrent_executions")
if (
    account_home_root != os.path.dirname(source_home)
    or account_profile != profile
    or os.path.basename(source_home) != profile
    or os.path.join(account_home_root, account_profile) != source_home
    or type(account_auth_max_document_bytes) is not int
    or account_auth_max_document_bytes <= 0
    or type(max_concurrent) is not int
    or max_concurrent != 1
):
    raise ValueError("account_binding")

command = nested("runtime", "codex", "command")
if (
    not isinstance(command, str)
    or not os.path.isabs(command)
    or os.path.normpath(command) != command
    or os.path.realpath(command) != command
):
    raise ValueError("command")
metadata = os.stat(command, follow_symlinks=False)
if (
    not stat.S_ISREG(metadata.st_mode)
    or not metadata.st_mode & stat.S_IXUSR
    or metadata.st_mode & 0o022
):
    raise ValueError("command")

token_path = nested("identity", "local_token_path")
project_ref = nested("project", "default")
if (
    not isinstance(project_ref, str)
    or not project_ref
    or "\n" in project_ref
    or "\r" in project_ref
):
    raise ValueError("project")
test_attestor = value.get("test_attestor")
attestor_provider = (
    test_attestor.get("provider")
    if isinstance(test_attestor, dict)
    else None
)
if attestor_provider == "bubblewrap":
    attestor_max_concurrent = test_attestor.get("max_concurrent_runs")
    attestor_max_subject_bytes = test_attestor.get("max_subject_bytes")
    bubblewrap = test_attestor.get("bubblewrap")
    bubblewrap_command = (
        bubblewrap.get("command")
        if isinstance(bubblewrap, dict)
        else None
    )
    repository = value.get("repository")
    repository_local = (
        repository.get("local")
        if isinstance(repository, dict)
        else None
    )
    repository_seed_path = (
        repository_local.get("seed_path")
        if isinstance(repository_local, dict)
        else None
    )
    if (
        type(attestor_max_concurrent) is not int
        or not 1 <= attestor_max_concurrent <= 64
        or type(attestor_max_subject_bytes) is not int
        or not 1048576 <= attestor_max_subject_bytes <= 8589934592
        or not isinstance(bubblewrap_command, str)
        or not os.path.isabs(bubblewrap_command)
        or os.path.normpath(bubblewrap_command) != bubblewrap_command
        or os.path.realpath(bubblewrap_command) != bubblewrap_command
        or not isinstance(repository_seed_path, str)
        or not repository_seed_path
        or not os.path.isabs(repository_seed_path)
        or os.path.normpath(repository_seed_path) != repository_seed_path
    ):
        raise ValueError("test_attestor")
    bubblewrap_metadata = os.stat(bubblewrap_command, follow_symlinks=False)
    if (
        not stat.S_ISREG(bubblewrap_metadata.st_mode)
        or not bubblewrap_metadata.st_mode & stat.S_IXUSR
        or bubblewrap_metadata.st_mode & 0o022
    ):
        raise ValueError("test_attestor")
    launcher_socket = "-"
elif attestor_provider == "microvm":
    microvm = test_attestor.get("microvm")
    launcher_socket = (
        microvm.get("launcher_socket")
        if isinstance(microvm, dict)
        else None
    )
    if not isinstance(launcher_socket, str) or not launcher_socket:
        raise ValueError("test_attestor")
    attestor_max_concurrent = 0
    attestor_max_subject_bytes = 0
    bubblewrap_command = "-"
    repository_seed_path = "-"
else:
    attestor_provider = "other"
    attestor_max_concurrent = 0
    attestor_max_subject_bytes = 0
    bubblewrap_command = "-"
    repository_seed_path = "-"
    launcher_socket = "-"
print(listen)
print(token_path)
print(project_ref)
print(account_auth_max_document_bytes)
print(attestor_provider)
print(attestor_max_concurrent)
print(attestor_max_subject_bytes)
print(bubblewrap_command)
print(repository_seed_path)
print(launcher_socket)
PY
)" 2>/dev/null || fail "orquesta_config_invalid"

mapfile -t validation_lines <<<"$validation_output"
[ "${#validation_lines[@]}" -eq 10 ] || fail "orquesta_config_invalid"
listen="${validation_lines[0]}"
token_path="${validation_lines[1]}"
project_ref="${validation_lines[2]}"
account_auth_max_document_bytes="${validation_lines[3]}"
attestor_provider="${validation_lines[4]}"
attestor_max_concurrent="${validation_lines[5]}"
attestor_max_subject_bytes="${validation_lines[6]}"
bubblewrap_command="${validation_lines[7]}"
repository_seed_path="${validation_lines[8]}"
launcher_socket="${validation_lines[9]}"
[ -n "$listen" ] &&
  [ -n "$token_path" ] &&
  [ -n "$project_ref" ] &&
  [[ "$account_auth_max_document_bytes" =~ ^[1-9][0-9]*$ ]] ||
  fail "orquesta_config_invalid"

[ "$(stat -Lc '%s' -- "$SOURCE_AUTH")" -gt 0 ] &&
  [ "$(stat -Lc '%s' -- "$SOURCE_AUTH")" -le \
  "$account_auth_max_document_bytes" ] || fail "source_auth_too_large"
python3 - "$SOURCE_AUTH" <<'PY' >/dev/null 2>&1 ||
import json
import sys

with open(sys.argv[1], "rb") as handle:
    value = json.load(handle)
if not isinstance(value, dict) or not value:
    raise SystemExit(1)
PY
  fail "source_auth_invalid"

state="$(daemon_state)"
case "$state" in
  running)
    [ ! -e "$START_FAILED_FILE" ] ||
      fail "daemon_previous_start_not_ready"
    running_binary_sha="$(read_identity_file "$BINARY_SHA_FILE")" ||
      fail "daemon_contract_invalid"
    running_config_sha="$(read_identity_file "$CONFIG_SHA_FILE")" ||
      fail "daemon_contract_invalid"
    [ "$running_binary_sha" = "$REQUESTED_BINARY_SHA" ] &&
      [ "$running_config_sha" = "$REQUESTED_CONFIG_SHA" ] ||
      fail "daemon_already_running_different_contract"
    if [ "$attestor_provider" = "microvm" ]; then
      probe_microvm_launcher_socket "$launcher_socket"
    fi
    rm -f -- "$config_temporary"
    pid="$(read_identity_file "$PID_FILE")"
    printf 'orquesta_profile_server: status=running profile=%s pid=%s\n' \
      "$profile" "$pid"
    exit 0
    ;;
  foreign|invalid)
    rm -f -- "$config_temporary"
    fail "daemon_identity_mismatch"
    ;;
  stopped)
    remove_stale_identity
    ;;
esac

preflight_bubblewrap_capacity() {
  git_command="$(PATH="$exec_path" command -v git 2>/dev/null || true)"
  if [ -z "$git_command" ] || ! private_executable "$git_command"; then
    fail_action \
      "test_attestor_git_unavailable" \
      "install_trusted_git_in_exec_path"
  fi

  set +e
  python3 - \
    "$git_command" \
    "$repository_seed_path" \
    "$attestor_max_concurrent" \
    "$attestor_max_subject_bytes" \
    "$bubblewrap_command" <<'PY' >/dev/null 2>&1
import fcntl
import os
import posixpath
import resource
import stat
import subprocess
import sys

(
    git_command,
    repository,
    concurrent_text,
    max_subject_text,
    bubblewrap_command,
) = sys.argv[1:]
descriptor_global_reserve = 64
descriptor_run_reserve = 16
# Hasta 128 argumentos del RequiredTestSpec más CLI y holgura de protocolo.
argument_count_margin = 192
fixed_runner_arguments = 72
max_subject_entries = 1_000_000

try:
    concurrent = int(concurrent_text)
    max_subject_bytes = int(max_subject_text)
    metadata = os.lstat(repository)
    if (
        concurrent < 1
        or max_subject_bytes < 1048576
        or not os.path.isabs(repository)
        or os.path.normpath(repository) != repository
        or os.path.realpath(repository) != repository
        or not stat.S_ISDIR(metadata.st_mode)
    ):
        raise ValueError("repository")
except (OSError, ValueError):
    raise SystemExit(41)

try:
    soft, hard = resource.getrlimit(resource.RLIMIT_NOFILE)
    infinity = resource.RLIM_INFINITY
    open_descriptors = len(os.listdir("/proc/self/fd"))
    if (
        soft in (-1, infinity)
        or hard in (-1, infinity)
        or soft <= 0
        or hard <= 0
        or soft > hard
    ):
        raise ValueError("rlimit")
    available = soft - open_descriptors - descriptor_global_reserve
    per_run = available // concurrent - descriptor_run_reserve
except (OSError, ValueError):
    raise SystemExit(42)

if per_run <= 0:
    raise SystemExit(40)

try:
    tree_result = subprocess.run(
        [
            git_command,
            "--no-pager",
            "-C",
            repository,
            "rev-parse",
            "--verify",
            "HEAD^{tree}",
        ],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    tree_oid = tree_result.stdout.strip()
    if (
        tree_result.returncode != 0
        or len(tree_oid) not in (40, 64)
        or any(character not in b"0123456789abcdef" for character in tree_oid)
    ):
        raise ValueError("head")
    process = subprocess.Popen(
        [
            git_command,
            "--no-pager",
            "-C",
            repository,
            "ls-tree",
            "-r",
            "-z",
            "--full-tree",
            tree_oid.decode("ascii"),
        ],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
    )
except (OSError, UnicodeError, ValueError):
    raise SystemExit(41)

pending = b""
regular_entries = 0
symlink_entries = 0
directories = set()
try:
    while True:
        chunk = process.stdout.read(65536)
        if not chunk:
            break
        pending += chunk
        records = pending.split(b"\0")
        pending = records.pop()
        if len(pending) > 8192:
            raise ValueError("entry")
        for record in records:
            header, separator, name_bytes = record.partition(b"\t")
            fields = header.split(b" ")
            if not separator or len(fields) != 3:
                raise ValueError("entry")
            mode = fields[0]
            try:
                name = name_bytes.decode("utf-8")
            except UnicodeDecodeError:
                raise ValueError("entry")
            if (
                not name
                or name.startswith("/")
                or posixpath.normpath(name) != name
                or any(part in ("", ".", "..") for part in name.split("/"))
            ):
                raise ValueError("entry")
            parent = posixpath.dirname(name) or "."
            depth = 0
            while parent != ".":
                depth += 1
                if depth > 256:
                    raise ValueError("entry")
                directories.add(parent)
                parent = posixpath.dirname(parent) or "."
            if mode in (b"100644", b"100755"):
                regular_entries += 1
                if regular_entries > per_run:
                    process.kill()
                    process.wait()
                    raise SystemExit(40)
            elif mode == b"120000":
                symlink_entries += 1
            elif mode == b"160000":
                raise ValueError("gitlink")
            else:
                raise ValueError("mode")
    if pending or process.wait() != 0:
        raise ValueError("tree")
except SystemExit:
    raise
except (OSError, ValueError):
    process.kill()
    process.wait()
    raise SystemExit(41)

entry_budget = min(max_subject_bytes // 1024 + 1, max_subject_entries)
if regular_entries + symlink_entries + len(directories) > entry_budget:
    raise SystemExit(43)

required_arguments = (
    fixed_runner_arguments
    + 2 * len(directories)
    + 5 * regular_entries
    + 3 * symlink_entries
    + argument_count_margin
)
probe_base = [
    "--unshare-all",
    "--unshare-user",
    "--disable-userns",
    "--assert-userns-disabled",
    "--die-with-parent",
    "--new-session",
    "--clearenv",
    "--ro-bind",
    "/",
    "/",
]
if required_arguments < len(probe_base):
    raise SystemExit(44)
probe_arguments = probe_base + ["--unshare-ipc"] * (
    required_arguments - len(probe_base)
)
payload = b"\0".join(value.encode("utf-8") for value in probe_arguments) + b"\0"
if len(payload) > max_subject_bytes:
    raise SystemExit(43)

try:
    descriptor = os.memfd_create(
        "orquesta-bwrap-argument-probe",
        os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING,
    )
    try:
        if os.write(descriptor, payload) != len(payload):
            raise OSError("short write")
        os.fchmod(descriptor, 0o400)
        fcntl.fcntl(
            descriptor,
            fcntl.F_ADD_SEALS,
            fcntl.F_SEAL_WRITE
            | fcntl.F_SEAL_GROW
            | fcntl.F_SEAL_SHRINK
            | fcntl.F_SEAL_SEAL,
        )
        os.lseek(descriptor, 0, os.SEEK_SET)
        probe = subprocess.run(
            [
                bubblewrap_command,
                "--args",
                str(descriptor),
                "--",
                "/bin/true",
            ],
            pass_fds=(descriptor,),
            stdin=subprocess.DEVNULL,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            env={},
            check=False,
        )
    finally:
        os.close(descriptor)
except OSError:
    raise SystemExit(44)

if probe.returncode != 0:
    diagnostic = probe.stderr[:4096].decode("utf-8", errors="replace").lower()
    if "exceeded maximum number of arguments" in diagnostic:
        raise SystemExit(43)
    raise SystemExit(44)
PY
  preflight_status="$?"
  set -e
  case "$preflight_status" in
    0) ;;
    40)
      fail_action \
        "test_attestor_nofile_insufficient" \
        "increase_process_nofile_limit"
      ;;
    41)
      fail_action \
        "test_attestor_repository_head_unavailable" \
        "repair_configured_repository_head"
      ;;
    42)
      fail_action \
        "test_attestor_nofile_probe_failed" \
        "configure_finite_process_nofile_limits"
      ;;
    43)
      fail_action \
        "test_attestor_bubblewrap_arguments_insufficient" \
        "configure_bubblewrap_with_sufficient_argument_capacity"
      ;;
    *)
      fail_action \
        "test_attestor_bubblewrap_probe_failed" \
        "repair_configured_bubblewrap"
      ;;
  esac
}

if [ "$attestor_provider" = "bubblewrap" ]; then
  preflight_bubblewrap_capacity
elif [ "$attestor_provider" = "microvm" ]; then
  probe_microvm_launcher_socket "$launcher_socket"
fi

exec 9<>"$GLOBAL_LEASE_LOCK"
flock -n 9 || fail "profile_in_use"

readonly SOURCE_BINDING="$(
  printf '%s' "$source_codex_home" | sha256sum | awk '{print $1}'
)"
if [ -e "$SOURCE_BINDING_FILE" ]; then
  stored_source_binding="$(read_identity_file "$SOURCE_BINDING_FILE")" ||
    fail "source_binding_invalid"
  [ "$stored_source_binding" = "$SOURCE_BINDING" ] ||
    fail "profile_account_mismatch"
else
  safe_write_value "$SOURCE_BINDING_FILE" "$SOURCE_BINDING"
fi

mv -- "$config_temporary" "$CONFIG_TARGET"

: >"$LOG_OUT"
: >"$LOG_ERR"
chmod 600 -- "$LOG_OUT" "$LOG_ERR"

(
  exec 8>&-
  exec setsid env -i \
    "HOME=$DAEMON_HOME" \
    "CODEX_HOME=$DAEMON_HOME" \
    "PATH=$exec_path" \
    "$binary" serve --config "$CONFIG_TARGET"
) >"$LOG_OUT" 2>"$LOG_ERR" </dev/null &
started_pid="$!"

expected_binary_id="$(stat -Lc '%d:%i' -- "$binary")"
started_ref=""
binary_id=""
for _ in $(seq 1 100); do
  pid_alive "$started_pid" || break
  candidate_start_ref="$(process_start_ref "$started_pid" 2>/dev/null || true)"
  candidate_binary_id="$(
    stat -Lc '%d:%i' -- "/proc/$started_pid/exe" 2>/dev/null || true
  )"
  if [ -n "$candidate_start_ref" ] &&
    [ "$candidate_binary_id" = "$expected_binary_id" ]; then
    started_ref="$candidate_start_ref"
    binary_id="$candidate_binary_id"
    break
  fi
  sleep 0.02
done
[ -n "$started_ref" ] && [ -n "$binary_id" ] || {
  kill -TERM "$started_pid" 2>/dev/null || true
  fail "daemon_start_identity_unavailable"
}
readiness_nonce="$(
  printf '%s\n%s\n%s\n' "$profile" "$started_pid" "$started_ref" |
    sha256sum |
    awk '{print $1}'
)"
readonly READINESS_REQUEST_REF="request:profile-server-readiness:$readiness_nonce"
safe_write_value "$PID_FILE" "$started_pid"
safe_write_value "$START_REF_FILE" "$started_ref"
safe_write_value "$BINARY_ID_FILE" "$binary_id"
safe_write_value "$BINARY_SHA_FILE" "$REQUESTED_BINARY_SHA"
safe_write_value "$CONFIG_SHA_FILE" "$REQUESTED_CONFIG_SHA"

ready=0
microvm_readiness_failure=0
if pid_alive "$started_pid"; then
  if python3 - "$listen" "$token_path" "$project_ref" \
    "$READINESS_REQUEST_REF" <<'PY' >/dev/null 2>&1
import errno
import json
import os
import stat
import sys
import time
import urllib.error
import urllib.request

host_port, token_path, project_ref, request_ref = sys.argv[1:]
metadata = os.stat(token_path, follow_symlinks=False)
if (
    not stat.S_ISREG(metadata.st_mode)
    or stat.S_IMODE(metadata.st_mode) not in (0o400, 0o600)
    or metadata.st_nlink != 1
    or metadata.st_size < 1
    or metadata.st_size > 4096
):
    raise SystemExit(1)
with open(token_path, "rb") as handle:
    token = handle.read().strip()
if not token or b"\r" in token or b"\n" in token:
    raise SystemExit(1)
body = json.dumps(
    {
        "version": "1",
        "request_ref": request_ref,
        "project_ref": project_ref,
        "payload": {},
    },
    separators=(",", ":"),
).encode("utf-8")
request = urllib.request.Request(
    f"http://{host_port}/api/v1/commands/orquesta.system.status",
    data=body,
    method="POST",
    headers={
        "Authorization": "Bearer " + token.decode("utf-8"),
        "Content-Type": "application/json",
    },
)
connect_deadline = time.monotonic() + 5.0
while True:
    try:
        with urllib.request.urlopen(request, timeout=15.0) as response:
            result = json.load(response)
        break
    except urllib.error.URLError as error:
        reason = error.reason
        if (
            isinstance(reason, OSError)
            and reason.errno == errno.ECONNREFUSED
            and time.monotonic() < connect_deadline
        ):
            time.sleep(0.1)
            continue
        raise
data = result.get("data")
if (
    result.get("command_id") != "orquesta.system.status"
    or result.get("command_version") != "1"
    or result.get("request_ref") != request_ref
    or result.get("failure") is not None
    or not isinstance(result.get("audit_ref"), str)
    or not result["audit_ref"]
    or not isinstance(data, dict)
    or any(
        not isinstance(data.get(key), int) or data[key] < 0
        for key in ("goals", "running_goals", "pending_actions", "quarantined_actions")
    )
):
    raise SystemExit(1)
PY
  then
    if [ "$attestor_provider" = "microvm" ]; then
      if run_microvm_launcher_probe "$launcher_socket"; then
        microvm_readiness_failure=0
        ready=1
      else
        microvm_readiness_failure="$?"
      fi
    else
      ready=1
    fi
  fi
fi

if [ "$ready" -ne 1 ]; then
  safe_write_value "$START_FAILED_FILE" "readiness_failed"
  if pid_alive "$started_pid" && daemon_identity_matches "$started_pid"; then
    "$PIDFD_SIGNAL" \
      --pid "$started_pid" \
      --start-ref "$started_ref" \
      --signal SIGTERM >/dev/null 2>&1 || true
  fi
  for _ in $(seq 1 300); do
    pid_alive "$started_pid" || {
      remove_stale_identity
      if [ "$microvm_readiness_failure" -ne 0 ]; then
        fail_microvm_launcher_probe "$microvm_readiness_failure"
      fi
      fail "daemon_not_ready"
    }
    sleep 0.1
  done
  fail "daemon_not_ready_cleanup_pending"
fi

pid_alive "$started_pid" &&
  daemon_identity_matches "$started_pid" ||
  fail "daemon_identity_lost_after_readiness"

printf 'orquesta_profile_server: status=running profile=%s pid=%s\n' \
  "$profile" "$started_pid"
