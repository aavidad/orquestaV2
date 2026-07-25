#!/usr/bin/env bash
set -euo pipefail

readonly SAFE_PATH="/usr/sbin:/usr/bin:/sbin:/bin"
PATH="$SAFE_PATH"
export PATH

readonly PROFILE="host-128g-16"
readonly FIRECRACKER_VERSION="1.16.1"
readonly INSTALL_PARENT="/usr/local/lib/orquesta"
readonly INSTALL_ROOT="$INSTALL_PARENT/firecracker"
readonly LIBEXEC_ROOT="/usr/local/libexec"
readonly CONFIG_PARENT="/etc/orquesta"
readonly CONFIG_ROOT="$CONFIG_PARENT/firecracker"
readonly UNIT_ROOT="/etc/systemd/system"
readonly RECEIPT_ROOT="$CONFIG_ROOT/receipts"
readonly SYSTEMD_UNIT="/etc/systemd/system/orquesta-firecracker-attestor.service"
readonly TRANSACTION_LOCK_PARENT="/run"
readonly TRANSACTION_LOCK_PATH="$TRANSACTION_LOCK_PARENT/orquesta-firecracker-attestor-installer.lock"
readonly TRANSACTION_LOCK_TIMEOUT_SECONDS=30
readonly RUNTIME_PARENT="/srv/orquesta-self/runtime"
readonly RUNTIME_BACKING_ROOT="$RUNTIME_PARENT/firecracker-attestor"
readonly SOCKET_ROOT="/run/orquesta"
readonly RUNTIME_ROOT="$SOCKET_ROOT"
readonly RUNTIME_MARKER="$RUNTIME_ROOT/.orquesta-firecracker-launcher-root"
readonly RUNTIME_BACKING_MARKER="$RUNTIME_BACKING_ROOT/.orquesta-firecracker-launcher-root"
readonly RUNTIME_MARKER_CONTENT="schema=orquesta_firecracker_launcher_root.v1"
readonly NETNS_NAME="orquesta-firecracker-attestor-empty"
readonly NETNS_PATH="/run/netns/$NETNS_NAME"
readonly CGROUP_ROOT="/sys/fs/cgroup"
readonly PARENT_CGROUP_RELATIVE="orquesta-firecracker-attestor"
readonly PARENT_CGROUP="$CGROUP_ROOT/$PARENT_CGROUP_RELATIVE"
readonly PRIMITIVES_MARKER="$CONFIG_ROOT/primitives.marker"
readonly SOCKET_PATH="$RUNTIME_ROOT/firecracker-launcher.sock"
readonly MAX_SUBJECT_BYTES=536870912
readonly MAX_INPUT_METADATA_BYTES=1048576
readonly MAX_INPUT_BYTES=$((MAX_SUBJECT_BYTES + MAX_INPUT_METADATA_BYTES + 1024))
readonly MAX_OUTPUT_DRIVE_BYTES=4194304
readonly MAX_CAPTURED_OUTPUT_BYTES=67108864
readonly GUEST_MEMORY_MIB=4096
readonly MAX_MEMORY_BYTES=5368709120
readonly MAX_PIDS=512
readonly MAX_CPU_QUOTA_MICROS=200000
readonly MAX_CONCURRENT_RUNS=16
readonly MAX_TIMEOUT="15m"
readonly CLEANUP_TIMEOUT="30s"
readonly MAX_CLEANUP_ENTRIES=4096
readonly MAX_CLEANUP_DEPTH=32
readonly MAX_DIAGNOSTIC_BYTES=1048576
readonly OPERATIONAL_RESERVE_BYTES=2147483648
readonly PER_RUN_DISK_OVERHEAD_BYTES=134217728
readonly MAX_E2E_EVIDENCE_BYTES=16777216

MODE=""
REQUESTED_PROFILE=""
LAUNCHER_SOURCE=""
FIRECRACKER_SOURCE=""
JAILER_SOURCE=""
KERNEL_SOURCE=""
GUEST_SOURCE=""
GUEST_MANIFEST_SOURCE=""
EXPECTED_LAUNCHER_SHA256=""
EXPECTED_FIRECRACKER_SHA256=""
EXPECTED_JAILER_SHA256=""
EXPECTED_KERNEL_SHA256=""
EXPECTED_GUEST_SHA256=""
EXPECTED_GUEST_MANIFEST_SHA256=""
ALLOWED_UID=""
ALLOWED_GID=""
JAIL_UID=""
JAIL_GID=""
E2E_RECEIPT=""
E2E_EVIDENCE=""
WORK_ROOT=""
TRANSACTION_LOCK_FD=""

usage() {
  printf '%s\n' \
    "Uso:" \
    "  $0 (--dry-run|--check|--apply|--activate) \\" \
    "    --profile host-128g-16 \\" \
    "    --launcher-source /ruta/absoluta/orquesta-firecracker-launcher \\" \
    "    --firecracker-source /ruta/absoluta/firecracker \\" \
    "    --jailer-source /ruta/absoluta/jailer \\" \
    "    --kernel-source /ruta/absoluta/vmlinux \\" \
    "    --guest-source /ruta/absoluta/guest.cpio.gz \\" \
    "    --guest-manifest-source /ruta/absoluta/guest.manifest.json \\" \
    "    --launcher-sha256 HEX --firecracker-sha256 HEX --jailer-sha256 HEX \\" \
    "    --kernel-sha256 HEX --guest-sha256 HEX --guest-manifest-sha256 HEX \\" \
    "    --allowed-uid UID --allowed-gid GID --jail-uid UID --jail-gid GID" \
    "" \
    "--activate exige además --e2e-receipt /ruta/absoluta/receipt y" \
    "--e2e-evidence /ruta/absoluta/evidence.json."
}

die() {
  printf 'error=%s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "missing_command:$1"
}

parse_uint32() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[0-9]+$ ]] || die "${label}_not_uint32"
  (( 10#$value <= 4294967295 )) || die "${label}_out_of_range"
}

parse_sha256() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[0-9a-f]{64}$ ]] || die "${label}_not_sha256"
}

