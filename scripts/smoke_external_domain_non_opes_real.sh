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
work_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-external-domain-non-opes.XXXXXX")}"
smoke_temp_root_prepare "$work_root" "$smoke_root_source"
state_dir="$work_root/state"
project_dir="$work_root/project"
runtime_dir="$project_dir/.orquesta-runtime"
bin_dir="$work_root/bin"
out_dir="${SMOKE_OUT_DIR:-$work_root/out}"
required_test_output_dir="$out_dir/required-test-output"
external_app_dir="$work_root/external-app"
server_pid=""
external_app_pid=""
base_url=""
external_base_url=""
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
  stop_external_app || true
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

wait_external_app_ready() {
  local addr_file="$external_app_dir/addr.txt"
  local addr=""
  for _ in $(seq 1 80); do
    if [[ -s "$addr_file" ]]; then
      addr="$(cat "$addr_file")"
      if curl -fsS -m 2 "http://$addr/healthz" >/dev/null; then
        external_base_url="http://$addr"
        return 0
      fi
    fi
    sleep 0.25
  done
  return 1
}

start_external_app() {
  mkdir -p "$external_app_dir"
  write_external_app
  python3 "$external_app_dir/app.py" \
    "$external_app_dir/state.json" \
    "$external_app_dir/addr.txt" \
    >"$out_dir/external-app.stdout.log" \
    2>"$out_dir/external-app.stderr.log" &
  external_app_pid="$!"
  if ! wait_external_app_ready; then
    echo "external app temporal no llego a health OK" >&2
    tail -n 80 "$out_dir/external-app.stderr.log" >&2 || true
    exit 1
  fi
  echo "external app lista: $external_base_url pid=$external_app_pid"
}

stop_external_app() {
  if [[ -z "$external_app_pid" ]]; then
    return 0
  fi
  if kill -0 "$external_app_pid" >/dev/null 2>&1; then
    kill -INT "$external_app_pid" >/dev/null 2>&1 || true
    for _ in $(seq 1 20); do
      if ! kill -0 "$external_app_pid" >/dev/null 2>&1; then
        external_app_pid=""
        return 0
      fi
      sleep 0.2
    done
    kill -TERM "$external_app_pid" >/dev/null 2>&1 || true
    wait "$external_app_pid" >/dev/null 2>&1 || true
  fi
  external_app_pid=""
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
  ORQUESTA_DOMAIN_WORK_FILE_ENABLED="" \
  ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL="$external_base_url" \
  ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH="/api/domain-work/jobs" \
  ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH="/api/domain-work/artifacts" \
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

post_json_required() {
  local url="$1"
  local payload_file="$2"
  local output="$3"
  local status
  status="$(curl -sS -m "$request_timeout" \
    -H 'Content-Type: application/json' \
    -H "X-Correlation-ID: corr-external-domain-non-opes-$smoke_id" \
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
    -H "X-Correlation-ID: corr-external-domain-non-opes-$smoke_id" \
    -X POST "$url" \
    --data-binary "@$payload_file" \
    -o "$output" \
    -w '%{http_code}'
}

json_value() {
  local path="$1"
  local expr="$2"
  jq -r "$expr // empty" "$path"
}

write_external_app() {
  cat >"$external_app_dir/app.py" <<'PY'
import hashlib
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import sys

state_path, addr_path = sys.argv[1:3]

def load_state():
    try:
        with open(state_path, encoding="utf-8") as fh:
            return json.load(fh)
    except FileNotFoundError:
        return {"jobs": {}, "jobs_by_key": {}, "artifacts": {}, "artifacts_by_key": {}}

def save_state(state):
    os.makedirs(os.path.dirname(state_path), exist_ok=True)
    tmp = state_path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as fh:
        json.dump(state, fh, ensure_ascii=True, indent=2, sort_keys=True)
        fh.write("\n")
    os.replace(tmp, state_path)

def stable_ref(prefix, *parts):
    digest = hashlib.sha256("\x00".join(parts).encode("utf-8")).hexdigest()[:16]
    return f"{prefix}-{digest}"

def compact(value):
    return isinstance(value, str) and value.strip() and not any(ch in value for ch in " /\\\t\r\n")

def read_json(handler):
    length = int(handler.headers.get("Content-Length") or "0")
    return json.loads(handler.rfile.read(length).decode("utf-8"))

def write_json(handler, status, payload):
    data = json.dumps(payload, ensure_ascii=True).encode("utf-8") + b"\n"
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(data)))
    handler.end_headers()
    handler.wfile.write(data)

