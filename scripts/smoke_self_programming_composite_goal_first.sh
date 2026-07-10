#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$repo_root/scripts/lib/smoke_common.sh"
# shellcheck source=scripts/lib/go_tool.sh
source "$repo_root/scripts/lib/go_tool.sh"
# shellcheck source=scripts/lib/isolated_test_env.sh
source "$repo_root/scripts/lib/isolated_test_env.sh"

if [[ "${ORQUESTA_SELF_PROGRAMMING_COMPOSITE_SMOKE_CONFIRM:-0}" != "1" ]]; then
  echo "confirmacion requerida: exporta ORQUESTA_SELF_PROGRAMMING_COMPOSITE_SMOKE_CONFIRM=1" >&2
  exit 2
fi

goal_backend="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"
case "$goal_backend" in
  app_server_tmux) ;;
  stdio|app_server_proxy)
    echo "backend goal-first no permitido para self-programming compuesto: $goal_backend" >&2
    exit 2
    ;;
  *)
    echo "backend goal-first no soportado para self-programming compuesto: $goal_backend" >&2
    exit 2
    ;;
esac

if [[ "$(id -u)" == "0" ]]; then
  echo "smoke self-programming compuesto bloqueado: no ejecutar como root" >&2
  exit 2
fi

for forbidden in \
  ORQUESTA_OPES_BASE_URL \
  OPES_BASE_URL \
  ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL \
  ORQUESTA_OPES_BRIDGE_ENABLED \
  ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED
do
  if [[ -n "${!forbidden:-}" && "${!forbidden:-}" != "false" && "${!forbidden:-}" != "0" ]]; then
    echo "smoke self-programming compuesto bloqueado: $forbidden debe estar desactivado" >&2
    exit 2
  fi
done

orquesta_go_tool_ensure_path
smoke_require_tools go

if [[ -n "${ORQUESTA_SELF_PROGRAMMING_SMOKE_ROOT:-}" ]]; then
  smoke_root="$ORQUESTA_SELF_PROGRAMMING_SMOKE_ROOT"
else
  smoke_parent="${ORQUESTA_SMOKE_PARENT:-/srv/orquesta-self/runtime}"
  mkdir -p "$smoke_parent"
  ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES="$smoke_parent${ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES:+:$ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES}"
  export ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES
  smoke_root="$(mktemp -d "$smoke_parent/orquesta-selfprogramming-composite.XXXXXX")"
fi
smoke_temp_root_prepare "$smoke_root" "self-programming-composite"
trap 'smoke_temp_root_cleanup "$smoke_root" "${ORQUESTA_KEEP_SMOKE_DIR:-0}"' EXIT

summary="$smoke_root/self_programming_composite_summary.txt"
archive_dir="$smoke_root/autoprogramming-promotion-archive"
mkdir -p "$archive_dir"
orquesta_use_isolated_test_env "$smoke_root/test-env"
export ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux
export ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true
export ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT="$smoke_root"
export ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=true
export ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR="$archive_dir"
export ORQUESTA_OPES_BRIDGE_DRY_RUN=true
export ORQUESTA_OPES_REGISTRY_FINALPKG_DRY_RUN=true

{
  echo "smoke=self_programming_composite_goal_first"
  echo "smoke_root=$smoke_root"
  echo "goal_backend=$ORQUESTA_CODEX_GOAL_BACKEND"
  echo "promotion_archive_dir_ref=smoke-root:autoprogramming-promotion-archive"
  echo "opes_touched=false"
} >"$summary"

cd "$repo_root"

echo "validando perfil self-programming aislado..."
go test -count=1 ./cmd/orquesta-server -run 'TestSelfProgrammingOnlyConfigV0'
echo "self_programming_only_config=ok" | tee -a "$summary"

echo "ejecutando smoke real goal-first app_server_tmux..."
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_SMOKE_PARENT="$smoke_root" \
ORQUESTA_KEEP_SMOKE_DIR="${ORQUESTA_KEEP_SMOKE_DIR:-0}" \
  "$repo_root/scripts/smoke_goal_first_app_server_real.sh" | tee "$smoke_root/goal_first_app_server_real.log"
echo "goal_tmux_real_closure_accepted=ok" | tee -a "$summary"

echo "validando promocion/archive/replay local de autoprogramacion goal-first..."
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackAutoprogrammingPromotionV0GoalFirstE2ERepoTemporalReplayV0'
echo "promotion_archive_replay=ok" | tee -a "$summary"

echo "validando conector archive idempotente..."
go test -count=1 ./modulos/orquesta-runtime-worktree \
  -run 'TestGitStagingPromotionConnectorV0PromocionaYArchivaSinBorrarV0'
echo "archive_manifest_idempotent=ok" | tee -a "$summary"

echo "tests_passed=true" | tee -a "$summary"
echo "smoke_self_programming_composite_goal_first=ok"
echo "summary=$summary"
