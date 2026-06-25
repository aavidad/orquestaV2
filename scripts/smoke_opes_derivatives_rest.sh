#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

DEFAULT_SEQUENCE="research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site,generate_help_manual_assets,finalize_temario_package"
MODE="${ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE:-dry-run-once}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/derivatives-rest-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
ORQUESTA_BASE_URL_EFFECTIVE="${ORQUESTA_BASE_URL:-}"
SEQUENCE="${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-$DEFAULT_SEQUENCE}"
LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"
INPUT_LEDGER_PATH="${ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH:-$SMOKE_OUT_DIR/external-bridge-input-ledger.json}"
MAX_TICKS="${ORQUESTA_OPES_BRIDGE_MAX_TICKS:-20}"
TICK_SLEEP_SECONDS="${ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS:-5}"
FAKE_SERVER="${ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER:-0}"
FAKE_PENDING_TYPE="${ORQUESTA_OPES_DERIVATIVES_FAKE_PENDING_TYPE:-assemble_topic}"
FAKE_DIR=""
FAKE_PID=""

is_run_until_mode() {
  case "$1" in
    run-until-assemble | run-until-finalize | run-until-final)
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
  if [[ -n "$FAKE_DIR" && -d "$FAKE_DIR" ]]; then
    smoke_temp_root_cleanup "$FAKE_DIR" "${ORQUESTA_KEEP_SMOKE_DIR:-0}"
  fi
}
trap cleanup EXIT

