#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
. "$ROOT_DIR/scripts/runtime_connector.sh"

usage() {
  cat <<'EOF'
Uso:
  scripts/agente_console.sh <agente> <ruta_proyecto> <cwd_trabajo> <conector> <runtime_cmd> [titulo]

Lanza una consola interactiva del agente, reanuda la sesión externa si existe y,
al salir, guarda automáticamente en Orquesta:
  - external_session_id
  - resumen_continuidad
  - cwd
  - branch

La detección de external_session_id depende del estado ya guardado en Orquesta
o del hook ORQUESTA_RUNTIME_CONNECTOR_SCRIPT. El script ya no inspecciona
almacenamientos locales del runtime por su cuenta.
EOF
}

if [[ $# -lt 5 ]]; then
  usage
  exit 1
fi

AGENTE="$1"
RUTA_PROYECTO="$2"
CWD_TRABAJO="$3"
CONECTOR="$4"
RUNTIME_CMD="$5"
TITULO="${6:-$AGENTE · $(basename "$RUTA_PROYECTO")}"

if [[ ! -x "$ORQUESTA_BIN" ]]; then
  echo "No encuentro el binario: $ORQUESTA_BIN" >&2
  exit 1
fi

if [[ ! -d "$CWD_TRABAJO" ]]; then
  echo "No existe el directorio de trabajo: $CWD_TRABAJO" >&2
  exit 1
fi

query_previous_field() {
  local field="$1"
  "$ORQUESTA_BIN" sesion continuar "$AGENTE" \
    --proyecto "$RUTA_PROYECTO" \
    --cwd "$CWD_TRABAJO" \
    --campo "$field" 2>/dev/null || true
}

current_branch() {
  git -C "$CWD_TRABAJO" branch --show-current 2>/dev/null || true
}

save_and_close() {
  local detected_id summary branch
  branch="$(current_branch)"
  detected_id="$(runtime_detect_external_session_id "$RUNTIME_CMD" "$CWD_TRABAJO" "$START_TS" "$BOOTSTRAP_TOKEN" "$branch" "$PREV_EXTERNAL_ID")"

  local prompt_default="$PREV_SUMMARY"
  if [[ -z "$prompt_default" ]]; then
    prompt_default="sin resumen"
  fi

  printf '\n'
  read -r -p "Resumen de continuidad [$prompt_default]: " summary < /dev/tty || true
  if [[ -z "${summary:-}" ]]; then
    summary="$PREV_SUMMARY"
  fi
  if [[ -z "${summary:-}" ]]; then
    summary="sesión finalizada sin resumen adicional"
  fi

  if [[ -n "$detected_id" ]]; then
    "$ORQUESTA_BIN" sesion guardar "$AGENTE" \
      --proyecto "$RUTA_PROYECTO" \
      --cwd "$CWD_TRABAJO" \
      --herramienta "$CONECTOR" \
      --branch "$branch" \
      --external-session-id "$detected_id" \
      --resumen "$summary" \
      --estado "pausada" >/dev/null 2>&1 || true
  else
    "$ORQUESTA_BIN" sesion guardar "$AGENTE" \
      --proyecto "$RUTA_PROYECTO" \
      --cwd "$CWD_TRABAJO" \
      --herramienta "$CONECTOR" \
      --branch "$branch" \
      --resumen "$summary" \
      --estado "pausada" >/dev/null 2>&1 || true
  fi

  "$ORQUESTA_BIN" sesion fin "$AGENTE" >/dev/null 2>&1 || true

  printf '\nSesión guardada.\n'
  if resume_hint="$(runtime_resume_hint "$RUNTIME_CMD" "$detected_id" 2>/dev/null)"; then
    printf 'Reanudar con: %s\n' "$resume_hint"
  elif [[ -z "$detected_id" && -z "$PREV_EXTERNAL_ID" ]]; then
    printf 'Aviso: no se detectó external_session_id; configura ORQUESTA_RUNTIME_CONNECTOR_SCRIPT si el conector soporta reanudación nativa.\n'
  fi
}

trap save_and_close EXIT

printf '\033]0;%s\007' "$TITULO"

PREV_EXTERNAL_ID="$(query_previous_field "external-session-id")"
PREV_SUMMARY="$(query_previous_field "resumen")"

START_TS="$(date +%s)"
BRANCH="$(current_branch)"
BOOTSTRAP_TOKEN="AGENTE=$AGENTE PROYECTO=$(basename "$RUTA_PROYECTO") CWD=$CWD_TRABAJO START_TS=$START_TS"
BOOTSTRAP_PROMPT="Contexto Orquesta: $BOOTSTRAP_TOKEN. Esta sesion pertenece al agente $AGENTE sobre el proyecto $(basename "$RUTA_PROYECTO"). Si no hay mas instrucciones del usuario, mantente en espera dentro del directorio actual."

session_args=(
  "$ORQUESTA_BIN" sesion inicio "$AGENTE"
  --proyecto "$RUTA_PROYECTO"
  --conector "$CONECTOR"
  --cwd "$CWD_TRABAJO"
  --branch "$BRANCH"
)
if [[ -n "$PREV_EXTERNAL_ID" ]]; then
  session_args+=(--external-session-id "$PREV_EXTERNAL_ID")
fi
if [[ -n "$PREV_SUMMARY" ]]; then
  session_args+=(--resumen "$PREV_SUMMARY")
fi
"${session_args[@]}" >/dev/null || true

cd "$CWD_TRABAJO"

runtime_launch "$RUNTIME_CMD" "$CWD_TRABAJO" "$PREV_EXTERNAL_ID" "$BOOTSTRAP_PROMPT"
