#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

smoke_require_confirm ORQUESTA_OPES_DOMAIN_SMOKE_CONFIRM 1 \
  "smoke domain_work real desactivado: exporta ORQUESTA_OPES_DOMAIN_SMOKE_CONFIRM=1"

ORQUESTA_BASE_URL="$(smoke_require_orquesta_base_url ORQUESTA_SERVER_URL)"
ORQUESTA_OPES_BASE_URL="${ORQUESTA_OPES_BASE_URL:-http://127.0.0.1:18082}"
smoke_require_opes_temporal_destination "$ORQUESTA_OPES_BASE_URL" "ORQUESTA_OPES_BASE_URL"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/orquesta-opes-smoke-results/$SMOKE_ID}"

domain_work_create_payload() {
  local topic_id="$1"
  local chapter_id="$2"
  local request_id="request-ref-opes-smoke-${SMOKE_ID}-job"
  local correlation_id="corr-opes-smoke-${SMOKE_ID}"
  local idempotency_key="idem-opes-smoke-${SMOKE_ID}-job"
  local run_ref="run-ref-opes-smoke-${SMOKE_ID}"
  local task_ref="task-ref-opes-smoke-${SMOKE_ID}"
  jq -n \
    --arg request_id "$request_id" \
    --arg correlation_id "$correlation_id" \
    --arg idempotency_key "$idempotency_key" \
    --arg run_ref "$run_ref" \
    --arg task_ref "$task_ref" \
    --arg topic_id "$topic_id" \
    --arg chapter_id "$chapter_id" \
    '{
      request_id: $request_id,
      correlation_id: $correlation_id,
      action: "create_job",
      job_request: {
        schema_version: "domain_work_job_request.v0",
        request_id: $request_id,
        correlation_id: $correlation_id,
        idempotency_key: $idempotency_key,
        requested_by: "orquesta",
        domain_ref: "opes",
        interface_refs: ["opes-rest-v0"],
        work_kind: "draft_content_block",
        objective: "Crear un bloque documental real para smoke Orquesta OPES",
        input_fields: [
          {name: "program_id", value: "program-ref-smoke"},
          {name: "topic_id", value: $topic_id},
          {name: "chapter_id", value: $chapter_id},
          {name: "block_type", value: "technical"},
          {name: "language_code", value: "es"}
        ],
        acceptance_criteria: [
          "contenido en markdown",
          "estructura clara"
        ],
        external_refs: [
          {kind: "run_ref", ref: $run_ref},
          {kind: "task_ref", ref: $task_ref}
        ]
      }
    }'
}

domain_work_artifact_payload() {
  local job_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local request_id="request-ref-opes-smoke-${SMOKE_ID}-artifact"
  local correlation_id="corr-opes-smoke-${SMOKE_ID}"
  local idempotency_key="idem-opes-smoke-${SMOKE_ID}-artifact"
  local artifact_ref="artifact-ref-opes-smoke-${SMOKE_ID}"
  local run_ref="run-ref-opes-smoke-${SMOKE_ID}"
  local task_ref="task-ref-opes-smoke-${SMOKE_ID}"
  local delivery_ref="delivery-ref-opes-smoke-${SMOKE_ID}"
  jq -n \
    --arg request_id "$request_id" \
    --arg correlation_id "$correlation_id" \
    --arg idempotency_key "$idempotency_key" \
    --arg artifact_ref "$artifact_ref" \
    --arg run_ref "$run_ref" \
    --arg task_ref "$task_ref" \
    --arg delivery_ref "$delivery_ref" \
    --arg job_ref "$job_ref" \
    --arg topic_id "$topic_id" \
    --arg chapter_id "$chapter_id" \
    '{
      request_id: $request_id,
      correlation_id: $correlation_id,
      action: "submit_artifact",
      artifact_submission: {
        schema_version: "domain_work_artifact_submission.v0",
        request_id: $request_id,
        correlation_id: $correlation_id,
        idempotency_key: $idempotency_key,
        requested_by: "orquesta",
        domain_ref: "opes",
        job_ref: $job_ref,
        artifact_ref: $artifact_ref,
        artifact_type: "content_block",
        summary: "Bloque documental real de smoke generado por Orquesta",
        payload_fields: [
          {name: "topic_id", value: $topic_id},
          {name: "chapter_id", value: $chapter_id},
          {name: "block_type", value: "technical"},
          {name: "title", value: "Bloque de prueba Orquesta"},
          {name: "body", value: "# Bloque de prueba Orquesta\n\nContenido de prueba en markdown producido mediante el puente Orquesta -> OPES."},
          {name: "content_type", value: "text/markdown"},
          {name: "language_code", value: "es"}
        ],
        external_refs: [
          {kind: "run_ref", ref: $run_ref},
          {kind: "task_ref", ref: $task_ref},
          {kind: "delivery_ref", ref: $delivery_ref}
        ],
        complete_job: true
      }
    }'
}

url_ref() {
  local value="${1%/}"
  local digest=""
  if command -v sha256sum >/dev/null 2>&1; then
    digest="$(printf '%s' "$value" | sha256sum | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    digest="$(printf '%s' "$value" | shasum -a 256 | awk '{print $1}')"
  else
    echo "url-ref-unavailable"
    return 0
  fi
  echo "url-ref-${digest}"
}

