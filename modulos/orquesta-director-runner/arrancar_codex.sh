#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "arrancar_codex_modulo_no_disponible: wrapper historico; usa servidor residente/cola OrquestaV2." >&2
printf '%s\n' "ruta_vigente: go run ./cmd/orquesta-server run; luego usa API/CLI publica con write-set, ACK, checkpoint y shutdown gobernados." >&2
exit 2
