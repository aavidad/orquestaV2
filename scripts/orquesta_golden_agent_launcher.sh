#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
uso: scripts/orquesta_golden_agent_launcher.sh --agent-command CMD [--skill-ref REF] [--stdin-prompt]

Launcher puente opt-in para golden tasks. Consume ORQUESTA_GOLDEN_TASK_* del
harness, genera un prompt/contrato estable y ejecuta CMD con variables
GOLDEN_AGENT_* para que cualquier agente/proveedor escriba result.json.

El launcher sale 0 si pudo escribir result.json, aunque el agente falle; el
estado de fallo queda dentro de result.json para que el evaluador no pierda
diagnostico.
USAGE
}

agent_command=""
stdin_prompt=0
skill_refs=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent-command)
      agent_command="${2:-}"
      shift 2
      ;;
    --skill-ref)
      skill_refs+=("${2:-}")
      shift 2
      ;;
    --stdin-prompt)
      stdin_prompt=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "argumento no soportado: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$agent_command" ]]; then
  usage
  exit 2
fi

task_id="${ORQUESTA_GOLDEN_TASK_ID:?ORQUESTA_GOLDEN_TASK_ID requerido}"
request_path="${ORQUESTA_GOLDEN_TASK_REQUEST:?ORQUESTA_GOLDEN_TASK_REQUEST requerido}"
result_dir="${ORQUESTA_GOLDEN_TASK_RESULT_DIR:?ORQUESTA_GOLDEN_TASK_RESULT_DIR requerido}"
result_file="$result_dir/result.json"
prompt_path="$result_dir/agent_prompt.md"
stdout_path="$result_dir/agent_stdout.log"
stderr_path="$result_dir/agent_stderr.log"
manifest_path="$result_dir/agent_launcher_manifest.json"

mkdir -p "$result_dir"

python3 - "$request_path" "$prompt_path" "$manifest_path" "$task_id" "$result_dir" "${skill_refs[*]-}" <<'PY'
import json
from pathlib import Path
import sys

request_path = Path(sys.argv[1])
prompt_path = Path(sys.argv[2])
manifest_path = Path(sys.argv[3])
task_id = sys.argv[4]
result_dir = Path(sys.argv[5])
skill_refs = [value for value in sys.argv[6].split() if value]

packet = json.loads(request_path.read_text(encoding="utf-8"))
task = packet.get("task") if isinstance(packet, dict) else {}
if not isinstance(task, dict):
    task = {}

def bullets(values):
    out = []
    for value in values if isinstance(values, list) else []:
        if isinstance(value, dict):
            out.append("- " + json.dumps(value, ensure_ascii=False, sort_keys=True))
        else:
            out.append("- " + str(value))
    return "\n".join(out) if out else "- none"

objective = task.get("objective") or ""
prompt = f"""# Orquesta Golden Task Contract

task_id: {task.get('task_id') or task_id}
task_class: {task.get('task_class') or ''}
task_set_ref: {packet.get('task_set_ref') if isinstance(packet, dict) else ''}

## Objective

{objective}

## Write Set

{bullets(task.get('write_set'))}

## Expected Files

{bullets(task.get('expected_files'))}

## Required Tests

{bullets(task.get('required_tests'))}

## Forbidden Path Prefixes

{bullets((task.get('forbidden_path_prefixes') or []) + (task.get('global_forbidden_path_prefixes') or []))}

## Skill Refs

{bullets(skill_refs)}

## Output Contract

Write JSON to `$GOLDEN_AGENT_RESULT_PATH`.
Required top-level fields:

- `task_id`: `{task.get('task_id') or task_id}`
- `status`: `complete` or `failed`
- `touched_files`: relative paths only
- `artifact_paths`: relative paths only
- `tests`: list of objects with `command` and `status`
- `evidence`: object with compact refs or notes
- optional `metrics`: token/diff/rework fields if known

Keep the diff inside the write-set. Do not touch OPES production paths. Do not
add helpers, abstractions, files, config, tests or docs unless the task needs
them to satisfy this contract. If blocked, write `status=failed` with a compact
reason and evidence instead of inventing a green result.
"""

