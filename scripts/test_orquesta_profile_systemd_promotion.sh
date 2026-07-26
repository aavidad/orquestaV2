#!/usr/bin/env bash

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
readonly SCRIPT_DIR
readonly SUBJECT="$SCRIPT_DIR/orquesta_profile_systemd_promotion.sh"
readonly REAL_HELPER="$SCRIPT_DIR/lib/orquesta_profile_systemd_promotion.py"
TEST_ROOT="$(mktemp -d)"
readonly TEST_ROOT
readonly PROFILE="Codex12"
readonly UNIT="orquesta-v23-Codex12.service"
readonly MAIN_PID=30001
readonly DAEMON_PID=30002
readonly CANDIDATE_REVISION="86dedc1c44e4fa161eb68a70413ae81a3d585622"
readonly REPOSITORY_HEAD="c741e5b8c4045993927ddb2f576d9d5f60fd0714"
readonly ASSET_DIGEST="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
declare -a SOCKET_PIDS=()

cleanup() {
  local pid
  for pid in "${SOCKET_PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  done
  find "$TEST_ROOT" -depth -mindepth 1 -delete 2>/dev/null || true
  rmdir "$TEST_ROOT" 2>/dev/null || true
}
trap cleanup EXIT

fail_test() {
  printf 'test_orquesta_profile_systemd_promotion: status=failed case=%s\n' \
    "$1" >&2
  exit 1
}

sha256_of() {
  sha256sum -- "$1" | awk '{print $1}'
}

write_proc_stat() {
  local target="$1" pid="$2" start_ref="$3"
  printf '%s (orquesta) S' "$pid" >"$target"
  for _ in $(seq 1 18); do
    printf ' 0' >>"$target"
  done
  printf ' %s\n' "$start_ref" >>"$target"
}

write_fake_tools() {
  local root="$1"
  mkdir -m 700 "$root/tools"
  printf '%s\n' '#!/bin/true' >"$root/tools/binary"
  printf '%s\n' '#!/bin/true' >"$root/tools/live-binary"
  printf '%s\n' '#!/bin/true' >"$root/tools/bubblewrap"
  printf '%s\n' '#!/bin/true' >"$root/tools/systemd-run"
  printf '%s\n' '#!/bin/true' >"$root/tools/firecracker-probe"

  cat >"$root/tools/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="$(dirname "$(dirname "$0")")"
case " $* " in
  *" status "*) exit 0 ;;
  *" rev-parse "*) cat "$root/state/repository-head"; exit 0 ;;
  *" merge-base --is-ancestor "*)
    [ -e "$root/state/candidate-is-ancestor" ] || exit 1
    exit 0
    ;;
  *) exit 90 ;;
esac
EOF

  cat >"$root/tools/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="$(dirname "$(dirname "$0")")"
[ "${1:-}" = version ] && [ "${2:-}" = -m ] || exit 90
head="$(<"$root/state/candidate-revision")"
printf '%s\n' \
  "$3: go1.25.11" \
  $'\tpath\torquesta/cmd/orquesta' \
  $'\tbuild\t-trimpath=true' \
  $'\tbuild\tCGO_ENABLED=0' \
  $'\tbuild\tvcs.revision='"$head" \
  $'\tbuild\tvcs.modified=false'
EOF

  cat >"$root/tools/profile" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="$(dirname "$(dirname "$0")")"
action="${1:-}"
shift || true
runtime_base=""
profile=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --runtime-base) runtime_base="$2"; shift 2 ;;
    --profile) profile="$2"; shift 2 ;;
    *) shift ;;
  esac
done
run="$runtime_base/$profile/run"
case "$action" in
  start)
    if [ -e "$root/state/live" ]; then
      printf 'orquesta_profile_server: status=error reason_code=daemon_already_running_different_contract\n' >&2
      exit 1
    fi
    exit 91
    ;;
  stop)
    if [ -e "$root/state/stop-fail" ]; then
      printf 'orquesta_profile_server: status=error reason_code=injected_stop_failure\n' >&2
      exit 1
    fi
    printf '%s\n' stop >>"$root/log/profile"
    rm -f "$run/server.pid" "$run/server.start_ref" "$run/server.binary_id" \
      "$run/server.binary_sha256" "$run/server.config_sha256"
    rm -f "$root/state/live"
    find "$root/proc/30002" -depth -mindepth 0 -delete 2>/dev/null || true
    printf '%s\n' 30001 \
      >"$root/cgroup/orquesta-v23-Codex12.service/orquesta-control/cgroup.procs"
    printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
    ;;
  status)
    if [ -f "$run/server.pid" ]; then
      printf 'orquesta_profile_server: status=running profile=%s pid=%s\n' \
        "$profile" "$(<"$run/server.pid")"
      exit 0
    fi
    printf 'orquesta_profile_server: status=stopped profile=%s\n' "$profile"
    exit 3
    ;;
  *) exit 92 ;;
