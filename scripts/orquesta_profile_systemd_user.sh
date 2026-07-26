#!/usr/bin/env bash
# Adapta un perfil Orquesta a una unidad systemd --user delegada.
#
# No realiza backup, migraciones, rollback ni atestaciones. Esos gates
# pertenecen al procedimiento de promoción que compone este adaptador.

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

readonly PROGRAM="orquesta_profile_systemd_user"
readonly CONTROL_GROUP_NAME="orquesta-control"
readonly PROFILE_STATUS_PREFIX="orquesta_profile_server:"
readonly READINESS_DEFAULT=30
readonly COLLECTION_DEFAULT=10
readonly SYSTEMD_PROPERTIES=(
  "Delegate=yes"
  "DelegateSubgroup=$CONTROL_GROUP_NAME"
  "MemoryAccounting=yes"
  "TasksAccounting=yes"
  "KillMode=mixed"
  "SendSIGKILL=yes"
  "TimeoutStopSec=30s"
  "Restart=no"
  "LimitNOFILE=32768"
  "LimitFSIZE=536870912"
  "LimitAS=4294967296"
  "LimitCPU=900"
  "TasksMax=150992"
)
readonly SYSTEMD_SHOW_PROPERTIES="LoadState,ActiveState,SubState,Result,ExecMainCode,ExecMainStatus,MainPID,ControlGroup,InvocationID,Type,CollectMode,Delegate,DelegateSubgroup,MemoryAccounting,TasksAccounting,KillMode,SendSIGKILL,TimeoutStopUSec,Restart,LimitNOFILE,LimitNOFILESoft,LimitFSIZE,LimitFSIZESoft,LimitAS,LimitASSoft,LimitCPU,LimitCPUSoft,TasksMax,RuntimeMaxUSec,CPUQuotaPerSecUSec,MemoryMax,MemorySwapMax"

fail() {
  printf '%s: status=error reason_code=%s\n' "$PROGRAM" "$1" >&2
  exit "${2:-1}"
}

usage() {
  cat <<'EOF'
uso:
  orquesta_profile_systemd_user.sh [--check] CONTRATO
  orquesta_profile_systemd_user.sh --apply start CONTRATO
  orquesta_profile_systemd_user.sh --apply stop-profile CONTRATO
  orquesta_profile_systemd_user.sh --apply collect CONTRATO

CONTRATO:
  --unit NOMBRE.service
  --profile NOMBRE
  --repository-root RUTA
  --profile-script RUTA --expected-profile-script-sha256 SHA256
  --runtime-base RUTA
  --source-codex-home RUTA
  --binary RUTA --expected-binary-sha256 SHA256
  --config RUTA --expected-config-sha256 SHA256
  --exec-path PATH
  --systemctl RUTA --expected-systemctl-sha256 SHA256
  --systemd-run RUTA --expected-systemd-run-sha256 SHA256
  --proc-root RUTA
  --cgroup-mount RUTA
  [--readiness-timeout SEGUNDOS]
  [--collection-timeout SEGUNDOS]

Sin --apply la operación es --check y no modifica unidad, perfil ni cgroup.
stop-profile deja la unidad viva para permitir un backup externo con el daemon
ya detenido. collect exige el perfil detenido y recoge solo la unidad exacta.
EOF
}

require_option_value() {
  [ "$#" -ge 2 ] && [ -n "$2" ] || fail "argument_value_missing" 2
}

operation="check"
apply_requested=0
if [ "${1:-}" = "--check" ]; then
  shift
elif [ "${1:-}" = "--apply" ]; then
  [ "$#" -ge 2 ] || fail "apply_operation_missing" 2
  apply_requested=1
  operation="$2"
  shift 2
fi

case "$operation" in
  check) [ "$apply_requested" -eq 0 ] || fail "check_must_be_read_only" 2 ;;
  start|stop-profile|collect)
    [ "$apply_requested" -eq 1 ] || fail "apply_required" 2
    ;;
  *) fail "operation_invalid" 2 ;;
esac

unit=""
profile=""
repository_root=""
profile_script=""
expected_profile_script_sha=""
runtime_base=""
source_codex_home=""
binary=""
expected_binary_sha=""
config=""
expected_config_sha=""
exec_path=""
systemctl_command=""
expected_systemctl_sha=""
systemd_run_command=""
expected_systemd_run_sha=""
proc_root=""
cgroup_mount=""
readiness_timeout="$READINESS_DEFAULT"
collection_timeout="$COLLECTION_DEFAULT"

