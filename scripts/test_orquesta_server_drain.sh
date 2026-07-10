#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cache_root="${ORQUESTA_DRAIN_TEST_CACHE_ROOT:-/tmp/orquesta-f3-r2-drain-tests}"
mkdir -p "$cache_root"
chmod 700 "$cache_root"
test_root="$(mktemp -d "$cache_root/test.XXXXXX")"
trap '[ "${ORQUESTA_KEEP_DRAIN_TEST_DIR:-0}" = 1 ] || rm -rf "$test_root"' EXIT

make_socket() {
  : >"$1"
  chmod 600 "$1"
}

make_proc() {
  local proc_root="$1" pid="$2" ppid="$3" pgid="$4" sid="$5" start="$6" exe="$7" cwd="$8"
  shift 8
  mkdir -p "$proc_root/$pid/fd"
  printf '%s (fixture) S %s %s %s 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 %s 0 0 0\n' "$pid" "$ppid" "$pgid" "$sid" "$start" >"$proc_root/$pid/stat"
  python3 - "$proc_root/$pid/cmdline" "$@" <<'PY'
import sys
with open(sys.argv[1], "wb") as handle:
    handle.write(b"\0".join(value.encode() for value in sys.argv[2:]) + b"\0")
PY
  ln -s "$exe" "$proc_root/$pid/exe"
  ln -s "$cwd" "$proc_root/$pid/cwd"
}

write_owner_marker() {
  local runtime="$1"
  python3 - "$runtime" <<'PY'
import json, os, sys
root = sys.argv[1]
payload = {
  "schema_version": "orquesta_codex_app_server_tmux_owner_generation.v0",
  "owner_ref": "orquesta-codex-goal-app-server-tmux-v0",
  "session_name": "orquesta-goal-fixture", "socket_ref": "socket-ref-codex-goal-app-server-tmux",
  "socket_path": os.path.join(root, "goal.sock"), "generation_ref": "generation-ref-fixture",
  "lease_owner_pid": 100, "lease_owner_start_ref": "1000",
  "app_server_pid": 112, "app_server_start_ref": "1120", "app_server_process_group_id": 110,
  "socket_owner_pid": 112, "socket_owner_start_ref": "1120",
  "tmux_session_id": "$1", "tmux_session_created": "777",
  "tmux_pane_pid": 110, "tmux_pane_start_ref": "1100",
}
with open(os.path.join(root, "goal.sock.owner.json"), "w", encoding="utf-8") as handle:
    json.dump(payload, handle, sort_keys=True)
PY
}

make_fixture() {
  local case_root="$1"
  local proc_root="$case_root/proc" state="$case_root/state" runtime="$case_root/runtime" uso="$case_root/uso-app"
  mkdir -p "$proc_root/net" "$state/runtime" "$state/state" "$state/logs" "$runtime" "$uso"
  chmod 700 "$case_root" "$proc_root" "$state" "$state/runtime" "$state/state" "$state/logs" "$runtime" "$uso"
  : >"$runtime/orquesta-server-claude"
  : >"$uso/uso-app"
  printf 'http://127.0.0.1:19071\n' >"$state/runtime/base_url.txt"
  printf '100\n' >"$state/server.pid"
  printf '{"state":"running"}\n' >"$state/state/plan_state.json"
  printf 'checkpoint\n' >"$state/checkpoint_started_goal.txt"
  printf '{"status":"complete"}\n' >"$state/orquesta_goal_result_goal.json"
  printf '{"status":"accepted"}\n' >"$state/agent_ack_goal.json"
  printf 'goal log\n' >"$state/logs/goal.log"
  printf '  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n   0: 0100007F:4A7F 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1001        0 555\n' >"$proc_root/net/tcp"
  printf '  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n' >"$proc_root/net/tcp6"

  make_proc "$proc_root" 100 1 100 100 1000 "$runtime/orquesta-server-claude" "$ROOT" \
    "$runtime/orquesta-server-claude" run
  ln -s 'socket:[555]' "$proc_root/100/fd/3"
  make_proc "$proc_root" 110 100 110 110 1100 /usr/bin/bash "$runtime" bash -lc 'codex app-server'
  make_proc "$proc_root" 112 110 110 110 1120 /usr/bin/codex "$runtime" codex app-server
  make_proc "$proc_root" 111 112 110 110 1110 /usr/local/go/bin/go "$runtime" go test ./modulos/example
  make_proc "$proc_root" 200 1 200 200 2000 "$uso/uso-app" "$uso" "$uso/uso-app" serve
  make_proc "$proc_root" 201 200 200 200 2010 /usr/bin/node "$runtime" node child.js
  make_socket "$runtime/goal.sock"
  make_socket "$case_root/tmux-control.sock"
  write_owner_marker "$runtime"
  printf 'orquesta-goal-fixture\t$1\t777\t%%1\t110\tbash -lc codex app-server\t%s/tmux-control.sock\n' "$case_root" >"$case_root/tmux.tsv"
  chmod -R go-w "$state" "$runtime" "$uso"
}

