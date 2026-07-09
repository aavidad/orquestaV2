#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"

SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_ROOT="${SMOKE_ROOT:-${TMPDIR:-/tmp}/orquesta-opes-lifecycle-real-$SMOKE_ID}"
SMOKE_OUT_DIR="$SMOKE_ROOT/out"
STATE_DIR="$SMOKE_ROOT/state"
COURSE_ROOT="$SMOKE_ROOT/course"
REGISTRY_PATH="$SMOKE_ROOT/REGISTRO_TRABAJO_TEMAS_OPES.json"
APP_CHANGE_STATE="$STATE_DIR/run-state/app_change_v0.json"
RUNS_DIR="$STATE_DIR/orchestration-state/runs"
FAKE_ORQUESTA_PID=""
DERIVATIVES_OUT="$SMOKE_OUT_DIR/derivatives"
FINALPKG_OUT="$SMOKE_OUT_DIR/finalpkg"
KEEP_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-1}"

usage() {
  cat <<'EOF'
Uso:
  scripts/smoke_opes_lifecycle_real.sh [--help]

Ejecuta el smoke OPES lifecycle acotado. Por defecto usa fixtures/fake server
locales para reproducir las 24 fases sin tocar OPES productivo.

Para una ejecucion con efectos contra OPES real/temporal, no uses este arnes
sin declarar antes, en la composicion externa, las guardas requeridas:
  ORQUESTA_OPES_BASE_URL
  ORQUESTA_BASE_URL
  ORQUESTA_OPES_TEMPORAL_CONFIRM=1
  ORQUESTA_OPES_BRIDGE_CONFIRM=1
  ORQUESTA_OPES_BRIDGE_LIMIT=1
  ORQUESTA_OPES_BRIDGE_JOB_REF o PROGRAM_ID/TOPIC_ID/CORRELATION_ID

Si esas settings faltan, el goal debe cerrar blocked con
missing_required_settings en vez de improvisar una ejecucion real.
EOF
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
esac

cleanup() {
  if [[ -n "$FAKE_ORQUESTA_PID" ]]; then
    kill "$FAKE_ORQUESTA_PID" >/dev/null 2>&1 || true
    wait "$FAKE_ORQUESTA_PID" >/dev/null 2>&1 || true
  fi
  smoke_temp_root_cleanup "$SMOKE_ROOT" "$KEEP_DIR"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

require_tools() {
  smoke_require_tools bash curl go jq python3 tee
}

start_fake_orquesta_for_finalpkg() {
  local server_py="$FINALPKG_OUT/fake_orquesta_finalpkg.py"
  local url_file="$FINALPKG_OUT/fake_orquesta_url.txt"
  mkdir -p "$FINALPKG_OUT"
  cat >"$server_py" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

url_file = sys.argv[1]
requests_file = sys.argv[2]

class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        return

    def do_POST(self):
        if self.path != "/api/v0/external-work/run":
            self.send_json(404, {"error": "unexpected_path", "path": self.path})
            return
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length)
        try:
            payload = json.loads(body.decode("utf-8") or "{}")
        except Exception as exc:
            self.send_json(400, {"error": "invalid_json", "detail": str(exc)})
            return
        with open(requests_file, "a", encoding="utf-8") as fh:
            fh.write(json.dumps(payload, ensure_ascii=False) + "\n")
        request = payload.get("external_work_run_request") or {}
        run_ref = request.get("run_ref") or "run-ref-finalpkg-missing"
        self.send_json(200, {
            "estado": "ok",
            "run_ref": run_ref,
            "route_policy": "goal_first",
            "director_execution_mode": "goal_first",
            "goal_ref": "goal-ref-" + run_ref,
            "external_goal_ref": "thread-ref-" + run_ref,
            "domain_receipt_refs": ["domain-receipt-ref-" + run_ref],
            "evidence_refs": ["evidence-ref-finalpkg-fake-orquesta-submit"],
        })

    def send_json(self, status, body):
        raw = json.dumps(body).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

server = HTTPServer(("127.0.0.1", 0), Handler)
with open(url_file, "w", encoding="utf-8") as fh:
    fh.write(f"http://127.0.0.1:{server.server_port}\n")
server.serve_forever()
PY
  : >"$FINALPKG_OUT/finalpkg_requests.jsonl"
  python3 "$server_py" "$url_file" "$FINALPKG_OUT/finalpkg_requests.jsonl" \
    >"$FINALPKG_OUT/fake_orquesta_server.log" 2>&1 &
  FAKE_ORQUESTA_PID="$!"
  for _ in $(seq 1 50); do
    if [[ -s "$url_file" ]]; then
      cat "$url_file"
      return 0
    fi
    sleep 0.1
  done
  echo "fake Orquesta finalpkg no arranco" >&2
  exit 1
}

write_finalpkg_fixture() {
  mkdir -p "$COURSE_ROOT/tema_001/paquete_final" "$COURSE_ROOT/tema_002/paquete_final" \
    "$STATE_DIR/run-state" "$RUNS_DIR"
  python3 - "$REGISTRY_PATH" "$APP_CHANGE_STATE" "$COURSE_ROOT" <<'PY'
import json
import sys
from pathlib import Path

registry_path, state_path, course_root = map(Path, sys.argv[1:])
registry_path.write_text(json.dumps({
    "courses": {
        "course-ref-lifecycle-live": {
            "topics": {
                "001": {},
                "002": {},
            },
        },
    },
}, ensure_ascii=False, indent=2), encoding="utf-8")
template = {
    "schema_version": "app_change_request.v0",
    "request_id": "request-ref-template-finalpkg-001",
    "user_intent": "Finalizar paquete local verificable del tema 001",
    "acceptance_criteria": ["paquete final verificable"],
    "external_work": {
        "job_ref": "job-ref-opes-a1-t001-finalpkg-20260612",
        "work_kind": "finalize_temario_package",
        "input_fields": [
            {"name": "course_id", "value": "course-ref-lifecycle-live"},
            {"name": "topic_id", "value": "001"},
            {"name": "expected_artifact_type", "value": "completed_syllabus_package"},
        ],
    },
}
state_path.write_text(json.dumps({
    "records": [{"request": {"run_ref": "run-ref-opes-a1-t001-finalpkg-20260612", **template}}],
}, ensure_ascii=False, indent=2), encoding="utf-8")
PY
}

run_derivatives_lifecycle() {
  mkdir -p "$DERIVATIVES_OUT"
  (
    cd "$repo_root"
    ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
    ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
    ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
    ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
    ORQUESTA_OPES_BRIDGE_MAX_TICKS="${ORQUESTA_OPES_LIFECYCLE_MAX_TICKS:-40}" \
    ORQUESTA_OPES_BRIDGE_LIMIT=1 \
    ORQUESTA_KEEP_SMOKE_DIR=1 \
    SMOKE_ID="$SMOKE_ID-derivatives" \
    SMOKE_OUT_DIR="$DERIVATIVES_OUT" \
    scripts/smoke_opes_derivatives_rest.sh
  ) | tee "$DERIVATIVES_OUT/stdout.log"
  jq -e '.sequence_complete == true and (.issues | length == 0) and (.covered_work_kinds | index("finalize_temario_package"))' \
    "$DERIVATIVES_OUT/goal_receipts_manifest.json" >/dev/null
}

run_finalpkg_live_config() {
  local fake_orquesta_url
  fake_orquesta_url="$(start_fake_orquesta_for_finalpkg)"
  write_finalpkg_fixture
  (
    cd "$repo_root"
    ORQUESTA_DRY_RUN=0 \
    ORQUESTA_OPES_REGISTRY_FINALPKG_CONFIRM=1 \
    ORQUESTA_OPES_COURSE_ID=course-ref-lifecycle-live \
    ORQUESTA_TEMPLATE_RUN_REF=run-ref-opes-a1-t001-finalpkg-20260612 \
    ORQUESTA_TEMPLATE_TOPIC_ID=001 \
    python3 scripts/opes_a1_finalpkg_registry_launcher.py \
      --registry "$REGISTRY_PATH" \
      --course-id course-ref-lifecycle-live \
      --course-root "$COURSE_ROOT" \
      --app-change-state "$APP_CHANGE_STATE" \
      --orchestration-runs-dir "$RUNS_DIR" \
      --template-run-ref run-ref-opes-a1-t001-finalpkg-20260612 \
      --template-topic-id 001 \
      --orquesta-base-url "$fake_orquesta_url" \
      --queue-ref queue-ref-lifecycle-live \
      --batch-size 1 \
      --max-in-flight 1
  ) | tee "$FINALPKG_OUT/launcher_summary.json"
  jq -e '.dry_run == false and .selected[0].topic_id == "002" and .selected[0].status == "ok"' \
    "$FINALPKG_OUT/launcher_summary.json" >/dev/null
  python3 - "$FINALPKG_OUT/finalpkg_requests.jsonl" "$FINALPKG_OUT/finalpkg_assertions.json" <<'PY'
import json
import sys
from pathlib import Path

requests = [json.loads(line) for line in Path(sys.argv[1]).read_text(encoding="utf-8").splitlines() if line.strip()]
if len(requests) != 1:
    raise SystemExit(f"unexpected finalpkg request count: {len(requests)}")
request = requests[0]["external_work_run_request"]
app_change = request["app_change_request"]
external = app_change["external_work"]
fields = external.get("input_fields") or []
field_values = {item.get("name"): item.get("value") for item in fields if isinstance(item, dict)}
assertions = {
    "schema_version": "orquesta_opes_lifecycle_finalpkg_assertions.v0",
    "dry_run": False,
    "run_ref": request.get("run_ref"),
    "job_ref": external.get("job_ref"),
    "work_kind": external.get("work_kind"),
    "queue_ref": request.get("queue_ref"),
    "live_config_refs": {
        "course_id": "course-ref-lifecycle-live",
        "template_run_ref": "run-ref-opes-a1-t001-finalpkg-20260612",
        "template_topic_id": "001",
    },
    "topic_id_field": field_values.get("topic_id"),
    "expected_artifact_type": field_values.get("expected_artifact_type"),
    "status": "valid",
}
checks = [
    request.get("run_ref") == "run-ref-opes-a1-t002-finalpkg-20260612",
    external.get("job_ref") == "job-ref-opes-a1-t002-finalpkg-20260612",
    external.get("work_kind") == "finalize_temario_package",
    request.get("queue_ref") == "queue-ref-lifecycle-live",
    field_values.get("topic_id") == "002",
    field_values.get("expected_artifact_type") == "completed_syllabus_package",
]
if not all(checks):
    assertions["status"] = "invalid"
    Path(sys.argv[2]).write_text(json.dumps(assertions, ensure_ascii=False, indent=2), encoding="utf-8")
    raise SystemExit("finalpkg request assertions failed")
Path(sys.argv[2]).write_text(json.dumps(assertions, ensure_ascii=False, indent=2), encoding="utf-8")
PY
}

run_settlement_contract() {
  (
    cd "$repo_root"
    go test -count=1 ./modulos/orquesta-opes-director \
      -run 'TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro'
  ) | tee "$SMOKE_OUT_DIR/settlement_go_test.log"
  grep -q '^ok[[:space:]]\+orquesta/modulos/orquesta-opes-director' "$SMOKE_OUT_DIR/settlement_go_test.log"
}

assert_no_residual_processes() {
  local pids=()
  [[ -n "$FAKE_ORQUESTA_PID" ]] && pids+=("$FAKE_ORQUESTA_PID")
  local pid
  for pid in "${pids[@]}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
      wait "$pid" >/dev/null 2>&1 || true
    fi
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "proceso residual smoke vivo: $pid" >&2
      exit 1
    fi
  done
  FAKE_ORQUESTA_PID=""
}

write_lifecycle_result() {
  python3 - "$SMOKE_OUT_DIR" <<'PY'
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
derivatives = json.loads((root / "derivatives" / "goal_receipts_manifest.json").read_text(encoding="utf-8"))
finalpkg = json.loads((root / "finalpkg" / "finalpkg_assertions.json").read_text(encoding="utf-8"))
result = {
    "schema_version": "orquesta_opes_lifecycle_smoke_result.v0",
    "status": "passed",
    "derivatives_sequence_complete": bool(derivatives.get("sequence_complete")),
    "derivatives_covered_work_kinds": derivatives.get("covered_work_kinds") or [],
    "finalpkg_dry_run": finalpkg.get("dry_run"),
    "finalpkg_run_ref": finalpkg.get("run_ref"),
    "finalpkg_live_config_refs": finalpkg.get("live_config_refs") or {},
    "settlement_status": "settled_final",
    "settlement_reason": "final_package_closure_evidence_complete",
    "no_residual_processes": True,
    "evidence_paths": [
        "derivatives/goal_receipts_manifest.json",
        "finalpkg/finalpkg_assertions.json",
        "settlement_go_test.log",
    ],
}
(root / "opes_lifecycle_result.json").write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
print(json.dumps(result, ensure_ascii=False))
PY
}

main() {
  require_tools
  smoke_temp_root_prepare "$SMOKE_ROOT" "${SMOKE_ROOT_SOURCE:-generated}"
  mkdir -p "$SMOKE_OUT_DIR" "$DERIVATIVES_OUT" "$FINALPKG_OUT"
  run_derivatives_lifecycle
  run_finalpkg_live_config
  run_settlement_contract
  assert_no_residual_processes
  write_lifecycle_result | tee "$SMOKE_OUT_DIR/opes_lifecycle_result.stdout.json"
}

main "$@"
