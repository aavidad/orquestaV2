#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $# -eq 0 ]]; then
  tarea="Lee AGENTS.md, README.md y docs/*.md. Responde ACK-ESPERA con el siguiente microcorte recomendado."
else
  tarea="$*"
fi

cd "$script_dir"
exec codex "$tarea"
