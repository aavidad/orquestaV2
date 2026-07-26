#!/usr/bin/env bash
# Promoción reentrante de un perfil Orquesta en systemd --user delegado.
#
# El modo por defecto solo inspecciona. --apply es el único modo que para,
# crea backup, recoge o arranca la unidad. La política del núcleo no vive aquí.

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

readonly PROGRAM="orquesta_profile_systemd_promotion"
readonly PHASE_SCHEMA="orquesta_profile_systemd_promotion_phase.v1"
readonly RECEIPT_SCHEMA="orquesta_profile_systemd_promotion_receipt.v1"
readonly CONTROL_GROUP_NAME="orquesta-control"
readonly USER_UNIT_PROPERTIES="LoadState,ActiveState,SubState,Result,ExecMainCode,ExecMainStatus,MainPID,ControlGroup,InvocationID,Type,CollectMode,Delegate,DelegateSubgroup,MemoryAccounting,TasksAccounting,KillMode,SendSIGKILL,TimeoutStopUSec,Restart,LimitNOFILE,LimitNOFILESoft,LimitFSIZE,LimitFSIZESoft,LimitAS,LimitASSoft,LimitCPU,LimitCPUSoft,TasksMax,RuntimeMaxUSec,CPUQuotaPerSecUSec,MemoryMax,MemorySwapMax"
readonly FIRECRACKER_UNIT_PROPERTIES="LoadState,ActiveState,SubState,UnitFileState,Result,MainPID,InvocationID,FragmentPath,NeedDaemonReload"

fail() {
  printf '%s: status=error reason_code=%s\n' "$PROGRAM" "$1" >&2
  exit "${2:-1}"
}

usage() {
  cat <<'EOF'
uso:
  orquesta_profile_systemd_promotion.sh [--check] CONTRATO
  orquesta_profile_systemd_promotion.sh --apply CONTRATO

El contrato exige rutas canónicas y hashes para repo/HEAD/upstream, binario,
configs primaria y rollback, profile, adaptador systemd, helper, herramientas,
SQLite, recibo de upgrade y gate vivo Firecracker. El receipt físico root:root
0400 es precondición del wrapper privilegiado: este proceso de usuario conserva
solo su ref y no afirma haberlo releído. Use --help-contract para ver los
nombres exactos. --check no crea locks, backups, fases ni receipts.
EOF
}

contract_usage() {
  cat <<'EOF'
--unit --profile --repository-root --repository-head --candidate-revision
--expected-upstream-ref
--expected-config-target-ref --git --expected-git-sha256 --go
--expected-go-sha256 --profile-script --expected-profile-script-sha256
--systemd-adapter --expected-systemd-adapter-sha256 --promotion-helper
--expected-promotion-helper-sha256 --runtime-base --source-codex-home --binary
--expected-binary-sha256 --primary-config --expected-primary-config-sha256
--rollback-config --expected-rollback-config-sha256 --bubblewrap
--expected-bubblewrap-sha256 --sqlite-db --backup-dir --promotion-state-dir
--expected-cgroup-root --exec-path --systemctl --expected-systemctl-sha256
--systemd-run --expected-systemd-run-sha256 --proc-root --cgroup-mount
--firecracker-probe --expected-firecracker-probe-sha256 --firecracker-unit
--firecracker-root-gate-ref --expected-firecracker-unit-sha256
--expected-firecracker-unit-file-uid --expected-firecracker-unit-file-gid
--launcher-socket --expected-launcher-socket-uid
--expected-launcher-socket-gid --expected-asset-digest
--sqlite-upgrade-receipt --expected-sqlite-upgrade-receipt-sha256
--expected-primary-max-concurrent-runs --expected-rollback-max-concurrent-runs
[--readiness-timeout] [--collection-timeout]
EOF
}

mode="check"
case "${1:-}" in
  --check) shift ;;
  --apply) mode="apply"; shift ;;
  --help|-h) usage; exit 0 ;;
  --help-contract) contract_usage; exit 0 ;;
esac

declare -A value=()
while [ "$#" -gt 0 ]; do
  option="$1"
  case "$option" in
    --unit|--profile|--repository-root|--repository-head|--candidate-revision|\
    --expected-upstream-ref|\
    --expected-config-target-ref|--git|--expected-git-sha256|--go|\
    --expected-go-sha256|--profile-script|--expected-profile-script-sha256|\
    --systemd-adapter|--expected-systemd-adapter-sha256|--promotion-helper|\
    --expected-promotion-helper-sha256|--runtime-base|--source-codex-home|\
    --binary|--expected-binary-sha256|--primary-config|\
    --expected-primary-config-sha256|--rollback-config|\
    --expected-rollback-config-sha256|--bubblewrap|\
    --expected-bubblewrap-sha256|--sqlite-db|--backup-dir|\
    --promotion-state-dir|--expected-cgroup-root|--exec-path|--systemctl|\
    --expected-systemctl-sha256|--systemd-run|--expected-systemd-run-sha256|\
    --proc-root|--cgroup-mount|--firecracker-probe|\
    --expected-firecracker-probe-sha256|--firecracker-unit|\
    --firecracker-root-gate-ref|--expected-firecracker-unit-sha256|\
    --expected-firecracker-unit-file-uid|\
    --expected-firecracker-unit-file-gid|--launcher-socket|\
    --expected-launcher-socket-uid|\
    --expected-launcher-socket-gid|--expected-asset-digest|\
    --sqlite-upgrade-receipt|--expected-sqlite-upgrade-receipt-sha256|\
    --expected-primary-max-concurrent-runs|\
    --expected-rollback-max-concurrent-runs|--readiness-timeout|\
    --collection-timeout)
      [ "$#" -ge 2 ] && [ -n "$2" ] || fail "argument_value_missing" 2
      key="${option#--}"
      [ -z "${value[$key]+set}" ] || fail "argument_repeated" 2
      value["$key"]="$2"
      shift 2
      ;;
    *) fail "argument_unknown" 2 ;;
  esac
done

