#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

smoke_require_confirm \
  ORQUESTA_OPES_REVIEW_PROVIDER_SMOKE_CONFIRM \
  1 \
  "smoke real desactivado: exporta ORQUESTA_OPES_REVIEW_PROVIDER_SMOKE_CONFIRM=1"

smoke_require_confirm \
  ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP \
  1 \
  "smoke legacy external-work desactivado: exporta ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1 para probar revisiones por loop historico"

PROJECT_DIR="${ORQUESTA_OPES_REVIEW_PROJECT_DIR:-}"
if [[ -z "$PROJECT_DIR" || ! -d "$PROJECT_DIR" ]]; then
  echo "define ORQUESTA_OPES_REVIEW_PROJECT_DIR con el paquete OPES local a revisar" >&2
  exit 2
fi

SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_ROOT_SOURCE="generated"
if [[ -n "${SMOKE_ROOT:-}" ]]; then
  SMOKE_ROOT_SOURCE="env:SMOKE_ROOT"
fi
SMOKE_ROOT="${SMOKE_ROOT:-/tmp/orquesta-opes-reviews-providers/$SMOKE_ID}"
STATE_DIR="$SMOKE_ROOT/state"
RUNTIME_DIR="$SMOKE_ROOT/runtime"
OUT_DIR="$SMOKE_ROOT/out"
BIN_DIR="$SMOKE_ROOT/bin"
EXTERNAL_DIR="$SMOKE_ROOT/external-domain"
PROVIDERS="${ORQUESTA_OPES_REVIEW_PROVIDERS:-codex,gemini,claude}"
TIMEOUT_SECONDS="${ORQUESTA_OPES_REVIEW_TIMEOUT_SECONDS:-1200}"
POLL_SECONDS="${ORQUESTA_OPES_REVIEW_POLL_SECONDS:-8}"
KEEP_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-1}"
REVIEW_OBJECTIVE="${ORQUESTA_OPES_REVIEW_OBJECTIVE:-Revisar si el paquete OPES esta listo como temario completo: contenido, tests, audios Microsoft, HTML, tutor, fuentes, trazabilidad y cierre.}"
REVIEW_ACCEPTANCE_NOTE="${ORQUESTA_OPES_REVIEW_ACCEPTANCE_NOTE:-emitir veredicto defendible; si es apto para produccion, declarar ACEPTADO_PRODUCCION con evidencia; si no, rework causal concreto}"
REVIEW_FILE_SUFFIX="${ORQUESTA_OPES_REVIEW_FILE_SUFFIX:-orquesta_real}"

server_pid=""
external_pid=""
base_url=""
external_addr=""

provider_enabled() {
  local provider="$1"
  case ",$PROVIDERS," in
    *",$provider,"*) return 0 ;;
    *) return 1 ;;
  esac
}

