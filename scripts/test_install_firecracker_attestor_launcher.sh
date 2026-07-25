#!/usr/bin/env bash
set -euo pipefail

TEST_SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_SCRIPT_DIR
readonly INSTALLER="$TEST_SCRIPT_DIR/install_firecracker_attestor_launcher.sh"
SHELLCHECK_COMMAND="$(command -v shellcheck)"
readonly SHELLCHECK_COMMAND
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-firecracker-installer-test.XXXXXXXX")"
readonly TEST_ROOT
trap 'rm -rf -- "$TEST_ROOT"' EXIT

fail() {
  printf 'installer_test_failed=%s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local path="$1"
  local expected="$2"
  grep -Fq -- "$expected" "$path" || fail "missing:$expected"
}

assert_not_contains() {
  local path="$1"
  local forbidden="$2"
  if grep -Fq -- "$forbidden" "$path"; then
    fail "forbidden:$forbidden"
  fi
}

assert_line() {
  local path="$1"
  local expected="$2"
  grep -Fxq -- "$expected" "$path" || fail "missing_line:$expected"
}

write_fake_executable() {
  local path="$1"
  local product="$2"
  printf '%s\n' \
    '#!/usr/bin/env sh' \
    "if [ \"\${1:-}\" = \"--version\" ]; then printf '%s v1.16.1\\n' '$product'; exit 0; fi" \
    'exit 0' >"$path"
  chmod 0700 "$path"
}

readonly LAUNCHER="$TEST_ROOT/orquesta-firecracker-launcher"
readonly SUPERVISOR="$TEST_ROOT/orquesta-firecracker-attestor-e2e"
readonly FIRECRACKER="$TEST_ROOT/firecracker"
readonly JAILER="$TEST_ROOT/jailer"
readonly KERNEL="$TEST_ROOT/vmlinux"
readonly GUEST="$TEST_ROOT/guest.cpio.gz"
readonly MANIFEST="$TEST_ROOT/guest.manifest.json"
readonly OUTPUT="$TEST_ROOT/dry-run.txt"
readonly CONFIG="$TEST_ROOT/rendered-config.json"
readonly HELPER="$TEST_ROOT/rendered-helper.sh"
readonly PRIMITIVES_UNIT="$TEST_ROOT/rendered-primitives-unit.service"
readonly UNIT="$TEST_ROOT/rendered-unit.service"

write_fake_executable "$LAUNCHER" "orquesta-firecracker-launcher"
write_fake_executable "$SUPERVISOR" "orquesta-firecracker-attestor-e2e"
write_fake_executable "$FIRECRACKER" "Firecracker"
write_fake_executable "$JAILER" "jailer"
printf 'kernel-fixture\n' >"$KERNEL"
printf 'guest-fixture\n' >"$GUEST"
chmod 0600 "$KERNEL" "$GUEST"
guest_sha="$(sha256sum "$GUEST" | awk '{print $1}')"
printf '%s\n' \
  "{\"schema_version\":\"orquesta_test_attestor_guest.v0\",\"platform\":\"linux/amd64\",\"source_commit\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"runner_sha256\":\"sha256:1111111111111111111111111111111111111111111111111111111111111111\",\"busybox_sha256\":\"sha256:2222222222222222222222222222222222222222222222222222222222222222\",\"busybox_version\":\"v1.36.1\",\"toolchain_tree_sha256\":\"sha256:3333333333333333333333333333333333333333333333333333333333333333\",\"toolchain_version\":\"go1.24.5\",\"image_sha256\":\"sha256:$guest_sha\",\"unpacked_bytes\":33554432,\"minimum_guest_memory_mib\":470,\"memory_contract\":{\"scratch_fixed_reserve_bytes\":67108864,\"scratch_cache_reserve_bytes\":268435456,\"tmpfs_percent\":75,\"kernel_runtime_headroom_percent\":25,\"formula\":\"ceil(ceil((unpacked_bytes+scratch_fixed_reserve_bytes+scratch_cache_reserve_bytes)/MiB)*100/tmpfs_percent)\"},\"build\":{\"cgo_enabled\":false,\"trimpath\":true,\"buildvcs\":false,\"runner_double_build\":true,\"source\":\"exact_commit_private_export\",\"archive\":\"newc\",\"owner\":\"0:0\",\"mtime_epoch\":0,\"gzip_name_time\":false,\"toolchain_directories\":\"0555\",\"toolchain_executables\":\"0555\",\"toolchain_data\":\"0444\",\"toolchain_symlinks\":\"relative_internal\",\"toolchain_nobody_go_test\":true}}" \
  >"$MANIFEST"
chmod 0600 "$MANIFEST"
launcher_sha="$(sha256sum "$LAUNCHER" | awk '{print $1}')"
supervisor_sha="$(sha256sum "$SUPERVISOR" | awk '{print $1}')"
firecracker_sha="$(sha256sum "$FIRECRACKER" | awk '{print $1}')"
jailer_sha="$(sha256sum "$JAILER" | awk '{print $1}')"
kernel_sha="$(sha256sum "$KERNEL" | awk '{print $1}')"
manifest_sha="$(sha256sum "$MANIFEST" | awk '{print $1}')"

readonly -a BASE_ARGUMENTS=(
  --profile host-128g-16
  --launcher-source "$LAUNCHER"
  --supervisor-source "$SUPERVISOR"
  --firecracker-source "$FIRECRACKER"
  --jailer-source "$JAILER"
  --kernel-source "$KERNEL"
  --guest-source "$GUEST"
  --guest-manifest-source "$MANIFEST"
  --launcher-sha256 "$launcher_sha"
  --supervisor-sha256 "$supervisor_sha"
  --firecracker-sha256 "$firecracker_sha"
  --jailer-sha256 "$jailer_sha"
  --kernel-sha256 "$kernel_sha"
  --guest-sha256 "$guest_sha"
  --guest-manifest-sha256 "$manifest_sha"
  --allowed-uid 1000
  --allowed-gid 1000
  --jail-uid 65534
  --jail-gid 65534
)

run_with_argument_pair() {
  local target_flag="$1"
  local replacement_state="$2"
  local replacement_value="${3:-}"
  local -a arguments=()
  local skip_value="false"
  local current
  for current in "${BASE_ARGUMENTS[@]}"; do
    if [[ "$skip_value" == "true" ]]; then
      skip_value="false"
      continue
    fi
    if [[ "$current" == "$target_flag" ]]; then
      skip_value="true"
      if [[ "$replacement_state" == "replace" ]]; then
        arguments+=("$target_flag" "$replacement_value")
      fi
      continue
    fi
    arguments+=("$current")
  done
  "$INSTALLER" --dry-run "${arguments[@]}"
}

"$INSTALLER" --dry-run "${BASE_ARGUMENTS[@]}" >"$OUTPUT"

if run_with_argument_pair --supervisor-source remove \
  >"$TEST_ROOT/missing-supervisor-source.out" 2>&1; then
  fail "missing_supervisor_source_accepted"
