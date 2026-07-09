#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"
# shellcheck source=scripts/lib/go_tool.sh
source "$repo_root/scripts/lib/go_tool.sh"

smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root="$ORQUESTA_SMOKE_ROOT"
else
  if [[ -n "${ORQUESTA_SMOKE_PARENT:-}" ]]; then
    smoke_parent="$ORQUESTA_SMOKE_PARENT"
  elif [[ -n "${ORQUESTA_CODEX_RUNTIME_WORKDIR:-}" ]]; then
    smoke_parent="${ORQUESTA_CODEX_RUNTIME_WORKDIR%/}/smokes"
  else
    smoke_parent="/workspace/runtime/smokes"
  fi
  if ! mkdir -p "$smoke_parent" 2>/dev/null; then
    smoke_parent="/workspace/orquesta-smokes"
  fi
  if ! mkdir -p "$smoke_parent" 2>/dev/null; then
    smoke_parent="${TMPDIR:-/tmp}"
  fi
  ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES="$smoke_parent${ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES:+:$ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES}"
  export ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES
  smoke_root="$(mktemp -d "$smoke_parent/orquesta-goal-first-app-server.XXXXXX")"
fi
smoke_temp_root_prepare "$smoke_root" "$smoke_root_source"

state_dir="$smoke_root/state"
project_dir="$smoke_root/project"
idle_project_dir="$smoke_root/orquesta-idle"
runtime_dir="$smoke_root/runtime"
bin_dir="$smoke_root/bin"
payload_file="$smoke_root/start_request.json"
start_response="$smoke_root/start_response.json"
observe_payload="$smoke_root/observe_request.json"
observe_response="$smoke_root/observe_response.json"
status_before_control_payload="$smoke_root/status_before_control_request.json"
status_before_control_response="$smoke_root/status_before_control_response.json"
status_after_control_payload="$smoke_root/status_after_control_request.json"
status_after_control_response="$smoke_root/status_after_control_response.json"
status_before_shutdown_payload="$smoke_root/status_before_shutdown_request.json"
status_before_shutdown_response="$smoke_root/status_before_shutdown_response.json"
control_payload="$smoke_root/run_control_request.json"
control_response="$smoke_root/run_control_response.json"
shutdown_coordination_payload="$smoke_root/shutdown_coordination_request.json"
shutdown_coordination_response="$smoke_root/shutdown_coordination_response.json"
post_stop_observe_payload="$smoke_root/observe_after_forced_stop_request.json"
post_stop_observe_response="$smoke_root/observe_after_forced_stop_response.json"
shutdown_response="$smoke_root/shutdown_response.json"
server_stdout="$smoke_root/server.stdout.log"
server_stderr="$smoke_root/server.stderr.log"
daemon_stdout="$smoke_root/codex-daemon.stdout.log"
daemon_stderr="$smoke_root/codex-daemon.stderr.log"
preflight_stdout="$smoke_root/codex-preflight.stdout.log"
preflight_stderr="$smoke_root/codex-preflight.stderr.log"
server_pid=""
base_url=""

keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
request_timeout="${ORQUESTA_GOAL_FIRST_SMOKE_REQUEST_TIMEOUT_SECONDS:-90}"
polls="${ORQUESTA_GOAL_FIRST_SMOKE_POLLS:-120}"
sleep_seconds="${ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS:-5}"
goal_backend="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"
high_consumption_mode="${ORQUESTA_GOAL_FIRST_SMOKE_HIGH_CONSUMPTION_MODE:-0}"
forced_stop_mode="${SMOKE_GOAL_FIRST_FORCED_STOP_MODE:-0}"
shutdown_coordination_mode="${SMOKE_GOAL_FIRST_SHUTDOWN_COORDINATION_MODE:-0}"
shutdown_coordination_polls="${ORQUESTA_GOAL_FIRST_SHUTDOWN_COORDINATION_POLLS:-12}"
shutdown_coordination_sleep_seconds="${ORQUESTA_GOAL_FIRST_SHUTDOWN_COORDINATION_SLEEP_SECONDS:-1}"

cleanup() {
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"
  local tmux_owner="$runtime_dir/goal-srv/owner.json"
  if [[ -f "$tmux_owner" ]] && command -v python3 >/dev/null 2>&1 && command -v tmux >/dev/null 2>&1; then
    local tmux_session
    tmux_session="$(python3 - "$tmux_owner" <<'PY' 2>/dev/null || true
import json, sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
session = str(data.get("session_name", "")).strip()
owner = str(data.get("owner_ref", "")).strip()
if owner == "orquesta-codex-goal-app-server-tmux-v0" and session.startswith("orquesta-goal-"):
    print(session)
PY
)"
    if [[ -n "$tmux_session" ]]; then
      tmux kill-session -t "$tmux_session" >/dev/null 2>&1 || true
    fi
  fi
  smoke_temp_root_cleanup "$smoke_root" "$keep_dir"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