class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        return

    def do_GET(self):
        if self.path == "/healthz":
            write_json(self, 200, {"status": "ok"})
            return
        write_json(self, 404, {"error": "not_found"})

    def do_POST(self):
        if self.path == "/api/domain-work/jobs":
            self.create_job()
            return
        if self.path == "/api/domain-work/artifacts":
            self.submit_artifact()
            return
        write_json(self, 404, {"error": "not_found"})

    def create_job(self):
        request = read_json(self)
        state = load_state()
        domain_ref = (request.get("domain_ref") or "").strip()
        idem = (request.get("idempotency_key") or "").strip()
        if not compact(domain_ref) or not compact(idem):
            write_json(self, 400, {"error": "invalid_ref"})
            return
        key = domain_ref + "\x00" + idem
        if key in state["jobs_by_key"]:
            write_json(self, 200, {"job": state["jobs"][state["jobs_by_key"][key]]})
            return
        job_ref = stable_ref("job-ref-non-opes", domain_ref, idem)
        job = {
            "schema_version": "domain_work_job.v0",
            "status": "accepted",
            "job_ref": job_ref,
            "domain_ref": domain_ref,
            "work_kind": (request.get("work_kind") or "").strip(),
            "correlation_id": (request.get("correlation_id") or "").strip(),
            "idempotency_key": idem,
            "external_refs": request.get("external_refs") or [],
            "evidence_refs": request.get("evidence_refs") or [],
        }
        state["jobs"][job_ref] = job
        state["jobs_by_key"][key] = job_ref
        save_state(state)
        write_json(self, 200, {"job": job})

    def submit_artifact(self):
        submission = read_json(self)
        state = load_state()
        domain_ref = (submission.get("domain_ref") or "").strip()
        idem = (submission.get("idempotency_key") or "").strip()
        job_ref = (submission.get("job_ref") or "").strip()
        artifact_ref = (submission.get("artifact_ref") or "").strip()
        if not compact(domain_ref) or not compact(idem) or not compact(job_ref) or not compact(artifact_ref):
            write_json(self, 400, {"error": "invalid_ref"})
            return
        if job_ref not in state["jobs"]:
            write_json(self, 404, {"error": "job_not_found"})
            return
        key = domain_ref + "\x00" + idem
        if key in state["artifacts_by_key"]:
            write_json(self, 200, {"receipt": state["artifacts"][state["artifacts_by_key"][key]]["receipt"]})
            return
        receipt_ref = stable_ref("receipt-ref-non-opes", domain_ref, idem, artifact_ref)
        receipt = {
            "schema_version": "domain_work_artifact_receipt.v0",
            "status": "accepted",
            "job_ref": job_ref,
            "artifact_ref": artifact_ref,
            "receipt_ref": receipt_ref,
            "correlation_id": (submission.get("correlation_id") or "").strip(),
            "idempotency_key": idem,
            "external_refs": submission.get("external_refs") or [],
            "evidence_refs": submission.get("evidence_refs") or [],
        }
        state["artifacts"][receipt_ref] = {"submission": submission, "receipt": receipt}
        state["artifacts_by_key"][key] = receipt_ref
        if submission.get("complete_job"):
            state["jobs"][job_ref]["status"] = "completed"
        save_state(state)
        write_json(self, 200, {"receipt": receipt})