readonly REQUIRED_KEYS=(
  unit profile repository-root repository-head candidate-revision
  expected-upstream-ref
  expected-config-target-ref git expected-git-sha256 go expected-go-sha256
  profile-script expected-profile-script-sha256 systemd-adapter
  expected-systemd-adapter-sha256 promotion-helper
  expected-promotion-helper-sha256 runtime-base source-codex-home binary
  expected-binary-sha256 primary-config expected-primary-config-sha256
  rollback-config expected-rollback-config-sha256 bubblewrap
  expected-bubblewrap-sha256 sqlite-db backup-dir promotion-state-dir
  expected-cgroup-root exec-path systemctl expected-systemctl-sha256
  systemd-run expected-systemd-run-sha256 proc-root cgroup-mount
  firecracker-probe expected-firecracker-probe-sha256 firecracker-unit
  firecracker-root-gate-ref expected-firecracker-unit-sha256
  expected-firecracker-unit-file-uid expected-firecracker-unit-file-gid
  launcher-socket expected-launcher-socket-uid expected-launcher-socket-gid
  expected-asset-digest sqlite-upgrade-receipt
  expected-sqlite-upgrade-receipt-sha256
  expected-primary-max-concurrent-runs
  expected-rollback-max-concurrent-runs
)
for key in "${REQUIRED_KEYS[@]}"; do
  [ -n "${value[$key]:-}" ] || fail "contract_incomplete" 2
done

readonly unit="${value[unit]}" profile="${value[profile]}"
readonly repository_root="${value[repository-root]}"
readonly repository_head="${value[repository-head]}"
readonly candidate_revision="${value[candidate-revision]}"
readonly expected_upstream_ref="${value[expected-upstream-ref]}"
readonly expected_config_target_ref="${value[expected-config-target-ref]}"
readonly git_command="${value[git]}" go_command="${value[go]}"
readonly profile_script="${value[profile-script]}"
readonly systemd_adapter="${value[systemd-adapter]}"
readonly promotion_helper="${value[promotion-helper]}"
readonly runtime_base="${value[runtime-base]}"
readonly source_codex_home="${value[source-codex-home]}"
readonly binary="${value[binary]}"
readonly primary_config="${value[primary-config]}"
readonly rollback_config="${value[rollback-config]}"
readonly bubblewrap="${value[bubblewrap]}"
readonly sqlite_db="${value[sqlite-db]}" backup_dir="${value[backup-dir]}"
readonly promotion_state_dir="${value[promotion-state-dir]}"
readonly expected_cgroup_root="${value[expected-cgroup-root]}"
readonly exec_path="${value[exec-path]}"
readonly systemctl_command="${value[systemctl]}"
readonly systemd_run_command="${value[systemd-run]}"
readonly proc_root="${value[proc-root]}" cgroup_mount="${value[cgroup-mount]}"
readonly firecracker_probe="${value[firecracker-probe]}"
readonly firecracker_unit="${value[firecracker-unit]}"
readonly firecracker_root_gate_ref="${value[firecracker-root-gate-ref]}"
readonly launcher_socket="${value[launcher-socket]}"
readonly sqlite_upgrade_receipt="${value[sqlite-upgrade-receipt]}"
readonly readiness_timeout="${value[readiness-timeout]:-30}"
readonly collection_timeout="${value[collection-timeout]:-10}"
current_uid="$(id -u)"
readonly current_uid

readonly SHA_KEYS=(
  expected-git-sha256 expected-go-sha256 expected-profile-script-sha256
  expected-systemd-adapter-sha256 expected-promotion-helper-sha256
  expected-binary-sha256 expected-primary-config-sha256
  expected-rollback-config-sha256 expected-bubblewrap-sha256
  expected-systemctl-sha256 expected-systemd-run-sha256
  expected-firecracker-probe-sha256 expected-firecracker-unit-sha256
  expected-asset-digest
  expected-sqlite-upgrade-receipt-sha256
)
for key in "${SHA_KEYS[@]}"; do
  [[ "${value[$key]}" =~ ^[0-9a-f]{64}$ ]] ||
    fail "expected_sha256_invalid" 2
done
[[ "$profile" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$ ]] ||
  fail "profile_invalid" 2
[[ "$unit" =~ ^orquesta-v[1-9][0-9]*-"$profile"\.service$ ]] ||
  fail "unit_name_invalid" 2
[[ "$repository_head" =~ ^[0-9a-f]{40}$ ]] ||
  fail "repository_head_invalid" 2
[[ "$candidate_revision" =~ ^[0-9a-f]{40}$ ]] ||
  fail "candidate_revision_invalid" 2
for key in expected-firecracker-unit-file-uid \
  expected-firecracker-unit-file-gid expected-launcher-socket-uid \
  expected-launcher-socket-gid; do
  [[ "${value[$key]}" =~ ^[0-9]+$ ]] || fail "numeric_contract_invalid" 2
done
[[ "$firecracker_root_gate_ref" =~ ^[A-Za-z0-9][A-Za-z0-9:._/-]{0,199}$ ]] ||
  fail "firecracker_root_gate_ref_invalid" 2
[[ "${value[expected-primary-max-concurrent-runs]}" =~ ^[1-9][0-9]*$ ]] &&
  [ "${value[expected-primary-max-concurrent-runs]}" -eq 16 ] ||
  fail "primary_concurrency_not_bound_to_physical_receipt" 2
[[ "${value[expected-rollback-max-concurrent-runs]}" =~ ^[1-9][0-9]*$ ]] &&
  [ "${value[expected-rollback-max-concurrent-runs]}" -le 64 ] ||
  fail "rollback_concurrency_invalid" 2
for timeout in "$readiness_timeout" "$collection_timeout"; do
  [[ "$timeout" =~ ^[1-9][0-9]{0,2}$ ]] && [ "$timeout" -le 300 ] ||
    fail "timeout_invalid" 2
done

canonical_existing() {
  local candidate="$1" canonical
  [ "${candidate#/}" != "$candidate" ] &&
    [[ "$candidate" != *$'\n'* ]] && [[ "$candidate" != *$'\r'* ]] &&
    [ ! -L "$candidate" ] || return 1
  canonical="$(realpath -e -- "$candidate" 2>/dev/null)" || return 1
  [ "$canonical" = "$candidate" ]
}