esac
EOF

  cat >"$root/tools/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="$(dirname "$(dirname "$0")")"
user=0
operation=""
unit=""
for argument in "$@"; do
  case "$argument" in
    --user) user=1 ;;
    show|stop) operation="$argument" ;;
    --property=*) ;;
    *) unit="$argument" ;;
  esac
done
if [ "$user" -eq 0 ]; then
  [ "$operation" = show ] || exit 90
  if [ ! -e "$root/state/firecracker-ready" ]; then
    printf '%s\n' LoadState=not-found
    exit 0
  fi
  cat <<OUT
LoadState=loaded
ActiveState=active
SubState=running
UnitFileState=enabled
Result=success
MainPID=40001
InvocationID=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
FragmentPath=$root/firecracker.unit
NeedDaemonReload=no
OUT
  exit 0
fi
[ "$unit" = orquesta-v23-Codex12.service ] || exit 91
case "$operation" in
  show)
    if [ "$(<"$root/state/unit")" = not-found ]; then
      printf '%s\n' LoadState=not-found
      exit 0
    fi
    cat <<OUT
LoadState=loaded
ActiveState=active
SubState=running
Result=success
ExecMainCode=0
ExecMainStatus=0
MainPID=30001
ControlGroup=/orquesta-v23-Codex12.service
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
OUT
    ;;
  stop) printf '%s\n' not-found >"$root/state/unit" ;;
  *) exit 92 ;;
esac
EOF

  cat >"$root/tools/helper" <<EOF
#!/usr/bin/env bash
set -euo pipefail
root="\$(dirname "\$(dirname "\$0")")"
if [ "\${1:-}" = backup ] && [ -e "\$root/state/backup-fail" ]; then
  printf '%s\n' 'promotion_helper: status=error reason_code=injected_backup_failure' >&2
  exit 1
fi
if [ "\${1:-}" = backup ] && [ -e "\$root/state/backup-target-only-fail" ]; then
  "$REAL_HELPER" "\$@" >/dev/null
  find "\$root/runtime/$PROFILE/state/backups" -maxdepth 1 \
    -type f -name '*.receipt' -delete
  printf '%s\n' 'promotion_helper: status=error reason_code=injected_target_only_failure' >&2
  exit 1
fi
exec "$REAL_HELPER" "\$@"
EOF

  cat >"$root/tools/adapter" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root="$(dirname "$(dirname "$0")")"
operation=check
if [ "${1:-}" = --apply ]; then
  operation="$2"
  shift 2
fi
config=""
binary=""
repository=""
runtime=""
profile=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --config) config="$2"; shift 2 ;;
    --binary) binary="$2"; shift 2 ;;
    --repository-root) repository="$2"; shift 2 ;;
    --runtime-base) runtime="$2"; shift 2 ;;
    --profile) profile="$2"; shift 2 ;;
    *) shift ;;
  esac
done
kind=primary
case "$config" in */rollback.toml) kind=rollback ;; esac
case "$operation" in
  check)
    [ "$(<"$root/state/unit")" = loaded ] &&
      [ "$(<"$root/state/mode")" = "$kind" ] || {
        printf 'orquesta_profile_systemd_user: status=error reason_code=unit_not_ready\n' >&2
        exit 1
      }
    printf 'orquesta_profile_systemd_user: status=running action=check\n'
    ;;
  collect)
    printf 'collect|%s|%s\n' "$config" "$binary" >>"$root/log/adapter"
    if [ -e "$root/state/collect-fail" ]; then
      printf 'orquesta_profile_systemd_user: status=error reason_code=unit_collection_timeout\n' >&2
      exit 1
    fi
    printf '%s\n' not-found >"$root/state/unit"
    ;;
  start)
    printf 'start|%s|%s\n' "$config" "$binary" >>"$root/log/adapter"
    db="$(awk -F'"' '/^path = / {print $2; exit}' "$config")"
    /usr/bin/python3 - "$db" "$repository" <<'PY'