resolve_command() {
  local raw="$1"
  if [[ -z "$raw" ]]; then
    return 1
  fi
  if [[ "$raw" == /* ]]; then
    printf '%s' "$raw"
    return 0
  fi
  command -v "$raw"
}

codex_app_server_error_reason() {
  local file="$1"
  if grep -qi "managed standalone codex install not found" "$file" 2>/dev/null; then
    printf '%s' "codex_app_server_standalone_missing"
    return 0
  fi
  if grep -Eqi "app-server-control|control socket|failed to connect" "$file" 2>/dev/null; then
    printf '%s' "codex_app_server_control_socket_missing"
    return 0
  fi
  printf '%s' "codex_app_server_unavailable"
}

codex_app_server_preflight() {
  local command_path="$1"
  : >"$preflight_stdout"
  : >"$preflight_stderr"
  if ! "$command_path" app-server --help >>"$preflight_stdout" 2>>"$preflight_stderr"; then
    echo "smoke_goal_first_app_server_preflight=blocked"
    echo "reason=codex_app_server_cli_missing"
    echo "codex_command=$command_path"
    tail -n 40 "$preflight_stderr" >&2 || true
    return 2
  fi
  if ! "$command_path" app-server daemon --help >>"$preflight_stdout" 2>>"$preflight_stderr"; then
    echo "smoke_goal_first_app_server_preflight=blocked"
    echo "reason=codex_app_server_daemon_cli_missing"
    echo "codex_command=$command_path"
    tail -n 40 "$preflight_stderr" >&2 || true
    return 2
  fi
  if "$command_path" app-server daemon version >>"$preflight_stdout" 2>>"$preflight_stderr"; then
    echo "smoke_goal_first_app_server_preflight=ok"
    echo "codex_command=$command_path"
    tail -n 1 "$preflight_stdout" || true
    return 0
  fi
  echo "smoke_goal_first_app_server_preflight=blocked"
  echo "reason=$(codex_app_server_error_reason "$preflight_stderr")"
  echo "codex_command=$command_path"
  tail -n 40 "$preflight_stderr" >&2 || true
  return 2
}

codex_app_server_ready() {
  local command_path="$1"
  "$command_path" app-server daemon version >"$daemon_stdout" 2>"$daemon_stderr"
}

configure_smoke_codex_code_home_source() {
  if [[ -n "${ORQUESTA_CODEX_CODE_HOME:-}" ]]; then
    echo "codex_app_server_auth_source=ORQUESTA_CODEX_CODE_HOME"
    return 0
  fi
  if [[ -n "${CODEX_HOME:-}" &&
    -f "$CODEX_HOME/auth.json" &&
    -f "$CODEX_HOME/config.toml" ]]; then
    export ORQUESTA_CODEX_CODE_HOME="$CODEX_HOME"
    echo "codex_app_server_auth_source=CODEX_HOME"
    return 0
  fi
  if [[ -n "${HOME:-}" &&
    -f "$HOME/.codex/auth.json" &&
    -f "$HOME/.codex/config.toml" ]]; then
    export ORQUESTA_CODEX_CODE_HOME="$HOME/.codex"
    echo "codex_app_server_auth_source=default_codex_home"
    return 0
  fi
  echo "codex_app_server_auth_source=default"
  return 0
}

json_get() {
  local file="$1"
  local expr="$2"
  python3 - "$file" "$expr" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
value = data
for part in sys.argv[2].split("."):
    if not part:
        continue
    if isinstance(value, dict):
        value = value.get(part, "")
    else:
        value = ""
if isinstance(value, bool):
    print("true" if value else "false")
elif value is None:
    print("")
else:
    print(value)
PY
}

json_contains_string() {
  local file="$1"
  local needle="$2"
  python3 - "$file" "$needle" <<'PY'
import json, sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
needle = sys.argv[2]

def walk(value):
    if isinstance(value, str):
        return needle in value
    if isinstance(value, list):
        return any(walk(item) for item in value)
    if isinstance(value, dict):
        return any(walk(item) for item in value.values())
    return False

sys.exit(0 if walk(data) else 1)
PY
}

post_autoprogramming_status_snapshot() {
  local label="$1"
  local payload="$2"
  local response="$3"
  local expected_state="${4:-visible}"
  local status
  cat >"$payload" <<JSON
{
  "request_id": "$request_id-status-$label",
  "correlation_id": "$request_id-status-$label",
  "run_ref": "$run_ref",
  "include_process_refs": true,
  "include_agent_progress": true
}
JSON
  status="$(
    curl -sS -m "$request_timeout" -o "$response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/autoprogramming/status" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-status-$label" \
      --data-binary "@$payload"
  )"
  echo "POST /api/v0/autoprogramming/status $label -> HTTP $status"
  if [[ "$status" -lt 200 || "$status" -gt 299 ]]; then
    echo "autoprogramming/status $label no devolvio 2xx:" >&2
    smoke_print_file_excerpt "$response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  python3 - "$response" "$run_ref" "$goal_ref" "$external_goal_ref" "$expected_state" "$label" <<'PY'
import json
import sys

path, run_ref, goal_ref, external_goal_ref, expected_state, label = sys.argv[1:7]
refs = {value for value in (run_ref, goal_ref, external_goal_ref) if value}
with open(path, encoding="utf-8") as fh:
    payload = json.load(fh)

matches = []

def has_ref(value):
    if isinstance(value, str):
        return value in refs
    if isinstance(value, list):
        return any(has_ref(item) for item in value)
    if isinstance(value, dict):
        return any(has_ref(item) for item in value.values())
    return False

def walk(value):
    if isinstance(value, dict):
        if has_ref(value):
            matches.append(value)
        for item in value.values():
            walk(item)
    elif isinstance(value, list):
        for item in value:
            walk(item)

walk(payload)
if not matches:
    raise SystemExit(f"autoprogramming/status {label}: run_not_visible")

if expected_state == "not_running":
    for item in matches:
        for key in ("goal_status", "status"):
            if str(item.get(key, "")).strip() == "running":
                raise SystemExit(f"autoprogramming/status {label}: run_still_running")
PY
  echo "autoprogramming_status_${label}_visible=true"
  if [[ "$expected_state" == "not_running" ]]; then
    echo "autoprogramming_status_${label}_not_running=true"
  fi
}

find_tmux_owner_file() {
  local owner
  owner="$(find "$runtime_dir" -path '*/owner.json' -type f -print -quit 2>/dev/null || true)"
  if [[ -n "$owner" ]]; then
    printf '%s' "$owner"
    return 0
  fi
  owner="$(find "${TMPDIR:-/tmp}" -maxdepth 2 -path "${TMPDIR:-/tmp}/oq-gsrv-$(id -u)-*/owner.json" -type f -print -quit 2>/dev/null || true)"
  if [[ -n "$owner" ]]; then
    printf '%s' "$owner"
    return 0
  fi
  return 1
}

json_owner_get() {
  json_get "$1" "$2"
}

tmux_pane_pid_for_session() {
  local session="$1"
  tmux display-message -p -t "$session" "#{pane_pid}" 2>/dev/null | head -n 1
}

find_tmux_socket_for_owner() {
  local owner_file="$1"
  find "$(dirname "$owner_file")" -maxdepth 1 -type s -print -quit 2>/dev/null || true
}

app_server_process_pids_for_socket() {
  local socket_path="$1"
  if [[ -z "$socket_path" ]]; then
    return 0
  fi
  ps -eo pid=,args= | python3 -c '
import os
import sys
socket = sys.argv[1]
current_pid = str(os.getpid())
for line in sys.stdin:
    parts = line.strip().split(None, 1)
    if len(parts) != 2 or parts[0] == current_pid:
        continue
    if "codex" in parts[1] and "app-server" in parts[1] and socket in parts[1]:
        print(parts[0])
' "$socket_path"
}

app_server_process_count_for_socket() {
  app_server_process_pids_for_socket "$1" | wc -l | tr -d ' '
}

stop_app_server_processes_for_socket() {
  local socket_path="$1"
  local pids pid
  pids="$(app_server_process_pids_for_socket "$socket_path" || true)"
  if [[ -z "$pids" ]]; then
    return 0
  fi
  for pid in $pids; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  for _ in $(seq 1 40); do
    if [[ "$(app_server_process_count_for_socket "$socket_path")" == "0" ]]; then
      return 0
    fi
    sleep 0.25
  done
  for pid in $(app_server_process_pids_for_socket "$socket_path" || true); do
    kill -KILL "$pid" >/dev/null 2>&1 || true
  done
  for _ in $(seq 1 40); do
    if [[ "$(app_server_process_count_for_socket "$socket_path")" == "0" ]]; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

cleanup_app_server_tmux_for_shutdown_retry() {
  local owner_file="$1"
  local session_name="$2"
  local pane_pid="$3"
  local socket_path="$4"
  if [[ -z "$owner_file" || -z "$session_name" ]]; then
    return 1
  fi
  if [[ "$session_name" != orquesta-goal-* ]]; then
    return 1
  fi
  echo "app_server_tmux_cleanup_retry=session:$session_name"
  if tmux has-session -t "$session_name" >/dev/null 2>&1; then
    tmux kill-session -t "$session_name" >/dev/null 2>&1 || true
  fi
  if [[ -n "$pane_pid" ]]; then
    for _ in $(seq 1 40); do
      if ! kill -0 "$pane_pid" >/dev/null 2>&1; then
        break
      fi
      sleep 0.25
    done
  fi
  if [[ -n "$socket_path" ]]; then
    stop_app_server_processes_for_socket "$socket_path" || return 1
    rm -f "$socket_path"
  fi
  rm -f "$owner_file"
  return 0
}

print_app_server_failure_diagnostics() {
  local generated_apps_count logs_db auth_missing
  generated_apps_count="$( (find "$project_dir/generated-apps" -type f -print -quit 2>/dev/null || true) | wc -l | tr -d ' ')"
  if [[ "$generated_apps_count" == "0" ]]; then
    echo "generated_apps_present=0" >&2
  else
    echo "generated_apps_present=1" >&2
  fi
  logs_db="$(find "$runtime_dir" -path '*/codex-home/logs_2.sqlite' -type f -print -quit 2>/dev/null || true)"
  if [[ -z "$logs_db" ]]; then
    return 0
  fi
  auth_missing="$(python3 - "$logs_db" <<'PY' 2>/dev/null || true
import sqlite3, sys
db = sys.argv[1]
needle = "%Missing bearer or basic authentication%"
fallback = "%401 Unauthorized%"
try:
    con = sqlite3.connect(db)
    row = con.execute(
        "select 1 from logs where cast(feedback_log_body as text) like ? "
        "or cast(feedback_log_body as text) like ? limit 1",
        (needle, fallback),
    ).fetchone()
except sqlite3.Error:
    row = None
print("1" if row else "0")
PY
)"
  if [[ "$auth_missing" == "1" ]]; then
    echo "codex_app_server_failure_reason=codex_app_server_auth_missing" >&2
    echo "codex_app_server_auth_missing=true" >&2
  fi
}

print_server_readiness_failure_diagnostics() {
  local addr="$1"
  local response_file status
  if [[ -z "$addr" ]]; then
    echo "readiness_diagnostic=addr_unavailable" >&2
    return 0
  fi
  response_file="$smoke_root/server_readiness_failure.json"
  status="$(curl -sS -m 2 -o "$response_file" -w "%{http_code}" "http://$addr/api/v0/server/readiness" 2>/dev/null || true)"
  if [[ -z "$status" || "$status" == "000" ]]; then
    echo "readiness_diagnostic=endpoint_unavailable" >&2
    return 0
  fi
  echo "readiness_http_status=$status" >&2
  if [[ ! -s "$response_file" ]]; then
    echo "readiness_diagnostic=body_empty" >&2
    return 0
  fi
  python3 - "$response_file" <<'PY' >&2 || true
import json
import re
import sys

max_value_len = 180
max_items = 3

def compact_code(value):
    value = str(value or "").strip()
    value = re.sub(r"[^A-Za-z0-9_.:-]+", "-", value).strip("-")
    return value[:max_value_len]

def public_text(value):
    value = str(value or "").strip()
    if not value:
        return ""
    lower = value.lower()
    forbidden = (
        "/", "\\", "home", "token", "secret", "bearer", "authorization",
        "password", "oauth", "apikey", "api_key", "prompt", "transcript",
    )
    if any(item in lower for item in forbidden):
        return "[redacted]"
    value = re.sub(r"https?://[^/@\s]+@", "http://[redacted]@", value)
    value = re.sub(r"\b[A-Za-z_]*(token|secret|password|key)[A-Za-z_]*=([^\s,;]+)", r"\1=[redacted]", value, flags=re.I)
    return value[:max_value_len]

try:
    with open(sys.argv[1], encoding="utf-8") as fh:
        payload = json.load(fh)
except Exception:
    print("readiness_diagnostic=body_unparseable")
    sys.exit(0)

for key in (
    "ready",
    "status",
    "availability_status",
    "availability_reason",
    "liveness_status",
    "startup_ready",
    "startup_status",
    "startup_message",
    "external_bridge_status",
    "external_bridge_last_error_code",
    "idle_self_improvement_goal_status",
    "idle_self_improvement_goal_reason_code",
):
    if key not in payload:
        continue
    value = payload.get(key)
    if isinstance(value, bool):
        print(f"readiness_{key}={str(value).lower()}")
    elif key.endswith("_message"):
        text = public_text(value)
        if text:
            print(f"readiness_{key}={text}")
    else:
        code = compact_code(value)
        if code:
            print(f"readiness_{key}={code}")

for index, item in enumerate(payload.get("diagnostics") or []):
    if index >= max_items or not isinstance(item, dict):
        break
    code = compact_code(item.get("code"))
    scope = compact_code(item.get("scope"))
    message = public_text(item.get("message"))
    if code:
        print(f"readiness_diagnostic_{index}_code={code}")
    if scope:
        print(f"readiness_diagnostic_{index}_scope={scope}")
    if message:
        print(f"readiness_diagnostic_{index}_message={message}")
PY
}

assert_app_server_tmux_shutdown_ready() {
  if [[ "$goal_backend" != "app_server_tmux" ]]; then
    return 0
  fi
  local owner_file session_name pane_pid socket_path shutdown_status shutdown_ready exit_pending shutdown_pid app_processes_alive
  session_name=""
  pane_pid=""
  socket_path=""
  owner_file="$(find_tmux_owner_file || true)"
  if [[ -n "$owner_file" ]]; then
    session_name="$(json_owner_get "$owner_file" "session_name")"
    if [[ -z "$session_name" ]]; then
      echo "owner.json sin session_name: $owner_file" >&2
      exit 1
    fi
    socket_path="$(find_tmux_socket_for_owner "$owner_file")"
    if tmux has-session -t "$session_name" >/dev/null 2>&1; then
      pane_pid="$(tmux_pane_pid_for_session "$session_name")"
      if [[ -z "$pane_pid" ]] || ! kill -0 "$pane_pid" >/dev/null 2>&1; then
        echo "pane_pid invalido antes del shutdown: session=$session_name pane_pid=$pane_pid" >&2
        exit 1
      fi
      if [[ -z "$socket_path" || ! -S "$socket_path" ]]; then
        echo "socket tmux no encontrado antes del shutdown: owner=$owner_file socket=$socket_path" >&2
        exit 1
      fi
    else
      echo "tmux session ya estaba cerrada antes del shutdown: $session_name"
    fi
    echo "app_server_tmux_owner_json=$owner_file"
    echo "app_server_tmux_session_name=$session_name"
    if [[ -n "$pane_pid" ]]; then
      echo "app_server_tmux_pane_pid=$pane_pid"
    fi
    if [[ -n "$socket_path" ]]; then
      echo "app_server_tmux_socket=$socket_path"
    fi
  else
    echo "app_server_tmux_owner_json=absent_before_shutdown"
  fi

  shutdown_status="$(
    curl -sS -m "$request_timeout" -o "$shutdown_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/server/shutdown" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-shutdown" \
      --data-binary "{\"request_id\":\"$request_id-shutdown\",\"correlation_id\":\"$request_id-shutdown\",\"idempotency_key\":\"idem-$request_id-shutdown\",\"requested_by\":\"orquesta-director\",\"reason\":\"smoke_goal_first_app_server_real\",\"cleanup_goal_backends\":true}"
  )"
  echo "POST /api/v0/server/shutdown -> HTTP $shutdown_status"
  if [[ "$shutdown_status" -lt 200 || "$shutdown_status" -gt 299 ]]; then
    if [[ "$shutdown_status" == "409" && "$(json_get "$shutdown_response" "status")" == "backend_still_running" ]]; then
      if cleanup_app_server_tmux_for_shutdown_retry "$owner_file" "$session_name" "$pane_pid" "$socket_path"; then
        shutdown_status="$(
          curl -sS -m "$request_timeout" -o "$shutdown_response" -w "%{http_code}" \
            -X POST "$base_url/api/v0/server/shutdown" \
            -H "Content-Type: application/json" \
            -H "Accept: application/json" \
            -H "X-Correlation-ID: $request_id-shutdown-retry" \
            --data-binary "{\"request_id\":\"$request_id-shutdown-retry\",\"correlation_id\":\"$request_id-shutdown-retry\",\"idempotency_key\":\"idem-$request_id-shutdown-retry\",\"requested_by\":\"orquesta-director\",\"reason\":\"smoke_goal_first_app_server_real_backend_cleanup_retry\",\"cleanup_goal_backends\":true}"
        )"
        echo "POST /api/v0/server/shutdown retry -> HTTP $shutdown_status"
      fi
    fi
  fi
  if [[ "$shutdown_status" -lt 200 || "$shutdown_status" -gt 299 ]]; then
    smoke_print_file_excerpt "$shutdown_response"
    exit 1
  fi
  shutdown_ready="$(json_get "$shutdown_response" "shutdown_ready")"
  if [[ "$shutdown_ready" != "true" ]]; then
    echo "shutdown_ready no fue true; respuesta:" >&2
    smoke_print_file_excerpt "$shutdown_response"
    exit 1
  fi
  exit_pending="$(json_get "$shutdown_response" "exit_pending")"
  shutdown_pid="$(json_get "$shutdown_response" "pid")"
  if [[ "$exit_pending" != "true" || -z "$shutdown_pid" ]]; then
    echo "shutdown_ready=true sin exit_pending/pid; respuesta:" >&2
    smoke_print_file_excerpt "$shutdown_response"
    exit 1
  fi
  if [[ "$shutdown_pid" != "$server_pid" ]]; then
    echo "shutdown pid inesperado: respuesta=$shutdown_pid esperado=$server_pid" >&2
    smoke_print_file_excerpt "$shutdown_response"
    exit 1
  fi
  for _ in $(seq 1 80); do
    if ! kill -0 "$shutdown_pid" >/dev/null 2>&1; then
      wait "$shutdown_pid" >/dev/null 2>&1 || true
      server_pid=""
      break
    fi
    sleep 0.25
  done
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
    echo "shutdown_ready=true pero proceso servidor sigue vivo; enviando senal cooperativa local"
    smoke_shutdown_orquesta_server "$server_pid" "" 0 25 "$runtime_dir"
    if kill -0 "$server_pid" >/dev/null 2>&1; then
      echo "el servidor siguio vivo tras shutdown_ready=true y senal cooperativa" >&2
      exit 1
    fi
    wait "$server_pid" >/dev/null 2>&1 || true
    server_pid=""
  fi
  if [[ -n "$session_name" ]] && tmux has-session -t "$session_name" >/dev/null 2>&1; then
    echo "tmux session sigue viva tras shutdown hook: $session_name" >&2
    exit 1
  fi
  if [[ -n "$owner_file" && -e "$owner_file" ]]; then
    echo "owner.json sigue existiendo tras shutdown hook: $owner_file" >&2
    exit 1
  fi
  if [[ -n "$socket_path" && -e "$socket_path" ]]; then
    echo "socket sigue existiendo tras shutdown hook: $socket_path" >&2
    exit 1
  fi
  if [[ -n "$pane_pid" ]] && kill -0 "$pane_pid" >/dev/null 2>&1; then
    echo "pane_pid sigue vivo tras shutdown hook: $pane_pid" >&2
    exit 1
  fi
  app_processes_alive="$(app_server_process_count_for_socket "$socket_path")"
  if [[ "$app_processes_alive" != "0" ]]; then
    echo "quedan procesos codex app-server para socket $socket_path: $app_processes_alive" >&2
    exit 1
  fi
  echo "app_server_tmux_shutdown_ready=true"
  echo "app_server_tmux_processes_alive=0"
}

fail_after_app_server_tmux_shutdown_ready() {
  local status="${1:-1}"
  print_app_server_failure_diagnostics
  if [[ "$goal_backend" == "app_server_tmux" && -n "$base_url" ]]; then
    echo "goal-first no cerro aceptado; comprobando shutdown app_server_tmux antes de fallar" >&2
    assert_app_server_tmux_shutdown_ready
  fi
  exit "$status"
}

shutdown_high_consumption_smoke() {
  local owner_file socket_path app_processes_alive
  socket_path=""
  owner_file="$(find_tmux_owner_file || true)"
  if [[ -n "$owner_file" ]]; then
    socket_path="$(find_tmux_socket_for_owner "$owner_file")"
  fi
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
    echo "servidor temporal sigue vivo tras cleanup BUG-088: $server_pid" >&2
    exit 1
  fi
  if [[ -n "$server_pid" ]]; then
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  server_pid=""
  if [[ -n "$owner_file" && -e "$owner_file" ]]; then
    echo "owner.json sigue existiendo tras cleanup BUG-088: $owner_file" >&2
    exit 1
  fi
  if [[ -n "$socket_path" && -e "$socket_path" ]]; then
    echo "socket sigue existiendo tras cleanup BUG-088: $socket_path" >&2
    exit 1
  fi
  app_processes_alive="$(app_server_process_count_for_socket "$socket_path")"
  if [[ "$app_processes_alive" != "0" ]]; then
    echo "quedan procesos codex app-server para socket $socket_path: $app_processes_alive" >&2
    exit 1
  fi
  echo "app_server_tmux_processes_alive=0"
}

run_forced_stop_smoke() {
  local control_status control_estado control_status_value control_final_status control_goal_status_after
  local post_stop_observe_status post_stop_goal_status post_stop_closure_status post_stop_recommended_action

  post_autoprogramming_status_snapshot "before_forced_stop" "$status_before_control_payload" "$status_before_control_response" "visible"

  cat >"$control_payload" <<JSON
{
  "request_id": "$request_id-run-control-forced-stop",
  "correlation_id": "$request_id-run-control-forced-stop",
  "action": "stop",
  "run_ref": "$run_ref",
  "requested_by": "smoke_goal_first_forced_stop_backend_real",
  "reason": "smoke forced stop backend vivo tras alto consumo",
  "forced": true,
  "evidence_refs": ["evidence-ref-smoke-goal-first-forced-stop-backend-real"]
}
JSON
  control_status="$(
    curl -sS -m "$request_timeout" -o "$control_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/runs/control" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-run-control-forced-stop" \
      --data-binary "@$control_payload"
  )"
  echo "POST /api/v0/runs/control forced stop -> HTTP $control_status"
  if [[ "$control_status" -lt 200 || "$control_status" -gt 299 ]]; then
    echo "runs/control forced stop no devolvio 2xx:" >&2
    smoke_print_file_excerpt "$control_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi

  control_estado="$(json_get "$control_response" "estado")"
  control_status_value="$(json_get "$control_response" "status")"
  control_final_status="$(json_get "$control_response" "final_status")"
  control_goal_status_after="$(json_get "$control_response" "goal_status_after")"
  echo "run_control_estado=$control_estado"
  echo "run_control_status=$control_status_value"
  echo "run_control_final_status=$control_final_status"
  echo "run_control_goal_status_after=$control_goal_status_after"
  if [[ "$control_estado" != "ok" ||
    "$control_status_value" != "stopped" ||
    "$control_final_status" != "stopped" ||
    "$control_goal_status_after" == "running" ]]; then
    echo "runs/control forced stop no confirmo estado terminal seguro:" >&2
    smoke_print_file_excerpt "$control_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi

  cat >"$post_stop_observe_payload" <<JSON
{
  "request_id": "$request_id-observe-after-forced-stop",
  "correlation_id": "$request_id-observe-after-forced-stop",
  "run_ref": "$run_ref",
  "requested_by": "smoke-goal-first-forced-stop-backend-real"
}
JSON
  post_stop_observe_status="$(
    curl -sS -m "$request_timeout" -o "$post_stop_observe_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/apps/director/goal/observe" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-observe-after-forced-stop" \
      --data-binary "@$post_stop_observe_payload"
  )"
  echo "POST /api/v0/apps/director/goal/observe after forced stop -> HTTP $post_stop_observe_status"
  if [[ "$post_stop_observe_status" != "200" && "$post_stop_observe_status" != "504" ]]; then
    echo "observe posterior a forced stop devolvio HTTP inesperado:" >&2
    smoke_print_file_excerpt "$post_stop_observe_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  post_stop_goal_status="$(json_get "$post_stop_observe_response" "goal_status")"
  post_stop_closure_status="$(json_get "$post_stop_observe_response" "closure_status")"
  post_stop_recommended_action="$(json_get "$post_stop_observe_response" "recommended_action")"
  echo "observe_after_forced_stop_goal_status=$post_stop_goal_status"
  echo "observe_after_forced_stop_closure_status=$post_stop_closure_status"
  echo "observe_after_forced_stop_recommended_action=$post_stop_recommended_action"
  if [[ "$post_stop_goal_status" == "running" || "$post_stop_goal_status" != "blocked" ]]; then
    echo "observe posterior a forced stop publico estado no terminal replanificable:" >&2
    smoke_print_file_excerpt "$post_stop_observe_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  if ! json_contains_string "$post_stop_observe_response" "evidence-ref-observe-goal-run-control-terminal" &&
    ! json_contains_string "$post_stop_observe_response" "evidence-ref-run-control-goal-forced-stop-terminal" &&
    ! json_contains_string "$post_stop_observe_response" "evidence-ref-run-control-goal-forced-terminal-reconciled" &&
    ! json_contains_string "$post_stop_observe_response" "evidence-ref-run-control-terminal-after-goal-forced-stop" &&
    ! json_contains_string "$post_stop_observe_response" "evidence-ref-run-control-terminal-after-goal-reconcile"; then
    echo "observe posterior a forced stop no conserva evidencia terminal de RunControl:" >&2
    smoke_print_file_excerpt "$post_stop_observe_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  post_autoprogramming_status_snapshot "after_forced_stop" "$status_after_control_payload" "$status_after_control_response" "not_running"

  echo "smoke_goal_first_forced_stop_backend_real=ok"
  echo "run_control_status=$control_status_value"
  echo "run_control_final_status=$control_final_status"
  echo "run_control_goal_status_after=$control_goal_status_after"
  shutdown_high_consumption_smoke
  echo "smoke_root=$smoke_root"
  exit 0
}