safe_absolute_future_path() {
  local candidate="$1"
  [ "${candidate#/}" != "$candidate" ] &&
    [ "$candidate" = "${candidate%/}" ] &&
    [[ "$candidate" != *$'\n'* ]] && [[ "$candidate" != *$'\r'* ]] &&
    [[ "$candidate" != *"//"* ]] && [[ "$candidate" != *"/../"* ]] &&
    [[ "$candidate" != */.. ]] && [[ "$candidate" != *"/./"* ]] &&
    [[ "$candidate" != */. ]]
}

trusted_owner() {
  local owner
  owner="$(stat -Lc '%u' -- "$1" 2>/dev/null)" || return 1
  [ "$owner" = "$current_uid" ] || [ "$owner" = 0 ]
}

trusted_directory() {
  local mode
  canonical_existing "$1" && [ -d "$1" ] && trusted_owner "$1" || return 1
  mode="$(stat -Lc '%a' -- "$1")"
  [ $((8#$mode & 8#022)) -eq 0 ]
}

private_directory() {
  canonical_existing "$1" && [ -d "$1" ] &&
    [ "$(stat -Lc '%u:%a' -- "$1")" = "$current_uid:700" ]
}

trusted_executable() {
  local mode
  canonical_existing "$1" && [ -f "$1" ] && [ -x "$1" ] &&
    trusted_owner "$1" || return 1
  mode="$(stat -Lc '%a' -- "$1")"
  [ $((8#$mode & 8#022)) -eq 0 ]
}

private_file() {
  canonical_existing "$1" && [ -f "$1" ] &&
    [ "$(stat -Lc '%u:%h' -- "$1")" = "$current_uid:1" ] &&
    case "$(stat -Lc '%a' -- "$1")" in 400|600) true ;; *) false ;; esac
}

hash_file() {
  sha256sum -- "$1" | awk '{print $1}'
}

require_hash() {
  [ "$(hash_file "$1" 2>/dev/null)" = "$2" ] || fail "${3}_drift"
}

validate_paths_and_hashes() {
  local path key
  for path in "$repository_root" "$proc_root" "$cgroup_mount"; do
    trusted_directory "$path" || fail "trusted_directory_invalid"
  done
  for path in "$runtime_base" "$source_codex_home" "$backup_dir" \
    "$promotion_state_dir"; do
    private_directory "$path" || fail "private_directory_invalid"
  done
  for path in "$git_command" "$go_command" "$profile_script" \
    "$systemd_adapter" "$promotion_helper" "$bubblewrap" "$systemctl_command" \
    "$systemd_run_command" "$firecracker_probe"; do
    trusted_executable "$path" || fail "trusted_executable_invalid"
  done
  if [ "$(stat -Lc '%u:%a:%h' -- "$binary")" != "$current_uid:500:1" ] ||
    ! canonical_existing "$binary"; then
    fail "binary_not_private"
  fi
  for path in "$primary_config" "$rollback_config" "$sqlite_db"; do
    private_file "$path" || fail "private_file_invalid"
  done
  safe_absolute_future_path "$expected_cgroup_root" ||
    fail "cgroup_root_invalid"
  [ "$backup_dir" = "$(dirname "$sqlite_db")/backups" ] ||
    fail "backup_dir_not_bound_to_sqlite"
  IFS=: read -r -a exec_directories <<<"$exec_path"
  [ "${#exec_directories[@]}" -gt 0 ] || fail "exec_path_invalid"
  for path in "${exec_directories[@]}"; do
    if [ -z "$path" ] || ! trusted_directory "$path"; then
      fail "exec_path_invalid"
    fi
  done
  for key in "${SHA_KEYS[@]}"; do
    case "$key" in
      expected-git-sha256) path="$git_command" ;;
      expected-go-sha256) path="$go_command" ;;
      expected-profile-script-sha256) path="$profile_script" ;;
      expected-systemd-adapter-sha256) path="$systemd_adapter" ;;
      expected-promotion-helper-sha256) path="$promotion_helper" ;;
      expected-binary-sha256) path="$binary" ;;
      expected-primary-config-sha256) path="$primary_config" ;;
      expected-rollback-config-sha256) path="$rollback_config" ;;
      expected-bubblewrap-sha256) path="$bubblewrap" ;;
      expected-systemctl-sha256) path="$systemctl_command" ;;
      expected-systemd-run-sha256) path="$systemd_run_command" ;;
      expected-firecracker-probe-sha256) path="$firecracker_probe" ;;
      expected-sqlite-upgrade-receipt-sha256) path="$sqlite_upgrade_receipt" ;;
      *) continue ;;
    esac
    require_hash "$path" "${value[$key]}" "${key#expected-}"
  done
}

EFFECTIVE_CONFIG_PATH=""
validate_subject() {
  local output
  validate_paths_and_hashes
  [ -z "$("$git_command" -C "$repository_root" status \
    --porcelain=v1 --untracked-files=normal)" ] || fail "repository_not_clean"
  [ "$("$git_command" -C "$repository_root" rev-parse HEAD)" = \
    "$repository_head" ] || fail "repository_head_drift"
  [ "$("$git_command" -C "$repository_root" rev-parse \
    "$expected_upstream_ref")" = "$repository_head" ] ||
    fail "repository_not_synchronized"
  "$git_command" -C "$repository_root" merge-base --is-ancestor \
    "$candidate_revision" "$repository_head" ||
    fail "candidate_revision_not_ancestor"
  output="$("$go_command" version -m "$binary")" ||
    fail "binary_build_info_unavailable"
  if ! {
    grep -Fqx $'\tpath\torquesta/cmd/orquesta' <<<"$output" &&
    grep -Fqx $'\tbuild\t-trimpath=true' <<<"$output" &&
    grep -Fqx $'\tbuild\tCGO_ENABLED=0' <<<"$output" &&
    grep -Fqx $'\tbuild\tvcs.revision='"$candidate_revision" <<<"$output" &&
    grep -Fqx $'\tbuild\tvcs.modified=false' <<<"$output"
  }; then
    fail "binary_subject_invalid"
  fi
  EFFECTIVE_CONFIG_PATH="$(
    "$promotion_helper" configs \
      --primary "$primary_config" --rollback "$rollback_config" \
      --sqlite "$sqlite_db" --cgroup "$expected_cgroup_root" \
      --launcher-socket "$launcher_socket" \
      --asset-digest "${value[expected-asset-digest]}" \
      --primary-concurrency "${value[expected-primary-max-concurrent-runs]}" \
      --rollback-concurrency "${value[expected-rollback-max-concurrent-runs]}" \
      --bubblewrap "$bubblewrap" --repository "$repository_root" \
      --target-ref "$expected_config_target_ref" --profile "$profile"
  )" || fail "config_pair_invalid"
  "$promotion_helper" upgrade-receipt --path "$sqlite_upgrade_receipt" \
    --sha "${value[expected-sqlite-upgrade-receipt-sha256]}" \
    --owner-uid "$current_uid" \
    --binary "$binary" --binary-sha "${value[expected-binary-sha256]}" \
    --revision "$candidate_revision" \
    --profile-sha "${value[expected-profile-script-sha256]}" \
    --adapter-sha "${value[expected-systemd-adapter-sha256]}" \
    --systemctl-sha "${value[expected-systemctl-sha256]}" \
    --systemd-run-sha "${value[expected-systemd-run-sha256]}" ||
    fail "sqlite_upgrade_receipt_invalid"
}