make_commands() {
  local case_root="$1" mode="$2"
  sed "s#__CASE_ROOT__#$case_root#g; s#__MODE__#$mode#g" >"$case_root/pidfd-fake" <<'SH'
#!/usr/bin/env bash
set -eu
pid="" signal=""
while [ "$#" -gt 0 ]; do
  case "$1" in --pid) pid="$2"; shift 2;; --start-ref) start="$2"; shift 2;; --signal) signal="$2"; shift 2;; *) exit 2;; esac
done
printf '%s %s\n' "$signal" "$pid" >>'__CASE_ROOT__/signal.log'
if [ '__MODE__' = fail ]; then
  printf '{"outcome":"error","error":"synthetic_failure"}\n'
  exit 6
fi
if [ '__MODE__' = tmux_replace ] && [ "$signal" = SIGTERM ] && [ "$pid" = 100 ]; then
  sed -i 's/\t777\t/\t778\t/' '__CASE_ROOT__/tmux.tsv'
fi
if [ '__MODE__' = mutate ]; then
  if [ "$signal" = SIGTERM ] && [ "$pid" = 100 ]; then
    rm -rf '__CASE_ROOT__/proc/100'
    rm -f '__CASE_ROOT__/state/server.pid' '__CASE_ROOT__/state/runtime/base_url.txt'
  fi
  if [ "$signal" = SIGKILL ]; then rm -rf '__CASE_ROOT__/proc/'"$pid"; fi
fi
printf '{"outcome":"sent"}\n'
SH
  sed "s#__CASE_ROOT__#$case_root#g; s#__MODE__#$mode#g" >"$case_root/tmux-fake" <<'SH'
#!/usr/bin/env bash
set -eu
if printf '%s\n' "$*" | grep -q list-panes; then cat '__CASE_ROOT__/tmux.tsv'; exit 0; fi
if printf '%s\n' "$*" | grep -q kill-session; then
  printf '%s\n' "$*" >>'__CASE_ROOT__/tmux.log'
  if [ '__MODE__' = mutate ]; then
    : >'__CASE_ROOT__/tmux.tsv'
    rm -rf '__CASE_ROOT__/proc/110'
    rm -f '__CASE_ROOT__/runtime/goal.sock.owner.json' '__CASE_ROOT__/runtime/goal.sock'
  fi
  exit 0
fi
exit 2
SH
  sed "s#__CASE_ROOT__#$case_root#g" >"$case_root/curl-fake" <<'SH'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >>'__CASE_ROOT__/curl.log'
for value in "$@"; do case "$value" in @*) cp "${value#@}" '__CASE_ROOT__/shutdown-payload.json';; esac; done
SH
  chmod +x "$case_root/pidfd-fake" "$case_root/tmux-fake" "$case_root/curl-fake"
}

run_drain() {
  local case_root="$1" mode="$2" output="$3"
  local args=(--dry-run)
  if [ "$mode" = real ]; then args=(--drain --confirm-drain orquesta-server-drain); fi
  set +e
  DRAIN_TEST_MODE=1 \
  ORQUESTA_DRAIN_TEST_SYNTHETIC_SOCKET_FILES=1 \
  ORQUESTA_DRAIN_PROC_ROOT="$case_root/proc" \
  ORQUESTA_DRAIN_RUNTIME_ROOT="$case_root/runtime" \
  ORQUESTA_DRAIN_STATE_ROOT="$case_root/state" \
  ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$case_root/uso-app" \
  ORQUESTA_DRAIN_BACKUP_ROOT="$case_root/backups" \
  ORQUESTA_DRAIN_RECEIPT="$case_root/receipt.json" \
  ORQUESTA_DRAIN_PIDFD_HELPER="$case_root/pidfd-fake" \
  ORQUESTA_DRAIN_CURL_COMMAND="$case_root/curl-fake" \
  ORQUESTA_DRAIN_TMUX_COMMAND="$case_root/tmux-fake" \
  ORQUESTA_DRAIN_WAIT_SECONDS=0 \
    "$ROOT/scripts/orquesta_server_drain.sh" "${args[@]}" >"$output" 2>&1
  local code=$?
  set -e
  return "$code"
}

