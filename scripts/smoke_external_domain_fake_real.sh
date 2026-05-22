#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_id="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
work_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-external-domain-fake.XXXXXX")}"
state_dir="$work_root/state"
project_dir="$work_root/project"
runtime_dir="$project_dir/.orquesta-runtime"
bin_dir="$work_root/bin"
out_dir="${SMOKE_OUT_DIR:-$work_root/out}"
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
  if [[ "$keep_dir" == "1" ]]; then
    echo "directorio conservado: $work_root" >&2
  else
    rm -rf "$work_root"
  fi
}

trap cleanup EXIT
trap 'exit 130' INT TERM

state_field() {
  local field="$1"
  python3 - "$state_dir/orquesta_server_state_v0.json" "$field" <<'PY'
import json
import sys

path, field = sys.argv[1:3]
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
  for _ in $(seq 1 80); do
    addr="$(state_field addr)"
    local pid
    pid="$(state_field pid)"
    if [[ "$pid" == "$expected_pid" && -n "$addr" ]] &&
      curl -fsS -m 2 "http://$addr/healthz" >/dev/null; then
      base_url="http://$addr"
      return 0
    fi
    sleep 0.25
  done
  return 1
}

start_server() {
  local stdout_log="$out_dir/server.stdout.log"
  local stderr_log="$out_dir/server.stderr.log"

  ORQUESTA_SERVER_ADDR="127.0.0.1:0" \
  ORQUESTA_SERVER_STATE_DIR="$state_dir" \
  ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir" \
  ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir" \
  ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-60000}" \
  ORQUESTA_STARTUP_CLEANUP_MODE="off" \
  ORQUESTA_OPES_BASE_URL="" \
  OPES_BASE_URL="" \
  ORQUESTA_DOMAIN_WORK_FILE_ENABLED="1" \
  ORQUESTA_CODEX_COMMAND="$bin_dir/codex-fake" \
    "$bin_dir/orquesta-server" run >"$stdout_log" 2>"$stderr_log" &
  server_pid="$!"

  if ! wait_server_ready "$server_pid"; then
    echo "orquesta-server temporal no llego a health OK" >&2
    tail -n 80 "$stderr_log" >&2 || true
    exit 1
  fi
  echo "servidor listo: $base_url pid=$server_pid"
}

stop_server() {
  if [[ -z "$server_pid" ]]; then
    return 0
  fi
  if kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -INT "$server_pid" >/dev/null 2>&1 || true
    for _ in $(seq 1 30); do
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        server_pid=""
        return 0
      fi
      sleep 0.2
    done
    kill -TERM "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  server_pid=""
}

post_json() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  curl -fsS -m "$request_timeout" \
    -H 'Content-Type: application/json' \
    -H "X-Correlation-ID: corr-external-domain-fake-$smoke_id" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output"
}

post_json_expect_status() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  local want_status="$4"
  local got_status
  got_status="$(curl -sS -m "$request_timeout" \
    -H 'Content-Type: application/json' \
    -H "X-Correlation-ID: corr-external-domain-fake-$smoke_id" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output" \
    -w '%{http_code}')"
  if [[ "$got_status" != "$want_status" ]]; then
    echo "HTTP inesperado para $url: got=$got_status want=$want_status" >&2
    cat "$output" >&2 || true
    exit 1
  fi
}

write_fake_codex() {
  mkdir -p "$bin_dir"
  cat >"$bin_dir/codex-fake" <<'SH'
#!/usr/bin/env sh
echo "codex fake no debe ejecutarse en smoke external domain" >&2
exit 42
SH
  chmod 700 "$bin_dir/codex-fake"
}