import hashlib
import pathlib
import sqlite3
import sys
db, repository = sys.argv[1:]
connection = sqlite3.connect(db)
for version, name in (
    (17, "017_intake.sql"),
    (18, "018_intake_dossiers.sql"),
    (19, "019_intake_dossier_confirmations.sql"),
):
    path = pathlib.Path(repository) / "internal/adapters/state/sqlite/migrations" / name
    checksum = "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()
    connection.execute(
        "INSERT OR IGNORE INTO schema_migrations(version,name,checksum) VALUES(?,?,?)",
        (version, name, checksum),
    )
connection.execute("PRAGMA user_version=19")
connection.execute("CREATE TABLE IF NOT EXISTS post_cut(value TEXT NOT NULL)")
connection.execute("INSERT INTO post_cut(value) SELECT 'preserved' WHERE NOT EXISTS(SELECT 1 FROM post_cut)")
connection.commit()
connection.close()
PY
    if [ "$kind" = primary ] && [ -e "$root/state/primary-fail" ]; then
      printf '%s\n' not-found >"$root/state/unit"
      printf 'orquesta_profile_systemd_user: status=error reason_code=systemd_run_false_positive\n' >&2
      exit 1
    fi
    printf '%s\n' loaded >"$root/state/unit"
    printf '%s\n' "$kind" >"$root/state/mode"
    mkdir -m 700 -p "$root/proc/30002" "$runtime/$profile/run"
    ln -s "$binary" "$root/proc/30002/exe"
    cp "$root/state/daemon.stat" "$root/proc/30002/stat"
    printf '%s\n' '0::/orquesta-v23-Codex12.service/orquesta-control' \
      >"$root/proc/30002/cgroup"
    printf '%s\n' 30002 >"$runtime/$profile/run/server.pid"
    printf '%s\n' 12345 >"$runtime/$profile/run/server.start_ref"
    stat -Lc '%d:%i' "$root/proc/30002/exe" >"$runtime/$profile/run/server.binary_id"
    sha256sum "$binary" | awk '{print $1}' >"$runtime/$profile/run/server.binary_sha256"
    sha256sum "$config" | awk '{print $1}' >"$runtime/$profile/run/server.config_sha256"
    chmod 600 "$runtime/$profile/run/"*
    printf '%s\n%s\n' 30001 30002 \
      >"$root/cgroup/orquesta-v23-Codex12.service/orquesta-control/cgroup.procs"
    provider=microvm
    concurrency=16
    [ "$kind" = primary ] || { provider=bubblewrap; concurrency=2; }
    effective="$(awk -F'"' '/^effective_path = / {print $2; exit}' "$config")"
    /usr/bin/python3 - "$effective" "$provider" "$concurrency" "$db" \
      "$root/cgroup/orquesta-v23-Codex12.service" \
      "$root/launcher.sock" <<'PY'
import json
import sys
path, provider, concurrency, db, cgroup, socket = sys.argv[1:]
entries = [
    {"key":"state.sqlite.path","value":db},
    {"key":"runtime.codex.cgroup_root","value":cgroup},
    {"key":"test_attestor.resources.cgroup_root","value":cgroup},
    {"key":"test_attestor.provider","value":provider},
    {"key":"test_attestor.max_concurrent_runs","value":int(concurrency)},
]
if provider == "microvm":
    entries += [
        {"key":"test_attestor.microvm.launcher_socket","value":socket},
        {"key":"test_attestor.microvm.expected_asset_digest","value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
    ]
with open(path, "w", encoding="utf-8") as handle:
    json.dump({"document_type":"orquesta.effective_config","entries":entries}, handle)
PY
    chmod 600 "$effective"
    printf 'orquesta_profile_systemd_user: status=running action=start\n'
    ;;
  *) exit 93 ;;
esac
EOF
  chmod 500 "$root/tools/"*
}

