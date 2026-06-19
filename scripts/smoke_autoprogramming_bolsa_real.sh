#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_id="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
smoke_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-bolsa-real.XXXXXX")}"
smoke_temp_root_prepare "$smoke_root" "$smoke_root_source"

state_dir="$smoke_root/state"
project_dir="$smoke_root/project"
runtime_dir="$smoke_root/runtime"
bin_dir="$smoke_root/bin"
payload_dir="$smoke_root/payloads"
result_dir="$smoke_root/results"
server_stdout="$smoke_root/orquesta-server.stdout.log"
server_stderr="$smoke_root/orquesta-server.stderr.log"
app_stdout="$smoke_root/bolsa-server.stdout.log"
app_stderr="$smoke_root/bolsa-server.stderr.log"
server_pid=""
app_pid=""
base_url=""
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
request_timeout="${ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS:-180}"
status_polls="${ORQUESTA_BOLSA_REAL_STATUS_POLLS:-120}"
status_sleep="${ORQUESTA_BOLSA_REAL_STATUS_SLEEP_SECONDS:-10}"
supervise_ticks="${ORQUESTA_BOLSA_REAL_SUPERVISE_TICKS:-2}"

cleanup() {
  if [[ -n "$app_pid" ]] && kill -0 "$app_pid" >/dev/null 2>&1; then
    kill -INT "$app_pid" >/dev/null 2>&1 || true
    wait "$app_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$base_url" ]]; then
    curl -sS -m 5 -X POST "$base_url/api/v0/server/shutdown" \
      -H "Content-Type: application/json" \
      -d '{"request_id":"req-bolsa-real-smoke-shutdown","correlation_id":"corr-bolsa-real-smoke-shutdown","forced":true}' \
      >/dev/null 2>&1 || true
  fi
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -INT "$server_pid" >/dev/null 2>&1 || true
    for _ in $(seq 1 30); do
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        break
      fi
      sleep 0.2
    done
    if kill -0 "$server_pid" >/dev/null 2>&1; then
      kill -TERM "$server_pid" >/dev/null 2>&1 || true
    fi
    wait "$server_pid" >/dev/null 2>&1 || true
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

state_field() {
  local field="$1"
  python3 - "$state_dir/orquesta_server_state_v0.json" "$field" <<'PY'
import json
import sys

try:
    with open(sys.argv[1], encoding="utf-8") as fh:
        data = json.load(fh)
except FileNotFoundError:
    print("")
    raise SystemExit(0)
print(data.get(sys.argv[2], ""))
PY
}

wait_server_ready() {
  for _ in $(seq 1 100); do
    local addr
    addr="$(state_field addr)"
    local pid
    pid="$(state_field pid)"
    if [[ "$pid" == "$server_pid" && -n "$addr" ]] &&
      curl -fsS -m 2 "http://$addr/api/v0/server/readiness" >/dev/null; then
      base_url="http://$addr"
      return 0
    fi
    sleep 0.25
  done
  return 1
}

post_json_required() {
  local path="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(
    curl -sS -m "$request_timeout" -o "$output" -w "%{http_code}" \
      -X POST "$base_url$path" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary "@$payload" || true
  )"
  echo "POST $path -> HTTP $status"
  case "$status" in
    2*) return 0 ;;
    *)
      head -c "${SMOKE_ERROR_BYTES:-4000}" "$output" >&2 || true
      echo >&2
      return 1
      ;;
  esac
}

