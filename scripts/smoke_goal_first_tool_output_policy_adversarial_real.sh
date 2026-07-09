#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Smoke real BUG-079: fuerza camino adversarial donde el agente debe pedir
# checkpoint temprano y crear una prueba que intentaria emitir stdout gigante.
# El script delegado exige tool_output_policy_transport=accepted y falla si ve
# evidence-ref-codex-app-server-thread-output-sanitized o
# codex_app_server_thread_read_response_too_large antes de cierre/replan.
export ORQUESTA_GOAL_FIRST_SMOKE_TOOL_OUTPUT_POLICY_ADVERSARIAL_MODE=1
export ORQUESTA_CODEX_GOAL_BACKEND="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"
export ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"
export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-danger-full-access}"
export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-true}"
export ORQUESTA_SERVER_GOAL_OBSERVER_INTERVAL_MS="${ORQUESTA_SERVER_GOAL_OBSERVER_INTERVAL_MS:-1000}"
export ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS="${ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS:-5}"
export ORQUESTA_SERVER_GOAL_OBSERVER_FINGERPRINT_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_FINGERPRINT_ENABLED:-false}"
export ORQUESTA_GOAL_FIRST_SMOKE_POLLS="${ORQUESTA_GOAL_FIRST_SMOKE_POLLS:-60}"
export ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS="${ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS:-3}"

exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"
