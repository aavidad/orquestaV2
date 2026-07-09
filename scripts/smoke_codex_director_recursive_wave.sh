#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$ROOT_DIR/scripts/lib/smoke_common.sh"
smoke_root_source="generated"
if [[ -n "${ORQUESTA_SMOKE_ROOT:-}" ]]; then
  smoke_root_source="env:ORQUESTA_SMOKE_ROOT"
fi
SMOKE_ROOT="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-codex-director-recursive-smoke.XXXXXX")}"
smoke_temp_root_prepare "$SMOKE_ROOT" "$smoke_root_source"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$SMOKE_ROOT/$SMOKE_ID"
PROJECT_DIR="$OUT_DIR/project"
RUNTIME_DIR="$OUT_DIR/runtime/codex-waves/recursive-wave-smoke"
SUMMARY_PATH="$OUT_DIR/summary.json"
FAKE_CODEX="$OUT_DIR/codex-fake"
KEEP_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-0}"

cleanup() {
  smoke_temp_root_cleanup "$SMOKE_ROOT" "$KEEP_DIR"
}
trap cleanup EXIT

mkdir -p "$PROJECT_DIR" "$OUT_DIR"

DRY_RUN_FLAG=""
REAL_MODE=0
REAL_CONFIG_ONLY=0
CODEX_COMMAND="${ORQUESTA_CODEX_COMMAND:-$FAKE_CODEX}"
if [[ "${ORQUESTA_CODEX_DIRECTOR_RECURSIVE_REAL_CONFIG_ONLY:-0}" == "1" ]]; then
  REAL_CONFIG_ONLY=1
fi
if [[ "${ORQUESTA_CODEX_DIRECTOR_RECURSIVE_REAL_CONFIRM:-0}" == "1" || "$REAL_CONFIG_ONLY" == "1" ]]; then
  REAL_MODE=1
  if [[ -z "${ORQUESTA_CODEX_COMMAND:-}" ]]; then
    echo "ORQUESTA_CODEX_COMMAND requerido para modo real/config real" >&2
    exit 2
  fi
  CODEX_COMMAND="$ORQUESTA_CODEX_COMMAND"
  if [[ "$REAL_CONFIG_ONLY" == "1" ]]; then
    DRY_RUN_FLAG="--dry-run"
  fi
else
  if [[ "${ORQUESTA_CODEX_DIRECTOR_RECURSIVE_DRY_RUN:-0}" == "1" ]]; then
    DRY_RUN_FLAG="--dry-run"
  fi
  cat >"$FAKE_CODEX" <<'SH'
#!/usr/bin/env sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift || break
done
input=$(cat)
if [ -n "$out" ]; then
  printf 'fake recursive delivery\n' > "$out"
fi
printf '%s\n' "$input"
exit 0
SH
  chmod 700 "$FAKE_CODEX"
fi

cd "$ROOT_DIR"

go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackDirectorRecursiveWaveOfflineCierraSubarbolConWaitsAcotadosV0'
echo "offline_recursive_closure_review_ok=true"

go run ./cmd/orquesta-server codex-launch-director-wave \
  $DRY_RUN_FLAG \
  --purge-runtime \
  --confirm-purge-runtime "recursive-wave-smoke" \
  --agents 1 \
  --allow-unmanaged-launch \
  --unmanaged-launch-reason "smoke recursivo local con codex-fake; no toca OPES ni remoto" \
  --confirm-unmanaged-launch "recursive-wave-smoke" \
  --allow-recursive-delegation \
  --max-delegation-depth 2 \
  --max-subagents-per-agent 2 \
  --recursive-agent-budget 7 \
  --wave-ref recursive-wave-smoke \
  --project-dir "$PROJECT_DIR" \
  --runtime-dir "$RUNTIME_DIR" \
  --command "$CODEX_COMMAND" \
  --reasoning-effort medium \
  --sandbox "${ORQUESTA_CODEX_WAVE_SANDBOX:-workspace-write}" \
  --approval-policy "${ORQUESTA_CODEX_WAVE_APPROVAL_POLICY:-never}" \
  --objective "Smoke recursivo Codex acotado: padre, hijos y nietos con linaje durable; no cerrar sin review causal." \
  --write-set "cmd/orquesta-server/codex_director_wave_command_v0.go,cmd/orquesta-server/codex_director_wave_command_v0_test.go,scripts/smoke_codex_director_recursive_wave.sh" \
  --required-tests "go test -count=1 ./cmd/orquesta-server -run TestCodexLaunchDirectorWaveCommandV0Recursive" \
  --branch-ref "branch-recursive-wave-smoke" \
  --worktree-ref "worktree-recursive-wave-smoke" \
  >"$SUMMARY_PATH"