while [ "$#" -gt 0 ]; do
  option="$1"
  case "$option" in
    --unit)
      require_option_value "$@"
      [ -z "$unit" ] || fail "argument_repeated" 2
      unit="$2"
      shift 2
      ;;
    --profile)
      require_option_value "$@"
      [ -z "$profile" ] || fail "argument_repeated" 2
      profile="$2"
      shift 2
      ;;
    --repository-root)
      require_option_value "$@"
      [ -z "$repository_root" ] || fail "argument_repeated" 2
      repository_root="$2"
      shift 2
      ;;
    --profile-script)
      require_option_value "$@"
      [ -z "$profile_script" ] || fail "argument_repeated" 2
      profile_script="$2"
      shift 2
      ;;
    --expected-profile-script-sha256)
      require_option_value "$@"
      [ -z "$expected_profile_script_sha" ] || fail "argument_repeated" 2
      expected_profile_script_sha="$2"
      shift 2
      ;;
    --runtime-base)
      require_option_value "$@"
      [ -z "$runtime_base" ] || fail "argument_repeated" 2
      runtime_base="$2"
      shift 2
      ;;
    --source-codex-home)
      require_option_value "$@"
      [ -z "$source_codex_home" ] || fail "argument_repeated" 2
      source_codex_home="$2"
      shift 2
      ;;
    --binary)
      require_option_value "$@"
      [ -z "$binary" ] || fail "argument_repeated" 2
      binary="$2"
      shift 2
      ;;
    --expected-binary-sha256)
      require_option_value "$@"
      [ -z "$expected_binary_sha" ] || fail "argument_repeated" 2
      expected_binary_sha="$2"
      shift 2
      ;;
    --config)
      require_option_value "$@"
      [ -z "$config" ] || fail "argument_repeated" 2
      config="$2"
      shift 2
      ;;
    --expected-config-sha256)
      require_option_value "$@"
      [ -z "$expected_config_sha" ] || fail "argument_repeated" 2
      expected_config_sha="$2"
      shift 2
      ;;
    --exec-path)
      require_option_value "$@"
      [ -z "$exec_path" ] || fail "argument_repeated" 2
      exec_path="$2"
      shift 2
      ;;
    --systemctl)
      require_option_value "$@"
      [ -z "$systemctl_command" ] || fail "argument_repeated" 2
      systemctl_command="$2"
      shift 2
      ;;
    --expected-systemctl-sha256)
      require_option_value "$@"
      [ -z "$expected_systemctl_sha" ] || fail "argument_repeated" 2
      expected_systemctl_sha="$2"
      shift 2
      ;;
    --systemd-run)
      require_option_value "$@"
      [ -z "$systemd_run_command" ] || fail "argument_repeated" 2
      systemd_run_command="$2"
      shift 2
      ;;
    --expected-systemd-run-sha256)
      require_option_value "$@"
      [ -z "$expected_systemd_run_sha" ] || fail "argument_repeated" 2
      expected_systemd_run_sha="$2"
      shift 2
      ;;
    --proc-root)
      require_option_value "$@"
      [ -z "$proc_root" ] || fail "argument_repeated" 2
      proc_root="$2"
      shift 2
      ;;
    --cgroup-mount)
      require_option_value "$@"
      [ -z "$cgroup_mount" ] || fail "argument_repeated" 2
      cgroup_mount="$2"
      shift 2
      ;;
    --readiness-timeout)
      require_option_value "$@"
      readiness_timeout="$2"
      shift 2
      ;;
    --collection-timeout)
      require_option_value "$@"
      collection_timeout="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *) fail "argument_unknown" 2 ;;
  esac
done

for required in \
  "$unit" "$profile" "$repository_root" "$profile_script" \
  "$expected_profile_script_sha" "$runtime_base" "$source_codex_home" \
  "$binary" "$expected_binary_sha" "$config" "$expected_config_sha" \
  "$exec_path" "$systemctl_command" "$expected_systemctl_sha" \
  "$systemd_run_command" "$expected_systemd_run_sha" "$proc_root" \
  "$cgroup_mount"; do
  [ -n "$required" ] || fail "contract_incomplete" 2
