#!/usr/bin/env bash
set -euo pipefail

# First-cut Firecracker boot probe. It is deliberately not a TestAttestor
# provider: it only proves that a local, pinned Firecracker can boot a verified
# host kernel with a disposable initramfs and no configured network device.

SUPPORTED_VERSION="1.16.1"
MARKER='ORQUESTA_MICROVM_READY_V0 {"schema_version":0,"network_devices":0,"init":"busybox_static_smoke"}'
PATH_SAFE="/usr/sbin:/usr/bin:/sbin:/bin"

CONFIRM_REAL=0
KEEP_TEMP=0
PREFLIGHT_ONLY=0
TIMEOUT_SECONDS=60
TMP_PARENT="/tmp"
FIRECRACKER_BIN="/usr/local/bin/firecracker"
JAILER_BIN="/usr/local/bin/jailer"
BUSYBOX_BIN="/usr/bin/busybox"
KERNEL_RELEASE="$(uname -r)"
KERNEL_BASE_RELEASE="${KERNEL_RELEASE%-generic}"
KERNEL_IMAGE="/boot/vmlinuz-$KERNEL_RELEASE"
EXTRACT_VMLINUX="/usr/src/linux-headers-$KERNEL_BASE_RELEASE/scripts/extract-vmlinux"
LOCAL_VMLINUX=""
CANONICAL_VMLINUX="/srv/orquesta-self/firecracker/vmlinux-$KERNEL_RELEASE"
[[ -e "$CANONICAL_VMLINUX" ]] && LOCAL_VMLINUX="$CANONICAL_VMLINUX"

RUN_DIR=""
RUN_MARKER=""
LAUNCH_PID=""
LAUNCH_PGID=""
CGROUP_DIR=""
CGROUP_PROCS="-"

usage() {
  cat <<'EOF'
Usage:
  smoke_firecracker_boot.sh --preflight-only [options]
  smoke_firecracker_boot.sh --confirm-real [--keep-temp] [options]

Options:
  --firecracker PATH       pinned Firecracker binary (default /usr/local/bin/firecracker)
  --jailer PATH            pinned Jailer binary, identity checked but not invoked
  --busybox PATH           statically linked BusyBox
  --kernel-image PATH      local compressed host kernel
  --extract-vmlinux PATH   local extract-vmlinux tool
  --vmlinux PATH           local uncompressed host vmlinux; skips extraction
                           (canonical auto-detected path: /srv/orquesta-self/firecracker/)
  --tmp-parent PATH        parent for the isolated run directory (default /tmp)
  --timeout SECONDS        1..60, default 60
  --keep-temp              retain the marked run directory for diagnosis

This first cut uses Firecracker directly with --no-api. Jailer is pinned and
verified, but is not invoked because unprivileged jailer composition is a later
provider concern. No network interface, drive, TAP, SSH or cloud-init is used.
The guest rootfs is an initramfs and therefore lives in RAM. This probe has no
Git workspace; a future commit made in tmpfs is not durable until it is exported
to durable storage or pushed.
EOF
}

fail() {
  local code="$1"
  local action="$2"
  printf 'ORQUESTA_FIRECRACKER_PREFLIGHT_ERROR_V0 code=%s action=%s\n' \
    "$code" "$action" >&2
  return 1
}

parse_args() {
  while (($#)); do
    case "$1" in
      --confirm-real) CONFIRM_REAL=1 ;;
      --keep-temp) KEEP_TEMP=1 ;;
      --preflight-only) PREFLIGHT_ONLY=1 ;;
      --firecracker|--jailer|--busybox|--kernel-image|--extract-vmlinux|--vmlinux|--tmp-parent|--timeout)
        (($# >= 2)) || { usage >&2; exit 64; }
        case "$1" in
          --firecracker) FIRECRACKER_BIN="$2" ;;
          --jailer) JAILER_BIN="$2" ;;
          --busybox) BUSYBOX_BIN="$2" ;;
          --kernel-image) KERNEL_IMAGE="$2" ;;
          --extract-vmlinux) EXTRACT_VMLINUX="$2" ;;
          --vmlinux) LOCAL_VMLINUX="$2" ;;
          --tmp-parent) TMP_PARENT="$2" ;;
          --timeout) TIMEOUT_SECONDS="$2" ;;
        esac
        shift
        ;;
      -h|--help) usage; exit 0 ;;
      *) usage >&2; exit 64 ;;
    esac
    shift
  done
  if ((CONFIRM_REAL == 1 && PREFLIGHT_ONLY == 1)); then
    usage >&2
    exit 64
  fi
  if ((CONFIRM_REAL == 0 && PREFLIGHT_ONLY == 0)); then
    fail "real_confirmation_required" "rerun_with_--confirm-real_or_--preflight-only"
    exit 64
  fi
  if [[ ! "$TIMEOUT_SECONDS" =~ ^[0-9]+$ ]] ||
    ((TIMEOUT_SECONDS < 1 || TIMEOUT_SECONDS > 60)); then
    fail "timeout_invalid" "choose_timeout_between_1_and_60_seconds"
    exit 64
  fi
}