python3 - "$SUMMARY_PATH" "$RUNTIME_DIR" <<'PY'
import json
import os
import sys
import time

path = sys.argv[1]
root_runtime_dir = os.path.normpath(sys.argv[2])
with open(path, "r", encoding="utf-8") as fh:
    summary = json.load(fh)
real_config_only = os.environ.get("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_REAL_CONFIG_ONLY") == "1"
real_mode = os.environ.get("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_REAL_CONFIRM") == "1" or real_config_only

issues = summary.get("issues") or []
if issues:
    raise SystemExit(f"issues={issues}")

budget = summary.get("agent_budget") or {}
if budget.get("planned_agents") != 7 or budget.get("max_agents") != 7 or budget.get("exceeded"):
    raise SystemExit(f"budget_invalido={budget}")

root_agents = (summary.get("launch") or {}).get("agents") or []
children = summary.get("child_launches") or []
if len(root_agents) != 1 or len(children) != 1:
    raise SystemExit("root/children count invalido")

child = children[0]
if child.get("parent_agent_ref") != root_agents[0].get("agent_ref"):
    raise SystemExit("parent_agent_ref hijo no empata")
if child.get("delegation_depth") != 1 or child.get("max_delegation_depth") != 2:
    raise SystemExit(f"depth hijo invalida={child}")
if child.get("max_subagents_per_agent") != 2 or child.get("subtree_agent_budget") != 6:
    raise SystemExit(f"fanout/budget hijo invalido={child}")
if not child.get("review_required_before_close"):
    raise SystemExit("hijo permitiria cierre sin review")

child_agents = (child.get("launch") or {}).get("agents") or []
grandchildren = child.get("child_launches") or []
if len(child_agents) != 2 or len(grandchildren) != 2:
    raise SystemExit("child/grandchild count invalido")

grandchild = grandchildren[0]
if grandchild.get("parent_agent_ref") != child_agents[0].get("agent_ref"):
    raise SystemExit("parent_agent_ref nieto no empata")
if grandchild.get("delegation_depth") != 2 or grandchild.get("child_launches"):
    raise SystemExit(f"depth/cierre nieto invalido={grandchild}")
if not grandchild.get("review_required_before_close"):
    raise SystemExit("nieto permitiria cierre sin review")

nodes = []

def collect_launch(launch, depth, parent_agent_ref=""):
    for agent in launch.get("agents") or []:
        nodes.append({
            "depth": depth,
            "parent_agent_ref": parent_agent_ref,
            "launch": launch,
            "agent": agent,
        })

def collect_children(children):
    for child in children or []:
        collect_launch(child.get("launch") or {}, child.get("delegation_depth"), child.get("parent_agent_ref", ""))
        collect_children(child.get("child_launches") or [])

collect_launch(summary.get("launch") or {}, 0)
collect_children(summary.get("child_launches") or [])
if len(nodes) != 7:
    raise SystemExit(f"agent_tree_count_invalido={len(nodes)}")

seen = set()
for node in nodes:
    agent = node["agent"]
    agent_ref = agent.get("agent_ref") or ""
    if not agent_ref:
        raise SystemExit(f"agent_ref_vacio={agent}")
    if agent_ref in seen:
        raise SystemExit(f"agent_ref_duplicado={agent_ref}")
    if node["depth"] and node["parent_agent_ref"] not in seen:
        raise SystemExit(f"parent_agent_ref_no_observable={node}")
    seen.add(agent_ref)

root_wave_ref = (summary.get("launch") or {}).get("wave_ref") or ""
registries = {}

def registry_path_for_launch(launch):
    public_path = launch.get("registry_path") or ""
    if public_path:
        return public_path
    wave_ref = launch.get("wave_ref") or ""
    if not wave_ref:
        raise SystemExit(f"wave_ref_vacio={launch}")
    if wave_ref == root_wave_ref:
        return os.path.join(root_runtime_dir, "codex_wave_registry_v0.json")
    return os.path.join(root_runtime_dir, "children", wave_ref, "codex_wave_registry_v0.json")

