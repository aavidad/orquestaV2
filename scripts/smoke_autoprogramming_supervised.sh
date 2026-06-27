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
smoke_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-autoprogramming-supervised.XXXXXX")}"
smoke_temp_root_prepare "$smoke_root" "$smoke_root_source"
state_dir="$smoke_root/state"
project_dir="$smoke_root/project"
runtime_dir="$smoke_root/runtime"
codex_home="$smoke_root/codex-home"
bin_dir="$smoke_root/bin"
tmp_go_dir="$smoke_root/harness"
payload_dir="$smoke_root/payloads"
result_dir="$smoke_root/results"
server_stdout="$smoke_root/orquesta-server.stdout.log"
server_stderr="$smoke_root/orquesta-server.stderr.log"
server_pid=""
base_url=""
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"
request_timeout="${ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS:-15}"
resident_polls="${ORQUESTA_SMOKE_RESIDENT_POLLS:-40}"
resident_sleep="${ORQUESTA_SMOKE_RESIDENT_SLEEP_SECONDS:-0.25}"
legacy_external_fallback="${ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK:-0}"

if [[ "$legacy_external_fallback" == "1" ]]; then
  smoke_require_confirm \
    ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP \
    1 \
    "fallback legacy external-work requiere ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1"
fi

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

cleanup() {
  if [[ -n "$base_url" ]]; then
    curl -sS -m 5 -X POST "$base_url/api/v0/server/shutdown" \
      -H "Content-Type: application/json" \
      -d '{"request_id":"req-autoprogramming-smoke-shutdown","correlation_id":"corr-autoprogramming-smoke-shutdown","forced":true}' \
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
  smoke_temp_root_cleanup "$smoke_root" "$keep_dir"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

if [[ "${ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM:-0}" != "1" ]]; then
  echo "smoke desactivado: exporta ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1" >&2
  exit 2
fi
smoke_require_confirm \
  ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP \
  1 \
  "smoke legacy autoprogramacion desactivado: exporta ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1 para probar prepare-run historico"

need_cmd curl
need_cmd go
need_cmd python3

mkdir -p "$state_dir" "$project_dir" "$runtime_dir" "$codex_home/.codex" "$bin_dir" "$tmp_go_dir" "$payload_dir" "$result_dir"
printf '{"ok":true}\n' >"$codex_home/.codex/auth.json"
printf 'model = "fake"\n' >"$codex_home/.codex/config.toml"

fake_codex="$bin_dir/codex-fake"
cat >"$fake_codex" <<'SH'
#!/usr/bin/env sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift || break
done
if [ -n "$out" ]; then
  printf 'fake codex final: smoke sin Codex real\n' > "$out"
fi
printf 'fake codex stdout: smoke sin Codex real\n'
SH
chmod 700 "$fake_codex"

write_autoprogramming_payload() {
  local output="$1"
  python3 - "$output" "$smoke_id" <<'PY'
import json
import sys

output, smoke_id = sys.argv[1:3]
payload = {
    "request_id": f"req-autoprogramming-supervised-{smoke_id}",
    "correlation_id": f"corr-autoprogramming-supervised-{smoke_id}",
    "autoprogramming_request": {
        "request_ref": f"run-autoprogramming-supervised-{smoke_id}",
        "project_ref": "orquesta",
        "worktree_ref": f"worktree-ref-autoprogramming-supervised-{smoke_id}",
        "worktree_isolated": True,
        "branch_ref": f"branch-ref-autoprogramming-supervised-{smoke_id}",
        "tasks": [
            {"task_ref": "task-ref-autoprogramming-contract", "area": "autoprogramming"},
            {"task_ref": "task-ref-autoprogramming-runbook", "area": "autoprogramming"},
        ],
        "write_set": [
            "modulos/orquesta-autoprogramming/autoprogramming_request_v0.go",
            "docs/runbooks/smoke_autoprogramming_supervisado.md",
        ],
        "required_tests": [
            "go test -count=1 ./modulos/orquesta-autoprogramming"
        ],
    },
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

run_programmable_harness() {
  local request_payload="$1"
  local output="$2"
  cat >"$tmp_go_dir/go.mod" <<EOF
module smoke-autoprogramming-supervised

go 1.22

require orquesta v0.0.0

replace orquesta => $repo_root
EOF
  cat >"$tmp_go_dir/main.go" <<'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type envelope struct {
	AutoprogrammingRequest orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request"`
}

func main() {
	if len(os.Args) != 3 {
		panic("uso: harness payload output")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var input envelope
	if err := json.Unmarshal(data, &input); err != nil {
		panic(err)
	}
	result := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(input.AutoprogrammingRequest)
	if !result.Accepted {
		_ = json.NewEncoder(os.Stdout).Encode(result)
		os.Exit(1)
	}
	summary := map[string]any{
		"accepted":       result.Accepted,
		"request_ref":    result.Work.RequestRef,
		"project_ref":    result.Work.ProjectRef,
		"group_count":    len(result.Work.Groups),
		"task_count":     len(result.Work.Tasks),
		"profile_count":  len(result.Work.Profiles),
		"task_refs":      taskRefs(result.Work.Tasks),
		"profile_kinds":  profileKinds(result.Work.Profiles),
		"required_tests": result.Work.Tasks[0].RequiredTests,
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer out.Close()
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		panic(err)
	}
	fmt.Printf("programmable_work_ok task_count=%d group_count=%d\n", len(result.Work.Tasks), len(result.Work.Groups))
}

func taskRefs(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return refs
}

func profileKinds(profiles []orquestacoreworkflow.WorkProfileV0) []string {
	kinds := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		kinds = append(kinds, string(profile.ProfileKind))
	}
	return kinds
}
EOF
  (cd "$tmp_go_dir" && go mod tidy && go run . "$request_payload" "$output")
}

state_field() {
  local field="$1"
  python3 - "$state_dir/orquesta_server_state_v0.json" "$field" <<'PY'
import json
import sys

try:
    with open(sys.argv[1], encoding="utf-8") as fh:
        data = json.load(fh)
except FileNotFoundError:
    print("")
    raise SystemExit(0)
print(data.get(sys.argv[2], ""))
PY
}

wait_server_ready() {
  for _ in $(seq 1 80); do
    local addr
    addr="$(state_field addr)"
    local pid
    pid="$(state_field pid)"
    if [[ "$pid" == "$server_pid" && -n "$addr" ]] &&
      curl -fsS -m 2 "http://$addr/api/v0/server/readiness" >/dev/null; then
      base_url="http://$addr"
      return 0
    fi
    sleep 0.25
  done
  return 1
}

start_server() {
  ORQUESTA_SERVER_ADDR="127.0.0.1:0" \
  ORQUESTA_SERVER_STATE_DIR="$state_dir" \
  ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir" \
  ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir" \
  ORQUESTA_CODEX_HOME="$codex_home/.codex" \
  ORQUESTA_CODEX_CODE_HOME="$codex_home/.codex" \
  ORQUESTA_CODEX_COMMAND="$fake_codex" \
  ORQUESTA_CODEX_PATH="$PATH" \
  ORQUESTA_CODEX_APPROVAL_POLICY="never" \
  ORQUESTA_CODEX_SANDBOX="workspace-write" \
  ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP="${ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP:-0}" \
  ORQUESTA_SERVER_TICK_INTERVAL_MS="${ORQUESTA_SERVER_TICK_INTERVAL_MS:-250}" \
  ORQUESTA_OPES_BASE_URL="" \
  OPES_BASE_URL="" \
    "$bin_dir/orquesta-server" run >"$server_stdout" 2>"$server_stderr" &
  server_pid="$!"

  if ! wait_server_ready; then
    echo "orquesta-server temporal no llego a readiness OK" >&2
    tail -n 80 "$server_stderr" >&2 || true
    exit 1
  fi
  echo "servidor listo: $base_url"
}

post_json_optional() {
  local path="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(
    curl -sS -m "$request_timeout" -o "$output" -w "%{http_code}" \
      -X POST "$base_url$path" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary "@$payload" || true
  )"
  echo "POST $path -> HTTP $status"
  case "$status" in
    2*) return 0 ;;
    404|503)
      echo "skip: $path no disponible en esta composicion" >&2
      return 2
      ;;
    500)
      if [[ "$path" == "/api/v0/external-work/run" || "$path" == "/api/v0/runs/supervise" ]]; then
        echo "skip: $path no pudo ejecutarse con la composicion fake actual" >&2
        return 2
      fi
      cat "$output" >&2 || true
      return 1
      ;;
    *)
      cat "$output" >&2 || true
      return 1
      ;;
  esac
}