run_shutdown_coordination_smoke() {
  local owner_file session_name pane_pid socket_path
  local shutdown_status shutdown_status_value shutdown_ready exit_pending shutdown_pid active_work_count
  local recommended_action goal_actions_count app_processes_alive runs_requested runs_stopped run_control_statuses run_control_all_stopped

  pane_pid=""
  socket_path=""
  owner_file="$(find_tmux_owner_file || true)"
  if [[ -n "$owner_file" ]]; then
    session_name="$(json_owner_get "$owner_file" "session_name")"
    socket_path="$(find_tmux_socket_for_owner "$owner_file")"
    if [[ -n "$session_name" ]] && tmux has-session -t "$session_name" >/dev/null 2>&1; then
      pane_pid="$(tmux_pane_pid_for_session "$session_name")"
    fi
    echo "app_server_tmux_owner_json=$owner_file"
    echo "app_server_tmux_session_name=$session_name"
    if [[ -n "$pane_pid" ]]; then
      echo "app_server_tmux_pane_pid=$pane_pid"
    fi
    if [[ -n "$socket_path" ]]; then
      echo "app_server_tmux_socket=$socket_path"
    fi
  else
    session_name=""
    echo "app_server_tmux_owner_json=absent_before_shutdown_coordination"
  fi

  post_autoprogramming_status_snapshot "before_shutdown" "$status_before_shutdown_payload" "$status_before_shutdown_response" "visible"

  for i in $(seq 1 "$shutdown_coordination_polls"); do
    cat >"$shutdown_coordination_payload" <<JSON
{
  "request_id": "$request_id-shutdown-coordination-$i",
  "correlation_id": "$request_id-shutdown-coordination-$i",
  "idempotency_key": "idem-$request_id-shutdown-coordination-$i",
  "requested_by": "orquesta-director",
  "reason": "smoke_goal_first_shutdown_coordination_real",
  "forced": true,
  "cleanup_goal_backends": true,
  "evidence_refs": ["evidence-ref-smoke-goal-first-shutdown-coordination-real"]
}
JSON
    shutdown_status="$(
      curl -sS -m "$request_timeout" -o "$shutdown_coordination_response" -w "%{http_code}" \
        -X POST "$base_url/api/v0/server/shutdown" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -H "X-Correlation-ID: $request_id-shutdown-coordination-$i" \
        --data-binary "{\"request_id\":\"$request_id-shutdown-coordination-$i\",\"correlation_id\":\"$request_id-shutdown-coordination-$i\",\"idempotency_key\":\"idem-$request_id-shutdown-coordination-$i\",\"requested_by\":\"orquesta-director\",\"reason\":\"smoke_goal_first_shutdown_coordination_real\",\"forced\":true,\"cleanup_goal_backends\":true,\"evidence_refs\":[\"evidence-ref-smoke-goal-first-shutdown-coordination-real\"]}"
    )"
    echo "POST /api/v0/server/shutdown coordination poll=$i -> HTTP $shutdown_status"
    if [[ "$shutdown_status" -lt 200 || "$shutdown_status" -gt 299 ]] && [[ "$shutdown_status" != "409" ]]; then
      smoke_print_file_excerpt "$shutdown_coordination_response"
      fail_after_app_server_tmux_shutdown_ready 1
    fi

    shutdown_status_value="$(json_get "$shutdown_coordination_response" "status")"
    shutdown_ready="$(json_get "$shutdown_coordination_response" "shutdown_ready")"
    exit_pending="$(json_get "$shutdown_coordination_response" "exit_pending")"
    shutdown_pid="$(json_get "$shutdown_coordination_response" "pid")"
    active_work_count="$(json_get "$shutdown_coordination_response" "active_work_count")"
    runs_requested="$(json_get "$shutdown_coordination_response" "runs_requested")"
    runs_stopped="$(json_get "$shutdown_coordination_response" "runs_stopped")"
    run_control_statuses="$(python3 - "$shutdown_coordination_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    payload = json.load(fh)