write_configs() {
  local root="$1" provider="$2" target="$3"
  local concurrency=16
  [ "$provider" = microvm ] || concurrency=2
  cat >"$target" <<EOF
[server]
listen = "127.0.0.1:38217"
[state.sqlite]
path = "$root/runtime/$PROFILE/state/orquesta.sqlite"
[runtime.codex]
cgroup_root = "$root/cgroup/orquesta-v23-Codex12.service"
account_profile = "$PROFILE"
[test_attestor]
provider = "$provider"
max_concurrent_runs = $concurrency
[test_attestor.resources]
cgroup_root = "$root/cgroup/orquesta-v23-Codex12.service"
[config]
effective_path = "$root/effective.json"
[identity]
local_token_path = "$root/token"
[repository.local]
seed_path = "$root/repository"
target_ref = "refs/heads/test"
EOF
  if [ "$provider" = microvm ]; then
    cat >>"$target" <<EOF
[test_attestor.microvm]
launcher_socket = "$root/launcher.sock"
expected_asset_digest = "$ASSET_DIGEST"
EOF
  else
    cat >>"$target" <<EOF
[test_attestor.bubblewrap]
command = "$root/tools/bubblewrap"
EOF
  fi
  chmod 600 "$target"
}

write_upgrade_receipt() {
  local root="$1" target="$2"
  /usr/bin/python3 - "$root" "$target" <<'PY'
import hashlib
import json
import pathlib
import sys
root, target = sys.argv[1:]
sha = lambda name: hashlib.sha256((pathlib.Path(root)/"tools"/name).read_bytes()).hexdigest()
document = {
 "schema_version":"orquesta_sqlite_upgrade_audit.v0","result":"pass",
 "candidate":{
  "binary_path":root+"/tools/binary","binary_sha256":sha("binary"),
  "repository_revision":"86dedc1c44e4fa161eb68a70413ae81a3d585622",
  "binary_vcs_modified":False,"profile_script_sha256":sha("profile"),
  "systemd_adapter_sha256":sha("adapter"),"systemctl_sha256":sha("systemctl"),
  "systemd_run_sha256":sha("systemd-run")},
 "source":{"user_version":16},
 "first_start":{"source_user_version":16,"result_user_version":19,
  "schema_migrations_added":[17,18,19],"system_status":"passed"},
 "restart":{"result_user_version":19,"schema_migrations_duplicated":False,
  "system_status":"passed"},
 "closure":{"candidate_processes":0,"database_open_processes":0,
  "live_profile_touched":False,"unit_load_state":"not-found"},
 "result_database":{"quick_check":"ok","foreign_key_check_rows":0,"user_version":19}}
pathlib.Path(target).write_text(json.dumps(document,sort_keys=True),encoding="utf-8")
PY
  chmod 600 "$target"
}

start_socket() {
  local root="$1"
  /usr/bin/python3 - "$root/launcher.sock" <<'PY' &
import signal
import socket
import sys
import time
server = socket.socket(socket.AF_UNIX, socket.SOCK_SEQPACKET)
server.bind(sys.argv[1])
signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
while True:
    time.sleep(1)
PY
  SOCKET_PIDS+=("$!")
  for _ in $(seq 1 100); do
    [ -S "$root/launcher.sock" ] && break
    sleep 0.01
  done
  chmod 660 "$root/launcher.sock"
}

