#!/usr/bin/env bash

smoke_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$smoke_lib_dir/smoke_common.sh"

smoke_require_commands() {
  smoke_require_tools curl go jq
}

smoke_compact_ref() {
  printf '%s' "$1" | sed -E 's#[[:space:]/\\]+#-#g; s/^-+//; s/-+$//'
}

smoke_cleanup() {
  if [[ -n "$base_url" ]]; then
    curl -sS -m 5 -X POST "$base_url/api/v0/server/shutdown" \
      -H "Content-Type: application/json" \
      -d '{"request_id":"req-smoke-shutdown","correlation_id":"corr-smoke-shutdown","forced":true}' \
      >/dev/null 2>&1 || true
  fi
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -INT "$server_pid" >/dev/null 2>&1 || true
    for _ in $(seq 1 30); do
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
  if [[ "$KEEP_DIR" == "1" ]]; then
    smoke_temp_root_cleanup "$SMOKE_ROOT" 1
  else
    smoke_temp_root_cleanup "$SMOKE_ROOT" 0
  fi
}

smoke_start_orquesta_server() {
  mkdir -p "$STATE_DIR" "$BIN_DIR"
  echo "compilando orquesta-server..."
  go build -o "$BIN_DIR/orquesta-server" ./cmd/orquesta-server

  export ORQUESTA_SERVER_ADDR="127.0.0.1:0"
  export ORQUESTA_SERVER_STATE_DIR="$STATE_DIR"
  export ORQUESTA_CODEX_PROJECT_WORKDIR="$PROJECT_DIR"
  export ORQUESTA_CODEX_RUNTIME_WORKDIR="$RUNTIME_DIR"
  export ORQUESTA_CODEX_GOAL_BACKEND="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL"
  export ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-1000}"
  export ORQUESTA_SERVER_MAX_RUNS_PER_TICK="${ORQUESTA_SERVER_MAX_RUNS_PER_TICK:-1}"
  export ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK="${ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK:-2}"
  export ORQUESTA_CODEX_MAX_BATCH_READY="${ORQUESTA_CODEX_MAX_BATCH_READY:-2}"
  export ORQUESTA_CODEX_MAX_CONCURRENCY="${ORQUESTA_CODEX_MAX_CONCURRENCY:-2}"
  export ORQUESTA_CODEX_WAIT_INTERVAL_MS="${ORQUESTA_CODEX_WAIT_INTERVAL_MS:-1000}"
  export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
  export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-workspace-write}"

  "$BIN_DIR/orquesta-server" run \
    >"$SMOKE_OUT_DIR/orquesta-server.stdout.log" \
    2>"$SMOKE_OUT_DIR/orquesta-server.stderr.log" &
  server_pid="$!"

  local state_file="$STATE_DIR/orquesta_server_state_v0.json"
  if smoke_wait_orquesta_readiness_from_state_file "$state_file" 80 0.5 base_url; then
    echo "orquesta-server listo: $base_url"
    return
  fi
  echo "orquesta-server no llego a readiness OK" >&2
  tail -n 80 "$SMOKE_OUT_DIR/orquesta-server.stderr.log" >&2 || true
  exit 1
}

smoke_create_opes_domain_objects() {
  smoke_post_json "$OPES_BASE_URL/api/topics" \
    '{"title":"Smoke Orquesta Agente Real","subject_area":"juridico"}' \
    "$SMOKE_OUT_DIR/opes_topic.json"
  local topic_id
  topic_id="$(smoke_json_id "$SMOKE_OUT_DIR/opes_topic.json")"
  if [[ -z "$topic_id" ]]; then
    echo "OPES no devolvio topic id" >&2
    exit 1
  fi

  smoke_post_json "$OPES_BASE_URL/api/topics/$topic_id/chapters" \
    '{"title":"Capitulo de integracion","order":1}' \
    "$SMOKE_OUT_DIR/opes_chapter.json"
  local chapter_id
  chapter_id="$(smoke_json_id "$SMOKE_OUT_DIR/opes_chapter.json")"
  if [[ -z "$chapter_id" ]]; then
    echo "OPES no devolvio chapter id" >&2
    exit 1
  fi
  jq -n --arg topic_id "$topic_id" --arg chapter_id "$chapter_id" \
    '{topic_id:$topic_id, chapter_id:$chapter_id}'
}

smoke_create_opes_external_job() {
  local run_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local idem="idem-opes-agent-smoke-$SMOKE_ID-job"
  smoke_write_opes_job_request "$run_ref" "$topic_id" "$chapter_id" "$idem" \
    "$SMOKE_OUT_DIR/opes_job_request.json"
  smoke_post_json "$OPES_BASE_URL/api/jobs" \
    "$(cat "$SMOKE_OUT_DIR/opes_job_request.json")" \
    "$SMOKE_OUT_DIR/opes_job_response.json"

  local job_ref
  job_ref="$(jq -r '.id // .job.id // empty' "$SMOKE_OUT_DIR/opes_job_response.json")"
  if [[ -z "$job_ref" ]]; then
    echo "OPES no devolvio job id" >&2
    cat "$SMOKE_OUT_DIR/opes_job_response.json" >&2
    exit 1
  fi
  printf '%s' "$job_ref"
}

