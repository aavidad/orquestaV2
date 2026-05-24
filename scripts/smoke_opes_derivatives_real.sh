#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -z "${ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM:-}" &&
  -n "${ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM:-}" ]]; then
  export ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM="$ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM"
fi

cd "$repo_root"
exec scripts/smoke_opes_derivatives_rest.sh "$@"