fi
assert_contains \
  "$TEST_ROOT/missing-supervisor-source.out" \
  "error=all_source_flags_required"

if run_with_argument_pair --supervisor-sha256 remove \
  >"$TEST_ROOT/missing-supervisor-sha.out" 2>&1; then
  fail "missing_supervisor_sha_accepted"
fi
assert_contains \
  "$TEST_ROOT/missing-supervisor-sha.out" \
  "error=all_expected_sha256_flags_required"

if run_with_argument_pair \
  --supervisor-sha256 replace \
  0000000000000000000000000000000000000000000000000000000000000000 \
  >"$TEST_ROOT/supervisor-hash-mismatch.out" 2>&1; then
  fail "wrong_supervisor_hash_accepted"
fi
assert_contains \
  "$TEST_ROOT/supervisor-hash-mismatch.out" \
  "error=supervisor_sha256_mismatch"

readonly TAMPERED_SUPERVISOR="$TEST_ROOT/orquesta-firecracker-attestor-e2e-tampered"
cp -- "$SUPERVISOR" "$TAMPERED_SUPERVISOR"
printf '%s\n' "# tamper" >>"$TAMPERED_SUPERVISOR"
chmod 0700 "$TAMPERED_SUPERVISOR"
if run_with_argument_pair \
  --supervisor-source replace "$TAMPERED_SUPERVISOR" \
  >"$TEST_ROOT/supervisor-tamper.out" 2>&1; then
  fail "tampered_supervisor_accepted"
fi
assert_contains \
  "$TEST_ROOT/supervisor-tamper.out" \
  "error=supervisor_sha256_mismatch"

if run_with_argument_pair --supervisor-source replace relative-supervisor \
  >"$TEST_ROOT/supervisor-relative-path.out" 2>&1; then
  fail "relative_supervisor_path_accepted"
fi
assert_contains \
  "$TEST_ROOT/supervisor-relative-path.out" \
  "error=supervisor_source_not_absolute"

awk '/-----BEGIN CONFIG JSON-----/{emit=1;next}/-----END CONFIG JSON-----/{emit=0}emit' "$OUTPUT" >"$CONFIG"
awk '/-----BEGIN PRIMITIVES HELPER-----/{emit=1;next}/-----END PRIMITIVES HELPER-----/{emit=0}emit' "$OUTPUT" >"$HELPER"
awk '/-----BEGIN PRIMITIVES SYSTEMD UNIT-----/{emit=1;next}/-----END PRIMITIVES SYSTEMD UNIT-----/{emit=0}emit' "$OUTPUT" >"$PRIMITIVES_UNIT"
awk '/-----BEGIN SYSTEMD UNIT-----/{emit=1;next}/-----END SYSTEMD UNIT-----/{emit=0}emit' "$OUTPUT" >"$UNIT"
chmod 0700 "$HELPER"
bash -n "$HELPER"
"$SHELLCHECK_COMMAND" "$HELPER"

python3 - "$CONFIG" "$LAUNCHER" "$FIRECRACKER" "$JAILER" "$KERNEL" "$GUEST" "$MANIFEST" <<'PY'
import hashlib
import json
import pathlib
import sys