parse_arguments() {
  while (($# > 0)); do
    case "$1" in
      --dry-run|--check|--apply|--activate)
        [[ -z "$MODE" ]] || die "multiple_modes"
        MODE="${1#--}"
        shift
        ;;
      --profile|--launcher-source|--firecracker-source|--jailer-source|--kernel-source|--guest-source|--guest-manifest-source|--launcher-sha256|--firecracker-sha256|--jailer-sha256|--kernel-sha256|--guest-sha256|--guest-manifest-sha256|--allowed-uid|--allowed-gid|--jail-uid|--jail-gid|--e2e-receipt|--e2e-evidence)
        (($# >= 2)) || die "missing_value:$1"
        case "$1" in
          --profile) REQUESTED_PROFILE="$2" ;;
          --launcher-source) LAUNCHER_SOURCE="$2" ;;
          --firecracker-source) FIRECRACKER_SOURCE="$2" ;;
          --jailer-source) JAILER_SOURCE="$2" ;;
          --kernel-source) KERNEL_SOURCE="$2" ;;
          --guest-source) GUEST_SOURCE="$2" ;;
          --guest-manifest-source) GUEST_MANIFEST_SOURCE="$2" ;;
          --launcher-sha256) EXPECTED_LAUNCHER_SHA256="$2" ;;
          --firecracker-sha256) EXPECTED_FIRECRACKER_SHA256="$2" ;;
          --jailer-sha256) EXPECTED_JAILER_SHA256="$2" ;;
          --kernel-sha256) EXPECTED_KERNEL_SHA256="$2" ;;
          --guest-sha256) EXPECTED_GUEST_SHA256="$2" ;;
          --guest-manifest-sha256) EXPECTED_GUEST_MANIFEST_SHA256="$2" ;;
          --allowed-uid) ALLOWED_UID="$2" ;;
          --allowed-gid) ALLOWED_GID="$2" ;;
          --jail-uid) JAIL_UID="$2" ;;
          --jail-gid) JAIL_GID="$2" ;;
          --e2e-receipt) E2E_RECEIPT="$2" ;;
          --e2e-evidence) E2E_EVIDENCE="$2" ;;
        esac
        shift 2
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown_argument:$1"
        ;;
    esac
  done

  [[ -n "$MODE" ]] || die "mode_required"
  [[ "$REQUESTED_PROFILE" == "$PROFILE" ]] || die "profile_must_be:$PROFILE"
  local value
  for value in \
    "$LAUNCHER_SOURCE" "$FIRECRACKER_SOURCE" "$JAILER_SOURCE" \
    "$KERNEL_SOURCE" "$GUEST_SOURCE" "$GUEST_MANIFEST_SOURCE"; do
    [[ -n "$value" ]] || die "all_source_flags_required"
  done
  for value in \
    "$EXPECTED_LAUNCHER_SHA256" "$EXPECTED_FIRECRACKER_SHA256" \
    "$EXPECTED_JAILER_SHA256" "$EXPECTED_KERNEL_SHA256" \
    "$EXPECTED_GUEST_SHA256" "$EXPECTED_GUEST_MANIFEST_SHA256"; do
    [[ -n "$value" ]] || die "all_expected_sha256_flags_required"
  done
  parse_sha256 "launcher_sha256" "$EXPECTED_LAUNCHER_SHA256"
  parse_sha256 "firecracker_sha256" "$EXPECTED_FIRECRACKER_SHA256"
  parse_sha256 "jailer_sha256" "$EXPECTED_JAILER_SHA256"
  parse_sha256 "kernel_sha256" "$EXPECTED_KERNEL_SHA256"
  parse_sha256 "guest_sha256" "$EXPECTED_GUEST_SHA256"
  parse_sha256 "guest_manifest_sha256" "$EXPECTED_GUEST_MANIFEST_SHA256"
  for value in "$ALLOWED_UID" "$ALLOWED_GID" "$JAIL_UID" "$JAIL_GID"; do
    [[ -n "$value" ]] || die "all_identity_flags_required"
  done
  parse_uint32 "allowed_uid" "$ALLOWED_UID"
  parse_uint32 "allowed_gid" "$ALLOWED_GID"
  parse_uint32 "jail_uid" "$JAIL_UID"
  parse_uint32 "jail_gid" "$JAIL_GID"
  (( 10#$ALLOWED_UID > 0 && 10#$ALLOWED_GID > 0 )) ||
    die "allowed_identity_must_be_nonroot"
  (( 10#$JAIL_UID > 0 && 10#$JAIL_GID > 0 )) || die "jail_identity_must_be_nonzero"
  (( 10#$ALLOWED_UID < 4294967295 && 10#$ALLOWED_GID < 4294967295 &&
    10#$JAIL_UID < 4294967295 && 10#$JAIL_GID < 4294967295 )) ||
    die "identity_sentinel_forbidden"
  [[ "$ALLOWED_UID" != "$JAIL_UID" ]] || die "allowed_uid_must_differ_from_jail_uid"
  [[ "$ALLOWED_GID" != "$JAIL_GID" ]] || die "allowed_gid_must_differ_from_jail_gid"
  if [[ "$MODE" == "activate" ]]; then
    [[ -n "$E2E_RECEIPT" ]] || die "activate_requires_e2e_receipt"
    [[ -n "$E2E_EVIDENCE" ]] || die "activate_requires_e2e_evidence"
  elif [[ -n "$E2E_RECEIPT" || -n "$E2E_EVIDENCE" ]]; then
    die "e2e_bundle_only_valid_with_activate"
  fi
}

canonical_source() {
  local label="$1"
  local source_path="$2"
  [[ "$source_path" == /* ]] || die "${label}_source_not_absolute"
  [[ -f "$source_path" && ! -L "$source_path" ]] || die "${label}_source_not_regular"
  [[ "$(realpath -e -- "$source_path")" == "$source_path" ]] ||
    die "${label}_source_not_canonical"
  [[ "$(stat -c '%h' -- "$source_path")" == "1" ]] || die "${label}_source_link_count"
  case "$label" in
    launcher|firecracker|jailer)
      [[ -x "$source_path" ]] || die "${label}_source_not_executable"
      ;;
  esac
  local source_mode
  source_mode="$(stat -c '%a' -- "$source_path")"
  (( (8#$source_mode & 0022) == 0 )) || die "${label}_source_writable_by_group_or_world"
}

verify_root_trusted_source() {
  local label="$1"
  local source_path="$2"
  [[ "$(stat -c '%u:%g' -- "$source_path")" == "0:0" ]] ||
    die "${label}_source_not_root_owned"
  local current_path source_mode
  current_path="$(dirname -- "$source_path")"
  while :; do
    [[ -d "$current_path" && ! -L "$current_path" ]] ||
      die "${label}_ancestor_type:$current_path"
    [[ "$(stat -c '%u:%g' -- "$current_path")" == "0:0" ]] ||
      die "${label}_ancestor_not_root_owned:$current_path"
    source_mode="$(stat -c '%a' -- "$current_path")"
    (( (8#$source_mode & 0022) == 0 )) ||
      die "${label}_ancestor_writable:$current_path"
    [[ "$current_path" == "/" ]] && break
    current_path="$(dirname -- "$current_path")"
  done
}

verify_expected_sha256() {
  local label="$1"
  local actual="$2"
  local expected="$3"
  [[ "$actual" == "$expected" ]] || die "${label}_sha256_mismatch"
}

sha256_file() {
  sha256sum -- "$1" | awk '{print $1}'
}

verify_source_stable() {
  local label="$1"
  local source_path="$2"
  local before after
  before="$(stat -c '%d:%i:%s:%Y:%h' -- "$source_path")"
  sha256_file "$source_path" >/dev/null
  after="$(stat -c '%d:%i:%s:%Y:%h' -- "$source_path")"
  [[ "$before" == "$after" ]] || die "${label}_source_changed_during_read"
}

verify_binary_version() {
  local label="$1"
  local source_path="$2"
  local output
  output="$("$source_path" --version 2>&1)" || die "${label}_version_command_failed"
  grep -Eq '(^|[^0-9])v?1[.]16[.]1([^0-9.]|$)' <<<"$output" ||
    die "${label}_version_not_$FIRECRACKER_VERSION"
}

verify_guest_manifest() {
  local manifest_path="$1"
  local guest_sha="$2"
  python3 - "$manifest_path" "$guest_sha" "$GUEST_MEMORY_MIB" <<'PY'
import json
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
guest_sha = sys.argv[2]
configured_guest_memory_mib = int(sys.argv[3])
try:
    raw = path.read_bytes()
    if not 0 < len(raw) <= 64 * 1024:
        raise ValueError("size")
    document = json.loads(raw)
except Exception as exc:
    raise SystemExit(f"guest_manifest_invalid:{type(exc).__name__}") from exc

top_keys = {
    "schema_version", "platform", "source_commit", "runner_sha256",
    "busybox_sha256", "busybox_version", "toolchain_tree_sha256",
    "toolchain_version", "image_sha256",
    "unpacked_bytes", "minimum_guest_memory_mib", "memory_contract", "build",
}
memory_keys = {
    "scratch_fixed_reserve_bytes", "scratch_cache_reserve_bytes",
    "tmpfs_percent", "kernel_runtime_headroom_percent", "formula",
}
build_keys = {
    "cgo_enabled", "trimpath", "buildvcs", "runner_double_build", "source",
    "archive", "owner", "mtime_epoch", "gzip_name_time",
    "toolchain_directories", "toolchain_executables", "toolchain_data",
    "toolchain_symlinks", "toolchain_nobody_go_test",
}
if not isinstance(document, dict) or set(document) != top_keys:
    raise SystemExit("guest_manifest_top_level")
memory = document["memory_contract"]
build = document.get("build")
if not isinstance(memory, dict) or set(memory) != memory_keys:
    raise SystemExit("guest_manifest_memory_contract")
if not isinstance(build, dict) or set(build) != build_keys:
    raise SystemExit("guest_manifest_build")

digest = re.compile(r"sha256:[0-9a-f]{64}\Z")
commit = re.compile(r"(?:[0-9a-f]{40}|[0-9a-f]{64})\Z")
integer = lambda value: isinstance(value, int) and not isinstance(value, bool)
version = re.compile(r"[0-9A-Za-z._+-]+\Z")
valid_version = lambda value, prefix: (
    isinstance(value, str)
    and len(prefix) < len(value) <= 128
    and value.startswith(prefix)
    and version.fullmatch(value) is not None
)
unpacked = document["unpacked_bytes"]
if not integer(unpacked) or unpacked <= 0 or unpacked > 2**64 - 1:
    raise SystemExit("guest_manifest_unpacked_bytes")
required = unpacked + 64 * 1024 * 1024 + 256 * 1024 * 1024
required_mib = (required + 1024 * 1024 - 1) // (1024 * 1024)
minimum_mib = (required_mib * 100 + 75 - 1) // 75
formula = "ceil(ceil((unpacked_bytes+scratch_fixed_reserve_bytes+scratch_cache_reserve_bytes)/MiB)*100/tmpfs_percent)"

valid = (
    document["schema_version"] == "orquesta_test_attestor_guest.v0"
    and document["platform"] == "linux/amd64"
    and isinstance(document["source_commit"], str)
    and commit.fullmatch(document["source_commit"]) is not None
    and all(
        isinstance(document[key], str) and digest.fullmatch(document[key]) is not None
        for key in ("runner_sha256", "busybox_sha256", "toolchain_tree_sha256")
    )
    and valid_version(document["busybox_version"], "v")
    and valid_version(document["toolchain_version"], "go")
    and document["image_sha256"] == f"sha256:{guest_sha}"
    and integer(document["minimum_guest_memory_mib"])
    and document["minimum_guest_memory_mib"] == minimum_mib
    and minimum_mib <= configured_guest_memory_mib
    and memory == {
        "scratch_fixed_reserve_bytes": 64 * 1024 * 1024,
        "scratch_cache_reserve_bytes": 256 * 1024 * 1024,
        "tmpfs_percent": 75,
        "kernel_runtime_headroom_percent": 25,
        "formula": formula,
    }
    and all(
        integer(memory[key])
        for key in (
            "scratch_fixed_reserve_bytes", "scratch_cache_reserve_bytes",
            "tmpfs_percent", "kernel_runtime_headroom_percent",
        )
    )
    and isinstance(memory["formula"], str)
    and build == {
        "cgo_enabled": False,
        "trimpath": True,
        "buildvcs": False,
        "runner_double_build": True,
        "source": "exact_commit_private_export",
        "archive": "newc",
        "owner": "0:0",
        "mtime_epoch": 0,
        "gzip_name_time": False,
        "toolchain_directories": "0555",
        "toolchain_executables": "0555",
        "toolchain_data": "0444",
        "toolchain_symlinks": "relative_internal",
        "toolchain_nobody_go_test": True,
    }
    and build["cgo_enabled"] is False
    and build["trimpath"] is True
    and build["buildvcs"] is False
    and build["runner_double_build"] is True
    and build["gzip_name_time"] is False
    and build["toolchain_nobody_go_test"] is True
    and integer(build["mtime_epoch"])
    and all(
        isinstance(build[key], str)
        for key in (
            "source", "archive", "owner", "toolchain_directories",
            "toolchain_executables", "toolchain_data", "toolchain_symlinks",
        )
    )
)
if not valid:
    raise SystemExit("guest_manifest_contract")
PY
}

render_primitives_marker() {
  printf '%s\n' \
    "schema=orquesta_firecracker_primitives.v1" \
    "runtime_backing=$RUNTIME_BACKING_ROOT" \
    "runtime_mount=$RUNTIME_ROOT" \
    "netns=$NETNS_PATH" \
    "cgroup_parent=$PARENT_CGROUP"
}

render_primitives_helper() {
  local marker_path="$1"
  cat <<EOF
#!/usr/bin/env bash
set -euo pipefail
PATH="$SAFE_PATH"
export PATH
readonly MARKER="$marker_path"
readonly EXPECTED_MARKER='schema=orquesta_firecracker_primitives.v1
runtime_backing=$RUNTIME_BACKING_ROOT
runtime_mount=$RUNTIME_ROOT
netns=$NETNS_PATH
cgroup_parent=$PARENT_CGROUP'
readonly NETNS_NAME="$NETNS_NAME"
readonly NETNS_PATH="$NETNS_PATH"
readonly CGROUP_ROOT="$CGROUP_ROOT"
readonly PARENT_CGROUP="$PARENT_CGROUP"
readonly RUNTIME_ROOT="$RUNTIME_ROOT"
readonly RUNTIME_PARENT="$RUNTIME_PARENT"
readonly RUNTIME_BACKING_ROOT="$RUNTIME_BACKING_ROOT"
readonly RUNTIME_GID="$ALLOWED_GID"
readonly RUNTIME_MARKER="$RUNTIME_MARKER"
readonly RUNTIME_BACKING_MARKER="$RUNTIME_BACKING_MARKER"
readonly RUNTIME_MARKER_CONTENT="$RUNTIME_MARKER_CONTENT"

fail() { printf 'error=primitive_%s\n' "\$1" >&2; exit 1; }
[[ "\$#" == 1 && ( "\$1" == "--ensure" || "\$1" == "--check" ) ]] || fail usage
readonly MODE="\$1"
[[ "\$(id -u)" == 0 ]] || fail root_required
[[ -f "\$MARKER" && ! -L "\$MARKER" ]] || fail marker_type
[[ "\$(stat -c '%u:%g:%a:%h' -- "\$MARKER")" == "0:0:400:1" ]] || fail marker_metadata
[[ "\$(<"\$MARKER")" == "\$EXPECTED_MARKER" ]] || fail marker_content

[[ -d /srv/orquesta-self && ! -L /srv/orquesta-self ]] || fail runtime_base_type
[[ "\$(stat -c '%u:%g:%a' -- /srv/orquesta-self)" == "0:0:755" ]] ||
  fail runtime_base_metadata
if [[ ! -e "\$RUNTIME_PARENT" ]]; then
  [[ "\$MODE" == "--ensure" ]] || fail runtime_root_absent
  mkdir "\$RUNTIME_PARENT"
  chown 0:0 "\$RUNTIME_PARENT"
  chmod 0755 "\$RUNTIME_PARENT"
fi
[[ ! -L "\$RUNTIME_PARENT" && -d "\$RUNTIME_PARENT" ]] || fail runtime_parent_type
[[ "\$(stat -c '%u:%g:%a' -- "\$RUNTIME_PARENT")" == "0:0:755" ]] ||
  fail runtime_parent_metadata
if [[ ! -e "\$RUNTIME_BACKING_ROOT" ]]; then
  [[ "\$MODE" == "--ensure" ]] || fail runtime_backing_absent
  mkdir "\$RUNTIME_BACKING_ROOT"
  chown "0:\$RUNTIME_GID" "\$RUNTIME_BACKING_ROOT"
  chmod 0750 "\$RUNTIME_BACKING_ROOT"
  printf '%s\n' "\$RUNTIME_MARKER_CONTENT" >"\$RUNTIME_BACKING_MARKER"
  chown 0:0 "\$RUNTIME_BACKING_MARKER"
  chmod 0400 "\$RUNTIME_BACKING_MARKER"
fi
[[ ! -L "\$RUNTIME_BACKING_ROOT" && -d "\$RUNTIME_BACKING_ROOT" ]] ||
  fail runtime_backing_type
[[ "\$(stat -c '%u:%g:%a' -- "\$RUNTIME_BACKING_ROOT")" == "0:\$RUNTIME_GID:750" ]] ||
  fail runtime_backing_metadata
[[ -f "\$RUNTIME_BACKING_MARKER" && ! -L "\$RUNTIME_BACKING_MARKER" ]] ||
  fail runtime_marker_type
[[ "\$(stat -c '%u:%g:%a:%h' -- "\$RUNTIME_BACKING_MARKER")" == "0:0:400:1" ]] ||
  fail runtime_marker_metadata
[[ "\$(<"\$RUNTIME_BACKING_MARKER")" == "\$RUNTIME_MARKER_CONTENT" ]] ||
  fail runtime_marker_content
if [[ ! -e "\$RUNTIME_ROOT" ]]; then
  [[ "\$MODE" == "--ensure" ]] || fail runtime_mount_absent
  mkdir "\$RUNTIME_ROOT"
  chown "0:\$RUNTIME_GID" "\$RUNTIME_ROOT"
  chmod 0750 "\$RUNTIME_ROOT"
fi
[[ ! -L "\$RUNTIME_ROOT" && -d "\$RUNTIME_ROOT" ]] || fail runtime_mount_type
if ! mountpoint -q -- "\$RUNTIME_ROOT"; then
  [[ "\$MODE" == "--ensure" ]] || fail runtime_not_mounted
  [[ -z "\$(find "\$RUNTIME_ROOT" -mindepth 1 -maxdepth 1 -print -quit)" ]] ||
    fail runtime_mount_target_not_empty
  mount --bind "\$RUNTIME_BACKING_ROOT" "\$RUNTIME_ROOT"
  mount -o remount,bind,rw,nosuid "\$RUNTIME_ROOT"
fi
[[ "\$(findmnt -rn -M "\$RUNTIME_ROOT" -o TARGET)" == "\$RUNTIME_ROOT" ]] ||
  fail runtime_mount_target
[[ "\$(findmnt -rn -M "\$RUNTIME_ROOT" -o FSROOT)" == "\$RUNTIME_BACKING_ROOT" ]] ||
  fail runtime_mount_source
[[ "\$(stat -c '%d:%i' -- "\$RUNTIME_ROOT")" == "\$(stat -c '%d:%i' -- "\$RUNTIME_BACKING_ROOT")" ]] ||
  fail runtime_mount_identity
runtime_options="\$(findmnt -rn -M "\$RUNTIME_ROOT" -o OPTIONS)"
case ",\$runtime_options," in *,rw,*) ;; *) fail runtime_mount_not_rw ;; esac
case ",\$runtime_options," in *,nosuid,*) ;; *) fail runtime_mount_suid ;; esac
case ",\$runtime_options," in *,noexec,*) fail runtime_mount_noexec ;; esac
case ",\$runtime_options," in *,nodev,*) fail runtime_mount_nodev ;; esac
[[ "\$(stat -c '%u:%g:%a' -- "\$RUNTIME_ROOT")" == "0:\$RUNTIME_GID:750" ]] ||
  fail runtime_root_metadata
[[ -f "\$RUNTIME_MARKER" && ! -L "\$RUNTIME_MARKER" ]] || fail runtime_bound_marker_type
[[ "\$(stat -c '%u:%g:%a:%h' -- "\$RUNTIME_MARKER")" == "0:0:400:1" ]] ||
  fail runtime_bound_marker_metadata
[[ "\$(<"\$RUNTIME_MARKER")" == "\$RUNTIME_MARKER_CONTENT" ]] ||
  fail runtime_bound_marker_content

if [[ ! -e "\$NETNS_PATH" ]]; then
  [[ "\$MODE" == "--ensure" ]] || fail netns_absent
  ip netns add "\$NETNS_NAME"
  chown 0:0 "\$NETNS_PATH"
  chmod 0600 "\$NETNS_PATH"
  ip -n "\$NETNS_NAME" link set dev lo down
  ip -n "\$NETNS_NAME" address flush dev lo
  ip -n "\$NETNS_NAME" route flush table all
  ip -n "\$NETNS_NAME" -6 route flush table all
fi
[[ ! -L "\$NETNS_PATH" && -f "\$NETNS_PATH" ]] || fail netns_type
[[ "\$(stat -c '%u:%g:%a:%h' -- "\$NETNS_PATH")" == "0:0:600:1" ]] || fail netns_metadata
[[ "\$(stat -f -c '%T' -- "\$NETNS_PATH")" == "nsfs" ]] || fail netns_filesystem
[[ "\$(ip -n "\$NETNS_NAME" -o link show | wc -l)" == "1" ]] || fail netns_link_count
ip -n "\$NETNS_NAME" -o link show dev lo | grep -q 'state DOWN' || fail loopback_not_down
if ip -n "\$NETNS_NAME" -o link show dev lo | grep -q '<[^>]*UP'; then fail loopback_up; fi
[[ -z "\$(ip -n "\$NETNS_NAME" -o address show)" ]] || fail netns_addresses
[[ -z "\$(ip -n "\$NETNS_NAME" route show table all)" ]] || fail netns_ipv4_routes
[[ -z "\$(ip -n "\$NETNS_NAME" -6 route show table all)" ]] || fail netns_ipv6_routes

[[ "\$(stat -f -c '%T' -- "\$CGROUP_ROOT")" == "cgroup2fs" ]] || fail cgroup_not_v2
for controller in cpu memory pids; do
  grep -qw "\$controller" "\$CGROUP_ROOT/cgroup.controllers" || fail "controller_missing_\$controller"
  if ! grep -qw "\$controller" "\$CGROUP_ROOT/cgroup.subtree_control"; then
    [[ "\$MODE" == "--ensure" ]] || fail "root_controller_disabled_\$controller"
    printf '+%s\n' "\$controller" >"\$CGROUP_ROOT/cgroup.subtree_control"
  fi
done
if [[ ! -d "\$PARENT_CGROUP" ]]; then
  [[ "\$MODE" == "--ensure" ]] || fail cgroup_parent_absent
  mkdir "\$PARENT_CGROUP"
  chown 0:0 "\$PARENT_CGROUP"
  chmod 0755 "\$PARENT_CGROUP"
fi
[[ ! -L "\$PARENT_CGROUP" && -d "\$PARENT_CGROUP" ]] || fail cgroup_parent_type
[[ "\$(stat -c '%u:%g:%a' -- "\$PARENT_CGROUP")" == "0:0:755" ]] || fail cgroup_parent_metadata
[[ -z "\$(<"\$PARENT_CGROUP/cgroup.procs")" ]] || fail cgroup_parent_has_processes
[[ -z "\$(find "\$PARENT_CGROUP" -mindepth 1 -maxdepth 1 -type d -print -quit)" ]] || fail cgroup_parent_has_children
for controller in cpu memory pids; do
  if ! grep -qw "\$controller" "\$PARENT_CGROUP/cgroup.subtree_control"; then
    [[ "\$MODE" == "--ensure" ]] || fail "parent_controller_disabled_\$controller"
    printf '+%s\n' "\$controller" >"\$PARENT_CGROUP/cgroup.subtree_control"
  fi
done
[[ -e "\$PARENT_CGROUP/memory.swap.max" ]] || fail memory_swap_controller_unavailable
printf 'code=ok\n'
EOF
}

canonical_asset_digest() {
  local firecracker_sha="$1"
  local jailer_sha="$2"
  local kernel_sha="$3"
  local guest_sha="$4"
  local manifest_sha="$5"
  python3 - \
    "$firecracker_sha" "$jailer_sha" "$kernel_sha" "$guest_sha" "$manifest_sha" <<'PY'
import hashlib
import json
import sys

# El orden reproduce exactamente la estructura Go de canonicalAssetDigest.
payload = {
    "schema": "orquesta.firecracker-launcher.assets.v1",
    "firecracker_sha256": sys.argv[1],
    "jailer_sha256": sys.argv[2],
    "kernel_sha256": sys.argv[3],
    "guest_sha256": sys.argv[4],
    "guest_manifest_sha256": sys.argv[5],
}
canonical = json.dumps(
    payload, separators=(",", ":"), ensure_ascii=True
).encode("ascii")
print(hashlib.sha256(canonical).hexdigest())
PY
}

render_config() {
  local firecracker_path="$1"
  local firecracker_sha="$2"
  local jailer_path="$3"
  local jailer_sha="$4"
  local kernel_path="$5"
  local kernel_sha="$6"
  local guest_path="$7"
  local guest_sha="$8"
  local manifest_path="$9"
  local manifest_sha="${10}"
  python3 - \
    "$SOCKET_PATH" "$RUNTIME_ROOT" \
    "$firecracker_path" "$firecracker_sha" "$jailer_path" "$jailer_sha" \
    "$kernel_path" "$kernel_sha" "$guest_path" "$guest_sha" \
    "$manifest_path" "$manifest_sha" "$NETNS_PATH" "$CGROUP_ROOT" "$PARENT_CGROUP_RELATIVE" \
    "$ALLOWED_UID" "$ALLOWED_GID" "$JAIL_UID" "$JAIL_GID" \
    "$MAX_INPUT_BYTES" "$MAX_OUTPUT_DRIVE_BYTES" "$MAX_CAPTURED_OUTPUT_BYTES" \
    "$MAX_MEMORY_BYTES" "$MAX_PIDS" "$MAX_CPU_QUOTA_MICROS" "$MAX_CONCURRENT_RUNS" \
    "$MAX_TIMEOUT" "$CLEANUP_TIMEOUT" "$MAX_CLEANUP_ENTRIES" \
    "$MAX_CLEANUP_DEPTH" "$MAX_DIAGNOSTIC_BYTES" <<'PY'
import json
import sys

keys = [
    "socket_path", "runtime_root",
    "firecracker_command", "firecracker_sha256",
    "jailer_command", "jailer_sha256",
    "kernel_image", "kernel_sha256",
    "guest_image", "guest_sha256",
    "guest_manifest", "guest_manifest_sha256",
    "netns_path", "cgroup_root", "parent_cgroup",
    "allowed_uid", "allowed_gid", "jail_uid", "jail_gid",
    "max_input_bytes", "max_output_drive_bytes", "max_captured_output_bytes",
    "max_memory_bytes", "max_pids", "max_cpu_quota_micros",
    "max_concurrent_runs", "max_timeout", "cleanup_timeout",
    "max_cleanup_entries", "max_cleanup_depth", "max_diagnostic_bytes",
]
values = sys.argv[1:]
integer_keys = {
    "allowed_uid", "allowed_gid", "jail_uid", "jail_gid",
    "max_input_bytes", "max_output_drive_bytes", "max_captured_output_bytes",
    "max_memory_bytes", "max_pids", "max_cpu_quota_micros",
    "max_concurrent_runs", "max_cleanup_entries", "max_cleanup_depth",
    "max_diagnostic_bytes",
}
document = {
    key: int(value) if key in integer_keys else value
    for key, value in zip(keys, values, strict=True)
}
print(json.dumps(document, sort_keys=True, separators=(",", ":"), ensure_ascii=True))
PY
}

render_unit() {
  local helper_path="$1"
  local launcher_path="$2"
  local config_path="$3"
  local primitives_unit_name="$4"
  cat <<EOF
[Unit]
Description=Orquesta Firecracker test attestor launcher
Requires=$primitives_unit_name
After=local-fs.target $primitives_unit_name
ConditionPathExists=/dev/kvm

[Service]
Type=simple
User=0
Group=$ALLOWED_GID
UMask=0077
ExecStartPre=$helper_path --check
ExecStart=$launcher_path --config $config_path
Restart=on-failure
RestartSec=2s
TimeoutStopSec=45s
KillMode=control-group
NoNewPrivileges=yes
CapabilityBoundingSet=CAP_CHOWN CAP_DAC_OVERRIDE CAP_FOWNER CAP_FSETID CAP_KILL CAP_MKNOD CAP_SETGID CAP_SETUID CAP_SYS_ADMIN CAP_SYS_CHROOT CAP_SYS_RESOURCE
AmbientCapabilities=
PrivateTmp=yes
PrivateDevices=no
DevicePolicy=closed
DeviceAllow=/dev/kvm rwm
DeviceAllow=/dev/net/tun rwm
DeviceAllow=/dev/null rw
DeviceAllow=/dev/urandom r
PrivateNetwork=no
IPAddressDeny=any
RestrictAddressFamilies=AF_UNIX AF_NETLINK
PrivateMounts=yes
ProtectSystem=strict
ProtectHome=yes
ProtectControlGroups=no
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectClock=yes
ProtectHostname=yes
RestrictNamespaces=no
RestrictRealtime=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
RemoveIPC=yes
SystemCallArchitectures=native
ReadWritePaths=$RUNTIME_BACKING_ROOT $RUNTIME_ROOT /run/netns $PARENT_CGROUP
ReadOnlyPaths=$CONFIG_ROOT $INSTALL_ROOT $launcher_path $helper_path

[Install]
WantedBy=multi-user.target
EOF
}

render_primitives_unit() {
  local helper_path="$1"
  cat <<EOF
[Unit]
Description=Prepare pinned primitives for Orquesta Firecracker attestor
After=local-fs.target
Before=orquesta-firecracker-attestor.service

[Service]
Type=oneshot
RemainAfterExit=yes
User=0
Group=0
UMask=0077
ExecStart=$helper_path --ensure
NoNewPrivileges=yes
CapabilityBoundingSet=CAP_CHOWN CAP_DAC_OVERRIDE CAP_FOWNER CAP_NET_ADMIN CAP_SYS_ADMIN
AmbientCapabilities=
PrivateDevices=no
PrivateNetwork=no
PrivateMounts=no
PrivateTmp=no
ProtectSystem=no
ProtectHome=no
ProtectControlGroups=no
ProtectKernelTunables=no
RestrictNamespaces=no
RestrictAddressFamilies=AF_UNIX AF_NETLINK
RestrictRealtime=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
RemoveIPC=yes
SystemCallArchitectures=native

[Install]
WantedBy=multi-user.target
EOF
}

validate_config_document() {
  local config_path="$1"
  python3 - "$config_path" <<'PY'
import json
import pathlib
import sys

expected = {
    "socket_path", "runtime_root",
    "firecracker_command", "firecracker_sha256",
    "jailer_command", "jailer_sha256",
    "kernel_image", "kernel_sha256",
    "guest_image", "guest_sha256",
    "guest_manifest", "guest_manifest_sha256",
    "netns_path", "cgroup_root", "parent_cgroup",
    "allowed_uid", "allowed_gid", "jail_uid", "jail_gid",
    "max_input_bytes", "max_output_drive_bytes", "max_captured_output_bytes",
    "max_memory_bytes", "max_pids", "max_cpu_quota_micros",
    "max_concurrent_runs", "max_timeout", "cleanup_timeout",
    "max_cleanup_entries", "max_cleanup_depth", "max_diagnostic_bytes",
}
raw = pathlib.Path(sys.argv[1]).read_bytes()
document = json.loads(raw)
if set(document) != expected:
    raise SystemExit("config_keyset")
canonical = json.dumps(document, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode() + b"\n"
if raw != canonical:
    raise SystemExit("config_not_canonical")
PY
}

require_root() {
  [[ "$(id -u)" == "0" ]] || die "root_required_for_$MODE"
}

verify_lock_parent() {
  local parent="$1"
  local owner="$2"
  local group="$3"
  [[ "$parent" == /* && -d "$parent" && ! -L "$parent" ]] ||
    die "transaction_lock_parent_type"
  [[ "$(realpath -e -- "$parent")" == "$parent" ]] ||
    die "transaction_lock_parent_not_canonical"
  [[ "$(stat -c '%u:%g' -- "$parent")" == "$owner:$group" ]] ||
    die "transaction_lock_parent_owner"
  local parent_mode
  parent_mode="$(stat -c '%a' -- "$parent")"
  (( (8#$parent_mode & 0022) == 0 )) || die "transaction_lock_parent_writable"
}

acquire_transaction_lock() {
  local lock_path="${1:-$TRANSACTION_LOCK_PATH}"
  local parent="${2:-$TRANSACTION_LOCK_PARENT}"
  local owner="${3:-0}"
  local group="${4:-0}"
  local timeout_seconds="${5:-$TRANSACTION_LOCK_TIMEOUT_SECONDS}"
  if [[ ! "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] ||
    (( 10#$timeout_seconds > 300 )); then
    die "transaction_lock_timeout_invalid"
  fi
  [[ "$lock_path" == "$parent/"* && "$(dirname -- "$lock_path")" == "$parent" ]] ||
    die "transaction_lock_path"
  verify_lock_parent "$parent" "$owner" "$group"
  if [[ ! -e "$lock_path" && ! -L "$lock_path" ]]; then
    install -o "$owner" -g "$group" -m 0600 -- /dev/null "$lock_path"
  fi
  [[ -f "$lock_path" && ! -L "$lock_path" ]] || die "transaction_lock_type"
  [[ "$(realpath -e -- "$lock_path")" == "$lock_path" ]] ||
    die "transaction_lock_not_canonical"
  [[ "$(stat -c '%u:%g:%a:%h' -- "$lock_path")" == "$owner:$group:600:1" ]] ||
    die "transaction_lock_metadata"
  local path_identity lock_fd descriptor_identity
  path_identity="$(stat -c '%d:%i' -- "$lock_path")"
  exec {lock_fd}<>"$lock_path" || die "transaction_lock_open"
  descriptor_identity="$(
    stat -Lc '%d:%i:%u:%g:%a:%h' -- "/proc/self/fd/$lock_fd"
  )"
  if [[ "$descriptor_identity" != "$path_identity:$owner:$group:600:1" ]]; then
    exec {lock_fd}>&-
    die "transaction_lock_changed"
  fi
  if ! flock -w "$timeout_seconds" "$lock_fd"; then
    exec {lock_fd}>&-
    die "transaction_lock_timeout"
  fi
  [[ "$(stat -c '%d:%i' -- "$lock_path")" == "$path_identity" ]] || {
    exec {lock_fd}>&-
    die "transaction_lock_replaced"
  }
  TRANSACTION_LOCK_FD="$lock_fd"
}

release_transaction_lock() {
  if [[ -n "$TRANSACTION_LOCK_FD" ]]; then
    exec {TRANSACTION_LOCK_FD}>&-
    TRANSACTION_LOCK_FD=""
  fi
}

verify_directory_exact() {
  local path="$1"
  local owner="$2"
  local group="$3"
  local mode="$4"
  [[ -d "$path" && ! -L "$path" ]] || die "directory_type:$path"
  [[ "$(stat -c '%u:%g:%a' -- "$path")" == "$owner:$group:${mode#0}" ]] ||
    die "directory_metadata:$path"
}

ensure_directory_exact() {
  local path="$1"
  local owner="$2"
  local group="$3"
  local mode="$4"
  if [[ ! -e "$path" ]]; then
    install -d -o "$owner" -g "$group" -m "$mode" -- "$path"
  fi
  verify_directory_exact "$path" "$owner" "$group" "$mode"
}

verify_immutable_file() {
  local path="$1"
  local expected_sha="$2"
  local owner="$3"
  local group="$4"
  local mode="$5"
  [[ -f "$path" && ! -L "$path" ]] || die "immutable_type:$path"
  [[ "$(stat -c '%u:%g:%a:%h' -- "$path")" == "$owner:$group:${mode#0}:1" ]] ||
    die "immutable_metadata:$path"
  [[ "$(sha256_file "$path")" == "$expected_sha" ]] || die "immutable_hash:$path"
}

install_immutable_file() {
  local source_path="$1"
  local destination="$2"
  local expected_sha="$3"
  local owner="$4"
  local group="$5"
  local mode="$6"
  if [[ -e "$destination" || -L "$destination" ]]; then
    verify_immutable_file "$destination" "$expected_sha" "$owner" "$group" "$mode"
    return
  fi
  local temporary="${destination}.new.$$"
  [[ ! -e "$temporary" && ! -L "$temporary" ]] || die "temporary_path_exists:$temporary"
  install -o "$owner" -g "$group" -m "$mode" -- "$source_path" "$temporary"
  verify_immutable_file "$temporary" "$expected_sha" "$owner" "$group" "$mode"
  mv -T -- "$temporary" "$destination"
  verify_immutable_file "$destination" "$expected_sha" "$owner" "$group" "$mode"
}

check_host_capacity() {
  local guest_size="$1"
  local kernel_size="$2"
  local memory_bytes available_bytes required_bytes per_run_bytes
  memory_bytes="$(( $(awk '/^MemTotal:/ {print $2}' /proc/meminfo) * 1024 ))"
  local required_memory_bytes
  required_memory_bytes=$((MAX_MEMORY_BYTES * MAX_CONCURRENT_RUNS + OPERATIONAL_RESERVE_BYTES))
  (( memory_bytes >= required_memory_bytes )) ||
    die "host_memory_insufficient:required=$required_memory_bytes:available=$memory_bytes"
  per_run_bytes=$((MAX_INPUT_BYTES + MAX_OUTPUT_DRIVE_BYTES + guest_size + kernel_size + PER_RUN_DISK_OVERHEAD_BYTES))
  required_bytes=$((per_run_bytes * MAX_CONCURRENT_RUNS))
  required_bytes=$((required_bytes + required_bytes / 10))
  available_bytes="$(df -PB1 "$RUNTIME_BACKING_ROOT" | awk 'NR == 2 {print $4}')"
  (( available_bytes >= required_bytes )) ||
    die "runtime_disk_insufficient:required=$required_bytes:available=$available_bytes"
}

ensure_runtime_root() {
  ensure_directory_exact "/srv/orquesta-self" 0 0 0755
  ensure_directory_exact "$RUNTIME_PARENT" 0 0 0755
  ensure_directory_exact "$RUNTIME_BACKING_ROOT" 0 "$ALLOWED_GID" 0750
  local marker_source="$WORK_ROOT/runtime.marker"
  printf '%s\n' "$RUNTIME_MARKER_CONTENT" >"$marker_source"
  local marker_sha
  marker_sha="$(sha256_file "$marker_source")"
  install_immutable_file "$marker_source" "$RUNTIME_BACKING_MARKER" "$marker_sha" 0 0 0400
}

verify_runtime_root() {
  [[ -d "$RUNTIME_BACKING_ROOT" && ! -L "$RUNTIME_BACKING_ROOT" ]] ||
    die "runtime_backing_type"
  [[ "$(stat -c '%u:%g:%a' -- "$RUNTIME_BACKING_ROOT")" == "0:$ALLOWED_GID:750" ]] ||
    die "runtime_backing_metadata"
  [[ -d "$RUNTIME_ROOT" && ! -L "$RUNTIME_ROOT" ]] || die "runtime_root_type"
  [[ "$(stat -c '%u:%g:%a' -- "$RUNTIME_ROOT")" == "0:$ALLOWED_GID:750" ]] ||
    die "runtime_root_metadata"
  [[ -f "$RUNTIME_MARKER" && ! -L "$RUNTIME_MARKER" ]] || die "runtime_marker_type"
  [[ "$(stat -c '%u:%g:%a:%h' -- "$RUNTIME_MARKER")" == "0:0:400:1" ]] ||
    die "runtime_marker_metadata"
  [[ "$(<"$RUNTIME_MARKER")" == "$RUNTIME_MARKER_CONTENT" ]] || die "runtime_marker_content"
}

validate_receipt() {
  local receipt="$1"
  local config_sha="$2"
  local unit_sha="$3"
  local launcher_sha="$4"
  local primitives_unit_sha="$5"
  local asset_digest="$6"
  [[ "$receipt" == /* && -f "$receipt" && ! -L "$receipt" ]] || die "receipt_type"
  [[ "$(realpath -e -- "$receipt")" == "$receipt" ]] || die "receipt_path_not_canonical"
  [[ "$(stat -c '%u:%g:%a:%h' -- "$receipt")" == "0:0:400:1" ]] || die "receipt_metadata"
  verify_root_trusted_source "receipt" "$receipt"
  if ! python3 - \
    "$receipt" "$config_sha" "$unit_sha" "$primitives_unit_sha" \
    "$launcher_sha" "$asset_digest" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
config_sha, unit_sha, primitives_unit_sha, launcher_sha, asset_digest = sys.argv[2:]
raw = path.read_bytes()
try:
    text = raw.decode("ascii")
except UnicodeDecodeError as exc:
    raise SystemExit("receipt_not_ascii") from exc
lines = text.splitlines(keepends=True)
if len(lines) != 20 or any(not line.endswith("\n") for line in lines):
    raise SystemExit("receipt_line_framing")
values = [line[:-1] for line in lines]
digest = re.compile(r"[0-9a-f]{64}\Z")
for index, prefix in ((7, "evidence_sha256="), (8, "policy_digest=")):
    if not values[index].startswith(prefix):
        raise SystemExit("receipt_digest_field")
    if digest.fullmatch(values[index][len(prefix):]) is None:
        raise SystemExit("receipt_digest_format")
expected = [
    "schema=orquesta_firecracker_activation_receipt.v2",
    "status=passed",
    f"config_sha256={config_sha}",
    f"unit_sha256={unit_sha}",
    f"primitives_unit_sha256={primitives_unit_sha}",
    f"launcher_sha256={launcher_sha}",
    f"asset_digest={asset_digest}",
    values[7],
    values[8],
    "e2e_suite=orquesta.firecracker-attestor.physical-16.v1",
    "max_concurrent_runs=16",
    "physical_microvm_count=16",
    "concurrent_high_water=16",
    "all_attestations_valid=true",
    "zero_residual_runs=true",
    "network_absent=true",
    "api_absent=true",
    "vsock_absent=true",
    "serial_absent=true",
    "memory_swap_max_zero=true",
]
if values != expected or raw != ("\n".join(expected) + "\n").encode("ascii"):
    raise SystemExit("receipt_contract")
PY
  then
    die "receipt_content"
  fi
}

validate_evidence() {
  local evidence="$1"
  local receipt="$2"
  local config_sha="$3"
  local unit_sha="$4"
  local launcher_sha="$5"
  local primitives_unit_sha="$6"
  local asset_digest="$7"
  [[ "$evidence" == /* && -f "$evidence" && ! -L "$evidence" ]] ||
    die "evidence_type"
  [[ "$(realpath -e -- "$evidence")" == "$evidence" ]] ||
    die "evidence_path_not_canonical"
  [[ "$(stat -c '%u:%g:%a:%h' -- "$evidence")" == "0:0:400:1" ]] ||
    die "evidence_metadata"
  verify_root_trusted_source "evidence" "$evidence"
  if ! python3 - \
    "$evidence" "$receipt" "$config_sha" "$unit_sha" "$primitives_unit_sha" \
    "$launcher_sha" "$asset_digest" "$MAX_E2E_EVIDENCE_BYTES" <<'PY'
import datetime
import hashlib
import json
import pathlib
import re
import sys

(
    evidence_path, receipt_path, config_sha, unit_sha, primitives_unit_sha,
    launcher_sha, asset_digest, max_bytes,
) = sys.argv[1:]
raw = pathlib.Path(evidence_path).read_bytes()
if not 0 < len(raw) <= int(max_bytes):
    raise SystemExit("evidence_size")
receipt_raw = pathlib.Path(receipt_path).read_bytes()
receipt_lines = receipt_raw.decode("ascii").splitlines()
if len(receipt_lines) != 20:
    raise SystemExit("receipt_lines")
receipt_values = dict(line.split("=", 1) for line in receipt_lines)
evidence_sha = receipt_values.get("evidence_sha256", "")
policy_digest = receipt_values.get("policy_digest", "")
if hashlib.sha256(raw).hexdigest() != evidence_sha:
    raise SystemExit("evidence_hash")

def strict_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate_key")
        result[key] = value
    return result

def reject_constant(value):
    raise ValueError("nonfinite_number:" + value)

try:
    document = json.loads(
        raw, object_pairs_hook=strict_object, parse_constant=reject_constant,
    )
except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
    raise SystemExit("evidence_json") from exc

digest = re.compile(r"[0-9a-f]{64}\Z")
run_id = re.compile(r"orq-[a-z2-7]{52}\Z")
invocation_id = re.compile(r"[0-9a-f]{32}\Z")
integer = lambda value: isinstance(value, int) and not isinstance(value, bool)
exact_keys = lambda value, keys: isinstance(value, dict) and set(value) == keys

root_keys = {
    "schema", "suite", "status", "started_at", "finished_at", "candidate",
    "policy_digest", "unit", "phase_one", "phase_sixteen", "attestations",
    "cleanup",
}
candidate_keys = {
    "unit_sha256", "primitives_unit_sha256", "launcher_sha256",
    "config_sha256", "supervisor_sha256", "asset_digest",
}
unit_keys = {
    "unit_name", "main_pid", "invocation_id", "active", "fragment_path",
    "loaded", "need_daemon_reload",
}
phase_keys = {
    "requested_runs", "high_water_runs", "samples", "run_ids",
    "firecracker_pids", "limits_exact", "memory_swap_max_zero",
    "network_absent", "api_absent", "vsock_absent", "serial_absent",
    "unit_identity_stable",
}
attestation_keys = {
    "ref", "run_id", "subject_digest", "receipt_ref", "policy_digest", "valid",
}
cleanup_keys = {
    "stable_samples", "residual_runs", "residual_cgroups",
    "residual_processes", "unit_stopped", "socket_absent",
}
if not exact_keys(document, root_keys):
    raise SystemExit("evidence_root")
if (
    document["schema"] != "orquesta.firecracker-attestor.physical-16.evidence.v2"
    or document["suite"] != "orquesta.firecracker-attestor.physical-16.v1"
    or document["status"] != "passed"
    or document["policy_digest"] != policy_digest
    or digest.fullmatch(policy_digest) is None
):
    raise SystemExit("evidence_identity")
for field in ("started_at", "finished_at"):
    value = document[field]
    if not isinstance(value, str) or not value.endswith("Z"):
        raise SystemExit("evidence_time")
    try:
        datetime.datetime.fromisoformat(value[:-1] + "+00:00")
    except ValueError as exc:
        raise SystemExit("evidence_time") from exc

candidate = document["candidate"]
if not exact_keys(candidate, candidate_keys):
    raise SystemExit("evidence_candidate_keys")
expected_candidate = {
    "config_sha256": config_sha,
    "unit_sha256": unit_sha,
    "primitives_unit_sha256": primitives_unit_sha,
    "launcher_sha256": launcher_sha,
    "asset_digest": asset_digest,
}
if any(candidate.get(key) != value for key, value in expected_candidate.items()):
    raise SystemExit("evidence_candidate")
if digest.fullmatch(candidate.get("supervisor_sha256", "")) is None:
    raise SystemExit("evidence_supervisor")

unit = document["unit"]
expected_unit_name = "orquesta-firecracker-attestor-" + unit_sha + ".service"
expected_fragment_path = "/etc/systemd/system/" + expected_unit_name
if (
    not exact_keys(unit, unit_keys)
    or unit["unit_name"] != expected_unit_name
    or not integer(unit["main_pid"]) or unit["main_pid"] <= 0
    or invocation_id.fullmatch(unit["invocation_id"]) is None
    or unit["active"] is not True or unit["loaded"] is not True
    or unit["need_daemon_reload"] is not False
    or unit["fragment_path"] != expected_fragment_path
):
    raise SystemExit("evidence_unit")

def validate_phase(name, expected):
    phase = document[name]
    if (
        not exact_keys(phase, phase_keys)
        or phase["requested_runs"] != expected
        or phase["high_water_runs"] != expected
        or not integer(phase["samples"]) or phase["samples"] <= 0
        or not isinstance(phase["run_ids"], list)
        or len(phase["run_ids"]) != expected
        or not isinstance(phase["firecracker_pids"], list)
        or len(phase["firecracker_pids"]) != expected
        or any(run_id.fullmatch(value) is None for value in phase["run_ids"])
        or len(set(phase["run_ids"])) != expected
        or any(not integer(value) or value <= 0 for value in phase["firecracker_pids"])
        or len(set(phase["firecracker_pids"])) != expected
        or any(
            phase[field] is not True
            for field in (
                "limits_exact", "memory_swap_max_zero", "network_absent",
                "api_absent", "vsock_absent", "serial_absent",
                "unit_identity_stable",
            )
        )
    ):
        raise SystemExit("evidence_" + name)
    return phase["run_ids"]

one_ids = validate_phase("phase_one", 1)
sixteen_ids = validate_phase("phase_sixteen", 16)
all_ids = one_ids + sixteen_ids
if len(set(all_ids)) != 17:
    raise SystemExit("evidence_run_ids")

attestations = document["attestations"]
if not isinstance(attestations, list) or len(attestations) != 17:
    raise SystemExit("evidence_attestations")
refs, outcome_ids, subjects, receipt_refs = [], [], [], []
for outcome in attestations:
    if (
        not exact_keys(outcome, attestation_keys)
        or not isinstance(outcome["ref"], str) or not outcome["ref"]
        or outcome["ref"] != outcome["receipt_ref"]
        or run_id.fullmatch(outcome["run_id"]) is None
        or digest.fullmatch(outcome["subject_digest"]) is None
        or outcome["policy_digest"] != policy_digest
        or outcome["valid"] is not True
    ):
        raise SystemExit("evidence_attestation")
    refs.append(outcome["ref"])
    outcome_ids.append(outcome["run_id"])
    subjects.append(outcome["subject_digest"])
    receipt_refs.append(outcome["receipt_ref"])
if set(outcome_ids) != set(all_ids) or len(set(outcome_ids)) != 17:
    raise SystemExit("evidence_attestation_identity")
by_run = {
    current_run_id: (ref, subject, receipt_ref)
    for current_run_id, ref, subject, receipt_ref
    in zip(outcome_ids, refs, subjects, receipt_refs, strict=True)
}
for phase_ids in (one_ids, sixteen_ids):
    phase_values = [by_run[current_run_id] for current_run_id in phase_ids]
    for position in range(3):
        if len({value[position] for value in phase_values}) != len(phase_ids):
            raise SystemExit("evidence_attestation_phase_identity")

cleanup = document["cleanup"]
if (
    not exact_keys(cleanup, cleanup_keys)
    or not integer(cleanup["stable_samples"]) or cleanup["stable_samples"] < 2
    or any(
        not integer(cleanup[field]) or cleanup[field] != 0
        for field in ("residual_runs", "residual_cgroups", "residual_processes")
    )
    or cleanup["unit_stopped"] is not True
    or cleanup["socket_absent"] is not True
):
    raise SystemExit("evidence_cleanup")
PY
  then
    die "evidence_content"
  fi
}

validate_activation_bundle() {
  local receipt="$1"
  local evidence="$2"
  shift 2
  validate_receipt "$receipt" "$@"
  validate_evidence "$evidence" "$receipt" "$@"
}

read_unit_enabled_state() {
  local unit_name="$1"
  local state
  state="$(systemctl is-enabled "$unit_name" 2>/dev/null || :)"
  case "$state" in
    enabled|disabled) printf '%s\n' "$state" ;;
    *) die "unit_enabled_state_unsupported:$unit_name:$state" ;;
  esac
}

read_unit_active_state() {
  local unit_name="$1"
  local state
  state="$(systemctl is-active "$unit_name" 2>/dev/null || :)"
  case "$state" in
    active|inactive) printf '%s\n' "$state" ;;
    *) die "unit_active_state_unsupported:$unit_name:$state" ;;
  esac
}

restore_unit_enabled_state() {
  local unit_name="$1"
  local expected="$2"
  case "$expected" in
    enabled)
      systemctl enable "$unit_name" &&
        systemctl is-enabled --quiet "$unit_name"
      ;;
    disabled)
      systemctl disable "$unit_name" &&
        ! systemctl is-enabled --quiet "$unit_name"
      ;;
    *) return 1 ;;
  esac
}

restore_unit_active_state() {
  local unit_name="$1"
  local expected="$2"
  case "$expected" in
    active)
      systemctl restart "$unit_name" &&
        systemctl is-active --quiet "$unit_name"
      ;;
    inactive)
      systemctl stop "$unit_name" &&
        ! systemctl is-active --quiet "$unit_name"
      ;;
    *) return 1 ;;
  esac
}

validate_content_addressed_attestor_unit() {
  local unit_path="$1"
  local canonical_unit="$2"
  [[ "$unit_path" == /* && "$canonical_unit" == /* ]] ||
    die "activation_unit_path_not_absolute"
  [[ "$(realpath -e -- "$unit_path")" == "$unit_path" ]] ||
    die "activation_unit_path_not_canonical"
  [[ -f "$unit_path" && ! -L "$unit_path" ]] || die "activation_unit_type"
  [[ "$(dirname -- "$unit_path")" == "$(dirname -- "$canonical_unit")" ]] ||
    die "activation_unit_parent"
  local unit_name expected_sha
  unit_name="$(basename -- "$unit_path")"
  [[ "$unit_name" =~ ^orquesta-firecracker-attestor-([0-9a-f]{64})[.]service$ ]] ||
    die "activation_unit_name"
  expected_sha="${BASH_REMATCH[1]}"
  [[ "$(sha256_file "$unit_path")" == "$expected_sha" ]] ||
    die "activation_unit_digest"
}

activate_unit() {
  local unit_path="$1"
  local canonical_unit="${2:-$SYSTEMD_UNIT}"
  validate_content_addressed_attestor_unit "$unit_path" "$canonical_unit"
  local candidate_name
  candidate_name="$(basename -- "$unit_path")"

  local previous_target="" previous_path="" previous_name=""
  local had_previous="false"
  if [[ -L "$canonical_unit" ]]; then
    previous_target="$(readlink -- "$canonical_unit")"
    [[ "$previous_target" == /* ]] || die "previous_unit_target_not_absolute"
    previous_path="$(realpath -e -- "$canonical_unit")" ||
      die "previous_unit_path_unresolvable"
    [[ "$previous_target" == "$previous_path" ]] ||
      die "previous_unit_target_not_direct"
    validate_content_addressed_attestor_unit "$previous_path" "$canonical_unit"
    previous_name="$(basename -- "$previous_path")"
    had_previous="true"
  elif [[ -e "$canonical_unit" ]]; then
    die "canonical_unit_not_symlink"
  fi

  # No se consulta ni muta systemd antes de cerrar la allowlist exacta de
  # candidato y target previo.
  local candidate_enabled_before candidate_active_before
  local previous_enabled_before="" previous_active_before=""
  candidate_enabled_before="$(read_unit_enabled_state "$candidate_name")"
  candidate_active_before="$(read_unit_active_state "$candidate_name")"
  if [[ "$had_previous" == "true" ]]; then
    if [[ "$previous_name" == "$candidate_name" ]]; then
      previous_enabled_before="$candidate_enabled_before"
      previous_active_before="$candidate_active_before"
    else
      previous_enabled_before="$(read_unit_enabled_state "$previous_name")"
      previous_active_before="$(read_unit_active_state "$previous_name")"
    fi
  fi

  local temporary="${canonical_unit}.new.$$"
  [[ ! -e "$temporary" && ! -L "$temporary" ]] || die "unit_temporary_exists"
  ln -s -- "$unit_path" "$temporary"
  mv -T -- "$temporary" "$canonical_unit"

  # systemctl no permite habilitar un alias enlazado. Se habilita siempre la
  # unidad versionada real y el alias canónico queda solo como punto estable.
  if systemctl daemon-reload &&
    { [[ "$had_previous" != "true" || "$previous_name" == "$candidate_name" ||
      "$previous_active_before" != "active" ]] ||
      systemctl stop "$previous_name"; } &&
    systemctl enable "$candidate_name" &&
    systemctl restart "$candidate_name" &&
    systemctl is-active --quiet "$candidate_name" &&
    { [[ "$had_previous" != "true" || "$previous_name" == "$candidate_name" ||
      "$previous_enabled_before" != "enabled" ]] ||
      systemctl disable "$previous_name"; }; then
    printf '%s\n' \
      "activation=complete" \
      "enabled_unit=$candidate_name" \
      "canonical_unit=$canonical_unit"
    return
  fi

  printf 'activation=failed_rolling_back\n' >&2
  local rollback_ok="true"
  if [[ "$had_previous" == "true" ]]; then
    local rollback="${canonical_unit}.rollback.$$"
    if ! ln -s -- "$previous_target" "$rollback" ||
      ! mv -T -- "$rollback" "$canonical_unit"; then
      rollback_ok="false"
    fi
  elif ! unlink -- "$canonical_unit"; then
    rollback_ok="false"
  fi
  systemctl daemon-reload || rollback_ok="false"

  if [[ "$had_previous" != "true" || "$previous_name" != "$candidate_name" ]]; then
    restore_unit_enabled_state "$candidate_name" "$candidate_enabled_before" ||
      rollback_ok="false"
    restore_unit_active_state "$candidate_name" "$candidate_active_before" ||
      rollback_ok="false"
  fi
  if [[ "$had_previous" == "true" ]]; then
    restore_unit_enabled_state "$previous_name" "$previous_enabled_before" ||
      rollback_ok="false"
    restore_unit_active_state "$previous_name" "$previous_active_before" ||
      rollback_ok="false"
  fi
  [[ "$rollback_ok" == "true" ]] || die "activation_failed_rollback_incomplete"
  die "activation_failed"
}

make_work_root() {
  WORK_ROOT="$(mktemp -d "/tmp/orquesta-firecracker-install.XXXXXXXX")"
}

cleanup_work_root() {
  if [[ -n "$WORK_ROOT" && -d "$WORK_ROOT" ]]; then
    rm -rf -- "$WORK_ROOT"
  fi
  release_transaction_lock
}

main() {
  parse_arguments "$@"
  require_command sha256sum
  require_command stat
  require_command python3
  require_command awk
  require_command install
  require_command mv
  require_command find
  require_command grep
  require_command realpath

  local label path
  while IFS=: read -r label path; do
    canonical_source "$label" "$path"
    verify_source_stable "$label" "$path"
  done <<EOF
launcher:$LAUNCHER_SOURCE
firecracker:$FIRECRACKER_SOURCE
jailer:$JAILER_SOURCE
kernel:$KERNEL_SOURCE
guest:$GUEST_SOURCE
guest_manifest:$GUEST_MANIFEST_SOURCE
EOF

  local launcher_sha firecracker_sha jailer_sha kernel_sha guest_sha manifest_sha
  launcher_sha="$(sha256_file "$LAUNCHER_SOURCE")"
  firecracker_sha="$(sha256_file "$FIRECRACKER_SOURCE")"
  jailer_sha="$(sha256_file "$JAILER_SOURCE")"
  kernel_sha="$(sha256_file "$KERNEL_SOURCE")"
  guest_sha="$(sha256_file "$GUEST_SOURCE")"
  manifest_sha="$(sha256_file "$GUEST_MANIFEST_SOURCE")"
  verify_expected_sha256 "launcher" "$launcher_sha" "$EXPECTED_LAUNCHER_SHA256"
  verify_expected_sha256 "firecracker" "$firecracker_sha" "$EXPECTED_FIRECRACKER_SHA256"
  verify_expected_sha256 "jailer" "$jailer_sha" "$EXPECTED_JAILER_SHA256"
  verify_expected_sha256 "kernel" "$kernel_sha" "$EXPECTED_KERNEL_SHA256"
  verify_expected_sha256 "guest" "$guest_sha" "$EXPECTED_GUEST_SHA256"
  verify_expected_sha256 \
    "guest_manifest" "$manifest_sha" "$EXPECTED_GUEST_MANIFEST_SHA256"
  local asset_digest
  asset_digest="$(
    canonical_asset_digest \
      "$firecracker_sha" "$jailer_sha" "$kernel_sha" "$guest_sha" "$manifest_sha"
  )"
  parse_sha256 "asset_digest" "$asset_digest"
  if [[ "$(id -u)" == "0" ]]; then
    while IFS=: read -r label path; do
      verify_root_trusted_source "$label" "$path"
    done <<EOF
launcher:$LAUNCHER_SOURCE
firecracker:$FIRECRACKER_SOURCE
jailer:$JAILER_SOURCE
kernel:$KERNEL_SOURCE
guest:$GUEST_SOURCE
guest_manifest:$GUEST_MANIFEST_SOURCE
EOF
  fi
  verify_binary_version "firecracker" "$FIRECRACKER_SOURCE"
  verify_binary_version "jailer" "$JAILER_SOURCE"
  verify_guest_manifest "$GUEST_MANIFEST_SOURCE" "$guest_sha"

  local launcher_path firecracker_path jailer_path kernel_path guest_path manifest_path
  launcher_path="$LIBEXEC_ROOT/orquesta-firecracker-launcher-$launcher_sha"
  firecracker_path="$INSTALL_ROOT/firecracker-$firecracker_sha"
  jailer_path="$INSTALL_ROOT/jailer-$jailer_sha"
  kernel_path="$INSTALL_ROOT/vmlinux-$kernel_sha"
  guest_path="$INSTALL_ROOT/guest-$guest_sha.cpio.gz"
  manifest_path="$INSTALL_ROOT/guest-manifest-$manifest_sha.json"

  make_work_root
  trap cleanup_work_root EXIT
  local marker_file helper_file config_file primitives_unit_file unit_file
  marker_file="$WORK_ROOT/primitives.marker"
  helper_file="$WORK_ROOT/primitives-helper"
  config_file="$WORK_ROOT/launcher.json"
  primitives_unit_file="$WORK_ROOT/orquesta-firecracker-primitives.service"
  unit_file="$WORK_ROOT/orquesta-firecracker-attestor.service"
  render_primitives_marker >"$marker_file"
  render_primitives_helper "$PRIMITIVES_MARKER" >"$helper_file"
  chmod 0755 "$helper_file"
  local helper_sha helper_path
  helper_sha="$(sha256_file "$helper_file")"
  helper_path="$LIBEXEC_ROOT/orquesta-firecracker-primitives-$helper_sha"
  render_config \
    "$firecracker_path" "$firecracker_sha" "$jailer_path" "$jailer_sha" \
    "$kernel_path" "$kernel_sha" "$guest_path" "$guest_sha" \
    "$manifest_path" "$manifest_sha" >"$config_file"
  validate_config_document "$config_file"
  local config_sha config_path
  config_sha="$(sha256_file "$config_file")"
  config_path="$CONFIG_ROOT/launcher-$config_sha.json"
  render_primitives_unit "$helper_path" >"$primitives_unit_file"
  local primitives_unit_sha primitives_unit_name primitives_unit_path
  primitives_unit_sha="$(sha256_file "$primitives_unit_file")"
  primitives_unit_name="orquesta-firecracker-primitives-$primitives_unit_sha.service"
  primitives_unit_path="$UNIT_ROOT/$primitives_unit_name"
  render_unit \
    "$helper_path" "$launcher_path" "$config_path" "$primitives_unit_name" \
    >"$unit_file"
  local unit_sha unit_path
  unit_sha="$(sha256_file "$unit_file")"
  unit_path="$UNIT_ROOT/orquesta-firecracker-attestor-$unit_sha.service"

  if [[ "$MODE" == "dry-run" ]]; then
    printf '%s\n' \
      "profile=$PROFILE" \
      "max_subject_bytes=$MAX_SUBJECT_BYTES" \
      "guest_memory_mib=$GUEST_MEMORY_MIB" \
      "operational_reserve_bytes=$OPERATIONAL_RESERVE_BYTES" \
      "microvm_count=$MAX_CONCURRENT_RUNS" \
      "per_microvm_memory_bytes=$MAX_MEMORY_BYTES" \
      "aggregate_microvm_memory_bytes=$((MAX_MEMORY_BYTES * MAX_CONCURRENT_RUNS))" \
      "expected_asset_digest=$asset_digest" \
      "orquesta_config_key=test_attestor.microvm.expected_asset_digest" \
      "launcher_path=$launcher_path" \
      "config_path=$config_path" \
      "primitives_unit_path=$primitives_unit_path" \
      "unit_path=$unit_path" \
      "activation=not_performed"
    printf '%s\n' "-----BEGIN CONFIG JSON-----"
    command cat "$config_file"
    printf '%s\n' "-----END CONFIG JSON-----" "-----BEGIN PRIMITIVES HELPER-----"
    command cat "$helper_file"
    printf '%s\n' "-----END PRIMITIVES HELPER-----" "-----BEGIN PRIMITIVES SYSTEMD UNIT-----"
    command cat "$primitives_unit_file"
    printf '%s\n' "-----END PRIMITIVES SYSTEMD UNIT-----" "-----BEGIN SYSTEMD UNIT-----"
    command cat "$unit_file"
    printf '%s\n' "-----END SYSTEMD UNIT-----"
    exit 0
  fi

  require_root
  require_command flock
  acquire_transaction_lock
  require_command ip
  require_command df
  require_command findmnt
  require_command mount
  require_command mountpoint
  require_command systemctl
  require_command uname
  [[ "$(uname -m)" == "x86_64" ]] || die "host_architecture_must_be_x86_64"
  [[ -c /dev/kvm && ! -L /dev/kvm ]] || die "kvm_device_unavailable"

  local marker_sha
  marker_sha="$(sha256_file "$marker_file")"
  if [[ "$MODE" == "apply" ]]; then
    ensure_directory_exact "$LIBEXEC_ROOT" 0 0 0755
    ensure_directory_exact "$INSTALL_PARENT" 0 0 0755
    ensure_directory_exact "$INSTALL_ROOT" 0 0 0755
    ensure_directory_exact "$CONFIG_PARENT" 0 0 0755
    ensure_directory_exact "$CONFIG_ROOT" 0 0 0755
    ensure_directory_exact "$UNIT_ROOT" 0 0 0755
    ensure_directory_exact "$RECEIPT_ROOT" 0 0 0700
    ensure_runtime_root
    check_host_capacity "$(stat -c '%s' -- "$GUEST_SOURCE")" "$(stat -c '%s' -- "$KERNEL_SOURCE")"
    install_immutable_file "$LAUNCHER_SOURCE" "$launcher_path" "$launcher_sha" 0 0 0755
    install_immutable_file "$FIRECRACKER_SOURCE" "$firecracker_path" "$firecracker_sha" 0 0 0755
    install_immutable_file "$JAILER_SOURCE" "$jailer_path" "$jailer_sha" 0 0 0755
    install_immutable_file "$KERNEL_SOURCE" "$kernel_path" "$kernel_sha" 0 0 0444
    install_immutable_file "$GUEST_SOURCE" "$guest_path" "$guest_sha" 0 0 0444
    install_immutable_file "$GUEST_MANIFEST_SOURCE" "$manifest_path" "$manifest_sha" 0 0 0444
    install_immutable_file "$marker_file" "$PRIMITIVES_MARKER" "$marker_sha" 0 0 0400
    install_immutable_file "$helper_file" "$helper_path" "$helper_sha" 0 0 0755
    install_immutable_file "$config_file" "$config_path" "$config_sha" 0 0 0400
    install_immutable_file "$primitives_unit_file" "$primitives_unit_path" "$primitives_unit_sha" 0 0 0444
    install_immutable_file "$unit_file" "$unit_path" "$unit_sha" 0 0 0444
    "$helper_path" --ensure
    printf '%s\n' \
      "apply=staged" \
      "expected_asset_digest=$asset_digest" \
      "config_path=$config_path" \
      "primitives_unit_path=$primitives_unit_path" \
      "unit_path=$unit_path" \
      "activation=not_performed" \
      "next=e2e_then_activate"
    exit 0
  fi

  verify_directory_exact "$LIBEXEC_ROOT" 0 0 0755
  verify_directory_exact "$INSTALL_PARENT" 0 0 0755
  verify_directory_exact "$INSTALL_ROOT" 0 0 0755
  verify_directory_exact "$CONFIG_PARENT" 0 0 0755
  verify_directory_exact "$CONFIG_ROOT" 0 0 0755
  verify_directory_exact "$UNIT_ROOT" 0 0 0755
  verify_directory_exact "$RECEIPT_ROOT" 0 0 0700
  verify_directory_exact "/srv/orquesta-self" 0 0 0755
  verify_directory_exact "$RUNTIME_PARENT" 0 0 0755
  verify_runtime_root
  verify_immutable_file "$LAUNCHER_SOURCE" "$launcher_sha" "$(stat -c '%u' "$LAUNCHER_SOURCE")" "$(stat -c '%g' "$LAUNCHER_SOURCE")" "$(stat -c '%a' "$LAUNCHER_SOURCE")"
  verify_immutable_file "$launcher_path" "$launcher_sha" 0 0 0755
  verify_immutable_file "$firecracker_path" "$firecracker_sha" 0 0 0755
  verify_immutable_file "$jailer_path" "$jailer_sha" 0 0 0755
  verify_immutable_file "$kernel_path" "$kernel_sha" 0 0 0444
  verify_immutable_file "$guest_path" "$guest_sha" 0 0 0444
  verify_immutable_file "$manifest_path" "$manifest_sha" 0 0 0444
  verify_immutable_file "$PRIMITIVES_MARKER" "$marker_sha" 0 0 0400
  verify_immutable_file "$helper_path" "$helper_sha" 0 0 0755
  verify_immutable_file "$config_path" "$config_sha" 0 0 0400
  verify_immutable_file "$primitives_unit_path" "$primitives_unit_sha" 0 0 0444
  verify_immutable_file "$unit_path" "$unit_sha" 0 0 0444
  "$helper_path" --check
  check_host_capacity \
    "$(stat -c '%s' -- "$GUEST_SOURCE")" \
    "$(stat -c '%s' -- "$KERNEL_SOURCE")"

  if [[ "$MODE" == "check" ]]; then
    printf '%s\n' \
      "check=ok" \
      "expected_asset_digest=$asset_digest" \
      "config_path=$config_path" \
      "unit_path=$unit_path"
    exit 0
  fi

  validate_activation_bundle \
    "$E2E_RECEIPT" "$E2E_EVIDENCE" "$config_sha" "$unit_sha" "$launcher_sha" \
    "$primitives_unit_sha" "$asset_digest"
  local evidence_sha
  evidence_sha="$(sha256_file "$E2E_EVIDENCE")"
  local installed_evidence="$RECEIPT_ROOT/$evidence_sha.evidence.json"
  local installed_receipt="$RECEIPT_ROOT/$unit_sha.receipt"
  install_immutable_file \
    "$E2E_EVIDENCE" "$installed_evidence" "$evidence_sha" 0 0 0400
  install_immutable_file "$E2E_RECEIPT" "$installed_receipt" "$(sha256_file "$E2E_RECEIPT")" 0 0 0400
  validate_activation_bundle \
    "$installed_receipt" "$installed_evidence" "$config_sha" "$unit_sha" "$launcher_sha" \
    "$primitives_unit_sha" "$asset_digest"
  activate_unit "$unit_path"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
