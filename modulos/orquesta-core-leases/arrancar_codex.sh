#!/usr/bin/env bash
set -euo pipefail

TASK="${*:-ACK-ESPERA}"

cat <<PROMPT
Estas en modulos/orquesta-core-leases.
Lee AGENTS.md, README.md y docs/*.md antes de actuar.
Microtarea: ${TASK}

Reglas: sin reloj interno no determinista, sin runtime real, sin DB/provider/model/HOME/OAuth/PID.
PROMPT