config_path = pathlib.Path(sys.argv[1])
source_paths = [pathlib.Path(item) for item in sys.argv[2:]]
raw = config_path.read_bytes()
document = json.loads(raw)
expected_keys = {
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
assert set(document) == expected_keys
assert raw == (
    json.dumps(document, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()
    + b"\n"
)
launcher, firecracker, jailer, kernel, guest, manifest = source_paths
digests = {path: hashlib.sha256(path.read_bytes()).hexdigest() for path in source_paths}
assert document["socket_path"] == "/run/orquesta/firecracker-launcher.sock"
assert document["runtime_root"] == "/run/orquesta"
assert document["parent_cgroup"] == "orquesta-firecracker-attestor"
assert document["netns_path"] == "/run/netns/orquesta-firecracker-attestor-empty"
assert document["max_input_bytes"] == 512 * 1024 * 1024 + 1024 * 1024 + 1024
assert document["max_output_drive_bytes"] == 4 * 1024 * 1024
assert document["max_captured_output_bytes"] == 64 * 1024 * 1024
assert document["max_memory_bytes"] == 5 * 1024 * 1024 * 1024
assert document["max_pids"] == 512
assert document["max_cpu_quota_micros"] == 200_000
assert document["max_concurrent_runs"] == 16
assert document["allowed_uid"] == 1000 and document["jail_uid"] == 65534
assert document["allowed_gid"] == 1000 and document["jail_gid"] == 65534
assert document["firecracker_sha256"] == digests[firecracker]
assert document["jailer_sha256"] == digests[jailer]
assert document["kernel_sha256"] == digests[kernel]
assert document["guest_sha256"] == digests[guest]
assert document["guest_manifest_sha256"] == digests[manifest]
for field, source in (
    ("firecracker_command", firecracker),
    ("jailer_command", jailer),
    ("kernel_image", kernel),
    ("guest_image", guest),
    ("guest_manifest", manifest),
):
    assert digests[source] in document[field]
assert document["guest_image"].endswith(".cpio.gz")
assert digests[launcher] in pathlib.Path(
    next(
        line.split("=", 1)[1]
        for line in pathlib.Path(sys.argv[1]).with_name("dry-run.txt").read_text().splitlines()
        if line.startswith("launcher_path=")
    )
).name
PY

assert_contains "$OUTPUT" "max_subject_bytes=536870912"
assert_contains "$OUTPUT" "guest_memory_mib=4096"
assert_contains "$OUTPUT" "operational_reserve_bytes=2147483648"
assert_contains "$OUTPUT" "aggregate_microvm_memory_bytes=85899345920"
expected_asset_digest="$(
  python3 - "$firecracker_sha" "$jailer_sha" "$kernel_sha" "$guest_sha" "$manifest_sha" <<'PY'
import hashlib
import json
import sys

payload = {
    "schema": "orquesta.firecracker-launcher.assets.v1",
    "firecracker_sha256": sys.argv[1],
    "jailer_sha256": sys.argv[2],
    "kernel_sha256": sys.argv[3],
    "guest_sha256": sys.argv[4],
    "guest_manifest_sha256": sys.argv[5],
}
print(hashlib.sha256(json.dumps(
    payload, separators=(",", ":"), ensure_ascii=True
).encode("ascii")).hexdigest())
PY
)"
assert_contains "$OUTPUT" "expected_asset_digest=$expected_asset_digest"
assert_contains "$OUTPUT" "orquesta_config_key=test_attestor.microvm.expected_asset_digest"
assert_contains \
  "$OUTPUT" \
  "supervisor_path=/usr/local/libexec/orquesta-firecracker-attestor-e2e-$supervisor_sha"
assert_not_contains "$OUTPUT" "supervisor_path=$SUPERVISOR"
assert_contains "$OUTPUT" "activation=not_performed"
assert_not_contains "$OUTPUT" "bubblewrap"

# shellcheck disable=SC2016
assert_contains "$HELPER" 'ip netns add "$NETNS_NAME"'
assert_contains "$HELPER" 'link set dev lo down'
assert_not_contains "$HELPER" 'link set dev lo up'
assert_contains "$HELPER" 'runtime_root_absent'
assert_contains "$HELPER" 'runtime_marker_metadata'
# shellcheck disable=SC2016
assert_contains "$HELPER" 'mount --bind "$RUNTIME_BACKING_ROOT" "$RUNTIME_ROOT"'
# shellcheck disable=SC2016
assert_contains "$HELPER" 'mount -o remount,bind,rw,nosuid "$RUNTIME_ROOT"'
# shellcheck disable=SC2016
assert_contains "$HELPER" 'findmnt -rn -M "$RUNTIME_ROOT" -o TARGET'
# shellcheck disable=SC2016
assert_contains "$HELPER" 'findmnt -rn -M "$RUNTIME_ROOT" -o FSROOT'
assert_contains "$HELPER" 'runtime_mount_noexec'
assert_contains "$HELPER" 'runtime_mount_nodev'
assert_contains "$HELPER" 'cgroup.subtree_control'
assert_contains "$HELPER" 'memory.swap.max'
assert_contains "$HELPER" 'cgroup_parent_has_children'
assert_not_contains "$HELPER" 'rm -'

assert_contains "$UNIT" "NoNewPrivileges=yes"
assert_contains "$UNIT" "Requires=orquesta-firecracker-primitives-"
assert_contains "$UNIT" "ExecStartPre=/usr/local/libexec/orquesta-firecracker-primitives-"
assert_contains "$UNIT" " --check"
assert_contains "$UNIT" "CapabilityBoundingSet="
assert_contains "$UNIT" "CAP_SYS_ADMIN"
assert_contains "$UNIT" "PrivateDevices=no"
assert_line "$UNIT" "DeviceAllow=/dev/kvm rwm"
assert_line "$UNIT" "DeviceAllow=/dev/net/tun rwm"
assert_contains "$UNIT" "PrivateNetwork=no"
assert_contains "$UNIT" "PrivateMounts=yes"
assert_contains "$UNIT" "ProtectControlGroups=no"
assert_contains "$UNIT" "RestrictNamespaces=no"
assert_contains "$UNIT" "IPAddressDeny=any"
assert_contains "$UNIT" "RestrictAddressFamilies=AF_UNIX AF_NETLINK"
assert_not_contains "$UNIT" "PrivateDevices=yes"
assert_not_contains "$UNIT" "PrivateNetwork=yes"
assert_not_contains "$UNIT" "ProtectControlGroups=yes"
assert_not_contains "$UNIT" "RestrictNamespaces=yes"
assert_not_contains "$UNIT" "SystemCallFilter="
assert_contains "$UNIT" "ReadWritePaths=/srv/orquesta-self/runtime/firecracker-attestor /run/orquesta"

assert_contains "$PRIMITIVES_UNIT" "Type=oneshot"
assert_contains "$PRIMITIVES_UNIT" "ExecStart=/usr/local/libexec/orquesta-firecracker-primitives-"
assert_contains "$PRIMITIVES_UNIT" " --ensure"
assert_contains "$PRIMITIVES_UNIT" "CAP_NET_ADMIN"
assert_contains "$PRIMITIVES_UNIT" "PrivateMounts=no"
assert_contains "$PRIMITIVES_UNIT" "ProtectSystem=no"

readonly LAUNCHER_CGROUP_SOURCE="$TEST_SCRIPT_DIR/../internal/adapters/attestor/firecrackerlauncher/cgroup_linux.go"
readonly LAUNCHER_LIFECYCLE_SOURCE="$TEST_SCRIPT_DIR/../internal/adapters/attestor/firecrackerlauncher/runner_lifecycle_linux.go"
assert_contains "$LAUNCHER_CGROUP_SOURCE" '"memory.swap.max":  "0"'
assert_contains "$LAUNCHER_LIFECYCLE_SOURCE" '"--cgroup", "memory.swap.max=0"'
assert_contains "$LAUNCHER_LIFECYCLE_SOURCE" 'Do not pass --new-pid-ns or --daemonize'

# Las funciones de copia inmutable son idempotentes sin privilegios cuando se
# ejercitan con el uid/gid del test; la segunda pasada conserva inode y bytes.
# shellcheck disable=SC1090
source "$INSTALLER"

# `validate_receipt` exige root real en producción. Estas funciones sustituyen
# solo metadata para probar el contrato sin privilegios ni efectos del host.
fake_receipt_stat() {
  local format="${2:-}"
  local path="${*: -1}"
  case "$format" in
    '%u:%g:%a:%h')
      printf '%s\n' "${FAKE_RECEIPT_METADATA:-0:0:400:1}"
      ;;
    '%u:%g')
      printf '%s\n' "0:0"
      ;;
    '%a')
      if [[ -n "${FAKE_RECEIPT_UNSAFE_ANCESTOR:-}" &&
        "$path" == "$FAKE_RECEIPT_UNSAFE_ANCESTOR" ]]; then
        printf '%s\n' "777"
      else
        printf '%s\n' "755"
      fi
      ;;
    *)
      command stat "$@"
      ;;
  esac
}

validate_receipt_with_fake_root() (
  # shellcheck disable=SC2329
  stat() {
    fake_receipt_stat "$@"
  }
  validate_receipt "$@"
)

validate_bundle_with_fake_root() (
  # shellcheck disable=SC2329
  stat() {
    fake_receipt_stat "$@"
  }
  validate_activation_bundle "$@"
)

run_fake_receipt_activation() (
  local receipt="$1"
  local marker="$2"
  shift 2
  # shellcheck disable=SC2329
  stat() {
    fake_receipt_stat "$@"
  }
  activate_unit() {
    printf '%s\n' "activated" >"$marker"
  }
  validate_receipt "$receipt" "$@"
  activate_unit "/fake/content-addressed-unit"
)

run_fake_bundle_activation() (
  local receipt="$1"
  local evidence="$2"
  local marker="$3"
  shift 3
  # shellcheck disable=SC2329
  stat() {
    fake_receipt_stat "$@"
  }
  activate_unit() {
    printf '%s\n' "activated" >"$marker"
  }
  validate_activation_bundle "$receipt" "$evidence" "$@"
  activate_unit "/fake/content-addressed-unit"
)