runs = payload.get("runs") or []
print(",".join(str(run.get("control_status", "")).strip() for run in runs))
PY
)"
    run_control_all_stopped="$(python3 - "$shutdown_coordination_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    payload = json.load(fh)
runs = payload.get("runs") or []
print("true" if runs and all(str(run.get("control_status", "")).strip() == "stopped" for run in runs) else "false")
PY
)"
    recommended_action="$(json_get "$shutdown_coordination_response" "recommended_action")"
    goal_actions_count="$(python3 - "$shutdown_coordination_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(len(json.load(fh).get("goal_actions") or []))
PY
)"
    echo "shutdown_coordination_poll=$i status=$shutdown_status_value shutdown_ready=$shutdown_ready runs_requested=${runs_requested:-0} runs_stopped=${runs_stopped:-0} run_control_statuses=$run_control_statuses all_runs_stopped=$run_control_all_stopped active_work_count=${active_work_count:-0} goal_actions=$goal_actions_count recommended_action=$recommended_action"

    if [[ "$shutdown_ready" == "true" ]]; then
      if [[ "${runs_requested:-0}" -lt 1 || "${runs_stopped:-0}" -lt "${runs_requested:-0}" || "$run_control_all_stopped" != "true" ]]; then
        echo "shutdown coordination listo sin coordinar run goal-first fuera de cola:" >&2
        smoke_print_file_excerpt "$shutdown_coordination_response"
        exit 1
      fi
      if [[ "$exit_pending" != "true" || -z "$shutdown_pid" ]]; then
        echo "shutdown coordination listo sin exit_pending/pid:" >&2
        smoke_print_file_excerpt "$shutdown_coordination_response"
        exit 1
      fi
      if [[ "$shutdown_pid" != "$server_pid" ]]; then
        echo "shutdown coordination pid inesperado: respuesta=$shutdown_pid esperado=$server_pid" >&2
        smoke_print_file_excerpt "$shutdown_coordination_response"
        exit 1
      fi
      for _ in $(seq 1 80); do
        if ! kill -0 "$shutdown_pid" >/dev/null 2>&1; then
          wait "$shutdown_pid" >/dev/null 2>&1 || true
          server_pid=""
          break
        fi
        sleep 0.25
      done
      if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
        echo "shutdown coordination ready=true pero proceso servidor sigue vivo" >&2
        exit 1
      fi
      if [[ -n "$session_name" ]] && tmux has-session -t "$session_name" >/dev/null 2>&1; then
        echo "tmux session sigue viva tras shutdown coordination: $session_name" >&2
        exit 1
      fi
      if [[ -n "$owner_file" && -e "$owner_file" ]]; then
        echo "owner.json sigue existiendo tras shutdown coordination: $owner_file" >&2
        exit 1
      fi
      if [[ -n "$socket_path" && -e "$socket_path" ]]; then
        echo "socket sigue existiendo tras shutdown coordination: $socket_path" >&2
        exit 1
      fi
      if [[ -n "$pane_pid" ]] && kill -0 "$pane_pid" >/dev/null 2>&1; then
        echo "pane_pid sigue vivo tras shutdown coordination: $pane_pid" >&2
        exit 1
      fi
      app_processes_alive="$(app_server_process_count_for_socket "$socket_path")"
      if [[ "$app_processes_alive" != "0" ]]; then
        echo "quedan procesos codex app-server tras shutdown coordination: $app_processes_alive" >&2
        exit 1
      fi
      echo "app_server_tmux_shutdown_ready=true"
      echo "shutdown_coordination_runs_requested=$runs_requested"
      echo "shutdown_coordination_runs_stopped=$runs_stopped"
      echo "shutdown_coordination_run_control_statuses=$run_control_statuses"
      echo "shutdown_coordination_all_runs_stopped=$run_control_all_stopped"
      echo "app_server_tmux_processes_alive=0"
      echo "smoke_goal_first_shutdown_coordination_real=ok"
      echo "smoke_root=$smoke_root"
      exit 0
    fi

    sleep "$shutdown_coordination_sleep_seconds"
  done

  echo "shutdown coordination no llego a shutdown_ready=true; ultima respuesta:" >&2
  smoke_print_file_excerpt "$shutdown_coordination_response"
  fail_after_app_server_tmux_shutdown_ready 1
}