start_fake_opes() {
  smoke_require_tool python3
  if [[ "$MODE" != "dry-run-once" ]] && ! is_run_until_mode "$MODE"; then
    echo "fake OPES solo soporta dry-run-once, run-until-finalize o run-until-assemble" >&2
    exit 2
  fi
  if [[ -z "$FAKE_PENDING_TYPE" ]]; then
    echo "ORQUESTA_OPES_DERIVATIVES_FAKE_PENDING_TYPE no puede estar vacio" >&2
    exit 2
  fi
  FAKE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/opes-derivatives-fake.XXXXXX")"
  smoke_temp_root_prepare "$FAKE_DIR" "generated"
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
mode = sys.argv[5]
active_index = sequence.index(pending_type) if mode == "dry-run-once" else 0
expected_scan_limit = limit
if mode != "dry-run-once":
    expected_scan_limit = str(min(max(int(limit), 1) * 10, 100))
runs = {}

payload_by_type = {
    "research_exam_precedents": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "scope": "examenes y temarios relacionados fake",
        "category": "operario",
    },
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
    "generate_question_bank": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "question_bank_ref": "question-bank-ref-fake-001",
        "minimum_questions": 50,
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
    "review_codex": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "review_scope": "curso completo fake",
        "review_role": "codex",
    },
    "review_gemini": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "review_scope": "curso completo fake",
        "review_role": "gemini",
    },
    "review_claude": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "review_scope": "curso completo fake",
        "review_role": "claude",
    },
    "review_pair_codex_gemini": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "pair_ref": "codex-gemini",
    },
    "review_pair_codex_claude": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "pair_ref": "codex-claude",
    },
    "review_pair_gemini_claude": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "pair_ref": "gemini-claude",
    },
    "review_director_consolidation": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "review_matrix_ref": "review-matrix-fake-001",
    },
    "generate_audio_asset": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "assembled_topic_artifact_id": "artifact-assembled-topic-fake-001",
        "audio_profile_ref": "audio-profile-accessible-es-001",
        "language_code": "es",
    },
    "generate_tutor_assets": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "assembled_topic_artifact_id": "artifact-assembled-topic-fake-001",
        "question_bank_artifact_id": "artifact-question-bank-fake-001",
    },
    "generate_learning_games": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "assembled_topic_artifact_id": "artifact-assembled-topic-fake-001",
        "question_bank_artifact_id": "artifact-question-bank-fake-001",
        "tutor_package_artifact_id": "artifact-tutor-fake-001",
    },
    "generate_html_site": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "assembled_topic_artifact_id": "artifact-assembled-topic-fake-001",
        "audio_manifest_artifact_id": "artifact-audio-fake-001",
        "tutor_package_artifact_id": "artifact-tutor-fake-001",
        "learning_games_package_artifact_id": "artifact-learning-games-fake-001",
    },
    "generate_help_manual_assets": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "local_html_site_artifact_id": "artifact-html-site-fake-001",
        "branding_rule_ref": "uso-branding-rule-fake-001",
        "help_manual_guide_ref": "screenshot-help-manuals-fake-001",
    },
    "finalize_temario_package": {
        "program_id": "program-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "local_html_site_artifact_id": "artifact-html-site-fake-001",
        "help_manual_package_artifact_id": "artifact-help-manual-fake-001",
        "learning_games_package_artifact_id": "artifact-learning-games-fake-001",
        "review_matrix_ref": "review-matrix-fake-001",
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
        global active_index
        parsed = urlparse(self.path)
        query = parse_qs(parsed.query)
        if parsed.path != "/api/jobs":
            send_json(self, 404, {"error": "unexpected_path", "path": parsed.path})
            return
        job_type = query.get("job_type", [""])[0]
        if job_type not in sequence:
            send_json(self, 400, {"error": "unexpected_job_type", "got": job_type, "sequence": sequence})
            return
        if query.get("status", [""])[0] != "pending":
            send_json(self, 400, {"error": "missing_status_filter", "query": query})
            return
        if query.get("execution_mode", [""])[0] != "external":
            send_json(self, 400, {"error": "missing_execution_mode_filter", "query": query})
            return
        if query.get("limit", [""])[0] != expected_scan_limit:
            send_json(self, 400, {"error": "unexpected_limit", "got": query.get("limit", [""])[0], "want": expected_scan_limit})
            return
        job_index = sequence.index(job_type)
        if active_index >= len(sequence) or job_index != active_index:
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

    def do_POST(self):
        global active_index
        parsed = urlparse(self.path)
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length)
        try:
            payload = json.loads(body.decode("utf-8") or "{}")
        except Exception as exc:
            send_json(self, 400, {"error": "invalid_json", "detail": str(exc)})
            return
        if parsed.path == "/api/v0/external-work/run":
            request = payload.get("external_work_run_request") or {}
            app_change = request.get("app_change_request") or {}
            external_work = app_change.get("external_work") or {}
            job_ref = external_work.get("job_ref") or ""
            work_kind = external_work.get("work_kind") or ""
            artifact_type = ""
            for field in external_work.get("input_fields") or []:
                if field.get("name") == "expected_artifact_type":
                    artifact_type = field.get("value") or ""
            if active_index >= len(sequence) or work_kind != sequence[active_index]:
                send_json(self, 400, {"error": "unexpected_external_work", "work_kind": work_kind, "active_index": active_index})
                return
            run_ref = "run-ref-fake-" + job_ref
            runs[run_ref] = {"job_ref": job_ref, "work_kind": work_kind, "artifact_type": artifact_type, "stage": active_index, "supervised": False}
            send_json(self, 200, {"run_ref": run_ref, "estado": "accepted"})
            return
        if parsed.path == "/api/v0/runs/supervise":
            run_ref = payload.get("run_ref") or ""
            run = runs.get(run_ref)
            if not run:
                send_json(self, 404, {"error": "run_not_found", "run_ref": run_ref})
                return
            if not run["supervised"]:
                run["supervised"] = True
                if run["stage"] == active_index:
                    active_index += 1
            send_json(self, 200, {"run_ref": run_ref, "status": "supervised", "work_kind": run["work_kind"], "artifact_type": run["artifact_type"], "next_stage_index": active_index})
            return
        send_json(self, 404, {"error": "unexpected_path", "path": parsed.path})

if pending_type not in sequence:
    raise SystemExit(f"pending type {pending_type!r} no esta en secuencia {sequence!r}")

server = HTTPServer(("127.0.0.1", 0), Handler)
with open(url_file, "w", encoding="utf-8") as fh:
    fh.write(f"http://127.0.0.1:{server.server_port}\n")
server.serve_forever()
PY
  python3 "$server_py" "$url_file" "$SEQUENCE" "$LIMIT" "$FAKE_PENDING_TYPE" "$MODE" &
  FAKE_PID="$!"
  for _ in $(seq 1 50); do
    if [[ -s "$url_file" ]]; then
      OPES_BASE_URL_EFFECTIVE="$(cat "$url_file")"
      if is_run_until_mode "$MODE"; then
        ORQUESTA_BASE_URL_EFFECTIVE="$OPES_BASE_URL_EFFECTIVE"
      fi
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
  if ! smoke_is_local_url "$OPES_BASE_URL_EFFECTIVE" &&
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
    echo "input_ledger_path=$INPUT_LEDGER_PATH"
    echo "max_ticks=$MAX_TICKS"
    echo "tick_sleep_seconds=$TICK_SLEEP_SECONDS"
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
  export ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH="$INPUT_LEDGER_PATH"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_derivatives_rest_drain_summary.json"
}

run_execute_drain_once_to() {
  local output_file="$1"
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
  export ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH="$INPUT_LEDGER_PATH"
  go run ./cmd/orquesta-server opes-drain-once | tee "$output_file"
}

