#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"
smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
smoke_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-codex-required-test-smoke.XXXXXX")}"
smoke_temp_root_prepare "$smoke_root" "$smoke_root_source"
state_dir="$smoke_root/state"
project_dir="$smoke_root/project"
runtime_dir="$smoke_root/runtime"
codex_home="$smoke_root/codex-home"
output_dir="$smoke_root/required-test-output"
overlay_dir="$smoke_root/overlay"
bin_dir="$smoke_root/bin"
result_file="$smoke_root/smoke_result.json"
server_stdout="$smoke_root/orquesta-server.stdout.log"
server_stderr="$smoke_root/orquesta-server.stderr.log"
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
server_pid=""
base_url=""

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

resolve_command() {
  local raw="$1"
  if [[ -z "$raw" ]]; then
    return 1
  fi
  if [[ "$raw" == /* ]]; then
    printf '%s' "$raw"
    return 0
  fi
  command -v "$raw"
}

cleanup() {
  smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"
  smoke_temp_root_cleanup "$smoke_root" "$keep_dir"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

if [[ "${ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_CONFIRM:-0}" != "1" ]]; then
  echo "smoke desactivado: exporta ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_CONFIRM=1" >&2
  exit 2
fi

need_cmd go
need_cmd python3
need_cmd curl

codex_command="$(resolve_command "${ORQUESTA_CODEX_COMMAND:-}")" || {
  echo "falta ORQUESTA_CODEX_COMMAND con ruta/comando Codex real; no se ejecutara, solo se valida configuracion" >&2
  exit 2
}
if [[ ! -x "$codex_command" ]]; then
  echo "ORQUESTA_CODEX_COMMAND no es ejecutable: $codex_command" >&2
  exit 2
fi
go_command="$(resolve_command "${ORQUESTA_REQUIRED_TEST_GO_COMMAND:-go}")" || {
  echo "no se pudo resolver go para ORQUESTA_REQUIRED_TEST_GO_COMMAND" >&2
  exit 127
}

mkdir -p "$state_dir" "$project_dir" "$runtime_dir" "$codex_home/.codex" "$output_dir" "$overlay_dir" "$bin_dir"

cat >"$project_dir/go.mod" <<'EOF'
module example.com/orquesta-required-test-smoke

go 1.22
EOF
cat >"$project_dir/calc.go" <<'EOF'
package calc

func Add(a int, b int) int {
	return a + b
}
EOF
cat >"$project_dir/calc_test.go" <<'EOF'
package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatalf("Add fallo")
	}
}
EOF

overlay_test="$overlay_dir/smoke_codex_required_test_runner_state_file_overlay_test.go"
overlay_json="$overlay_dir/overlay.json"
overlay_target="$repo_root/cmd/orquesta-server/smoke_codex_required_test_runner_state_file_overlay_test.go"

cat >"$overlay_test" <<'EOF'
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestSmokeCodexRequiredTestRunnerStateFileOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_CONFIRM")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_CONFIRM=1")
	}
	if err := validateCodexCommandAvailableV0(); err != nil {
		t.Fatalf("config Codex invalida: %v", err)
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatalf("este smoke no debe cablear OPES")
	}

	ctx := context.Background()
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("RequiredTestRunner no cableado")
	}

	runRef := "run-smoke-required-test-state-file-001"
	planRef := "plan-smoke-required-test-state-file-001"
	taskRef := "task-smoke-required-test-state-file-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-smoke-required-test-state-file-001"
	reviewRequestRef := "review-request-smoke-required-test-state-file-001"
	reviewResultRef := "review-result-smoke-required-test-state-file-001"
	acceptedReviewRef := "accepted-review-smoke-required-test-state-file-001"

	run := serverRequiredTestRunV0(
		runRef,
		taskRef,
		agentRef,
		deliveryRef,
		reviewRequestRef,
		reviewResultRef,
		acceptedReviewRef,
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, runRef, serverRequiredTestEventsV0(
		t,
		runRef,
		taskRef,
		agentRef,
		deliveryRef,
		reviewRequestRef,
		reviewResultRef,
		acceptedReviewRef,
	)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, serverRequiredTestWorkflowTaskV0(runRef, taskRef)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(
		ctx,
		serverRequiredTestPlanStateV0(runRef, planRef, taskRef, agentRef, deliveryRef, reviewResultRef, acceptedReviewRef),
	); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-22T16:30:00Z",
		CorrelationID:              "corr-smoke-required-test-state-file-001",
		MaxBursts:                  1,
		MaxStepsPerBurst:           1,
		MaxDispatchesPerWait:       1,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: status=%s closed=%v validations=%v closures=%v", result.Run.Status, result.Run.ClosedTasks, result.Run.Validations, result.Run.Closures)
	}

	recovered, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0 recovery: %v", err)
	}
	state, err := recovered.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 recovery: %v", err)
	}
	replanStep := serverRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("plan state sin cierre/evidencia: state=%+v replan=%+v", state, replanStep)
	}
	evidence, err := recovered.Stores.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, runRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0 recovery: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].TaskRef != taskRef ||
		evidence[0].DeliveryRef != deliveryRef ||
		evidence[0].ReviewRequestID != reviewRequestRef ||
		evidence[0].ReviewResultRef != reviewResultRef ||
		evidence[0].AcceptedReviewRef != acceptedReviewRef {
		t.Fatalf("evidence invalida: %+v", evidence)
	}
	outputRef := serverRequiredTestOutputRefV0(evidence[0].EvidenceRefs)
	if outputRef == "" {
		t.Fatalf("evidence sin artefacto de salida: %+v", evidence[0])
	}
	outputPath := filepath.Join(strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR")), strings.TrimPrefix(outputRef, "required-test-output-v0/"))
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("leer output required test: %v", err)
	}
	if !strings.Contains(string(content), "status=passed") ||
		!strings.Contains(string(content), "test_command=go test ./...") ||
		strings.Contains(string(content), strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROJECT_WORKDIR"))) ||
		strings.Contains(string(content), strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"))) {
		t.Fatalf("output required test invalido: %q", string(content))
	}

	resultFile := strings.TrimSpace(os.Getenv("ORQUESTA_SMOKE_RESULT_FILE"))
	if resultFile == "" {
		return
	}
	payload := map[string]any{
		"run_ref":       runRef,
		"plan_ref":      planRef,
		"task_ref":      taskRef,
		"state_status":  string(state.Status),
		"run_status":    string(result.Run.Status),
		"evidence_ref":  evidence[0].EvidenceRef,
		"evidence_refs": evidence[0].EvidenceRefs,
		"test_command":  evidence[0].TestCommand,
		"test_status":   string(evidence[0].Status),
		"output_ref":    outputRef,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal smoke result: %v", err)
	}
	if err := os.WriteFile(resultFile, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write smoke result: %v", err)
	}
}
EOF

python3 - "$overlay_target" "$overlay_test" "$overlay_json" <<'PY'
import json
import sys

target, source, overlay = sys.argv[1:]
with open(overlay, "w", encoding="utf-8") as fh:
    json.dump({"Replace": {target: source}}, fh)
PY

export ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_CONFIRM=1
export ORQUESTA_CODEX_COMMAND="$codex_command"
export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir"
export ORQUESTA_CODEX_HOME="$codex_home"
export ORQUESTA_CODEX_CODE_HOME="$codex_home/.codex"
export ORQUESTA_CODEX_PATH="${ORQUESTA_CODEX_PATH:-$PATH}"
export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-workspace-write}"
export ORQUESTA_SERVER_STATE_DIR="$state_dir"
export ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED=1
export ORQUESTA_REQUIRED_TEST_GO_COMMAND="$go_command"
export ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS=""
export ORQUESTA_REQUIRED_TEST_OUTPUT_DIR="$output_dir"
export ORQUESTA_REQUIRED_TEST_ENV="${ORQUESTA_REQUIRED_TEST_ENV:-CGO_ENABLED=0}"
export ORQUESTA_OPES_BASE_URL=""
export OPES_BASE_URL=""
export ORQUESTA_SMOKE_RESULT_FILE="$result_file"

echo "smoke_root=$smoke_root"
echo "codex_config_validated=$codex_command"
echo "required_test_go_command=$go_command"
echo "ejecutando smoke: RequiredTestRunner + state-file; Codex real no se lanzara"

(
  cd "$repo_root"
  go test -count=1 -timeout "${ORQUESTA_CODEX_REQUIRED_TEST_SMOKE_TIMEOUT:-180s}" \
    -overlay "$overlay_json" \
    ./cmd/orquesta-server \
    -run '^TestSmokeCodexRequiredTestRunnerStateFileOptInV0$' \
    -v
)

python3 - "$result_file" "$state_dir" "$output_dir" "$runtime_dir" <<'PY'
import json
import os
import sys

result_file, state_dir, output_dir, runtime_dir = sys.argv[1:]
with open(result_file, encoding="utf-8") as fh:
    result = json.load(fh)
if result.get("run_status") not in {"closed", "cerrada"} or result.get("state_status") != "closed":
    raise SystemExit(f"estado inesperado: {result}")
if result.get("test_status") != "passed" or result.get("test_command") != "go test ./...":
    raise SystemExit(f"evidencia inesperada: {result}")

state_root = os.path.join(state_dir, "orchestration-state")
json_files = []
for root, _, files in os.walk(state_root):
    for name in files:
        if name.endswith(".json"):
            path = os.path.join(root, name)
            with open(path, encoding="utf-8") as fh:
                json.load(fh)
            json_files.append(path)
if not json_files:
    raise SystemExit("state-file sin JSON persistido")

evidence_docs = []
evidence_root = os.path.join(state_root, "required_test_evidence")
for root, _, files in os.walk(evidence_root):
    for name in files:
        if not name.endswith(".json"):
            continue
        path = os.path.join(root, name)
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
        evidence_docs.append(data)
if len(evidence_docs) != 1:
    raise SystemExit(f"numero de evidencias inesperado: {len(evidence_docs)}")
evidence = evidence_docs[0]["evidence"]
if evidence.get("status") != "passed" or evidence.get("test_command") != "go test ./...":
    raise SystemExit(f"documento de evidencia invalido: {evidence}")

output_ref = result["output_ref"]
rel = output_ref.removeprefix("required-test-output-v0/")
output_path = os.path.join(output_dir, rel)
with open(output_path, encoding="utf-8") as fh:
    output = fh.read()
if "status=passed" not in output:
    raise SystemExit("output required-test sin status=passed")

runtime_entries = []
if os.path.isdir(runtime_dir):
    for root, dirs, files in os.walk(runtime_dir):
        runtime_entries.extend(os.path.join(root, item) for item in dirs + files)
if runtime_entries:
    raise SystemExit(f"runtime Codex no deberia tener artefactos: {runtime_entries[:5]}")

print(f"run_ref={result['run_ref']}")
print(f"plan_ref={result['plan_ref']}")
print(f"evidence_ref={result['evidence_ref']}")
print(f"state_json_files={len(json_files)}")
print("codex_executed=false")
print("opes_touched=false")
PY

echo "validando shutdown limpio de orquesta-server temporal..."
(
  cd "$repo_root"
  go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server
)

export ORQUESTA_SERVER_ADDR="127.0.0.1:0"
"$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
server_pid="$!"

state_file="$state_dir/orquesta_server_state_v0.json"
server_ready="0"
for _ in $(seq 1 60); do
  if [[ -s "$state_file" ]]; then
    server_addr="$(python3 - "$state_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("addr", ""))
PY
)"
    if [[ -n "$server_addr" ]] && curl -fsS -m 2 "http://$server_addr/api/v0/server/readiness" >/dev/null; then
      base_url="http://$server_addr"
      server_ready="1"
      break
    fi
  fi
  sleep 0.5
done

if [[ "$server_ready" != "1" ]]; then
  echo "orquesta-server no llego a readiness OK" >&2
  tail -n 80 "$server_stderr" >&2 || true
  exit 1
fi

shutdown_response="$smoke_root/server_shutdown_response.json"
shutdown_status="$(
  curl -sS -m 10 -o "$shutdown_response" -w "%{http_code}" \
    -X POST "$base_url/api/v0/server/shutdown" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d '{"request_id":"req-smoke-required-test-shutdown","correlation_id":"corr-smoke-required-test-shutdown","forced":true}'
)"
if [[ "$shutdown_status" -lt 200 || "$shutdown_status" -gt 299 ]]; then
  echo "shutdown devolvio HTTP $shutdown_status" >&2
  cat "$shutdown_response" >&2 || true
  exit 1
fi

for _ in $(seq 1 30); do
  if ! kill -0 "$server_pid" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done
if kill -0 "$server_pid" >/dev/null 2>&1; then
  server_status=""
  for _ in $(seq 1 30); do
    server_status="$(python3 - "$state_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("status", ""))
PY
)"
    if [[ "$server_status" == "stopped" ]]; then
      break
    fi
    sleep 0.2
  done
  if [[ "$server_status" != "stopped" ]]; then
    echo "diagnostico: state-file aun no marco stopped tras shutdown, status=$server_status" >&2
  fi
  kill -INT "$server_pid" >/dev/null 2>&1 || true
  for _ in $(seq 1 30); do
    if ! kill -0 "$server_pid" >/dev/null 2>&1; then
      break
    fi
    sleep 0.2
  done
fi
if kill -0 "$server_pid" >/dev/null 2>&1; then
  echo "orquesta-server temporal no salio tras INT posterior a shutdown" >&2
  tail -n 80 "$server_stderr" >&2 || true
  exit 1
fi
wait "$server_pid" >/dev/null 2>&1 || true
server_pid=""
base_url=""
echo "server_shutdown=clean"
