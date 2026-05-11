#!/usr/bin/env bash
set -euo pipefail

TASK="${*:-ACK-ESPERA}"

cat <<PROMPT
Estas en modulos/orquesta-core-concurrency.
Lee AGENTS.md, README.md y docs/*.md antes de actuar.
Microtarea: ${TASK}

Reglas: decisiones deterministas sobre refs/read_set/write_set; sin Git real, filesystem productivo, DB/provider/model/HOME/OAuth.
PROMPT
