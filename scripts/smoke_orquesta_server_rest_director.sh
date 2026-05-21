#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-rest-smoke.XXXXXX")"
state_dir="$work_root/state"
project_dir="$work_root/project"
runtime_dir="$project_dir/.orquesta-runtime"
bin_dir="$work_root/bin"
payload_file="$work_root/app_spec_request.json"
director_response="$work_root/director_response.json"
stats_payload="$work_root/director_stats_request.json"
server_stdout="$work_root/server.stdout.log"
server_stderr="$work_root/server.stderr.log"
server_pid=""

polls="${ORQUESTA_SMOKE_STATS_POLLS:-3}"
sleep_seconds="${ORQUESTA_SMOKE_STATS_SLEEP_SECONDS:-2}"
request_timeout="${ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS:-45}"
stats_timeout="${ORQUESTA_SMOKE_STATS_TIMEOUT_SECONDS:-10}"
min_started_agents="${ORQUESTA_SMOKE_MIN_STARTED_AGENTS:-${ORQUESTA_SMOKE_MIN_PARALLEL_AGENTS:-1}}"
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
stats_verified="0"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

cleanup() {
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -INT "$server_pid" >/dev/null 2>&1 || true
    for _ in $(seq 1 25); do
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
  if [[ "$keep_dir" == "1" ]]; then
    echo "directorio conservado: $work_root" >&2
  else
    rm -rf "$work_root"
  fi
}

trap cleanup EXIT
trap 'exit 130' INT TERM

need_cmd curl
need_cmd go
need_cmd python3

mkdir -p "$state_dir" "$runtime_dir" "$bin_dir" "$project_dir"

request_id="request-ref-server-rest-smoke-$(date -u +%Y%m%dT%H%M%SZ)"
cat >"$payload_file" <<JSON
{
  "request_id": "$request_id",
  "correlation_id": "$request_id",
  "max_bursts": 4,
  "max_steps_per_burst": 4,
  "max_dispatches_per_wait": 2,
  "max_commands": 8,
  "max_outbox_per_cycle": 4,
  "max_external_waits": 1,
  "app_spec_request": {
    "schema_version": "app_spec_request.v0",
    "request_id": "$request_id",
    "source": "orquesta-cli",
    "locale": "es",
    "request_kind": "crear_app_completa",
    "execution_mode": "normal",
    "nombre": "inventario mini rest",
    "objetivo": "Crear una app pequena completa para gestionar articulos con API REST en Go, web HTML minima y persistencia por conector.",
    "descripcion": "La app generada debe exponer CRUD de articulos por REST, una vista web simple y persistencia detras de un puerto/repositorio. Puede usar SQLite como requisito local de la app generada, sin acoplar Orquesta a una DB concreta.",
    "tipo_app": "mixed",
    "usuarios_objetivo": ["operador interno"],
    "plataformas": ["web", "api"],
    "integraciones": [
      {
        "tipo": "persistencia",
        "nombre": "repositorio_persistente",
        "proposito": "Guardar articulos mediante un conector de persistencia inyectado.",
        "requerido": true,
        "restricciones": ["no acoplar dominio a proveedor concreto"]
      }
    ],
    "preferencias_tecnicas": {
      "lenguaje": "go",
      "framework": "net/http",
      "arquitectura": "hexagonal",
      "restricciones": [
        "API REST Go",
        "web HTML minima",
        "SQLite solo como requisito local de la app generada, detras de puerto de persistencia",
        "sin DB hardcodeada en Orquesta"
      ]
    },
    "datos": {
      "db_required": true,
      "necesidad_funcional": "Persistir articulos y recuperarlos entre reinicios de la app generada.",
      "tipos_datos": ["articulo", "categoria"],
      "sensibilidad": "baja",
      "retencion": "local"
    },
    "deploy": {
      "target": "local",
      "restricciones": ["ejecutable local reproducible"]
    },
    "calidad": {
      "pruebas": "media",
      "accesibilidad": "basica",
      "observabilidad": true
    },
    "i18n": {
      "enabled": true,
      "default_locale": "es",
      "locales": ["es"]
    },
    "agentes": {
      "autonomia": "baja",
      "preferencias": ["mantener cambios pequenos y verificables"]
    },
    "restricciones": [
      "hexagonal puro",
      "macroarchivos prohibidos",
      "entrega pequena y verificable"
    ]
  }
}
JSON

echo "compilando servidor temporal..."
go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server

export ORQUESTA_SERVER_ADDR="127.0.0.1:0"
export ORQUESTA_SERVER_STATE_DIR="$state_dir"
export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir"
export ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-1000}"
export ORQUESTA_CODEX_WAIT_INTERVAL_MS="${ORQUESTA_CODEX_WAIT_INTERVAL_MS:-1000}"

echo "arrancando orquesta-server..."
"$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
server_pid="$!"

