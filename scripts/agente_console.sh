#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
ORQUESTA_DB_PATH="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"
CODEX_STATE_DB="${CODEX_STATE_DB:-$HOME/.codex/state_5.sqlite}"

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

sql_escape() {
  printf "%s" "$1" | sed "s/'/''/g"
}

query_single_value() {
  local sql="$1"
  sqlite3 "$ORQUESTA_DB_PATH" "$sql" 2>/dev/null | head -n 1
}

query_previous_session() {
  local cwd_sql
  cwd_sql="$(sql_escape "$CWD_TRABAJO")"
  query_single_value "
    SELECT COALESCE(external_session_id,'') || '|' || COALESCE(resumen_continuidad,'')
    FROM sesiones s
    WHERE lower(s.agente) = lower('$(sql_escape "$AGENTE")')
      AND cwd = '$cwd_sql'
    ORDER BY id DESC
    LIMIT 1;
  "
}

detect_codex_thread_id() {
  if [[ ! -f "$CODEX_STATE_DB" ]]; then
    return 0
  fi
  local cwd_sql token_sql branch_sql
  cwd_sql="$(sql_escape "$CWD_TRABAJO")"
  token_sql="$(sql_escape "$BOOTSTRAP_TOKEN")"
  branch_sql="$(sql_escape "$BRANCH")"
  sqlite3 "$CODEX_STATE_DB" "
    SELECT id
    FROM threads
    WHERE source = 'cli'
      AND cwd = '$cwd_sql'
      AND created_at >= $((START_TS - 5))
    ORDER BY
      CASE WHEN first_user_message LIKE '%$token_sql%' THEN 0 ELSE 1 END,
      CASE WHEN title LIKE '%$token_sql%' THEN 0 ELSE 1 END,
      CASE WHEN git_branch = '$branch_sql' THEN 0 ELSE 1 END,
      updated_at DESC,
      created_at DESC,
      id DESC
    LIMIT 1;
  " 2>/dev/null | head -n 1
}

current_branch() {
  git -C "$CWD_TRABAJO" branch --show-current 2>/dev/null || true
}

save_and_close() {
  local detected_id summary branch
  branch="$(current_branch)"
  detected_id="$(detect_codex_thread_id)"
  if [[ -z "$detected_id" ]]; then
    detected_id="$PREV_EXTERNAL_ID"
  fi

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
  if [[ -n "$detected_id" ]]; then
    printf 'Reanudar con: %s resume %s\n' "$RUNTIME_CMD" "$detected_id"
  fi
}

trap save_and_close EXIT

printf '\033]0;%s\007' "$TITULO"

PREV_RAW="$(query_previous_session || true)"
PREV_EXTERNAL_ID="${PREV_RAW%%|*}"
if [[ "$PREV_RAW" == *"|"* ]]; then
  PREV_SUMMARY="${PREV_RAW#*|}"
else
  PREV_SUMMARY=""
fi

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

case "$RUNTIME_CMD" in
  codex)
    if [[ -n "$PREV_EXTERNAL_ID" ]]; then
      codex resume -C "$CWD_TRABAJO" "$PREV_EXTERNAL_ID"
    else
      codex -C "$CWD_TRABAJO" "$BOOTSTRAP_PROMPT"
    fi
    ;;
  *)
    bash -lc "$RUNTIME_CMD"
    ;;
esac
