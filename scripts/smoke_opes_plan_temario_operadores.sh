#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

MODE="${ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_MODE:-dry-run-once}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/plan-temario-operadores-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
ORQUESTA_BASE_URL_EFFECTIVE="${ORQUESTA_BASE_URL:-}"
JOB_REF="${ORQUESTA_OPES_BRIDGE_JOB_REF:-}"
LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"
FAKE_SERVER="${ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER:-0}"
FAKE_DIR=""
FAKE_PID=""

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta herramienta requerida: $1" >&2
    exit 2
  fi
}

is_local_url() {
  case "$1" in
    http://127.0.0.1|http://127.0.0.1:*|http://localhost|http://localhost:*|http://[::1]|http://[::1]:*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

cleanup() {
  if [[ -n "$FAKE_PID" ]]; then
    kill "$FAKE_PID" >/dev/null 2>&1 || true
    wait "$FAKE_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

start_fake_opes() {
  require_tool python3
  if [[ "$MODE" != "dry-run-once" ]]; then
    echo "fake OPES solo soporta dry-run-once" >&2
    exit 2
  fi
  JOB_REF="${JOB_REF:-job-ref-fake-plan-temario-operadores-001}"
  FAKE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/opes-plan-temario-fake.XXXXXX")"
  local server_py="$FAKE_DIR/fake_opes.py"
  local url_file="$FAKE_DIR/url.txt"
  cat >"$server_py" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse

url_file = sys.argv[1]
job_ref = sys.argv[2]

def job():
    return {
        "id": job_ref,
        "type": "plan_temario",
        "status": "pending",
        "execution_mode": "external",
        "payload_json": json.dumps({
            "program_id": "program-ref-fake-operadores-001",
            "document_kind": "temario_oposicion",
            "language_code": "es",
            "level": "AP",
            "title": "Temario Operario AP fake",
            "expected_artifact_type": "document_plan",
            "reasoning": "xhigh"
        }),
        "correlation_id": "corr-" + job_ref,
        "idempotency_key": "idem-" + job_ref,
        "requested_by": "opes-fake"
    }

class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path == "/api/jobs/" + job_ref:
            body = json.dumps(job()).encode("utf-8")
        elif parsed.path == "/api/jobs":
            body = json.dumps([job()]).encode("utf-8")
        else:
            self.send_response(404)
            self.end_headers()
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

server = HTTPServer(("127.0.0.1", 0), Handler)
with open(url_file, "w", encoding="utf-8") as fh:
    fh.write(f"http://127.0.0.1:{server.server_port}\n")
server.serve_forever()
PY
  python3 "$server_py" "$url_file" "$JOB_REF" &
  FAKE_PID="$!"
  for _ in $(seq 1 50); do
    if [[ -s "$url_file" ]]; then
      OPES_BASE_URL_EFFECTIVE="$(cat "$url_file")"
      return
    fi
    sleep 0.1
  done
  echo "fake OPES no arranco" >&2
  exit 1
}

require_plan_temario_guard() {
  if [[ "$FAKE_SERVER" == "1" ]]; then
    start_fake_opes
  else
    if [[ "${ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_CONFIRM:-0}" != "1" ]]; then
      echo "smoke plan_temario desactivado: exporta ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_CONFIRM=1" >&2
      exit 2
    fi
    if [[ "${ORQUESTA_OPES_TEMPORAL_CONFIRM:-0}" != "1" ]]; then
      echo "falta confirmacion de instancia OPES temporal: exporta ORQUESTA_OPES_TEMPORAL_CONFIRM=1" >&2
      exit 2
    fi
  fi
  if [[ -z "$OPES_BASE_URL_EFFECTIVE" ]]; then
    echo "falta ORQUESTA_OPES_BASE_URL u OPES_BASE_URL apuntando a OPES temporal" >&2
    exit 2
  fi
  if ! is_local_url "$OPES_BASE_URL_EFFECTIVE" &&
    [[ "${ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL:-0}" != "1" ]]; then
    echo "OPES_BASE_URL no parece local: $OPES_BASE_URL_EFFECTIVE" >&2
    echo "si es temporal no local, exporta ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL=1" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-}" ]]; then
    echo "plan_temario usa JOB_TYPE=plan_temario y JOB_REF exacto, no JOB_TYPE_SEQUENCE" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_TYPE:-}" && "${ORQUESTA_OPES_BRIDGE_JOB_TYPE}" != "plan_temario" ]]; then
    echo "ORQUESTA_OPES_BRIDGE_JOB_TYPE debe ser plan_temario" >&2
    exit 2
  fi
  if [[ -z "$JOB_REF" ]]; then
    echo "falta ORQUESTA_OPES_BRIDGE_JOB_REF exacto para plan_temario Operario" >&2
    exit 2
  fi
}

write_metadata() {
  mkdir -p "$SMOKE_OUT_DIR"
  {
    echo "smoke_id=$SMOKE_ID"
    echo "mode=$MODE"
    echo "fake_server=$FAKE_SERVER"
    echo "opes_base_url=$OPES_BASE_URL_EFFECTIVE"
    echo "orquesta_base_url=$ORQUESTA_BASE_URL_EFFECTIVE"
    echo "job_type=plan_temario"
    echo "job_ref=$JOB_REF"
    echo "limit=$LIMIT"
    echo "output_dir=$SMOKE_OUT_DIR"
  } >"$SMOKE_OUT_DIR/metadata.txt"
}

run_dry_run_once() {
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=1
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario
  export ORQUESTA_OPES_BRIDGE_JOB_REF="$JOB_REF"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_plan_temario_dry_run_summary.json"
}

run_execute_drain_once() {
  if [[ "${ORQUESTA_OPES_PLAN_TEMARIO_EXECUTE:-0}" != "1" ]]; then
    echo "falta confirmacion de efectos: exporta ORQUESTA_OPES_PLAN_TEMARIO_EXECUTE=1" >&2
    exit 2
  fi
  if [[ -z "$ORQUESTA_BASE_URL_EFFECTIVE" ]]; then
    echo "falta ORQUESTA_BASE_URL explicito para crear run plan_temario" >&2
    exit 2
  fi
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_BASE_URL="$ORQUESTA_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_CONFIRM=1
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=0
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario
  export ORQUESTA_OPES_BRIDGE_JOB_REF="$JOB_REF"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_plan_temario_drain_summary.json"
}

main() {
  require_tool go
  require_tool tee
  require_plan_temario_guard
  write_metadata
  cd "$repo_root"

  case "$MODE" in
    dry-run-once)
      run_dry_run_once
      ;;
    drain-once)
      run_execute_drain_once
      ;;
    *)
      echo "modo no soportado: $MODE (usa dry-run-once o drain-once)" >&2
      exit 2
      ;;
  esac
}

main "$@"
