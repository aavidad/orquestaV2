#!/usr/bin/env bash
set -euo pipefail

TASK="${*:-ACK-ESPERA}"

cat <<PROMPT
Estas en modulos/orquesta-core-replanner.
Lee AGENTS.md, README.md y docs/*.md antes de actuar.
Microtarea: ${TASK}

Reglas: contratos pequenos, hexagonal, refs opacas, sin DB/provider/model/HOME/OAuth/runtime real.
Si falta informacion de otro modulo, emite CONSULTA AL DIRECTOR.
PROMPT
