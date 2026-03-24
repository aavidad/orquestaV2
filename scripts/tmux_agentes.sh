#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export ORQUESTA_TERMINAL_BACKEND="tmux"

exec "$ROOT_DIR/scripts/launch_agentes.sh" "$@"
