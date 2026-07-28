#!/usr/bin/env bash
# Lanza una única composición pool de Orquesta como unidad systemd --user.
# No crea PID files, stores, copias de credenciales ni lifecycle paralelo.

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

readonly PROGRAM="orquesta_pool_systemd_user"
readonly CONTROL_GROUP_NAME="orquesta-control"
readonly DESCRIPTION="Orquesta pool server"
readonly SHOW_PROPERTIES="Description,LoadState,ActiveState,SubState,Result,MainPID,ControlGroup,Type,CollectMode,Delegate,DelegateSubgroup,KillMode,SendSIGKILL,Restart"
readonly SYSTEMD_PROPERTIES=(
  "Delegate=yes"
  "DelegateSubgroup=$CONTROL_GROUP_NAME"
  "KillMode=control-group"
  "SendSIGKILL=yes"
  "Restart=no"
)

fail() {
  printf '%s: status=error reason_code=%s\n' "$PROGRAM" "$1" >&2
  exit "${2:-1}"
}

usage() {
  cat <<'EOF'
uso:
  orquesta_pool_systemd_user.sh start CONTRATO
  orquesta_pool_systemd_user.sh stop CONTRATO
  orquesta_pool_systemd_user.sh status CONTRATO

CONTRATO:
  --unit NOMBRE.service
  --repository-root RUTA_ABSOLUTA
  --bootstrap-home RUTA_ABSOLUTA
  --binary RUTA_ABSOLUTA --expected-binary-sha256 SHA256
  --config RUTA_ABSOLUTA --expected-config-sha256 SHA256
  --exec-path PATH_ABSOLUTO
  --systemctl RUTA_ABSOLUTA --expected-systemctl-sha256 SHA256
  --systemd-run RUTA_ABSOLUTA --expected-systemd-run-sha256 SHA256
  --proc-root RUTA_ABSOLUTA
  --cgroup-mount RUTA_ABSOLUTA
  --readiness-timeout-seconds ENTERO
  --collection-timeout-seconds ENTERO

start y stop son idempotentes. status devuelve 0 si la unidad exacta está
activa y 3 si está detenida. Todos los valores son argv; no se leen variables
de entorno ni se transportan secretos.
EOF
}

require_option_value() {
  [ "$#" -ge 2 ] && [ -n "$2" ] || fail "argument_value_missing" 2
}

action="${1:-}"
case "$action" in
  start|stop|status) shift ;;
  --help|-h)
    usage
    exit 0
    ;;
  *) fail "action_invalid" 2 ;;
esac

unit=""
repository_root=""
bootstrap_home=""
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
readiness_timeout=""
collection_timeout=""

while [ "$#" -gt 0 ]; do
  option="$1"
  require_option_value "$@"
  value="$2"
  case "$option" in
    --unit)
      [ -z "$unit" ] || fail "argument_repeated" 2
      unit="$value"
      ;;
    --repository-root)
      [ -z "$repository_root" ] || fail "argument_repeated" 2
      repository_root="$value"
      ;;
    --bootstrap-home)
      [ -z "$bootstrap_home" ] || fail "argument_repeated" 2
      bootstrap_home="$value"
      ;;
    --binary)
      [ -z "$binary" ] || fail "argument_repeated" 2
      binary="$value"
      ;;
    --expected-binary-sha256)
      [ -z "$expected_binary_sha" ] || fail "argument_repeated" 2
      expected_binary_sha="$value"
      ;;
    --config)
      [ -z "$config" ] || fail "argument_repeated" 2
      config="$value"
      ;;
    --expected-config-sha256)
      [ -z "$expected_config_sha" ] || fail "argument_repeated" 2
      expected_config_sha="$value"
      ;;
    --exec-path)
      [ -z "$exec_path" ] || fail "argument_repeated" 2
      exec_path="$value"
      ;;
    --systemctl)
      [ -z "$systemctl_command" ] || fail "argument_repeated" 2
      systemctl_command="$value"
      ;;
    --expected-systemctl-sha256)
      [ -z "$expected_systemctl_sha" ] || fail "argument_repeated" 2
      expected_systemctl_sha="$value"
      ;;
    --systemd-run)
      [ -z "$systemd_run_command" ] || fail "argument_repeated" 2
      systemd_run_command="$value"
      ;;
    --expected-systemd-run-sha256)
      [ -z "$expected_systemd_run_sha" ] || fail "argument_repeated" 2
      expected_systemd_run_sha="$value"
      ;;
    --proc-root)
      [ -z "$proc_root" ] || fail "argument_repeated" 2
      proc_root="$value"
      ;;
    --cgroup-mount)
      [ -z "$cgroup_mount" ] || fail "argument_repeated" 2
      cgroup_mount="$value"
      ;;
    --readiness-timeout-seconds)
      [ -z "$readiness_timeout" ] || fail "argument_repeated" 2
      readiness_timeout="$value"
      ;;
    --collection-timeout-seconds)
      [ -z "$collection_timeout" ] || fail "argument_repeated" 2
      collection_timeout="$value"
      ;;
    *) fail "argument_unknown" 2 ;;
  esac
  shift 2
