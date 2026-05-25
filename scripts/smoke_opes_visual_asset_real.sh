#!/usr/bin/env bash
set -euo pipefail

if [[ "${ORQUESTA_OPES_VISUAL_SMOKE_CONFIRM:-0}" != "1" ]]; then
  echo "smoke visual real desactivado: exporta ORQUESTA_OPES_VISUAL_SMOKE_CONFIRM=1" >&2
  exit 2
fi

ORQUESTA_BASE_URL="${ORQUESTA_BASE_URL:-http://127.0.0.1:18787}"
OPES_BASE_URL="${OPES_BASE_URL:-http://127.0.0.1:18082}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/orquesta-opes-visual-smoke/$SMOKE_ID/out}"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta herramienta requerida: $1" >&2
    exit 2
  fi
}

json_id() {
  jq -r '.id // .ID // empty' "$1"
}

post_json() {
  local url="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(curl -sS -o "$output" -w "%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "$payload")"
  if [[ "$status" -lt 200 || "$status" -gt 299 ]]; then
    echo "POST $url devolvio HTTP $status" >&2
    cat "$output" >&2 || true
    exit 1
  fi
}

get_json() {
  local url="$1"
  local output="$2"
  curl -sS -f "$url" >"$output"
}

json_count() {
  jq 'if type == "array" then length else (.artifacts // .blocks // [] | length) end' \
    "$1"
}

create_job_payload() {
  local topic_id="$1"
  local chapter_id="$2"
  local request_id="request-ref-visual-smoke-${SMOKE_ID}-job"
  local correlation_id="corr-visual-smoke-${SMOKE_ID}"
  local idempotency_key="idem-visual-smoke-${SMOKE_ID}-job"
  local run_ref="run-ref-visual-smoke-${SMOKE_ID}"
  jq -n \
    --arg request_id "$request_id" \
    --arg correlation_id "$correlation_id" \
    --arg idempotency_key "$idempotency_key" \
    --arg run_ref "$run_ref" \
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
        requested_by: "orquesta-visual-smoke",
        domain_ref: "opes",
        interface_refs: ["opes-rest-v0"],
        work_kind: "generate_visual_asset",
        objective: "Crear un recurso visual SVG de prueba para un temario OPES",
        input_fields: [
          {name: "topic_id", value: $topic_id},
          {name: "chapter_id", value: $chapter_id},
          {name: "asset_type", value: "diagram"},
          {name: "format", value: "svg"},
          {name: "title", value: "Red en estrella"},
          {name: "caption", value: "Topologia con un nodo central."},
          {name: "alt_text", value: "Switch central conectado a cuatro equipos cliente."},
          {name: "placement", value: "after_block"},
          {name: "language_code", value: "es"}
        ],
        acceptance_criteria: [
          "visual_asset completo",
          "svg autocontenido",
          "sin scripts ni placeholders"
        ],
        external_refs: [
          {kind: "run_ref", ref: $run_ref},
          {kind: "smoke_id", ref: $request_id}
        ]
      }
    }'
}

visual_svg() {
  cat <<'SVG'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420" role="img" aria-label="Red en estrella">
  <rect width="800" height="420" fill="#ffffff"/>
  <circle cx="400" cy="210" r="42" fill="#e8f1ff" stroke="#1f4e79" stroke-width="4"/>
  <text x="400" y="216" text-anchor="middle" font-family="Arial" font-size="22" fill="#17324d">Switch</text>
  <g stroke="#1f4e79" stroke-width="3">
    <line x1="400" y1="168" x2="180" y2="90"/>
    <line x1="442" y1="210" x2="620" y2="110"/>
    <line x1="400" y1="252" x2="185" y2="330"/>
    <line x1="358" y1="210" x2="620" y2="315"/>
  </g>
  <g fill="#f7fafc" stroke="#4a5568" stroke-width="3">
    <rect x="115" y="55" width="130" height="70" rx="6"/>
    <rect x="555" y="75" width="130" height="70" rx="6"/>
    <rect x="120" y="295" width="130" height="70" rx="6"/>
    <rect x="555" y="280" width="130" height="70" rx="6"/>
  </g>
  <g font-family="Arial" font-size="20" fill="#1a202c" text-anchor="middle">
    <text x="180" y="97">Equipo 1</text>
    <text x="620" y="117">Equipo 2</text>
    <text x="185" y="337">Equipo 3</text>
    <text x="620" y="322">Equipo 4</text>
  </g>
  <text x="400" y="388" text-anchor="middle" font-family="Arial" font-size="18" fill="#2d3748">Todos los equipos dependen del nodo central.</text>
</svg>
SVG
}