new_fixture() {
  local name="$1" root
  root="$TEST_ROOT/$name"
  FIXTURE="$root"
  mkdir -m 700 "$root" "$root/repository" "$root/state" "$root/log" \
    "$root/runtime" "$root/source-home" "$root/backup-placeholder" \
    "$root/promotion" "$root/proc" "$root/cgroup"
  mkdir -p \
    "$root/repository/internal/adapters/state/sqlite/migrations" \
    "$root/runtime/$PROFILE/state/backups" "$root/runtime/$PROFILE/run" \
    "$root/cgroup/orquesta-v23-Codex12.service/orquesta-control" \
    "$root/proc/$MAIN_PID" \
    "$root/proc/$DAEMON_PID"
  chmod 700 \
    "$root/repository/internal/adapters/state/sqlite/migrations" \
    "$root/runtime/$PROFILE/state/backups" "$root/runtime/$PROFILE/run" \
    "$root/cgroup/orquesta-v23-Codex12.service/orquesta-control" \
    "$root/proc/$MAIN_PID" "$root/proc/$DAEMON_PID"
  cp "$SCRIPT_DIR/../internal/adapters/state/sqlite/migrations/"0{17,18,19}_*.sql \
    "$root/repository/internal/adapters/state/sqlite/migrations/"
  printf '%s\n' "$REPOSITORY_HEAD" >"$root/state/repository-head"
  printf '%s\n' "$CANDIDATE_REVISION" >"$root/state/candidate-revision"
  : >"$root/state/candidate-is-ancestor"
  printf '%s\n' loaded >"$root/state/unit"
  printf '%s\n' old >"$root/state/mode"
  : >"$root/state/live"
  : >"$root/state/firecracker-ready"
  : >"$root/log/adapter"
  : >"$root/log/profile"
  write_fake_tools "$root"
  write_proc_stat "$root/proc/$MAIN_PID/stat" "$MAIN_PID" 11111
  write_proc_stat "$root/proc/$DAEMON_PID/stat" "$DAEMON_PID" 12345
  cp "$root/proc/$DAEMON_PID/stat" "$root/state/daemon.stat"
  ln -s /usr/bin/sleep "$root/proc/$MAIN_PID/exe"
  ln -s "$root/tools/live-binary" "$root/proc/$DAEMON_PID/exe"
  printf '%s\n' '0::/orquesta-v23-Codex12.service/orquesta-control' \
    >"$root/proc/$MAIN_PID/cgroup"
  printf '%s\n' '0::/orquesta-v23-Codex12.service/orquesta-control' \
    >"$root/proc/$DAEMON_PID/cgroup"
  : >"$root/cgroup/orquesta-v23-Codex12.service/cgroup.procs"
  printf '%s\n' 'cpu memory pids' \
    >"$root/cgroup/orquesta-v23-Codex12.service/cgroup.controllers"
  printf '%s\n' 'cpu memory pids' \
    >"$root/cgroup/orquesta-v23-Codex12.service/cgroup.subtree_control"
  printf '%s\n%s\n' "$MAIN_PID" "$DAEMON_PID" \
    >"$root/cgroup/orquesta-v23-Codex12.service/orquesta-control/cgroup.procs"
  chmod 700 "$root/cgroup/orquesta-v23-Codex12.service" \
    "$root/cgroup/orquesta-v23-Codex12.service/orquesta-control"
  printf '%s\n' "$DAEMON_PID" >"$root/runtime/$PROFILE/run/server.pid"
  printf '%s\n' 12345 >"$root/runtime/$PROFILE/run/server.start_ref"
  stat -Lc '%d:%i' "$root/proc/$DAEMON_PID/exe" \
    >"$root/runtime/$PROFILE/run/server.binary_id"
  sha256_of "$root/tools/live-binary" \
    >"$root/runtime/$PROFILE/run/server.binary_sha256"
  printf '%064d\n' 0 >"$root/runtime/$PROFILE/run/server.config_sha256"
  chmod 600 "$root/runtime/$PROFILE/run/"*
  /usr/bin/python3 - "$root/runtime/$PROFILE/state/orquesta.sqlite" <<'PY'
import sqlite3
import sys
db=sqlite3.connect(sys.argv[1])
db.execute("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,name TEXT UNIQUE NOT NULL,checksum TEXT NOT NULL)")
for version in range(1,17):
    db.execute("INSERT INTO schema_migrations VALUES(?,?,?)",(version,f"{version:03d}_old.sql","sha256:"+"0"*64))
db.execute("CREATE TABLE durable(value TEXT NOT NULL)")
db.execute("INSERT INTO durable VALUES('before-cut')")
db.execute("PRAGMA user_version=16")
db.commit()
db.close()
PY
  chmod 600 "$root/runtime/$PROFILE/state/orquesta.sqlite"
  write_configs "$root" microvm "$root/primary.toml"
  write_configs "$root" bubblewrap "$root/rollback.toml"
  printf '%s\n' token >"$root/token"
  chmod 600 "$root/token"
  printf '%s\n' '{"token":"fixture"}' >"$root/source-home/auth.json"
  chmod 600 "$root/source-home/auth.json"
  printf '%s\n' '[Service]' 'Type=exec' >"$root/firecracker.unit"
  chmod 644 "$root/firecracker.unit"
  FIRECRACKER_UNIT_SHA="$(sha256_of "$root/firecracker.unit")"
  write_upgrade_receipt "$root" "$root/upgrade.json"
  start_socket "$root"
  CURRENT_UID="$(id -u)"
  CURRENT_GID="$(id -g)"
  CONTRACT=(
    --unit "$UNIT" --profile "$PROFILE"
    --repository-root "$root/repository"
    --repository-head "$REPOSITORY_HEAD"
    --candidate-revision "$CANDIDATE_REVISION"
    --expected-upstream-ref refs/remotes/origin/test
    --expected-config-target-ref refs/heads/test
    --git "$root/tools/git" --expected-git-sha256 "$(sha256_of "$root/tools/git")"
    --go "$root/tools/go" --expected-go-sha256 "$(sha256_of "$root/tools/go")"
    --profile-script "$root/tools/profile"
    --expected-profile-script-sha256 "$(sha256_of "$root/tools/profile")"
    --systemd-adapter "$root/tools/adapter"
    --expected-systemd-adapter-sha256 "$(sha256_of "$root/tools/adapter")"
    --promotion-helper "$root/tools/helper"
    --expected-promotion-helper-sha256 "$(sha256_of "$root/tools/helper")"
    --runtime-base "$root/runtime" --source-codex-home "$root/source-home"
    --binary "$root/tools/binary"
    --expected-binary-sha256 "$(sha256_of "$root/tools/binary")"
    --primary-config "$root/primary.toml"
    --expected-primary-config-sha256 "$(sha256_of "$root/primary.toml")"
    --rollback-config "$root/rollback.toml"
    --expected-rollback-config-sha256 "$(sha256_of "$root/rollback.toml")"
    --bubblewrap "$root/tools/bubblewrap"
    --expected-bubblewrap-sha256 "$(sha256_of "$root/tools/bubblewrap")"
    --sqlite-db "$root/runtime/$PROFILE/state/orquesta.sqlite"
    --backup-dir "$root/runtime/$PROFILE/state/backups"
    --promotion-state-dir "$root/promotion"
    --expected-cgroup-root "$root/cgroup/orquesta-v23-Codex12.service"
    --exec-path "$root/tools:/usr/bin"
    --systemctl "$root/tools/systemctl"
    --expected-systemctl-sha256 "$(sha256_of "$root/tools/systemctl")"
    --systemd-run "$root/tools/systemd-run"
    --expected-systemd-run-sha256 "$(sha256_of "$root/tools/systemd-run")"
    --proc-root "$root/proc" --cgroup-mount "$root/cgroup"
    --firecracker-probe "$root/tools/firecracker-probe"
    --expected-firecracker-probe-sha256 "$(sha256_of "$root/tools/firecracker-probe")"
    --firecracker-unit "orquesta-firecracker-attestor-$FIRECRACKER_UNIT_SHA.service"
    --firecracker-root-gate-ref "root-wrapper:$FIRECRACKER_UNIT_SHA"
    --expected-firecracker-unit-sha256 "$FIRECRACKER_UNIT_SHA"
    --expected-firecracker-unit-file-uid "$CURRENT_UID"
    --expected-firecracker-unit-file-gid "$CURRENT_GID"
    --launcher-socket "$root/launcher.sock"
    --expected-launcher-socket-uid "$CURRENT_UID"
    --expected-launcher-socket-gid "$CURRENT_GID"
    --expected-asset-digest "$ASSET_DIGEST"
    --sqlite-upgrade-receipt "$root/upgrade.json"
    --expected-sqlite-upgrade-receipt-sha256 "$(sha256_of "$root/upgrade.json")"
    --expected-primary-max-concurrent-runs 16
    --expected-rollback-max-concurrent-runs 2
    --readiness-timeout 1 --collection-timeout 1
  )
}