verify_firecracker_gate() {
  local output line name field expected_unit firecracker_fragment
  declare -A property=()
  expected_unit="orquesta-firecracker-attestor-${value[expected-firecracker-unit-sha256]}.service"
  [ "$firecracker_unit" = "$expected_unit" ] ||
    fail "firecracker_unit_invalid"
  output="$("$systemctl_command" show "$firecracker_unit" \
    "--property=$FIRECRACKER_UNIT_PROPERTIES")" ||
    fail "firecracker_unit_show_failed"
  while IFS= read -r line; do
    [[ "$line" == *=* ]] || continue
    name="${line%%=*}"
    property["$name"]="${line#*=}"
  done <<<"$output"
  for field in "LoadState=loaded" "ActiveState=active" "SubState=running" \
    "UnitFileState=enabled" "Result=success" "NeedDaemonReload=no"; do
    name="${field%%=*}"
    [ "${property[$name]:-}" = "${field#*=}" ] ||
      fail "firecracker_unit_not_ready"
  done
  [[ "${property[MainPID]:-}" =~ ^[1-9][0-9]*$ ]] &&
    [[ "${property[InvocationID]:-}" =~ ^[0-9a-f]{32}$ ]] ||
    fail "firecracker_unit_identity_invalid"
  firecracker_fragment="${property[FragmentPath]:-}"
  canonical_existing "$firecracker_fragment" &&
    [ "$(stat -Lc '%u:%g:%a:%h' -- "$firecracker_fragment")" = \
      "${value[expected-firecracker-unit-file-uid]}:${value[expected-firecracker-unit-file-gid]}:644:1" ] &&
    [ "$(hash_file "$firecracker_fragment")" = \
      "${value[expected-firecracker-unit-sha256]}" ] ||
    fail "firecracker_unit_file_identity_invalid"
  canonical_existing "$launcher_socket" && [ -S "$launcher_socket" ] &&
    [ "$(stat -Lc '%u:%g:%a:%h' -- "$launcher_socket")" = \
      "${value[expected-launcher-socket-uid]}:${value[expected-launcher-socket-gid]}:660:1" ] ||
    fail "firecracker_socket_unsafe"
  "$firecracker_probe" --socket "$launcher_socket" \
    --trusted-uid 0 --trusted-gid 0 >/dev/null ||
    fail "firecracker_socket_probe_failed"
}

declare -A USER_UNIT=()
OBS_MAIN_PID=""
OBS_DAEMON_PID=""
OBS_DAEMON_START_REF=""
OBS_DAEMON_BINARY_ID=""
OBS_INVOCATION_ID=""
OBS_LIVE_BINARY_SHA=""
OBS_LIVE_CONFIG_SHA=""
OBS_PROCESS_CGROUP=""
OBS_CONTROL_CGROUP=""
load_user_unit() {
  local output line name
  output="$("$systemctl_command" --user show "$unit" \
    "--property=$USER_UNIT_PROPERTIES")" || fail "user_unit_show_failed"
  USER_UNIT=()
  while IFS= read -r line; do
    [[ "$line" == *=* ]] || continue
    name="${line%%=*}"
    USER_UNIT["$name"]="${line#*=}"
  done <<<"$output"
  [ -n "${USER_UNIT[LoadState]:-}" ] || fail "user_unit_state_missing"
}