state_file="$state_dir/orquesta_server_state_v0.json"
server_addr=""
server_ready="0"
for _ in $(seq 1 60); do
  if [[ -s "$state_file" ]]; then
    server_addr="$(python3 - "$state_file" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("addr", ""))
PY
)"
    if [[ -n "$server_addr" ]] && curl -fsS -m 2 "http://$server_addr/healthz" >/dev/null; then
      server_ready="1"
      break
    fi
  fi
  sleep 0.5
done

if [[ "$server_ready" != "1" ]]; then
  echo "el servidor no llego a health OK; addr=$server_addr; stderr:" >&2
  tail -n 80 "$server_stderr" >&2 || true
  exit 1
fi

base_url="http://$server_addr"
echo "servidor listo: $base_url"

director_status="$(
  curl -sS -m "$request_timeout" -o "$director_response" -w "%{http_code}" \
    -X POST "$base_url/api/v0/apps/director" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: $request_id" \
    --data-binary "@$payload_file"
)"

echo "POST /api/v0/apps/director -> HTTP $director_status"
if [[ "$director_status" -lt 200 || "$director_status" -gt 299 ]]; then
  cat "$director_response" >&2
  exit 1
fi

run_ref="$(python3 - "$director_response" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("run_ref", ""))
PY
)"

if [[ -z "$run_ref" ]]; then
  echo "respuesta sin run_ref:" >&2
  cat "$director_response" >&2
  exit 1
fi

echo "run_ref=$run_ref"

for i in $(seq 1 "$polls"); do
  cat >"$stats_payload" <<JSON
{
  "request_id": "$request_id-stats-$i",
  "correlation_id": "$request_id-stats-$i",
  "locale": "es",
  "run_ref": "$run_ref",
  "include_process_refs": true,
  "include_agent_progress": true,
  "include_agent_usage": true
}
JSON
  stats_response="$work_root/director_stats_$i.json"
  stats_status="$(
    curl -sS -m "$stats_timeout" -o "$stats_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/director/stats" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-stats-$i" \
      --data-binary "@$stats_payload"
  )"
  echo "POST /api/v0/director/stats poll=$i -> HTTP $stats_status"
  if [[ "$stats_status" -lt 200 || "$stats_status" -gt 299 ]]; then
    cat "$stats_response" >&2
    exit 1
  fi
  set +e
  python3 - "$stats_response" "$min_started_agents" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
min_started_agents = int(sys.argv[2])
stats = data.get("stats") or data.get("director_stats") or {}
counts = stats.get("counts") or {}
progress = stats.get("progress") or {}
agents = stats.get("agents") or []
usage = stats.get("usage_summary")
print("estado=%s run_ref=%s phase=%s tasks=%s agents_started=%s" % (
    data.get("estado", ""),
    data.get("run_ref") or stats.get("run_ref", ""),
    stats.get("current_phase", ""),
    counts.get("tasks_total", ""),
    counts.get("agents_started", ""),
))
print("control_registered=%s no_signal=%s progress_source=%s usage_agents=%s" % (
    counts.get("agents_control_registered", ""),
    len(progress.get("no_signal_agent_refs") or []),
    progress.get("source_status", ""),
    (usage or {}).get("agents_observed", ""),
))
issues = []
agents_started = int(counts.get("agents_started") or 0)
if agents_started < min_started_agents:
    issues.append("agents_started<%s" % min_started_agents)
if not isinstance(progress, dict) or not progress.get("source_status"):
    issues.append("progress_missing")
if not isinstance(usage, dict) or int(usage.get("agents_observed") or 0) < min_started_agents:
    issues.append("usage_missing")
started_agents = [agent for agent in agents if agent.get("started")]
if len(started_agents) < min_started_agents:
    issues.append("agent_details_missing")
if any(not isinstance(agent.get("process"), dict) or not agent["process"].get("process_ref") for agent in started_agents):
    issues.append("process_refs_missing")
if any(not isinstance(agent.get("usage"), dict) or not agent["usage"].get("quota_status") for agent in started_agents):
    issues.append("agent_usage_missing")
if issues:
    print("stats_no_verificadas=%s" % ",".join(issues))
    sys.exit(2)
print("stats_verificadas=process_refs,progress,usage,director_start")
PY
  stats_validation_status="$?"
  set -e
  if [[ "$stats_validation_status" == "0" ]]; then
    stats_verified="1"
  elif [[ "$stats_validation_status" != "2" ]]; then
    exit "$stats_validation_status"
  fi
  if [[ "$i" -lt "$polls" ]]; then
    sleep "$sleep_seconds"
  fi
done

if [[ "$stats_verified" != "1" ]]; then
  echo "stats sin verificacion completa: faltan agentes arrancados, process refs, progress o usage" >&2
  exit 1
fi

echo "smoke completado; parada limpia en cleanup"
