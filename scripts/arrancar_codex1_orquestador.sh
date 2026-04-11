#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
CODEX_LAUNCHER="${CODEX_LAUNCHER:-$HOME/codex-perfiles/bin/codex-perfil}"
PERFIL="${1:-Codex1}"

if [[ ! -x "$CODEX_LAUNCHER" ]]; then
  echo "No encuentro el launcher: $CODEX_LAUNCHER" >&2
  exit 1
fi

cd "$ROOT_DIR"

export ORQUESTA_DB="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"
export ORQUESTA_SERVER_URL="${ORQUESTA_SERVER_URL:-http://127.0.0.1:16543}"

PROMPT=$(cat <<'EOF'
Continúa como orquestador autónomo del proyecto Orquesta.

Contexto operativo:
- Proyecto canónico: orquestador
- Repo canónico: Trabajo/orquesta
- BD canónica: $ORQUESTA_DB
- Daemon canónico: $ORQUESTA_SERVER_URL
- claude1 está en una rama propia y no se pisa su frente
- No borrar nada sin preguntar
- No reprogramar código ya hecho y validado; antes comprobar código+tests+evidencia
- Prioridad actual: revisar tareas pendientes, deduplicar frentes ya resueltos y orquestar a Codex2/3/4/5/6 en lo que realmente falta
- Codex5 puede usarse otra vez si hace falta
- Los votos no importan; lo importante es no duplicar trabajo de código

Hecho relevante ya comprobado:
- #405 ya está materialmente implementada
- hay duplicidad en tareas de voto
- #344 y #406 no son frentes vírgenes; hay base ya implementada
- #226 ya tiene parte del soporte multi-backend hecho
- claude1 tiene reservado #409 y su rama propia

Sigue sin preguntarme salvo que sea imprescindible.
EOF
)

exec "$CODEX_LAUNCHER" "$PERFIL" --cd "$ROOT_DIR" "$PROMPT"
