#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

DEFAULT_SEQUENCE="update_topic_registry,research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site,generate_help_manual_assets,finalize_temario_package"
MODE="${ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE:-dry-run-once}"
PREFLIGHT_TARGET_MODE="${ORQUESTA_OPES_DERIVATIVES_PREFLIGHT_TARGET_MODE:-run-until-finalize}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/derivatives-rest-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
ORQUESTA_BASE_URL_EFFECTIVE="${ORQUESTA_BASE_URL:-}"
SEQUENCE="${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-$DEFAULT_SEQUENCE}"
LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"
INPUT_LEDGER_PATH="${ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH:-$SMOKE_OUT_DIR/external-bridge-input-ledger.json}"
MAX_TICKS="${ORQUESTA_OPES_BRIDGE_MAX_TICKS:-40}"
TICK_SLEEP_SECONDS="${ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS:-5}"
SCOPE_PROBE_NEGATIVE_SUFFIX="${ORQUESTA_OPES_SCOPE_PROBE_NEGATIVE_SUFFIX:-__orquesta_scope_probe_absent__}"
SCOPE_PROBE_OUTPUT="${ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT:-$SMOKE_OUT_DIR/opes_derivatives_scope_probe.json}"
FAKE_SERVER="${ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER:-0}"
FAKE_PENDING_TYPE="${ORQUESTA_OPES_DERIVATIVES_FAKE_PENDING_TYPE:-assemble_topic}"
FAKE_GOAL_FIRST="${ORQUESTA_OPES_DERIVATIVES_FAKE_GOAL_FIRST:-1}"
FAKE_DIR=""
FAKE_PID=""

is_effectful_mode() {
  case "$1" in
    drain-once | run-until-assemble | run-until-finalize | run-until-final)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

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
  if [[ "$MODE" != "dry-run-once" && "$MODE" != "scope-probe" ]] && ! is_run_until_mode "$MODE"; then
    echo "fake OPES solo soporta dry-run-once, scope-probe, run-until-finalize o run-until-assemble" >&2
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
goal_first = sys.argv[6] == "1"
active_index = sequence.index(pending_type) if mode == "dry-run-once" else 0
expected_scan_limit = str(min(max(max(int(limit), 1) * 100, 100), 500))
runs = {}

