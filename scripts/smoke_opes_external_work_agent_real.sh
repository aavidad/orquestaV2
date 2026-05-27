#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=scripts/lib/opes_agent_smoke_payloads.sh
source "$repo_root/scripts/lib/opes_agent_smoke_payloads.sh"
# shellcheck source=scripts/lib/opes_agent_smoke_ops.sh
source "$repo_root/scripts/lib/opes_agent_smoke_ops.sh"

if [[ "${ORQUESTA_OPES_AGENT_SMOKE_CONFIRM:-0}" != "1" ]]; then
  echo "smoke real desactivado: exporta ORQUESTA_OPES_AGENT_SMOKE_CONFIRM=1" >&2
  exit 2
fi

OPES_BASE_URL="${OPES_BASE_URL:-http://127.0.0.1:18082}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
smoke_root_source="generated"
if [[ -n "${SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:SMOKE_ROOT"
fi
SMOKE_ROOT="${SMOKE_ROOT:-/tmp/orquesta-opes-agent-smoke/$SMOKE_ID}"
SMOKE_OUT_DIR="$SMOKE_ROOT/out"
STATE_DIR="$SMOKE_ROOT/state"
PROJECT_DIR="$SMOKE_ROOT/project"
RUNTIME_DIR="$PROJECT_DIR/.orquesta-runtime"
BIN_DIR="$SMOKE_ROOT/bin"
TIMEOUT_SECONDS="${ORQUESTA_OPES_AGENT_SMOKE_TIMEOUT_SECONDS:-900}"
POLL_SECONDS="${ORQUESTA_OPES_AGENT_SMOKE_POLL_SECONDS:-10}"
KEEP_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-1}"

server_pid=""
base_url=""

trap smoke_cleanup EXIT
trap 'exit 130' INT TERM

main() {
  smoke_require_commands
  smoke_temp_root_prepare "$SMOKE_ROOT" "$smoke_root_source"
  mkdir -p "$SMOKE_OUT_DIR"

  smoke_get_json "$OPES_BASE_URL/api/health" "$SMOKE_OUT_DIR/opes_health.json"
  smoke_write_project_context
  smoke_start_orquesta_server

  local requested_run_ref
  requested_run_ref="run-external-work-smoke-$(smoke_compact_ref "$SMOKE_ID")"
  echo "requested_run_ref=$requested_run_ref"

  local ids
  ids="$(smoke_create_opes_domain_objects)"
  local topic_id
  local chapter_id
  topic_id="$(jq -r '.topic_id' <<<"$ids")"
  chapter_id="$(jq -r '.chapter_id' <<<"$ids")"
  echo "topic_id=$topic_id chapter_id=$chapter_id"

  local job_ref
  job_ref="$(smoke_create_opes_external_job "$requested_run_ref" "$topic_id" "$chapter_id")"
  echo "job_ref=$job_ref"

  local run_ref
  run_ref="$(smoke_request_orquesta_external_work_run "$requested_run_ref" "$topic_id" "$chapter_id" "$job_ref")"
  echo "run_ref=$run_ref"
  smoke_poll_until_artifact "$run_ref" "$job_ref" "$topic_id"
  smoke_write_summary "$run_ref" "$topic_id" "$chapter_id" "$job_ref"
}

cd "$repo_root"
main "$@"
