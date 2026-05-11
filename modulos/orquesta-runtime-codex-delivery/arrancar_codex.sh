#!/usr/bin/env bash
set -euo pipefail

TASK="${*:-ACK-ESPERA}"

cat <<PROMPT
Estas en modulos/orquesta-runtime-codex-delivery.
Lee AGENTS.md, README.md y docs/*.md antes de actuar.
Microtarea: ${TASK}

Reglas: adaptador externo; no filtrar DB/HOME/OAuth/provider/modelo/rutas absolutas al nucleo; usar worktree baseline/diff para validar write-set real cuando aplique; ficheros pequenos y tests focales.
PROMPT