write_activation_receipt() {
  local path="$1"
  local config_sha="$2"
  local unit_sha="$3"
  local launcher_digest="$4"
  local primitives_digest="$5"
  local assets_digest="$6"
  local evidence_digest="$7"
  local policy_digest="$8"
  printf '%s\n' \
    "schema=orquesta_firecracker_activation_receipt.v2" \
    "status=passed" \
    "config_sha256=$config_sha" \
    "unit_sha256=$unit_sha" \
    "primitives_unit_sha256=$primitives_digest" \
    "launcher_sha256=$launcher_digest" \
    "asset_digest=$assets_digest" \
    "evidence_sha256=$evidence_digest" \
    "policy_digest=$policy_digest" \
    "e2e_suite=orquesta.firecracker-attestor.physical-16.v1" \
    "max_concurrent_runs=16" \
    "physical_microvm_count=16" \
    "concurrent_high_water=16" \
    "all_attestations_valid=true" \
    "zero_residual_runs=true" \
    "network_absent=true" \
    "api_absent=true" \
    "vsock_absent=true" \
    "serial_absent=true" \
    "memory_swap_max_zero=true" >"$path"
  chmod 0400 "$path"
}

write_activation_evidence() {
  local path="$1"
  local config_sha="$2"
  local unit_sha="$3"
  local launcher_digest="$4"
  local primitives_digest="$5"
  local assets_digest="$6"
  local policy_digest="$7"
  local supervisor_digest="$8"
  python3 - \
    "$path" "$config_sha" "$unit_sha" "$launcher_digest" \
    "$primitives_digest" "$assets_digest" "$policy_digest" \
    "$supervisor_digest" <<'PY'
import json
import pathlib
import sys

(
    path, config_sha, unit_sha, launcher_sha, primitives_sha, asset_digest,
    policy_digest, supervisor_sha,
) = sys.argv[1:]
alphabet = "abcdefghijklmnopqrstuvwxyz234567"
run_ids = ["orq-" + "a" * 51 + alphabet[index] for index in range(17)]

def phase(ids, pids):
    return {
        "requested_runs": len(ids),
        "high_water_runs": len(ids),
        "samples": 2,
        "run_ids": ids,
        "firecracker_pids": pids,
        "limits_exact": True,
        "memory_swap_max_zero": True,
        "network_absent": True,
        "api_absent": True,
        "vsock_absent": True,
        "serial_absent": True,
        "unit_identity_stable": True,
    }

document = {
    "schema": "orquesta.firecracker-attestor.physical-16.evidence.v2",
    "suite": "orquesta.firecracker-attestor.physical-16.v1",
    "status": "passed",
    "started_at": "2026-07-25T10:00:00Z",
    "finished_at": "2026-07-25T10:01:00Z",
    "candidate": {
        "unit_sha256": unit_sha,
        "primitives_unit_sha256": primitives_sha,
        "launcher_sha256": launcher_sha,
        "config_sha256": config_sha,
        "supervisor_sha256": supervisor_sha,
        "asset_digest": asset_digest,
    },
    "policy_digest": policy_digest,
    "unit": {
        "unit_name": f"orquesta-firecracker-attestor-{unit_sha}.service",
        "main_pid": 1234,
        "invocation_id": "1" * 32,
        "active": True,
        "fragment_path": (
            f"/etc/systemd/system/orquesta-firecracker-attestor-{unit_sha}.service"
        ),
        "loaded": True,
        "need_daemon_reload": False,
    },
    "phase_one": phase(run_ids[:1], [1001]),
    "phase_sixteen": phase(run_ids[1:], list(range(2001, 2017))),
    "attestations": [
        {
            "ref": f"attestation-{0 if index == 1 else index:02d}",
            "run_id": current_run_id,
            "subject_digest": f"{1 if index == 1 else index + 1:064x}",
            "receipt_ref": f"attestation-{0 if index == 1 else index:02d}",
            "policy_digest": policy_digest,
            "valid": True,
        }
        for index, current_run_id in enumerate(run_ids)
    ],
    "cleanup": {
        "stable_samples": 2,
        "residual_runs": 0,
        "residual_cgroups": 0,
        "residual_processes": 0,
        "unit_stopped": True,
        "socket_absent": True,
    },
}
pathlib.Path(path).write_bytes(
    json.dumps(document, separators=(",", ":"), ensure_ascii=True).encode("ascii")
    + b"\n"
)
PY
  chmod 0400 "$path"
}

expect_receipt_rejected() {
  local label="$1"
  local path="$2"
  shift 2
  if validate_receipt_with_fake_root "$path" "$@" >/dev/null 2>&1; then
    fail "receipt_accepted:$label"
  fi
}

expect_bundle_activation_rejected() {
  local label="$1"
  local receipt="$2"
  local evidence="$3"
  local marker="$4"
  shift 4
  rm -f -- "$marker"
  if run_fake_bundle_activation \
    "$receipt" "$evidence" "$marker" "$@" >/dev/null 2>&1; then
    fail "bundle_accepted:$label"
  fi
  [[ ! -e "$marker" ]] || fail "bundle_activation_effect:$label"
}

readonly RECEIPT_CONFIG_SHA="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
readonly RECEIPT_UNIT_SHA="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
readonly RECEIPT_LAUNCHER_SHA="cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
readonly RECEIPT_PRIMITIVES_SHA="dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
readonly RECEIPT_ASSET_DIGEST="eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
readonly RECEIPT_EVIDENCE_SHA="ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
readonly RECEIPT_POLICY_DIGEST="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
readonly RECEIPT_SUPERVISOR_SHA="9999999999999999999999999999999999999999999999999999999999999999"
readonly RECEIPT_TEST_ROOT="$TEST_ROOT/receipts"
mkdir "$RECEIPT_TEST_ROOT"
readonly VALID_RECEIPT="$RECEIPT_TEST_ROOT/valid.receipt"
write_activation_receipt \
  "$VALID_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_EVIDENCE_SHA" "$RECEIPT_POLICY_DIGEST"
validate_receipt_with_fake_root \
  "$VALID_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

readonly VALID_EVIDENCE="$RECEIPT_TEST_ROOT/valid.evidence.json"
write_activation_evidence \
  "$VALID_EVIDENCE" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_POLICY_DIGEST" "$RECEIPT_SUPERVISOR_SHA"
BUNDLE_EVIDENCE_SHA="$(sha256sum "$VALID_EVIDENCE" | awk '{print $1}')"
readonly BUNDLE_EVIDENCE_SHA
readonly VALID_BUNDLE_RECEIPT="$RECEIPT_TEST_ROOT/valid-bundle.receipt"
write_activation_receipt \
  "$VALID_BUNDLE_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$BUNDLE_EVIDENCE_SHA" "$RECEIPT_POLICY_DIGEST"
validate_bundle_with_fake_root \
  "$VALID_BUNDLE_RECEIPT" "$VALID_EVIDENCE" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_SUPERVISOR_SHA"

write_receipt_for_evidence() {
  local receipt="$1"
  local evidence="$2"
  local policy="${3:-$RECEIPT_POLICY_DIGEST}"
  local evidence_sha
  evidence_sha="$(sha256sum "$evidence" | awk '{print $1}')"
  write_activation_receipt \
    "$receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
    "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
    "$evidence_sha" "$policy"
}

