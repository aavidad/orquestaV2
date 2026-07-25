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
firecracker_sha="$(sha256sum "$FIRECRACKER" | awk '{print $1}')"
jailer_sha="$(sha256sum "$JAILER" | awk '{print $1}')"
kernel_sha="$(sha256sum "$KERNEL" | awk '{print $1}')"
manifest_sha="$(sha256sum "$MANIFEST" | awk '{print $1}')"

readonly -a BASE_ARGUMENTS=(
  --profile host-128g-16
  --launcher-source "$LAUNCHER"
  --firecracker-source "$FIRECRACKER"
  --jailer-source "$JAILER"
  --kernel-source "$KERNEL"
  --guest-source "$GUEST"
  --guest-manifest-source "$MANIFEST"
  --launcher-sha256 "$launcher_sha"
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

"$INSTALLER" --dry-run "${BASE_ARGUMENTS[@]}" >"$OUTPUT"

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
fi

if "$INSTALLER" --dry-run \
  --profile host-128g-16 \
  --launcher-source "$LAUNCHER" \
  --firecracker-source "$FIRECRACKER" \
  --jailer-source "$JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
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
  --firecracker-source "$FIRECRACKER" \
  --jailer-source "$BAD_JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
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
  --firecracker-source "$HASH_GATED_FIRECRACKER" \
  --jailer-source "$JAILER" \
  --kernel-source "$KERNEL" \
  --guest-source "$GUEST" \
  --guest-manifest-source "$MANIFEST" \
  --launcher-sha256 "$launcher_sha" \
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

bash -n "$INSTALLER" "$0"
"$SHELLCHECK_COMMAND" "$INSTALLER" "$0"
printf 'installer_test=ok\n'