json_summary_field() {
  local file="$1"
  local field="$2"
  python3 - "$file" "$field" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
value = data.get(sys.argv[2], "")
if isinstance(value, list):
    print(",".join(str(item) for item in value))
else:
    print(value)
PY
}

json_summary_run_refs() {
  local file="$1"
  python3 - "$file" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
seen = set()
for result in data.get("results") or []:
    ref = (result.get("run_ref") or "").strip()
    if ref and ref not in seen:
        seen.add(ref)
        print(ref)
PY
}

write_supervise_payload() {
  local run_ref="$1"
  local tick="$2"
  local output_file="$3"
  python3 - "$run_ref" "$tick" >"$output_file" <<'PY'
import json
import sys
run_ref = sys.argv[1]
tick = sys.argv[2]
print(json.dumps({
    "run_ref": run_ref,
    "max_ticks": 8,
    "max_bursts": 16,
    "max_steps_per_burst": 8,
    "max_dispatches_per_wait": 8,
    "max_commands": 32,
    "max_outbox_per_cycle": 16,
    "max_external_waits": 2,
    "continue_message": "smoke derivados OPES tick " + tick + ": sigue hasta entregar artefacto"
}))
PY
}

post_supervise_run() {
  local run_ref="$1"
  local tick="$2"
  local payload_file="$SMOKE_OUT_DIR/supervise_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}.json"
  local response_file="$SMOKE_OUT_DIR/supervise_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_response.json"
  write_supervise_payload "$run_ref" "$tick" "$payload_file"
  local status
  status="$(curl -sS -m 120 -o "$response_file" -w "%{http_code}" \
    -X POST "$ORQUESTA_BASE_URL_EFFECTIVE/api/v0/runs/supervise" \
    -H "Content-Type: application/json" \
    --data-binary "@$payload_file")"
  if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then
    echo "supervise fallo run_ref=$run_ref status=$status response=$response_file" >&2
    cat "$response_file" >&2 || true
    exit 1
  fi
  cat "$response_file"
}

run_until_final_type() {
  smoke_require_tools python3 curl
  local final_type
  final_type="${SEQUENCE##*,}"
  local final_seen=0
  local final_summary=""
  for tick in $(seq 1 "$MAX_TICKS"); do
    local summary_file="$SMOKE_OUT_DIR/opes_derivatives_rest_tick_${tick}_drain_summary.json"
    run_execute_drain_once_to "$summary_file"
    local selected
    selected="$(json_summary_field "$summary_file" "selected_job_type")"
    if [[ -z "$selected" ]]; then
      if [[ "$final_seen" == "1" ]]; then
        final_summary="$summary_file"
        echo "run_until_status=completed"
        echo "run_until_mode=$MODE"
        echo "final_job_type=$final_type"
        echo "final_summary=$final_summary"
        return
      fi
      echo "sin pendientes antes de alcanzar $final_type en tick $tick" >&2
      echo "summary=$summary_file" >&2
      exit 1
    fi
    if [[ "$selected" == "$final_type" ]]; then
      final_seen=1
    fi
    local run_refs=()
    while IFS= read -r run_ref; do
      [[ -n "$run_ref" ]] && run_refs+=("$run_ref")
    done < <(json_summary_run_refs "$summary_file")
    if [[ "${#run_refs[@]}" -eq 0 ]]; then
      echo "tick $tick no produjo run_ref supervisable: $summary_file" >&2
      exit 1
    fi
    local run_ref
    for run_ref in "${run_refs[@]}"; do
      post_supervise_run "$run_ref" "$tick" |
        tee "$SMOKE_OUT_DIR/supervise_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_stdout.json"
    done
    if [[ "$selected" == "$final_type" ]]; then
      final_summary="$summary_file"
      echo "run_until_status=completed"
      echo "run_until_mode=$MODE"
      echo "final_job_type=$final_type"
      echo "final_summary=$final_summary"
      return
    fi
    if [[ "$tick" -lt "$MAX_TICKS" ]]; then
      sleep "$TICK_SLEEP_SECONDS"
    fi
  done
  echo "no se alcanzo $final_type en $MAX_TICKS ticks" >&2
  exit 1
}

main() {
  smoke_require_tools go tee
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
    run-until-assemble | run-until-finalize | run-until-final)
      run_until_final_type
      ;;
    *)
      echo "modo no soportado: $MODE (usa dry-run-once, drain-once, run-until-finalize o run-until-assemble)" >&2
      exit 2
      ;;
  esac
}

main "$@"