read_process_cgroup() {
  local content
  [ -f "$proc_root/$1/cgroup" ] && [ ! -L "$proc_root/$1/cgroup" ] ||
    return 1
  content="$(<"$proc_root/$1/cgroup")"
  [[ "$content" == 0::/* ]] && [[ "$content" != *$'\n'* ]] || return 1
  printf '%s' "${content#0::}"
}

process_start_ref() {
  local text tail
  local -a fields
  [ -f "$proc_root/$1/stat" ] && [ ! -L "$proc_root/$1/stat" ] || return 1
  text="$(<"$proc_root/$1/stat")"
  [[ "$text" == *") "* ]] || return 1
  tail="${text##*) }"
  read -r -a fields <<<"$tail"
  [ "${#fields[@]}" -gt 19 ] || return 1
  printf '%s' "${fields[19]}"
}

read_identity() {
  private_file "$1" || return 1
  local result
  result="$(<"$1")"
  [ -n "$result" ] && [[ "$result" != *$'\n'* ]] || return 1
  printf '%s' "$result"
}

require_live_unit_contract() {
  local field name main_pid main_executable expected_sleep control_group
  local process_cgroup parent control controller
  for field in "LoadState=loaded" "ActiveState=active" "SubState=running" \
    "Result=success" "ExecMainCode=0" "ExecMainStatus=0" "Type=exec" \
    "CollectMode=inactive-or-failed" "Delegate=yes" \
    "DelegateSubgroup=$CONTROL_GROUP_NAME" "MemoryAccounting=yes" \
    "TasksAccounting=yes" "KillMode=mixed" "SendSIGKILL=yes" \
    "TimeoutStopUSec=30s" "Restart=no" "LimitNOFILE=32768" \
    "LimitNOFILESoft=32768" "LimitFSIZE=536870912" \
    "LimitFSIZESoft=536870912" "LimitAS=4294967296" \
    "LimitASSoft=4294967296" "LimitCPU=900" "LimitCPUSoft=900" \
    "TasksMax=150992" "RuntimeMaxUSec=infinity" \
    "CPUQuotaPerSecUSec=infinity" "MemoryMax=infinity" \
    "MemorySwapMax=infinity"; do
    name="${field%%=*}"
    [ "${USER_UNIT[$name]:-}" = "${field#*=}" ] ||
      fail "live_unit_contract_invalid"
  done
  [[ "${USER_UNIT[InvocationID]:-}" =~ ^[0-9a-f]{32}$ ]] ||
    fail "live_unit_invocation_invalid"
  main_pid="${USER_UNIT[MainPID]:-}"
  [[ "$main_pid" =~ ^[1-9][0-9]*$ ]] ||
    fail "live_unit_main_pid_invalid"
  main_executable="$(realpath -e -- "$proc_root/$main_pid/exe")" ||
    fail "live_unit_main_executable_unavailable"
  expected_sleep="$(realpath -e -- /usr/bin/sleep)" ||
    fail "sleep_executable_unavailable"
  [ "$main_executable" = "$expected_sleep" ] ||
    fail "live_unit_main_executable_invalid"
  control_group="${USER_UNIT[ControlGroup]:-}"
  [ "${control_group##*/}" = "$unit" ] || fail "live_unit_cgroup_invalid"
  process_cgroup="$control_group/$CONTROL_GROUP_NAME"
  [ "$(read_process_cgroup "$main_pid")" = "$process_cgroup" ] ||
    fail "live_unit_main_cgroup_invalid"
  parent="$cgroup_mount/${control_group#/}"
  control="$parent/$CONTROL_GROUP_NAME"
  [ "$parent" = "$expected_cgroup_root" ] &&
    [ "$(stat -Lc '%u:%a' -- "$parent")" = "$current_uid:700" ] &&
    [ -z "$(tr -d '[:space:]' <"$parent/cgroup.procs")" ] ||
    fail "live_unit_delegation_invalid"
  for controller in cpu memory pids; do
    grep -qw "$controller" "$parent/cgroup.controllers" ||
      fail "live_unit_controller_missing"
  done
  if [ "$(xargs <"$parent/cgroup.subtree_control")" != "cpu memory pids" ] ||
    ! grep -qx "$main_pid" "$control/cgroup.procs"; then
    fail "live_unit_controller_contract_invalid"
  fi
  OBS_INVOCATION_ID="${USER_UNIT[InvocationID]}"
  OBS_MAIN_PID="$main_pid"
  OBS_PROCESS_CGROUP="$process_cgroup"
  OBS_CONTROL_CGROUP="$control"
}

readonly runtime_root="$runtime_base/$profile"
readonly run_root="$runtime_root/run"
readonly pid_file="$run_root/server.pid"
readonly start_ref_file="$run_root/server.start_ref"
readonly binary_id_file="$run_root/server.binary_id"
readonly binary_sha_file="$run_root/server.binary_sha256"
readonly config_sha_file="$run_root/server.config_sha256"

observe_live_profile_without_config_target() {
  local pid start_ref binary_id executable
  load_user_unit
  require_live_unit_contract
  pid="$(read_identity "$pid_file")" || fail "live_profile_pid_invalid"
  start_ref="$(read_identity "$start_ref_file")" ||
    fail "live_profile_start_ref_invalid"
  binary_id="$(read_identity "$binary_id_file")" ||
    fail "live_profile_binary_id_invalid"
  OBS_LIVE_BINARY_SHA="$(read_identity "$binary_sha_file")" ||
    fail "live_profile_binary_sha_invalid"
  OBS_LIVE_CONFIG_SHA="$(read_identity "$config_sha_file")" ||
    fail "live_profile_config_sha_invalid"
  [[ "$pid" =~ ^[1-9][0-9]*$ ]] &&
    [ "$(process_start_ref "$pid")" = "$start_ref" ] &&
    [ "$(stat -Lc '%d:%i' -- "$proc_root/$pid/exe")" = "$binary_id" ] ||
    fail "live_profile_identity_mismatch"
  executable="$(realpath -e -- "$proc_root/$pid/exe")" ||
    fail "live_profile_executable_unavailable"
  if ! trusted_executable "$executable" ||
    [ "$(hash_file "$executable")" != "$OBS_LIVE_BINARY_SHA" ] ||
    [ "$(read_process_cgroup "$pid")" != "$OBS_PROCESS_CGROUP" ] ||
    ! grep -qx "$pid" "$OBS_CONTROL_CGROUP/cgroup.procs"; then
    fail "live_profile_process_contract_invalid"
  fi
  OBS_DAEMON_PID="$pid"
  OBS_DAEMON_START_REF="$start_ref"
  OBS_DAEMON_BINARY_ID="$binary_id"
}

contract_sha="$(
  {
    printf '%s\0' "orquesta_profile_systemd_promotion_contract.v1"
    for key in "${REQUIRED_KEYS[@]}"; do
      printf '%s\0%s\0' "$key" "${value[$key]}"
    done
    printf '%s\0%s\0' "$readiness_timeout" "$collection_timeout"
  } | sha256sum | awk '{print $1}'
)"
readonly contract_sha
readonly backup_path="$backup_dir/orquesta-pre-promotion-${contract_sha:0:16}.sqlite"
readonly backup_receipt="$backup_dir/orquesta-pre-promotion-${contract_sha:0:16}.receipt"
readonly snapshot_file="$promotion_state_dir/pre-stop.snapshot"
readonly final_receipt="$promotion_state_dir/promotion.receipt"

phase_file() { printf '%s/phase.%s\n' "$promotion_state_dir" "$1"; }
phase_content() {
  printf 'schema=%s\ncontract_sha256=%s\nphase=%s\n' \
    "$PHASE_SCHEMA" "$contract_sha" "$1"
}

private_atomic_record() {
  local target="$1" content="$2"
  "$promotion_helper" publish-record --path "$target" \
    --directory "$promotion_state_dir" --content "$content" ||
    fail "private_record_publish_failed"
}

