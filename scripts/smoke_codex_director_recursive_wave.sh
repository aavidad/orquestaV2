#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMOKE_ROOT="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-codex-director-recursive-smoke.XXXXXX")}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$SMOKE_ROOT/$SMOKE_ID"
PROJECT_DIR="$OUT_DIR/project"
RUNTIME_DIR="$OUT_DIR/runtime/codex-waves/recursive-wave-smoke"
SUMMARY_PATH="$OUT_DIR/summary.json"
FAKE_CODEX="$OUT_DIR/codex-fake"
KEEP_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-0}"

cleanup() {
  if [[ "$KEEP_DIR" == "1" ]]; then
    echo "directorio conservado: $SMOKE_ROOT" >&2
  else
    rm -rf "$SMOKE_ROOT"
  fi
}
trap cleanup EXIT

mkdir -p "$PROJECT_DIR" "$OUT_DIR"

DRY_RUN_FLAG="--dry-run"
CODEX_COMMAND="${ORQUESTA_CODEX_COMMAND:-$FAKE_CODEX}"
if [[ "${ORQUESTA_CODEX_DIRECTOR_RECURSIVE_REAL_CONFIRM:-0}" == "1" ]]; then
  DRY_RUN_FLAG=""
  if [[ -z "${ORQUESTA_CODEX_COMMAND:-}" ]]; then
    echo "ORQUESTA_CODEX_COMMAND requerido para modo real" >&2
    exit 2
  fi
else
  cat >"$FAKE_CODEX" <<'SH'
#!/usr/bin/env sh
exit 0
SH
  chmod 700 "$FAKE_CODEX"
fi

cd "$ROOT_DIR"

go run ./cmd/orquesta-server codex-launch-director-wave \
  $DRY_RUN_FLAG \
  --purge-runtime \
  --agents 1 \
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

python3 - "$SUMMARY_PATH" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as fh:
    summary = json.load(fh)

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

print("recursive_wave_ok=true")
print("summary_path=" + path)
print("planned_agents=7")
print("dry_run=" + str(bool((summary.get("launch") or {}).get("dry_run"))).lower())
PY