assert_status() {
  python3 - "$1" "$2" "$3" <<'PY'
import json, sys
r = json.load(open(sys.argv[1], encoding="utf-8"))
assert r["drain_status"] == sys.argv[2], r
assert r["exit_code"] == int(sys.argv[3]), r
PY
}

chmod +x "$ROOT/scripts/lib/pidfd_signal.py" "$ROOT/scripts/lib/orquesta_drain_runtime.py"
bash -n "$ROOT/scripts/orquesta_server_drain.sh" "$ROOT/scripts/test_orquesta_server_drain.sh"
python3 -m py_compile "$ROOT/scripts/lib/pidfd_signal.py" "$ROOT/scripts/lib/orquesta_drain_runtime.py"

# pidfd real: un PID reciclado/mismatch no recibe señal; la identidad exacta sí.
sleep 30 & ephemeral_pid=$!
ephemeral_start="$(python3 - "$ephemeral_pid" <<'PY'
import sys
t=open('/proc/'+sys.argv[1]+'/stat').read(); print(t[t.rfind(')')+2:].split()[19])
PY
)"
set +e
"$ROOT/scripts/lib/pidfd_signal.py" --pid "$ephemeral_pid" --start-ref "${ephemeral_start}9" --signal SIGTERM >"$test_root/pidfd-mismatch.json"
pidfd_mismatch_rc=$?
set -e
[ "$pidfd_mismatch_rc" -ne 0 ] && kill -0 "$ephemeral_pid"
"$ROOT/scripts/lib/pidfd_signal.py" --pid "$ephemeral_pid" --start-ref "$ephemeral_start" --signal SIGTERM >"$test_root/pidfd-sent.json"
wait "$ephemeral_pid" 2>/dev/null || true
! kill -0 "$ephemeral_pid" 2>/dev/null
python3 - "$test_root/pidfd-mismatch.json" "$test_root/pidfd-sent.json" <<'PY'
import json, sys
assert json.load(open(sys.argv[1]))["outcome"] == "identity_mismatch"
assert json.load(open(sys.argv[2]))["outcome"] == "sent"
PY
echo "f3_r2_case=pidfd_real ok"

dry="$test_root/dry"
mkdir -m 700 "$dry"; make_fixture "$dry"; make_commands "$dry" mutate
run_drain "$dry" dry "$dry/output.log"
assert_status "$dry/receipt.json" dry_run 0
python3 - "$dry/receipt.json" <<'PY'
import json, os, sys
r=json.load(open(sys.argv[1])); before=r["before"]
assert r["schema_version"] == "orquesta_server_drain_receipt.v2"
assert not before["inventory_errors"] and not before["ambiguous"]
marker=before["owner_markers"][0]
for key in ("owner_ref","app_server_pid","app_server_start_ref","socket_owner_pid","socket_owner_start_ref","tmux_session_id","tmux_pane_pid","tmux_pane_start_ref"):
    assert marker.get(key) not in (None,"",0), key
by_pid={p["pid"]:p for p in before["processes"]}
assert by_pid["200"]["classification_reason"] == "protected_uso_app"
assert by_pid["201"]["classification_reason"] == "protected_uso_app_descendant"
assert r["backup"]["status"] == "complete" and r["backup"]["source_preserved"] is True
assert r["backup"]["copied"] and all(x["sha256"] and x["source_preserved"] for x in r["backup"]["copied"])
assert all(not os.path.islink(os.path.join(r["backup"]["backup_dir"], x["backup_relative"])) for x in r["backup"]["copied"])
assert not os.path.exists(os.path.join(os.path.dirname(sys.argv[1]), "signal.log"))
PY
echo "f3_r2_case=dry_run_marker_backup ok"

clean="$test_root/clean"
mkdir -m 700 "$clean"; make_fixture "$clean"; make_commands "$clean" mutate
run_drain "$clean" real "$clean/output.log"
assert_status "$clean/receipt.json" clean 0
python3 - "$clean/receipt.json" "$clean/shutdown-payload.json" "$clean/tmux.log" "$clean/signal.log" <<'PY'
import json, sys
r=json.load(open(sys.argv[1])); payload=json.load(open(sys.argv[2])); tmux=open(sys.argv[3]).read().strip(); signals=open(sys.argv[4]).read().splitlines()
assert payload["requested_by"] == "orquesta-director" and payload["cleanup_goal_backends"] is True
assert payload["request_id"] == payload["idempotency_key"]
assert "-S " in tmux and " kill-session -t $1" in tmux and "orquesta-goal-fixture" not in tmux.split("-t",1)[1]
assert any(line.startswith("SIGKILL ") for line in signals)
phases=[a["phase"] for a in r["actions"]]
assert phases.index("backup") < phases.index("http_shutdown") < phases.index("sigterm") < phases.index("tmux_exact") < phases.index("wait_after_tmux") < phases.index("sigkill_final")
assert r["uso_app_protected"]["intact"] and not r["uso_app_protected"]["signals_sent"]
PY
echo "f3_r2_case=clean_tmux_pidfd ok"

