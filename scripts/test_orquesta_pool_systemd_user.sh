#!/usr/bin/env bash

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

readonly PROGRAM="test_orquesta_pool_systemd_user"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
readonly SCRIPT_DIR
readonly SUBJECT="$SCRIPT_DIR/orquesta_pool_systemd_user.sh"
readonly UNIT="orquesta-v23-pool-test.service"
readonly PID="43210"

fail() {
  printf '%s: status=failed detail=%s\n' "$PROGRAM" "$1" >&2
  exit 1
}

assert_equal() {
  local expected="$1"
  local actual="$2"
  local detail="$3"
  [ "$actual" = "$expected" ] ||
    fail "$detail expected=[$expected] actual=[$actual]"
}

assert_contains() {
  local content="$1"
  local expected="$2"
  local detail="$3"
  [[ "$content" == *"$expected"* ]] ||
    fail "$detail missing=[$expected]"
}

test_root="$(mktemp -d)"
readonly test_root
trap 'rm -rf -- "$test_root"' EXIT

readonly repository_root="$test_root/repository root"
readonly bootstrap_home="$test_root/bootstrap-home"
readonly commands_root="$test_root/commands"
readonly tools_root="$test_root/tools"
readonly state_root="$test_root/state"
readonly proc_root="$test_root/proc"
readonly cgroup_mount="$test_root/cgroup"
readonly config_root="$test_root/config"
readonly binary="$tools_root/orquesta pool"
readonly config="$config_root/pool config.json"
readonly systemctl_command="$commands_root/systemctl"
readonly systemd_run_command="$commands_root/systemd-run"
readonly exec_path="$tools_root:/usr/bin"
readonly control_group="/user.slice/orquesta-tests/$UNIT"
readonly cgroup_parent="$cgroup_mount/${control_group#/}"

mkdir -p -- \
  "$repository_root" "$bootstrap_home" "$commands_root" "$tools_root" \
  "$state_root" "$proc_root" "$cgroup_mount" "$config_root"
chmod 0700 -- "$bootstrap_home" "$commands_root" "$tools_root" "$state_root"

cat >"$binary" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod 0500 -- "$binary"
printf '{}\n' >"$config"
chmod 0600 -- "$config"

cat >"$systemctl_command" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

root="${ORQUESTA_POOL_SYSTEMD_FAKE_ROOT:?}"
expected_unit="${ORQUESTA_POOL_SYSTEMD_FAKE_UNIT:?}"
state="$(<"$root/state/unit-state")"

[ "${1:-}" = "--user" ] || exit 80
shift
operation="${1:-}"
unit="${2:-}"
[ "$unit" = "$expected_unit" ] || exit 81

case "$operation" in
  show)
    if [ "$state" = "not-found" ]; then
      printf 'LoadState=not-found\n'
      exit 0
    fi
    case "$state" in
      running)
        active_state="active"
        sub_state="running"
        main_pid="$(<"$root/state/pid")"
        ;;
      inactive)
        active_state="inactive"
        sub_state="dead"
        main_pid="0"
        ;;
      *) exit 82 ;;
    esac
    cat <<PROPERTIES
Description=Orquesta pool server
LoadState=loaded
ActiveState=$active_state
SubState=$sub_state
Result=success
MainPID=$main_pid
ControlGroup=/user.slice/orquesta-tests/$unit
Type=exec
CollectMode=inactive-or-failed
Delegate=yes
DelegateSubgroup=orquesta-control
KillMode=control-group
SendSIGKILL=yes
Restart=no
PROPERTIES
    ;;
  stop)
    printf 'not-found\n' >"$root/state/unit-state"
    count="$(<"$root/state/stop-count")"
    printf '%s\n' "$((count + 1))" >"$root/state/stop-count"
    ;;
  *) exit 83 ;;
esac
EOF
chmod 0500 -- "$systemctl_command"

cat >"$systemd_run_command" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

root="${ORQUESTA_POOL_SYSTEMD_FAKE_ROOT:?}"
expected_unit="${ORQUESTA_POOL_SYSTEMD_FAKE_UNIT:?}"
mode="${ORQUESTA_POOL_SYSTEMD_FAKE_RUN_MODE:-success}"

printf '%s\0' "$@" >>"$root/state/systemd-run.argv"
count="$(<"$root/state/run-count")"
printf '%s\n' "$((count + 1))" >"$root/state/run-count"

[ "$mode" != "fail" ] || exit 90
[ "$mode" != "no-ready" ] || exit 0

unit=""
for argument in "$@"; do
  case "$argument" in
    --unit=*) unit="${argument#--unit=}" ;;
  esac
done
[ "$unit" = "$expected_unit" ] || exit 91