write_prepare_payload() {
  local output="$1"
  local source_spec="$2"
  python3 - "$output" "$source_spec" "$smoke_id" "${ORQUESTA_BOLSA_REAL_RUN_REF:-}" <<'PY'
import json
import sys

output, source_spec, smoke_id, run_ref_override = sys.argv[1:5]
with open(source_spec, encoding="utf-8") as fh:
    payload = json.load(fh)
run_ref = (run_ref_override or f"bolsa-real-smoke-{smoke_id}").strip()
payload["request_id"] = f"req-{run_ref}"
payload["correlation_id"] = f"corr-{run_ref}"
payload["requested_by"] = "orquesta-bolsa-real-smoke"
payload["occurred_at"] = "2026-06-19T12:00:00Z"
payload["max_bursts"] = int(payload.get("max_bursts") or 8)
payload["max_steps_per_burst"] = int(payload.get("max_steps_per_burst") or 8)
payload["max_dispatches_per_wait"] = int(payload.get("max_dispatches_per_wait") or 6)
payload["max_commands"] = int(payload.get("max_commands") or 32)
payload["max_outbox_per_cycle"] = int(payload.get("max_outbox_per_cycle") or 16)
request = payload.setdefault("autoprogramming_request", {})
request["request_ref"] = run_ref
request["worktree_ref"] = f"worktree-{run_ref}"
request["branch_ref"] = f"branch-{run_ref}"
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_status_payload() {
  local output="$1"
  local run_ref="$2"
  python3 - "$output" "$smoke_id" "$run_ref" <<'PY'
import json
import sys

output, smoke_id, run_ref = sys.argv[1:4]
payload = {
    "request_id": f"req-bolsa-real-status-{smoke_id}",
    "correlation_id": f"corr-bolsa-real-status-{smoke_id}",
    "run_ref": run_ref,
    "queue_ref": "global",
    "include_process_refs": True,
    "include_agent_progress": True,
    "include_agent_usage": True,
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_supervise_payload() {
  local output="$1"
  local run_ref="$2"
  python3 - "$output" "$smoke_id" "$run_ref" "$supervise_ticks" <<'PY'
import json
import sys

output, smoke_id, run_ref, ticks = sys.argv[1:5]
payload = {
    "request_id": f"req-bolsa-real-supervise-{smoke_id}",
    "correlation_id": f"corr-bolsa-real-supervise-{smoke_id}",
    "run_ref": run_ref,
    "queue_ref": "global",
    "continue_message": "sigue hasta cierre causal",
    "max_ticks": int(ticks),
    "max_runs_per_tick": 1,
    "max_executions": 70,
    "allow_repeated_runs": False,
    "max_bursts": 8,
    "max_steps_per_burst": 8,
    "max_dispatches_per_wait": 6,
    "max_commands": 32,
    "max_outbox_per_cycle": 16,
    "max_decision_cycles": 8,
    "max_external_waits": 8,
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

extract_prepare_run_ref() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("estado") != "ok" or data.get("accepted") is not True or not data.get("run_ref"):
    raise SystemExit(f"prepare-run inesperado: {data}")
print(data["run_ref"])
PY
}

status_is_closed_or_blocked() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
run = data.get("run") or {}
stats = run.get("stats") or {}
counts = stats.get("counts") or {}
closure = stats.get("closure") or {}
progress = stats.get("progress") or {}
print(
    "status_snapshot "
    f"status={stats.get('status','')} "
    f"phase={stats.get('current_phase','')} "
    f"tasks={counts.get('tasks_closed',0)}/{counts.get('tasks_total',0)} "
    f"open={counts.get('tasks_open',0)} "
    f"agents_in_flight={counts.get('agents_in_flight',0)} "
    f"failed={counts.get('agents_failed',0)} "
    f"deliveries={counts.get('deliveries',0)} "
    f"reviews={counts.get('review_results',0)} "
    f"closures={counts.get('closures',0)} "
    f"percent={progress.get('percent_complete',0)} "
    f"closure={closure.get('status','')}"
)
if closure.get("closed") is True and int(counts.get("tasks_open") or 0) == 0:
    raise SystemExit(0)
if closure.get("blocked") is True:
    status = str(stats.get("status") or "").strip().lower()
    if status in {"activa", "active"}:
        raise SystemExit(1)
    print("blocked_by=" + ",".join(closure.get("blocked_by") or []))
    print("blocker_refs=" + ",".join(closure.get("blocker_refs") or []))
    raise SystemExit(2)
raise SystemExit(1)
PY
}

wait_run_closed() {
  local run_ref="$1"
  local status_payload="$payload_dir/status.json"
  local status_response="$result_dir/status.json"
  local supervise_payload="$payload_dir/supervise.json"
  local supervise_response="$result_dir/supervise.json"
  write_status_payload "$status_payload" "$run_ref"
  write_supervise_payload "$supervise_payload" "$run_ref"
  for poll in $(seq 1 "$status_polls"); do
    post_json_required "/api/v0/autoprogramming/status" "$status_payload" "$status_response"
    set +e
    status_is_closed_or_blocked "$status_response"
    local status_code=$?
    set -e
    if [[ "$status_code" == "0" ]]; then
      echo "bolsa_real_run_closed=true"
      return 0
    fi
    if [[ "$status_code" == "2" ]]; then
      echo "bolsa_real_run_blocked=true" >&2
      return 1
    fi
    post_json_required "/api/v0/runs/supervise" "$supervise_payload" "$supervise_response"
    echo "poll=$poll sleeping=${status_sleep}s"
    sleep "$status_sleep"
  done
  echo "timeout esperando cierre de $run_ref" >&2
  return 1
}

prepare_project_dir() {
  local source_dir="${ORQUESTA_BOLSA_APP_SOURCE_DIR:-$repo_root/../Bolsa_Diputacion_app}"
  local mode="${ORQUESTA_BOLSA_PROJECT_MODE:-copy}"
  mkdir -p "$project_dir"
  case "$mode" in
    copy)
      if [[ ! -d "$source_dir" ]]; then
        echo "falta ORQUESTA_BOLSA_APP_SOURCE_DIR valido: $source_dir" >&2
        exit 2
      fi
      cp -a "$source_dir"/. "$project_dir"/
      ;;
    empty)
      ;;
    *)
      echo "ORQUESTA_BOLSA_PROJECT_MODE invalido: $mode" >&2
      exit 2
      ;;
  esac
  echo "bolsa_project_dir=$project_dir"
  echo "bolsa_project_mode=$mode"
}