payload_by_type = {
    "update_topic_registry": {
        "program_id": "program-ref-fake-operario-001",
        "course_id": "course-ref-fake-operario-001",
        "topic_id": "topic-ref-fake-operario-001",
        "topic_registry_ref": "topic-registry-ref-fake-operario-001",
        "operation": "claim_or_update",
    },
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
            if job_type in {"review_textual", "generate_tutor_assets", "generate_help_manual_assets"}:
                send_json(self, 200, [])
                return
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
        payload = payload_for(job_type)
        if query.get("program_id", [""])[0] and payload.get("program_id") != query.get("program_id", [""])[0]:
            send_json(self, 200, [])
            return
        if query.get("topic_id", [""])[0] and payload.get("topic_id") != query.get("topic_id", [""])[0]:
            send_json(self, 200, [])
            return
        correlation_id = "corr-ref-fake-" + job_type.replace("_", "-") + "-001"
        if query.get("correlation_id", [""])[0] and correlation_id != query.get("correlation_id", [""])[0]:
            send_json(self, 200, [])
            return
        send_json(self, 200, [{
                "id": "job-ref-fake-" + job_type.replace("_", "-") + "-001",
                "type": job_type,
                "status": "pending",
                "execution_mode": "external",
                "payload_json": json.dumps(payload),
                "correlation_id": correlation_id,
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
            goal_ref = "goal-ref-fake-" + job_ref
            external_goal_ref = "thread-ref-fake-" + job_ref
            runs[run_ref] = {
                "job_ref": job_ref,
                "work_kind": work_kind,
                "artifact_type": artifact_type,
                "stage": active_index,
                "supervised": False,
                "observed": False,
                "goal_first": goal_first,
                "goal_ref": goal_ref,
                "external_goal_ref": external_goal_ref,
            }
            if goal_first:
                send_json(self, 200, {
                    "run_ref": run_ref,
                    "estado": "ok",
                    "route_policy": "goal_first",
                    "director_execution_mode": "goal_first",
                    "goal_ref": goal_ref,
                    "external_goal_ref": external_goal_ref,
                    "next_actions": ["observe_goal"],
                })
                return
            send_json(self, 200, {"run_ref": run_ref, "estado": "accepted"})
            return
        if parsed.path == "/api/v0/runs/supervise":
            run_ref = payload.get("run_ref") or ""
            run = runs.get(run_ref)
            if not run:
                send_json(self, 404, {"error": "run_not_found", "run_ref": run_ref})
                return
            if run.get("goal_first"):
                send_json(self, 200, {
                    "run_ref": run_ref,
                    "estado": "ok",
                    "stop_reason": "goal_first_observe_required",
                    "director_execution_mode": "goal_first",
                    "goal_ref": run["goal_ref"],
                    "external_goal_ref": run["external_goal_ref"],
                    "next_actions": ["observe_goal", "do_not_supervise_goal_first_with_legacy_loop"],
                    "last": {
                        "status": "running",
                        "evidence_refs": ["evidence-ref-fake-goal-first-supervisor"],
                    },
                })
                return
            if not run["supervised"]:
                run["supervised"] = True
                if run["stage"] == active_index:
                    active_index += 1
            send_json(self, 200, {"run_ref": run_ref, "status": "supervised", "work_kind": run["work_kind"], "artifact_type": run["artifact_type"], "next_stage_index": active_index})
            return
        if parsed.path == "/api/v0/apps/director/goal/observe":
            run_ref = payload.get("run_ref") or ""
            run = runs.get(run_ref)
            if not run:
                send_json(self, 404, {"error": "run_not_found", "run_ref": run_ref})
                return
            if not run.get("goal_first"):
                send_json(self, 400, {"error": "run_not_goal_first", "run_ref": run_ref})
                return
            if not run["observed"]:
                run["observed"] = True
                if run["stage"] == active_index:
                    active_index += 1
            send_json(self, 200, {
                "estado": "ok",
                "run_ref": run_ref,
                "run_status": "closed",
                "director_execution_mode": "goal_first",
                "goal_ref": run["goal_ref"],
                "external_goal_ref": run["external_goal_ref"],
                "goal_status": "complete",
                "closure_status": "accepted",
                "closure_accepted": True,
                "artifact_refs": ["artifact-ref-fake-" + run["work_kind"]],
                "domain_receipt_refs": ["domain-receipt-ref-fake-" + run["job_ref"]],
                "evidence_refs": ["evidence-ref-fake-goal-observed"],
            })
            return
        send_json(self, 404, {"error": "unexpected_path", "path": parsed.path})

if pending_type not in sequence:
    raise SystemExit(f"pending type {pending_type!r} no esta en secuencia {sequence!r}")

server = HTTPServer(("127.0.0.1", 0), Handler)
with open(url_file, "w", encoding="utf-8") as fh:
    fh.write(f"http://127.0.0.1:{server.server_port}\n")
server.serve_forever()
PY
  python3 "$server_py" "$url_file" "$SEQUENCE" "$LIMIT" "$FAKE_PENDING_TYPE" "$MODE" "$FAKE_GOAL_FIRST" &
  FAKE_PID="$!"
  for _ in $(seq 1 50); do
    if [[ -s "$url_file" ]]; then
      OPES_BASE_URL_EFFECTIVE="$(cat "$url_file")"
      if is_run_until_mode "$MODE"; then
        ORQUESTA_BASE_URL_EFFECTIVE="$OPES_BASE_URL_EFFECTIVE"
      fi
      export ORQUESTA_OPES_TEMPORAL_CONFIRM="${ORQUESTA_OPES_TEMPORAL_CONFIRM:-1}"
      export ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF="${ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF:-evidence-ref-opes-derivatives-fake-goal-first}"
      if [[ "$MODE" != "scope-probe" ]]; then
        export ORQUESTA_OPES_BRIDGE_PROGRAM_ID="${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-program-ref-fake-operario-001}"
      fi
      export ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY="${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY:-available}"
      export ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY_REF="${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY_REF:-speech-synthesis-fake-opes-derivatives}"
      export ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS="${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS:-evidence-ref-opes-derivatives-fake-speech-synthesis}"
      return
    fi
    sleep 0.1
  done
  echo "fake OPES no arranco" >&2
  exit 1
}

require_temporal_opes() {
  if [[ "$FAKE_SERVER" == "1" ]]; then
    if [[ "$MODE" == "preflight-only" ]]; then
      OPES_BASE_URL_EFFECTIVE="${OPES_BASE_URL_EFFECTIVE:-http://127.0.0.1:0}"
      return
    fi
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

preflight_target_mode() {
  if [[ "$MODE" == "preflight-only" ]]; then
    printf '%s\n' "$PREFLIGHT_TARGET_MODE"
    return
  fi
  printf '%s\n' "$MODE"
}

preflight_scope_summary() {
  local scopes=()
  if [[ "${ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE:-0}" == "1" ]]; then
    scopes+=("dedicated_temporal_queue")
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_CORRELATION_ID:-}" ]]; then
    scopes+=("correlation_id")
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_TOPIC_ID:-}" ]]; then
    scopes+=("topic_id")
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" ]]; then
    scopes+=("program_id")
  fi
  if [[ "${#scopes[@]}" -eq 0 ]]; then
    printf 'none\n'
    return
  fi
  local old_ifs="$IFS"
  IFS=,
  printf '%s\n' "${scopes[*]}"
  IFS="$old_ifs"
}

sequence_has_job_type() {
  local needle="$1"
  local normalized="${SEQUENCE//,/ }"
  normalized="${normalized//;/ }"
  local item
  for item in $normalized; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

speech_synthesis_capability_available() {
  local capability="${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY:-}"
  case "${capability,,}" in
    1 | true | yes | y | si | available | enabled | ready | ok)
      return 0
      ;;
  esac
  return 1
}

speech_synthesis_evidence_refs_present() {
  local refs="${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS:-}"
  refs="${refs//,/ }"
  refs="${refs//;/ }"
  local ref
  for ref in $refs; do
    if [[ "$ref" == evidence-ref-* || "$ref" == receipt-ref-* || "$ref" == artifact-ref-* ]]; then
      return 0
    fi
  done
  return 1
}

compact_evidence_ref_present() {
  local refs="$1"
  refs="${refs//,/ }"
  refs="${refs//;/ }"
  local ref
  for ref in $refs; do
    if [[ "$ref" == evidence-ref-* || "$ref" == receipt-ref-* || "$ref" == artifact-ref-* ]]; then
      return 0
    fi
  done
  return 1
}

scope_probe_file_matches_program_id() {
  local file="$1"
  [[ -f "$file" ]] || return 1
  smoke_require_tool python3
  python3 - "$file" "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" "$OPES_BASE_URL_EFFECTIVE" <<'PY'
import hashlib
import json
import sys

path, expected_program_id, expected_base_url = sys.argv[1:]
try:
    with open(path, "r", encoding="utf-8") as fh:
        data = json.load(fh)
except Exception:
    raise SystemExit(1)
if data.get("scope_probe_status") != "ok":
    raise SystemExit(1)
if bool(data.get("fake_server")):
    raise SystemExit(1)
expected_hash = hashlib.sha256(expected_base_url.rstrip("/").encode("utf-8")).hexdigest()
if str(data.get("base_url_hash") or "").strip() != expected_hash:
    raise SystemExit(1)
scope_kind = data.get("scope_kind") or []
if isinstance(scope_kind, str):
    scope_kind = [scope_kind]
if "program_id" not in [str(item).strip() for item in scope_kind]:
    raise SystemExit(1)
job_type = str(data.get("job_type") or "").strip()
if not job_type:
    raise SystemExit(1)
try:
    seen = int(data.get("seen"))
except Exception:
    raise SystemExit(1)
if seen < 1:
    raise SystemExit(1)
scopes = data.get("scopes") or {}
if str(scopes.get("program_id") or "").strip() != expected_program_id.strip():
    raise SystemExit(1)
if "program_id" not in [str(item).strip() for item in (data.get("negative_checks") or [])]:
    raise SystemExit(1)
raise SystemExit(0)
PY
}

scope_filter_evidence_present() {
  compact_evidence_ref_present "${ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF:-}" ||
    scope_probe_file_matches_program_id "$SCOPE_PROBE_OUTPUT"
}

require_derivatives_real_preflight() {
  if [[ "$FAKE_SERVER" == "1" ]]; then
    return
  fi
  local target_mode
  target_mode="$(preflight_target_mode)"
  if [[ "$MODE" == "preflight-only" ]] && ! is_effectful_mode "$target_mode"; then
    echo "ORQUESTA_OPES_DERIVATIVES_PREFLIGHT_TARGET_MODE debe ser un modo con efectos: drain-once, run-until-assemble, run-until-finalize o run-until-final" >&2
    exit 2
  fi
  if ! is_effectful_mode "$target_mode"; then
    return
  fi
  if [[ "${ORQUESTA_OPES_DERIVATIVES_EXECUTE:-0}" != "1" ]]; then
    echo "smoke derivados real bloqueado: falta confirmacion de efectos ORQUESTA_OPES_DERIVATIVES_EXECUTE=1" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM:-0}" == "1" ]]; then
    echo "smoke derivados real bloqueado: no ejecutar esta cadena contra OPES productivo" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED:-0}" == "1" ]]; then
    echo "smoke derivados real bloqueado: ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED no es valido para la secuencia OPES temporal" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED:-0}" == "1" ]]; then
    echo "smoke derivados real bloqueado: no mezclar ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED con finalize_temario_package del bridge" >&2
    exit 2
  fi
  if [[ "$LIMIT" != "1" && "${ORQUESTA_OPES_DERIVATIVES_ALLOW_LIMIT_GT_1:-0}" != "1" ]]; then
    echo "smoke derivados real bloqueado: usa ORQUESTA_OPES_BRIDGE_LIMIT=1 en el primer pase real" >&2
    echo "si la cola temporal ya esta aislada y revisada, exporta ORQUESTA_OPES_DERIVATIVES_ALLOW_LIMIT_GT_1=1" >&2
    exit 2
  fi
  if [[ -z "$ORQUESTA_BASE_URL_EFFECTIVE" ]]; then
    echo "smoke derivados real bloqueado: falta ORQUESTA_BASE_URL explicito para crear runs goal-first en Orquesta temporal" >&2
    exit 2
  fi
  if [[ "$target_mode" == "run-until-finalize" ||
    "$target_mode" == "run-until-final" ||
    "$target_mode" == "drain-once" ]] &&
    sequence_has_job_type "generate_audio_asset" &&
    ! speech_synthesis_capability_available; then
    echo "smoke derivados real bloqueado: generate_audio_asset requiere ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available en la composicion temporal" >&2
    exit 2
  fi
  if [[ "$target_mode" == "run-until-finalize" ||
    "$target_mode" == "run-until-final" ||
    "$target_mode" == "drain-once" ]] &&
    sequence_has_job_type "generate_audio_asset" &&
    ! speech_synthesis_evidence_refs_present; then
    echo "smoke derivados real bloqueado: speech_synthesis disponible requiere ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS con refs de capacidad/runner temporal" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_TOPIC_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_CORRELATION_ID:-}" &&
    "${ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE:-0}" != "1" &&
    "${ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED:-0}" != "1" ]]; then
    echo "smoke derivados real bloqueado: program_id documental no basta; confirma filtro real con ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 o usa cola temporal dedicada/correlation_id/topic_id" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_TOPIC_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_CORRELATION_ID:-}" &&
    "${ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE:-0}" != "1" &&
    "${ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED:-0}" == "1" ]]; then
    if ! scope_filter_evidence_present; then
      echo "smoke derivados real bloqueado: SCOPE_FILTER_CONFIRMED=1 requiere ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF o scope-probe JSON valido en $SCOPE_PROBE_OUTPUT" >&2
      exit 2
    fi
  fi
  if [[ -z "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_TOPIC_ID:-}" &&
    -z "${ORQUESTA_OPES_BRIDGE_CORRELATION_ID:-}" &&
    "${ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE:-0}" != "1" ]]; then
    echo "smoke derivados real bloqueado: falta scope operativo; usa cola temporal dedicada, correlation_id, topic_id o program_id con filtro confirmado" >&2
    exit 2
  fi
  if [[ -z "${ORQUESTA_CODEX_GOAL_BACKEND:-}" &&
    "${ORQUESTA_OPES_DERIVATIVES_ORQUESTA_GOAL_FIRST_CONFIRMED:-0}" != "1" ]]; then
    echo "smoke derivados real bloqueado: falta Orquesta temporal goal-first; exporta ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio al arrancar servidor o confirma ORQUESTA_OPES_DERIVATIVES_ORQUESTA_GOAL_FIRST_CONFIRMED=1" >&2
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
    echo "preflight_target_mode=$PREFLIGHT_TARGET_MODE"
    echo "fake_server=$FAKE_SERVER"
    echo "fake_pending_type=$FAKE_PENDING_TYPE"
    echo "fake_goal_first=$FAKE_GOAL_FIRST"
    echo "opes_base_url=$OPES_BASE_URL_EFFECTIVE"
    echo "orquesta_base_url=$ORQUESTA_BASE_URL_EFFECTIVE"
    echo "sequence=$SEQUENCE"
    echo "limit=$LIMIT"
    echo "input_ledger_path=$INPUT_LEDGER_PATH"
    echo "max_ticks=$MAX_TICKS"
    echo "tick_sleep_seconds=$TICK_SLEEP_SECONDS"
    echo "scope_summary=$(preflight_scope_summary)"
    echo "scope_filter_confirmed=${ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED:-0}"
    echo "scope_filter_evidence_ref=${ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF:-}"
    echo "scope_probe_output=$SCOPE_PROBE_OUTPUT"
    echo "dedicated_temporal_queue=${ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE:-0}"
    echo "speech_synthesis_capability=${ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY:-}"
    echo "goal_backend=${ORQUESTA_CODEX_GOAL_BACKEND:-}"
    echo "goal_first_confirmed=${ORQUESTA_OPES_DERIVATIVES_ORQUESTA_GOAL_FIRST_CONFIRMED:-0}"
    echo "output_dir=$SMOKE_OUT_DIR"
  } >"$SMOKE_OUT_DIR/metadata.txt"
}

