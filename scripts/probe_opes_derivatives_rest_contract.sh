#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

default_sequence() {
  sed -n 's/^DEFAULT_SEQUENCE="\(.*\)"$/\1/p' "$repo_root/scripts/smoke_opes_derivatives_rest.sh" | head -n 1
}

SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/opes-contract-probe-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
SEQUENCE="${ORQUESTA_OPES_CONTRACT_PROBE_SEQUENCE:-${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-$(default_sequence)}}"
PROGRAM_ID="${ORQUESTA_OPES_CONTRACT_PROBE_PROGRAM_ID:-program-ref-orquesta-contract-probe-$SMOKE_ID}"
TOPIC_ID="${ORQUESTA_OPES_CONTRACT_PROBE_TOPIC_ID:-topic-ref-orquesta-contract-probe-$SMOKE_ID}"
CORRELATION_ID="${ORQUESTA_OPES_CONTRACT_PROBE_CORRELATION_ID:-corr-orquesta-contract-probe-$SMOKE_ID}"
REQUESTED_BY="${ORQUESTA_OPES_CONTRACT_PROBE_REQUESTED_BY:-orquesta-contract-probe}"
SUMMARY_FILE="${ORQUESTA_OPES_CONTRACT_PROBE_OUTPUT:-$SMOKE_OUT_DIR/opes_derivatives_rest_contract_probe.json}"
EXPECT_FULL_SEQUENCE="${ORQUESTA_OPES_CONTRACT_PROBE_EXPECT_FULL_SEQUENCE:-0}"
TRANSPORT_COMPAT="${ORQUESTA_OPES_CONTRACT_PROBE_TRANSPORT_COMPAT:-1}"

smoke_require_tools curl python3
smoke_require_confirm ORQUESTA_OPES_TEMPORAL_CONFIRM 1 "probe contrato OPES bloqueado: confirma instancia temporal con ORQUESTA_OPES_TEMPORAL_CONFIRM=1"
smoke_require_confirm ORQUESTA_OPES_CONTRACT_PROBE_CREATE 1 "probe contrato OPES bloqueado: crear jobs temporales requiere ORQUESTA_OPES_CONTRACT_PROBE_CREATE=1"

if [[ -z "$OPES_BASE_URL_EFFECTIVE" ]]; then
  echo "probe contrato OPES bloqueado: define ORQUESTA_OPES_BASE_URL apuntando a OPES temporal" >&2
  exit 2
fi
if ! smoke_is_local_url "$OPES_BASE_URL_EFFECTIVE"; then
  echo "probe contrato OPES bloqueado: ORQUESTA_OPES_BASE_URL debe ser loopback para este probe" >&2
  exit 2
fi
if [[ -z "$SEQUENCE" ]]; then
  echo "probe contrato OPES bloqueado: secuencia vacia" >&2
  exit 2
fi

smoke_temp_root_prepare "$SMOKE_OUT_DIR" "opes-contract-probe"

health_file="$SMOKE_OUT_DIR/opes_health.json"
if ! curl -fsS -m 5 "${OPES_BASE_URL_EFFECTIVE%/}/api/health" >"$health_file"; then
  echo "probe contrato OPES bloqueado: /api/health no responde en OPES temporal" >&2
  exit 2
fi

python3 - "$OPES_BASE_URL_EFFECTIVE" "$SEQUENCE" "$PROGRAM_ID" "$TOPIC_ID" "$CORRELATION_ID" "$REQUESTED_BY" "$SUMMARY_FILE" "$EXPECT_FULL_SEQUENCE" "$TRANSPORT_COMPAT" <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

base_url = sys.argv[1].rstrip("/")
sequence = [item.strip() for item in sys.argv[2].split(",") if item.strip()]
program_id = sys.argv[3].strip()
topic_id = sys.argv[4].strip()
correlation_id = sys.argv[5].strip()
requested_by = sys.argv[6].strip()
summary_file = sys.argv[7]
expect_full = sys.argv[8] == "1"
transport_compat = sys.argv[9] == "1"


