#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1
export ORQUESTA_OPES_DERIVATIVES_EXECUTE=1
export ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble
export ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS="${ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS:-0}"
export ORQUESTA_OPES_BRIDGE_MAX_TICKS="${ORQUESTA_OPES_BRIDGE_MAX_TICKS:-8}"
export ORQUESTA_OPES_BRIDGE_LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"

cd "$repo_root"

bash -n scripts/smoke_opes_derivatives_rest.sh
bash -n scripts/smoke_opes_plan_temario_operadores.sh
scripts/smoke_opes_derivatives_rest.sh