done

[[ "$profile" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$ ]] ||
  fail "profile_invalid" 2
[[ "$unit" =~ ^orquesta-v[1-9][0-9]*-"$profile"\.service$ ]] ||
  fail "unit_name_invalid" 2
[[ "$expected_profile_script_sha" =~ ^[0-9a-f]{64}$ ]] &&
  [[ "$expected_binary_sha" =~ ^[0-9a-f]{64}$ ]] &&
  [[ "$expected_config_sha" =~ ^[0-9a-f]{64}$ ]] &&
  [[ "$expected_systemctl_sha" =~ ^[0-9a-f]{64}$ ]] &&
  [[ "$expected_systemd_run_sha" =~ ^[0-9a-f]{64}$ ]] ||
  fail "expected_sha256_invalid" 2
[[ "$readiness_timeout" =~ ^[1-9][0-9]{0,2}$ ]] &&
  [ "$readiness_timeout" -le 300 ] ||
  fail "readiness_timeout_invalid" 2
[[ "$collection_timeout" =~ ^[1-9][0-9]{0,2}$ ]] &&
  [ "$collection_timeout" -le 300 ] ||
  fail "collection_timeout_invalid" 2

CURRENT_UID="$(id -u)"
readonly CURRENT_UID

canonical_existing() {
  candidate="$1"
  [ "${candidate#/}" != "$candidate" ] || return 1
  [[ "$candidate" != *$'\n'* ]] && [[ "$candidate" != *$'\r'* ]] || return 1
  [ ! -L "$candidate" ] || return 1
  canonical="$(realpath -e -- "$candidate" 2>/dev/null)" || return 1
  [ "$canonical" = "$candidate" ]
}

owner_is_trusted() {
  observed_owner="$(stat -Lc '%u' -- "$1" 2>/dev/null)" || return 1
  [ "$observed_owner" = "$CURRENT_UID" ] || [ "$observed_owner" = "0" ]
}

