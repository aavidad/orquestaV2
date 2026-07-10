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
CANONICAL_GIT_REMOTE_URL="git@github.com:aavidad/orquestador.git"
CANONICAL_GIT_REF="trabajo/plataforma-agentes"
GO_BIN="${ORQUESTA_CTL_GO_BINARY:-/srv/orquesta-self/tools/go/bin/go}"
BINARY_REAL=""
EXPECTED_BINARY_SHA256=""
EXPECTED_BINARY_NAME=""
EXPECTED_COMMIT_REF=""
EXPECTED_BUILD_REF=""

fail() { echo "orquesta_server_ctl: reason_code=$1 message=$1" >&2; exit 1; }

preflight() {
  [ "$(id -un)" = "$SVC_USER" ] || fail "wrong_service_user" \
    "debe ejecutarse como $SVC_USER (actual: $(id -un)); arrancar como otro usuario deja el estado con dueno equivocado"
  case "$BIN" in
    /*) ;;
    *) fail "binary_identity_invalid" "binary path debe ser absoluta" ;;
  esac
  [ -x "$BIN" ] || fail "binary_missing" "binario no disponible"
  command -v readlink >/dev/null 2>&1 || fail "binary_identity_invalid" "readlink no disponible"
  command -v sha256sum >/dev/null 2>&1 || fail "binary_identity_invalid" "sha256sum no disponible"
  command -v setsid >/dev/null 2>&1 || fail "binary_identity_invalid" "setsid no disponible"
  [ -x "$GO_BIN" ] || fail "binary_identity_invalid" "go no disponible"
  BINARY_REAL="$(readlink -f -- "$BIN" 2>/dev/null || true)"
  [ -n "$BINARY_REAL" ] && [ -f "$BINARY_REAL" ] && [ -x "$BINARY_REAL" ] || \
    fail "binary_identity_invalid" "target binario invalido"
  [ -d "$R/state" ] || fail "state_dir_missing" "no existe $R/state"
  if [ ! -d "$WORKDIR" ]; then
    fail "ctl_workdir_invalid" "workdir no existe o no es directorio: $WORKDIR"
  fi
  case "$WORKDIR" in
    /*) ;;
    *) fail "ctl_workdir_invalid" "workdir debe ser una ruta absoluta canonica" ;;
  esac
  if ! git -C "$WORKDIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    fail "ctl_workdir_invalid" "workdir no es un worktree git valido: $WORKDIR"
  fi
  if ! git -C "$WORKDIR" rev-parse --verify HEAD >/dev/null 2>&1; then
    fail "ctl_workdir_invalid" "workdir sin HEAD valido: $WORKDIR"
  fi
  worktree_root="$(git -C "$WORKDIR" rev-parse --show-toplevel 2>/dev/null || true)"
  workdir_physical="$(cd "$WORKDIR" && pwd -P)"
  if [ -z "$worktree_root" ] || [ "$WORKDIR" != "$workdir_physical" ] || [ "$worktree_root" != "$workdir_physical" ]; then
    fail "ctl_workdir_invalid" "workdir no es la raiz del worktree git: $WORKDIR"
  fi
  if [ -e "$WORKDIR/.orquesta-retired" ] || [ -e "$WORKDIR/RETIRED" ]; then
    fail "ctl_workdir_invalid" "workdir marcado retirado: $WORKDIR"
  fi
  if [ -e "$WORKDIR/.orquesta-stale" ] || [ -e "$WORKDIR/STALE" ]; then
    fail "ctl_workdir_stale" "workdir marcado stale: $WORKDIR"
  fi
  if [ -f "$WORKDIR/.git" ]; then
    gitdir="$(git -C "$WORKDIR" rev-parse --git-dir 2>/dev/null || true)"
    if [ -n "$gitdir" ] && [ ! -d "$gitdir" ]; then
      fail "ctl_workdir_invalid" "workdir gitdir retirado: $WORKDIR"
    fi
  fi
  upstream="$(git -C "$WORKDIR" rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)"
  branch="$(git -C "$WORKDIR" symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
  [ -n "$upstream" ] && [ -n "$branch" ] || fail "ctl_workdir_ref_not_canonical" \
    "workdir sin rama/upstream canonicos"
  remote="${upstream%%/*}"
  upstream_branch="${upstream#*/}"
  [ -n "$remote" ] && [ "$upstream_branch" = "$CANONICAL_GIT_REF" ] && \
    [ "$branch" = "$CANONICAL_GIT_REF" ] || \
    fail "ctl_workdir_ref_not_canonical" "rama local y upstream no coinciden"
  remote_url="$(git -C "$WORKDIR" remote get-url "$remote" 2>/dev/null || true)"
  [ "$remote_url" = "$CANONICAL_GIT_REMOTE_URL" ] || fail "ctl_workdir_remote_not_canonical" \
    "upstream no pertenece al remote GitHub canonico"
  head_sha="$(git -C "$WORKDIR" rev-parse HEAD)"
  upstream_sha="$(git -C "$WORKDIR" rev-parse "$upstream" 2>/dev/null || true)"
  [ -n "$upstream_sha" ] || fail "ctl_workdir_ref_not_canonical" "upstream sin revision verificable"
  if [ "$head_sha" != "$upstream_sha" ]; then
    if git -C "$WORKDIR" merge-base --is-ancestor HEAD "$upstream" >/dev/null 2>&1; then
      fail "ctl_workdir_stale" "workdir atrasado respecto al upstream canonico"
    fi
    fail "ctl_workdir_not_aligned" "HEAD no coincide exactamente con upstream"
  fi
  EXPECTED_BINARY_SHA256="$(sha256sum "$BINARY_REAL" 2>/dev/null | awk '{print $1}')"
  [ "${#EXPECTED_BINARY_SHA256}" -eq 64 ] || fail "binary_identity_invalid" "sha256 no disponible"
  EXPECTED_BINARY_NAME="$(basename "$BINARY_REAL")"
  EXPECTED_COMMIT_REF="$head_sha"
  EXPECTED_BUILD_REF="build-ref-orquesta-server-${head_sha%${head_sha#????????????}}"
  build_info="$($GO_BIN version -m "$BINARY_REAL" 2>/dev/null || true)"
  build_commit="$(printf '%s\n' "$build_info" | awk '$1 == "build" && $2 ~ /^vcs.revision=/ {sub(/^vcs.revision=/, "", $2); print $2; exit}')"
  build_modified="$(printf '%s\n' "$build_info" | awk '$1 == "build" && $2 ~ /^vcs.modified=/ {sub(/^vcs.modified=/, "", $2); print $2; exit}')"
  [ -n "$build_commit" ] || fail "binary_identity_invalid" "binario sin revision vcs"
  [ "$build_modified" = "false" ] || fail "ctl_binary_not_reproducible" "binario modificado"
  [ "$build_commit" = "$EXPECTED_COMMIT_REF" ] || fail "ctl_binary_commit_mismatch" "binario y HEAD distintos"
  if [ -n "$CONFIG" ] && [ ! -r "$CONFIG" ]; then
    fail "config_missing" "config canonica no legible: $CONFIG"
  fi
  mkdir -p "$R/logs"
  # Distinguir permiso-denegado de fichero-corrupto ANTES de arrancar:
  bad_owner=$(find "$R/state" ! -user "$SVC_USER" -print -quit 2>/dev/null || true)
  [ -z "$bad_owner" ] || fail "state_permission_denied" \
    "ficheros de estado con dueno distinto de $SVC_USER (ej: $bad_owner); corregir con chown -R $SVC_USER antes de arrancar"
}

