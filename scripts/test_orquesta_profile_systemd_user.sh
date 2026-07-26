#!/usr/bin/env bash

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
readonly SCRIPT_DIR
readonly SUBJECT="$SCRIPT_DIR/orquesta_profile_systemd_user.sh"
TEST_ROOT="$(mktemp -d)"
readonly TEST_ROOT
readonly PROFILE="Codex12"
readonly UNIT="orquesta-v23-Codex12.service"
readonly MAIN_PID="30001"
readonly DAEMON_PID="30002"
readonly CONTROL_GROUP="/user.slice/user-1000.slice/user@1000.service/app.slice/$UNIT"

cleanup() {
  find "$TEST_ROOT" -depth -mindepth 1 -delete 2>/dev/null || true
  rmdir "$TEST_ROOT" 2>/dev/null || true
}
trap cleanup EXIT

fail_test() {
  printf 'test_orquesta_profile_systemd_user: status=failed case=%s\n' "$1" >&2
  exit 1
}

sha256_of() {
  sha256sum -- "$1" | awk '{print $1}'
}

write_fake_commands() {
  root="$1"
  mkdir -m 700 -- "$root/commands"
  cat >"$root/commands/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="${ORQUESTA_SYSTEMD_FAKE_ROOT:?}"
operation=""
value_only=0
unit=""
for argument in "$@"; do
  case "$argument" in
    show|stop) operation="$argument" ;;
    --value) value_only=1 ;;
    --user|--property=*) ;;
    *) unit="$argument" ;;
  esac
done
[ "$unit" = "orquesta-v23-Codex12.service" ] || exit 91
case "$operation" in
  show)
    if [ "$value_only" -eq 1 ]; then
      cat -- "$root/state/load_state"
    else
      if [ -f "$root/state/transient_main_executable" ]; then
        transient_remaining="$(<"$root/state/transient_main_executable")"
        if [ "$transient_remaining" = "1" ]; then
          printf '%s\n' "0" >"$root/state/transient_main_executable"
        else
          ln -sfn -- /usr/bin/sleep "$root/proc/30001/exe"
          rm -f -- "$root/state/transient_main_executable"
        fi
      fi
      cat -- "$root/state/properties"
    fi
    ;;
  stop)
    printf '%s\n' "$unit" >>"$root/log/systemctl-stop"
    if [ "${ORQUESTA_SYSTEMD_FAKE_STOP_MODE:-collect}" = "collect" ]; then
      printf '%s\n' "not-found" >"$root/state/load_state"
      printf '%s\n' "LoadState=not-found" >"$root/state/properties"
    fi
    ;;
  *) exit 92 ;;
esac
EOF
  cat >"$root/commands/systemd-run" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="${ORQUESTA_SYSTEMD_FAKE_ROOT:?}"
printf '%s\0' "$@" >"$root/log/systemd-run.argv"
[ "${1:-}" = "--user" ] &&
  [ "${2:-}" = "--expand-environment=no" ] ||
  exit 97
case "${ORQUESTA_SYSTEMD_FAKE_RUN_MODE:-success}" in
  success)
    cp -- "$root/state/properties.ready" "$root/state/properties"
    printf '%s\n' "loaded" >"$root/state/load_state"
    ;;
  transient-success)
    ln -sfn -- /bin/bash "$root/proc/30001/exe"
    printf '%s\n' "1" >"$root/state/transient_main_executable"
    cp -- "$root/state/properties.ready" "$root/state/properties"
    printf '%s\n' "loaded" >"$root/state/load_state"
    ;;
  timeout)
    cp -- "$root/state/properties.activating" "$root/state/properties"
    printf '%s\n' "loaded" >"$root/state/load_state"
    ;;
  false-positive) ;;
  failure) exit 93 ;;
  *) exit 94 ;;
esac
EOF
  cat >"$root/commands/profile" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="${ORQUESTA_SYSTEMD_FAKE_ROOT:?}"
action="${1:-}"
shift
profile=""
runtime_base=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --profile) profile="$2"; shift 2 ;;
    --runtime-base) runtime_base="$2"; shift 2 ;;
    *) shift ;;
  esac
done
[ "$profile" = "Codex12" ] || exit 95
case "$action" in
  stop)
    pid="$(<"$runtime_base/$profile/run/server.pid")"
    rm -f -- \
      "$runtime_base/$profile/run/server.pid" \
      "$runtime_base/$profile/run/server.start_ref" \
      "$runtime_base/$profile/run/server.binary_id" \
      "$runtime_base/$profile/run/server.binary_sha256" \
      "$runtime_base/$profile/run/server.config_sha256"
    find "$root/proc/$pid" -depth -mindepth 0 -delete 2>/dev/null || true
    printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
    ;;
  start)
    printf '%s\n' "$*" >"$root/log/profile-start"
    ;;
  *) exit 96 ;;
