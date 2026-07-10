#!/usr/bin/env bash
# Runbook ejecutable de arranque/parada del servidor Orquesta remoto (TAREA-D7).
#
# Invariantes que este script protege (ver P8 en
# docs/auditoria_diseno_estructural_2026-07-06.md):
#  - El servidor corre SIEMPRE como el usuario de servicio (no root): la tanda
#    del 2026-07-05 arranco como root, dejo el estado con dueno root y el
#    siguiente arranque fallo con un error opaco (read_failed que era chown).
#  - Un unico perfil de envs canonico; nada de comandos a mano divergentes.
#  - Parada cooperativa (SIGINT) con verificacion, nunca kill -9 a la primera.
#
# Uso:  orquesta_server_ctl.sh start|stop|status
# Config por env (defaults = perfil remoto claude-director-20260705):
#   ORQUESTA_CTL_HOME     raiz de estado (default /srv/orquesta-self/claude-director-20260705)
#   ORQUESTA_CTL_BINARY   binario servidor (default /srv/orquesta-self/runtime/orquesta-server-claude)
#   ORQUESTA_CTL_USER     usuario de servicio esperado (default berserk)
#   ORQUESTA_CTL_ADDR     addr de escucha (default 127.0.0.1:19071)
#   ORQUESTA_CTL_WORKDIR  workdir de proyecto para agentes (default /srv/orquesta-self/worktrees/orquesta)
#   ORQUESTA_CTL_CONFIG   orquesta.config.json canonico; si no se indica, usa
#                         $ORQUESTA_CTL_WORKDIR/orquesta.config.json cuando exista

set -euo pipefail

R="${ORQUESTA_CTL_HOME:-/srv/orquesta-self/claude-director-20260705}"
BIN="${ORQUESTA_CTL_BINARY:-/srv/orquesta-self/runtime/orquesta-server-claude}"
SVC_USER="${ORQUESTA_CTL_USER:-berserk}"
ADDR="${ORQUESTA_CTL_ADDR:-127.0.0.1:19071}"
WORKDIR="${ORQUESTA_CTL_WORKDIR:-/srv/orquesta-self/worktrees/orquesta}"
CONFIG="${ORQUESTA_CTL_CONFIG:-}"
if [ -z "$CONFIG" ] && [ -f "$WORKDIR/orquesta.config.json" ]; then
  CONFIG="$WORKDIR/orquesta.config.json"
fi
STARTUP_SLEEP="${ORQUESTA_CTL_STARTUP_SLEEP:-8}"

fail() { echo "orquesta_server_ctl: reason_code=$1 $2" >&2; exit 1; }

preflight() {
  [ "$(id -un)" = "$SVC_USER" ] || fail "wrong_service_user" \
    "debe ejecutarse como $SVC_USER (actual: $(id -un)); arrancar como otro usuario deja el estado con dueno equivocado"
  [ -x "$BIN" ] || fail "binary_missing" "no existe o no es ejecutable: $BIN"
  [ -d "$R/state" ] || fail "state_dir_missing" "no existe $R/state"
  if [ ! -d "$WORKDIR" ]; then
    fail "ctl_workdir_invalid" "workdir no existe o no es directorio: $WORKDIR"
  fi
  if ! git -C "$WORKDIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    fail "ctl_workdir_invalid" "workdir no es un worktree git valido: $WORKDIR"
  fi
  if ! git -C "$WORKDIR" rev-parse --verify HEAD >/dev/null 2>&1; then
    fail "ctl_workdir_invalid" "workdir sin HEAD valido: $WORKDIR"
  fi
  if [ -f "$WORKDIR/.git" ]; then
    gitdir="$(git -C "$WORKDIR" rev-parse --git-dir 2>/dev/null || true)"
    if [ -n "$gitdir" ] && [ ! -d "$gitdir" ]; then
      fail "ctl_workdir_invalid" "workdir gitdir retirado: $WORKDIR"
    fi
  fi
  upstream="$(git -C "$WORKDIR" rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)"
  if [ -n "$upstream" ]; then
    head_sha="$(git -C "$WORKDIR" rev-parse HEAD)"
    upstream_sha="$(git -C "$WORKDIR" rev-parse "$upstream" 2>/dev/null || true)"
    if [ -n "$upstream_sha" ] && [ "$head_sha" != "$upstream_sha" ]; then
      if git -C "$WORKDIR" merge-base --is-ancestor HEAD "$upstream" >/dev/null 2>&1; then
        fail "ctl_workdir_stale" "workdir atrasado respecto a upstream=$upstream"
      fi
      if ! git -C "$WORKDIR" merge-base --is-ancestor "$upstream" HEAD >/dev/null 2>&1; then
        fail "ctl_workdir_not_aligned" "workdir diverge de upstream=$upstream"
      fi
    fi
  fi
  if [ -n "$CONFIG" ] && [ ! -r "$CONFIG" ]; then
    fail "config_missing" "config canonica no legible: $CONFIG"
  fi
  mkdir -p "$R/logs"
  # Distinguir permiso-denegado de fichero-corrupto ANTES de arrancar:
  bad_owner=$(find "$R/state" ! -user "$SVC_USER" -print -quit 2>/dev/null || true)
  [ -z "$bad_owner" ] || fail "state_permission_denied" \
    "ficheros de estado con dueno distinto de $SVC_USER (ej: $bad_owner); corregir con chown -R $SVC_USER antes de arrancar"
}