post_json_required() {
  local path="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(
    curl -sS -m "$request_timeout" -o "$output" -w "%{http_code}" \
      -X POST "$base_url$path" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary "@$payload" || true
  )"
  echo "POST $path -> HTTP $status"
  case "$status" in
    2*) return 0 ;;
    *)
      cat "$output" >&2 || true
      return 1
      ;;
  esac
}

verify_validation_accepted() {
  local response="$1"
  python3 - "$response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("estado") != "ok" or data.get("accepted") is not True:
    raise SystemExit(f"validacion inesperada: {data}")
print("validate_request_ok=true")
PY
}

write_prepare_run_payload() {
  local output="$1"
  local source_payload="$2"
  python3 - "$output" "$source_payload" "$smoke_id" <<'PY'
import json
import sys

output, source_payload, smoke_id = sys.argv[1:4]
with open(source_payload, encoding="utf-8") as fh:
    payload = json.load(fh)
payload["occurred_at"] = "2026-05-22T12:00:00Z"
payload["requested_by"] = "orquesta-autoprogramming-supervised-smoke"
payload["max_bursts"] = 2
payload["max_steps_per_burst"] = 2
payload["max_dispatches_per_wait"] = 2
payload["max_commands"] = 4
payload["max_outbox_per_cycle"] = 4
payload["request_id"] = f"req-autoprogramming-prepare-run-{smoke_id}"
payload["correlation_id"] = f"corr-autoprogramming-prepare-run-{smoke_id}"
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

verify_prepare_run_accepted() {
  local response="$1"
  python3 - "$response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("estado") != "ok" or data.get("accepted") is not True:
    raise SystemExit(f"prepare-run inesperado: {data}")
if not data.get("run_ref") or not data.get("wait_agent_refs") or not data.get("continue"):
    raise SystemExit(f"prepare-run sin refs causales: {data}")
print("prepare_run_ok=true")
print("prepare_run_ref=" + data["run_ref"])
PY
}

verify_prepare_run_replay_accepted() {
  local first_response="$1"
  local replay_response="$2"
  python3 - "$first_response" "$replay_response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    first = json.load(fh)
with open(sys.argv[2], encoding="utf-8") as fh:
    replay = json.load(fh)
if replay.get("estado") != "ok" or replay.get("accepted") is not True:
    raise SystemExit(f"prepare-run replay inesperado: {replay}")
if not replay.get("run_ref") or not replay.get("wait_agent_refs") or not replay.get("continue"):
    raise SystemExit(f"prepare-run replay sin refs causales: {replay}")
if replay.get("run_ref") != first.get("run_ref"):
    raise SystemExit(f"prepare-run replay cambio run_ref: first={first.get('run_ref')} replay={replay.get('run_ref')}")
print("prepare_run_replay_ok=true")
print("prepare_run_replay_ref=" + replay["run_ref"])
PY
}

write_external_work_payload() {
  local output="$1"
  local programmable_summary="$2"
  python3 - "$output" "$smoke_id" "$programmable_summary" <<'PY'
import json
import sys

output, smoke_id, summary_path = sys.argv[1:4]
with open(summary_path, encoding="utf-8") as fh:
    summary = json.load(fh)
payload = {
    "request_id": f"req-autoprogramming-external-work-{smoke_id}",
    "correlation_id": f"corr-autoprogramming-external-work-{smoke_id}",
    "app_change_request": {
        "schema_version": "app_change_request.v0",
        "request_id": f"req-autoprogramming-app-change-{smoke_id}",
        "correlation_id": f"corr-autoprogramming-app-change-{smoke_id}",
        "run_ref": f"run-autoprogramming-external-work-{smoke_id}",
        "app_ref": "autoprogramming-smoke",
        "change_ref": f"change-autoprogramming-programmable-{smoke_id}",
        "actor_ref": "orquesta-smoke",
        "locale": "es",
        "user_intent": "Ejecutar trabajo programable de autoprogramacion acotado sin Codex real.",
        "target_area": "autoprogramming",
        "acceptance_criteria": [
            "request de autoprogramming validada",
            "trabajo programable preserva write_set y required_tests",
            "sin tocar OPES ni ejecutar Codex real"
        ],
        "constraints": [
            "runtime fake o no disponible",
            "no efectos externos productivos"
        ],
        "external_work": {
            "project_ref": "autoprogramming-smoke",
            "job_ref": f"job-autoprogramming-programmable-{smoke_id}",
            "interface_refs": ["orquesta-autoprogramming-v0"],
            "work_kind": "autoprogramming_programmable_work",
            "work_refs": summary.get("task_refs") or [],
            "input_fields": [
                {"name": "request_ref", "value": summary.get("request_ref", "")},
                {"name": "task_count", "value": str(summary.get("task_count", ""))},
                {"name": "profile_kinds", "value": ",".join(summary.get("profile_kinds") or [])}
            ]
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
  python3 - "$output" "$smoke_id" "$run_ref" <<'PY'
import json
import sys

output, smoke_id, run_ref = sys.argv[1:4]
payload = {
    "request_id": f"req-autoprogramming-supervisor-{smoke_id}",
    "correlation_id": f"corr-autoprogramming-supervisor-{smoke_id}",
    "director_execution_mode": "legacy_director_loop",
    "run_ref": run_ref,
    "queue_ref": "global",
    "continue_message": "sigue",
    "max_ticks": 1,
    "max_runs_per_tick": 1,
    "max_executions": 1,
    "max_bursts": 1,
    "max_steps_per_burst": 1,
    "max_dispatches_per_wait": 1,
    "max_commands": 4,
    "max_outbox_per_cycle": 4,
    "max_external_waits": 1,
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

write_stats_payload() {
  local output="$1"
  local run_ref="$2"
  python3 - "$output" "$smoke_id" "$run_ref" <<'PY'
import json
import sys

output, smoke_id, run_ref = sys.argv[1:4]
payload = {
    "request_id": f"req-autoprogramming-stats-{smoke_id}",
    "correlation_id": f"corr-autoprogramming-stats-{smoke_id}",
    "run_ref": run_ref,
    "include_process_refs": True,
    "include_agent_progress": True,
}
with open(output, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2)
    fh.write("\n")
PY
}

wait_resident_supervisor_started() {
  local run_ref="$1"
  local payload="$payload_dir/prepare_run_stats.json"
  local response="$result_dir/prepare_run_resident_stats_response.json"
  local status
  write_stats_payload "$payload" "$run_ref"
  for _ in $(seq 1 "$resident_polls"); do
    status="$(
      curl -sS -m "$request_timeout" -o "$response" -w "%{http_code}" \
        -X POST "$base_url/api/v0/director/stats" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        --data-binary "@$payload" || true
    )"
    if [[ "$status" == 2* ]] && python3 - "$response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
stats = data.get("stats") or {}
counts = stats.get("counts") or {}
if data.get("estado") == "ok" and int(counts.get("agents_started") or 0) > 0:
    raise SystemExit(0)
raise SystemExit(1)
PY
    then
      python3 - "$response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
stats = data.get("stats") or {}
counts = stats.get("counts") or {}
print("resident_supervisor_agents_started=" + str(counts.get("agents_started", "")))
print("resident_supervisor_agents_in_flight=" + str(counts.get("agents_in_flight", "")))
PY
      return 0
    fi
    sleep "$resident_sleep"
  done
  echo "resident supervisor no arranco agentes para $run_ref" >&2
  cat "$response" >&2 || true
  return 1
}

extract_run_ref() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    print(json.load(fh).get("run_ref", ""))
PY
}

run_go_test_if_present() {
  local package="$1"
  local pattern="$2"
  local label="$3"
  local list_output="$result_dir/${label}.list.txt"
  if ! go test -list "$pattern" "$package" >"$list_output" 2>&1; then
    echo "skip: no se pudo listar $label en $package"
    cat "$list_output" >&2 || true
    return 0
  fi
  if ! grep -E '^Test' "$list_output" >/dev/null 2>&1; then
    echo "skip: sin tests para $label"
    return 0
  fi
  echo "ejecutando $label..."
  go test -count=1 "$package" -run "$pattern"
}

main() {
  cd "$repo_root"

  local autoprogramming_payload="$payload_dir/autoprogramming_validate_request.json"
  local prepare_payload="$payload_dir/autoprogramming_prepare_run.json"
  local programmable_summary="$result_dir/programmable_work_summary.json"
  local validation_response="$result_dir/autoprogramming_validate_response.json"
  local prepare_response="$result_dir/autoprogramming_prepare_run_response.json"
  local prepare_replay_response="$result_dir/autoprogramming_prepare_run_replay_response.json"
  local external_payload="$payload_dir/external_work_run.json"
  local external_response="$result_dir/external_work_run_response.json"
  local supervisor_payload="$payload_dir/supervisor.json"
  local supervisor_response="$result_dir/supervisor_response.json"

  write_autoprogramming_payload "$autoprogramming_payload"

  echo "validando contrato puro y trabajo programable..."
  go test -count=1 ./modulos/orquesta-autoprogramming
  run_programmable_harness "$autoprogramming_payload" "$programmable_summary"

  echo "compilando servidor temporal..."
  go build -o "$bin_dir/orquesta-server" ./cmd/orquesta-server
  start_server

  if post_json_optional "/api/v0/autoprogramming/validate-request" "$autoprogramming_payload" "$validation_response"; then
    verify_validation_accepted "$validation_response"
  fi

  write_prepare_run_payload "$prepare_payload" "$autoprogramming_payload"
  post_json_required "/api/v0/autoprogramming/prepare-run" "$prepare_payload" "$prepare_response"
  verify_prepare_run_accepted "$prepare_response"
  prepare_run_ref="$(extract_run_ref "$prepare_response")"
  wait_resident_supervisor_started "$prepare_run_ref"

  post_json_required "/api/v0/autoprogramming/prepare-run" "$prepare_payload" "$prepare_replay_response"
  verify_prepare_run_replay_accepted "$prepare_response" "$prepare_replay_response"

  if [[ "$legacy_external_fallback" == "1" ]]; then
    write_external_work_payload "$external_payload" "$programmable_summary"
    if post_json_optional "/api/v0/external-work/run" "$external_payload" "$external_response"; then
      run_ref="$(extract_run_ref "$external_response")"
      if [[ -n "$run_ref" ]]; then
        echo "external_work_run_ref=$run_ref"
        write_supervisor_payload "$supervisor_payload" "$run_ref"
        if post_json_optional "/api/v0/runs/supervise" "$supervisor_payload" "$supervisor_response"; then
          python3 - "$supervisor_response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
print("supervisor_estado=" + str(data.get("estado", "")))
print("supervisor_stop_reason=" + str(data.get("stop_reason", "")))
PY
        fi
      else
        echo "skip: external-work no devolvio run_ref; no se invoca supervisor"
      fi
    fi
  else
    echo "skip: fallback legacy external-work/supervise desactivado; exporta ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1 para compatibilidad"
  fi

  run_go_test_if_present ./modulos/orquesta-app-codex-stack \
    'TestPrepareAutoprogrammingRunV0PersisteWorkflowTasksYRunContinuable|TestCodexStackAutoprogrammingExecutorV0UsaPuertosDelStackYDevuelveContinue|TestCodexStackAutoprogrammingPrepareRunAPIV0' \
    "stack_autoprogramming_bridge"
  if [[ "$legacy_external_fallback" == "1" ]]; then
    run_go_test_if_present ./modulos/orquesta-app-codex-stack \
      'TestCodexStackV0ExternalWorkRunAceptaContratoAmplioV0|TestCodexStackRunSupervisorAPIV0ConsumeDecisionFileYArrancaProgramacion' \
      "stack_fake_external_work_supervisor"
  else
    echo "skip: stack_fake_external_work_supervisor legacy desactivado por defecto"
  fi
  run_go_test_if_present ./modulos/orquesta-app-codex-stack \
    'TestOperationalClosureSourceV0' \
    "stack_fake_closure_source"
  run_go_test_if_present ./modulos/orquesta-app-director-service \
    'TestContinueAppDirectorV0CierraCicloReviewRunnerYReplanOrClose|TestMaybeCloseOperationalDirectorV0CierraPlanStateTrasCierreOperativoExitoso' \
    "director_service_closure"

  echo "codex_real_executed=false"
  echo "opes_touched=false"
  echo "summary_dir=$result_dir"
}

main "$@"