not_group_or_world_writable() {
  observed_mode="$(stat -Lc '%a' -- "$1" 2>/dev/null)" || return 1
  [ $((8#$observed_mode & 8#022)) -eq 0 ]
}

trusted_directory() {
  canonical_existing "$1" &&
    [ -d "$1" ] &&
    owner_is_trusted "$1" &&
    not_group_or_world_writable "$1"
}

private_directory() {
  canonical_existing "$1" &&
    [ -d "$1" ] &&
    [ "$(stat -Lc '%u' -- "$1" 2>/dev/null)" = "$CURRENT_UID" ] &&
    [ "$(stat -Lc '%a' -- "$1" 2>/dev/null)" = "700" ]
}

trusted_executable() {
  canonical_existing "$1" &&
    [ -f "$1" ] &&
    [ -x "$1" ] &&
    owner_is_trusted "$1" &&
    not_group_or_world_writable "$1"
}

private_regular_file() {
  canonical_existing "$1" &&
    [ -f "$1" ] &&
    [ "$(stat -Lc '%u' -- "$1" 2>/dev/null)" = "$CURRENT_UID" ] &&
    case "$(stat -Lc '%a' -- "$1" 2>/dev/null)" in
      400|500|600|700) true ;;
      *) false ;;
    esac
}

sha256_of() {
  sha256sum -- "$1" | awk '{print $1}'
}

require_hash() {
  path="$1"
  expected="$2"
  reason="$3"
  observed="$(sha256_of "$path" 2>/dev/null)" || fail "${reason}_unreadable"
  [ "$observed" = "$expected" ] || fail "${reason}_drift"
}

trusted_directory "$repository_root" || fail "repository_root_unsafe"
private_directory "$runtime_base" || fail "runtime_base_not_private"
private_directory "$source_codex_home" || fail "source_codex_home_not_private"
trusted_directory "$proc_root" || fail "proc_root_unsafe"
trusted_directory "$cgroup_mount" || fail "cgroup_mount_unsafe"
trusted_executable "$profile_script" || fail "profile_script_unsafe"
private_regular_file "$binary" && [ -x "$binary" ] ||
  fail "binary_not_private_executable"
private_regular_file "$config" || fail "config_not_private"
trusted_executable "$systemctl_command" || fail "systemctl_unsafe"
trusted_executable "$systemd_run_command" || fail "systemd_run_unsafe"

require_hash "$profile_script" "$expected_profile_script_sha" "profile_script"
require_hash "$binary" "$expected_binary_sha" "binary"
require_hash "$config" "$expected_config_sha" "config"
require_hash "$systemctl_command" "$expected_systemctl_sha" "systemctl"
require_hash "$systemd_run_command" "$expected_systemd_run_sha" "systemd_run"

IFS=':' read -r -a exec_directories <<<"$exec_path"
[ "${#exec_directories[@]}" -gt 0 ] || fail "exec_path_invalid"
for exec_directory in "${exec_directories[@]}"; do
  if [ -z "$exec_directory" ] || ! trusted_directory "$exec_directory"; then
    fail "exec_path_invalid"
  fi
done

readonly RUNTIME_ROOT="$runtime_base/$profile"
readonly RUN_ROOT="$RUNTIME_ROOT/run"
readonly PID_FILE="$RUN_ROOT/server.pid"
readonly BINARY_SHA_FILE="$RUN_ROOT/server.binary_sha256"
readonly CONFIG_SHA_FILE="$RUN_ROOT/server.config_sha256"

declare -A UNIT_PROPERTIES=()

load_unit_properties() {
  local output line name value
  output="$("$systemctl_command" --user show "$unit" \
    "--property=$SYSTEMD_SHOW_PROPERTIES" 2>&1)" ||
    fail "systemctl_show_failed"
  UNIT_PROPERTIES=()
  while IFS= read -r line; do
    [[ "$line" == *=* ]] || continue
    name="${line%%=*}"
    value="${line#*=}"
    UNIT_PROPERTIES["$name"]="$value"
  done <<<"$output"
  [ -n "${UNIT_PROPERTIES[LoadState]:-}" ] || fail "unit_load_state_missing"
}

require_property() {
  name="$1"
  expected="$2"
  [ "${UNIT_PROPERTIES[$name]:-}" = "$expected" ] ||
    fail "unit_property_${name}_invalid"
}

read_private_identity() {
  path="$1"
  private_regular_file "$path" || return 1
  value="$(<"$path")"
  [ -n "$value" ] || return 1
  printf '%s' "$value"
}

read_process_cgroup() {
  pid="$1"
  cgroup_file="$proc_root/$pid/cgroup"
  [ -f "$cgroup_file" ] && [ ! -L "$cgroup_file" ] || return 1
  content="$(<"$cgroup_file")"
  [ "${#content}" -le 4096 ] || return 1
  lines=()
  mapfile -t lines <<<"$content"
  [ "${#lines[@]}" -eq 1 ] && [[ "${lines[0]}" == 0::/* ]] || return 1
  printf '%s' "${lines[0]#0::}"
}

process_executable() {
  pid="$1"
  realpath -e -- "$proc_root/$pid/exe" 2>/dev/null
}

process_is_observable() {
  pid="$1"
  [[ "$pid" =~ ^[1-9][0-9]*$ ]] &&
    [ -d "$proc_root/$pid" ] &&
    [ -r "$proc_root/$pid/cgroup" ] &&
    process_executable "$pid" >/dev/null
}

require_unit_base_contract() {
  require_property "LoadState" "loaded"
  require_property "ActiveState" "active"
  require_property "SubState" "running"
  require_property "Result" "success"
  require_property "ExecMainCode" "0"
  require_property "ExecMainStatus" "0"
  require_property "Type" "exec"
  require_property "CollectMode" "inactive-or-failed"
  require_property "Delegate" "yes"
  require_property "DelegateSubgroup" "$CONTROL_GROUP_NAME"
  require_property "MemoryAccounting" "yes"
  require_property "TasksAccounting" "yes"
  require_property "KillMode" "mixed"
  require_property "SendSIGKILL" "yes"
  require_property "TimeoutStopUSec" "30s"
  require_property "Restart" "no"
  require_property "LimitNOFILE" "32768"
  require_property "LimitNOFILESoft" "32768"
  require_property "LimitFSIZE" "536870912"
  require_property "LimitFSIZESoft" "536870912"
  require_property "LimitAS" "4294967296"
  require_property "LimitASSoft" "4294967296"
  require_property "LimitCPU" "900"
  require_property "LimitCPUSoft" "900"
  require_property "TasksMax" "150992"
  require_property "RuntimeMaxUSec" "infinity"
  require_property "CPUQuotaPerSecUSec" "infinity"
  require_property "MemoryMax" "infinity"
  require_property "MemorySwapMax" "infinity"

  invocation_id="${UNIT_PROPERTIES[InvocationID]:-}"
  [[ "$invocation_id" =~ ^[0-9a-f]{32}$ ]] ||
    fail "unit_invocation_id_invalid"
  main_pid="${UNIT_PROPERTIES[MainPID]:-}"
  process_is_observable "$main_pid" || fail "unit_main_pid_invalid"
  main_executable="$(process_executable "$main_pid")" ||
    fail "unit_main_executable_unavailable"
  expected_sleep="$(realpath -e -- /usr/bin/sleep 2>/dev/null)" ||
    fail "sleep_executable_unavailable"
  [ "$main_executable" = "$expected_sleep" ] ||
    fail "unit_main_executable_invalid"

  control_group="${UNIT_PROPERTIES[ControlGroup]:-}"
  [ "${control_group#/}" != "$control_group" ] &&
    [ "$control_group" = "${control_group%/}" ] &&
    [ "${control_group##*/}" = "$unit" ] ||
    fail "unit_control_group_invalid"
  case "$control_group" in
    *//*|*/../*|*/..|*/./*|*/.) fail "unit_control_group_invalid" ;;
  esac
  expected_process_cgroup="$control_group/$CONTROL_GROUP_NAME"
  [ "$(read_process_cgroup "$main_pid" 2>/dev/null)" = \
    "$expected_process_cgroup" ] || fail "unit_main_cgroup_invalid"

  cgroup_parent="$cgroup_mount/${control_group#/}"
  cgroup_control="$cgroup_parent/$CONTROL_GROUP_NAME"
  [ ! -L "$cgroup_parent" ] && [ ! -L "$cgroup_control" ] &&
    [ -d "$cgroup_parent" ] && [ -d "$cgroup_control" ] ||
    fail "delegated_cgroup_missing"
  [ "$(stat -Lc '%u' -- "$cgroup_parent" 2>/dev/null)" = "$CURRENT_UID" ] &&
    [ "$(stat -Lc '%u' -- "$cgroup_control" 2>/dev/null)" = "$CURRENT_UID" ] ||
    fail "delegated_cgroup_owner_invalid"
  [ "$(stat -Lc '%a' -- "$cgroup_parent" 2>/dev/null)" = "700" ] ||
    fail "delegated_cgroup_mode_invalid"
  [ -f "$cgroup_parent/cgroup.procs" ] &&
    [ -f "$cgroup_parent/cgroup.controllers" ] &&
    [ -f "$cgroup_parent/cgroup.subtree_control" ] &&
    [ -f "$cgroup_control/cgroup.procs" ] ||
    fail "delegated_cgroup_contract_missing"
  [ -z "$(tr -d '[:space:]' <"$cgroup_parent/cgroup.procs")" ] ||
    fail "delegated_cgroup_parent_not_empty"
  for controller in cpu memory pids; do
    grep -qw -- "$controller" "$cgroup_parent/cgroup.controllers" ||
      fail "delegated_cgroup_controller_unavailable"
  done
  [ "$(xargs <"$cgroup_parent/cgroup.subtree_control")" = \
    "cpu memory pids" ] || fail "delegated_cgroup_controllers_invalid"
  grep -qx -- "$main_pid" "$cgroup_control/cgroup.procs" ||
    fail "unit_main_pid_not_in_control_group"
}

require_running_profile_contract() {
  local daemon_pid persisted_binary_sha persisted_config_sha daemon_executable
  require_unit_base_contract
  daemon_pid="$(read_private_identity "$PID_FILE" 2>/dev/null)" ||
    fail "profile_pid_identity_invalid"
  [[ "$daemon_pid" =~ ^[1-9][0-9]*$ ]] ||
    fail "profile_pid_identity_invalid"
  persisted_binary_sha="$(read_private_identity "$BINARY_SHA_FILE" 2>/dev/null)" ||
    fail "profile_binary_identity_invalid"
  persisted_config_sha="$(read_private_identity "$CONFIG_SHA_FILE" 2>/dev/null)" ||
    fail "profile_config_identity_invalid"
  [ "$persisted_binary_sha" = "$expected_binary_sha" ] ||
    fail "profile_binary_hash_drift"
  [ "$persisted_config_sha" = "$expected_config_sha" ] ||
    fail "profile_config_hash_drift"
  process_is_observable "$daemon_pid" || fail "profile_pid_not_alive"
  daemon_executable="$(process_executable "$daemon_pid")" ||
    fail "profile_executable_unavailable"
  [ "$daemon_executable" = "$binary" ] ||
    fail "profile_executable_invalid"
  [ "$(read_process_cgroup "$daemon_pid" 2>/dev/null)" = \
    "$expected_process_cgroup" ] || fail "profile_cgroup_invalid"
  grep -qx -- "$daemon_pid" "$cgroup_control/cgroup.procs" ||
    fail "profile_pid_not_in_control_group"
  observed_daemon_pid="$daemon_pid"
}

unit_load_state() {
  local output
  output="$("$systemctl_command" --user show "$unit" \
    --property=LoadState --value 2>&1)" ||
    fail "systemctl_show_failed"
  output="${output//$'\n'/}"
  [ -n "$output" ] || fail "unit_load_state_missing"
  printf '%s' "$output"
}

wait_until_collected() {
  local deadline state
  deadline=$((SECONDS + collection_timeout))
  while :; do
    state="$(unit_load_state)"
    [ "$state" = "not-found" ] && return 0
    [ "$SECONDS" -lt "$deadline" ] || return 1
    sleep 0.1
  done
}

stop_and_collect_exact_unit() {
  local state
  state="$(unit_load_state)"
  [ "$state" = "not-found" ] && return 0
  "$systemctl_command" --user stop "$unit" >/dev/null ||
    return 1
  wait_until_collected
}

cleanup_failed_start() {
  stop_and_collect_exact_unit || fail "failed_start_cleanup_timeout"
}

case "$operation" in
  check)
    load_unit_properties
    [ "${UNIT_PROPERTIES[LoadState]}" != "not-found" ] ||
      fail "unit_not_found" 3
    require_running_profile_contract
    printf '%s: status=running action=check unit=%s profile=%s main_pid=%s daemon_pid=%s invocation_id=%s\n' \
      "$PROGRAM" "$unit" "$profile" "$main_pid" "$observed_daemon_pid" \
      "$invocation_id"
    ;;

  start)
    [ "$(unit_load_state)" = "not-found" ] ||
      fail "unit_name_conflict"
    # El cuerpo debe llegar literal; los únicos valores variables son argv.
    # shellcheck disable=SC2016
    bash_body='