start_orquesta_server() {
  local codex_command="$1"
  ORQUESTA_SERVER_ADDR="127.0.0.1:0" \
  ORQUESTA_SERVER_STATE_DIR="$state_dir" \
  ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir" \
  ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir" \
  ORQUESTA_CODEX_HOME="${ORQUESTA_CODEX_HOME:-$HOME}" \
  ORQUESTA_CODEX_CODE_HOME="${ORQUESTA_CODEX_CODE_HOME:-$HOME/.codex}" \
  ORQUESTA_CODEX_COMMAND="$codex_command" \
  ORQUESTA_CODEX_PATH="${ORQUESTA_CODEX_PATH:-$PATH}" \
  ORQUESTA_CODEX_MODEL="${ORQUESTA_CODEX_MODEL:-gpt-5.5}" \
  ORQUESTA_CODEX_REASONING_EFFORT="${ORQUESTA_CODEX_REASONING_EFFORT:-high}" \
  ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}" \
  ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-workspace-write}" \
  ORQUESTA_CODEX_DEFAULT_PARENTS_PER_TICK="${ORQUESTA_CODEX_DEFAULT_PARENTS_PER_TICK:-70}" \
  ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-1000}" \
  ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS="${ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS:-0}" \
  ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER="${ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER:-0}" \
  ORQUESTA_OPES_BASE_URL="" \
  OPES_BASE_URL="" \
    "$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
  server_pid="$!"
  if ! wait_server_ready; then
    echo "orquesta-server temporal no llego a readiness OK" >&2
    tail -n 120 "$server_stderr" >&2 || true
    exit 1
  fi
  echo "orquesta_base_url=$base_url"
}

start_bolsa_app_if_present() {
  if [[ ! -d "$project_dir/cmd/bolsa-server" ]]; then
    echo "skip_bolsa_app_start=no_cmd_bolsa_server"
    return 0
  fi
  local app_addr="${ORQUESTA_BOLSA_APP_ADDR:-127.0.0.1:0}"
  (cd "$project_dir" && BOLSA_HTTP_ADDR="$app_addr" go run -buildvcs=false ./cmd/bolsa-server) \
    >"$app_stdout" 2>"$app_stderr" &
  app_pid="$!"
  for _ in $(seq 1 80); do
    if [[ "$app_addr" == "127.0.0.1:0" ]]; then
      echo "skip_bolsa_app_start=ephemeral_addr_not_probeable"
      return 0
    fi
    if curl -fsS -m 2 "http://$app_addr/healthz" >/dev/null; then
      echo "bolsa_app_url=http://$app_addr/"
      curl -fsS -m 5 "http://$app_addr/api/portal" \
        -H "X-Auth-Mechanism: kerberos_ad" \
        -H "X-Auth-Subject: staff" \
        -H "Authorization: Bearer staff-token" \
        >"$result_dir/bolsa_api_portal.json"
      echo "bolsa_app_health_ok=true"
      return 0
    fi
    sleep 0.25
  done
  echo "bolsa app no arranco" >&2
  tail -n 80 "$app_stderr" >&2 || true
  return 1
}