run_ok() {
  local expected="$1"
  shift
  set +e
  OUTPUT="$("$@" 2>&1)"
  STATUS="$?"
  set -e
  [ "$STATUS" -eq 0 ] && [[ "$OUTPUT" == *"$expected"* ]] || {
    printf 'status=%s output=%s\n' "$STATUS" "$OUTPUT" >&2
    fail_test "$expected"
  }
}

run_fails() {
  local expected="$1"
  shift
  set +e
  OUTPUT="$("$@" 2>&1)"
  STATUS="$?"
  set -e
  [ "$STATUS" -ne 0 ] && [[ "$OUTPUT" == *"reason_code=$expected"* ]] || {
    printf 'status=%s output=%s\n' "$STATUS" "$OUTPUT" >&2
    fail_test "$expected"
  }
}

set_contract_value() {
  local key="$1" replacement="$2" index
  for index in "${!CONTRACT[@]}"; do
    if [ "${CONTRACT[$index]}" = "$key" ]; then
      CONTRACT[index + 1]="$replacement"
      return
    fi
  done
  fail_test "contract_key_not_found"
}

sqlite_value() {
  /usr/bin/python3 - "$1" "$2" <<'PY'
import sqlite3
import sys
db=sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True)
print(db.execute(sys.argv[2]).fetchone()[0])
db.close()
PY
}

