#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
ORQUESTA_DB_PATH="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"

usage() {
  cat <<'EOF'
Uso:
  scripts/inicio_agente.sh <agente> [--tarea <id>] [--proyecto <slug|ruta>] [--no-auto]

Hace el arranque manual base de un agente:
  1. inicia sesión en Orquesta
  2. lista sus tareas
  3. arranca una tarea si se indica o si hay una única candidata clara

Opciones:
  --tarea <id>       inicia explícitamente esa tarea
  --proyecto <ref>   pasa el proyecto a "orquesta sesion inicio"
  --no-auto          no intenta iniciar tarea automáticamente

Ejemplos:
  scripts/inicio_agente.sh Codex2
  scripts/inicio_agente.sh Codex3 --tarea 155
  scripts/inicio_agente.sh antigravity --tarea 161 --proyecto orquestador
EOF
}

if [[ ! -x "$ORQUESTA_BIN" ]]; then
  echo "No encuentro el binario ejecutable: $ORQUESTA_BIN" >&2
  exit 1
fi

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

AGENTE="$1"
shift

TASK_ID=""
PROYECTO=""
AUTO_START=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tarea)
      TASK_ID="${2:-}"
      shift 2
      ;;
    --proyecto)
      PROYECTO="${2:-}"
      shift 2
      ;;
    --no-auto)
      AUTO_START=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Argumento inesperado: $1" >&2
      usage
      exit 1
      ;;
  esac
done

sql_escape() {
  printf "%s" "$1" | sed "s/'/''/g"
}

query_task_rows() {
  local estados_sql="$1"
  sqlite3 -readonly "$ORQUESTA_DB_PATH" "
    SELECT id || '|' || estado || '|' || titulo
    FROM tareas
    WHERE lower(agente) = lower('$(sql_escape "$AGENTE")')
      AND estado IN ($estados_sql)
    ORDER BY
      CASE estado
        WHEN 'en_progreso' THEN 0
        WHEN 'asignada' THEN 1
        WHEN 'bloqueada' THEN 2
        ELSE 9
      END,
      id;
  " 2>/dev/null || true
}

query_task_status() {
  local task_id="$1"
  sqlite3 -readonly "$ORQUESTA_DB_PATH" "
    SELECT COALESCE(estado, '')
    FROM tareas
    WHERE id = $task_id
      AND lower(agente) = lower('$(sql_escape "$AGENTE")')
    LIMIT 1;
  " 2>/dev/null | head -n 1
}

start_session() {
  local cmd=("$ORQUESTA_BIN" sesion inicio "$AGENTE")
  if [[ -n "$PROYECTO" ]]; then
    cmd+=(--proyecto "$PROYECTO")
  fi
  "${cmd[@]}"
}

start_task_if_needed() {
  local chosen_id="$1"
  local current_status
  current_status="$(query_task_status "$chosen_id")"

  if [[ -z "$current_status" ]]; then
    echo
    echo "No encuentro la tarea $chosen_id asignada a $AGENTE." >&2
    return 1
  fi

  if [[ "$current_status" == "en_progreso" ]]; then
    echo
    echo "Tarea $chosen_id ya estaba en progreso."
    return 0
  fi

  echo
  "$ORQUESTA_BIN" tarea iniciar "$chosen_id" "$AGENTE"
}

start_session
echo
echo "Tareas activas de $AGENTE:"
mapfile -t ACTIVE_ROWS < <(query_task_rows "'en_progreso','asignada','bloqueada'")
if [[ ${#ACTIVE_ROWS[@]} -eq 0 ]]; then
  echo "  - sin tareas activas"
else
  for row in "${ACTIVE_ROWS[@]}"; do
    IFS='|' read -r row_id row_status row_title <<<"$row"
    printf '  - #%s [%s] %s\n' "$row_id" "$row_status" "$row_title"
  done
fi

if [[ -n "$TASK_ID" ]]; then
  start_task_if_needed "$TASK_ID"
  exit 0
fi

if [[ "$AUTO_START" -eq 0 ]]; then
  echo
  echo "Autoarranque desactivado. Inicia la tarea manualmente si procede."
  exit 0
fi

mapfile -t IN_PROGRESS_ROWS < <(query_task_rows "'en_progreso'")
mapfile -t ASSIGNED_ROWS < <(query_task_rows "'asignada'")

if [[ ${#IN_PROGRESS_ROWS[@]} -eq 1 ]]; then
  IFS='|' read -r only_id _only_status only_title <<<"${IN_PROGRESS_ROWS[0]}"
  echo
  echo "Continuación detectada: tarea $only_id ya en progreso."
  echo "Título: $only_title"
  exit 0
fi

if [[ ${#IN_PROGRESS_ROWS[@]} -eq 0 && ${#ASSIGNED_ROWS[@]} -eq 1 ]]; then
  IFS='|' read -r only_id _only_status _only_title <<<"${ASSIGNED_ROWS[0]}"
  start_task_if_needed "$only_id"
  exit 0
fi

echo
if [[ ${#IN_PROGRESS_ROWS[@]} -gt 1 ]]; then
  echo "Hay varias tareas en progreso para $AGENTE; no autoarranco ninguna."
elif [[ ${#ASSIGNED_ROWS[@]} -gt 1 ]]; then
  echo "Hay varias tareas asignadas para $AGENTE; no autoarranco ninguna."
else
  echo "No hay una tarea única clara para autoarranque."
fi
echo "Usa: scripts/inicio_agente.sh $AGENTE --tarea <id>"