write_summary() {
  local topic_id="$1"
  local chapter_id="$2"
  local job_ref="$3"
  local receipt_ref="$4"
  local block_id="$5"
  local block_count="$6"
  cat >"$SMOKE_OUT_DIR/summary.txt" <<EOF
smoke_id=$SMOKE_ID
orquesta_base_url_ref=$(url_ref "$ORQUESTA_BASE_URL")
opes_base_url_ref=$(url_ref "$ORQUESTA_OPES_BASE_URL")
topic_id=$topic_id
chapter_id=$chapter_id
job_ref=$job_ref
receipt_ref=$receipt_ref
block_id=$block_id
block_count_after_replay=$block_count
output_dir=$SMOKE_OUT_DIR
EOF
}

main() {
  smoke_require_tools curl jq
  mkdir -p "$SMOKE_OUT_DIR"

  smoke_get_json "$ORQUESTA_BASE_URL/api/v0/server/readiness" "$SMOKE_OUT_DIR/orquesta_readiness.json"
  smoke_get_json "$ORQUESTA_OPES_BASE_URL/api/health" "$SMOKE_OUT_DIR/opes_health.json"

  smoke_post_json "$ORQUESTA_OPES_BASE_URL/api/topics" \
    '{"title":"Smoke Orquesta OPES","subject_area":"psicologia"}' \
    "$SMOKE_OUT_DIR/topic.json"
  local topic_id
  topic_id="$(smoke_json_id "$SMOKE_OUT_DIR/topic.json")"
  if [[ -z "$topic_id" ]]; then
    echo "OPES no devolvio topic id" >&2
    exit 1
  fi

  smoke_post_json "$ORQUESTA_OPES_BASE_URL/api/topics/$topic_id/chapters" \
    '{"title":"Capitulo Smoke","order":1}' \
    "$SMOKE_OUT_DIR/chapter.json"
  local chapter_id
  chapter_id="$(smoke_json_id "$SMOKE_OUT_DIR/chapter.json")"
  if [[ -z "$chapter_id" ]]; then
    echo "OPES no devolvio chapter id" >&2
    exit 1
  fi

  local create_payload
  create_payload="$(domain_work_create_payload "$topic_id" "$chapter_id")"
  printf '%s\n' "$create_payload" >"$SMOKE_OUT_DIR/create_job_request.json"
  smoke_post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$create_payload" \
    "$SMOKE_OUT_DIR/create_job_response.json"
  local job_ref
  job_ref="$(jq -r '.job.job_ref // empty' "$SMOKE_OUT_DIR/create_job_response.json")"
  if [[ -z "$job_ref" ]]; then
    echo "Orquesta no devolvio job_ref" >&2
    exit 1
  fi

  local artifact_payload
  artifact_payload="$(domain_work_artifact_payload "$job_ref" "$topic_id" "$chapter_id")"
  printf '%s\n' "$artifact_payload" >"$SMOKE_OUT_DIR/submit_artifact_request.json"
  smoke_post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$artifact_payload" \
    "$SMOKE_OUT_DIR/submit_artifact_response.json"
  local receipt_ref
  receipt_ref="$(jq -r '.receipt.receipt_ref // empty' "$SMOKE_OUT_DIR/submit_artifact_response.json")"
  if [[ -z "$receipt_ref" ]]; then
    echo "Orquesta no devolvio receipt_ref" >&2
    exit 1
  fi

  smoke_get_json "$ORQUESTA_OPES_BASE_URL/api/topics/$topic_id/blocks" "$SMOKE_OUT_DIR/blocks.json"
  local block_count
  block_count="$(jq 'length' "$SMOKE_OUT_DIR/blocks.json")"
  if [[ "$block_count" != "1" ]]; then
    echo "bloques materializados inesperados antes de replay: $block_count" >&2
    exit 1
  fi

  smoke_post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$create_payload" \
    "$SMOKE_OUT_DIR/create_job_replay_response.json"
  smoke_post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$artifact_payload" \
    "$SMOKE_OUT_DIR/submit_artifact_replay_response.json"
  smoke_get_json "$ORQUESTA_OPES_BASE_URL/api/topics/$topic_id/blocks" "$SMOKE_OUT_DIR/blocks_after_replay.json"

  local block_count_after_replay
  block_count_after_replay="$(jq 'length' "$SMOKE_OUT_DIR/blocks_after_replay.json")"
  if [[ "$block_count_after_replay" != "1" ]]; then
    echo "idempotencia rota: bloques tras replay=$block_count_after_replay" >&2
    exit 1
  fi

  local block_id
  block_id="$(jq -r '.[0].ID // .[0].id // empty' "$SMOKE_OUT_DIR/blocks_after_replay.json")"
  write_summary "$topic_id" "$chapter_id" "$job_ref" "$receipt_ref" "$block_id" "$block_count_after_replay"
  cat "$SMOKE_OUT_DIR/summary.txt"
}

main "$@"