set -euo pipefail
umask 077
unit="$1"
cgroup_mount="$2"
repository_root="$3"
profile_script="$4"
profile="$5"
runtime_base="$6"
source_codex_home="$7"
binary="$8"
config="$9"
exec_path="${10}"
expected_binary_sha="${11}"
expected_config_sha="${12}"
self_line="$(</proc/self/cgroup)"
[[ "$self_line" == 0::/* ]] || exit 70
self_cgroup="${self_line#0::}"
case "$self_cgroup" in
  */"$unit"/orquesta-control) ;;
  *) exit 71 ;;
esac
parent="$cgroup_mount/${self_cgroup#/}"
parent="${parent%/orquesta-control}"
[ ! -L "$parent" ] && [ -d "$parent" ] || exit 72
[ "$(stat -Lc "%u" -- "$parent")" = "$(id -u)" ] || exit 73
chmod 0700 -- "$parent"
[ -z "$(tr -d "[:space:]" <"$parent/cgroup.procs")" ] || exit 74
for controller in cpu memory pids; do
  grep -qw -- "$controller" "$parent/cgroup.controllers" || exit 75
done
for controller in $(<"$parent/cgroup.subtree_control"); do
  case "$controller" in
    cpu|memory|pids) ;;
    *) printf -- "-%s" "$controller" >"$parent/cgroup.subtree_control" ;;
  esac
