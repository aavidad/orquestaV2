#!/usr/bin/env bash
set -euo pipefail

STATUS_SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$STATUS_SCRIPT_ROOT/scripts/lib/smoke_common.sh"

usage() {
  cat <<'USAGE'
Uso:
  scripts/orquesta_status_now.sh [--json] [--watch SEGUNDOS]

Variables:
  ORQUESTA_SERVER_URL                  URL base del servidor gestionado.
  ORQUESTA_RUNTIME_DIR                 Runtime con base_url.txt si no hay URL explicita.
  ORQUESTA_STATUS_QUEUE_REF            Cola a consultar. Default: global
  ORQUESTA_STATUS_QUEUE_LIMIT          Numero de tareas de cola. Default: 20
  ORQUESTA_STATUS_TIMEOUT_SECONDS      Timeout HTTP por llamada. Default: 5

Salida:
  Resumen operativo inmediato desde la API publica del servidor y conteo local
  de procesos Codex cuando se ejecuta en la misma maquina.
USAGE
}

require_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    echo "Falta dependencia requerida: $name" >&2
    exit 2
  fi
}

trim_trailing_slash() {
  local value="$1"
  printf '%s' "${value%/}"
}

http_get_json() {
  local url="$1"
  local fallback="$2"
  local target="$3"
  if ! curl -fsS --max-time "$TIMEOUT_SECONDS" "$url" >"$target" 2>"$target.err"; then
    jq -n --arg error "$(cat "$target.err")" "$fallback" >"$target"
  fi
}

http_post_json() {
  local url="$1"
  local body="$2"
  local fallback="$3"
  local target="$4"
  if ! curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "$url" \
    -H 'content-type: application/json' \
    -d "$body" >"$target" 2>"$target.err"; then
    jq -n --arg error "$(cat "$target.err")" "$fallback" >"$target"
  fi
}

collect_local_processes_json() {
  local lines
  lines="$(ps -eo pid,ppid,stat,etime,pcpu,pmem,cmd | grep -E 'codex --ask-for-approval|orquesta-server-latest run' | grep -v grep || true)"
  local server_count codex_parent_count codex_vendor_count codex_total_count
  server_count="$(printf '%s\n' "$lines" | awk '/orquesta-server-latest run/ {count++} END {print count + 0}')"
  codex_parent_count="$(printf '%s\n' "$lines" | awk '/node .*\/codex --ask-for-approval/ {count++} END {print count + 0}')"
  codex_vendor_count="$(printf '%s\n' "$lines" | awk '/vendor\/.*\/codex --ask-for-approval/ {count++} END {print count + 0}')"
  codex_total_count="$(printf '%s\n' "$lines" | awk '/codex --ask-for-approval/ {count++} END {print count + 0}')"
  jq -n \
    --argjson server_count "$server_count" \
    --argjson codex_parent_count "$codex_parent_count" \
    --argjson codex_vendor_count "$codex_vendor_count" \
    --argjson codex_total_count "$codex_total_count" \
    --arg lines "$lines" \
    '{
      server_processes: $server_count,
      codex_parent_processes: $codex_parent_count,
      codex_vendor_processes: $codex_vendor_count,
      codex_total_processes: $codex_total_count,
      raw_process_lines: ($lines | split("\n") | map(select(length > 0)))
    }'
}