run_preflight_only() {
  echo "preflight_status=ok"
  echo "preflight_target_mode=$(preflight_target_mode)"
  echo "preflight_scope=$(preflight_scope_summary)"
  echo "preflight_output_dir=$SMOKE_OUT_DIR"
}

opes_bridge_scan_limit() {
  local base="$LIMIT"
  if [[ "$base" -lt 1 ]]; then
    base=1
  fi
  local scan=$((base * 100))
  if [[ "$scan" -lt 100 ]]; then
    scan=100
  fi
  if [[ "$scan" -gt 500 ]]; then
    scan=500
  fi
  printf '%s\n' "$scan"
}

run_scope_probe() {
  smoke_require_tool python3
  local output_file="$SCOPE_PROBE_OUTPUT"
  mkdir -p "$(dirname "$output_file")"
  python3 - \
    "$OPES_BASE_URL_EFFECTIVE" \
    "$SEQUENCE" \
    "$(opes_bridge_scan_limit)" \
    "${ORQUESTA_OPES_BRIDGE_PROGRAM_ID:-}" \
    "${ORQUESTA_OPES_BRIDGE_TOPIC_ID:-}" \
    "${ORQUESTA_OPES_BRIDGE_CORRELATION_ID:-}" \
    "$SCOPE_PROBE_NEGATIVE_SUFFIX" \
    "$output_file" \
    "$FAKE_SERVER" <<'PY'
import datetime
import hashlib
import json
import sys
import urllib.error
import urllib.parse
import urllib.request

base_url, sequence_raw, limit, program_id, topic_id, correlation_id, negative_suffix, output_file, fake_server = sys.argv[1:]
sequence = [item.strip() for item in sequence_raw.replace(";", ",").split(",") if item.strip()]
scopes = {
    "program_id": program_id.strip(),
    "topic_id": topic_id.strip(),
    "correlation_id": correlation_id.strip(),
}
active_scopes = {key: value for key, value in scopes.items() if value}
if not active_scopes:
    raise SystemExit("scope-probe requiere ORQUESTA_OPES_BRIDGE_PROGRAM_ID, TOPIC_ID o CORRELATION_ID")
if not sequence:
    raise SystemExit("scope-probe requiere JOB_TYPE_SEQUENCE no vacia")

def normalize_jobs(raw):
    if isinstance(raw, list):
        return raw
    if isinstance(raw, dict):
        for key in ("jobs", "items", "results", "data"):
            value = raw.get(key)
            if isinstance(value, list):
                return value
    raise SystemExit("scope-probe recibio respuesta OPES no reconocida")

def payload_map(job):
    raw = job.get("payload_json") or job.get("payload") or "{}"
    if isinstance(raw, dict):
        return raw
    if not isinstance(raw, str) or not raw.strip():
        return {}
    try:
        parsed = json.loads(raw)
    except Exception:
        return {}
    return parsed if isinstance(parsed, dict) else {}

def job_scope_value(job, key):
    if key == "correlation_id":
        return str(job.get("correlation_id") or "").strip()
    payload = payload_map(job)
    value = payload.get(key)
    if value is None:
        value = (job.get("external_refs") or {}).get(key) if isinstance(job.get("external_refs"), dict) else None
    return str(value or "").strip()

def transport_job_type(work_kind):
    review_types = {
        "update_topic_registry",
        "claim_topic_registry",
        "release_topic_registry",
        "review_codex",
        "review_gemini",
        "review_claude",
        "review_pair_codex_gemini",
        "review_pair_codex_claude",
        "review_pair_gemini_claude",
        "review_director_consolidation",
        "review_director_final",
        "review_consensus_director",
        "generate_agent_candidate_codex",
        "generate_agent_candidate_gemini",
        "generate_agent_candidate_claude",
        "generate_provider_candidate",
        "vote_agent_candidates_codex",
        "vote_agent_candidates_gemini",
        "vote_agent_candidates_claude",
        "vote_provider_candidates",
        "select_agent_candidate_director",
        "select_provider_candidate_director",
    }
    games_types = {
        "generate_learning_games",
        "create_learning_games",
        "generate_course_games",
    }
    final_types = {
        "finalize_temario_package",
        "close_temario_package",
        "finalize_topic_package",
        "finalize_domain_package",
        "finalize_syllabus_package",
    }
    work_kind = str(work_kind or "").strip()
    if work_kind in review_types:
        return "review_textual"
    if work_kind in games_types:
        return "generate_tutor_assets"
    if work_kind in final_types:
        return "generate_help_manual_assets"
    return work_kind

def query_job_types(work_kind):
    out = []
    for candidate in (work_kind, transport_job_type(work_kind)):
        candidate = str(candidate or "").strip()
        if candidate and candidate not in out:
            out.append(candidate)
    return out

def effective_work_kind(job):
    payload = payload_map(job)
    value = payload.get("work_kind")
    if value is None:
        value = job.get("work_kind") or job.get("type") or job.get("job_type")
    return str(value or "").strip()

def get_jobs(job_type, overrides=None):
    params = {
        "execution_mode": "external",
        "status": "pending",
        "job_type": job_type,
        "limit": str(limit),
    }
    params.update(active_scopes)
    if overrides:
        params.update(overrides)
    url = base_url.rstrip("/") + "/api/jobs?" + urllib.parse.urlencode(params)
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            body = response.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")[:500]
        raise SystemExit(f"scope-probe GET {job_type} HTTP {exc.code}: {detail}")
    return normalize_jobs(json.loads(body.decode("utf-8") or "[]"))

positive = None
for job_type in sequence:
    for query_job_type in query_job_types(job_type):
        jobs = [job for job in get_jobs(query_job_type) if effective_work_kind(job) == job_type]
        if not jobs:
            continue
        mismatches = []
        for job in jobs:
            for key, expected in active_scopes.items():
                got = job_scope_value(job, key)
                if got != expected:
                    mismatches.append({
                        "job_ref": job.get("id") or job.get("job_ref") or "",
                        "scope": key,
                        "expected": expected,
                        "got": got,
                    })
        if mismatches:
            result = {
                "scope_probe_status": "failed_scope_mismatch",
                "job_type": job_type,
                "transport_job_type": query_job_type,
                "mismatches": mismatches,
            }
            with open(output_file, "w", encoding="utf-8") as fh:
                json.dump(result, fh, ensure_ascii=False, indent=2)
            raise SystemExit("scope-probe fallo: OPES devolvio jobs fuera del scope configurado")
        positive = {"job_type": job_type, "transport_job_type": query_job_type, "jobs": jobs}
        break
    if positive is not None:
        break

if positive is None:
    result = {
        "scope_probe_status": "no_matching_jobs",
        "scopes": active_scopes,
        "sequence": sequence,
    }
    with open(output_file, "w", encoding="utf-8") as fh:
        json.dump(result, fh, ensure_ascii=False, indent=2)
    raise SystemExit("scope-probe no encontro jobs pendientes para demostrar el filtro real")

negative_checks = []
for key, value in active_scopes.items():
    impossible = value + negative_suffix
    jobs = [
        job
        for job in get_jobs(positive["transport_job_type"], {key: impossible})
        if effective_work_kind(job) == positive["job_type"]
    ]
    if jobs:
        result = {
            "scope_probe_status": "failed_negative_filter",
            "job_type": positive["job_type"],
            "transport_job_type": positive["transport_job_type"],
            "scope": key,
            "unexpected_jobs": [job.get("id") or job.get("job_ref") or "" for job in jobs],
        }
        with open(output_file, "w", encoding="utf-8") as fh:
            json.dump(result, fh, ensure_ascii=False, indent=2)
        raise SystemExit(f"scope-probe fallo: OPES no respeto filtro negativo {key}")
    negative_checks.append(key)

result = {
    "scope_probe_status": "ok",
    "job_type": positive["job_type"],
    "transport_job_type": positive["transport_job_type"],
    "seen": len(positive["jobs"]),
    "scopes": active_scopes,
    "scope_kind": list(active_scopes.keys()),
    "base_url_hash": hashlib.sha256(base_url.rstrip("/").encode("utf-8")).hexdigest(),
    "fake_server": fake_server == "1",
    "created_at": datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z"),
    "negative_checks": negative_checks,
}
with open(output_file, "w", encoding="utf-8") as fh:
    json.dump(result, fh, ensure_ascii=False, indent=2)
print("scope_probe_status=ok")
print("scope_probe_job_type=" + positive["job_type"])
if positive["transport_job_type"] != positive["job_type"]:
    print("scope_probe_transport_job_type=" + positive["transport_job_type"])
print("scope_probe_seen=" + str(len(positive["jobs"])))
print("scope_probe_negative_checks=" + ",".join(negative_checks))
print("scope_probe_output=" + output_file)
PY
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

json_summary_run_records() {
  local file="$1"
  python3 - "$file" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
seen = set()
for result in data.get("results") or []:
    ref = (result.get("run_ref") or "").strip()
    if not ref or ref in seen:
        continue
    seen.add(ref)
    actions = [str(item).strip() for item in (result.get("next_actions") or [])]
    goal_first = (
        (result.get("route_policy") or "").strip() == "goal_first"
        or (result.get("director_execution_mode") or "").strip() == "goal_first"
        or bool((result.get("goal_ref") or "").strip())
        or bool((result.get("external_goal_ref") or "").strip())
        or "observe_goal" in actions
    )
    print(ref + "\t" + ("1" if goal_first else "0"))
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
    "director_execution_mode": "legacy_director_loop",
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
  smoke_require_confirm \
    ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP \
    1 \
    "fallback legacy desactivado: exporta ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1 para supervisar runs no goal-first"
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

write_observe_goal_payload() {
  local run_ref="$1"
  local tick="$2"
  local output_file="$3"
  python3 - "$run_ref" "$tick" >"$output_file" <<'PY'
import json
import sys
run_ref = sys.argv[1]
tick = sys.argv[2]
print(json.dumps({
    "request_id": "req-smoke-opes-derivatives-observe-goal-" + tick,
    "correlation_id": "corr-smoke-opes-derivatives-observe-goal-" + tick,
    "run_ref": run_ref,
    "requested_by": "smoke-opes-derivatives-rest"
}))
PY
}

post_observe_goal() {
  local run_ref="$1"
  local tick="$2"
  local payload_file="$SMOKE_OUT_DIR/observe_goal_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}.json"
  local response_file="$SMOKE_OUT_DIR/observe_goal_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_response.json"
  write_observe_goal_payload "$run_ref" "$tick" "$payload_file"
  local status
  status="$(curl -sS -m 120 -o "$response_file" -w "%{http_code}" \
    -X POST "$ORQUESTA_BASE_URL_EFFECTIVE/api/v0/apps/director/goal/observe" \
    -H "Content-Type: application/json" \
    --data-binary "@$payload_file")"
  if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then
    echo "observe goal fallo run_ref=$run_ref status=$status response=$response_file" >&2
    cat "$response_file" >&2 || true
    exit 1
  fi
  cat "$response_file"
}

supervise_response_requires_goal_observe() {
  local file="$1"
  python3 - "$file" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
actions = [str(item).strip() for item in (data.get("next_actions") or [])]
if (
    (data.get("stop_reason") or "").strip() == "goal_first_observe_required"
    or (data.get("director_execution_mode") or "").strip() == "goal_first"
    or "observe_goal" in actions
):
    raise SystemExit(0)
raise SystemExit(1)
PY
}

write_goal_receipts_manifest() {
  local output_file="$SMOKE_OUT_DIR/goal_receipts_manifest.json"
  python3 - "$SMOKE_OUT_DIR" "$output_file" <<'PY'
import glob
import json
import os
import sys

smoke_dir, output_file = sys.argv[1:]
runs = {}
for path in sorted(glob.glob(os.path.join(smoke_dir, "opes_derivatives_rest_tick_*_drain_summary.json"))):
    try:
        with open(path, "r", encoding="utf-8") as fh:
            summary = json.load(fh)
    except Exception:
        continue
    for result in summary.get("results") or []:
        run_ref = str(result.get("run_ref") or "").strip()
        if not run_ref:
            continue
        actions = [str(item).strip() for item in (result.get("next_actions") or [])]
        goal_first = (
            str(result.get("route_policy") or "").strip() == "goal_first"
            or str(result.get("director_execution_mode") or "").strip() == "goal_first"
            or bool(str(result.get("goal_ref") or "").strip())
            or bool(str(result.get("external_goal_ref") or "").strip())
            or "observe_goal" in actions
        )
        runs.setdefault(run_ref, {
            "run_ref": run_ref,
            "job_ref": str(result.get("job_ref") or "").strip(),
            "work_kind": str(result.get("work_kind") or "").strip(),
            "goal_ref": str(result.get("goal_ref") or "").strip(),
            "external_goal_ref": str(result.get("external_goal_ref") or "").strip(),
            "goal_first": goal_first,
            "artifact_refs": [],
            "domain_receipt_refs": [],
            "evidence_refs": [],
            "closure_accepted": False,
            "closure_status": "",
            "observed": False,
        })
        runs[run_ref]["goal_first"] = runs[run_ref]["goal_first"] or goal_first

for path in sorted(glob.glob(os.path.join(smoke_dir, "observe_goal_*_response.json"))):
    try:
        with open(path, "r", encoding="utf-8") as fh:
            response = json.load(fh)
    except Exception:
        continue
    run_ref = str(response.get("run_ref") or "").strip()
    if not run_ref:
        continue
    entry = runs.setdefault(run_ref, {
        "run_ref": run_ref,
        "job_ref": "",
        "work_kind": "",
        "goal_ref": "",
        "external_goal_ref": "",
        "goal_first": True,
        "artifact_refs": [],
        "domain_receipt_refs": [],
        "evidence_refs": [],
        "closure_accepted": False,
        "closure_status": "",
        "observed": False,
    })
    entry["observed"] = True
    entry["goal_ref"] = entry["goal_ref"] or str(response.get("goal_ref") or "").strip()
    entry["external_goal_ref"] = entry["external_goal_ref"] or str(response.get("external_goal_ref") or "").strip()
    entry["artifact_refs"] = [str(item).strip() for item in (response.get("artifact_refs") or []) if str(item).strip()]
    entry["domain_receipt_refs"] = [str(item).strip() for item in (response.get("domain_receipt_refs") or []) if str(item).strip()]
    entry["evidence_refs"] = [str(item).strip() for item in (response.get("evidence_refs") or []) if str(item).strip()]
    entry["closure_accepted"] = bool(response.get("closure_accepted"))
    entry["closure_status"] = str(response.get("closure_status") or "").strip()

entries = [runs[key] for key in sorted(runs)]
issues = []
for entry in entries:
    if not entry.get("goal_first"):
        continue
    if not entry.get("observed"):
        issues.append({"run_ref": entry["run_ref"], "code": "goal_not_observed"})
    if not entry.get("closure_accepted"):
        issues.append({"run_ref": entry["run_ref"], "code": "goal_closure_not_accepted"})
    if not entry.get("artifact_refs"):
        issues.append({"run_ref": entry["run_ref"], "code": "goal_artifact_refs_missing"})
    if not entry.get("domain_receipt_refs"):
        issues.append({"run_ref": entry["run_ref"], "code": "goal_domain_receipt_refs_missing"})

manifest = {
    "schema_version": "orquesta_opes_derivatives_goal_receipts_manifest.v0",
    "entries": entries,
    "issues": issues,
}
with open(output_file, "w", encoding="utf-8") as fh:
    json.dump(manifest, fh, ensure_ascii=False, indent=2)
if issues:
    print("goal_receipts_manifest_status=failed")
    print("goal_receipts_manifest=" + output_file)
    raise SystemExit(1)
print("goal_receipts_manifest_status=ok")
print("goal_receipts_manifest=" + output_file)
print("goal_receipts_manifest_entries=" + str(len(entries)))
PY
}

run_until_final_type() {
  smoke_require_tools python3 curl
  local final_type
  case "$MODE" in
    run-until-assemble)
      final_type="assemble_topic"
      ;;
    run-until-finalize | run-until-final)
      final_type="${SEQUENCE##*,}"
      ;;
    *)
      echo "modo run-until no soportado: $MODE" >&2
      exit 2
      ;;
  esac
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
        write_goal_receipts_manifest
        echo "run_until_status=completed"
        echo "run_until_mode=$MODE"
        echo "final_job_type=$final_type"
        echo "final_summary=$final_summary"
        echo "empty_after_final=true"
        return
      fi
      echo "sin pendientes antes de alcanzar $final_type en tick $tick" >&2
      echo "summary=$summary_file" >&2
      exit 1
    fi
    if [[ "$selected" == "$final_type" ]]; then
      final_seen=1
    fi
    local run_records=()
    while IFS= read -r record; do
      [[ -n "$record" ]] && run_records+=("$record")
    done < <(json_summary_run_records "$summary_file")
    if [[ "${#run_records[@]}" -eq 0 ]]; then
      echo "tick $tick no produjo run_ref supervisable: $summary_file" >&2
      exit 1
    fi
    local record
    for record in "${run_records[@]}"; do
      local run_ref=""
      local goal_first=""
      IFS=$'\t' read -r run_ref goal_first <<<"$record"
      if [[ "$goal_first" == "1" ]]; then
        post_observe_goal "$run_ref" "$tick" |
          tee "$SMOKE_OUT_DIR/observe_goal_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_stdout.json"
        continue
      fi
      local supervise_stdout="$SMOKE_OUT_DIR/supervise_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_stdout.json"
      post_supervise_run "$run_ref" "$tick" | tee "$supervise_stdout"
      if supervise_response_requires_goal_observe "$supervise_stdout"; then
        post_observe_goal "$run_ref" "$tick" |
          tee "$SMOKE_OUT_DIR/observe_goal_${tick}_${run_ref//[^a-zA-Z0-9_.-]/_}_stdout.json"
      fi
    done
    if [[ "$tick" -lt "$MAX_TICKS" ]]; then
      sleep "$TICK_SLEEP_SECONDS"
    fi
  done
  echo "no se alcanzo $final_type en $MAX_TICKS ticks" >&2
  exit 1
}

main() {
  smoke_require_tool tee
  require_temporal_opes
  require_sequence_only
  write_metadata
  if [[ "$MODE" == "scope-probe" ]]; then
    run_scope_probe
    return
  fi
  require_derivatives_real_preflight
  cd "$repo_root"

  case "$MODE" in
    preflight-only)
      run_preflight_only
      ;;
    scope-probe)
      run_scope_probe
      ;;
    dry-run-once)
      smoke_require_tool go
      run_dry_run_once
      ;;
    drain-once)
      smoke_require_tool go
      run_execute_drain_once
      ;;
    run-until-assemble | run-until-finalize | run-until-final)
      smoke_require_tool go
      run_until_final_type
      ;;
    *)
      echo "modo no soportado: $MODE (usa preflight-only, scope-probe, dry-run-once, drain-once, run-until-finalize, run-until-final o run-until-assemble)" >&2
      exit 2
      ;;
  esac
}

main "$@"
