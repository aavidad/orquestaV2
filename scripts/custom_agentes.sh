#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export ORQUESTA_TERMINAL_BACKEND="custom"
export ORQUESTA_TERMINAL_LAUNCHER="${ORQUESTA_TERMINAL_LAUNCHER:-$ROOT_DIR/scripts/custom_terminal_launcher.sh}"

exec "$ROOT_DIR/scripts/launch_agentes.sh" "$@"
