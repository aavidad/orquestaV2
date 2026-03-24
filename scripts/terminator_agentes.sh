#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -n "${ORQUESTA_TERMINATOR_WRAPPER:-}" && -z "${ORQUESTA_TERMINAL_WRAPPER:-}" ]]; then
  export ORQUESTA_TERMINAL_WRAPPER="$ORQUESTA_TERMINATOR_WRAPPER"
fi
export ORQUESTA_TERMINAL_BACKEND="${ORQUESTA_TERMINAL_BACKEND:-terminator}"

exec "$ROOT_DIR/scripts/launch_agentes.sh" "$@"