esac
EOF
  chmod 500 -- \
    "$root/commands/systemctl" \
    "$root/commands/systemd-run" \
    "$root/commands/profile"
}

write_ready_properties() {
  root="$1"
  target="$2"
  active_state="${3:-active}"
  sub_state="${4:-running}"
  cat >"$target" <<EOF
LoadState=loaded
ActiveState=$active_state
SubState=$sub_state
Result=success
ExecMainCode=0
ExecMainStatus=0
MainPID=$MAIN_PID
ControlGroup=$CONTROL_GROUP
InvocationID=0123456789abcdef0123456789abcdef
Type=exec
CollectMode=inactive-or-failed
Delegate=yes
DelegateSubgroup=orquesta-control
MemoryAccounting=yes
TasksAccounting=yes
KillMode=mixed
SendSIGKILL=yes
TimeoutStopUSec=30s
Restart=no
LimitNOFILE=32768
LimitNOFILESoft=32768
LimitFSIZE=536870912
LimitFSIZESoft=536870912
LimitAS=4294967296
LimitASSoft=4294967296
LimitCPU=900
LimitCPUSoft=900
TasksMax=150992
RuntimeMaxUSec=infinity
CPUQuotaPerSecUSec=infinity
MemoryMax=infinity
MemorySwapMax=infinity
EOF
}

new_fixture() {
  name="$1"
  FIXTURE="$TEST_ROOT/$name"
  mkdir -m 700 -- "$FIXTURE"
  mkdir -m 700 -- \
    "$FIXTURE/repository" \
    "$FIXTURE/runtime" \
    "$FIXTURE/source-home" \
    "$FIXTURE/tools" \
    "$FIXTURE/proc" \
    "$FIXTURE/cgroup" \
    "$FIXTURE/state" \
    "$FIXTURE/log"
  write_fake_commands "$FIXTURE"
  printf '%s\n' '#!/bin/false' >"$FIXTURE/tools/orquesta"
  chmod 500 -- "$FIXTURE/tools/orquesta"
  printf '%s\n' '[test_attestor]' 'provider = "neutral-fixture"' \
    >"$FIXTURE/config.toml"
  chmod 600 -- "$FIXTURE/config.toml"
  mkdir -m 700 -- "$FIXTURE/runtime/$PROFILE" "$FIXTURE/runtime/$PROFILE/run"
  printf '%s\n' "$DAEMON_PID" \
    >"$FIXTURE/runtime/$PROFILE/run/server.pid"
  printf '%s\n' "12345" \
    >"$FIXTURE/runtime/$PROFILE/run/server.start_ref"
  printf '%s\n' "1:2" \
    >"$FIXTURE/runtime/$PROFILE/run/server.binary_id"
  printf '%s\n' "$(sha256_of "$FIXTURE/tools/orquesta")" \
    >"$FIXTURE/runtime/$PROFILE/run/server.binary_sha256"
  printf '%s\n' "$(sha256_of "$FIXTURE/config.toml")" \
    >"$FIXTURE/runtime/$PROFILE/run/server.config_sha256"
  chmod 600 -- "$FIXTURE/runtime/$PROFILE/run/"*

  mkdir -m 700 -- "$FIXTURE/proc/$MAIN_PID" "$FIXTURE/proc/$DAEMON_PID"
  ln -s -- /usr/bin/sleep "$FIXTURE/proc/$MAIN_PID/exe"
  ln -s -- "$FIXTURE/tools/orquesta" "$FIXTURE/proc/$DAEMON_PID/exe"
  printf '0::%s/orquesta-control\n' "$CONTROL_GROUP" \
    >"$FIXTURE/proc/$MAIN_PID/cgroup"
  printf '0::%s/orquesta-control\n' "$CONTROL_GROUP" \
    >"$FIXTURE/proc/$DAEMON_PID/cgroup"

  parent="$FIXTURE/cgroup/${CONTROL_GROUP#/}"
  mkdir -p -- "$parent/orquesta-control"
  chmod 700 -- "$parent" "$parent/orquesta-control"
  : >"$parent/cgroup.procs"
  printf '%s\n' "cpu cpuset io memory pids" >"$parent/cgroup.controllers"
  printf '%s\n' "cpu memory pids" >"$parent/cgroup.subtree_control"
  printf '%s\n%s\n' "$MAIN_PID" "$DAEMON_PID" \
    >"$parent/orquesta-control/cgroup.procs"

  write_ready_properties "$FIXTURE" "$FIXTURE/state/properties.ready"
  write_ready_properties "$FIXTURE" "$FIXTURE/state/properties.activating" \
    "activating" "start"
  cp -- "$FIXTURE/state/properties.ready" "$FIXTURE/state/properties"
  printf '%s\n' "loaded" >"$FIXTURE/state/load_state"

  CONTRACT=(
    --unit "$UNIT"
    --profile "$PROFILE"
    --repository-root "$FIXTURE/repository"
    --profile-script "$FIXTURE/commands/profile"
    --expected-profile-script-sha256 "$(sha256_of "$FIXTURE/commands/profile")"
    --runtime-base "$FIXTURE/runtime"
    --source-codex-home "$FIXTURE/source-home"
    --binary "$FIXTURE/tools/orquesta"
    --expected-binary-sha256 "$(sha256_of "$FIXTURE/tools/orquesta")"
    --config "$FIXTURE/config.toml"
    --expected-config-sha256 "$(sha256_of "$FIXTURE/config.toml")"
    --exec-path "$FIXTURE/tools:/usr/bin"
    --systemctl "$FIXTURE/commands/systemctl"
    --expected-systemctl-sha256 "$(sha256_of "$FIXTURE/commands/systemctl")"
    --systemd-run "$FIXTURE/commands/systemd-run"
    --expected-systemd-run-sha256 "$(sha256_of "$FIXTURE/commands/systemd-run")"
    --proc-root "$FIXTURE/proc"
    --cgroup-mount "$FIXTURE/cgroup"
    --readiness-timeout 1
    --collection-timeout 1
  )
}