artifact_payload() {
  local job_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local svg="$4"
  local request_id="request-ref-visual-smoke-${SMOKE_ID}-artifact"
  local correlation_id="corr-visual-smoke-${SMOKE_ID}"
  local idempotency_key="idem-visual-smoke-${SMOKE_ID}-artifact"
  local artifact_ref="artifact-ref-visual-smoke-${SMOKE_ID}"
  local run_ref="run-ref-visual-smoke-${SMOKE_ID}"
  jq -n \
    --arg request_id "$request_id" \
    --arg correlation_id "$correlation_id" \
    --arg idempotency_key "$idempotency_key" \
    --arg artifact_ref "$artifact_ref" \
    --arg job_ref "$job_ref" \
    --arg topic_id "$topic_id" \
    --arg chapter_id "$chapter_id" \
    --arg run_ref "$run_ref" \
    --arg svg "$svg" \
    '{
      request_id: $request_id,
      correlation_id: $correlation_id,
      action: "submit_artifact",
      artifact_submission: {
        schema_version: "domain_work_artifact_submission.v0",
        request_id: $request_id,
        correlation_id: $correlation_id,
        idempotency_key: $idempotency_key,
        requested_by: "orquesta-visual-smoke",
        domain_ref: "opes",
        job_ref: $job_ref,
        artifact_ref: $artifact_ref,
        artifact_type: "visual_asset",
        summary: "Recurso visual SVG de red en estrella",
        payload_fields: [
          {name: "topic_id", value: $topic_id},
          {name: "chapter_id", value: $chapter_id},
          {name: "asset_type", value: "diagram"},
          {name: "format", value: "svg"},
          {name: "title", value: "Red en estrella"},
          {name: "caption", value: "Topologia con un nodo central."},
          {name: "alt_text", value: "Switch central conectado a cuatro equipos cliente."},
          {name: "body", value: $svg},
          {name: "placement", value: "after_block"},
          {name: "language_code", value: "es"},
          {name: "source_refs", values: ["material-redes-smoke"]},
          {name: "content_type", value: "image/svg+xml"}
        ],
        external_refs: [
          {kind: "run_ref", ref: $run_ref},
          {kind: "delivery_ref", ref: $artifact_ref}
        ],
        complete_job: true
      }
    }'
}

assert_visual_block() {
  local blocks_file="$1"
  if [[ "$(json_count "$blocks_file")" != "1" ]]; then
    echo "OPES no materializo exactamente un bloque visual" >&2
    cat "$blocks_file" >&2 || true
    exit 1
  fi
  local block_type
  block_type="$(jq -r 'if type == "array" then (.[0].type // .[0].Type) else (.blocks[0].type // .blocks[0].Type) end' "$blocks_file")"
  if [[ "$block_type" != "visual_asset" ]]; then
    echo "bloque materializado con tipo inesperado: $block_type" >&2
    exit 1
  fi
}

write_summary() {
  local topic_id="$1"
  local chapter_id="$2"
  local job_ref="$3"
  local receipt_ref="$4"
  local artifacts_count="$5"
  local blocks_count="$6"
  cat >"$SMOKE_OUT_DIR/summary.txt" <<EOF
smoke_id=$SMOKE_ID
orquesta_base_url=$ORQUESTA_BASE_URL
opes_base_url=$OPES_BASE_URL
topic_id=$topic_id
chapter_id=$chapter_id
job_ref=$job_ref
receipt_ref=$receipt_ref
artifacts_count=$artifacts_count
blocks_count=$blocks_count
output_dir=$SMOKE_OUT_DIR
EOF
}