smoke_request_orquesta_external_work_run() {
  local run_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local job_ref="$4"
  smoke_write_external_work_run_request "$run_ref" "$topic_id" "$chapter_id" \
    "$job_ref" "$SMOKE_OUT_DIR/external_work_run_request.json"
  smoke_post_json "$base_url/api/v0/external-work/run" \
    "$(cat "$SMOKE_OUT_DIR/external_work_run_request.json")" \
    "$SMOKE_OUT_DIR/external_work_run_response.json"
  local canonical_run_ref
  canonical_run_ref="$(jq -r '.run_ref // empty' "$SMOKE_OUT_DIR/external_work_run_response.json")"
  if [[ -z "$canonical_run_ref" ]]; then
    echo "Orquesta no devolvio run_ref en external_work_run" >&2
    cat "$SMOKE_OUT_DIR/external_work_run_response.json" >&2
    exit 1
  fi
  printf '%s' "$canonical_run_ref"
}

smoke_poll_until_artifact() {
  local run_ref="$1"
  local job_ref="$2"
  local topic_id="$3"
  local deadline
  deadline=$((SECONDS + TIMEOUT_SECONDS))
  local poll=0
  while (( SECONDS < deadline )); do
    poll=$((poll + 1))
    smoke_write_stats_request "$run_ref" "$job_ref" "$SMOKE_OUT_DIR/stats_request_$poll.json"
    curl -sS -m 20 -X POST "$base_url/api/v0/director/stats" \
      -H "Content-Type: application/json" \
      --data-binary "@$SMOKE_OUT_DIR/stats_request_$poll.json" \
      >"$SMOKE_OUT_DIR/stats_response_$poll.json" || true

    curl -sS -m 20 "$OPES_BASE_URL/api/jobs/$job_ref/artifacts" \
      >"$SMOKE_OUT_DIR/opes_artifacts_$poll.json" || true
    curl -sS -m 20 "$OPES_BASE_URL/api/topics/$topic_id/blocks" \
      >"$SMOKE_OUT_DIR/opes_blocks_$poll.json" || true

    local artifacts_count
    artifacts_count="$(smoke_json_count "$SMOKE_OUT_DIR/opes_artifacts_$poll.json")"
    local blocks_count
    blocks_count="$(smoke_json_count "$SMOKE_OUT_DIR/opes_blocks_$poll.json")"
    echo "poll=$poll artifacts=$artifacts_count blocks=$blocks_count"
    if [[ "$artifacts_count" != "0" || "$blocks_count" != "0" ]]; then
      cp "$SMOKE_OUT_DIR/stats_response_$poll.json" "$SMOKE_OUT_DIR/stats_response_final.json"
      cp "$SMOKE_OUT_DIR/opes_artifacts_$poll.json" "$SMOKE_OUT_DIR/opes_artifacts_final.json"
      cp "$SMOKE_OUT_DIR/opes_blocks_$poll.json" "$SMOKE_OUT_DIR/opes_blocks_final.json"
      smoke_assert_final_stats "$run_ref" "$job_ref"
      return
    fi
    sleep "$POLL_SECONDS"
  done
  echo "timeout esperando artefacto OPES" >&2
  exit 1
}

smoke_assert_final_stats() {
  local run_ref="$1"
  local job_ref="$2"
  local stats="$SMOKE_OUT_DIR/stats_response_final.json"
  if [[ "$(jq -r '.estado // empty' "$stats")" != "ok" ]]; then
    echo "stats finales no estan ok" >&2
    cat "$stats" >&2
    exit 1
  fi
  if [[ "$(jq -r '.run_ref // empty' "$stats")" != "$run_ref" ]]; then
    echo "stats finales no usan run_ref canonico" >&2
    cat "$stats" >&2
    exit 1
  fi
  if [[ "$(jq -r '.external_job.job_ref // empty' "$stats")" != "$job_ref" ]]; then
    echo "stats finales no exponen external_job" >&2
    cat "$stats" >&2
    exit 1
  fi
}

smoke_url_ref() {
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

smoke_write_summary() {
  local run_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local job_ref="$4"
  local artifacts_count
  local blocks_count
  artifacts_count="$(smoke_json_count "$SMOKE_OUT_DIR/opes_artifacts_final.json")"
  blocks_count="$(smoke_json_count "$SMOKE_OUT_DIR/opes_blocks_final.json")"
  cat >"$SMOKE_OUT_DIR/summary.txt" <<EOF
smoke_id=$SMOKE_ID
orquesta_base_url_ref=$(smoke_url_ref "$base_url")
opes_base_url_ref=$(smoke_url_ref "$OPES_BASE_URL")
run_ref=$run_ref
topic_id=$topic_id
chapter_id=$chapter_id
job_ref=$job_ref
artifacts_count=$artifacts_count
blocks_count=$blocks_count
output_dir=$SMOKE_OUT_DIR
EOF
  cat "$SMOKE_OUT_DIR/summary.txt"
}