run_ok() {
  expected="$1"
  shift
  set +e
  output="$(ORQUESTA_SYSTEMD_FAKE_ROOT="$FIXTURE" "$@" 2>&1)"
  status="$?"
  set -e
  [ "$status" -eq 0 ] || {
    printf '%s\n' "$output" >&2
    fail_test "$expected"
  }
  [[ "$output" == *"$expected"* ]] || {
    printf '%s\n' "$output" >&2
    fail_test "$expected"
  }
}

run_fails() {
  expected="$1"
  shift
  set +e
  output="$(ORQUESTA_SYSTEMD_FAKE_ROOT="$FIXTURE" "$@" 2>&1)"
  status="$?"
  set -e
  reason_codes="$(
    printf '%s\n' "$output" |
      grep -o 'reason_code=[^[:space:]]*' || true
  )"
  [ "$status" -ne 0 ] &&
    [ "$reason_codes" = "reason_code=$expected" ] || {
    printf 'status=%s output=%s\n' "$status" "$output" >&2
    fail_test "$expected"
  }
}

new_fixture check-read-only
rm -f -- "$FIXTURE/log/systemd-run.argv" "$FIXTURE/log/systemctl-stop"
run_ok "status=running action=check" "$SUBJECT" "${CONTRACT[@]}"
[ ! -e "$FIXTURE/log/systemd-run.argv" ] &&
  [ ! -e "$FIXTURE/log/systemctl-stop" ] ||
  fail_test "check_mutated"
run_fails "argument_unknown" "$SUBJECT" start "${CONTRACT[@]}"
run_fails "check_must_be_read_only" "$SUBJECT" --apply check "${CONTRACT[@]}"

new_fixture start-success
printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
run_ok "status=running action=start" env \
  ORQUESTA_SYSTEMD_FAKE_RUN_MODE=success \
  "$SUBJECT" --apply start "${CONTRACT[@]}"
python3 - "$FIXTURE/log/systemd-run.argv" "$FIXTURE/config.toml" <<'PY' ||
import sys
values = open(sys.argv[1], "rb").read().split(b"\0")
if values[-1] == b"":
    values.pop()
decoded = [value.decode() for value in values]
expected_prefix = [
    "--user", "--expand-environment=no", "--collect", "--service-type=exec",
    "--unit=orquesta-v23-Codex12.service",
    "--property=Delegate=yes",
    "--property=DelegateSubgroup=orquesta-control",
]
if decoded[:len(expected_prefix)] != expected_prefix:
    raise SystemExit(
        f"unexpected argv prefix {decoded[:len(expected_prefix)]!r}"
    )
if decoded.count("--expand-environment=no") != 1:
    raise SystemExit("environment expansion guard was not passed exactly once")
if sys.argv[2] not in decoded:
    raise SystemExit("config was not passed as one positional argument")
body = decoded[decoded.index("-c") + 1]
if '"$profile_script" start' not in body or "bash -lc" in body:
    raise SystemExit("unsafe or missing profile invocation")