new_fixture check-read-only
db_sha_before="$(sha256_of "$FIXTURE/runtime/$PROFILE/state/orquesta.sqlite")"
run_ok "status=ready action=check" "$SUBJECT" "${CONTRACT[@]}"
[ ! -e "$FIXTURE/promotion/promotion.lock" ] &&
  [ -z "$(find "$FIXTURE/runtime/$PROFILE/state/backups" -mindepth 1 -print)" ] &&
  [ ! -s "$FIXTURE/log/adapter" ] && [ ! -s "$FIXTURE/log/profile" ] &&
  [ "$(sha256_of "$FIXTURE/runtime/$PROFILE/state/orquesta.sqlite")" = \
    "$db_sha_before" ] || fail_test "check_mutated"

new_fixture firecracker-before-stop
rm -f "$FIXTURE/state/firecracker-ready"
run_fails "firecracker_unit_not_ready" "$SUBJECT" --apply "${CONTRACT[@]}"
[ -e "$FIXTURE/runtime/$PROFILE/run/server.pid" ] &&
  [ ! -s "$FIXTURE/log/profile" ] && [ ! -s "$FIXTURE/log/adapter" ] &&
  [ "$(sqlite_value "$FIXTURE/runtime/$PROFILE/state/orquesta.sqlite" \
    'PRAGMA user_version')" = 16 ] ||
  fail_test "firecracker_gate_after_stop"

new_fixture candidate-rebinding
rm -f "$FIXTURE/state/candidate-is-ancestor"
run_fails "candidate_revision_not_ancestor" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
[ ! -s "$FIXTURE/log/profile" ] && [ ! -s "$FIXTURE/log/adapter" ] ||
  fail_test "candidate_rebinding_mutated"

new_fixture rollback-shared-drift
printf '%s\n' '[workspace.local]' \
  'root = "/tmp/desvio-no-autorizado"' >>"$FIXTURE/rollback.toml"
set_contract_value --expected-rollback-config-sha256 \
  "$(sha256_of "$FIXTURE/rollback.toml")"
run_fails "config_pair_invalid" "$SUBJECT" --apply "${CONTRACT[@]}"
[ ! -s "$FIXTURE/log/profile" ] && [ ! -s "$FIXTURE/log/adapter" ] ||
  fail_test "rollback_shared_drift_mutated"

new_fixture snapshot-replacement
: >"$FIXTURE/state/stop-fail"
run_fails "profile_stop_failed" "$SUBJECT" --apply "${CONTRACT[@]}"
[ -e "$FIXTURE/promotion/phase.10-preflight" ] &&
  [ -e "$FIXTURE/runtime/$PROFILE/run/server.pid" ] ||
  fail_test "snapshot_not_published_before_stop"
write_proc_stat "$FIXTURE/proc/$DAEMON_PID/stat" "$DAEMON_PID" 54321
printf '%s\n' 54321 >"$FIXTURE/runtime/$PROFILE/run/server.start_ref"
rm -f "$FIXTURE/state/stop-fail"
run_fails "pre_stop_snapshot_live_mismatch" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
[ -e "$FIXTURE/runtime/$PROFILE/run/server.pid" ] &&
  [ ! -s "$FIXTURE/log/profile" ] ||
  fail_test "replacement_daemon_stopped"

new_fixture primary-success
run_ok "status=primary_ready mode=primary" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
grep -Fq "start|$FIXTURE/primary.toml|$FIXTURE/tools/binary" \
  "$FIXTURE/log/adapter" || fail_test "primary_not_same_binary"