def expected_artifact_type(job_type):
    mapping = {
        "update_topic_registry": "topic_registry_update",
        "research_exam_precedents": "exam_research_report",
        "draft_content_block": "content_block",
        "generate_visual_asset": "visual_asset",
        "generate_question_bank": "question_bank",
        "review_legal": "block_revision",
        "review_pedagogical": "block_revision",
        "review_quality": "block_revision",
        "review_codex": "agent_review_report",
        "review_gemini": "agent_review_report",
        "review_claude": "agent_review_report",
        "review_pair_codex_gemini": "agent_pair_review_report",
        "review_pair_codex_claude": "agent_pair_review_report",
        "review_pair_gemini_claude": "agent_pair_review_report",
        "review_director_consolidation": "director_review_matrix",
        "validate_topic": "block_revision",
        "assemble_topic": "assembled_topic",
        "generate_audio_asset": "audio_asset",
        "generate_tutor_assets": "tutor_bot_package",
        "generate_learning_games": "learning_games_package",
        "visual_asset_reuse": "visual_reuse_manifest",
        "generate_html_site": "local_html_site",
        "generate_help_manual_assets": "help_manual_package",
        "finalize_temario_package": "completed_syllabus_package",
    }
    return mapping.get(job_type, "work_delivery")


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


def input_for(index, job_type):
    payload = {
        "program_id": program_id,
        "topic_id": topic_id,
        "course_id": "course-ref-orquesta-contract-probe",
        "official_order": 1,
        "topic_title": "Prevencion de riesgos en operaciones auxiliares municipales",
        "official_epigraph_text": "Conceptos basicos de seguridad, senalizacion, equipos de proteccion individual y actuacion ante incidencias en trabajos auxiliares municipales.",
        "source_lesson_markdown": (
            "La prevencion de riesgos en operaciones auxiliares municipales exige reconocer peligros antes de iniciar la tarea, "
            "usar equipos de proteccion individual adecuados, respetar la senalizacion y mantener ordenada la zona de trabajo. "
            "Cuando aparece una incidencia, el personal debe detener la actividad si hay riesgo, avisar por el canal establecido "
            "y registrar la situacion para que pueda corregirse. Los contenidos del tema se organizan en identificacion de riesgos, "
            "medidas preventivas, uso de EPI, senalizacion, comunicacion de incidencias y repaso mediante casos breves."
        ),
        "section_plan": [
            {"section_ref": "sec-riesgos", "title": "Identificacion de riesgos"},
            {"section_ref": "sec-medidas", "title": "Medidas preventivas y EPI"},
            {"section_ref": "sec-incidencias", "title": "Comunicacion y actuacion ante incidencias"},
        ],
        "work_kind": job_type,
        "expected_artifact_type": expected_artifact_type(job_type),
        "probe_ref": f"orquesta-contract-probe-{index:02d}",
    }
    if job_type == "generate_audio_asset":
        payload["audio_profile_ref"] = "accessible-es"
        payload["assembled_topic_artifact_id"] = "artifact-ref-contract-probe-assembled-topic"
        payload["text_public_status"] = "pass"
        payload["audio_regeneration_mode"] = "selective_by_sidecar"
        payload["audio_manifest_ref"] = "audio-manifest-ref-contract-probe"
        payload["source_content_ref"] = "assembled-topic-final-text-ref-contract-probe"
    if job_type == "generate_visual_asset":
        payload["visual_ref"] = "visual-ref-contract-probe"
        payload["asset_type"] = "diagram"
    if job_type == "visual_asset_reuse":
        payload["common_visual_count"] = 1
        payload["common_visual_asset_refs"] = ["visual-ref-common-contract-probe"]
        payload["visual_assets_import_status"] = "pending"
        payload["assembled_topic_artifact_id"] = "artifact-ref-contract-probe-assembled-topic"
    if job_type == "generate_question_bank":
        payload["minimum_questions"] = 50
    if job_type == "assemble_topic":
        payload["document_plan_artifact_id"] = "artifact-ref-contract-probe-document-plan"
    if job_type == "generate_html_site":
        payload["assembled_topic_artifact_id"] = "artifact-ref-contract-probe-assembled-topic"
        payload["visual_reuse_manifest_ref"] = "visual-reuse-manifest-ref-contract-probe"
    if transport_compat:
        transport_type = transport_job_type(job_type)
        if transport_type != job_type:
            payload["transport_job_type"] = transport_type
    return payload