cleanup() {
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 40 "$RUNTIME_DIR"
  if [[ -n "$external_pid" ]] && kill -0 "$external_pid" >/dev/null 2>&1; then
    kill -INT "$external_pid" >/dev/null 2>&1 || true
    wait "$external_pid" >/dev/null 2>&1 || true
  fi
  smoke_temp_root_cleanup "$SMOKE_ROOT" "$KEEP_DIR"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

require_provider_commands() {
  smoke_require_tools curl go jq python3
  if provider_enabled codex; then
    smoke_require_tool "${ORQUESTA_CODEX_COMMAND:-codex}"
  fi
  if provider_enabled gemini; then
    smoke_require_tool "${ORQUESTA_GEMINI_COMMAND:-gemini}"
  fi
  if provider_enabled claude; then
    smoke_require_tool "${ORQUESTA_CLAUDE_COMMAND:-claude}"
  fi
}

start_external_backend() {
  mkdir -p "$EXTERNAL_DIR" "$OUT_DIR"
  python3 - "$EXTERNAL_DIR/state.json" "$EXTERNAL_DIR/addr.txt" \
    >"$OUT_DIR/external-domain.stdout.log" \
    2>"$OUT_DIR/external-domain.stderr.log" <<'PY' &
import json
import pathlib
import signal
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

state_path = pathlib.Path(sys.argv[1])
addr_path = pathlib.Path(sys.argv[2])
state = {"jobs": [], "submissions": []}


def persist():
    tmp = state_path.with_suffix(".tmp")
    tmp.write_text(json.dumps(state, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
    tmp.replace(state_path)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        return

    def _json(self, code, payload):
        data = json.dumps(payload, ensure_ascii=True).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/healthz":
            self._json(200, {"status": "ok"})
            return
        if self.path == "/state":
            self._json(200, state)
            return
        self._json(404, {"error": "not_found"})

    def do_POST(self):
        size = int(self.headers.get("Content-Length", "0") or "0")
        raw = self.rfile.read(size)
        try:
            payload = json.loads(raw.decode("utf-8") or "{}")
        except json.JSONDecodeError:
            self._json(400, {"error": "invalid_json"})
            return
        if self.path == "/api/domain-work/jobs":
            job_ref = payload.get("job_ref") or "job-http-" + str(len(state["jobs"]) + 1)
            job = {
                "schema_version": "domain_work_job.v0",
                "status": "accepted",
                "job_ref": job_ref,
                "domain_ref": payload.get("domain_ref", "opes"),
                "work_kind": payload.get("work_kind", ""),
                "correlation_id": payload.get("correlation_id", ""),
                "idempotency_key": payload.get("idempotency_key", ""),
                "external_refs": payload.get("external_refs") or [],
                "evidence_refs": ["domain-work-http-temporal-job"],
            }
            state["jobs"].append({"request": payload, "job": job})
            persist()
            self._json(200, job)
            return
        if self.path == "/api/domain-work/artifacts":
            artifact_ref = payload.get("artifact_ref") or "artifact-http-" + str(len(state["submissions"]) + 1)
            receipt = {
                "schema_version": "domain_work_artifact_receipt.v0",
                "status": "accepted",
                "job_ref": payload.get("job_ref", ""),
                "artifact_ref": artifact_ref,
                "receipt_ref": "receipt-" + artifact_ref,
                "correlation_id": payload.get("correlation_id", ""),
                "idempotency_key": payload.get("idempotency_key", ""),
                "external_refs": payload.get("external_refs") or [],
                "evidence_refs": ["domain-work-http-temporal-receipt"],
            }
            state["submissions"].append({"submission": payload, "receipt": receipt})
            persist()
            self._json(200, receipt)
            return
        self._json(404, {"error": "not_found"})


httpd = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
addr_path.write_text(f"{httpd.server_address[0]}:{httpd.server_address[1]}", encoding="utf-8")
signal.signal(signal.SIGINT, lambda *_: httpd.shutdown())
signal.signal(signal.SIGTERM, lambda *_: httpd.shutdown())
persist()
httpd.serve_forever()
PY
  external_pid="$!"
  for _ in $(seq 1 80); do
    if [[ -s "$EXTERNAL_DIR/addr.txt" ]]; then
      external_addr="$(cat "$EXTERNAL_DIR/addr.txt")"
      if curl -fsS -m 2 "http://$external_addr/healthz" >/dev/null; then
        return
      fi
    fi
    sleep 0.25
  done
  echo "backend HTTP temporal no llego a readiness" >&2
  exit 1
}

start_orquesta_server() {
  mkdir -p "$STATE_DIR" "$RUNTIME_DIR" "$BIN_DIR" "$OUT_DIR"
  echo "compilando orquesta-server..."
  (cd "$repo_root" && go build -o "$BIN_DIR/orquesta-server" ./cmd/orquesta-server)

  ORQUESTA_SERVER_ADDR="127.0.0.1:0" \
  ORQUESTA_SERVER_STATE_DIR="$STATE_DIR" \
  ORQUESTA_CODEX_PROJECT_WORKDIR="$PROJECT_DIR" \
  ORQUESTA_CODEX_RUNTIME_WORKDIR="$RUNTIME_DIR" \
  ORQUESTA_CODEX_COMMAND="${ORQUESTA_CODEX_COMMAND:-$(command -v codex 2>/dev/null || true)}" \
  ORQUESTA_CODEX_REASONING_EFFORT="${ORQUESTA_CODEX_REASONING_EFFORT:-medium}" \
  ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}" \
  ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-danger-full-access}" \
  ORQUESTA_CODEX_WAIT_INTERVAL_MS="${ORQUESTA_CODEX_WAIT_INTERVAL_MS:-2000}" \
  ORQUESTA_CODEX_NO_ACTIVITY_SECONDS="${ORQUESTA_CODEX_NO_ACTIVITY_SECONDS:-900}" \
  ORQUESTA_CODEX_MAX_EXPECTED_SECONDS="${ORQUESTA_CODEX_MAX_EXPECTED_SECONDS:-1800}" \
  ORQUESTA_CODEX_MAX_BATCH_READY="${ORQUESTA_CODEX_MAX_BATCH_READY:-3}" \
  ORQUESTA_CODEX_MAX_CONCURRENCY="${ORQUESTA_CODEX_MAX_CONCURRENCY:-3}" \
  ORQUESTA_GEMINI_ENABLED="${ORQUESTA_GEMINI_ENABLED:-1}" \
  ORQUESTA_GEMINI_PROJECT_WORKDIR="$PROJECT_DIR" \
  ORQUESTA_GEMINI_RUNTIME_WORKDIR="$RUNTIME_DIR" \
  ORQUESTA_GEMINI_COMMAND="${ORQUESTA_GEMINI_COMMAND:-$(command -v gemini 2>/dev/null || true)}" \
  ORQUESTA_GEMINI_APPROVAL_MODE="${ORQUESTA_GEMINI_APPROVAL_MODE:-auto_edit}" \
  ORQUESTA_GEMINI_OUTPUT_FORMAT="${ORQUESTA_GEMINI_OUTPUT_FORMAT:-text}" \
  ORQUESTA_CLAUDE_ENABLED="${ORQUESTA_CLAUDE_ENABLED:-1}" \
  ORQUESTA_CLAUDE_PROJECT_WORKDIR="$PROJECT_DIR" \
  ORQUESTA_CLAUDE_RUNTIME_WORKDIR="$RUNTIME_DIR" \
  ORQUESTA_CLAUDE_COMMAND="${ORQUESTA_CLAUDE_COMMAND:-$(command -v claude 2>/dev/null || true)}" \
  ORQUESTA_CLAUDE_PERMISSION_MODE="${ORQUESTA_CLAUDE_PERMISSION_MODE:-bypassPermissions}" \
  ORQUESTA_CLAUDE_OUTPUT_FORMAT="${ORQUESTA_CLAUDE_OUTPUT_FORMAT:-text}" \
  ORQUESTA_DOMAIN_WORK_FILE_ENABLED="" \
  ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL="http://$external_addr" \
  ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH="/api/domain-work/jobs" \
  ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH="/api/domain-work/artifacts" \
  ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE="smoke_local" \
  ORQUESTA_DOMAIN_WORK_HTTP_TIMEOUT_SECONDS="${ORQUESTA_DOMAIN_WORK_HTTP_TIMEOUT_SECONDS:-120}" \
  ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-60000}" \
  ORQUESTA_SERVER_MAX_RUNS_PER_TICK="${ORQUESTA_SERVER_MAX_RUNS_PER_TICK:-3}" \
  ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK="${ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK:-6}" \
  ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 \
  ORQUESTA_STARTUP_CLEANUP_MODE="off" \
    "$BIN_DIR/orquesta-server" run \
    >"$OUT_DIR/orquesta-server.stdout.log" \
    2>"$OUT_DIR/orquesta-server.stderr.log" &
  server_pid="$!"

  if smoke_wait_orquesta_readiness_from_state_file "$STATE_DIR/orquesta_server_state_v0.json" 120 0.5 base_url; then
    echo "orquesta-server listo: $base_url"
    return
  fi
  echo "orquesta-server no llego a readiness OK" >&2
  tail -n 80 "$OUT_DIR/orquesta-server.stderr.log" >&2 || true
  exit 1
}

utc_now() {
  python3 - <<'PY'
from datetime import datetime, timezone
print(datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"))
PY
}

write_external_work_payload() {
  local provider="$1"
  local output="$2"
  local run_ref="run-opes-review-${provider}-${SMOKE_ID}"
  local job_ref="job-opes-review-${provider}-${SMOKE_ID}"
  local work_kind="review_${provider}"
  local file_ref="reviews/review_${provider}_${REVIEW_FILE_SUFFIX}.md"
  REVIEW_OBJECTIVE="$REVIEW_OBJECTIVE" REVIEW_ACCEPTANCE_NOTE="$REVIEW_ACCEPTANCE_NOTE" python3 - "$output" "$SMOKE_ID" "$provider" "$run_ref" "$job_ref" "$work_kind" "$file_ref" "$(utc_now)" <<'PY'
import json
import os
import sys

output, smoke_id, provider, run_ref, job_ref, work_kind, file_ref, occurred_at = sys.argv[1:9]
review_objective = os.environ["REVIEW_OBJECTIVE"]
review_acceptance_note = os.environ["REVIEW_ACCEPTANCE_NOTE"]
payload = {
    "request_id": f"req-opes-review-{provider}-{smoke_id}",
    "correlation_id": f"corr-opes-review-{smoke_id}",
    "director_execution_mode": "legacy_director_loop",
    "external_work_run_request": {
        "schema_version": "external_work_run_request.v0",
        "request_id": f"req-opes-review-run-{provider}-{smoke_id}",
        "correlation_id": f"corr-opes-review-{smoke_id}",
        "run_ref": run_ref,
        "project_ref": "opes",
        "app_spec_ref": "app-spec-opes-review-providers",
        "queue_ref": "global",
        "priority_score": 95,
        "occurred_at": occurred_at,
        "requested_by": "orquesta-smoke-real-review",
        "app_change_request": {
            "schema_version": "app_change_request.v0",
            "request_id": f"req-opes-review-change-{provider}-{smoke_id}",
            "correlation_id": f"corr-opes-review-{smoke_id}",
            "run_ref": run_ref,
            "app_ref": "opes",
            "change_ref": f"change-opes-review-{provider}-{smoke_id}",
            "actor_ref": "orquesta-smoke-real-review",
            "locale": "es-ES",
            "user_intent": f"Ejecutar revision real {provider} del paquete OPES y devolver agent_review_report.",
            "target_area": "domain_work",
            "current_state_refs": ["opes-review-package", job_ref],
            "allowed_write_set": [file_ref],
            "acceptance_criteria": [
                f"crear {file_ref}",
                "informe con veredicto, hallazgos, riesgos, rework causal y material recuperable",
                review_acceptance_note,
                "declarar el fichero real en ACK.files",
                "no bloquear por heuristicas blandas ni palabras sueltas"
            ],
            "constraints": [
                "trabajar solo sobre el paquete local como workdir del proyecto",
                "no llamar REST ni APIs de OPES desde el agente",
                "no borrar ni reescribir artefactos existentes fuera del write-set"
            ],
            "external_work": {
                "project_ref": "opes",
                "job_ref": job_ref,
                "interface_refs": ["domain-work-http-temporal.v0", "opes-review-contract-v0"],
                "work_kind": work_kind,
                "work_refs": ["opes-review-package"],
                "input_fields": [
                    {"name": "package_ref", "value": "opes-review-package"},
                    {"name": "expected_artifact_type", "value": "agent_review_report"},
                    {"name": "output_file", "value": file_ref},
                    {"name": "manifest_ref", "value": "course_manifest.json"},
                    {"name": "source_refs", "values": [
                        "course_manifest.json",
                        "html_final/manifest.json",
                        "question_bank/tema_01.questions.json",
                        "question_bank/tema_20.questions.json",
                        "audio/generation_summary.json",
                        "audio/manifests/html_final/tema_01.html.json",
                        "html_final/index.html",
                        "html_final/tema_01.html",
                        "visuals_manifest.json",
                        "tutor/tutor_manifest.json",
                        "manual_ayuda/manual.md"
                    ]},
                    {"name": "objective", "value": review_objective},
                    {"name": "review_provider", "value": provider}
                ]
            }
        }
    }
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_supervisor_payload() {
  local provider="$1"
  local run_ref="$2"
  local attempt="$3"
  local output="$4"
  python3 - "$output" "$SMOKE_ID" "$provider" "$run_ref" "$attempt" "$(utc_now)" <<'PY'
import json
import sys

output, smoke_id, provider, run_ref, attempt, occurred_at = sys.argv[1:7]
payload = {
    "request_id": f"req-supervise-opes-review-{provider}-{attempt}-{smoke_id}",
    "correlation_id": f"corr-opes-review-{smoke_id}",
    "director_execution_mode": "legacy_director_loop",
    "run_ref": run_ref,
    "queue_ref": "global",
    "continue_message": "sigue",
    "occurred_at": occurred_at,
    "max_ticks": 1,
    "max_runs_per_tick": 1,
    "max_executions": 6,
    "allow_repeated_runs": True,
    "max_bursts": 16,
    "max_steps_per_burst": 8,
    "max_dispatches_per_wait": 8,
    "max_commands": 32,
    "max_outbox_per_cycle": 8,
    "max_decision_cycles": 4,
    "max_external_waits": 2
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

post_json_file() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  curl -sS -m 180 \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output" \
    -w "%{http_code}"
}

submission_count() {
  python3 - "$EXTERNAL_DIR/state.json" "$@" <<'PY'
import json
import pathlib
import sys

state_path = pathlib.Path(sys.argv[1])
want = set(sys.argv[2:])
try:
    data = json.loads(state_path.read_text(encoding="utf-8"))
except Exception:
    print(0)
    raise SystemExit(0)
seen = {item.get("submission", {}).get("job_ref") for item in data.get("submissions", [])}
print(len(want & seen))
PY
}

write_summary() {
  python3 - "$EXTERNAL_DIR/state.json" "$OUT_DIR/summary.json" "$@" <<'PY'
import json
import pathlib
import sys

state_path, output = sys.argv[1:3]
jobs = sys.argv[3:]
data = json.loads(pathlib.Path(state_path).read_text(encoding="utf-8"))
summary = []
for job in jobs:
    matches = [
        item for item in data.get("submissions", [])
        if item.get("submission", {}).get("job_ref") == job
    ]
    if not matches:
        summary.append({"job_ref": job, "missing": True})
        continue
    sub = matches[-1].get("submission", {})
    summary.append({
        "job_ref": job,
        "artifact_type": sub.get("artifact_type"),
        "artifact_ref": sub.get("artifact_ref"),
        "payload_fields": [field.get("name") for field in sub.get("payload_fields", [])],
    })
payload = {"submissions": summary}
pathlib.Path(output).write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(json.dumps(payload, ensure_ascii=False, indent=2))
if any(item.get("missing") for item in summary):
    raise SystemExit(3)
if any(item.get("artifact_type") != "agent_review_report" for item in summary):
    raise SystemExit(4)
PY
}

main() {
  require_provider_commands
  smoke_temp_root_prepare "$SMOKE_ROOT" "$SMOKE_ROOT_SOURCE"
  mkdir -p "$OUT_DIR" "$RUNTIME_DIR"
  start_external_backend
  start_orquesta_server

  local providers=()
  IFS=, read -r -a providers <<<"$PROVIDERS"
  local job_refs=()
  local provider run_ref job_ref payload response status

  for provider in "${providers[@]}"; do
    provider="$(printf '%s' "$provider" | xargs)"
    [[ -z "$provider" ]] && continue
    run_ref="run-opes-review-${provider}-${SMOKE_ID}"
    job_ref="job-opes-review-${provider}-${SMOKE_ID}"
    job_refs+=("$job_ref")
    payload="$OUT_DIR/external_work_${provider}.json"
    response="$OUT_DIR/external_work_${provider}_response.json"
    write_external_work_payload "$provider" "$payload"
    status="$(post_json_file "$base_url/api/v0/external-work/run" "$payload" "$response")"
    echo "external_work provider=$provider status=$status run=$run_ref job=$job_ref"
    if [[ "$status" != "200" ]]; then
      smoke_print_file_excerpt "$response"
      exit 1
    fi
  done

  local deadline=$((SECONDS + TIMEOUT_SECONDS))
  local attempt=0
  while (( SECONDS < deadline )); do
    attempt=$((attempt + 1))
    for provider in "${providers[@]}"; do
      provider="$(printf '%s' "$provider" | xargs)"
      [[ -z "$provider" ]] && continue
      run_ref="run-opes-review-${provider}-${SMOKE_ID}"
      payload="$OUT_DIR/supervise_${provider}_${attempt}.json"
      response="$OUT_DIR/supervise_${provider}_${attempt}_response.json"
      write_supervisor_payload "$provider" "$run_ref" "$attempt" "$payload"
      status="$(post_json_file "$base_url/api/v0/runs/supervise" "$payload" "$response" || true)"
      echo "supervise attempt=$attempt provider=$provider status=$status estado=$(jq -r '.estado // empty' "$response" 2>/dev/null || true)"
    done
    local done_count
    done_count="$(submission_count "${job_refs[@]}")"
    echo "submission_count=$done_count smoke_root=$SMOKE_ROOT"
    if [[ "$done_count" == "${#job_refs[@]}" ]]; then
      break
    fi
    sleep "$POLL_SECONDS"
  done

  write_summary "${job_refs[@]}"
  echo "SMOKE_ROOT=$SMOKE_ROOT"
  echo "PROJECT_DIR=$PROJECT_DIR"
  echo "RUNTIME_DIR=$RUNTIME_DIR"
  echo "SUMMARY=$OUT_DIR/summary.json"
  find "$RUNTIME_DIR" -name agent_ack.json -type f -print | sort
}

cd "$repo_root"
main "$@"