evidence_variant() {
  local name="$1"
  local mutation="$2"
  local path="$RECEIPT_TEST_ROOT/$name.evidence.json"
  python3 - "$VALID_EVIDENCE" "$path" "$mutation" <<'PY'
import json
import pathlib
import sys

source, destination, mutation = sys.argv[1:]
document = json.loads(pathlib.Path(source).read_bytes())
if mutation == "schema":
    document["schema"] = "orquesta.firecracker-attestor.physical-16.evidence.v1"
elif mutation == "candidate":
    document["candidate"]["config_sha256"] = "8" * 64
elif mutation == "policy":
    document["policy_digest"] = "7" * 64
elif mutation == "supervisor_hash":
    document["candidate"]["supervisor_sha256"] = "8" * 64
else:
    raise SystemExit("unknown mutation")
pathlib.Path(destination).write_bytes(
    json.dumps(document, separators=(",", ":"), ensure_ascii=True).encode("ascii")
    + b"\n"
)
PY
  chmod 0400 "$path"
  printf '%s\n' "$path"
}

readonly BUNDLE_ACTIVATION_MARKER="$RECEIPT_TEST_ROOT/bundle-activation.marker"
run_fake_bundle_activation \
  "$VALID_BUNDLE_RECEIPT" "$VALID_EVIDENCE" "$BUNDLE_ACTIVATION_MARKER" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_SUPERVISOR_SHA"
[[ "$(<"$BUNDLE_ACTIVATION_MARKER")" == "activated" ]] ||
  fail "valid_bundle_did_not_reach_activation"
rm -- "$BUNDLE_ACTIVATION_MARKER"

expect_bundle_activation_rejected \
  "missing_evidence" "$VALID_BUNDLE_RECEIPT" \
  "$RECEIPT_TEST_ROOT/missing.evidence.json" "$BUNDLE_ACTIVATION_MARKER" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_SUPERVISOR_SHA"

readonly HASH_MISMATCH_EVIDENCE="$RECEIPT_TEST_ROOT/hash-mismatch.evidence.json"
cp -- "$VALID_EVIDENCE" "$HASH_MISMATCH_EVIDENCE"
chmod 0600 "$HASH_MISMATCH_EVIDENCE"
printf ' ' >>"$HASH_MISMATCH_EVIDENCE"
chmod 0400 "$HASH_MISMATCH_EVIDENCE"
expect_bundle_activation_rejected \
  "evidence_hash" "$VALID_BUNDLE_RECEIPT" "$HASH_MISMATCH_EVIDENCE" \
  "$BUNDLE_ACTIVATION_MARKER" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_SUPERVISOR_SHA"

for mutation in schema candidate policy supervisor_hash; do
  bad_evidence="$(evidence_variant "$mutation" "$mutation")"
  bad_evidence_receipt="$RECEIPT_TEST_ROOT/$mutation-bundle.receipt"
  write_receipt_for_evidence "$bad_evidence_receipt" "$bad_evidence"
  expect_bundle_activation_rejected \
    "$mutation" "$bad_evidence_receipt" "$bad_evidence" \
    "$BUNDLE_ACTIVATION_MARKER" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
    "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
    "$RECEIPT_SUPERVISOR_SHA"
done

readonly TRUNCATED_EVIDENCE="$RECEIPT_TEST_ROOT/truncated.evidence.json"
head -c 128 -- "$VALID_EVIDENCE" >"$TRUNCATED_EVIDENCE"
chmod 0400 "$TRUNCATED_EVIDENCE"
readonly TRUNCATED_RECEIPT="$RECEIPT_TEST_ROOT/truncated-bundle.receipt"
write_receipt_for_evidence "$TRUNCATED_RECEIPT" "$TRUNCATED_EVIDENCE"
expect_bundle_activation_rejected \
  "truncated" "$TRUNCATED_RECEIPT" "$TRUNCATED_EVIDENCE" \
  "$BUNDLE_ACTIVATION_MARKER" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  "$RECEIPT_SUPERVISOR_SHA"

receipt_variant() {
  local name="$1"
  local old="$2"
  local new="$3"
  local path="$RECEIPT_TEST_ROOT/$name.receipt"
  cp -- "$VALID_RECEIPT" "$path"
  chmod 0600 "$path"
  sed -i "s|$old|$new|" "$path"
  chmod 0400 "$path"
  printf '%s\n' "$path"
}

bad_receipt="$(
  receipt_variant schema \
    "schema=orquesta_firecracker_activation_receipt.v2" \
    "schema=orquesta_firecracker_activation_receipt.v1"
)"
expect_receipt_rejected \
  "schema" "$bad_receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

bad_receipt="$(
  receipt_variant evidence \
    "evidence_sha256=$RECEIPT_EVIDENCE_SHA" \
    "evidence_sha256=FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
)"
expect_receipt_rejected \
  "evidence_sha256" "$bad_receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

bad_receipt="$(
  receipt_variant policy \
    "policy_digest=$RECEIPT_POLICY_DIGEST" \
    "policy_digest=0123456789abcdef"
)"
expect_receipt_rejected \
  "policy_digest" "$bad_receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

bad_receipt="$(
  receipt_variant high-water \
    "concurrent_high_water=16" \
    "concurrent_high_water=15"
)"
expect_receipt_rejected \
  "concurrent_high_water" "$bad_receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

for boolean_field in api_absent vsock_absent serial_absent; do
  bad_receipt="$(
    receipt_variant "$boolean_field" \
      "$boolean_field=true" \
      "$boolean_field=false"
  )"
  expect_receipt_rejected \
    "$boolean_field" "$bad_receipt" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
    "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"
done

readonly TAMPERED_RECEIPT="$RECEIPT_TEST_ROOT/tampered.receipt"
cp -- "$VALID_RECEIPT" "$TAMPERED_RECEIPT"
chmod 0600 "$TAMPERED_RECEIPT"
printf '%s\n' "unexpected=true" >>"$TAMPERED_RECEIPT"
chmod 0400 "$TAMPERED_RECEIPT"
expect_receipt_rejected \
  "tamper" "$TAMPERED_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"

if FAKE_RECEIPT_UNSAFE_ANCESTOR="$RECEIPT_TEST_ROOT" \
  validate_receipt_with_fake_root \
    "$VALID_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
    "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
    >/dev/null 2>&1; then
  fail "receipt_unsafe_ancestor_accepted"
fi

readonly RECEIPT_SYMLINK_ROOT="$TEST_ROOT/receipt-link"
ln -s -- "$RECEIPT_TEST_ROOT" "$RECEIPT_SYMLINK_ROOT"
if validate_receipt_with_fake_root \
  "$RECEIPT_SYMLINK_ROOT/valid.receipt" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" >/dev/null 2>&1; then
  fail "receipt_symlink_ancestor_accepted"
fi

if FAKE_RECEIPT_METADATA="0:0:600:1" validate_receipt_with_fake_root \
  "$VALID_RECEIPT" "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" \
  "$RECEIPT_LAUNCHER_SHA" "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" \
  >/dev/null 2>&1; then
  fail "receipt_wrong_mode_accepted"