phase_number=0
detect_phase() {
  local name path
  phase_number=0
  for name in 10-preflight 20-profile-stopped 30-backup-published \
    40-unit-collected 50-primary-start-attempted 60-primary-ready \
    70-rollback-started 80-rollback-ready 90-complete; do
    path="$(phase_file "$name")"
    if [ -e "$path" ]; then
      private_file "$path" &&
        [ "$(<"$path")" = "$(phase_content "$name")" ] ||
        fail "phase_marker_invalid"
      phase_number="${name%%-*}"
    fi
  done
  [ "$phase_number" -lt 20 ] || [ -e "$(phase_file 10-preflight)" ] ||
    fail "phase_dependency_missing"
  [ "$phase_number" -lt 30 ] || [ -e "$(phase_file 20-profile-stopped)" ] ||
    fail "phase_dependency_missing"
  [ "$phase_number" -lt 40 ] || [ -e "$(phase_file 30-backup-published)" ] ||
    fail "phase_dependency_missing"
  if [ "$phase_number" -ge 70 ]; then
    [ -e "$(phase_file 40-unit-collected)" ] ||
      fail "phase_dependency_missing"
  elif [ "$phase_number" -ge 60 ]; then
    [ -e "$(phase_file 50-primary-start-attempted)" ] ||
      fail "phase_dependency_missing"
  elif [ "$phase_number" -ge 50 ]; then
    [ -e "$(phase_file 40-unit-collected)" ] ||
      fail "phase_dependency_missing"
  fi
}

write_phase() {
  local name="$1" content
  content="$(phase_content "$name")"
  private_atomic_record "$(phase_file "$name")" "${content%$'\n'}"
  phase_number="${name%%-*}"
}

snapshot_content() {
  printf '%s\n' \
    "schema=orquesta_profile_systemd_promotion_pre_stop.v1" \
    "contract_sha256=$contract_sha" "unit=$unit" \
    "invocation_id=$OBS_INVOCATION_ID" "main_pid=$OBS_MAIN_PID" \
    "daemon_pid=$OBS_DAEMON_PID" \
    "daemon_start_ref=$OBS_DAEMON_START_REF" \
    "daemon_binary_id=$OBS_DAEMON_BINARY_ID" \
    "live_binary_sha256=$OBS_LIVE_BINARY_SHA" \
    "live_config_sha256=$OBS_LIVE_CONFIG_SHA" \
    "process_cgroup=$OBS_PROCESS_CGROUP" \
    "control_cgroup=$OBS_CONTROL_CGROUP" \
    "candidate_binary_sha256=${value[expected-binary-sha256]}"
}

write_snapshot() {
  local content
  content="$(snapshot_content)"
  private_atomic_record "$snapshot_file" "$content"
}

verify_snapshot() {
  local expected_process_cgroup
  local -a lines=()
  expected_process_cgroup="/${expected_cgroup_root#"$cgroup_mount/"}"
  expected_process_cgroup="$expected_process_cgroup/$CONTROL_GROUP_NAME"
  private_file "$snapshot_file" || fail "pre_stop_snapshot_invalid"
  mapfile -t lines <"$snapshot_file"
  if [ "${#lines[@]}" -ne 13 ] ||
    [ "${lines[0]}" != \
      "schema=orquesta_profile_systemd_promotion_pre_stop.v1" ] ||
    [ "${lines[1]}" != "contract_sha256=$contract_sha" ] ||
    [ "${lines[2]}" != "unit=$unit" ] ||
    ! [[ "${lines[3]#invocation_id=}" =~ ^[0-9a-f]{32}$ ]] ||
    ! [[ "${lines[4]#main_pid=}" =~ ^[1-9][0-9]*$ ]] ||
    ! [[ "${lines[5]#daemon_pid=}" =~ ^[1-9][0-9]*$ ]] ||
    ! [[ "${lines[6]#daemon_start_ref=}" =~ ^[1-9][0-9]*$ ]] ||
    ! [[ "${lines[7]#daemon_binary_id=}" =~ ^[0-9]+:[0-9]+$ ]] ||
    ! [[ "${lines[8]#live_binary_sha256=}" =~ ^[0-9a-f]{64}$ ]] ||
    ! [[ "${lines[9]#live_config_sha256=}" =~ ^[0-9a-f]{64}$ ]] ||
    [ "${lines[10]}" != "process_cgroup=$expected_process_cgroup" ] ||
    [ "${lines[11]}" != \
      "control_cgroup=$expected_cgroup_root/$CONTROL_GROUP_NAME" ] ||
    [ "${lines[12]}" != \
      "candidate_binary_sha256=${value[expected-binary-sha256]}" ]; then
    fail "pre_stop_snapshot_invalid"
  fi
}

verify_snapshot_matches_live() {
  local observed
  observed="$(snapshot_content)"
  [ "$(<"$snapshot_file")" = "$observed" ] ||
    fail "pre_stop_snapshot_live_mismatch"
}

declare -a ADAPTER_ARGS=()
adapter_args() {
  local config="$1" config_sha="$2"
  ADAPTER_ARGS=(
    --unit "$unit" --profile "$profile" --repository-root "$repository_root"
    --profile-script "$profile_script"
    --expected-profile-script-sha256 "${value[expected-profile-script-sha256]}"
    --runtime-base "$runtime_base" --source-codex-home "$source_codex_home"
    --binary "$binary"
    --expected-binary-sha256 "${value[expected-binary-sha256]}"
    --config "$config" --expected-config-sha256 "$config_sha"
    --exec-path "$exec_path" --systemctl "$systemctl_command"
    --expected-systemctl-sha256 "${value[expected-systemctl-sha256]}"
    --systemd-run "$systemd_run_command"
    --expected-systemd-run-sha256 "${value[expected-systemd-run-sha256]}"
    --proc-root "$proc_root" --cgroup-mount "$cgroup_mount"
    --readiness-timeout "$readiness_timeout"
    --collection-timeout "$collection_timeout"
  )
}