is_alive() {
  [ -f "$R/server.pid" ] && kill -0 "$(cat "$R/server.pid")" 2>/dev/null
}

case "${1:-}" in
start)
  preflight
  if is_alive; then
    echo "ya vivo pid=$(cat "$R/server.pid")"
    exit 0
  fi
  config_args=()
  if [ -n "$CONFIG" ]; then
    config_args=(--config "$CONFIG")
  fi
  nohup env PATH=/srv/orquesta-self/tools/npm-global/bin:/srv/orquesta-self/tools/go/bin:/usr/local/bin:/usr/bin:/bin \
    ORQUESTA_SERVER_ADDR="$ADDR" \
    ORQUESTA_SERVER_STATE_DIR="$R/state" \
    ORQUESTA_CODEX_RUNTIME_WORKDIR="$R/runtime" \
    ORQUESTA_CODEX_PROJECT_WORKDIR="$WORKDIR" \
    ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
    ORQUESTA_CODEX_CODE_HOME=/srv/orquesta-self/codex-home \
    ORQUESTA_CODEX_COMMAND="${ORQUESTA_CODEX_COMMAND:-/srv/orquesta-self/tools/npm-global/bin/codex}" \
    ORQUESTA_CODEX_APPROVAL_POLICY=never \
    ORQUESTA_CODEX_SANDBOX=workspace-write \
    ORQUESTA_CODEX_GOAL_TIMEOUT_MS=1800000 \
    ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=450000 \
    ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true \
    "$BIN" run "${config_args[@]}" >"$R/logs/stdout.log" 2>"$R/logs/stderr.log" &
  echo $! > "$R/server.pid"
  sleep "$STARTUP_SLEEP"
  if ! is_alive; then
    echo "--- stderr ---" >&2; tail -5 "$R/logs/stderr.log" >&2 || true
    fail "startup_died" "el proceso murio en el arranque; ver stderr arriba"
  fi
  if ! curl -fsS -m 8 "http://$ADDR/api/status" >/dev/null 2>&1; then
    fail "startup_not_ready" "proceso vivo pero /api/status no responde en $ADDR"
  fi
  echo "arrancado pid=$(cat "$R/server.pid") addr=$ADDR"
  ;;
stop)
  if ! is_alive; then
    echo "ya parado"
    exit 0
  fi
  PID=$(cat "$R/server.pid")
  # Parada comun del repo: shutdown por API -> SIGINT -> SIGTERM escalonados,
  # con limpieza del runtime tmux de agentes (invariante del repo: sin
  # procesos residuales tras parar el servidor).
  . "$(dirname "$0")/lib/smoke_common.sh"
  runtime_dir="$R/runtime"
  smoke_shutdown_orquesta_server "$PID" "http://$ADDR" 8 60 "$runtime_dir"
  if kill -0 "$PID" 2>/dev/null; then
    fail "stop_timeout" "sigue vivo tras el shutdown comun; revisar shutdown_status antes de escalar senal"
  fi
  echo "parado limpio (shutdown comun)"
  ;;
status)
  if is_alive; then
    echo "vivo pid=$(cat "$R/server.pid")"
    curl -fsS -m 8 "http://$ADDR/api/status" 2>/dev/null | head -c 400 || echo "(api no responde)"
    echo
  else
    echo "parado"
  fi
  ;;
*)
  echo "uso: $0 start|stop|status" >&2
  exit 2
  ;;
esac