fi

readonly FAKE_ACTIVATION_MARKER="$RECEIPT_TEST_ROOT/activation.marker"
run_fake_receipt_activation \
  "$VALID_RECEIPT" "$FAKE_ACTIVATION_MARKER" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST"
[[ "$(<"$FAKE_ACTIVATION_MARKER")" == "activated" ]] ||
  fail "valid_receipt_did_not_reach_activation"
rm -- "$FAKE_ACTIVATION_MARKER"
if run_fake_receipt_activation \
  "$TAMPERED_RECEIPT" "$FAKE_ACTIVATION_MARKER" \
  "$RECEIPT_CONFIG_SHA" "$RECEIPT_UNIT_SHA" "$RECEIPT_LAUNCHER_SHA" \
  "$RECEIPT_PRIMITIVES_SHA" "$RECEIPT_ASSET_DIGEST" >/dev/null 2>&1; then
  fail "tampered_receipt_reached_activation"
fi
[[ ! -e "$FAKE_ACTIVATION_MARKER" ]] ||
  fail "tampered_receipt_activation_effect"

golden_asset_digest="$(
  canonical_asset_digest \
    1111111111111111111111111111111111111111111111111111111111111111 \
    2222222222222222222222222222222222222222222222222222222222222222 \
    3333333333333333333333333333333333333333333333333333333333333333 \
    4444444444444444444444444444444444444444444444444444444444444444 \
    5555555555555555555555555555555555555555555555555555555555555555
)"
[[ "$golden_asset_digest" == \
  "1b248da4d891d9e08e1febb62e93ff1c317917035c1bb1382da2712c064a549d" ]] ||
  fail "canonical_asset_digest_golden"
immutable_source="$TEST_ROOT/immutable-source"
immutable_destination="$TEST_ROOT/immutable-destination"
printf 'immutable\n' >"$immutable_source"
chmod 0600 "$immutable_source"
immutable_sha="$(sha256sum "$immutable_source" | awk '{print $1}')"
test_uid="$(id -u)"
test_gid="$(id -g)"
install_immutable_file "$immutable_source" "$immutable_destination" "$immutable_sha" "$test_uid" "$test_gid" 0600
inode_before="$(stat -c '%i' "$immutable_destination")"
install_immutable_file "$immutable_source" "$immutable_destination" "$immutable_sha" "$test_uid" "$test_gid" 0600
inode_after="$(stat -c '%i' "$immutable_destination")"
[[ "$inode_before" == "$inode_after" ]] || fail "immutable_install_not_idempotent"

supervisor_destination="$TEST_ROOT/orquesta-firecracker-attestor-e2e-$supervisor_sha"
install_immutable_file \
  "$SUPERVISOR" "$supervisor_destination" "$supervisor_sha" \
  "$test_uid" "$test_gid" 0755
supervisor_inode_before="$(stat -c '%i' "$supervisor_destination")"
install_immutable_file \
  "$SUPERVISOR" "$supervisor_destination" "$supervisor_sha" \
  "$test_uid" "$test_gid" 0755
supervisor_inode_after="$(stat -c '%i' "$supervisor_destination")"
[[ "$supervisor_inode_before" == "$supervisor_inode_after" ]] ||
  fail "supervisor_install_not_idempotent"
chmod 0700 "$supervisor_destination"
printf '%s\n' "# tamper" >>"$supervisor_destination"
chmod 0755 "$supervisor_destination"
if (verify_immutable_file \
  "$supervisor_destination" "$supervisor_sha" "$test_uid" "$test_gid" 0755 \
  >/dev/null 2>&1); then
  fail "tampered_installed_supervisor_accepted"
fi

readonly LOCK_TEST_PARENT="$TEST_ROOT/transaction-lock"
readonly LOCK_TEST_PATH="$LOCK_TEST_PARENT/installer.lock"
readonly LOCK_READY="$LOCK_TEST_PARENT/ready"
readonly LOCK_RELEASE="$LOCK_TEST_PARENT/release"
readonly LOCK_EFFECT="$LOCK_TEST_PARENT/effect"
mkdir -m 0700 "$LOCK_TEST_PARENT"
mkfifo -m 0600 "$LOCK_RELEASE"
lock_test_uid="$(id -u)"
lock_test_gid="$(id -g)"
(
  acquire_transaction_lock \
    "$LOCK_TEST_PATH" "$LOCK_TEST_PARENT" "$lock_test_uid" "$lock_test_gid" 5
  printf '%s\n' "locked" >"$LOCK_READY"
  IFS= read -r release <"$LOCK_RELEASE"
  [[ "$release" == "release" ]] || exit 1
  release_transaction_lock
) &
lock_holder_pid=$!
lock_ready="false"
for _ in {1..500}; do
  if [[ -f "$LOCK_READY" ]]; then
    lock_ready="true"
    break
  fi
  kill -0 "$lock_holder_pid" 2>/dev/null ||
    fail "transaction_lock_holder_failed"
  sleep 0.01
done
[[ "$lock_ready" == "true" ]] || fail "transaction_lock_holder_timeout"
if (
  acquire_transaction_lock \
    "$LOCK_TEST_PATH" "$LOCK_TEST_PARENT" "$lock_test_uid" "$lock_test_gid" 1
  printf '%s\n' "unsafe" >"$LOCK_EFFECT"
) >"$LOCK_TEST_PARENT/contender.out" 2>&1; then
  fail "transaction_lock_contention_accepted"
fi
assert_contains "$LOCK_TEST_PARENT/contender.out" "error=transaction_lock_timeout"
[[ ! -e "$LOCK_EFFECT" ]] || fail "transaction_lock_contention_effect"
printf '%s\n' "release" >"$LOCK_RELEASE"
wait "$lock_holder_pid"
(
  acquire_transaction_lock \
    "$LOCK_TEST_PATH" "$LOCK_TEST_PARENT" "$lock_test_uid" "$lock_test_gid" 1
  printf '%s\n' "serialized" >"$LOCK_EFFECT"
  release_transaction_lock
)
[[ "$(<"$LOCK_EFFECT")" == "serialized" ]] ||
  fail "transaction_lock_postcondition"
lock_metadata="$(stat -c '%u:%g:%a:%h' "$LOCK_TEST_PATH")"
[[ "$lock_metadata" == "$lock_test_uid:$lock_test_gid:600:1" ]] ||
  fail "transaction_lock_metadata_postcondition"

tampered_config="$TEST_ROOT/tampered-config.json"
python3 - "$CONFIG" "$tampered_config" <<'PY'
import json
import pathlib
import sys

document = json.loads(pathlib.Path(sys.argv[1]).read_bytes())
document["unknown"] = True
pathlib.Path(sys.argv[2]).write_text(
    json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n"
)
PY
if (validate_config_document "$tampered_config") >/dev/null 2>&1; then
  fail "unknown_config_key_accepted"
fi