pid_alive() {
  pid="$1"
  kill -0 "$pid" 2>/dev/null || return 1
  if [ -r "/proc/$pid/stat" ] && [ "$(awk '{print $3}' "/proc/$pid/stat" 2>/dev/null || true)" = "Z" ]; then
    return 1
  fi
  return 0
}

is_alive() {
  [ -f "$R/server.pid" ] && pid_alive "$(cat "$R/server.pid")"
}

startup_ready() {
  readiness="$(curl -fsS -m 8 "http://$ADDR/api/v0/server/readiness" 2>/dev/null || true)"
  [ -n "$readiness" ] || return 1
  READINESS_JSON="$readiness" \
  EXPECTED_BINARY_SHA256="$EXPECTED_BINARY_SHA256" \
  EXPECTED_BINARY_NAME="$EXPECTED_BINARY_NAME" \
  EXPECTED_COMMIT_REF="$EXPECTED_COMMIT_REF" \
  EXPECTED_BUILD_REF="$EXPECTED_BUILD_REF" python3 - <<'PY'
import json
import os
import sys

try:
    value = json.loads(os.environ["READINESS_JSON"])
except (KeyError, json.JSONDecodeError):
    raise SystemExit(1)

ready = (
    value.get("schema_version") == "orquesta_server_readiness.v0"
    and value.get("ready") is True
    and value.get("liveness_status") == "ok"
    and value.get("startup_ready") is True
    and value.get("startup_status") == "startup_ready"
    and value.get("status") == "running"
    and value.get("availability_status") == "running"
    and isinstance(value.get("runtime_identity"), dict)
    and value["runtime_identity"].get("schema_version") == "orquesta_server_runtime_identity.v0"
    and value["runtime_identity"].get("binary_path_ref") == "server-runtime-binary-path"
    and value["runtime_identity"].get("binary_name") == os.environ["EXPECTED_BINARY_NAME"]
    and value["runtime_identity"].get("binary_sha256") == os.environ["EXPECTED_BINARY_SHA256"]
    and value["runtime_identity"].get("build_ref") == os.environ["EXPECTED_BUILD_REF"]
    and value["runtime_identity"].get("commit_ref") == os.environ["EXPECTED_COMMIT_REF"]
)
raise SystemExit(0 if ready else 1)
PY
}