done
printf "%s" "+cpu +memory +pids" >"$parent/cgroup.subtree_control"
[ "$(xargs <"$parent/cgroup.subtree_control")" = "cpu memory pids" ] || exit 76
cd -- "$repository_root"
"$profile_script" start \
  --profile "$profile" \
  --runtime-base "$runtime_base" \
  --source-codex-home "$source_codex_home" \
  --binary "$binary" \
  --config "$config" \
  --exec-path "$exec_path"
[ "$(<"$runtime_base/$profile/run/server.binary_sha256")" = "$expected_binary_sha" ] || exit 77
[ "$(<"$runtime_base/$profile/run/server.config_sha256")" = "$expected_config_sha" ] || exit 78
exec /usr/bin/sleep infinity
'
    systemd_run_args=(
      --user
      --expand-environment=no
      --collect
      --service-type=exec
      "--unit=$unit"
    )
    for property in "${SYSTEMD_PROPERTIES[@]}"; do
      systemd_run_args+=("--property=$property")
    done
    systemd_run_args+=(
      /bin/bash -c "$bash_body" "$PROGRAM"
      "$unit" "$cgroup_mount" "$repository_root" "$profile_script" "$profile"
      "$runtime_base" "$source_codex_home" "$binary" "$config" "$exec_path"
      "$expected_binary_sha" "$expected_config_sha"
    )
    "$systemd_run_command" "${systemd_run_args[@]}" >/dev/null ||
      fail "systemd_run_failed"

    readiness_deadline=$((SECONDS + readiness_timeout))
    readiness_reason="unit_readiness_timeout"
    while [ "$SECONDS" -lt "$readiness_deadline" ]; do
      current_load_state="$(unit_load_state)"
      if [ "$current_load_state" = "not-found" ]; then
        fail "systemd_run_false_positive"
      fi
      set +e
      readiness_output="$(
        set -e
        load_unit_properties
        require_running_profile_contract
        printf '%s\n' "$main_pid" "$observed_daemon_pid" "$invocation_id"
      )" 2>/dev/null
      readiness_status="$?"
      set -e
      if [ "$readiness_status" -eq 0 ]; then
        mapfile -t readiness_values <<<"$readiness_output"
        [ "${#readiness_values[@]}" -eq 3 ] || {
          cleanup_failed_start
          fail "unit_readiness_output_invalid"
        }
        printf '%s: status=running action=start unit=%s profile=%s main_pid=%s daemon_pid=%s invocation_id=%s\n' \
          "$PROGRAM" "$unit" "$profile" "${readiness_values[0]}" \
          "${readiness_values[1]}" "${readiness_values[2]}"
        exit 0
      fi
      sleep 0.1
    done
    cleanup_failed_start
    fail "$readiness_reason"
    ;;

  stop-profile)
    load_unit_properties
    require_running_profile_contract
    daemon_pid_before_stop="$observed_daemon_pid"
    stop_output="$("$profile_script" stop \
      --profile "$profile" \
      --runtime-base "$runtime_base" 2>&1)" ||
      fail "profile_stop_failed"
    [[ "$stop_output" == "$PROFILE_STATUS_PREFIX status=stopped profile="* ]] ||
      fail "profile_stop_contract_invalid"
    [ ! -e "$PID_FILE" ] || fail "profile_stop_identity_remaining"
    [ ! -e "$proc_root/$daemon_pid_before_stop/exe" ] ||
      fail "profile_stop_pid_remaining"
    load_unit_properties
    require_unit_base_contract
    printf '%s: status=profile_stopped action=stop-profile unit=%s profile=%s main_pid=%s invocation_id=%s\n' \
      "$PROGRAM" "$unit" "$profile" "$main_pid" "$invocation_id"
    ;;

  collect)
    load_unit_properties
    if [ "${UNIT_PROPERTIES[LoadState]}" = "not-found" ]; then
      printf '%s: status=collected action=collect unit=%s profile=%s\n' \
        "$PROGRAM" "$unit" "$profile"
      exit 0
    fi
    require_unit_base_contract
    if [ -e "$PID_FILE" ]; then
      remaining_pid="$(read_private_identity "$PID_FILE" 2>/dev/null || true)"
      if [ -n "$remaining_pid" ] && process_is_observable "$remaining_pid"; then
        fail "profile_still_running"
      fi
      fail "profile_identity_remaining"
    fi
    "$systemctl_command" --user stop "$unit" >/dev/null ||
      fail "unit_stop_failed"
    wait_until_collected || fail "unit_collection_timeout"
    printf '%s: status=collected action=collect unit=%s profile=%s invocation_id=%s\n' \
      "$PROGRAM" "$unit" "$profile" "$invocation_id"
    ;;
esac