pid="$(<"$root/state/pid")"
binary="$(<"$root/state/binary")"
config="$(<"$root/state/config")"
proc_root="$(<"$root/state/proc-root")"
cgroup_mount="$(<"$root/state/cgroup-mount")"
control_group="/user.slice/orquesta-tests/$unit"
cgroup_parent="$cgroup_mount/${control_group#/}"
cgroup_control="$cgroup_parent/orquesta-control"

mkdir -p -- "$proc_root/$pid" "$cgroup_control"
ln -s -- "$binary" "$proc_root/$pid/exe"
printf '%s\0' "$binary" serve --config "$config" >"$proc_root/$pid/cmdline"
printf '0::%s/orquesta-control\n' "$control_group" >"$proc_root/$pid/cgroup"
chmod 0700 -- "$cgroup_parent" "$cgroup_control"
printf '\n' >"$cgroup_parent/cgroup.procs"
printf 'cpu memory pids\n' >"$cgroup_parent/cgroup.controllers"
printf 'cpu memory pids\n' >"$cgroup_parent/cgroup.subtree_control"
printf '%s\n' "$pid" >"$cgroup_control/cgroup.procs"
printf 'running\n' >"$root/state/unit-state"
EOF
chmod 0500 -- "$systemd_run_command"

printf '%s\n' "$PID" >"$state_root/pid"
printf '%s\n' "$binary" >"$state_root/binary"
printf '%s\n' "$config" >"$state_root/config"
printf '%s\n' "$proc_root" >"$state_root/proc-root"
printf '%s\n' "$cgroup_mount" >"$state_root/cgroup-mount"

expected_binary_sha="$(sha256sum -- "$binary" | awk '{print $1}')"
readonly expected_binary_sha
expected_config_sha="$(sha256sum -- "$config" | awk '{print $1}')"
readonly expected_config_sha
expected_systemctl_sha="$(
  sha256sum -- "$systemctl_command" | awk '{print $1}'
)"
readonly expected_systemctl_sha
expected_systemd_run_sha="$(
  sha256sum -- "$systemd_run_command" | awk '{print $1}'
)"
readonly expected_systemd_run_sha

CONTRACT=(
  --unit "$UNIT"
  --repository-root "$repository_root"
  --bootstrap-home "$bootstrap_home"
  --binary "$binary"
  --expected-binary-sha256 "$expected_binary_sha"
  --config "$config"
  --expected-config-sha256 "$expected_config_sha"
  --exec-path "$exec_path"
  --systemctl "$systemctl_command"
  --expected-systemctl-sha256 "$expected_systemctl_sha"
  --systemd-run "$systemd_run_command"
  --expected-systemd-run-sha256 "$expected_systemd_run_sha"
  --proc-root "$proc_root"
  --cgroup-mount "$cgroup_mount"
  --readiness-timeout-seconds 1
  --collection-timeout-seconds 1
)

LAST_OUTPUT=""
LAST_STATUS=0
RUN_MODE="success"

invoke() {
  local action="$1"
  shift
  set +e
  LAST_OUTPUT="$(
    ORQUESTA_POOL_SYSTEMD_FAKE_ROOT="$test_root" \
      ORQUESTA_POOL_SYSTEMD_FAKE_UNIT="$UNIT" \
      ORQUESTA_POOL_SYSTEMD_FAKE_RUN_MODE="$RUN_MODE" \
      "$SUBJECT" "$action" "$@" 2>&1
  )"
  LAST_STATUS=$?
  set -e
}

invoke_contract() {
  local action="$1"
  invoke "$action" "${CONTRACT[@]}"
}

expect_ok() {
  local action="$1"
  invoke_contract "$action"
  assert_equal "0" "$LAST_STATUS" "$action-status"
}

expect_reason() {
  local action="$1"
  local reason="$2"
  invoke_contract "$action"
  [ "$LAST_STATUS" -ne 0 ] || fail "$action-accepted-$reason"
  assert_contains "$LAST_OUTPUT" "reason_code=$reason" "$action-reason"
}

reset_fake() {
  rm -rf -- "${proc_root:?}/$PID" "${cgroup_mount:?}/user.slice"
  printf 'not-found\n' >"$state_root/unit-state"
  printf '0\n' >"$state_root/run-count"
  printf '0\n' >"$state_root/stop-count"
  : >"$state_root/systemd-run.argv"
  RUN_MODE="success"
}

reset_fake

invoke --help
assert_equal "0" "$LAST_STATUS" "help-status"
assert_contains "$LAST_OUTPUT" "start CONTRATO" "help-contract"

expect_ok start
assert_contains "$LAST_OUTPUT" "status=running action=start" "start-output"
assert_equal "1" "$(<"$state_root/run-count")" "start-run-count"

