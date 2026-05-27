#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $# -eq 0 ]]; then
  tarea="Lee AGENTS.md, README.md y docs/*.md. Responde ACK-ESPERA con el siguiente microcorte recomendado."
else
  tarea="$*"
fi

if [[ "${ORQUESTA_MODULE_CODEX_WRAPPER_OPT_IN:-}" != "1" ]]; then
  printf '%s\n' "arrancar_codex_manual_requiere_opt_in: exporta ORQUESTA_MODULE_CODEX_WRAPPER_OPT_IN=1 solo para recuperacion manual." >&2
  printf '%s\n' "ruta_vigente: servidor residente/cola OrquestaV2 con write-set, ACK, checkpoint y shutdown gobernados." >&2
  exit 2
fi

cd "$script_dir"
exec codex "$tarea"