resolve_existing() {
  readlink -e -- "$1"
}

validate_safe_input() {
  local requested="$1"
  local kind="$2"
  local resolved owner mode
  [[ "$requested" = /* ]] ||
    fail "${kind}_path_not_absolute" "provide_absolute_${kind}_path" || return
  [[ ! -L "$requested" ]] ||
    fail "${kind}_symlink_rejected" "provide_regular_root_owned_${kind}" || return
  resolved="$(resolve_existing "$requested" 2>/dev/null)" ||
    fail "${kind}_missing" "install_or_provide_local_${kind}" || return
  [[ -f "$resolved" && -r "$resolved" ]] ||
    fail "${kind}_unreadable" "make_local_${kind}_root_owned_and_readable_without_sudo" || return
  owner="$(stat -c '%u' -- "$resolved")"
  mode="$(stat -c '%a' -- "$resolved")"
  [[ "$owner" = "0" ]] ||
    fail "${kind}_owner_unsafe" "use_root_owned_${kind}" || return
  (((8#$mode & 0022) == 0)) ||
    fail "${kind}_mode_unsafe" "remove_group_and_world_write_from_${kind}" || return
  if [[ "$kind" != "kernel" && "$kind" != "vmlinux" ]]; then
    [[ -x "$resolved" ]] ||
      fail "${kind}_not_executable" "make_${kind}_executable" || return
  fi
  printf '%s\n' "$resolved"
}

validate_version() {
  local binary="$1"
  local product="$2"
  local expected="$3"
  local output
  output="$(env -i PATH="$PATH_SAFE" LC_ALL=C "$binary" --version 2>&1)" ||
    fail "${product}_version_probe_failed" "repair_pinned_${product}" || return
  grep -Fqx "${product^} v${expected}" <<<"$output" ||
    fail "${product}_version_unsupported" "install_${product}_v${expected}" || return
}

verify_static_busybox() {
  local binary="$1"
  local description
  description="$(file -L -- "$binary")" ||
    fail "busybox_file_probe_failed" "repair_local_busybox" || return
  grep -Fq "ELF 64-bit" <<<"$description" &&
    grep -Fq "x86-64" <<<"$description" &&
    grep -Fq "statically linked" <<<"$description" ||
    fail "busybox_not_static_x86_64" "install_root_owned_static_x86_64_busybox" || return
  if readelf -l -- "$binary" | grep -q 'INTERP'; then
    fail "busybox_has_dynamic_interpreter" "install_root_owned_static_busybox"
    return
  fi
}

prepare_run_dir() {
  local parent
  [[ "$TMP_PARENT" = /* && -d "$TMP_PARENT" && -w "$TMP_PARENT" ]] ||
    fail "tmp_parent_invalid" "provide_absolute_writable_tmp_parent" || return
  parent="$(readlink -e -- "$TMP_PARENT")" ||
    fail "tmp_parent_unresolvable" "provide_existing_tmp_parent" || return
  [[ "$parent" != "/" ]] ||
    fail "tmp_parent_too_broad" "provide_dedicated_tmp_parent" || return
  RUN_DIR="$(mktemp -d "$parent/orquesta-firecracker-smoke.XXXXXX")"
  chmod 700 "$RUN_DIR"
  RUN_MARKER="$RUN_DIR/.orquesta-firecracker-smoke-owned"
  printf 'schema=orquesta_firecracker_smoke.v0\nuid=%s\n' "$(id -u)" >"$RUN_MARKER"
}

remove_run_dir() {
  local target="$1"
  local marker="$target/.orquesta-firecracker-smoke-owned"
  [[ "$target" = /* && "$target" != "/" &&
    "$(basename -- "$target")" = orquesta-firecracker-smoke.* &&
    -f "$marker" ]] ||
    fail "cleanup_target_unowned" "inspect_and_remove_only_the_marked_smoke_directory" || return
  grep -Fqx 'schema=orquesta_firecracker_smoke.v0' "$marker" ||
    fail "cleanup_marker_invalid" "inspect_marked_smoke_directory" || return
  rm -rf -- "$target"
}

cleanup_process_group() {
  [[ -n "$LAUNCH_PGID" && "$LAUNCH_PGID" =~ ^[0-9]+$ ]] || return 0
  if kill -0 -- "-$LAUNCH_PGID" 2>/dev/null; then
    kill -TERM -- "-$LAUNCH_PGID" 2>/dev/null || true
    for _ in {1..30}; do
      kill -0 -- "-$LAUNCH_PGID" 2>/dev/null || break
      sleep 0.1
    done
  fi
  if kill -0 -- "-$LAUNCH_PGID" 2>/dev/null; then
    kill -KILL -- "-$LAUNCH_PGID" 2>/dev/null || true
  fi
  for _ in {1..30}; do
    kill -0 -- "-$LAUNCH_PGID" 2>/dev/null || return 0
    sleep 0.1
  done
  fail "process_group_leaked" "stop_process_group_${LAUNCH_PGID}_and_investigate"
}

cleanup_cgroup() {
  [[ -n "$CGROUP_DIR" ]] || return 0
  local populated=""
  if [[ -r "$CGROUP_DIR/cgroup.events" ]]; then
    populated="$(awk '$1 == "populated" {print $2}' "$CGROUP_DIR/cgroup.events")"
  fi
  [[ -z "$populated" || "$populated" = "0" ]] ||
    fail "cgroup_still_populated" "stop_owned_firecracker_processes" || return
  rmdir -- "$CGROUP_DIR" 2>/dev/null ||
    fail "cgroup_cleanup_failed" "inspect_owned_firecracker_cgroup"
  CGROUP_DIR=""
  CGROUP_PROCS="-"
}

cleanup() {
  local saved_status=$?
  trap - EXIT INT TERM
  cleanup_process_group || saved_status=1
  cleanup_cgroup || saved_status=1
  if [[ -n "$RUN_DIR" && -d "$RUN_DIR" ]]; then
    if ((KEEP_TEMP == 1)); then
      printf 'ORQUESTA_FIRECRACKER_TEMP_V0 retained=true path=%s\n' "$RUN_DIR" >&2
    else
      remove_run_dir "$RUN_DIR" || saved_status=1
    fi
  fi
  exit "$saved_status"
}

prepare_delegated_cgroup() {
  local relative parent candidate controllers
  [[ -r /sys/fs/cgroup/cgroup.controllers ]] || return 0
  relative="$(awk -F: '$1 == "0" {print $3; exit}' /proc/self/cgroup)"
  [[ -n "$relative" && "$relative" = /* ]] || return 0
  parent="/sys/fs/cgroup$relative"
  [[ -d "$parent" && -w "$parent/cgroup.procs" && -w "$parent/cgroup.subtree_control" ]] ||
    return 0
  controllers="$(cat "$parent/cgroup.subtree_control")"
  [[ " $controllers " = *" cpu "* &&
    " $controllers " = *" memory "* &&
    " $controllers " = *" pids "* ]] || return 0
  candidate="$parent/orquesta-firecracker-${RUN_DIR##*.}"
  mkdir -- "$candidate" 2>/dev/null || return 0
  if [[ ! -w "$candidate/cgroup.procs" ]]; then
    rmdir -- "$candidate" 2>/dev/null || true
    return 0
  fi
  CGROUP_DIR="$candidate"
  CGROUP_PROCS="$candidate/cgroup.procs"
  if ! printf '50000 100000\n' >"$candidate/cpu.max" ||
    ! printf '268435456\n' >"$candidate/memory.max" ||
    ! printf '32\n' >"$candidate/pids.max"; then
    cleanup_cgroup
    return 0
  fi
}

extract_and_verify_vmlinux() {
  local source_tool patched_tool extract_tmp vmlinux="$RUN_DIR/vmlinux"
  if [[ -n "$LOCAL_VMLINUX" ]]; then
    cp -- "$LOCAL_VMLINUX" "$vmlinux"
  else
    source_tool="$EXTRACT_VMLINUX"
    patched_tool="$RUN_DIR/extract-vmlinux"
    extract_tmp="$RUN_DIR/extract-tmp"
    mkdir -m 700 "$extract_tmp"
    # Ubuntu's canonical helper hardcodes /tmp. The disposable copy redirects
    # that one internal scratch file into this smoke's declared temp root.
    # TMPDIR must remain literal in the disposable helper copy.
    # shellcheck disable=SC2016
    sed 's#mktemp /tmp/vmlinux-XXX#mktemp "${TMPDIR:?}/vmlinux-XXX"#' \
      "$source_tool" >"$patched_tool"
    if grep -Fq 'mktemp /tmp/' "$patched_tool"; then
      fail "extract_vmlinux_undeclared_tmp" "provide_helper_honoring_declared_tmpdir"
      return
    fi
    chmod 500 "$patched_tool"
    env -i PATH="$PATH_SAFE" LC_ALL=C TMPDIR="$extract_tmp" \
      "$patched_tool" "$KERNEL_IMAGE" >"$vmlinux"
  fi
  chmod 400 "$vmlinux"
  file -L -- "$vmlinux" | grep -Fq 'ELF 64-bit' &&
    readelf -h -- "$vmlinux" | grep -Fq 'Machine:                           Advanced Micro Devices X86-64' &&
    readelf -l -- "$vmlinux" | grep -q 'LOAD' ||
    fail "vmlinux_verification_failed" "provide_verified_local_x86_64_host_kernel" || return
  printf '%s\n' "$vmlinux"
}

build_initramfs() {
  local rootfs="$RUN_DIR/initramfs-root"
  local initramfs="$RUN_DIR/initramfs.cpio.gz"
  mkdir -m 700 "$rootfs"
  mkdir "$rootfs/bin" "$rootfs/dev" "$rootfs/proc" "$rootfs/sys"
  cp -- "$BUSYBOX_BIN" "$rootfs/bin/busybox"
  chmod 500 "$rootfs/bin/busybox"
  cat >"$rootfs/init" <<EOF
#!/bin/busybox sh
/bin/busybox mount -t proc proc /proc
/bin/busybox mount -t sysfs sysfs /sys
/bin/busybox printf '%s\\n' '$MARKER'
/bin/busybox sync
/bin/busybox reboot -f
/bin/busybox poweroff -f
while true; do /bin/busybox sleep 1; done
EOF
  chmod 500 "$rootfs/init"
  (
    cd "$rootfs"
    find . -print0 | sort -z |
      cpio --null --quiet -o --format=newc --owner=0:0
  ) | gzip -n -9 >"$initramfs"
  chmod 400 "$initramfs"
  printf '%s\n' "$initramfs"
}

create_and_validate_config() {
  local config="$1"
  local vmlinux="$2"
  local initramfs="$3"
  local boot_args='console=ttyS0 reboot=k panic=1 pci=off init=/init'
  env -i PATH="$PATH_SAFE" LC_ALL=C python3 - "$config" "$vmlinux" "$initramfs" "$boot_args" <<'PY'
import json
import os
import sys

config_path, kernel_path, initrd_path, boot_args = sys.argv[1:]
config = {
    "boot-source": {
        "kernel_image_path": kernel_path,
        "initrd_path": initrd_path,
        "boot_args": boot_args,
    },
    "machine-config": {
        "vcpu_count": 1,
        "mem_size_mib": 128,
        "smt": False,
        "track_dirty_pages": False,
    },
    "drives": [],
}
with open(config_path, "x", encoding="utf-8") as stream:
    json.dump(config, stream, sort_keys=True, separators=(",", ":"))
    stream.write("\n")

with open(config_path, encoding="utf-8") as stream:
    observed = json.load(stream)
if observed != config or set(observed) != {"boot-source", "machine-config", "drives"}:
    raise SystemExit("config round-trip mismatch")
if observed["drives"]:
    raise SystemExit("drive configuration forbidden")
if any("network" in key.lower() for key in observed):
    raise SystemExit("network key forbidden")
run_root = os.path.realpath(os.path.dirname(config_path))
for candidate in (kernel_path, initrd_path):
    if os.path.commonpath((run_root, os.path.realpath(candidate))) != run_root:
        raise SystemExit("boot artifact outside run root")
PY
  chmod 400 "$config"
}

launch_microvm() {
  local config="$1"
  local console="$RUN_DIR/console.log"
  local firecracker_log="$RUN_DIR/firecracker.log"
  local cpu_receipt="$RUN_DIR/cpu.txt"
  local instance_id="orquesta-smoke-${RUN_DIR##*.}"
  prepare_delegated_cgroup
  (
    ulimit -c 0
    ulimit -n 256
    # Variables are positional inside the sanitized child shell.
    # shellcheck disable=SC2016
    exec setsid env -i PATH="$PATH_SAFE" LC_ALL=C /bin/sh -c '
      cgroup_procs=$1
      shift
      if [ "$cgroup_procs" != "-" ]; then
        printf "%s\n" "$$" >"$cgroup_procs"
      fi
      exec "$@"
    ' sh "$CGROUP_PROCS" \
      /usr/bin/time -f 'cpu_user_seconds=%U\ncpu_system_seconds=%S\nelapsed_seconds=%e\nmax_rss_kib=%M' \
      -o "$cpu_receipt" \
      timeout --foreground --signal=TERM --kill-after=3s "${TIMEOUT_SECONDS}s" \
      "$FIRECRACKER_BIN" --id "$instance_id" --no-api --config-file "$config"
  ) >"$console" 2>"$firecracker_log" &
  LAUNCH_PID=$!
  LAUNCH_PGID="$LAUNCH_PID"
  local status=0
  wait "$LAUNCH_PID" || status=$?
  LAUNCH_PID=""
  if kill -0 -- "-$LAUNCH_PGID" 2>/dev/null; then
    fail "firecracker_process_leaked" "inspect_owned_firecracker_process_group"
    return
  fi
  LAUNCH_PGID=""
  [[ "$status" = "0" ]] ||
    fail "firecracker_exit_${status}" "inspect_firecracker_console_and_kernel_compatibility" || return
  grep -Fq "$MARKER" "$console" ||
    fail "guest_marker_missing" "inspect_guest_boot_console" || return
  [[ ! -e "$RUN_DIR/firecracker.sock" ]] ||
    fail "api_socket_unexpected" "keep_--no-api_and_remove_socket_configuration" || return
  cat "$cpu_receipt"
}

main() {
  parse_args "$@"
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  [[ "$(uname -m)" = "x86_64" ]] ||
    fail "architecture_unsupported" "run_on_x86_64_host" || exit 1
  [[ "$(id -u)" != "0" ]] ||
    fail "root_execution_rejected" "run_as_unprivileged_kvm_user_without_sudo" || exit 1
  [[ -c /dev/kvm && -r /dev/kvm && -w /dev/kvm ]] ||
    fail "kvm_unavailable" "grant_current_user_read_write_access_to_/dev/kvm" || exit 1

  FIRECRACKER_BIN="$(validate_safe_input "$FIRECRACKER_BIN" firecracker)" || exit 1
  JAILER_BIN="$(validate_safe_input "$JAILER_BIN" jailer)" || exit 1
  BUSYBOX_BIN="$(validate_safe_input "$BUSYBOX_BIN" busybox)" || exit 1
  validate_version "$FIRECRACKER_BIN" firecracker "$SUPPORTED_VERSION" || exit 1
  validate_version "$JAILER_BIN" jailer "$SUPPORTED_VERSION" || exit 1
  verify_static_busybox "$BUSYBOX_BIN" || exit 1

  if [[ -n "$LOCAL_VMLINUX" ]]; then
    LOCAL_VMLINUX="$(validate_safe_input "$LOCAL_VMLINUX" vmlinux)" || exit 1
  else
    KERNEL_IMAGE="$(validate_safe_input "$KERNEL_IMAGE" kernel)" || exit 1
    EXTRACT_VMLINUX="$(validate_safe_input "$EXTRACT_VMLINUX" extract_vmlinux)" || exit 1
  fi
  for tool in python3 file readelf cpio gzip find sort timeout setsid sha256sum cut /usr/bin/time; do
    command -v "$tool" >/dev/null ||
      fail "host_tool_missing" "install_local_${tool##*/}" || exit 1
  done

  prepare_run_dir || exit 1
  local vmlinux initramfs config="$RUN_DIR/firecracker.json"
  vmlinux="$(extract_and_verify_vmlinux)" || exit 1
  initramfs="$(build_initramfs)" || exit 1
  create_and_validate_config "$config" "$vmlinux" "$initramfs"
  printf 'ORQUESTA_FIRECRACKER_PREFLIGHT_V0 status=ready firecracker=%s jailer=%s arch=x86_64 kvm=rw busybox=static kernel=host_vmlinux_verified network=absent api=disabled timeout_seconds=%s\n' \
    "$SUPPORTED_VERSION" "$SUPPORTED_VERSION" "$TIMEOUT_SECONDS"
  printf 'kernel_sha256=%s\ninitramfs_sha256=%s\nconfig_sha256=%s\n' \
    "$(sha256sum "$vmlinux" | cut -d' ' -f1)" \
    "$(sha256sum "$initramfs" | cut -d' ' -f1)" \
    "$(sha256sum "$config" | cut -d' ' -f1)"
  if ((PREFLIGHT_ONLY == 1)); then
    return
  fi
  launch_microvm "$config"
  printf 'ORQUESTA_FIRECRACKER_SMOKE_V0 status=pass marker=ORQUESTA_MICROVM_READY_V0 network=absent api_socket=absent process_leaks=0\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