done

for required in \
  "$unit" "$repository_root" "$bootstrap_home" "$binary" \
  "$expected_binary_sha" "$config" "$expected_config_sha" "$exec_path" \
  "$systemctl_command" "$expected_systemctl_sha" "$systemd_run_command" \
  "$expected_systemd_run_sha" "$proc_root" "$cgroup_mount" \
  "$readiness_timeout" "$collection_timeout"; do
  [ -n "$required" ] || fail "argument_required" 2
done

[[ "$unit" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,126}\.service$ ]] &&
  [[ "$unit" != *@* ]] &&
  [[ "$unit" != *%* ]] ||
  fail "unit_invalid" 2
for timeout in "$readiness_timeout" "$collection_timeout"; do
  [[ "$timeout" =~ ^[1-9][0-9]*$ ]] &&
    [ "$timeout" -le 600 ] ||
    fail "timeout_invalid" 2
done
for digest in \
  "$expected_binary_sha" "$expected_config_sha" \
  "$expected_systemctl_sha" "$expected_systemd_run_sha"; do
  [[ "$digest" =~ ^[0-9a-f]{64}$ ]] || fail "sha256_invalid" 2
done

canonical_directory() {
  local supplied="$1"
  local resolved
  [ "${supplied#/}" != "$supplied" ] && [ ! -L "$supplied" ] ||
    return 1
  resolved="$(realpath -e -- "$supplied" 2>/dev/null)" || return 1
  [ "$resolved" = "$supplied" ] && [ -d "$resolved" ]
}

canonical_regular_file() {
  local supplied="$1"
  local resolved
  [ "${supplied#/}" != "$supplied" ] && [ ! -L "$supplied" ] ||
    return 1
  resolved="$(realpath -e -- "$supplied" 2>/dev/null)" || return 1
  [ "$resolved" = "$supplied" ] && [ -f "$resolved" ]
}

private_directory() {
  local directory="$1"
  canonical_directory "$directory" &&
    [ -O "$directory" ] &&
    [ -n "$(find "$directory" -maxdepth 0 -type d -perm 0700 -print -quit)" ]
}

non_writable_regular_file() {
  local file="$1"
  canonical_regular_file "$file" &&
    [ -z "$(find "$file" -maxdepth 0 -type f -perm /022 -print -quit)" ]
}

executable_regular_file() {
  local file="$1"
  non_writable_regular_file "$file" && [ -x "$file" ]
}

sha256_file() {
  sha256sum -- "$1" | awk '{print $1}'
}

canonical_directory "$repository_root" || fail "repository_root_invalid"
private_directory "$bootstrap_home" || fail "bootstrap_home_invalid"
[ ! -e "$bootstrap_home/auth.json" ] &&
  [ ! -L "$bootstrap_home/auth.json" ] ||
  fail "bootstrap_home_contains_auth"
executable_regular_file "$binary" || fail "binary_invalid"
non_writable_regular_file "$config" || fail "config_invalid"
executable_regular_file "$systemctl_command" || fail "systemctl_invalid"
executable_regular_file "$systemd_run_command" || fail "systemd_run_invalid"
canonical_directory "$proc_root" || fail "proc_root_invalid"
canonical_directory "$cgroup_mount" || fail "cgroup_mount_invalid"

