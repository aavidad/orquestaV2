#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$script_dir/../_comun/arrancar_codex_modulo.sh" "$script_dir" "$@"