bug088_second_artifact_exists() {
  [[ -s "$project_dir/generated-apps/bug088_second_artifact.txt" ]]
}

need_cmd python3

codex_command="$(resolve_command "${ORQUESTA_CODEX_COMMAND:-codex}")" || {
  echo "no se pudo resolver ORQUESTA_CODEX_COMMAND/codex" >&2
  exit 2
}
if [[ ! -x "$codex_command" ]]; then
  echo "ORQUESTA_CODEX_COMMAND no es ejecutable: $codex_command" >&2
  exit 2
fi

case "$goal_backend" in
  app_server_tmux) ;;
  app_server_proxy)
    echo "ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy no es backend operativo para este smoke; usa app_server_tmux" >&2
    exit 2
    ;;
  *)
    echo "ORQUESTA_CODEX_GOAL_BACKEND no soportado para este smoke: $goal_backend" >&2
    exit 2
    ;;
esac

if [[ "${ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY:-0}" == "1" ]]; then
  if [[ "$goal_backend" == "app_server_tmux" ]]; then
    need_cmd tmux
    "$codex_command" app-server --help >"$preflight_stdout" 2>"$preflight_stderr" || {
      echo "smoke_goal_first_app_server_preflight=blocked"
      echo "reason=codex_app_server_cli_missing"
      echo "codex_command=$codex_command"
      echo "goal_backend=$goal_backend"
      tail -n 40 "$preflight_stderr" >&2 || true
      exit 2
    }
    echo "smoke_goal_first_app_server_preflight=ok"
    echo "codex_command=$codex_command"
    echo "goal_backend=$goal_backend"
    exit 0
  fi
  codex_app_server_preflight "$codex_command"
  exit $?
