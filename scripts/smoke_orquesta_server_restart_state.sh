#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_id="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"
smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
work_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-restart-state.XXXXXX")}"
smoke_temp_root_prepare "$work_root" "$smoke_root_source"
state_dir="$work_root/state"
project_dir="$work_root/project"
runtime_dir="$project_dir/.orquesta-runtime"
bin_dir="$work_root/bin"
server_pid=""
base_url=""
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
request_timeout="${ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS:-10}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

cleanup() {
  stop_server || true
  smoke_temp_root_cleanup "$work_root" "$keep_dir"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

state_field() {
  local field="$1"
  python3 - "$state_dir/orquesta_server_state_v0.json" "$field" <<'PY'
import json
import sys

path = sys.argv[1]
field = sys.argv[2]
try:
    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)
except FileNotFoundError:
    print("")
    raise SystemExit(0)
print(data.get(field, ""))
PY
}

wait_server_ready() {
  local expected_pid="$1"
  local addr=""
  for _ in $(seq 1 60); do
    addr="$(state_field addr)"
    local pid
    pid="$(state_field pid)"
    if [[ "$pid" == "$expected_pid" && -n "$addr" ]] &&
      curl -fsS -m 2 "http://$addr/api/v0/server/readiness" >/dev/null; then
      base_url="http://$addr"
      return 0
    fi
    sleep 0.25
  done
  return 1
}

start_server() {
  local label="$1"
  local stdout_log="$work_root/server-$label.stdout.log"
  local stderr_log="$work_root/server-$label.stderr.log"

  ORQUESTA_SERVER_ADDR="127.0.0.1:0" \
  ORQUESTA_SERVER_STATE_DIR="$state_dir" \
  ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir" \
  ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir" \
  ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-60000}" \
  ORQUESTA_OPES_BASE_URL="" \
    "$bin_dir/orquesta-server" run >"$stdout_log" 2>"$stderr_log" &
  server_pid="$!"

  if ! wait_server_ready "$server_pid"; then
    echo "orquesta-server temporal no llego a readiness OK ($label)" >&2
    tail -n 80 "$stderr_log" >&2 || true
    exit 1
  fi
  echo "servidor $label listo: $base_url pid=$server_pid"
}

stop_server() {
  if [[ -z "$server_pid" ]]; then
    smoke_cleanup_codex_app_server_tmux_runtime "$runtime_dir"
    return 0
  fi
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"
  server_pid=""
}

write_set_priority_payload() {
  local output="$1"
  local run_ref="$2"
  local app_ref="$3"
  local priority="$4"
  python3 - "$output" "$smoke_id" "$run_ref" "$app_ref" "$priority" <<'PY'
import json
import sys
from datetime import datetime, timezone

output, smoke_id, run_ref, app_ref, priority = sys.argv[1:6]
payload = {
    "request_id": f"request-ref-restart-state-{smoke_id}-set-{priority}",
    "correlation_id": f"corr-restart-state-{smoke_id}-set-{priority}",
    "action": "set_priority",
    "queue_ref": "global",
    "run_ref": run_ref,
    "app_ref": app_ref,
    "priority_score": int(priority),
    "requested_by": "orquesta-smoke",
    "reason": "smoke temporal de continuidad de estado tras reinicio",
    "idempotency_key": f"idem-restart-state-{smoke_id}-set-{priority}",
    "evidence_refs": [f"evidence-ref-restart-state-{smoke_id}"],
    "occurred_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_rank_payload() {
  local output="$1"
  python3 - "$output" "$smoke_id" <<'PY'
import json
import sys
from datetime import datetime, timezone

output, smoke_id = sys.argv[1:3]
payload = {
    "request_id": f"request-ref-restart-state-{smoke_id}-rank",
    "correlation_id": f"corr-restart-state-{smoke_id}-rank",
    "action": "rank",
    "queue_ref": "global",
    "limit": 10,
    "occurred_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

post_json() {
  local path="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(
    curl -sS -m "$request_timeout" -o "$output" -w "%{http_code}" \
      -X POST "$base_url$path" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary "@$payload"
  )"
  echo "POST $path -> HTTP $status"
  if [[ "$status" -lt 200 || "$status" -gt 299 ]]; then
    cat "$output" >&2 || true
    exit 1
  fi
}

verify_rank_contains() {
  local response="$1"
  local run_ref="$2"
  local priority="$3"
  python3 - "$response" "$run_ref" "$priority" <<'PY'
import json
import sys

path, run_ref, priority = sys.argv[1:4]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("estado") != "ok":
    raise SystemExit(f"estado inesperado: {data.get('estado')}")
ranked = data.get("ranked") or []
for item in ranked:
    if item.get("run_ref") == run_ref and int(item.get("priority_score") or 0) == int(priority):
        print(f"rank_verificado={run_ref} priority={priority}")
        raise SystemExit(0)
raise SystemExit(f"no aparece {run_ref} con priority={priority}: {ranked}")
PY
}

main() {
  need_cmd curl
  need_cmd go
  need_cmd python3
  mkdir -p "$state_dir" "$project_dir" "$runtime_dir" "$bin_dir"

  cd "$repo_root"
  echo "compilando servidor temporal..."
  go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server

  local run_ref="run-ref-restart-state-$smoke_id"
  local app_ref="app-ref-restart-state-$smoke_id"
  local set_payload="$work_root/set_priority.json"
  local rank_payload="$work_root/rank.json"
  local set_response="$work_root/set_priority_response.json"
  local rank_before="$work_root/rank_before_restart.json"
  local rank_after="$work_root/rank_after_restart.json"
  local rank_updated="$work_root/rank_updated_after_restart.json"

  start_server "before"
  write_set_priority_payload "$set_payload" "$run_ref" "$app_ref" 80
  post_json "/api/v0/runs/queue/priority" "$set_payload" "$set_response"
  write_rank_payload "$rank_payload"
  post_json "/api/v0/runs/queue/priority" "$rank_payload" "$rank_before"
  verify_rank_contains "$rank_before" "$run_ref" 80
  stop_server

  start_server "after"
  post_json "/api/v0/runs/queue/priority" "$rank_payload" "$rank_after"
  verify_rank_contains "$rank_after" "$run_ref" 80

  write_set_priority_payload "$set_payload" "$run_ref" "$app_ref" 95
  post_json "/api/v0/runs/queue/priority" "$set_payload" "$set_response"
  post_json "/api/v0/runs/queue/priority" "$rank_payload" "$rank_updated"
  verify_rank_contains "$rank_updated" "$run_ref" 95

  cat >"$work_root/summary.txt" <<EOF
smoke_id=$smoke_id
state_dir=$state_dir
project_dir=$project_dir
run_ref=$run_ref
validated=run_queue_persists_after_restart
opes_touched=false
agents_launched=false
EOF
  cat "$work_root/summary.txt"
}

main "$@"