def registry_for_launch(launch):
    registry_path = registry_path_for_launch(launch)
    if registry_path not in registries:
        if not os.path.exists(registry_path):
            raise SystemExit(f"registry_no_existe={registry_path}")
        with open(registry_path, "r", encoding="utf-8") as fh:
            registry = json.load(fh)
        expected_refs = [agent.get("agent_ref") for agent in launch.get("agents") or []]
        registry_refs = [agent.get("agent_ref") for agent in registry.get("agents") or []]
        if registry.get("schema_version") != "orquesta_codex_wave_launch.v0":
            raise SystemExit(f"registry_schema_invalido={registry_path}")
        if registry.get("wave_ref") != launch.get("wave_ref") or registry_refs != expected_refs:
            raise SystemExit(f"registry_refs_no_coinciden={registry_path}")
        registries[registry_path] = registry
    return registries[registry_path]

for node in nodes:
    launch = node["launch"]
    registry = registry_for_launch(launch)
    registry_agents = {
        agent.get("agent_ref"): agent
        for agent in registry.get("agents") or []
    }
    public_agent = node["agent"]
    agent_ref = public_agent.get("agent_ref") or ""
    agent = registry_agents.get(agent_ref)
    if agent is None:
        raise SystemExit(f"agent_no_en_registry={agent_ref}")
    for key in ("prompt_path", "wrapper_path"):
        if not os.path.exists(agent.get(key) or ""):
            raise SystemExit(f"{key}_no_existe={agent}")
    wrapper_path = agent.get("wrapper_path") or ""
    with open(wrapper_path, "r", encoding="utf-8") as fh:
        wrapper = fh.read()
    last_message_path = agent.get("last_message_path") or ""
    runtime_work_dir = os.path.normpath(agent.get("runtime_work_dir") or "")
    if "--output-last-message" not in wrapper or last_message_path not in wrapper:
        raise SystemExit(f"wrapper_sin_last_message={agent_ref}")
    if 'model_reasoning_effort="xhigh"' in wrapper or (
        'model_reasoning_effort="medium"' not in wrapper and
        'model_reasoning_effort="high"' not in wrapper
    ):
        raise SystemExit(f"wrapper_reasoning_effort_invalido={agent_ref}")
    if not os.path.normpath(last_message_path).startswith(runtime_work_dir + os.sep):
        raise SystemExit(f"last_message_fuera_runtime={agent_ref}")
    if (summary.get("launch") or {}).get("dry_run"):
        if agent.get("status") != "dry_run":
            raise SystemExit(f"status_no_dry_run={agent}")
        continue
    if not agent.get("process_ref") or not agent.get("pid"):
        raise SystemExit(f"process_ref_pid_faltante={agent}")
    if agent.get("status") not in ("running", "stopped"):
        raise SystemExit(f"status_no_ejecutable={agent}")
    if not real_mode:
        deadline = time.time() + 2
        while time.time() < deadline and not os.path.exists(last_message_path):
            time.sleep(0.02)
        if not os.path.exists(last_message_path):
            raise SystemExit(f"last_message_no_observable={agent_ref}")
        stdout_path = agent.get("stdout_path") or ""
        if not os.path.exists(stdout_path):
            raise SystemExit(f"stdout_no_existe={agent_ref}")
        with open(stdout_path, "r", encoding="utf-8") as fh:
            stdout = fh.read()
        if "Director Operativo Orquesta aprobo esta ola" not in stdout and "Subagente Codex gobernado por Director Operativo Orquesta" not in stdout:
            raise SystemExit(f"stdout_sin_prompt_ejecutado={agent_ref}")

print("recursive_wave_ok=true")
print("summary_path=" + path)
print("planned_agents=7")
print("opaque_agent_refs=7")
print("registry_wrappers_ready=true")
print("fake_runtime_executed=" + str(not real_mode and not bool((summary.get("launch") or {}).get("dry_run"))).lower())
print("real_mode=" + str(real_mode).lower())
print("real_config_only=" + str(real_config_only).lower())
print("dry_run=" + str(bool((summary.get("launch") or {}).get("dry_run"))).lower())
PY