main() {
  require_tool curl
  require_tool jq
  mkdir -p "$SMOKE_OUT_DIR"

  get_json "$ORQUESTA_BASE_URL/api/v0/server/readiness" "$SMOKE_OUT_DIR/orquesta_readiness.json"
  get_json "$OPES_BASE_URL/api/health" "$SMOKE_OUT_DIR/opes_health.json"

  post_json "$OPES_BASE_URL/api/topics" \
    '{"title":"Smoke visual Orquesta OPES","subject_area":"redes"}' \
    "$SMOKE_OUT_DIR/topic.json"
  local topic_id
  topic_id="$(json_id "$SMOKE_OUT_DIR/topic.json")"
  if [[ -z "$topic_id" ]]; then
    echo "OPES no devolvio topic id" >&2
    exit 1
  fi

  post_json "$OPES_BASE_URL/api/topics/$topic_id/chapters" \
    '{"title":"Topologias de red","order":1}' \
    "$SMOKE_OUT_DIR/chapter.json"
  local chapter_id
  chapter_id="$(json_id "$SMOKE_OUT_DIR/chapter.json")"
  if [[ -z "$chapter_id" ]]; then
    echo "OPES no devolvio chapter id" >&2
    exit 1
  fi

  local create_payload
  create_payload="$(create_job_payload "$topic_id" "$chapter_id")"
  printf '%s\n' "$create_payload" >"$SMOKE_OUT_DIR/create_job_request.json"
  post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$create_payload" "$SMOKE_OUT_DIR/create_job_response.json"
  local job_ref
  job_ref="$(jq -r '.job.job_ref // empty' "$SMOKE_OUT_DIR/create_job_response.json")"
  if [[ -z "$job_ref" ]]; then
    echo "Orquesta no devolvio job_ref" >&2
    cat "$SMOKE_OUT_DIR/create_job_response.json" >&2 || true
    exit 1
  fi

  local svg
  svg="$(visual_svg)"
  local submit_payload
  submit_payload="$(artifact_payload "$job_ref" "$topic_id" "$chapter_id" "$svg")"
  printf '%s\n' "$submit_payload" >"$SMOKE_OUT_DIR/submit_artifact_request.json"
  post_json "$ORQUESTA_BASE_URL/api/v0/domain-work" \
    "$submit_payload" "$SMOKE_OUT_DIR/submit_artifact_response.json"
  local receipt_ref
  receipt_ref="$(jq -r '.receipt.receipt_ref // empty' "$SMOKE_OUT_DIR/submit_artifact_response.json")"
  if [[ -z "$receipt_ref" ]]; then
    echo "Orquesta no devolvio receipt_ref" >&2
    cat "$SMOKE_OUT_DIR/submit_artifact_response.json" >&2 || true
    exit 1
  fi

  get_json "$OPES_BASE_URL/api/jobs/$job_ref/artifacts" "$SMOKE_OUT_DIR/artifacts.json"
  get_json "$OPES_BASE_URL/api/topics/$topic_id/blocks" "$SMOKE_OUT_DIR/blocks.json"
  assert_visual_block "$SMOKE_OUT_DIR/blocks.json"

  local artifacts_count
  local blocks_count
  artifacts_count="$(json_count "$SMOKE_OUT_DIR/artifacts.json")"
  blocks_count="$(json_count "$SMOKE_OUT_DIR/blocks.json")"
  write_summary "$topic_id" "$chapter_id" "$job_ref" "$receipt_ref" "$artifacts_count" "$blocks_count"
  cat "$SMOKE_OUT_DIR/summary.txt"
}

main "$@"