process_group_alive() {
  pid="$1"
  [ -n "$pid" ] && kill -0 -- "-$pid" 2>/dev/null
}

process_start_ref() {
  pid="$1"
  [ -r "/proc/$pid/stat" ] || return 1
  awk '{print $22}' "/proc/$pid/stat" 2>/dev/null
}

started_identity_unchanged() {
  pid="$1"
  expected_start_ref="$2"
  current_start_ref="$(process_start_ref "$pid" 2>/dev/null || true)"
  [ -z "$current_start_ref" ] || [ "$current_start_ref" = "$expected_start_ref" ]
}

signal_started_group() {
  signal_name="$1"
  pid="$2"
  expected_start_ref="$3"
  started_identity_unchanged "$pid" "$expected_start_ref" || return 1
  process_group_alive "$pid" || return 0
  if ! kill "-$signal_name" -- "-$pid" 2>/dev/null; then
    process_group_alive "$pid" && return 1
  fi
  return 0
}

cleanup_failed_start() {
  pid="$(cat "$R/server.pid" 2>/dev/null || true)"
  start_ref="$(cat "$R/server.startref" 2>/dev/null || true)"
  [ -n "$pid" ] && [ -n "$start_ref" ] || return 1
  signal_started_group INT "$pid" "$start_ref" || return 1
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    process_group_alive "$pid" || return 0
    sleep 0.1
  done
	started_identity_unchanged "$pid" "$start_ref" || return 1
	signal_started_group TERM "$pid" "$start_ref" || return 1
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    process_group_alive "$pid" || return 0
    sleep 0.1
  done
	started_identity_unchanged "$pid" "$start_ref" || return 1
	signal_started_group KILL "$pid" "$start_ref" || return 1
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    process_group_alive "$pid" || return 0
    sleep 0.1
  done
	return 1
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
  nohup setsid env PATH=/srv/orquesta-self/tools/npm-global/bin:/srv/orquesta-self/tools/go/bin:/usr/local/bin:/usr/bin:/bin \
    ORQUESTA_SERVER_ADDR="$ADDR" \
    ORQUESTA_SERVER_STATE_DIR="$R/state" \
    ORQUESTA_SERVER_WORKTREE="$WORKDIR" \
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
	  started_pid=$!
	  echo "$started_pid" > "$R/server.pid"
	  started_ref="$(process_start_ref "$started_pid" 2>/dev/null || true)"
	  [ -n "$started_ref" ] || fail "startup_identity_unavailable" "identidad del daemon no verificable"
	  echo "$started_ref" > "$R/server.startref"
  sleep "$STARTUP_SLEEP"
	  if ! is_alive; then
	    cleanup_failed_start || fail "startup_cleanup_failed" "no se pudo demostrar cleanup del grupo iniciado"
	    fail "startup_died" "el proceso murio en el arranque; ver stderr arriba"
	  fi
	  if ! startup_ready; then
	    cleanup_failed_start || fail "startup_cleanup_failed" "no se pudo demostrar cleanup del grupo iniciado"
    fail "startup_not_ready" "proceso iniciado sin readiness startup_ready exacta"
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