PY
  fail_test "start_argv_contract"

new_fixture readiness-transient-success
printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
run_ok "status=running action=start" env \
  ORQUESTA_SYSTEMD_FAKE_RUN_MODE=transient-success \
  "$SUBJECT" --apply start "${CONTRACT[@]}"
[[ "$output" != *"status=error"* ]] &&
  [[ "$output" != *"reason_code="* ]] ||
  fail_test "readiness_transient_diagnostic_leaked"

new_fixture unit-conflict
write_ready_properties "$FIXTURE" "$FIXTURE/state/properties" \
  "deactivating" "stop-sigterm"
run_fails "unit_name_conflict" "$SUBJECT" --apply start "${CONTRACT[@]}"
[ ! -e "$FIXTURE/log/systemd-run.argv" ] || fail_test "unit_conflict_launched"

new_fixture false-positive
printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
run_fails "systemd_run_false_positive" env \
  ORQUESTA_SYSTEMD_FAKE_RUN_MODE=false-positive \
  "$SUBJECT" --apply start "${CONTRACT[@]}"

new_fixture path-quoting
quoted="$FIXTURE/repository/espacio \" \$() [dato]"
mkdir -m 700 -- "$quoted"
CONTRACT[5]="$quoted"
printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
run_ok "status=running action=start" env \
  ORQUESTA_SYSTEMD_FAKE_RUN_MODE=success \
  "$SUBJECT" --apply start "${CONTRACT[@]}"
python3 - "$FIXTURE/log/systemd-run.argv" "$quoted" <<'PY' ||
import sys
values = open(sys.argv[1], "rb").read().split(b"\0")
needle = sys.argv[2].encode()
if values.count(needle) != 1:
    raise SystemExit("quoted path did not remain one exact argv")
PY
  fail_test "path_quoting"

new_fixture binary-drift
CONTRACT[17]="$(printf '0%.0s' {1..64})"
run_fails "binary_drift" "$SUBJECT" "${CONTRACT[@]}"

new_fixture pid-cgroup
printf '0::%s/otro\n' "$CONTROL_GROUP" \
  >"$FIXTURE/proc/$DAEMON_PID/cgroup"
run_fails "profile_cgroup_invalid" "$SUBJECT" "${CONTRACT[@]}"

new_fixture readiness-timeout
printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
run_fails "unit_readiness_timeout" env \
  ORQUESTA_SYSTEMD_FAKE_RUN_MODE=timeout \
  "$SUBJECT" --apply start "${CONTRACT[@]}"
grep -qx -- "$UNIT" "$FIXTURE/log/systemctl-stop" ||
  fail_test "readiness_timeout_not_collected"

new_fixture stop-profile
run_ok "status=profile_stopped action=stop-profile" \
  "$SUBJECT" --apply stop-profile "${CONTRACT[@]}"
[ ! -e "$FIXTURE/runtime/$PROFILE/run/server.pid" ] &&
  [ "$(cat "$FIXTURE/state/load_state")" = "loaded" ] ||
  fail_test "stop_profile_boundary"

new_fixture collect
rm -f -- "$FIXTURE/runtime/$PROFILE/run/server.pid"
run_ok "status=collected action=collect" \
  "$SUBJECT" --apply collect "${CONTRACT[@]}"
[ "$(cat "$FIXTURE/state/load_state")" = "not-found" ] ||
  fail_test "collect_state"

new_fixture collect-timeout
rm -f -- "$FIXTURE/runtime/$PROFILE/run/server.pid"
run_fails "unit_collection_timeout" env \
  ORQUESTA_SYSTEMD_FAKE_STOP_MODE=sticky \
  "$SUBJECT" --apply collect "${CONTRACT[@]}"

for provider in microvm bubblewrap; do
  new_fixture "provider-$provider"
  printf '%s\n' '[test_attestor]' "provider = \"$provider\"" \
    >"$FIXTURE/config.toml"
  chmod 600 -- "$FIXTURE/config.toml"
  CONTRACT[21]="$(sha256_of "$FIXTURE/config.toml")"
  printf '%s\n' "${CONTRACT[21]}" \
    >"$FIXTURE/runtime/$PROFILE/run/server.config_sha256"
  printf '%s\n' "not-found" >"$FIXTURE/state/load_state"
  printf '%s\n' "LoadState=not-found" >"$FIXTURE/state/properties"
  run_ok "status=running action=start" env \
    ORQUESTA_SYSTEMD_FAKE_RUN_MODE=success \
    "$SUBJECT" --apply start "${CONTRACT[@]}"
done

printf 'test_orquesta_profile_systemd_user: status=passed\n'
