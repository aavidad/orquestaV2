#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_id="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

smoke_require_confirm \
  ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP \
  1 \
  "smoke legacy external-work desactivado: exporta ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1 para probar el loop historico"

smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
work_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-external-domain-fake.XXXXXX")}"
smoke_temp_root_prepare "$work_root" "$smoke_root_source"
state_dir="$work_root/state"
project_dir="$work_root/project"
runtime_dir="$project_dir/.orquesta-runtime"
bin_dir="$work_root/bin"
out_dir="${SMOKE_OUT_DIR:-$work_root/out}"
required_test_output_dir="$out_dir/required-test-output"
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
      curl -fsS -m 2 "http://$addr/api/v0/server/readiness" >/dev/null; then
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
  ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP="true" \
  ORQUESTA_OPES_BASE_URL="" \
  OPES_BASE_URL="" \
  ORQUESTA_DOMAIN_WORK_FILE_ENABLED="1" \
  ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED="1" \
  ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS="validar=$bin_dir/validar" \
  ORQUESTA_REQUIRED_TEST_OUTPUT_DIR="$required_test_output_dir" \
  ORQUESTA_CODEX_COMMAND="$bin_dir/codex-fake" \
    "$bin_dir/orquesta-server" run >"$stdout_log" 2>"$stderr_log" &
  server_pid="$!"

  if ! wait_server_ready "$server_pid"; then
    echo "orquesta-server temporal no llego a readiness OK" >&2
    tail -n 80 "$stderr_log" >&2 || true
    exit 1
  fi
  echo "servidor listo: $base_url pid=$server_pid"
}

stop_server() {
  if [[ -z "$server_pid" ]]; then
    smoke_cleanup_codex_app_server_tmux_runtime "$runtime_dir"
    return 0
  fi
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"
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

write_fake_required_test_validator() {
  mkdir -p "$bin_dir"
  cat >"$bin_dir/validar" <<'SH'
#!/usr/bin/env sh
set -eu
if [ ! -d external ]; then
  echo "external artifact dir missing" >&2
  exit 1
fi
if ! find external -type f -name '*.md' -print -quit | grep -q .; then
  echo "external artifact missing" >&2
  exit 1
fi
if ! grep -R "artefacto fake neutral" external >/dev/null 2>&1; then
  echo "external artifact content invalid" >&2
  exit 1
fi
printf 'validacion no-OPES ok: %s\n' "$*"
SH
  chmod 700 "$bin_dir/validar"
}

write_fake_codex() {
  mkdir -p "$bin_dir"
  cat >"$bin_dir/codex-fake" <<'SH'
#!/usr/bin/env sh
set -eu
out=""
project_dir=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output-last-message)
      shift
      out="${1:-}"
      ;;
    -C)
      shift
      project_dir="${1:-}"
      ;;
  esac
  shift || break
done
if [ -z "$out" ]; then
  echo "codex fake sin --output-last-message" >&2
  exit 64
fi
runtime_dir="$(dirname "$out")"
packet="$runtime_dir/agent_packet.json"
ack="$runtime_dir/agent_ack.json"
python3 - "$packet" "$ack" "${project_dir:-$(pwd)}" "$out" <<'PY'
import json
import os
import pathlib
import sys

packet_path, ack_path, project_dir, last_message_path = sys.argv[1:5]
with open(packet_path, encoding="utf-8") as fh:
    packet = json.load(fh)

task = packet.get("task") or {}
delivery_refs = packet.get("delivery_refs") or {}
write_set = [x for x in (task.get("write_set") or ["external/fake/entrega.md"]) if str(x).strip()]

def delivery_file(target):
    target = str(target).strip().strip("/")
    if not target or target == ".":
        return "external/fake/external_summary.md"
    if any(ch in target for ch in "*?["):
        return "external/fake/external_summary.md"
    suffix = pathlib.PurePosixPath(target).suffix
    if suffix:
        return target
    return target + "/external_summary.md"

files = []
root = pathlib.Path(project_dir)
for target in write_set:
    rel = delivery_file(target)
    path = root / pathlib.PurePosixPath(rel)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        "artefacto fake neutral\n"
        f"task_ref={task.get('task_ref','')}\n"
        f"work_profile={task.get('work_profile_kind','')}\n",
        encoding="utf-8",
    )
    files.append(rel)