mapfile -d '' -t launched <"$state_root/systemd-run.argv"
assert_equal "23" "${#launched[@]}" "systemd-run-argc"
assert_equal "--user" "${launched[0]}" "systemd-run-user"
assert_equal "--expand-environment=no" "${launched[1]}" "systemd-run-expand"
assert_equal "--collect" "${launched[2]}" "systemd-run-collect"
assert_equal "--service-type=exec" "${launched[3]}" "systemd-run-type"
assert_equal "--unit=$UNIT" "${launched[4]}" "systemd-run-unit"
assert_equal "--property=Delegate=yes" "${launched[6]}" "systemd-run-delegate"
assert_equal \
  "--property=DelegateSubgroup=orquesta-control" \
  "${launched[7]}" \
  "systemd-run-subgroup"
assert_equal "/bin/bash" "${launched[11]}" "systemd-run-shell"
assert_equal "-c" "${launched[12]}" "systemd-run-shell-option"
[[ "${launched[13]}" != *%* ]] || fail "service-body-systemd-specifier"
assert_contains "${launched[13]}" "exec /usr/bin/env -i" "service-body-clean-env"
[[ "${launched[13]}" != *LoadCredential* ]] || fail "service-body-credential"
assert_equal "$repository_root" "${launched[18]}" "service-repository-argv"
assert_equal "$bootstrap_home" "${launched[19]}" "service-home-argv"
assert_equal "$binary" "${launched[20]}" "service-binary-argv"
assert_equal "$config" "${launched[21]}" "service-config-argv"
assert_equal "$exec_path" "${launched[22]}" "service-path-argv"

expect_ok start
assert_contains "$LAST_OUTPUT" "result=already_running" "start-idempotent"
assert_equal "1" "$(<"$state_root/run-count")" "start-idempotent-run-count"

expect_ok status
assert_contains "$LAST_OUTPUT" "main_pid=$PID" "status-main-pid"

expect_ok stop
assert_equal "1" "$(<"$state_root/stop-count")" "stop-count"
expect_ok stop
assert_contains "$LAST_OUTPUT" "result=already_stopped" "stop-idempotent"
assert_equal "1" "$(<"$state_root/stop-count")" "stop-idempotent-count"

invoke_contract status
assert_equal "3" "$LAST_STATUS" "stopped-status-code"
assert_contains "$LAST_OUTPUT" "status=stopped" "stopped-status-output"

reset_fake
expect_ok start
printf '%s\0' "$binary" wrong --config "$config" >"$proc_root/$PID/cmdline"
expect_reason status "unit_main_process_invalid"
assert_equal "1" "$(<"$state_root/run-count")" "foreign-process-no-relaunch"

reset_fake
expect_ok start
printf '999\n' >"$cgroup_parent/cgroup.procs"
expect_reason status "delegated_cgroup_parent_not_empty"

reset_fake
printf '{"changed":true}\n' >"$config"
expect_reason start "config_drift"
assert_equal "0" "$(<"$state_root/run-count")" "config-drift-no-launch"
printf '{}\n' >"$config"

reset_fake
printf 'never-read\n' >"$bootstrap_home/auth.json"
expect_reason start "bootstrap_home_contains_auth"
assert_equal "0" "$(<"$state_root/run-count")" "auth-no-launch"
rm -f -- "$bootstrap_home/auth.json"

reset_fake
original_unit="${CONTRACT[1]}"
CONTRACT[1]='pool%specifier.service'
expect_reason start "unit_invalid"
CONTRACT[1]="$original_unit"
assert_equal "0" "$(<"$state_root/run-count")" "invalid-unit-no-launch"

reset_fake
original_exec_path="${CONTRACT[15]}"
CONTRACT[15]="$exec_path:"
expect_reason start "exec_path_invalid"
CONTRACT[15]="$original_exec_path"
assert_equal "0" "$(<"$state_root/run-count")" "invalid-path-no-launch"

reset_fake
specifier_repository="$test_root/repository%unsafe"
mkdir -- "$specifier_repository"
original_repository="${CONTRACT[3]}"
CONTRACT[3]="$specifier_repository"
expect_reason start "systemd_specifier_unsafe"
CONTRACT[3]="$original_repository"
assert_equal "0" "$(<"$state_root/run-count")" "specifier-no-launch"

reset_fake
RUN_MODE="fail"
expect_reason start "systemd_run_failed"
assert_equal "1" "$(<"$state_root/run-count")" "failed-launch-count"

reset_fake
RUN_MODE="no-ready"
expect_reason start "unit_readiness_failed"
assert_equal "1" "$(<"$state_root/stop-count")" "readiness-cleanup-stop"

reset_fake
printf 'inactive\n' >"$state_root/unit-state"
expect_ok stop
assert_equal "1" "$(<"$state_root/stop-count")" "inactive-stop-count"

printf '%s: status=passed\n' "$PROGRAM"