[ "$(sha256_file "$binary")" = "$expected_binary_sha" ] ||
  fail "binary_drift"
[ "$(sha256_file "$config")" = "$expected_config_sha" ] ||
  fail "config_drift"
[ "$(sha256_file "$systemctl_command")" = "$expected_systemctl_sha" ] ||
  fail "systemctl_drift"
[ "$(sha256_file "$systemd_run_command")" = "$expected_systemd_run_sha" ] ||
  fail "systemd_run_drift"

IFS=':' read -r -a exec_directories <<<"$exec_path"
[[ ":$exec_path:" != *::* ]] &&
  [ "${#exec_directories[@]}" -gt 0 ] ||
  fail "exec_path_invalid"
for directory in "${exec_directories[@]}"; do
  canonical_directory "$directory" || fail "exec_path_invalid"
done
for service_argument in \
  "$repository_root" "$bootstrap_home" "$binary" "$config" "$exec_path" \
  "$proc_root" "$cgroup_mount"; do
  [[ "$service_argument" != *%* ]] || fail "systemd_specifier_unsafe"
done

declare -A UNIT_PROPERTIES=()

load_unit_properties() {
  local output line name value
  output="$(
    "$systemctl_command" --user show "$unit" --no-pager \
      "--property=$SHOW_PROPERTIES"
  )" || fail "systemctl_show_failed"
  UNIT_PROPERTIES=()
  while IFS= read -r line; do
    [ -n "$line" ] || continue
    name="${line%%=*}"
    value="${line#*=}"
    [ "$name" != "$line" ] || fail "unit_properties_invalid"
    [ -z "${UNIT_PROPERTIES[$name]+present}" ] ||
      fail "unit_properties_duplicate"
    UNIT_PROPERTIES["$name"]="$value"
  done <<<"$output"
  [ -n "${UNIT_PROPERTIES[LoadState]:-}" ] ||
    fail "unit_load_state_missing"
}

require_property() {
  local name="$1"
  local expected="$2"
  [ "${UNIT_PROPERTIES[$name]:-}" = "$expected" ] ||
    fail "unit_property_${name}_invalid"
}

require_static_unit_contract() {
  require_property "Description" "$DESCRIPTION"
  require_property "LoadState" "loaded"
  require_property "Type" "exec"
  require_property "CollectMode" "inactive-or-failed"
  require_property "Delegate" "yes"
  require_property "DelegateSubgroup" "$CONTROL_GROUP_NAME"
  require_property "KillMode" "control-group"
  require_property "SendSIGKILL" "yes"
  require_property "Restart" "no"
}

read_process_cgroup() {
  local pid="$1"
  local cgroup_file="$proc_root/$pid/cgroup"
  local content
  [ -f "$cgroup_file" ] && [ ! -L "$cgroup_file" ] || return 1
  content="$(<"$cgroup_file")"
  [[ "$content" == 0::* ]] &&
    [[ "$content" != *$'\n'* ]] ||
    return 1
  printf '%s' "${content#0::}"
}

process_command_matches() {
  local pid="$1"
  local executable
  local -a arguments=()
  executable="$(realpath -e -- "$proc_root/$pid/exe" 2>/dev/null)" ||
    return 1
  [ "$executable" = "$binary" ] || return 1
  mapfile -d '' -t arguments <"$proc_root/$pid/cmdline" || return 1
  [ "${#arguments[@]}" -eq 4 ] &&
    [ "${arguments[0]}" = "$binary" ] &&
    [ "${arguments[1]}" = "serve" ] &&
    [ "${arguments[2]}" = "--config" ] &&
    [ "${arguments[3]}" = "$config" ]
}

