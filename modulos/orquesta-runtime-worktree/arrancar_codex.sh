#!/usr/bin/env bash
set -euo pipefail

TASK="${*:-ACK-ESPERA}"

cat <<PROMPT
Estas en modulos/orquesta-runtime-worktree.
Lee AGENTS.md, README.md y docs/*.md antes de actuar.
Microtarea: ${TASK}

Reglas: conector externo de filesystem; sin DB/proveedor/modelo/HOME/OAuth; no filtrar rutas absolutas al nucleo; ficheros pequenos y tests focales.
PROMPT