validate_bolsa_ui_contract_if_present() {
  local index_file="$project_dir/web/static/index.html"
  local app_file="$project_dir/web/static/app.js"
  if [[ ! -f "$index_file" || ! -f "$app_file" ]]; then
    echo "skip_bolsa_ui_contract=no_static_web"
    return 0
  fi
  python3 - "$index_file" "$app_file" <<'PY'
import re
import sys

index_path, app_path = sys.argv[1:3]
index = open(index_path, encoding="utf-8").read()
app = open(app_path, encoding="utf-8").read()

visible_buttons = len(re.findall(r"<button\b", index))
requirements = {
    "module_navigation": [".module-link", "setActiveModule", "addEventListener"],
    "search_form": [".search-form", "submit", "state.search"],
    "filter_form": [".filter-bar", "state.filters", "filteredRows"],
    "tabs": [".tabs [role='tab']", "setActiveTab", "aria-selected"],
    "table_row_actions": ["handleRowAction", "stopPropagation", "tr.addEventListener"],
    "notifications": [".operator-tools .quiet-action", "notificaciones"],
    "export": ["exportRows", "Blob", "download"],
}
missing = []
if visible_buttons == 0:
    missing.append("visible_buttons")
for name, needles in requirements.items():
    absent = [needle for needle in needles if needle not in app]
    if absent:
        missing.append(f"{name}:{','.join(absent)}")
if missing:
    raise SystemExit("bolsa_ui_contract_failed=" + ";".join(missing))
print(f"bolsa_ui_contract_ok=true buttons={visible_buttons} requirements={len(requirements)}")
PY
}

main() {
  local confirm="${ORQUESTA_AUTOPROGRAMMING_BOLSA_REAL_CONFIRM:-${ORQUESTA_BOLSA_REAL_SMOKE_CONFIRM:-0}}"
  local codex_confirm="${ORQUESTA_AUTOPROGRAMMING_BOLSA_REAL_CODEX_CONFIRMED:-${ORQUESTA_BOLSA_REAL_CODEX_EXECUTION_CONFIRMED:-0}}"
  if [[ "$confirm" != "1" || "$codex_confirm" != "1" ]]; then
    echo "smoke real desactivado: exporta ORQUESTA_BOLSA_REAL_SMOKE_CONFIRM=1 y ORQUESTA_BOLSA_REAL_CODEX_EXECUTION_CONFIRMED=1" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BASE_URL:-}" || -n "${OPES_BASE_URL:-}" ]]; then
    echo "OPES debe estar desactivado para este smoke" >&2
    exit 2
  fi

  need_cmd curl
  need_cmd go
  need_cmd python3
  codex_command="$(resolve_command "${ORQUESTA_CODEX_COMMAND:-codex}")" || {
    echo "falta ORQUESTA_CODEX_COMMAND/codex ejecutable" >&2
    exit 2
  }

  mkdir -p "$state_dir" "$runtime_dir" "$bin_dir" "$payload_dir" "$result_dir"
  prepare_project_dir

  local source_spec="${ORQUESTA_BOLSA_SPEC_PATH:-${ORQUESTA_BOLSA_SPEC_JSON:-$project_dir/orquesta_spec_nucleo.json}}"
  if [[ ! -f "$source_spec" ]]; then
    echo "falta spec Bolsa: ajusta ORQUESTA_BOLSA_SPEC_PATH o copia orquesta_spec_nucleo.json" >&2
    exit 2
  fi
  local prepare_payload="$payload_dir/prepare_run.json"
  local prepare_response="$result_dir/prepare_run.json"
  write_prepare_payload "$prepare_payload" "$source_spec"

  echo "compilando orquesta-server..."
  (cd "$repo_root" && go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server)
  start_orquesta_server "$codex_command"

  post_json_required "/api/v0/autoprogramming/prepare-run" "$prepare_payload" "$prepare_response"
  run_ref="$(extract_prepare_run_ref "$prepare_response")"
  echo "bolsa_run_ref=$run_ref"
  wait_run_closed "$run_ref"

  echo "validando app generada/copiada..."
  (cd "$project_dir" && go test -count=1 ./...)
  validate_bolsa_ui_contract_if_present
  start_bolsa_app_if_present

  echo "codex_real_executed=true"
  echo "opes_touched=false"
  echo "summary_dir=$result_dir"
}

main "$@"
