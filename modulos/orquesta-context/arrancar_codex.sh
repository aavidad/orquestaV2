#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/../_comun/arrancar_codex_modulo.sh" "$SCRIPT_DIR" "$@"