fi

if [[ "${ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM:-0}" != "1" ||
  "${ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED:-0}" != "1" ]]; then
  echo "confirmacion doble requerida: exporta ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 y ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1" >&2
  exit 2
fi

if [[ -n "${ORQUESTA_OPES_BASE_URL:-}" || -n "${OPES_BASE_URL:-}" ]]; then
  echo "OPES debe estar desactivado para este smoke" >&2
  exit 2
fi

orquesta_go_tool_ensure_path
need_cmd go
need_cmd curl

mkdir -p "$state_dir" "$project_dir" "$idle_project_dir" "$runtime_dir" "$bin_dir" \
  "$project_dir/docs" \
  "$project_dir/modulos/orquesta-factory/docs" \
  "$project_dir/modulos/orquesta-web/docs"

if [[ "$high_consumption_mode" == "1" ]]; then
  cat >"$project_dir/AGENTS.md" <<'EOF'
# Smoke temporal goal-first BUG-088

Trabaja solo dentro de este proyecto temporal. Primero escribe
`generated-apps/checkpoint_started_bug088.txt` con una linea de estado. Despues
intenta escribir `generated-apps/bug088_second_artifact.txt`. No publiques HOME,
tokens ni rutas privadas. No escribas `ORQUESTA_GOAL_RESULT_V0` hasta que exista
un segundo artefacto no checkpoint.
EOF
else
  cat >"$project_dir/AGENTS.md" <<'EOF'
# Smoke temporal goal-first

Trabaja solo dentro de este proyecto temporal. Crea una app pequena, hexagonal y
verificable bajo `generated-apps/`. No publiques HOME, tokens ni rutas privadas.
EOF
fi

cat >"$project_dir/docs/orquesta_goal_first_codex_2026-06-25.md" <<'EOF'
# Goal-first smoke