prompt_path.write_text(prompt, encoding="utf-8")
manifest = {
    "schema_version": "orquesta_golden_agent_launcher_manifest.v0",
    "task_id": task.get("task_id") or task_id,
    "request_path": str(request_path),
    "prompt_path": str(prompt_path),
    "result_path": str(result_dir / "result.json"),
    "skill_refs": skill_refs,
}
manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

set +e
if [[ "$stdin_prompt" -eq 1 ]]; then
  GOLDEN_AGENT_TASK_ID="$task_id" \
  GOLDEN_AGENT_REQUEST_PATH="$request_path" \
  GOLDEN_AGENT_PROMPT_PATH="$prompt_path" \
  GOLDEN_AGENT_RESULT_DIR="$result_dir" \
  GOLDEN_AGENT_RESULT_PATH="$result_file" \
  GOLDEN_AGENT_SKILL_REFS="$(IFS=,; echo "${skill_refs[*]-}")" \
  bash -lc "$agent_command" <"$prompt_path" >"$stdout_path" 2>"$stderr_path"
else
  GOLDEN_AGENT_TASK_ID="$task_id" \
  GOLDEN_AGENT_REQUEST_PATH="$request_path" \
  GOLDEN_AGENT_PROMPT_PATH="$prompt_path" \
  GOLDEN_AGENT_RESULT_DIR="$result_dir" \
  GOLDEN_AGENT_RESULT_PATH="$result_file" \
  GOLDEN_AGENT_SKILL_REFS="$(IFS=,; echo "${skill_refs[*]-}")" \
  bash -lc "$agent_command" >"$stdout_path" 2>"$stderr_path"
fi
agent_exit_code="$?"
set -e

python3 - "$result_file" "$task_id" "$agent_exit_code" "$prompt_path" "$stdout_path" "$stderr_path" "$manifest_path" "${skill_refs[*]-}" <<'PY'
import json
from pathlib import Path
import sys

result_path = Path(sys.argv[1])
task_id = sys.argv[2]
agent_exit_code = int(sys.argv[3])
prompt_path = Path(sys.argv[4])
stdout_path = Path(sys.argv[5])
stderr_path = Path(sys.argv[6])
manifest_path = Path(sys.argv[7])
skill_refs = [value for value in sys.argv[8].split() if value]

issues = []
if result_path.is_file():
    try:
        payload = json.loads(result_path.read_text(encoding="utf-8"))
        if not isinstance(payload, dict):
            payload = {}
            issues.append("result_json_not_object")
    except json.JSONDecodeError:
        payload = {}
        issues.append("result_json_invalid")
else:
    payload = {}
    issues.append("result_json_missing")

payload.setdefault("schema_version", "orquesta_golden_agent_result.v0")
payload.setdefault("task_id", task_id)
payload.setdefault("touched_files", [])
payload.setdefault("artifact_paths", [])
payload.setdefault("tests", [])
evidence = payload.get("evidence")
if not isinstance(evidence, dict):
    evidence = {}

if agent_exit_code != 0:
    issues.append("agent_command_failed")
if not payload.get("status") or payload.get("status") == "complete" and issues:
    payload["status"] = "failed" if issues else "complete"

evidence["golden_agent_launcher"] = {
    "schema_version": "orquesta_golden_agent_launcher_evidence.v0",
    "agent_exit_code": agent_exit_code,
    "prompt_path": str(prompt_path),
    "stdout_path": str(stdout_path),
    "stderr_path": str(stderr_path),
    "manifest_path": str(manifest_path),
    "skill_refs": skill_refs,
    "issues": issues,
}
payload["evidence"] = evidence
payload["agent_exit_code"] = agent_exit_code
if issues:
    payload["launcher_issues"] = issues

result_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

exit 0