adapter() {
  local operation="$1" config="$2" config_sha="$3"
  adapter_args "$config" "$config_sha"
  if [ "$operation" = check ]; then
    "$systemd_adapter" "${ADAPTER_ARGS[@]}"
  else
    "$systemd_adapter" --apply "$operation" "${ADAPTER_ARGS[@]}"
  fi
}

profile_stopped() {
  local output status
  set +e
  output="$("$profile_script" status --profile "$profile" \
    --runtime-base "$runtime_base" 2>&1)"
  status="$?"
  set -e
  [ "$status" -eq 3 ] &&
    [ "$output" = "orquesta_profile_server: status=stopped profile=$profile" ]
}

verify_candidate() {
  local kind="$1" config config_sha provider concurrency output
  if [ "$kind" = primary ]; then
    config="$primary_config"
    config_sha="${value[expected-primary-config-sha256]}"
    provider=microvm
    concurrency="${value[expected-primary-max-concurrent-runs]}"
  else
    config="$rollback_config"
    config_sha="${value[expected-rollback-config-sha256]}"
    provider=bubblewrap
    concurrency="${value[expected-rollback-max-concurrent-runs]}"
  fi
  adapter check "$config" "$config_sha" >/dev/null ||
    fail "candidate_unit_invalid"
  output="$("$profile_script" status --profile "$profile" \
    --runtime-base "$runtime_base")" ||
    fail "profile_status_not_running"
  [[ "$output" == "orquesta_profile_server: status=running profile=$profile pid="* ]] ||
    fail "profile_status_not_running"
  "$promotion_helper" effective --path "$EFFECTIVE_CONFIG_PATH" \
    --provider "$provider" --concurrency "$concurrency" --sqlite "$sqlite_db" \
    --cgroup "$expected_cgroup_root" --launcher-socket "$launcher_socket" \
    --asset-digest "${value[expected-asset-digest]}" ||
    fail "effective_config_invalid"
  "$promotion_helper" sqlite-after --path "$sqlite_db" \
    --repository "$repository_root" || fail "sqlite_post_start_invalid"
}

candidate_running() {
  local kind="$1"
  if [ "$kind" = primary ]; then
    adapter check "$primary_config" \
      "${value[expected-primary-config-sha256]}" >/dev/null 2>&1
  else
    adapter check "$rollback_config" \
      "${value[expected-rollback-config-sha256]}" >/dev/null 2>&1
  fi
}

preflight_against_live() {
  local output status reasons
  set +e
  output="$("$profile_script" start --profile "$profile" \
    --runtime-base "$runtime_base" --source-codex-home "$source_codex_home" \
    --binary "$binary" --config "$primary_config" \
    --exec-path "$exec_path" 2>&1)"
  status="$?"
  set -e
  reasons="$(printf '%s\n' "$output" |
    grep -o 'reason_code=[^[:space:]]*' || true)"
  [ "$status" -ne 0 ] &&
    [ "$reasons" = reason_code=daemon_already_running_different_contract ] ||
    fail "live_preflight_unexpected"
}

stop_live_profile() {
  local output
  output="$("$profile_script" stop --profile "$profile" \
    --runtime-base "$runtime_base")" || fail "profile_stop_failed"
  if [ "$output" != \
    "orquesta_profile_server: status=stopped profile=$profile" ] ||
    [ -e "$proc_root/$OBS_DAEMON_PID/exe" ] || [ -e "$pid_file" ] ||
    ! profile_stopped; then
    fail "profile_stop_not_confirmed"
  fi
}

ensure_no_sqlite_writer() {
  local fd target fdinfo flags access
  shopt -s nullglob
  for fd in "$proc_root"/[1-9][0-9]*/fd/[0-9]*; do
    target="$(readlink "$fd" 2>/dev/null || true)"
    case "$target" in "$sqlite_db"|"$sqlite_db-wal"|"$sqlite_db-shm") ;; *) continue ;; esac
    fdinfo="${fd%/fd/*}/fdinfo/${fd##*/}"
    flags="$(awk '$1=="flags:" {print $2}' "$fdinfo" 2>/dev/null)"
    [[ "$flags" =~ ^[0-7]+$ ]] || fail "sqlite_open_mode_invalid"
    access=$((8#$flags & 3))
    [ "$access" -eq 0 ] || fail "sqlite_writer_present"
  done
  shopt -u nullglob
}

verify_backup() {
  "$promotion_helper" verify-backup --target "$backup_path" \
    --receipt "$backup_receipt" --contract "$contract_sha" >/dev/null ||
    fail "sqlite_backup_evidence_invalid"
}

collect_unit() {
  adapter collect "$primary_config" \
    "${value[expected-primary-config-sha256]}" >/dev/null
  load_user_unit
  [ "${USER_UNIT[LoadState]}" = not-found ] || fail "unit_not_collected"
}

prepare_rollback_unit() {
  local output status
  set +e
  output="$("$profile_script" stop --profile "$profile" \
    --runtime-base "$runtime_base" 2>&1)"
  status="$?"
  set -e
  [ "$status" -eq 0 ] &&
    [ "$output" = "orquesta_profile_server: status=stopped profile=$profile" ] ||
    fail "rollback_profile_stop_failed"
  adapter collect "$rollback_config" \
    "${value[expected-rollback-config-sha256]}" >/dev/null
}

write_final_receipt() {
  local result="$1" config_sha active backup_sha content
  verify_backup
  backup_sha="$("$promotion_helper" verify-backup --target "$backup_path" \
    --receipt "$backup_receipt" --contract "$contract_sha")"
  if [ "$result" = primary ]; then
    config_sha="${value[expected-primary-config-sha256]}"
    active=true
  else
    config_sha="${value[expected-rollback-config-sha256]}"
    active=false
  fi
  content="$(
    printf '%s\n' "schema=$RECEIPT_SCHEMA" "contract_sha256=$contract_sha" \
      "result=$result" "binary_sha256=${value[expected-binary-sha256]}" \
      "config_sha256=$config_sha" "backup_sha256=$backup_sha" \
      "sqlite_user_version=19" "firecracker_config_active=$active" \
      "firecracker_root_gate_ref=$firecracker_root_gate_ref" \
      "firecracker_root_evidence_scope=operator_reference_only" \
      "firecracker_application_attestation=pending" \
      "backup_restore_performed=false"
  )"
  private_atomic_record "$final_receipt" "$content"
  write_phase 90-complete
}