! grep -Fq "$FIXTURE/rollback.toml" "$FIXTURE/log/adapter" ||
  fail_test "unexpected_rollback"
if [ "$(sqlite_value "$FIXTURE/runtime/$PROFILE/state/orquesta.sqlite" \
  'PRAGMA user_version')" != 19 ] ||
  ! grep -Fqx 'firecracker_application_attestation=pending' \
    "$FIXTURE/promotion/promotion.receipt" ||
  ! grep -Fqx 'backup_restore_performed=false' \
    "$FIXTURE/promotion/promotion.receipt"; then
  fail_test "primary_receipt"
fi
operations_before="$(wc -l <"$FIXTURE/log/adapter")"
run_ok "status=complete mode=primary" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
[ "$(wc -l <"$FIXTURE/log/adapter")" -eq "$operations_before" ] ||
  fail_test "primary_reentry_repeated_effect"

new_fixture backup-failure-reentry
: >"$FIXTURE/state/backup-fail"
run_fails "sqlite_backup_failed" "$SUBJECT" --apply "${CONTRACT[@]}"
grep -Fqx stop "$FIXTURE/log/profile" &&
  ! grep -Fq 'collect|' "$FIXTURE/log/adapter" &&
  [ -e "$FIXTURE/promotion/phase.20-profile-stopped" ] ||
  fail_test "backup_failure_crossed_collect"
rm -f "$FIXTURE/state/backup-fail"
run_ok "status=primary_ready mode=primary" \
  "$SUBJECT" --apply "${CONTRACT[@]}"

new_fixture target-only-backup-reentry
: >"$FIXTURE/state/backup-target-only-fail"
run_fails "sqlite_backup_failed" "$SUBJECT" --apply "${CONTRACT[@]}"
[ "$(find "$FIXTURE/runtime/$PROFILE/state/backups" -maxdepth 1 \
  -type f -name '*.sqlite' | wc -l)" -eq 1 ] &&
  [ "$(find "$FIXTURE/runtime/$PROFILE/state/backups" -maxdepth 1 \
    -type f -name '*.receipt' | wc -l)" -eq 0 ] ||
  fail_test "target_only_backup_not_reproduced"
rm -f "$FIXTURE/state/backup-target-only-fail"
run_ok "status=primary_ready mode=primary" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
[ "$(find "$FIXTURE/runtime/$PROFILE/state/backups" -maxdepth 1 \
  -type f -name '*.receipt' | wc -l)" -eq 1 ] ||
  fail_test "target_only_backup_not_recovered"

new_fixture primary-false-positive-rollback
: >"$FIXTURE/state/primary-fail"
run_ok "status=available mode=rollback" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
if ! grep -Fq "start|$FIXTURE/primary.toml|$FIXTURE/tools/binary" \
  "$FIXTURE/log/adapter" ||
  ! grep -Fq "start|$FIXTURE/rollback.toml|$FIXTURE/tools/binary" \
    "$FIXTURE/log/adapter"; then
  sed -n '1,20p' "$FIXTURE/log/adapter" >&2
  fail_test "rollback_changed_binary"
fi
if [ "$(sqlite_value "$FIXTURE/runtime/$PROFILE/state/orquesta.sqlite" \
  "SELECT COUNT(*) FROM post_cut WHERE value='preserved'")" != 1 ] ||
  ! grep -Fqx 'result=rollback' "$FIXTURE/promotion/promotion.receipt" ||
  ! grep -Fqx 'backup_restore_performed=false' \
    "$FIXTURE/promotion/promotion.receipt"; then
  fail_test "rollback_restored_backup_or_changed_db"
fi
operations_before="$(wc -l <"$FIXTURE/log/adapter")"
rm -f "$FIXTURE/state/firecracker-ready"
run_ok "status=complete action=check mode=rollback" \
  "$SUBJECT" "${CONTRACT[@]}"
run_ok "status=complete mode=rollback" \
  "$SUBJECT" --apply "${CONTRACT[@]}"
[ "$(wc -l <"$FIXTURE/log/adapter")" -eq "$operations_before" ] ||
  fail_test "rollback_reentry_repeated_effect"

printf 'test_orquesta_profile_systemd_promotion: status=passed\n'