tampered_manifest="$TEST_ROOT/tampered-manifest.json"
python3 - "$MANIFEST" "$tampered_manifest" <<'PY'
import json
import pathlib
import sys

document = json.loads(pathlib.Path(sys.argv[1]).read_bytes())
del document["build"]["toolchain_nobody_go_test"]
pathlib.Path(sys.argv[2]).write_text(
    json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n"
)
PY
chmod 0600 "$tampered_manifest"
if (verify_guest_manifest "$tampered_manifest" "$guest_sha") >/dev/null 2>&1; then
  fail "incomplete_guest_manifest_accepted"
fi

tampered_version_manifest="$TEST_ROOT/tampered-version-manifest.json"
python3 - "$MANIFEST" "$tampered_version_manifest" <<'PY'
import json
import pathlib
import sys

document = json.loads(pathlib.Path(sys.argv[1]).read_bytes())
document["busybox_version"] = "BusyBox v1.36.1"
pathlib.Path(sys.argv[2]).write_text(
    json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n"
)
PY
chmod 0600 "$tampered_version_manifest"
if (verify_guest_manifest "$tampered_version_manifest" "$guest_sha") \
  >/dev/null 2>&1; then
  fail "noncanonical_busybox_version_accepted"
fi

if [[ "$(id -u)" != "0" ]]; then
  if (verify_root_trusted_source "launcher" "$LAUNCHER") >/dev/null 2>&1; then
    fail "nonroot_source_accepted_as_root_trusted"
  fi
  if (verify_root_trusted_source "supervisor" "$SUPERVISOR") >/dev/null 2>&1; then
    fail "nonroot_supervisor_accepted_as_root_trusted"
  fi
fi

if "$INSTALLER" --dry-run \
  --profile host-128g-16 \
  --launcher-source "$LAUNCHER" \
  --supervisor-source "$SUPERVISOR" \
  --firecracker-source "$FIRECRACKER" \
  --jailer-source "$JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
  --supervisor-sha256 "$supervisor_sha" \
  --firecracker-sha256 "$firecracker_sha" \
  --jailer-sha256 "$jailer_sha" \
  --kernel-sha256 "$kernel_sha" \
  --guest-sha256 "$guest_sha" \
  --guest-manifest-sha256 "$manifest_sha" \
  --allowed-uid 65534 --allowed-gid 1000 --jail-uid 65534 --jail-gid 65534 \
  >"$TEST_ROOT/invalid-identities.out" 2>&1; then
  fail "equal_allowed_and_jail_identity_accepted"
fi

readonly BAD_JAILER="$TEST_ROOT/jailer-wrong-version"
write_fake_executable "$BAD_JAILER" "jailer"
sed -i 's/v1[.]16[.]1/v1.16.10/' "$BAD_JAILER"
bad_jailer_sha="$(sha256sum "$BAD_JAILER" | awk '{print $1}')"
if "$INSTALLER" --dry-run \
  --profile host-128g-16 \
  --launcher-source "$LAUNCHER" \
  --supervisor-source "$SUPERVISOR" \
  --firecracker-source "$FIRECRACKER" \
  --jailer-source "$BAD_JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
  --supervisor-sha256 "$supervisor_sha" \
  --firecracker-sha256 "$firecracker_sha" \
  --jailer-sha256 "$bad_jailer_sha" \
  --kernel-sha256 "$kernel_sha" \
  --guest-sha256 "$guest_sha" \
  --guest-manifest-sha256 "$manifest_sha" \
  --allowed-uid 1000 --allowed-gid 1000 --jail-uid 65534 --jail-gid 65534 \
  >"$TEST_ROOT/invalid-version.out" 2>&1; then
  fail "wrong_jailer_version_accepted"
fi

readonly MUST_NOT_EXECUTE="$TEST_ROOT/must-not-execute"
readonly HASH_GATED_FIRECRACKER="$TEST_ROOT/firecracker-hash-gated"
printf '%s\n' \
  '#!/usr/bin/env sh' \
  "printf 'executed\\n' >'$MUST_NOT_EXECUTE'" \
  "printf 'Firecracker v1.16.1\\n'" >"$HASH_GATED_FIRECRACKER"
chmod 0700 "$HASH_GATED_FIRECRACKER"
if "$INSTALLER" --dry-run \
  --profile host-128g-16 \
  --launcher-source "$LAUNCHER" \
  --supervisor-source "$SUPERVISOR" \
  --firecracker-source "$HASH_GATED_FIRECRACKER" \
  --jailer-source "$JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
  --supervisor-sha256 "$supervisor_sha" \
  --firecracker-sha256 0000000000000000000000000000000000000000000000000000000000000000 \
  --jailer-sha256 "$jailer_sha" \
  --kernel-sha256 "$kernel_sha" \
  --guest-sha256 "$guest_sha" \
  --guest-manifest-sha256 "$manifest_sha" \
  --allowed-uid 1000 --allowed-gid 1000 --jail-uid 65534 --jail-gid 65534 \
  >"$TEST_ROOT/hash-gate.out" 2>&1; then
  fail "wrong_expected_hash_accepted"
fi
[[ ! -e "$MUST_NOT_EXECUTE" ]] || fail "source_executed_before_hash_gate"

# Regresión systemd: el alias canónico nunca se pasa a `enable`; se habilita
# la unidad versionada real. Un fallo restaura symlink, enable y actividad.
declare -A FAKE_ENABLED=()
declare -A FAKE_ACTIVE=()
FAKE_SYSTEMCTL_LOG="$TEST_ROOT/systemctl.log"
FAKE_FAIL_RESTART_UNIT=""

systemctl() {
  local operation="${1:-}"
  shift || true
  printf '%s %s\n' "$operation" "$*" >>"$FAKE_SYSTEMCTL_LOG"
  local quiet="false"
  if [[ "${1:-}" == "--quiet" ]]; then
    quiet="true"
    shift
  fi
  local unit_name="${1:-}"
  case "$operation" in
    daemon-reload) return 0 ;;
    is-enabled)
      [[ "$quiet" == "true" ]] ||
        printf '%s\n' "${FAKE_ENABLED[$unit_name]:-disabled}"
      [[ "${FAKE_ENABLED[$unit_name]:-disabled}" == "enabled" ]]
      ;;
    is-active)
      [[ "$quiet" == "true" ]] ||
        printf '%s\n' "${FAKE_ACTIVE[$unit_name]:-inactive}"
      [[ "${FAKE_ACTIVE[$unit_name]:-inactive}" == "active" ]]
      ;;
    enable)
      FAKE_ENABLED["$unit_name"]="enabled"
      ;;
    disable)
      FAKE_ENABLED["$unit_name"]="disabled"
      ;;
    restart)
      [[ "$unit_name" != "$FAKE_FAIL_RESTART_UNIT" ]] || return 1
      FAKE_ACTIVE["$unit_name"]="active"
      ;;
    stop)
      FAKE_ACTIVE["$unit_name"]="inactive"
      ;;
    *) return 1 ;;
  esac
}