read_final_result() {
  local backup_sha
  verify_backup
  backup_sha="$("$promotion_helper" verify-backup --target "$backup_path" \
    --receipt "$backup_receipt" --contract "$contract_sha")" ||
    fail "sqlite_backup_invalid"
  "$promotion_helper" final-receipt --path "$final_receipt" \
    --owner-uid "$current_uid" --contract "$contract_sha" \
    --binary-sha "${value[expected-binary-sha256]}" \
    --primary-config-sha "${value[expected-primary-config-sha256]}" \
    --rollback-config-sha "${value[expected-rollback-config-sha256]}" \
    --backup-sha "$backup_sha" --root-gate-ref "$firecracker_root_gate_ref" ||
    fail "final_receipt_invalid"
}

validate_subject
detect_phase
[ "$phase_number" -eq 0 ] || verify_snapshot

if [ "$mode" = check ]; then
  if [ "$phase_number" -eq 0 ]; then
    verify_firecracker_gate
    observe_live_profile_without_config_target
    "$promotion_helper" sqlite-before --path "$sqlite_db" ||
      fail "sqlite_pre_cut_invalid"
    printf '%s: status=ready action=check phase=preflight profile=%s unit=%s\n' \
      "$PROGRAM" "$profile" "$unit"
  elif [ "$phase_number" -eq 90 ]; then
    result="$(read_final_result)"
    [ "$result" != primary ] || verify_firecracker_gate
    verify_candidate "$result"
    printf '%s: status=complete action=check mode=%s profile=%s unit=%s\n' \
      "$PROGRAM" "$result" "$profile" "$unit"
  else
    [ "$phase_number" -lt 30 ] || verify_backup
    printf '%s: status=resumable action=check phase=%s profile=%s unit=%s\n' \
      "$PROGRAM" "$phase_number" "$profile" "$unit"
  fi
  exit 0
fi

readonly lock_file="$promotion_state_dir/promotion.lock"
if [ ! -e "$lock_file" ]; then
  (umask 077; : >"$lock_file") || fail "promotion_lock_create_failed"
  chmod 600 "$lock_file"
fi
private_file "$lock_file" || fail "promotion_lock_unsafe"
exec 9<>"$lock_file"
flock -n 9 || fail "promotion_already_running"
detect_phase

if [ "$phase_number" -eq 0 ]; then
  verify_firecracker_gate
  observe_live_profile_without_config_target
  "$promotion_helper" sqlite-before --path "$sqlite_db" ||
    fail "sqlite_pre_cut_invalid"
  preflight_against_live
  write_snapshot
  write_phase 10-preflight
fi

if [ "$phase_number" -eq 10 ]; then
  if [ -e "$pid_file" ]; then
    observe_live_profile_without_config_target
    verify_snapshot_matches_live
    stop_live_profile
  else
    profile_stopped || fail "profile_not_stopped_on_reentry"
  fi
  write_phase 20-profile-stopped
fi

if [ "$phase_number" -eq 20 ]; then
  load_user_unit
  require_live_unit_contract
  profile_stopped || fail "profile_not_stopped_before_backup"
  ensure_no_sqlite_writer
  "$promotion_helper" backup --source "$sqlite_db" --target "$backup_path" \
    --directory "$backup_dir" --receipt "$backup_receipt" \
    --contract "$contract_sha" >/dev/null || fail "sqlite_backup_failed"
  write_phase 30-backup-published
fi

if [ "$phase_number" -eq 30 ]; then
  verify_backup
  collect_unit
  write_phase 40-unit-collected
fi

if [ "$phase_number" -eq 40 ]; then
  if (verify_firecracker_gate); then
    write_phase 50-primary-start-attempted
  else
    write_phase 70-rollback-started
  fi
fi

if [ "$phase_number" -eq 50 ]; then
  primary_ok=0
  if candidate_running primary; then
    (verify_candidate primary) && primary_ok=1
  else
    set +e
    primary_output="$(adapter start "$primary_config" \
      "${value[expected-primary-config-sha256]}" 2>&1)"
    primary_status="$?"
    set -e
    if [ "$primary_status" -eq 0 ] && (verify_candidate primary); then
      primary_ok=1
    else
      primary_reason="$(printf '%s\n' "$primary_output" |
        grep -o 'reason_code=[^[:space:]]*' | tail -n 1 || true)"
      [ -n "$primary_reason" ] || primary_reason=reason_code=availability_failed
      printf '%s: status=degraded primary_%s\n' \
        "$PROGRAM" "$primary_reason" >&2
    fi
  fi
  if [ "$primary_ok" -eq 1 ]; then
    write_phase 60-primary-ready
  else
    write_phase 70-rollback-started
  fi
fi

if [ "$phase_number" -eq 60 ]; then
  if (verify_firecracker_gate && verify_candidate primary); then
    write_final_receipt primary
    printf '%s: status=primary_ready mode=primary firecracker_application_attestation=pending profile=%s unit=%s\n' \
      "$PROGRAM" "$profile" "$unit"
    exit 0
  fi
  write_phase 70-rollback-started
fi

if [ "$phase_number" -eq 70 ]; then
  if ! candidate_running rollback; then
    prepare_rollback_unit
    adapter start "$rollback_config" \
      "${value[expected-rollback-config-sha256]}" >/dev/null ||
      fail "rollback_start_failed"
  fi
  verify_candidate rollback
  write_phase 80-rollback-ready
fi

if [ "$phase_number" -eq 80 ]; then
  verify_candidate rollback
  write_final_receipt rollback
  printf '%s: status=available mode=rollback firecracker_config_active=false backup_restore_performed=false profile=%s unit=%s\n' \
    "$PROGRAM" "$profile" "$unit"
  exit 0
fi

if [ "$phase_number" -eq 90 ]; then
  result="$(read_final_result)"
  [ "$result" != primary ] || verify_firecracker_gate
  verify_candidate "$result"
  printf '%s: status=complete mode=%s profile=%s unit=%s\n' \
    "$PROGRAM" "$result" "$profile" "$unit"
  exit 0
fi

fail "promotion_state_unhandled"