residual="$test_root/residual"
mkdir -m 700 "$residual"; make_fixture "$residual"; make_commands "$residual" fail
if run_drain "$residual" real "$residual/output.log"; then echo "residual devolvió rc0" >&2; exit 1; else residual_rc=$?; fi
[ "$residual_rc" -eq 4 ]; assert_status "$residual/receipt.json" residual 4
grep -q 'orquesta_server_drain=not_ok' "$residual/output.log"
python3 - "$residual/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert any(a["action"].startswith("signal_") and a["outcome"] == "failed" for a in r["actions"])
assert not any(a["action"].startswith("signal_") and a["outcome"] == "sent" for a in r["actions"])
PY

backup_fail="$test_root/backup-fail"
mkdir -m 700 "$backup_fail"; make_fixture "$backup_fail"; make_commands "$backup_fail" mutate
ln -s /etc/passwd "$backup_fail/state/agent_ack_external"
if run_drain "$backup_fail" real "$backup_fail/output.log"; then echo "backup symlink devolvió rc0" >&2; exit 1; else backup_rc=$?; fi
[ "$backup_rc" -eq 3 ]; assert_status "$backup_fail/receipt.json" refused 3
[ ! -e "$backup_fail/signal.log" ] && [ ! -e "$backup_fail/curl.log" ]
python3 - "$backup_fail/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert not r["backup"]["source_preserved"] and r["backup"]["errors"]
PY

inventory_fail="$test_root/inventory-fail"
mkdir -m 700 "$inventory_fail"; make_fixture "$inventory_fail"; make_commands "$inventory_fail" mutate
printf 'malformed\n' >"$inventory_fail/proc/112/stat"
if run_drain "$inventory_fail" real "$inventory_fail/output.log"; then echo "inventario roto devolvió rc0" >&2; exit 1; else inventory_rc=$?; fi
[ "$inventory_rc" -eq 3 ]; assert_status "$inventory_fail/receipt.json" refused 3
[ ! -e "$inventory_fail/signal.log" ] && [ ! -e "$inventory_fail/curl.log" ]

tmux_fail="$test_root/tmux-fail"
mkdir -m 700 "$tmux_fail"; make_fixture "$tmux_fail"; make_commands "$tmux_fail" mutate
printf '#!/usr/bin/env bash\nexit 8\n' >"$tmux_fail/tmux-fake"; chmod +x "$tmux_fail/tmux-fake"
if run_drain "$tmux_fail" dry "$tmux_fail/output.log"; then echo "error tmux devolvió rc0" >&2; exit 1; else tmux_fail_rc=$?; fi
[ "$tmux_fail_rc" -eq 3 ]; assert_status "$tmux_fail/receipt.json" refused 3
python3 - "$tmux_fail/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert any(x.startswith('tmux_list_failed:') for x in r['before']['inventory_errors'])
PY

tmux_replace="$test_root/tmux-replace"
mkdir -m 700 "$tmux_replace"; make_fixture "$tmux_replace"; make_commands "$tmux_replace" tmux_replace
if run_drain "$tmux_replace" real "$tmux_replace/output.log"; then echo "tmux reemplazado devolvió rc0" >&2; exit 1; else tmux_replace_rc=$?; fi
[ "$tmux_replace_rc" -eq 4 ]; assert_status "$tmux_replace/receipt.json" residual 4
[ ! -e "$tmux_replace/tmux.log" ]
python3 - "$tmux_replace/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); assert any(a['phase']=='tmux_exact' and a['outcome']=='identity_changed' for a in r['actions'])
PY
echo "f3_r2_case=fail_closed_matrix ok"

url_fail="$test_root/url-fail"
mkdir -m 700 "$url_fail"; make_fixture "$url_fail"; make_commands "$url_fail" mutate
printf 'http://127.0.0.1:19071@evil.example\n' >"$url_fail/state/runtime/base_url.txt"
if run_drain "$url_fail" real "$url_fail/output.log"; then echo "URL injection devolvió rc0" >&2; exit 1; else url_rc=$?; fi
[ "$url_rc" -eq 3 ]; assert_status "$url_fail/receipt.json" refused 3
[ ! -e "$url_fail/curl.log" ] && [ ! -e "$url_fail/signal.log" ]

