#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"
# shellcheck source=scripts/lib/go_tool.sh
source "$repo_root/scripts/lib/go_tool.sh"

smoke_require_confirm \
  SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL \
  1 \
  "smoke Claude server bloqueado: exporta SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 para usar proveedor real"

smoke_require_tools python3 curl timeout jq
orquesta_go_tool_ensure_path

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

smoke_root_source="generated"
if [[ -n "${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_ROOT:-}" ]]; then
  smoke_root_source="env:SMOKE_CLAUDE_GOAL_PROCESS_SERVER_ROOT"
  smoke_root="$SMOKE_CLAUDE_GOAL_PROCESS_SERVER_ROOT"
elif [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
  smoke_root="$ORQUESTA_SMOKE_ROOT"
else
  smoke_parent="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_PARENT:-${TMPDIR:-/tmp}}"
  mkdir -p "$smoke_parent"
  ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES="$smoke_parent${ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES:+:$ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES}"
  export ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES
  smoke_root="$(mktemp -d "$smoke_parent/orquesta-claude-process-server.XXXXXX")"
fi
smoke_temp_root_prepare "$smoke_root" "$smoke_root_source"

state_dir="$smoke_root/state"
project_dir="$smoke_root/project"
idle_project_dir="$smoke_root/orquesta-idle"
runtime_dir="$smoke_root/runtime"
claude_runtime_dir="$runtime_dir/claude-goal"
bin_dir="$smoke_root/bin"
payload_file="$smoke_root/start_request.json"
start_response="$smoke_root/start_response.json"
observe_payload="$smoke_root/observe_request.json"
observe_response="$smoke_root/observe_response.json"
shutdown_response="$smoke_root/shutdown_response.json"
server_stdout="$smoke_root/server.stdout.log"
server_stderr="$smoke_root/server.stderr.log"
preflight_stdout="$smoke_root/claude-preflight.stdout.log"
preflight_stderr="$smoke_root/claude-preflight.stderr.log"
server_pid=""
base_url=""
run_ref=""
goal_ref=""
external_goal_ref=""

keep_dir="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_KEEP_DIR:-${ORQUESTA_KEEP_SMOKE_DIR:-0}}"
request_timeout="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REQUEST_TIMEOUT_SECONDS:-120}"
polls="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_POLLS:-90}"
sleep_seconds="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SLEEP_SECONDS:-5}"
budget="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MAX_BUDGET_USD:-0.60}"
preflight_budget="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_PREFLIGHT_BUDGET_USD:-0.20}"
model="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MODEL:-${ORQUESTA_CLAUDE_MODEL:-sonnet}}"
safe_mode="${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE:-1}"
if [[ "$safe_mode" != "0" && "$safe_mode" != "1" ]]; then
  echo "SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE debe ser 0 o 1" >&2
  exit 2
fi

stop_claude_processes_from_runtime_manifest() {
  if [[ ! -d "$claude_runtime_dir" ]]; then
    return 0
  fi
  python3 - "$claude_runtime_dir" <<'PY' || true
import json, os, signal, sys, time

runtime = os.path.realpath(sys.argv[1])
for name in os.listdir(runtime):
    if not (name.startswith("claude_goal_process_state_") and name.endswith(".json")):
        continue
    path = os.path.join(runtime, name)
    try:
        with open(path, encoding="utf-8") as fh:
            state = json.load(fh)
    except (OSError, json.JSONDecodeError):
        continue
    pid = state.get("pid")
    if not isinstance(pid, int) or pid <= 1:
        continue
    proc = f"/proc/{pid}"
    try:
        cmdline = open(os.path.join(proc, "cmdline"), "rb").read().replace(b"\0", b" ").decode("utf-8", "ignore")
    except OSError:
        continue
    if runtime not in cmdline or "claude_goal_wrapper_" not in cmdline:
        continue
    try:
        os.kill(pid, signal.SIGTERM)
    except ProcessLookupError:
        continue
    except OSError:
        continue
    for _ in range(20):
        try:
            os.kill(pid, 0)
        except ProcessLookupError:
            break
        time.sleep(0.1)
    else:
        try:
            os.kill(pid, signal.SIGKILL)
        except OSError:
            pass
PY
}

cleanup() {
  # El helper comun envia /api/v0/server/shutdown con cleanup_goal_backends=true.
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"
  stop_claude_processes_from_runtime_manifest
  smoke_temp_root_cleanup "$smoke_root" "$keep_dir"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

claude_command="$(resolve_command "${ORQUESTA_CLAUDE_COMMAND:-claude}")" || {
  echo "no se pudo resolver ORQUESTA_CLAUDE_COMMAND/claude" >&2
  exit 2
}
if [[ ! -x "$claude_command" ]]; then
  echo "ORQUESTA_CLAUDE_COMMAND no es ejecutable: $claude_command" >&2
  exit 2
fi

if [[ "${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SKIP_PREFLIGHT:-0}" != "1" ]]; then
  : >"$preflight_stdout"
  : >"$preflight_stderr"
  preflight_args=(
    -p
    --model "$model"
    --permission-mode "${ORQUESTA_CLAUDE_PERMISSION_MODE:-bypassPermissions}"
    --output-format "${ORQUESTA_CLAUDE_OUTPUT_FORMAT:-text}"
    --max-budget-usd "$preflight_budget"
  )
  if [[ "$safe_mode" == "1" ]]; then
    preflight_args+=(--safe-mode)
  fi
  if ! printf 'Responde solo: OK\n' |
    timeout "$request_timeout" "$claude_command" "${preflight_args[@]}" \
      >"$preflight_stdout" 2>"$preflight_stderr"; then
    echo "preflight Claude real fallido" >&2
    smoke_print_file_excerpt "$preflight_stderr"
    exit 2
  fi
  if [[ "$(tr -d '[:space:]' <"$preflight_stdout")" != "OK" ]]; then
    echo "preflight Claude real no devolvio OK:" >&2
    smoke_print_file_excerpt "$preflight_stdout"
    exit 2
  fi
fi

mkdir -p "$state_dir" "$project_dir" "$idle_project_dir" "$runtime_dir" "$claude_runtime_dir" "$bin_dir"

request_id="request-ref-claude-process-server-$(date -u +%Y%m%dT%H%M%SZ)"
cat >"$payload_file" <<JSON
{
  "request_id": "$request_id",
  "correlation_id": "$request_id",
  "requested_by": "smoke-goal-first-claude-process-server-real",
  "director_execution_mode": "goal_first",
  "app_spec_request": {
    "schema_version": "app_spec_request.v0",
    "request_id": "$request_id",
    "source": "orquesta-smoke",
    "locale": "es",
    "request_kind": "crear_app_completa",
    "execution_mode": "normal",
    "nombre": "Smoke Claude Process Server",
    "objetivo": "Crear una app local minima de smoke bajo generated-apps: un servicio Go net/http con README, test o verificacion local documentada y resultado durable orquesta_goal_result.v0 completo.",
    "descripcion": "Smoke real acotado de Orquesta server con backend goal-first claude_process. Debe crear una app pequena verificable, sin tocar rutas fuera del write-set y con cierre accepted por Orquesta.",
    "tipo_app": "mixed",
    "usuarios_objetivo": ["operador de smoke"],
    "plataformas": ["web", "api"],
    "preferencias_tecnicas": {
      "lenguaje": "go",
      "framework": "net/http",
      "arquitectura": "hexagonal",
      "restricciones": [
        "entrega minima",
        "sin dependencias externas obligatorias",
        "sin tocar rutas fuera de generated-apps",
        "resultado durable JSON puro desde el primer byte"
      ]
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
      "crear al menos README.md y codigo fuente Go bajo el write-set autorizado",
      "ejecutar o justificar una prueba local basica",
      "escribir orquesta_goal_result_v0.json con status complete, artifact_refs no vacio, artifact_paths no vacio, materialized_artifacts con artifact_ref y required_test_results passed",
      "no escribir markdown, fences ni texto alrededor del JSON durable"
    ]
  }
}
JSON

echo "compilando orquesta-server..."
(cd "$repo_root" && go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server)

export ORQUESTA_SERVER_ADDR="127.0.0.1:0"
export ORQUESTA_SERVER_STATE_DIR="$state_dir"
export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir/codex"
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR="$idle_project_dir"
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE=0
export ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=0
export ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false
export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED=false
export ORQUESTA_CODEX_GOAL_BACKEND=claude_process
export ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"
export ORQUESTA_CLAUDE_COMMAND="$claude_command"
export ORQUESTA_CLAUDE_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CLAUDE_RUNTIME_WORKDIR="$claude_runtime_dir"
export ORQUESTA_CLAUDE_HOME="${SMOKE_CLAUDE_HOME:-${ORQUESTA_CLAUDE_HOME:-${HOME:-}}}"
export ORQUESTA_CLAUDE_PATH="${ORQUESTA_CLAUDE_PATH:-$PATH}"
export ORQUESTA_CLAUDE_MODEL="$model"
export ORQUESTA_CLAUDE_PERMISSION_MODE="${ORQUESTA_CLAUDE_PERMISSION_MODE:-bypassPermissions}"
export ORQUESTA_CLAUDE_OUTPUT_FORMAT="${ORQUESTA_CLAUDE_OUTPUT_FORMAT:-text}"
claude_extra_args="--max-budget-usd $budget --no-session-persistence"
if [[ "$safe_mode" == "1" ]]; then
  claude_extra_args="$claude_extra_args --safe-mode"
fi
claude_extra_args="$claude_extra_args${ORQUESTA_CLAUDE_EXTRA_ARGS:+ $ORQUESTA_CLAUDE_EXTRA_ARGS}${SMOKE_CLAUDE_GOAL_PROCESS_SERVER_EXTRA_ARGS:+ $SMOKE_CLAUDE_GOAL_PROCESS_SERVER_EXTRA_ARGS}"
export ORQUESTA_CLAUDE_EXTRA_ARGS="$claude_extra_args"
export ORQUESTA_OPES_BASE_URL=""
export OPES_BASE_URL=""

echo "arrancando orquesta-server con claude_process..."
"$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
server_pid="$!"

state_file="$state_dir/orquesta_server_state_v0.json"
if ! smoke_wait_orquesta_readiness_from_state_file "$state_file" 80 0.5 base_url; then
  echo "orquesta-server temporal no llego a readiness OK; stderr:" >&2
  tail -n 80 "$server_stderr" >&2 || true
  exit 1
fi
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
  "requested_by": "smoke-goal-first-claude-process-server-real"
}
JSON
  observe_status="$(
    curl -sS -m "$request_timeout" -o "$observe_response" -w "%{http_code}" \
      -X POST "$base_url/api/v0/apps/director/goal/observe" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -H "X-Correlation-ID: $request_id-observe-$i" \
      --data-binary "@$observe_payload" || true
  )"
  if [[ "$observe_status" == "504" ]]; then
    echo "poll=$i observe_status=504 transient_timeout"
    sleep "$sleep_seconds"
    continue
  fi
  if [[ "$observe_status" -lt 200 || "$observe_status" -gt 299 ]]; then
    echo "observe HTTP $observe_status" >&2
    smoke_print_file_excerpt "$observe_response"
    exit 1
  fi
  goal_status="$(json_get "$observe_response" "goal_status")"
  run_status="$(json_get "$observe_response" "run_status")"
  director_execution_mode="$(json_get "$observe_response" "director_execution_mode")"
  closure_status="$(json_get "$observe_response" "closure_status")"
  closure_accepted="$(json_get "$observe_response" "closure_accepted")"
  echo "poll=$i mode=$director_execution_mode goal_status=$goal_status run_status=$run_status closure_status=$closure_status closure_accepted=$closure_accepted"
  if [[ "$director_execution_mode" == "goal_first" &&
    "$goal_status" == "complete" &&
    "$run_status" == "cerrada" &&
    "$closure_status" == "accepted" &&
    "$closure_accepted" == "true" ]]; then
    terminal="1"
    break
  fi
  if [[ "$goal_status" == "blocked" || "$closure_status" == "blocked" ]]; then
    echo "goal-first Claude process quedo bloqueado:" >&2
    smoke_print_file_excerpt "$observe_response"
    exit 1
  fi
  sleep "$sleep_seconds"
done

if [[ "$terminal" != "1" ]]; then
  echo "timeout esperando cierre accepted con claude_process server" >&2
  smoke_print_file_excerpt "$observe_response"
  exit 1
fi

result_file="$(find "$project_dir/generated-apps" -name 'orquesta_goal_result*.json' -type f -print -quit 2>/dev/null || true)"
if [[ -z "$result_file" ]]; then
  echo "no se encontro orquesta_goal_result*.json bajo generated-apps" >&2
  find "$project_dir" -maxdepth 4 -type f -print >&2 || true
  exit 1
fi
if [[ "$(json_get "$result_file" "status")" != "complete" ]]; then
  echo "result durable no esta complete: $result_file" >&2
  smoke_print_file_excerpt "$result_file"
  exit 1
fi

echo "smoke_goal_first_claude_process_server_real=ok"
echo "result_file=$result_file"
echo "smoke_root=$smoke_root"