httpd = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
addr = f"{httpd.server_address[0]}:{httpd.server_address[1]}"
os.makedirs(os.path.dirname(addr_path), exist_ok=True)
with open(addr_path, "w", encoding="utf-8") as fh:
    fh.write(addr)
try:
    httpd.serve_forever()
except KeyboardInterrupt:
    pass
PY
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
if ! grep -R "artefacto externo neutral" external >/dev/null 2>&1; then
  echo "external artifact content invalid" >&2
  exit 1
fi
printf 'validacion app externa no-OPES ok: %s\n' "$*"
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
import pathlib
import sys

packet_path, ack_path, project_dir, last_message_path = sys.argv[1:5]
with open(packet_path, encoding="utf-8") as fh:
    packet = json.load(fh)

task = packet.get("task") or {}
delivery_refs = packet.get("delivery_refs") or {}
write_set = [x for x in (task.get("write_set") or ["external/non-opes/entrega.md"]) if str(x).strip()]

def delivery_file(target):
    target = str(target).strip().strip("/")
    if not target or target == "." or any(ch in target for ch in "*?["):
        return "external/non-opes/entrega.md"
    if pathlib.PurePosixPath(target).suffix:
        return target
    return target + "/entrega.md"

files = []
root = pathlib.Path(project_dir)
for target in write_set:
    rel = delivery_file(target)
    path = root / pathlib.PurePosixPath(rel)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        "artefacto externo neutral\n"
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
    "notes": ["codex fake: artefacto neutral generado para app externa no-OPES"],
}
tmp = ack_path + ".tmp"
with open(tmp, "w", encoding="utf-8") as fh:
    json.dump(ack, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
pathlib.Path(tmp).replace(ack_path)
pathlib.Path(last_message_path).write_text("codex fake final: smoke app externa no-OPES\n", encoding="utf-8")
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
    "request_id": f"request-ref-external-domain-non-opes-{smoke_id}",
    "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
    "action": "create_job",
    "job_request": {
        "schema_version": "domain_work_job_request.v0",
        "request_id": f"request-ref-external-domain-non-opes-{smoke_id}",
        "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
        "idempotency_key": f"idem-external-domain-non-opes-{smoke_id}",
        "requested_by": "orquesta-smoke",
        "domain_ref": f"domain-ref-non-opes-{smoke_id}",
        "interface_refs": ["domain-work.v0", "external-non-opes.v0"],
        "work_kind": "compose_external_summary",
        "work_refs": [f"work-ref-non-opes-{smoke_id}"],
        "objective": "Crear trabajo neutral para app externa no-OPES usando solo refs opacas.",
        "input_fields": [
            {"name": "title", "value": "Smoke externo no OPES"},
            {"name": "scope_ref", "value": f"scope-ref-non-opes-{smoke_id}"}
        ],
        "input_refs": [f"source-ref-non-opes-{smoke_id}"],
        "constraints": ["sin OPES", "sin DB compartida", "submitter HTTP opt-in"],
        "acceptance_criteria": ["job aceptado", "artefacto recibido por submitter real"],
        "external_refs": [
            {"kind": "entity_ref", "ref": f"entity-ref-non-opes-{smoke_id}"},
            {"kind": "artifact_ref", "ref": f"artifact-ref-non-opes-{smoke_id}"}
        ],
        "evidence_refs": [f"evidence-ref-non-opes-{smoke_id}"]
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
    "request_id": f"request-ref-external-domain-non-opes-submit-{smoke_id}",
    "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
    "action": "submit_artifact",
    "artifact_submission": {
        "schema_version": "domain_work_artifact_submission.v0",
        "request_id": f"request-ref-external-domain-non-opes-submit-{smoke_id}",
        "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
        "idempotency_key": f"idem-external-domain-non-opes-submit-manual-{smoke_id}",
        "requested_by": "orquesta-smoke",
        "domain_ref": f"domain-ref-non-opes-{smoke_id}",
        "job_ref": job_ref,
        "artifact_ref": f"artifact-ref-non-opes-manual-{smoke_id}",
        "artifact_type": "external_summary",
        "summary": "Artefacto manual de smoke enviado al submitter HTTP neutral.",
        "external_refs": [{"kind": "entity_ref", "ref": f"entity-ref-non-opes-{smoke_id}"}],
        "evidence_refs": [f"evidence-ref-non-opes-manual-{smoke_id}"],
        "complete_job": False
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
    "request_id": f"request-ref-external-domain-non-opes-run-{smoke_id}",
    "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
    "external_work_run_request": {
        "schema_version": "external_work_run_request.v0",
        "request_id": f"request-ref-external-domain-non-opes-run-{smoke_id}",
        "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
        "run_ref": f"run-external-domain-non-opes-{smoke_id}",
        "project_ref": f"external-domain-non-opes-{smoke_id}",
        "app_spec_ref": f"app-spec-external-domain-non-opes-{smoke_id}",
        "queue_ref": "global",
        "priority_score": 90,
        "occurred_at": "2026-05-22T12:00:00Z",
        "requested_by": "orquesta-smoke",
        "app_change_request": {
            "schema_version": "app_change_request.v0",
            "request_id": f"request-ref-external-domain-non-opes-change-{smoke_id}",
            "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
            "run_ref": f"run-external-domain-non-opes-{smoke_id}",
            "app_ref": f"external-domain-non-opes-{smoke_id}",
            "change_ref": f"change-ref-external-domain-non-opes-{smoke_id}",
            "actor_ref": "orquesta-smoke",
            "locale": "es",
            "user_intent": "Resolver trabajo externo neutral con refs opacas y devolver artefacto a app HTTP no-OPES.",
            "target_area": "domain_work",
            "current_state_refs": [
                f"domain-ref-non-opes-{smoke_id}",
                job_ref,
                f"scope-ref-non-opes-{smoke_id}"
            ],
            "acceptance_criteria": [
                "artefacto externo generado",
                "ACK causal registrado",
                "DomainWork submit_artifact devuelve receipt",
                "review aceptada por evidencia de fichero",
                "sin OPES ni DB compartida"
            ],
            "constraints": [
                "usar solo refs opacas",
                "runtime fake acotado",
                "submitter HTTP neutral opt-in",
                "no llamar OPES"
            ],
            "external_work": {
                "project_ref": f"domain-ref-non-opes-{smoke_id}",
                "job_ref": job_ref,
                "interface_refs": ["external-non-opes.v0", "domain-work.v0"],
                "work_kind": "compose_external_summary",
                "work_refs": [f"work-ref-non-opes-{smoke_id}"],
                "input_fields": [
                    {"name": "title", "value": "Smoke externo no OPES"},
                    {"name": "expected_artifact_type", "value": "work_delivery"},
                    {"name": "source_refs", "values": [f"source-ref-non-opes-{smoke_id}"]},
                    {"name": "scope_ref", "value": f"scope-ref-non-opes-{smoke_id}"}
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
  python3 - "$output" "$smoke_id" "$run_ref" "$request_suffix" "$occurred_at" <<'PY'
import json
import sys

output, smoke_id, run_ref, request_suffix, occurred_at = sys.argv[1:6]
payload = {
    "request_id": f"request-ref-external-domain-non-opes-{request_suffix}-{smoke_id}",
    "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
    "run_ref": run_ref,
    "queue_ref": "global",
    "continue_message": "sigue",
    "occurred_at": occurred_at,
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
    "request_id": f"request-ref-external-domain-non-opes-stats-{smoke_id}",
    "correlation_id": f"corr-external-domain-non-opes-{smoke_id}",
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

verify_external_cycle() {
  local stats_response="$1"
  python3 - "$stats_response" "$external_app_dir/state.json" <<'PY'
import json
import sys

stats_path, external_state_path = sys.argv[1:3]
with open(stats_path, encoding="utf-8") as fh:
    data = json.load(fh)
with open(external_state_path, encoding="utf-8") as fh:
    external_state = json.load(fh)
if data.get("estado") != "ok":
    raise SystemExit(f"stats no ok: {data}")
counts = (data.get("stats") or {}).get("counts") or {}
needed = {"tasks_total": 1, "agents_started": 1, "deliveries": 1}
for key, minimum in needed.items():
    if int(counts.get(key) or 0) < minimum:
        raise SystemExit(f"stats falta {key}>={minimum}: {json.dumps(data, ensure_ascii=True)}")
external_job = data.get("external_job") or {}
if not external_job.get("task_ref") or not external_job.get("agent_ref") or not external_job.get("delivery_refs"):
    raise SystemExit(f"external_job incompleto: {external_job}")
artifacts = external_state.get("artifacts") or {}
if len(artifacts) < 2:
    raise SystemExit(f"submitter externo no recibio artefactos esperados: {external_state}")
submissions = [item.get("submission") or {} for item in artifacts.values()]
if not any((s.get("external_refs") or []) for s in submissions):
    raise SystemExit("submitter externo no recibio refs opacas")
print("external_cycle_delivery_refs=" + ",".join(external_job.get("delivery_refs") or []))
print("external_app_artifact_count=" + str(len(artifacts)))
PY
}

verify_operational_close() {
  local stats_response="$1"
  python3 - "$state_dir/orchestration-state" "$stats_response" <<'PY'
import glob
import json
import os
import sys

state_root, stats_path = sys.argv[1:3]
with open(stats_path, encoding="utf-8") as fh:
    stats_response = json.load(fh)
counts = ((stats_response.get("stats") or {}).get("counts") or {})
plan_files = sorted(glob.glob(os.path.join(state_root, "operational_director_plan_states", "*", "*.json")))
if not plan_files:
    raise SystemExit("no hay operational_director_plan_state")
with open(plan_files[-1], encoding="utf-8") as fh:
    envelope = json.load(fh)
state = envelope.get("state") or {}
steps = state.get("steps") or []
active = next((step for step in steps if step.get("step_id") == state.get("active_step_id")), {})
review = next((step for step in steps if step.get("kind") == "review_deliveries"), {})
required_evidence_refs = []
for step in steps:
    required_evidence_refs.extend(step.get("required_test_evidence_refs") or [])
print("operational_plan_ref=" + str(state.get("plan_ref") or ""))
print("operational_plan_status=" + str(state.get("status") or ""))
print("operational_active_step_status=" + str(active.get("status") or ""))
print("operational_review_step_status=" + str(review.get("status") or ""))
print("operational_reviews=" + str(counts.get("reviews", 0)))
print("operational_accepted_reviews=" + str(counts.get("accepted_reviews", 0)))
print("operational_required_test_evidence_refs=" + ",".join(required_evidence_refs))
if state.get("status") != "closed":
    raise SystemExit(f"estado operativo no cerrado: {state}")
if review.get("status") != "accepted":
    raise SystemExit(f"review_deliveries no aceptada: {review}")
if not required_evidence_refs:
    raise SystemExit("faltan evidencias durables de required tests")
PY
}

write_summary() {
  local job_ref="$1"
  local replay_ref="$2"
  local manual_receipt_ref="$3"
  local run_ref="$4"
  local stats_response="$5"
  local operational_evidence="$6"
  cat >"$out_dir/summary.txt" <<EOF
smoke=external-domain-non-opes-real
server=$base_url
external_app=$external_base_url
job_ref=$job_ref
replay_job_ref=$replay_ref
manual_receipt_ref=$manual_receipt_ref
external_run_ref=$run_ref
stats_response=$stats_response
operational_evidence=$operational_evidence
external_app_state=$external_app_dir/state.json
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
  mkdir -p "$bin_dir" "$out_dir" "$project_dir" "$runtime_dir" "$required_test_output_dir"

  write_fake_required_test_validator
  write_fake_codex
  (cd "$repo_root" && go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server)
  start_external_app
  start_server

  local create_payload="$out_dir/create_job_request.json"
  local create_response="$out_dir/create_job_response.json"
  local replay_response="$out_dir/create_job_replay_response.json"
  write_create_payload "$create_payload"
  post_json_required "$base_url/api/v0/domain-work" "$create_payload" "$create_response"
  post_json_required "$base_url/api/v0/domain-work" "$create_payload" "$replay_response"

  local job_ref replay_ref domain_ref
  job_ref="$(json_value "$create_response" '.job.job_ref')"
  replay_ref="$(json_value "$replay_response" '.job.job_ref')"
  domain_ref="$(json_value "$create_response" '.job.domain_ref')"
  if [[ -z "$job_ref" || "$replay_ref" != "$job_ref" || "$domain_ref" != "domain-ref-non-opes-$smoke_id" ]]; then
    echo "create/replay invalido" >&2
    cat "$create_response" >&2
    cat "$replay_response" >&2
    exit 1
  fi

  local submit_payload="$out_dir/submit_artifact_request.json"
  local submit_response="$out_dir/submit_artifact_response.json"
  write_submit_payload "$submit_payload" "$job_ref"
  post_json_required "$base_url/api/v0/domain-work" "$submit_payload" "$submit_response"
  local manual_receipt_ref
  manual_receipt_ref="$(json_value "$submit_response" '.receipt.receipt_ref')"
  if [[ -z "$manual_receipt_ref" ]]; then
    echo "submitter HTTP neutral no devolvio receipt" >&2
    cat "$submit_response" >&2
    exit 1
  fi

  local external_payload="$out_dir/external_work_run_request.json"
  local external_response="$out_dir/external_work_run_response.json"
  write_external_work_payload "$external_payload" "$job_ref"
  post_json_required "$base_url/api/v0/external-work/run" "$external_payload" "$external_response"
  local external_run_ref
  external_run_ref="$(json_value "$external_response" '.run_ref')"
  if [[ "$external_run_ref" != "run-external-domain-non-opes-$smoke_id" ]]; then
    echo "external-work/run invalido" >&2
    cat "$external_response" >&2
    exit 1
  fi

  local supervisor_payload="$out_dir/supervisor_request.json"
  local supervisor_response="$out_dir/supervisor_response.json"
  local stats_payload="$out_dir/stats_request.json"
  local stats_response="$out_dir/stats_response.json"
  write_supervisor_payload "$supervisor_payload" "$external_run_ref"
  write_stats_payload "$stats_payload" "$external_run_ref" "$job_ref"

  for _ in $(seq 1 12); do
    post_json_status "$base_url/api/v0/runs/supervise" "$supervisor_payload" "$supervisor_response" >/dev/null
    post_json_required "$base_url/api/v0/director/stats" "$stats_payload" "$stats_response"
    if verify_external_cycle "$stats_response" >/dev/null 2>&1; then
      break
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

  local operational_evidence="$out_dir/operational_close.txt"
  {
    echo "post_delivery_supervisor_http=$post_delivery_supervisor_status"
    echo "post_delivery_supervisor_response=$post_delivery_supervisor_response"
    verify_operational_close "$stats_response"
  } | tee "$operational_evidence"

  local ack_count artifact_count
  ack_count="$(find "$runtime_dir" -name agent_ack.json -type f | wc -l | tr -d ' ')"
  artifact_count="$(find "$project_dir/external" -type f 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "$ack_count" -lt 1 || "$artifact_count" -lt 1 ]]; then
    echo "runtime fake no produjo ACK/artefacto: ack_count=$ack_count artifact_count=$artifact_count" >&2
    exit 1
  fi

  write_summary "$job_ref" "$replay_ref" "$manual_receipt_ref" "$external_run_ref" "$stats_response" "$operational_evidence"
  cat "$out_dir/summary.txt"
}

main "$@"