def http_json(method, url, payload=None):
    data = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        data = json.dumps(payload, sort_keys=True).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = response.read().decode("utf-8")
            parsed = json.loads(body) if body.strip() else None
            return response.status, parsed, body
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(body) if body.strip() else None
        except json.JSONDecodeError:
            parsed = None
        return exc.code, parsed, body


accepted = []
rejected = []
responses = []
for index, job_type in enumerate(sequence, start=1):
    request_job_type = transport_job_type(job_type) if transport_compat else job_type
    payload = {
        "job_type": request_job_type,
        "input": input_for(index, job_type),
        "correlation_id": correlation_id,
        "idempotency_key": f"orquesta-contract-probe:{correlation_id}:{index:02d}:{job_type}",
        "requested_by": requested_by,
        "external_refs": {
            "program_id": program_id,
            "topic_id": topic_id,
            "contract_probe": "opes_derivatives_rest",
        },
    }
    status, parsed, raw_body = http_json("POST", f"{base_url}/api/jobs", payload)
    item = {
        "index": index,
        "job_type": job_type,
        "transport_job_type": request_job_type,
        "http_status": status,
    }
    if 200 <= status <= 299:
        job = {}
        if isinstance(parsed, dict):
            job = parsed.get("job") if isinstance(parsed.get("job"), dict) else parsed
        item["job_id"] = str(job.get("id", ""))
        item["created"] = bool(parsed.get("created", False)) if isinstance(parsed, dict) else False
        accepted.append(item)
    else:
        error_value = ""
        if isinstance(parsed, dict):
            error_value = str(parsed.get("error", ""))
        item["error"] = error_value
        item["body_excerpt"] = raw_body[:300]
        rejected.append(item)
    responses.append(item)

query = urllib.parse.urlencode(
    {
        "execution_mode": "external",
        "status": "pending",
        "program_id": program_id,
        "correlation_id": correlation_id,
        "requested_by": requested_by,
        "limit": "100",
    }
)
list_status, listed, list_body = http_json("GET", f"{base_url}/api/jobs?{query}")
listed_items = listed if isinstance(listed, list) else []
listed_types = [str(item.get("type", "")) for item in listed_items if isinstance(item, dict)]

if not accepted:
    status = "blocked"
elif rejected:
    status = "incomplete"
else:
    status = "ok"

summary = {
    "schema_version": "orquesta_opes_derivatives_rest_contract_probe.v0",
    "created_at_unix": int(time.time()),
    "status": status,
    "transport_compat": transport_compat,
    "sequence_count": len(sequence),
    "accepted_count": len(accepted),
    "rejected_count": len(rejected),
    "accepted_job_types": [item["job_type"] for item in accepted],
    "accepted_transport_job_types": [item["transport_job_type"] for item in accepted],
    "rejected_job_types": [item["job_type"] for item in rejected],
    "rejected_transport_job_types": [item["transport_job_type"] for item in rejected],
    "program_id": program_id,
    "topic_id": topic_id,
    "correlation_id": correlation_id,
    "requested_by": requested_by,
    "responses": responses,
    "pending_scope_query": {
        "http_status": list_status,
        "count": len(listed_items),
        "job_types": listed_types,
        "body_excerpt": "" if isinstance(listed, list) else list_body[:300],
    },
}
os.makedirs(os.path.dirname(summary_file), exist_ok=True)
with open(summary_file, "w", encoding="utf-8") as handle:
    json.dump(summary, handle, indent=2, sort_keys=True)
    handle.write("\n")

print(f"contract_probe_status={status}")
print(f"transport_compat={str(transport_compat).lower()}")
print("accepted_job_types=" + ",".join(summary["accepted_job_types"]))
print("accepted_transport_job_types=" + ",".join(summary["accepted_transport_job_types"]))
print("rejected_job_types=" + ",".join(summary["rejected_job_types"]))
print("rejected_transport_job_types=" + ",".join(summary["rejected_transport_job_types"]))
print(f"accepted_count={len(accepted)}")
print(f"rejected_count={len(rejected)}")
print(f"summary_file={summary_file}")

if expect_full and rejected:
    raise SystemExit(1)
PY
