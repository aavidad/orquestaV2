#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMOKE="$ROOT/scripts/smoke_firecracker_boot.sh"
TMP="$(mktemp -d /tmp/orquesta-firecracker-smoke-test.XXXXXX)"
trap 'rm -rf -- "$TMP"' EXIT

bash -n "$SMOKE"
bash -n "$0"
# shellcheck source=scripts/smoke_firecracker_boot.sh
source "$SMOKE"

assert_failure_contains() {
  local expected="$1"
  shift
  local output status=0
  output="$("$@" 2>&1)" || status=$?
  [[ "$status" != "0" && "$output" = *"$expected"* ]] || {
    printf 'firecracker_smoke_test_failed expected=%s status=%s output=%s\n' \
      "$expected" "$status" "$output" >&2
    exit 1
  }
}

make_fake_version_tool() {
  local path="$1"
  local product="$2"
  local version="$3"
  cat >"$path" <<EOF
#!/usr/bin/env bash
printf '%s\\n' '$product v$version'
EOF
  chmod 700 "$path"
}

test_confirmation_and_timeout_guards() {
  assert_failure_contains 'code=real_confirmation_required' "$SMOKE"
  assert_failure_contains 'code=timeout_invalid' "$SMOKE" --preflight-only --timeout 61
  # The positional parameter belongs to the child shell.
  # shellcheck disable=SC2016
  assert_failure_contains 'code=architecture_unsupported' \
    bash -c 'source "$1"; uname() { printf "aarch64\n"; }; main --preflight-only' sh "$SMOKE"
}

test_version_guard_with_fakes() {
  make_fake_version_tool "$TMP/firecracker-good" Firecracker "$SUPPORTED_VERSION"
  make_fake_version_tool "$TMP/firecracker-bad" Firecracker 0.0.0
  validate_version "$TMP/firecracker-good" firecracker "$SUPPORTED_VERSION"
  assert_failure_contains 'code=firecracker_version_unsupported' \
    validate_version "$TMP/firecracker-bad" firecracker "$SUPPORTED_VERSION"
}

test_static_guard_with_fakes() {
  local fake_tools="$TMP/static-tools"
  mkdir "$fake_tools"
  cat >"$fake_tools/file" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' 'fake: ELF 64-bit LSB executable, x86-64, statically linked'
EOF
  cat >"$fake_tools/readelf" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' 'Elf file type is EXEC'
EOF
  chmod 700 "$fake_tools/file" "$fake_tools/readelf"
  PATH="$fake_tools:$PATH_SAFE" verify_static_busybox "$TMP/firecracker-good"

  cat >"$fake_tools/file" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' 'fake: ELF 64-bit LSB executable, x86-64, dynamically linked'
EOF
  chmod 700 "$fake_tools/file"
  # The positional parameters belong to the child shell.
  # shellcheck disable=SC2016
  assert_failure_contains 'code=busybox_not_static_x86_64' \
    env PATH="$fake_tools:$PATH_SAFE" bash -c \
      'source "$1"; verify_static_busybox "$2"' sh "$SMOKE" "$TMP/firecracker-good"
}

test_config_has_no_network_surface() {
  local run="$TMP/config-run"
  local config="$run/firecracker.json"
  mkdir "$run"
  touch "$run/vmlinux" "$run/initramfs.cpio.gz"
  create_and_validate_config "$config" "$run/vmlinux" "$run/initramfs.cpio.gz"
  python3 - "$config" <<'PY'
import json
import sys
with open(sys.argv[1], encoding="utf-8") as stream:
    config = json.load(stream)
assert set(config) == {"boot-source", "machine-config", "drives"}
assert "network-interfaces" not in config
assert config["drives"] == []
assert config["machine-config"]["vcpu_count"] == 1
assert config["machine-config"]["mem_size_mib"] == 128
PY
}

test_cleanup_kills_only_owned_group_and_removes_marked_root() {
  RUN_DIR="$(mktemp -d "$TMP/orquesta-firecracker-smoke.XXXXXX")"
  RUN_MARKER="$RUN_DIR/.orquesta-firecracker-smoke-owned"
  printf 'schema=orquesta_firecracker_smoke.v0\nuid=%s\n' "$(id -u)" >"$RUN_MARKER"
  touch "$RUN_DIR/firecracker.sock"
  setsid sleep 30 &
  LAUNCH_PID=$!
  LAUNCH_PGID="$LAUNCH_PID"
  for _ in {1..30}; do
    kill -0 -- "-$LAUNCH_PGID" 2>/dev/null && break
    sleep 0.01
  done
  KEEP_TEMP=0
  CGROUP_DIR=""
  cleanup_process_group
  wait "$LAUNCH_PID" 2>/dev/null || true
  remove_run_dir "$RUN_DIR"
  [[ ! -e "$RUN_DIR" ]] || {
    printf 'firecracker_smoke_test_failed cleanup_left_run_dir\n' >&2
    exit 1
  }
  RUN_DIR=""
  LAUNCH_PID=""
  LAUNCH_PGID=""
}

test_cleanup_rejects_unmarked_root() {
  local unmarked="$TMP/orquesta-firecracker-smoke.unmarked"
  mkdir "$unmarked"
  assert_failure_contains 'code=cleanup_target_unowned' remove_run_dir "$unmarked"
  [[ -d "$unmarked" ]] || {
    printf 'firecracker_smoke_test_failed unmarked_root_removed\n' >&2
    exit 1
  }
}

test_fake_launch_contract() {
  local run="$TMP/fake-launch"
  local config="$run/firecracker.json"
  mkdir "$run"
  cat >"$run/fake-firecracker" <<EOF
#!/usr/bin/env bash
set -euo pipefail
[[ " \$* " = *" --no-api "* ]]
[[ " \$* " != *" --api-sock "* ]]
printf '%s\\n' '$MARKER'
EOF
  chmod 700 "$run/fake-firecracker"
  touch "$run/vmlinux" "$run/initramfs.cpio.gz"
  create_and_validate_config "$config" "$run/vmlinux" "$run/initramfs.cpio.gz"
  RUN_DIR="$run"
  FIRECRACKER_BIN="$run/fake-firecracker"
  TIMEOUT_SECONDS=5
  CGROUP_DIR=""
  CGROUP_PROCS="-"
  LAUNCH_PID=""
  LAUNCH_PGID=""
  output="$(launch_microvm "$config")"
  [[ "$output" = *"cpu_user_seconds="* ]]
  [[ ! -e "$run/firecracker.sock" ]]
  RUN_DIR=""
  LAUNCH_PID=""
  LAUNCH_PGID=""
}

run_fake_suite() {
  test_confirmation_and_timeout_guards
  test_version_guard_with_fakes
  test_static_guard_with_fakes
  test_config_has_no_network_surface
  test_cleanup_kills_only_owned_group_and_removes_marked_root
  test_cleanup_rejects_unmarked_root
  test_fake_launch_contract
  printf 'ORQUESTA_FIRECRACKER_SMOKE_TEST_V0 status=pass mode=fake\n'
}

case "${1:-}" in
  "")
    run_fake_suite
    ;;
  --preflight)
    shift
    run_fake_suite
    "$SMOKE" --preflight-only "$@"
    ;;
  --real)
    [[ "${2:-}" = "--confirm-real" ]] || {
      printf 'real test requires: --real --confirm-real\n' >&2
      exit 64
    }
    shift 2
    run_fake_suite
    "$SMOKE" --confirm-real "$@"
    ;;
  *)
    printf 'Usage: test_smoke_firecracker_boot.sh [--preflight | --real --confirm-real]\n' >&2
    exit 64
    ;;
esac