ack = {
    "schema_version": "codex_agent_ack.v0",
    "request_id": packet.get("request_id", ""),
    "correlation_id": packet.get("correlation_id", ""),
    "ack_ref": delivery_refs.get("ack_ref", ""),
    "target_module": packet.get("target_module", ""),
    "task_ref": task.get("task_ref", ""),
    "status": "completed",
    "files": files,
    "tests": task.get("required_tests") or [],
    "notes": ["codex fake: artefacto neutral generado sin Codex real"],
}
tmp = ack_path + ".tmp"
with open(tmp, "w", encoding="utf-8") as fh:
    json.dump(ack, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
os.replace(tmp, ack_path)
pathlib.Path(last_message_path).write_text("codex fake final: smoke no-OPES\n", encoding="utf-8")
PY
printf 'codex fake stdout: ack generado\n'
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
        "summary": "Artefacto fake aceptado por el submitter file-based local.",
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

write_external_work_payload() {
  local output="$1"
  local job_ref="$2"
  python3 - "$output" "$smoke_id" "$job_ref" <<'PY'
import json
import sys

output, smoke_id, job_ref = sys.argv[1:4]
payload = {
    "request_id": f"request-ref-external-domain-fake-run-{smoke_id}",
    "correlation_id": f"corr-external-domain-fake-{smoke_id}",
    "director_execution_mode": "legacy_director_loop",
    "external_work_run_request": {
        "schema_version": "external_work_run_request.v0",
        "request_id": f"request-ref-external-domain-fake-run-{smoke_id}",
        "correlation_id": f"corr-external-domain-fake-{smoke_id}",
        "run_ref": f"run-external-domain-fake-{smoke_id}",
        "project_ref": f"external-domain-fake-{smoke_id}",
        "app_spec_ref": f"app-spec-external-domain-fake-{smoke_id}",
        "queue_ref": "global",
        "priority_score": 90,
        "occurred_at": "2026-05-22T12:00:00Z",
        "requested_by": "orquesta-smoke",
        "app_change_request": {
            "schema_version": "app_change_request.v0",
            "request_id": f"request-ref-external-domain-fake-change-{smoke_id}",
            "correlation_id": f"corr-external-domain-fake-{smoke_id}",
            "run_ref": f"run-external-domain-fake-{smoke_id}",
            "app_ref": f"external-domain-fake-{smoke_id}",
            "change_ref": f"change-ref-external-domain-fake-{smoke_id}",
            "actor_ref": "orquesta-smoke",
            "locale": "es",
            "user_intent": "Resolver trabajo externo neutral con refs opacas y devolver artefacto fake.",
            "target_area": "domain_work",
            "current_state_refs": [
                f"domain-ref-fake-{smoke_id}",
                job_ref,
                f"scope-ref-fake-{smoke_id}"
            ],
            "acceptance_criteria": [
                "artefacto externo generado",
                "ACK causal registrado",
                "review aceptada por evidencia de fichero",
                "sin OPES ni DB compartida"
            ],
            "constraints": [
                "usar solo refs opacas",
                "runtime fake acotado",
                "no llamar OPES"
            ],
            "external_work": {
                "project_ref": f"external-domain-fake-{smoke_id}",
                "job_ref": job_ref,
                "interface_refs": ["external-domain-fake.v0", "domain-work.v0"],
                "work_kind": "generation",
                "work_refs": [f"work-ref-fake-{smoke_id}"],
                "input_fields": [
                    {"name": "title", "value": "Smoke externo fake"},
                    {"name": "expected_artifact_type", "value": "external_summary"},
                    {"name": "source_refs", "values": [f"source-ref-fake-{smoke_id}"]},
                    {"name": "scope_ref", "value": f"scope-ref-fake-{smoke_id}"}
                ]
            }
        }
    }
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_supervisor_payload() {
  local output="$1"
  local run_ref="$2"
  local request_suffix="${3:-supervise}"
  local occurred_at="${4:-2026-05-22T12:01:00Z}"
  python3 - "$output" "$smoke_id" "$run_ref" <<'PY'
import json
import sys

output, smoke_id, run_ref = sys.argv[1:4]
payload = {
    "request_id": f"request-ref-external-domain-fake-supervise-{smoke_id}",
    "correlation_id": f"corr-external-domain-fake-{smoke_id}",
    "director_execution_mode": "legacy_director_loop",
    "run_ref": run_ref,
    "queue_ref": "global",
    "continue_message": "sigue",
    "occurred_at": "2026-05-22T12:01:00Z",
    "max_ticks": 1,
    "max_runs_per_tick": 1,
    "max_executions": 8,
    "allow_repeated_runs": True,
    "max_bursts": 16,
    "max_steps_per_burst": 8,
    "max_dispatches_per_wait": 8,
    "max_commands": 32,
    "max_outbox_per_cycle": 8,
    "max_decision_cycles": 4,
    "max_external_waits": 4
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
  python3 - "$output" "$smoke_id" "$request_suffix" "$occurred_at" <<'PY'
import json
import sys

path, smoke_id, request_suffix, occurred_at = sys.argv[1:5]
with open(path, encoding="utf-8") as fh:
    payload = json.load(fh)
payload["request_id"] = f"request-ref-external-domain-fake-{request_suffix}-{smoke_id}"
payload["occurred_at"] = occurred_at
with open(path, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_stats_payload() {
  local output="$1"
  local run_ref="$2"
  local job_ref="$3"
  python3 - "$output" "$smoke_id" "$run_ref" "$job_ref" <<'PY'
import json
import sys

output, smoke_id, run_ref, job_ref = sys.argv[1:5]
payload = {
    "request_id": f"request-ref-external-domain-fake-stats-{smoke_id}",
    "correlation_id": f"corr-external-domain-fake-{smoke_id}",
    "run_ref": run_ref,
    "external_job_ref": job_ref,
    "include_process_refs": True,
    "include_agent_progress": True
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

post_json_required() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  local status
  status="$(curl -sS -m "$request_timeout" \
    -H 'Content-Type: application/json' \
    -H "X-Correlation-ID: corr-external-domain-fake-$smoke_id" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output" \
    -w '%{http_code}')"
  if [[ "$status" != 2* ]]; then
    echo "HTTP inesperado para $url: got=$status" >&2
    cat "$output" >&2 || true
    exit 1
  fi
}

post_json_status() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  curl -sS -m "$request_timeout" \
    -H 'Content-Type: application/json' \
    -H "X-Correlation-ID: corr-external-domain-fake-$smoke_id" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output" \
    -w '%{http_code}'
}

verify_external_cycle() {
  local stats_response="$1"
  python3 - "$stats_response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("estado") != "ok":
    raise SystemExit(f"stats no ok: {data}")
stats = data.get("stats") or {}
counts = stats.get("counts") or {}
external_job = data.get("external_job") or {}
needed = {
    "tasks_total": 1,
    "agents_started": 1,
    "deliveries": 1,
}
for key, minimum in needed.items():
    if int(counts.get(key) or 0) < minimum:
        raise SystemExit(f"stats falta {key}>={minimum}: {json.dumps(data, ensure_ascii=True)}")
if not external_job.get("task_ref") or not external_job.get("agent_ref") or not external_job.get("delivery_refs"):
    raise SystemExit(f"external_job incompleto: {external_job}")
closure = stats.get("closure") or {}
print("external_cycle_delivery_refs=" + ",".join(external_job.get("delivery_refs") or []))
print("external_cycle_review_results=" + str(counts.get("review_results", "")))
print("external_cycle_closure_status=" + str(closure.get("status", "")))
print("external_cycle_closed=" + str(closure.get("closed", False)).lower())
if int(counts.get("reviews") or 0) == 0:
    print("external_cycle_review_pending=true")
PY
}

verify_operational_live_gap() {
  local stats_response="$1"
  python3 - "$state_dir/orchestration-state" "$stats_response" <<'PY'
import glob
import json
import os
import sys

state_root, stats_path = sys.argv[1:3]
with open(stats_path, encoding="utf-8") as fh:
    stats_response = json.load(fh)

stats = stats_response.get("stats") or {}
counts = stats.get("counts") or {}
if int(counts.get("deliveries") or 0) < 1:
    raise SystemExit("operational gap requiere delivery registrada")

plan_files = sorted(glob.glob(os.path.join(state_root, "operational_director_plan_states", "*", "*.json")))
if not plan_files:
    raise SystemExit("no hay operational_director_plan_state tras reentrada viva")

with open(plan_files[-1], encoding="utf-8") as fh:
    envelope = json.load(fh)
state = envelope.get("state") or {}
steps = state.get("steps") or []
active_step_id = state.get("active_step_id") or ""
active = next((step for step in steps if step.get("step_id") == active_step_id), {})
review_step = next((step for step in steps if step.get("kind") == "review_deliveries"), {})

print("operational_plan_ref=" + str(state.get("plan_ref") or ""))
plan_status = str(state.get("status") or "")
active_kind = str(active.get("kind") or "")
active_status = str(active.get("status") or "")
closure_reason = str(state.get("closure_reason") or active.get("reason") or "")

required_evidence_refs = []
for step in steps:
    required_evidence_refs.extend(step.get("required_test_evidence_refs") or [])

print("operational_plan_status=" + plan_status)
print("operational_active_step=" + str(active_step_id))
print("operational_active_step_status=" + active_status)
print("operational_closure_reason=" + closure_reason)
print("operational_review_step_status=" + str(review_step.get("status") or ""))
print("operational_reviews=" + str(counts.get("reviews", 0)))
print("operational_review_results=" + str(counts.get("review_results", 0)))
print("operational_accepted_reviews=" + str(counts.get("accepted_reviews", 0)))
print("operational_required_test_evidence_refs=" + ",".join(required_evidence_refs))

if plan_status != "closed":
    raise SystemExit(f"estado operativo no cerrado: {plan_status} reason={closure_reason} active={active}")
if active_kind != "replan_or_close" or active_status not in ("accepted", "closed"):
    raise SystemExit(f"active step no cerro replan_or_close: {active}")
if review_step.get("status") != "accepted":
    raise SystemExit(f"review_deliveries inesperado: {review_step.get('status')}")
if not required_evidence_refs:
    raise SystemExit("no hay evidencia durable de required tests")
for key in ("reviews", "review_results", "accepted_reviews"):
    if int(counts.get(key) or 0) < 1:
        raise SystemExit(f"la ruta viva no genero {key}: {counts}")
PY
}

write_summary() {
  local job_ref="$1"
  local replay_ref="$2"
  local run_ref="$3"
  local stats_response="$4"
  local operational_evidence="$5"
  local snapshot="$state_dir/domain-work-jobs/domain_work_jobs_v0.json"
  local closure_status
  closure_status="$(json_value "$stats_response" '.stats.closure.status')"
  cat >"$out_dir/summary.txt" <<EOF
smoke=external-domain-fake
server=$base_url
job_ref=$job_ref
replay_job_ref=$replay_ref
external_run_ref=$run_ref
stats_response=$stats_response
closure_status=$closure_status
operational_evidence=$operational_evidence
snapshot=$snapshot
codex_fake_executed=true
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

  mkdir -p "$required_test_output_dir"
  write_fake_required_test_validator
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
  post_json "$base_url/api/v0/domain-work" "$submit_payload" "$submit_response"
  local submit_estado submit_status submit_job_ref submit_receipt_ref
  submit_estado="$(json_value "$submit_response" '.estado')"
  submit_status="$(json_value "$submit_response" '.receipt.status')"
  submit_job_ref="$(json_value "$submit_response" '.receipt.job_ref')"
  submit_receipt_ref="$(json_value "$submit_response" '.receipt.receipt_ref')"
  if [[ "$submit_estado" != "ok" || "$submit_status" != "accepted" || "$submit_job_ref" != "$job_ref" || -z "$submit_receipt_ref" ]]; then
    echo "submit_artifact file-based invalido" >&2
    cat "$submit_response" >&2
    exit 1
  fi

  local snapshot="$state_dir/domain-work-jobs/domain_work_jobs_v0.json"
  local artifact_snapshot="$state_dir/domain-work-jobs/domain_work_artifacts_v0.json"
  if [[ ! -s "$snapshot" ]]; then
    echo "snapshot domain-work no creado: $snapshot" >&2
    exit 1
  fi
  if [[ ! -s "$artifact_snapshot" ]]; then
    echo "snapshot domain-work artifacts no creado: $artifact_snapshot" >&2
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
  local artifact_snapshot_count
  artifact_snapshot_count="$(jq '[.. | objects | select(.artifact_ref? == "artifact-ref-fake-'"$smoke_id"'")] | length' "$artifact_snapshot")"
  if [[ "$artifact_snapshot_count" -lt 1 ]]; then
    echo "snapshot artifact no conserva artifact_ref esperado" >&2
    cat "$artifact_snapshot" >&2
    exit 1
  fi

  local external_payload="$out_dir/external_work_run_request.json"
  local external_response="$out_dir/external_work_run_response.json"
  write_external_work_payload "$external_payload" "$job_ref"
  post_json_required "$base_url/api/v0/external-work/run" "$external_payload" "$external_response"
  local external_estado external_run_ref
  external_estado="$(json_value "$external_response" '.estado')"
  external_run_ref="$(json_value "$external_response" '.run_ref')"
  if [[ "$external_estado" != "ok" || "$external_run_ref" != "run-external-domain-fake-$smoke_id" ]]; then
    echo "external-work/run invalido" >&2
    cat "$external_response" >&2
    exit 1
  fi

  local supervisor_payload="$out_dir/supervisor_request.json"
  local supervisor_response="$out_dir/supervisor_response.json"
  write_supervisor_payload "$supervisor_payload" "$external_run_ref"

  local stats_payload="$out_dir/stats_request.json"
  local stats_response="$out_dir/stats_response.json"
  write_stats_payload "$stats_payload" "$external_run_ref" "$job_ref"
  local supervisor_ok=0
  for attempt in $(seq 1 12); do
    local supervisor_status
    supervisor_status="$(post_json_status "$base_url/api/v0/runs/supervise" "$supervisor_payload" "$supervisor_response")"
    if [[ "$supervisor_status" == 2* && "$(json_value "$supervisor_response" '.estado')" == "ok" ]]; then
      supervisor_ok=1
    fi
    post_json_required "$base_url/api/v0/director/stats" "$stats_payload" "$stats_response"
    if verify_external_cycle "$stats_response"; then
      break
    fi
    if [[ "$supervisor_status" != 2* && "$supervisor_ok" != "1" ]]; then
      local deliveries
      deliveries="$(json_value "$stats_response" '.stats.counts.deliveries')"
      if [[ "${deliveries:-0}" -lt 1 ]]; then
        echo "supervisor invalido intento=$attempt http=$supervisor_status" >&2
        cat "$supervisor_response" >&2 || true
        exit 1
      fi
    fi
    sleep 0.5
  done
  verify_external_cycle "$stats_response"

  local post_delivery_supervisor_payload="$out_dir/post_delivery_supervisor_request.json"
  local post_delivery_supervisor_response="$out_dir/post_delivery_supervisor_response.json"
  write_supervisor_payload "$post_delivery_supervisor_payload" "$external_run_ref" "post-delivery-supervise" "2026-05-22T12:02:00Z"
  local post_delivery_supervisor_status
  post_delivery_supervisor_status="$(post_json_status "$base_url/api/v0/runs/supervise" "$post_delivery_supervisor_payload" "$post_delivery_supervisor_response")"
  post_json_required "$base_url/api/v0/director/stats" "$stats_payload" "$stats_response"

  local operational_evidence="$out_dir/operational_live_gap.txt"
  {
    echo "post_delivery_supervisor_http=$post_delivery_supervisor_status"
    echo "post_delivery_supervisor_response=$post_delivery_supervisor_response"
    echo "post_delivery_supervisor_error_code=$(json_value "$post_delivery_supervisor_response" '.errores_publicos[0].code')"
    verify_operational_live_gap "$stats_response"
  } | tee "$operational_evidence"

  local ack_count artifact_count
  ack_count="$(find "$runtime_dir" -name agent_ack.json -type f | wc -l | tr -d ' ')"
  artifact_count="$(find "$project_dir/external" -type f 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "$ack_count" -lt 1 || "$artifact_count" -lt 1 ]]; then
    echo "codex-fake no produjo ACK/artefacto: ack_count=$ack_count artifact_count=$artifact_count" >&2
    find "$runtime_dir" -maxdepth 4 -type f >&2 || true
    find "$project_dir" -maxdepth 5 -type f >&2 || true
    exit 1
  fi

  write_summary "$job_ref" "$replay_ref" "$external_run_ref" "$stats_response" "$operational_evidence"
  cat "$out_dir/summary.txt"
}

main "$@"