Codex Goal actua como director operativo interno. Orquesta valida el cierre
externamente. Al terminar, devuelve `ORQUESTA_GOAL_RESULT_V0` con refs opacas de
artefactos y evidencias producidas.
EOF

cat >"$project_dir/modulos/orquesta-factory/docs/contratos.md" <<'EOF'
# Contrato factory minimo

La app generada debe separar dominio/aplicacion, puertos y adaptadores. Para el
smoke basta una app local pequena con pruebas o verificacion documentada.
EOF

cat >"$project_dir/modulos/orquesta-web/docs/guia_nueva_app_opciones_2026-06-25.md" <<'EOF'
# Opciones nueva app

Arquitectura hexagonal por defecto si no rompe el contrato. Accesibilidad basica
aceptable para smoke. Persistencia puede ser en memoria si la solicitud no exige
base de datos.
EOF

if [[ "$goal_backend" == "app_server_tmux" ]]; then
  need_cmd tmux
  "$codex_command" app-server --help >"$daemon_stdout" 2>"$daemon_stderr" || {
    echo "codex app-server no esta disponible para tmux; reason=codex_app_server_cli_missing; stderr:" >&2
    tail -n 80 "$daemon_stderr" >&2 || true
    exit 2
  }
  echo "Codex app-server preparado para tmux"
else
  echo "comprobando daemon Codex app-server..."
  if codex_app_server_ready "$codex_command"; then
    echo "daemon Codex app-server accesible"
  else
    echo "arrancando daemon Codex app-server si hace falta..."
    "$codex_command" app-server daemon start >"$daemon_stdout" 2>"$daemon_stderr" || {
      echo "no se pudo arrancar codex app-server daemon; reason=$(codex_app_server_error_reason "$daemon_stderr"); stderr:" >&2
      tail -n 80 "$daemon_stderr" >&2 || true
      exit 2
    }
  fi
fi

echo "compilando servidor temporal..."
go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server

if [[ "$high_consumption_mode" == "1" ]]; then
  request_id="request-ref-goal-first-bug088-$(date -u +%Y%m%dT%H%M%SZ)"
  app_name="Smoke Goal First BUG088"
  app_objective="BUG-088: crear primero generated-apps/checkpoint_started_bug088.txt; despues crear generated-apps/bug088_second_artifact.txt y resultado final solo si ese segundo artefacto existe."
  app_description="Smoke acotado de alto consumo goal-first real: validar que Orquesta permite segundo artefacto util o prepara replan gobernado por el observador residente sin dejar app-server residual."
  app_restriction_one="checkpoint_started_bug088.txt debe ser el primer artefacto durable"
  app_restriction_two="no escribir resultado terminal antes del segundo artefacto"
else
  request_id="request-ref-goal-first-real-$(date -u +%Y%m%dT%H%M%SZ)"
  app_name="Smoke Goal First"
  app_objective="Crear una app local minima para gestionar notas con API HTTP y una pantalla HTML sencilla."
  app_description="Smoke acotado de goal-first real: generar una app pequena bajo generated-apps, separando dominio/aplicacion, puertos y adaptadores, con una verificacion local documentada."
  app_restriction_one="hexagonal puro"
  app_restriction_two="entrega pequena y verificable"
fi
cat >"$payload_file" <<JSON
{
  "request_id": "$request_id",
  "correlation_id": "$request_id",
  "director_execution_mode": "goal_first",
  "app_spec_request": {
    "schema_version": "app_spec_request.v0",
    "request_id": "$request_id",
    "source": "orquesta-smoke",
    "locale": "es",
    "request_kind": "crear_app_completa",
    "execution_mode": "normal",
    "nombre": "$app_name",
    "objetivo": "$app_objective",
    "descripcion": "$app_description",
    "tipo_app": "mixed",
    "usuarios_objetivo": ["operador de smoke"],
    "plataformas": ["web", "api"],
    "preferencias_tecnicas": {
      "lenguaje": "go",
      "framework": "net/http",
      "arquitectura": "hexagonal",
      "restricciones": ["app pequena", "sin dependencias externas obligatorias", "sin tocar rutas fuera de generated-apps"]
    },
    "calidad": {
      "pruebas": "basica",
      "accesibilidad": "basica",
      "observabilidad": false
    },
    "i18n": {
      "enabled": true,
      "default_locale": "es",
      "locales": ["es"]
    },
    "restricciones": [
      "$app_restriction_one",
      "$app_restriction_two",
      "resultado final con marcador ORQUESTA_GOAL_RESULT_V0"
    ]
  }
}
JSON

export ORQUESTA_SERVER_ADDR="127.0.0.1:0"
export ORQUESTA_SERVER_STATE_DIR="$state_dir"
export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR="$idle_project_dir"
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=0
export ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false
if [[ "$high_consumption_mode" == "1" ]]; then
  export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-true}"
  export ORQUESTA_SERVER_GOAL_OBSERVER_INTERVAL_MS="${ORQUESTA_SERVER_GOAL_OBSERVER_INTERVAL_MS:-1000}"
  export ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS="${ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS:-5}"
  export ORQUESTA_SERVER_GOAL_OBSERVER_FINGERPRINT_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_FINGERPRINT_ENABLED:-false}"
  export ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS="${ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS:-1}"
  export ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_MAX_WAIT_SECONDS="${ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_MAX_WAIT_SECONDS:-1}"
  export ORQUESTA_AUTOPROGRAMMING_NO_CHECKPOINT_WARNING_MAX_WAIT_SECONDS="${ORQUESTA_AUTOPROGRAMMING_NO_CHECKPOINT_WARNING_MAX_WAIT_SECONDS:-1}"
else
  export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-false}"
fi
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir"
export ORQUESTA_CODEX_COMMAND="$codex_command"
export ORQUESTA_CODEX_PATH="${ORQUESTA_CODEX_PATH:-$PATH}"
configure_smoke_codex_code_home_source
export ORQUESTA_CODEX_GOAL_BACKEND="$goal_backend"
export ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"
export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
# Smoke aislado: el proyecto, runtime y CODEX_HOME son temporales bajo smoke_root.
# workspace-write en app-server no materializa herramientas locales de forma
# fiable en este entorno; el smoke valida cierre real y conserva override por env.
export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-danger-full-access}"
export ORQUESTA_CODEX_MODEL="${ORQUESTA_CODEX_MODEL:-gpt-5.5}"
export ORQUESTA_CODEX_REASONING_EFFORT="${ORQUESTA_CODEX_REASONING_EFFORT:-medium}"
export ORQUESTA_OPES_BASE_URL=""
export OPES_BASE_URL=""

echo "arrancando orquesta-server..."
"$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
server_pid="$!"

state_file="$state_dir/orquesta_server_state_v0.json"
server_addr=""
server_ready="0"
for _ in $(seq 1 80); do
  if [[ -s "$state_file" ]]; then
    server_addr="$(python3 - "$state_file" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("addr", ""))
PY
)"
    if [[ -n "$server_addr" ]] && curl -fsS -m 2 "http://$server_addr/api/v0/server/readiness" >/dev/null; then
      server_ready="1"
      break
    fi
  fi
  sleep 0.5
done

if [[ "$server_ready" != "1" ]]; then
  echo "el servidor no llego a readiness; addr=$server_addr; stderr:" >&2
  print_server_readiness_failure_diagnostics "$server_addr"
  tail -n 80 "$server_stderr" >&2 || true
  exit 1
fi