mention="$test_root/mention"
mkdir -m 700 "$mention"; make_fixture "$mention"; make_commands "$mention" mutate
make_proc "$mention/proc" 301 1 301 301 3010 "$mention/runtime/orquesta-server-shadow" /tmp "$mention/runtime/orquesta-server-shadow" "text=$mention/uso-app"
if run_drain "$mention" dry "$mention/output.log"; then echo "residuo ambiguo ocultado por texto uso-app" >&2; exit 1; else mention_rc=$?; fi
[ "$mention_rc" -eq 3 ]
python3 - "$mention/receipt.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1])); p=next(x for x in r["before"]["processes"] if x["pid"]=="301")
assert p["classification"] == "ambiguous" and p["classification"] != "protected"
PY

# Lock exclusivo: el segundo drain no entra mientras el primero conserva flock.
lockcase="$test_root/lock"
mkdir -m 700 "$lockcase"; make_fixture "$lockcase"; make_commands "$lockcase" mutate
DRAIN_TEST_MODE=1 ORQUESTA_DRAIN_TEST_SYNTHETIC_SOCKET_FILES=1 ORQUESTA_DRAIN_TEST_HOLD_LOCK_SECONDS=2 ORQUESTA_DRAIN_PROC_ROOT="$lockcase/proc" ORQUESTA_DRAIN_RUNTIME_ROOT="$lockcase/runtime" ORQUESTA_DRAIN_STATE_ROOT="$lockcase/state" ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$lockcase/uso-app" ORQUESTA_DRAIN_BACKUP_ROOT="$lockcase/backups" ORQUESTA_DRAIN_RECEIPT="$lockcase/first.json" ORQUESTA_DRAIN_PIDFD_HELPER="$lockcase/pidfd-fake" ORQUESTA_DRAIN_CURL_COMMAND="$lockcase/curl-fake" ORQUESTA_DRAIN_TMUX_COMMAND="$lockcase/tmux-fake" "$ROOT/scripts/orquesta_server_drain.sh" --dry-run >"$lockcase/first.log" 2>&1 & first_pid=$!
for _ in $(seq 1 30); do [ -s "$lockcase/state/state/orquesta_server_drain.lock" ] && break; sleep 0.05; done
set +e
DRAIN_TEST_MODE=1 ORQUESTA_DRAIN_TEST_SYNTHETIC_SOCKET_FILES=1 ORQUESTA_DRAIN_PROC_ROOT="$lockcase/proc" ORQUESTA_DRAIN_RUNTIME_ROOT="$lockcase/runtime" ORQUESTA_DRAIN_STATE_ROOT="$lockcase/state" ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$lockcase/uso-app" ORQUESTA_DRAIN_BACKUP_ROOT="$lockcase/backups" ORQUESTA_DRAIN_RECEIPT="$lockcase/second.json" ORQUESTA_DRAIN_PIDFD_HELPER="$lockcase/pidfd-fake" ORQUESTA_DRAIN_CURL_COMMAND="$lockcase/curl-fake" ORQUESTA_DRAIN_TMUX_COMMAND="$lockcase/tmux-fake" "$ROOT/scripts/orquesta_server_drain.sh" --dry-run >"$lockcase/second.log" 2>&1
second_rc=$?
set -e
[ "$second_rc" -eq 73 ]; grep -q 'drain_lock_busy' "$lockcase/second.log"; wait "$first_pid"
echo "f3_r2_case=exclusive_lock ok"

# Hooks/proc sintética están prohibidos sin DRAIN_TEST_MODE=1.
set +e
ORQUESTA_DRAIN_PROC_ROOT="$dry/proc" ORQUESTA_DRAIN_RUNTIME_ROOT="$dry/runtime" ORQUESTA_DRAIN_STATE_ROOT="$dry/state" ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$dry/uso-app" ORQUESTA_DRAIN_BACKUP_ROOT="$dry/override-backups" ORQUESTA_DRAIN_TMUX_COMMAND="$dry/tmux-fake" "$ROOT/scripts/orquesta_server_drain.sh" --dry-run >"$test_root/override.log" 2>&1
override_rc=$?
set -e
[ "$override_rc" -ne 0 ]; grep -q 'test_override_forbidden_in_real_mode' "$test_root/override.log"

# El test no depende de rg: quitarlo de PATH no puede convertir un fallo en verde.
PATH=/usr/bin:/bin bash -n "$ROOT/scripts/orquesta_server_drain.sh" "$ROOT/scripts/test_orquesta_server_drain.sh"

echo "test_orquesta_server_drain=ok"