fetch_snapshot_json() {
  local tmpdir request_id server_file auto_file local_file body
  tmpdir="$(mktemp -d)"
  smoke_temp_root_prepare "$tmpdir" "generated"
  trap 'smoke_temp_root_cleanup "$tmpdir" 0' RETURN
  request_id="status-now-$(date -u +%Y%m%dT%H%M%SZ)"
  server_file="$tmpdir/server.json"
  auto_file="$tmpdir/autoprogramming.json"
  local_file="$tmpdir/local.json"

  body="$(jq -n \
    --arg request_id "$request_id" \
    --arg queue_ref "$QUEUE_REF" \
    --argjson queue_limit "$QUEUE_LIMIT" \
    '{
      request_id: $request_id,
      queue_ref: $queue_ref,
      queue_limit: $queue_limit,
      include_process_refs: true,
      include_agent_progress: true,
      include_agent_usage: true
    }')"

  http_get_json \
    "$SERVER_URL/api/v0/server/status" \
    '{status:"unreachable", error:$error}' \
    "$server_file"
  http_post_json \
    "$SERVER_URL/api/v0/autoprogramming/status" \
    "$body" \
    '{estado:"unreachable", errores_publicos:[{code:"http_error",field:"transport",message:$error}]}' \
    "$auto_file"
  collect_local_processes_json >"$local_file"

  jq -n \
    --arg checked_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg server_url "$SERVER_URL" \
    --slurpfile server "$server_file" \
    --slurpfile autoprogramming "$auto_file" \
    --slurpfile local_processes "$local_file" \
    '{
      checked_at: $checked_at,
      server_url: $server_url,
      server: $server[0],
      autoprogramming: $autoprogramming[0],
      local_processes: $local_processes[0]
    }'
}

print_human_summary() {
  jq -r '
    def text($v): if $v == null or $v == "" then "-" else ($v | tostring) end;
    def queue_items:
      (.autoprogramming.queue.ranked // [])[0:10][]
      | "  - #" + (.rank | tostring) + " " + text(.status) + " " + text(.run_ref);
    "Orquesta estado inmediato",
    "fecha_utc: " + text(.checked_at),
    "api: " + text(.server_url),
    "",
    "servidor: " + text(.server.status)
      + " pid=" + text(.server.pid)
      + " heartbeat=" + text(.server.last_heartbeat_at),
    "supervisor: " + text(.server.last_supervisor_status)
      + " ticks=" + text(.server.supervisor_ticks)
      + " ejecuciones=" + text(.server.supervisor_executions)
      + " cola=" + text(.server.last_supervisor_queue_size)
      + " idle=" + text(.server.idle_self_improvement_reason),
    "autoprogramacion: " + text(.autoprogramming.estado)
      + " tareas_cola=" + text(.autoprogramming.queue.count)
      + " activas=" + text((.autoprogramming.operator.active_runs // []) | length),
    "procesos_locales: servidor=" + text(.local_processes.server_processes)
      + " codex_padre=" + text(.local_processes.codex_parent_processes)
      + " codex_vendor=" + text(.local_processes.codex_vendor_processes),
    "",
    "cola_top:",
    (queue_items // "  - sin datos de cola")
  '
}

MODE="human"
WATCH_SECONDS=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --json)
      MODE="json"
      shift
      ;;
    --watch)
      WATCH_SECONDS="${2:-}"
      if [[ -z "$WATCH_SECONDS" || ! "$WATCH_SECONDS" =~ ^[0-9]+$ || "$WATCH_SECONDS" -lt 1 ]]; then
        echo "--watch requiere segundos enteros positivos" >&2
        exit 2
      fi
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Argumento no soportado: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

require_command curl
require_command jq

SERVER_URL="$(trim_trailing_slash "$(smoke_require_orquesta_base_url ORQUESTA_SERVER_URL)")"
QUEUE_REF="${ORQUESTA_STATUS_QUEUE_REF:-global}"
QUEUE_LIMIT="${ORQUESTA_STATUS_QUEUE_LIMIT:-20}"
TIMEOUT_SECONDS="${ORQUESTA_STATUS_TIMEOUT_SECONDS:-5}"

if [[ ! "$QUEUE_LIMIT" =~ ^[0-9]+$ || "$QUEUE_LIMIT" -lt 1 ]]; then
  echo "ORQUESTA_STATUS_QUEUE_LIMIT debe ser entero positivo" >&2
  exit 2
fi
if [[ ! "$TIMEOUT_SECONDS" =~ ^[0-9]+$ || "$TIMEOUT_SECONDS" -lt 1 ]]; then
  echo "ORQUESTA_STATUS_TIMEOUT_SECONDS debe ser entero positivo" >&2
  exit 2
fi

while true; do
  if [[ "$MODE" == "json" ]]; then
    fetch_snapshot_json
  else
    fetch_snapshot_json | print_human_summary
  fi
  if [[ -z "$WATCH_SECONDS" ]]; then
    break
  fi
  sleep "$WATCH_SECONDS"
done