base_url="http://$server_addr"
echo "servidor listo: $base_url"

start_status="$(
  curl -sS -m "$request_timeout" -o "$start_response" -w "%{http_code}" \
    -X POST "$base_url/api/v0/apps/director" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: $request_id" \
    --data-binary "@$payload_file"
)"

echo "POST /api/v0/apps/director -> HTTP $start_status"
if [[ "$start_status" -lt 200 || "$start_status" -gt 299 ]]; then
  smoke_print_file_excerpt "$start_response"
  exit 1
fi

run_ref="$(json_get "$start_response" "run_ref")"
goal_ref="$(json_get "$start_response" "goal_ref")"
external_goal_ref="$(json_get "$start_response" "external_goal_ref")"
if [[ -z "$run_ref" || -z "$goal_ref" || -z "$external_goal_ref" ]]; then
  echo "respuesta inicial sin refs goal-first:" >&2
  smoke_print_file_excerpt "$start_response"
  exit 1
fi

echo "run_ref=$run_ref"
echo "goal_ref=$goal_ref"
echo "external_goal_ref=$external_goal_ref"

terminal="0"
for i in $(seq 1 "$polls"); do
  cat >"$observe_payload" <<JSON
{
  "request_id": "$request_id-observe-$i",
  "correlation_id": "$request_id-observe-$i",
  "run_ref": "$run_ref",
  "requested_by": "smoke-goal-first-app-server-real"
}
JSON
  observe_status="$(
    curl -sS -m "$request_timeout" -o "$observe_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/apps/director/goal/observe" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-observe-$i" \
      --data-binary "@$observe_payload"
  )"
  if [[ "$observe_status" -lt 200 || "$observe_status" -gt 299 ]]; then
    echo "observe HTTP $observe_status" >&2
    smoke_print_file_excerpt "$observe_response"
    if grep -q "codex_goal_observation_rejected" "$observe_response" 2>/dev/null; then
      echo "poll=$i observe_status=$observe_status goal_status=transient_observation_rejected"
      sleep "$sleep_seconds"
      continue
    fi
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  goal_status="$(json_get "$observe_response" "goal_status")"
  run_status="$(json_get "$observe_response" "run_status")"
  director_execution_mode="$(json_get "$observe_response" "director_execution_mode")"
  closure_status="$(json_get "$observe_response" "closure_status")"
  closure_accepted="$(json_get "$observe_response" "closure_accepted")"
  echo "poll=$i mode=$director_execution_mode goal_status=$goal_status run_status=$run_status closure_status=$closure_status closure_accepted=$closure_accepted"
  if [[ "$director_execution_mode" == "goal_first" && "$goal_status" == "complete" && "$run_status" == "cerrada" && "$closure_status" == "accepted" && "$closure_accepted" == "true" ]]; then
    terminal="1"
    break
  fi
  if [[ "$high_consumption_mode" == "1" ]] &&
    json_contains_string "$observe_response" "codex_app_server_goal_status_active_high_token_usage" &&
    json_contains_string "$observe_response" "evidence-ref-goal-observer-high-consumption-stop-requested" &&
    { json_contains_string "$observe_response" "evidence-ref-goal-materialized-partial-artifacts-written" ||
      bug088_second_artifact_exists; }; then
    if [[ "$shutdown_coordination_mode" == "1" ]]; then
      run_shutdown_coordination_smoke
    fi
    if [[ "$forced_stop_mode" == "1" ]]; then
      run_forced_stop_smoke
    fi
    terminal="bug088_second_artifact"
    break
  fi
  if [[ "$goal_status" == "blocked" || "$goal_status" == "invalid" || "$run_status" == "bloqueada" || "$closure_status" == "blocked" ]]; then
    if [[ "$high_consumption_mode" == "1" ]] &&
      { json_contains_string "$observe_response" "checkpoint_only_high_consumption" ||
        json_contains_string "$observe_response" "goal_active_no_checkpoint_high_consumption"; } &&
      json_contains_string "$observe_response" "replan_narrow_context"; then
      if [[ "$shutdown_coordination_mode" == "1" ]]; then
        run_shutdown_coordination_smoke
      fi
      if [[ "$forced_stop_mode" == "1" ]]; then
        echo "forced-stop smoke no encontro backend vivo antes de replan alto consumo:" >&2
        smoke_print_file_excerpt "$observe_response"
        fail_after_app_server_tmux_shutdown_ready 1
      fi
      terminal="bug088_replan"
      break
    fi
    echo "goal-first termino bloqueado; respuesta final:" >&2
    smoke_print_file_excerpt "$observe_response"
    fail_after_app_server_tmux_shutdown_ready 1
  fi
  sleep "$sleep_seconds"
done

if [[ "$terminal" != "1" && "$terminal" != "bug088_replan" && "$terminal" != "bug088_second_artifact" ]] &&
  [[ "$high_consumption_mode" == "1" ]] &&
  json_contains_string "$observe_response" "codex_app_server_goal_status_active_high_token_usage" &&
  json_contains_string "$observe_response" "evidence-ref-goal-observer-high-consumption-stop-requested" &&
  bug088_second_artifact_exists; then
  terminal="bug088_second_artifact"
fi

if [[ "$terminal" != "1" && "$terminal" != "bug088_replan" && "$terminal" != "bug088_second_artifact" ]]; then
  echo "timeout esperando cierre aceptado goal-first; ultimo observe:" >&2
  smoke_print_file_excerpt "$observe_response"
  fail_after_app_server_tmux_shutdown_ready 1
fi

if [[ "$terminal" == "bug088_replan" ]]; then
  echo "smoke_goal_first_high_consumption_real=ok"
  if json_contains_string "$observe_response" "checkpoint_only_high_consumption"; then
    echo "bug088_path=checkpoint_only_replan"
  else
    echo "bug088_path=no_checkpoint_replan"
  fi
  echo "recommended_action=replan_narrow_context"
  shutdown_high_consumption_smoke
  echo "smoke_root=$smoke_root"
  exit 0
fi

if [[ "$terminal" == "bug088_second_artifact" ]]; then
  echo "smoke_goal_first_high_consumption_real=ok"
  echo "bug088_path=second_artifact_or_partial_artifacts"
  echo "recommended_action=review_partial_artifacts"
  shutdown_high_consumption_smoke
  echo "smoke_root=$smoke_root"
  exit 0
fi

artifact_count="$(python3 - "$observe_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(len(json.load(fh).get("artifact_refs", [])))
PY
)"
evidence_count="$(python3 - "$observe_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(len(json.load(fh).get("evidence_refs", [])))
PY
)"

if [[ "$artifact_count" -lt 1 ]]; then
  echo "cierre aceptado sin artifact_refs" >&2
  smoke_print_file_excerpt "$observe_response"
  exit 1
fi
if [[ "$evidence_count" -lt 1 ]]; then
  echo "cierre aceptado sin evidence_refs" >&2
  smoke_print_file_excerpt "$observe_response"
  exit 1
fi

if [[ "$high_consumption_mode" == "1" ]]; then
  echo "smoke_goal_first_high_consumption_real=ok"
  echo "bug088_path=second_artifact_or_terminal_artifact"
fi
echo "smoke_goal_first_app_server_real=ok"
echo "artifact_refs=$artifact_count"
echo "evidence_refs=$evidence_count"
assert_app_server_tmux_shutdown_ready
echo "smoke_root=$smoke_root"
