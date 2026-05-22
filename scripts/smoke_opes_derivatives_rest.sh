#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DEFAULT_SEQUENCE="draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic"
MODE="${ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE:-dry-run-once}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/derivatives-rest-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
ORQUESTA_BASE_URL_EFFECTIVE="${ORQUESTA_BASE_URL:-}"
SEQUENCE="${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-$DEFAULT_SEQUENCE}"
LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"
FAKE_SERVER="${ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER:-0}"
FAKE_PENDING_TYPE="${ORQUESTA_OPES_DERIVATIVES_FAKE_PENDING_TYPE:-assemble_topic}"
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
  if [[ -z "$FAKE_PENDING_TYPE" ]]; then
    echo "ORQUESTA_OPES_DERIVATIVES_FAKE_PENDING_TYPE no puede estar vacio" >&2
    exit 2
  fi
  FAKE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/opes-derivatives-fake.XXXXXX")"
  local server_py="$FAKE_DIR/fake_opes.py"
  local url_file="$FAKE_DIR/url.txt"
  cat >"$server_py" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse

url_file = sys.argv[1]
sequence = [item.strip() for item in sys.argv[2].split(",") if item.strip()]
limit = sys.argv[3]
pending_type = sys.argv[4]
seen_types = []

payload_by_type = {
    "draft_content_block": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "section_ref": "section-ref-fake-001",
        "title": "Bloque fake Operario",
    },
    "generate_visual_asset": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "visual_ref": "visual-ref-fake-001",
        "objective": "Diagrama fake Operario",
    },
    "assemble_topic": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "document_plan_artifact_id": "artifact-plan-fake-001",
        "content_block_artifact_refs": ["artifact-block-fake-001"],
        "visual_asset_artifact_refs": ["artifact-visual-fake-001"],
        "review_artifact_refs": ["artifact-review-fake-001"],
        "validation_artifact_ref": "artifact-validation-fake-001",
    },
}

def payload_for(job_type):
    return payload_by_type.get(job_type, {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "document_plan_artifact_id": "artifact-plan-fake-001",
        "scope": "tema completo fake",
    })

def send_json(handler, status, body):
    raw = json.dumps(body).encode("utf-8")
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(raw)))
    handler.end_headers()
    handler.wfile.write(raw)

class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def do_GET(self):
        parsed = urlparse(self.path)
        query = parse_qs(parsed.query)
        if parsed.path != "/api/jobs":
            send_json(self, 404, {"error": "unexpected_path", "path": parsed.path})
            return
        job_type = query.get("job_type", [""])[0]
        expected = sequence[len(seen_types)] if len(seen_types) < len(sequence) else ""
        if job_type != expected:
            send_json(self, 400, {"error": "unexpected_job_type", "got": job_type, "want": expected, "seen": seen_types})
            return
        if query.get("status", [""])[0] != "pending":
            send_json(self, 400, {"error": "missing_status_filter", "query": query})
            return
        if query.get("execution_mode", [""])[0] != "external":
            send_json(self, 400, {"error": "missing_execution_mode_filter", "query": query})
            return
        if query.get("limit", [""])[0] != limit:
            send_json(self, 400, {"error": "unexpected_limit", "got": query.get("limit", [""])[0], "want": limit})
            return
        seen_types.append(job_type)
        if job_type != pending_type:
            send_json(self, 200, [])
            return
        send_json(self, 200, [{
                "id": "job-ref-fake-" + job_type.replace("_", "-") + "-001",
                "type": job_type,
                "status": "pending",
                "execution_mode": "external",
                "payload_json": json.dumps(payload_for(job_type)),
                "correlation_id": "corr-ref-fake-" + job_type.replace("_", "-") + "-001",
                "idempotency_key": "idem-ref-fake-" + job_type.replace("_", "-") + "-001",
                "requested_by": "opes-fake"
            }])

if pending_type not in sequence:
    raise SystemExit(f"pending type {pending_type!r} no esta en secuencia {sequence!r}")

server = HTTPServer(("127.0.0.1", 0), Handler)
with open(url_file, "w", encoding="utf-8") as fh:
    fh.write(f"http://127.0.0.1:{server.server_port}\n")
server.serve_forever()
PY
  python3 "$server_py" "$url_file" "$SEQUENCE" "$LIMIT" "$FAKE_PENDING_TYPE" &
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

require_temporal_opes() {
  if [[ "$FAKE_SERVER" == "1" ]]; then
    start_fake_opes
    return
  fi
  if [[ "${ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM:-0}" != "1" ]]; then
    echo "smoke derivados REST desactivado: exporta ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_TEMPORAL_CONFIRM:-0}" != "1" ]]; then
    echo "falta confirmacion de instancia OPES temporal: exporta ORQUESTA_OPES_TEMPORAL_CONFIRM=1" >&2
    exit 2
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
}

require_sequence_only() {
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_TYPE:-}" ]]; then
    echo "no mezclar ORQUESTA_OPES_BRIDGE_JOB_TYPE con JOB_TYPE_SEQUENCE en derivados" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_REF:-}" ]]; then
    echo "no mezclar ORQUESTA_OPES_BRIDGE_JOB_REF con JOB_TYPE_SEQUENCE en derivados" >&2
    exit 2
  fi
  if [[ -z "$SEQUENCE" ]]; then
    echo "ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE no puede estar vacio" >&2
    exit 2
  fi
}

write_metadata() {
  mkdir -p "$SMOKE_OUT_DIR"
  {
    echo "smoke_id=$SMOKE_ID"
    echo "mode=$MODE"
    echo "fake_server=$FAKE_SERVER"
    echo "fake_pending_type=$FAKE_PENDING_TYPE"
    echo "opes_base_url=$OPES_BASE_URL_EFFECTIVE"
    echo "orquesta_base_url=$ORQUESTA_BASE_URL_EFFECTIVE"
    echo "sequence=$SEQUENCE"
    echo "limit=$LIMIT"
    echo "output_dir=$SMOKE_OUT_DIR"
  } >"$SMOKE_OUT_DIR/metadata.txt"
}

run_dry_run_once() {
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=1
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE="$SEQUENCE"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_derivatives_rest_dry_run_summary.json"
}

run_execute_drain_once() {
  if [[ "${ORQUESTA_OPES_DERIVATIVES_EXECUTE:-0}" != "1" ]]; then
    echo "falta confirmacion de efectos: exporta ORQUESTA_OPES_DERIVATIVES_EXECUTE=1" >&2
    exit 2
  fi
  if [[ -z "$ORQUESTA_BASE_URL_EFFECTIVE" ]]; then
    echo "falta ORQUESTA_BASE_URL explicito para crear runs desde derivados" >&2
    exit 2
  fi
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_BASE_URL="$ORQUESTA_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_CONFIRM=1
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=0
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE="$SEQUENCE"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_derivatives_rest_drain_summary.json"
}

main() {
  require_tool go
  require_tool tee
  require_temporal_opes
  require_sequence_only
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