write_create_payload() {
  local output="$1"
  python3 - "$output" "$smoke_id" <<'PY'
import json
import sys

output, smoke_id = sys.argv[1:3]
payload = {
    "request_id": f"request-ref-external-domain-fake-{smoke_id}",
    "correlation_id": f"corr-external-domain-fake-{smoke_id}",
    "action": "create_job",
    "job_request": {
        "schema_version": "domain_work_job_request.v0",
        "request_id": f"request-ref-external-domain-fake-{smoke_id}",
        "correlation_id": f"corr-external-domain-fake-{smoke_id}",
        "idempotency_key": f"idem-external-domain-fake-{smoke_id}",
        "requested_by": "orquesta-smoke",
        "domain_ref": f"domain-ref-fake-{smoke_id}",
        "interface_refs": ["domain-work.v0"],
        "work_kind": "compose_external_summary",
        "work_refs": [f"work-ref-fake-{smoke_id}"],
        "objective": "Crear trabajo neutral para app externa fake usando solo refs opacas.",
        "input_fields": [
            {"name": "title", "value": "Smoke externo fake"},
            {"name": "scope_ref", "value": f"scope-ref-fake-{smoke_id}"}
        ],
        "input_refs": [f"source-ref-fake-{smoke_id}"],
        "constraints": ["sin OPES", "sin Codex real", "sin DB externa"],
        "acceptance_criteria": ["job aceptado", "replay mantiene job_ref"],
        "external_refs": [
            {"kind": "run_ref", "ref": f"run-ref-fake-{smoke_id}"},
            {"kind": "entity_ref", "ref": f"entity-ref-fake-{smoke_id}"},
            {"kind": "artifact_ref", "ref": f"artifact-ref-fake-{smoke_id}"}
        ],
        "evidence_refs": [f"evidence-ref-fake-{smoke_id}"]
    }
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_submit_payload() {
  local output="$1"
  local job_ref="$2"
  python3 - "$output" "$smoke_id" "$job_ref" <<'PY'
import json
import sys

output, smoke_id, job_ref = sys.argv[1:4]
payload = {
    "request_id": f"request-ref-external-domain-fake-submit-{smoke_id}",
    "correlation_id": f"corr-external-domain-fake-{smoke_id}",
    "action": "submit_artifact",
    "artifact_submission": {
        "schema_version": "domain_work_artifact_submission.v0",
        "request_id": f"request-ref-external-domain-fake-submit-{smoke_id}",
        "correlation_id": f"corr-external-domain-fake-{smoke_id}",
        "idempotency_key": f"idem-external-domain-fake-submit-{smoke_id}",
        "requested_by": "orquesta-smoke",
        "domain_ref": f"domain-ref-fake-{smoke_id}",
        "job_ref": job_ref,
        "artifact_ref": f"artifact-ref-fake-{smoke_id}",
        "artifact_type": "external_summary",
        "summary": "Artefacto fake no enviado porque no hay submitter externo configurado.",
        "external_refs": [
            {"kind": "run_ref", "ref": f"run-ref-fake-{smoke_id}"},
            {"kind": "entity_ref", "ref": f"entity-ref-fake-{smoke_id}"}
        ],
        "evidence_refs": [f"evidence-ref-fake-{smoke_id}"],
        "complete_job": True
    }
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

json_value() {
  local path="$1"
  local expr="$2"
  jq -r "$expr // empty" "$path"
}

write_summary() {
  local job_ref="$1"
  local replay_ref="$2"
  local snapshot="$state_dir/domain-work-jobs/domain_work_jobs_v0.json"
  cat >"$out_dir/summary.txt" <<EOF
smoke=external-domain-fake
server=$base_url
job_ref=$job_ref
replay_job_ref=$replay_ref
snapshot=$snapshot
codex_real_executed=false
opes_touched=false
output_dir=$out_dir
EOF
}

main() {
  need_cmd go
  need_cmd curl
  need_cmd jq
  need_cmd python3
  mkdir -p "$bin_dir" "$out_dir" "$project_dir" "$runtime_dir"

  write_fake_codex
  (cd "$repo_root" && go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server)
  start_server

  local create_payload="$out_dir/create_job_request.json"
  local create_response="$out_dir/create_job_response.json"
  local replay_response="$out_dir/create_job_replay_response.json"
  write_create_payload "$create_payload"

  post_json "$base_url/api/v0/domain-work" "$create_payload" "$create_response"
  local estado job_ref domain_ref run_ref
  estado="$(json_value "$create_response" '.estado')"
  job_ref="$(json_value "$create_response" '.job.job_ref')"
  domain_ref="$(json_value "$create_response" '.job.domain_ref')"
  run_ref="$(json_value "$create_response" '.job.external_refs[]? | select(.kind=="run_ref") | .ref' | head -n 1)"
  if [[ "$estado" != "ok" || -z "$job_ref" || "$domain_ref" != "domain-ref-fake-$smoke_id" ]]; then
    echo "create_job invalido" >&2
    cat "$create_response" >&2
    exit 1
  fi
  if [[ "$run_ref" != "run-ref-fake-$smoke_id" ]]; then
    echo "refs opacas no preservadas" >&2
    cat "$create_response" >&2
    exit 1
  fi

  post_json "$base_url/api/v0/domain-work" "$create_payload" "$replay_response"
  local replay_ref
  replay_ref="$(json_value "$replay_response" '.job.job_ref')"
  if [[ "$replay_ref" != "$job_ref" ]]; then
    echo "replay no idempotente: first=$job_ref replay=$replay_ref" >&2
    exit 1
  fi

  local submit_payload="$out_dir/submit_artifact_request.json"
  local submit_response="$out_dir/submit_artifact_response.json"
  write_submit_payload "$submit_payload" "$job_ref"
  post_json_expect_status "$base_url/api/v0/domain-work" "$submit_payload" "$submit_response" "400"
  local submit_code
  submit_code="$(json_value "$submit_response" '.errores_publicos[0].code')"
  if [[ "$submit_code" != "domain_work_submitter_no_disponible" ]]; then
    echo "submit_artifact no fallo por frontera neutral: code=$submit_code" >&2
    cat "$submit_response" >&2
    exit 1
  fi

  local snapshot="$state_dir/domain-work-jobs/domain_work_jobs_v0.json"
  if [[ ! -s "$snapshot" ]]; then
    echo "snapshot domain-work no creado: $snapshot" >&2
    exit 1
  fi
  local snapshot_count snapshot_run_ref
  snapshot_count="$(jq '[.. | objects | select(.job_ref? == "'"$job_ref"'")] | length' "$snapshot")"
  snapshot_run_ref="$(jq -r '.. | objects | select(.kind? == "run_ref") | .ref' "$snapshot" | head -n 1)"
  if [[ "$snapshot_count" -lt 1 || "$snapshot_run_ref" != "run-ref-fake-$smoke_id" ]]; then
    echo "snapshot no conserva job/refs esperados" >&2
    cat "$snapshot" >&2
    exit 1
  fi

  write_summary "$job_ref" "$replay_ref"
  cat "$out_dir/summary.txt"
}

main "$@"
