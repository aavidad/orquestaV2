#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
AGENTE="${1:-}"
PROYECTO="${2:-orquesta}"
PERFIL_BIN="${HOME}/Trabajo/codex-perfiles/bin/codex-perfil"

usage() {
  cat <<'EOF'
Uso:
  scripts/arrancar_agente_identificacion.sh <Agente> [proyecto]

Abre una sesion interactiva manual con codex-perfil para identificar la cuenta
OAuth observada por Orquesta. Al salir:
  - guarda/cierra la sesion
  - refresca presupuesto observado
  - muestra la cuenta observada del agente

Ejemplos:
  scripts/arrancar_agente_identificacion.sh Codex7
  scripts/arrancar_agente_identificacion.sh Codex8 orquesta
EOF
}

if [[ -z "$AGENTE" ]]; then
  usage
  exit 1
fi

if [[ ! -x "$ORQUESTA_BIN" ]]; then
  echo "No encuentro el binario $ORQUESTA_BIN" >&2
  exit 1
fi

if [[ ! -x "$PERFIL_BIN" ]]; then
  echo "No encuentro codex-perfil en $PERFIL_BIN" >&2
  exit 1
fi

WORKTREE_CANDIDATE="$ROOT_DIR/.orquesta-worktrees/${PROYECTO}-${AGENTE,,}"
if [[ -d "$WORKTREE_CANDIDATE" ]]; then
  CWD_TRABAJO="$WORKTREE_CANDIDATE"
else
  CWD_TRABAJO="$ROOT_DIR"
fi

echo "Agente: $AGENTE"
echo "Proyecto: $PROYECTO"
echo "Directorio: $CWD_TRABAJO"
echo "Perfil: $PERFIL_BIN"
echo
echo "Inicia sesion con la cuenta que quieras asociar a $AGENTE."
echo "Al salir se refrescara la cuenta observada y el presupuesto."
echo

"$ROOT_DIR/scripts/agente_console.sh" \
  "$AGENTE" \
  "$PROYECTO" \
  "$CWD_TRABAJO" \
  "codex-cli" \
  "$PERFIL_BIN $AGENTE" \
  "Identificacion OAuth · $AGENTE"

echo
echo "Refrescando observacion de cuenta y presupuesto para $AGENTE..."
"$ORQUESTA_BIN" agente presupuesto --refresh --agente "$AGENTE" --json || true
echo
echo "Cuenta observada:"
"$ORQUESTA_BIN" agente cuentas | grep -E "^${AGENTE}[[:space:]]" || true