activation_root="$TEST_ROOT/activation"
mkdir "$activation_root"
canonical_unit="$activation_root/orquesta-firecracker-attestor.service"
candidate_staging="$activation_root/candidate.staging"
printf '[Service]\nDescription=candidate\nExecStart=/bin/true\n' >"$candidate_staging"
candidate_sha="$(sha256sum "$candidate_staging" | awk '{print $1}')"
candidate_unit="$activation_root/orquesta-firecracker-attestor-$candidate_sha.service"
mv "$candidate_staging" "$candidate_unit"
candidate_name="$(basename "$candidate_unit")"
FAKE_ENABLED["$candidate_name"]="disabled"
FAKE_ACTIVE["$candidate_name"]="inactive"

unrelated_unit="$activation_root/ssh.service"
printf '[Service]\nDescription=unrelated\nExecStart=/bin/true\n' >"$unrelated_unit"
ln -s -- "$unrelated_unit" "$canonical_unit"
: >"$FAKE_SYSTEMCTL_LOG"
if (activate_unit "$candidate_unit" "$canonical_unit") \
  >"$TEST_ROOT/activation-unrelated.out" 2>&1; then
  fail "unrelated_previous_unit_accepted"
fi
[[ "$(readlink "$canonical_unit")" == "$unrelated_unit" ]] ||
  fail "unrelated_previous_alias_mutated"
[[ ! -s "$FAKE_SYSTEMCTL_LOG" ]] ||
  fail "unrelated_previous_reached_systemctl"
[[ -z "$(find "$activation_root" -maxdepth 1 \
  \( -name '*.new.*' -o -name '*.rollback.*' \) -print -quit)" ]] ||
  fail "unrelated_previous_created_activation_files"
assert_contains "$TEST_ROOT/activation-unrelated.out" "error=activation_unit_name"
unlink "$canonical_unit"

mismatched_unit="$activation_root/orquesta-firecracker-attestor-$(printf '0%.0s' {1..64}).service"
printf '[Service]\nDescription=mismatched\nExecStart=/bin/true\n' >"$mismatched_unit"
ln -s -- "$mismatched_unit" "$canonical_unit"
: >"$FAKE_SYSTEMCTL_LOG"
if (activate_unit "$candidate_unit" "$canonical_unit") \
  >"$TEST_ROOT/activation-mismatched.out" 2>&1; then
  fail "mismatched_previous_digest_accepted"
fi
[[ "$(readlink "$canonical_unit")" == "$mismatched_unit" ]] ||
  fail "mismatched_previous_alias_mutated"
[[ ! -s "$FAKE_SYSTEMCTL_LOG" ]] ||
  fail "mismatched_previous_reached_systemctl"
[[ -z "$(find "$activation_root" -maxdepth 1 \
  \( -name '*.new.*' -o -name '*.rollback.*' \) -print -quit)" ]] ||
  fail "mismatched_previous_created_activation_files"
assert_contains "$TEST_ROOT/activation-mismatched.out" "error=activation_unit_digest"
unlink "$canonical_unit"

: >"$FAKE_SYSTEMCTL_LOG"
activate_unit "$candidate_unit" "$canonical_unit" >"$TEST_ROOT/activation-success.out"
[[ "$(realpath -e "$canonical_unit")" == "$candidate_unit" ]] ||
  fail "activation_did_not_publish_candidate"
assert_contains "$FAKE_SYSTEMCTL_LOG" "enable $candidate_name"
assert_not_contains "$FAKE_SYSTEMCTL_LOG" "enable $(basename "$canonical_unit")"
assert_contains "$TEST_ROOT/activation-success.out" "enabled_unit=$candidate_name"
[[ "${FAKE_ENABLED[$candidate_name]}" == "enabled" &&
  "${FAKE_ACTIVE[$candidate_name]}" == "active" ]] ||
  fail "activation_candidate_state"

previous_staging="$activation_root/previous.staging"
printf '[Service]\nDescription=previous\nExecStart=/bin/true\n' >"$previous_staging"
previous_sha="$(sha256sum "$previous_staging" | awk '{print $1}')"
previous_unit="$activation_root/orquesta-firecracker-attestor-$previous_sha.service"
mv "$previous_staging" "$previous_unit"
ln -sfn -- "$previous_unit" "$canonical_unit"
previous_name="$(basename "$previous_unit")"
FAKE_ENABLED["$previous_name"]="enabled"
FAKE_ACTIVE["$previous_name"]="active"
FAKE_ENABLED["$candidate_name"]="disabled"
FAKE_ACTIVE["$candidate_name"]="inactive"
FAKE_FAIL_RESTART_UNIT="$candidate_name"
: >"$FAKE_SYSTEMCTL_LOG"
if (activate_unit "$candidate_unit" "$canonical_unit") \
  >"$TEST_ROOT/activation-failure.out" 2>&1; then
  fail "failed_activation_reported_success"
fi
[[ "$(readlink "$canonical_unit")" == "$previous_unit" ]] ||
  fail "activation_rollback_target"
assert_contains "$FAKE_SYSTEMCTL_LOG" "stop $previous_name"
assert_contains "$FAKE_SYSTEMCTL_LOG" "disable $candidate_name"
assert_contains "$FAKE_SYSTEMCTL_LOG" "stop $candidate_name"
assert_contains "$FAKE_SYSTEMCTL_LOG" "enable $previous_name"
assert_contains "$FAKE_SYSTEMCTL_LOG" "restart $previous_name"
assert_contains "$TEST_ROOT/activation-failure.out" "error=activation_failed"
FAKE_FAIL_RESTART_UNIT=""

if "$INSTALLER" --activate "${BASE_ARGUMENTS[@]}" \
  >"$TEST_ROOT/activate-without-receipt.out" 2>&1; then
  fail "activation_without_physical_e2e_receipt"
fi
assert_contains \
  "$TEST_ROOT/activate-without-receipt.out" \
  "error=activate_requires_e2e_receipt"

if "$INSTALLER" --activate "${BASE_ARGUMENTS[@]}" \
  --e2e-receipt "$VALID_BUNDLE_RECEIPT" \
  >"$TEST_ROOT/activate-without-evidence.out" 2>&1; then
  fail "activation_without_physical_e2e_evidence"
fi
assert_contains \
  "$TEST_ROOT/activate-without-evidence.out" \
  "error=activate_requires_e2e_evidence"

if "$INSTALLER" --dry-run "${BASE_ARGUMENTS[@]}" \
  --e2e-evidence "$VALID_EVIDENCE" \
  >"$TEST_ROOT/dry-run-with-evidence.out" 2>&1; then
  fail "evidence_accepted_outside_activation"
fi
assert_contains \
  "$TEST_ROOT/dry-run-with-evidence.out" \
  "error=e2e_bundle_only_valid_with_activate"

bash -n "$INSTALLER" "$0"
"$SHELLCHECK_COMMAND" "$INSTALLER" "$0"
printf 'installer_test=ok\n'