require_running_unit_contract() {
  local main_pid control_group expected_cgroup cgroup_parent cgroup_control
  require_static_unit_contract
  require_property "ActiveState" "active"
  require_property "SubState" "running"
  require_property "Result" "success"
  main_pid="${UNIT_PROPERTIES[MainPID]:-}"
  [[ "$main_pid" =~ ^[1-9][0-9]*$ ]] ||
    fail "unit_main_pid_invalid"
  process_command_matches "$main_pid" ||
    fail "unit_main_process_invalid"

  control_group="${UNIT_PROPERTIES[ControlGroup]:-}"
  [ "${control_group#/}" != "$control_group" ] &&
    [ "${control_group##*/}" = "$unit" ] &&
    [[ "$control_group" != *//* ]] &&
    [[ "$control_group" != */../* ]] &&
    [[ "$control_group" != */./* ]] ||
    fail "unit_control_group_invalid"
  expected_cgroup="$control_group/$CONTROL_GROUP_NAME"
  [ "$(read_process_cgroup "$main_pid" 2>/dev/null)" = "$expected_cgroup" ] ||
    fail "unit_main_cgroup_invalid"

  cgroup_parent="$cgroup_mount/${control_group#/}"
  cgroup_control="$cgroup_parent/$CONTROL_GROUP_NAME"
  [ ! -L "$cgroup_parent" ] && [ ! -L "$cgroup_control" ] &&
    [ -d "$cgroup_parent" ] && [ -d "$cgroup_control" ] &&
    [ -O "$cgroup_parent" ] && [ -O "$cgroup_control" ] ||
    fail "delegated_cgroup_invalid"
  [ "$(realpath -e -- "$cgroup_parent" 2>/dev/null)" = "$cgroup_parent" ] &&
    [ "$(realpath -e -- "$cgroup_control" 2>/dev/null)" = "$cgroup_control" ] ||
    fail "delegated_cgroup_invalid"
  [ -n "$(
    find "$cgroup_parent" -maxdepth 0 -type d -perm 0700 -print -quit
  )" ] || fail "delegated_cgroup_mode_invalid"
  [ -f "$cgroup_parent/cgroup.procs" ] &&
    [ -f "$cgroup_parent/cgroup.controllers" ] &&
    [ -f "$cgroup_parent/cgroup.subtree_control" ] &&
    [ -f "$cgroup_control/cgroup.procs" ] ||
    fail "delegated_cgroup_contract_missing"
  [ -z "$(tr -d '[:space:]' <"$cgroup_parent/cgroup.procs")" ] ||
    fail "delegated_cgroup_parent_not_empty"
  for controller in cpu memory pids; do
    grep -qw -- "$controller" "$cgroup_parent/cgroup.controllers" ||
      fail "delegated_cgroup_controller_missing"
  done
  [ "$(xargs <"$cgroup_parent/cgroup.subtree_control")" = \
    "cpu memory pids" ] ||
    fail "delegated_cgroup_controllers_invalid"
  grep -qx -- "$main_pid" "$cgroup_control/cgroup.procs" ||
    fail "unit_main_pid_not_in_control_group"
}

print_running() {
  printf '%s: status=running action=%s unit=%s main_pid=%s\n' \
    "$PROGRAM" "$action" "$unit" "${UNIT_PROPERTIES[MainPID]}"
}

unit_is_running() {
  [ "${UNIT_PROPERTIES[LoadState]:-}" = "loaded" ] &&
    [ "${UNIT_PROPERTIES[ActiveState]:-}" = "active" ] &&
    [ "${UNIT_PROPERTIES[SubState]:-}" = "running" ]
}

wait_for_running() {
  local deadline=$((SECONDS + readiness_timeout))
  while [ "$SECONDS" -lt "$deadline" ]; do
    load_unit_properties
    if unit_is_running &&
      (require_running_unit_contract) >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

wait_for_collection() {
  local deadline=$((SECONDS + collection_timeout))
  while [ "$SECONDS" -lt "$deadline" ]; do
    load_unit_properties
    [ "${UNIT_PROPERTIES[LoadState]}" = "not-found" ] && return 0
    sleep 0.1
  done
  return 1
}

case "$action" in
  status)
    load_unit_properties
    if [ "${UNIT_PROPERTIES[LoadState]}" = "not-found" ]; then
      printf '%s: status=stopped action=status unit=%s\n' "$PROGRAM" "$unit"
      exit 3
    fi
    if unit_is_running; then
      require_running_unit_contract
      print_running
      exit 0
    fi
    require_static_unit_contract
    printf '%s: status=stopped action=status unit=%s\n' "$PROGRAM" "$unit"
    exit 3
    ;;

  start)
    load_unit_properties
    if unit_is_running; then
      require_running_unit_contract
      printf '%s: status=running action=start result=already_running unit=%s main_pid=%s\n' \
        "$PROGRAM" "$unit" "${UNIT_PROPERTIES[MainPID]}"
      exit 0
    fi
    [ "${UNIT_PROPERTIES[LoadState]}" = "not-found" ] ||
      fail "unit_name_conflict"

    # Llega literal a /bin/bash; valores variables viajan solo como argv.
    # No contiene formatos con %, evitando specifiers accidentales de systemd.
    # shellcheck disable=SC2016
    service_body='
set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH
unit="$1"
cgroup_mount="$2"
proc_root="$3"
repository_root="$4"
bootstrap_home="$5"
binary="$6"
config="$7"
exec_path="$8"
self_line="$(<"$proc_root/self/cgroup")"
[[ "$self_line" == 0::/* ]] || exit 70
self_cgroup="${self_line#0::}"
case "$self_cgroup" in
  */"$unit"/orquesta-control) ;;
  *) exit 71 ;;
esac
parent="$(/usr/bin/dirname -- "$cgroup_mount/${self_cgroup#/}")"
[ ! -L "$parent" ] && [ -d "$parent" ] && [ -O "$parent" ] || exit 72
/usr/bin/chmod 0700 -- "$parent"
if read -r unexpected_pid <"$parent/cgroup.procs"; then
  exit 73
fi
for controller in cpu memory pids; do
  /usr/bin/grep -qw -- "$controller" "$parent/cgroup.controllers" || exit 74
done
for controller in $(<"$parent/cgroup.subtree_control"); do
  case "$controller" in
    cpu|memory|pids) ;;
    *) echo "-$controller" >"$parent/cgroup.subtree_control" ;;
  esac
done
echo "+cpu +memory +pids" >"$parent/cgroup.subtree_control"
[ "$(/usr/bin/xargs <"$parent/cgroup.subtree_control")" = "cpu memory pids" ] ||
  exit 75
[ ! -e "$bootstrap_home/auth.json" ] && [ ! -L "$bootstrap_home/auth.json" ] ||
  exit 76
cd -- "$repository_root"
exec /usr/bin/env -i \
  HOME="$bootstrap_home" \
  CODEX_HOME="$bootstrap_home" \
  PATH="$exec_path" \
  "$binary" serve --config "$config"
'
    systemd_run_args=(
      --user
      --expand-environment=no
      --collect
      --service-type=exec
      "--unit=$unit"
      "--description=$DESCRIPTION"
    )
    for property in "${SYSTEMD_PROPERTIES[@]}"; do
      systemd_run_args+=("--property=$property")
    done
    systemd_run_args+=(
      /bin/bash -c "$service_body" "$PROGRAM"
      "$unit" "$cgroup_mount" "$proc_root" "$repository_root"
      "$bootstrap_home" "$binary" "$config" "$exec_path"
    )
    "$systemd_run_command" "${systemd_run_args[@]}" >/dev/null ||
      fail "systemd_run_failed"
    if ! wait_for_running; then
      "$systemctl_command" --user stop "$unit" >/dev/null 2>&1 || true
      fail "unit_readiness_failed"
    fi
    require_running_unit_contract
    print_running
    ;;

  stop)
    load_unit_properties
    if [ "${UNIT_PROPERTIES[LoadState]}" = "not-found" ]; then
      printf '%s: status=stopped action=stop result=already_stopped unit=%s\n' \
        "$PROGRAM" "$unit"
      exit 0
    fi
    if unit_is_running; then
      require_running_unit_contract
    else
      require_static_unit_contract
    fi
    "$systemctl_command" --user stop "$unit" >/dev/null ||
      fail "systemctl_stop_failed"
    wait_for_collection || fail "unit_collection_timeout"
    printf '%s: status=stopped action=stop unit=%s\n' "$PROGRAM" "$unit"
    ;;
esac
