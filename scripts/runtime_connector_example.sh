#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Uso:
  scripts/runtime_connector_example.sh <detect|launch|resume-hint> ...

Contrato esperado por ORQUESTA_RUNTIME_CONNECTOR_SCRIPT.

Acciones:
  detect <runtime_cmd> <cwd> <start_ts> <bootstrap_token> <branch> [fallback_id]
  launch <runtime_cmd> <cwd> [external_session_id] [bootstrap_prompt]
  resume-hint <runtime_cmd> [external_session_id]

Ejemplo de integración:
  export ORQUESTA_RUNTIME_CONNECTOR_SCRIPT="$PWD/scripts/runtime_connector_example.sh"
  export ORQUESTA_RUNTIME_SESSION_FILE="$HOME/.orquesta/runtime-session-id"
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

ACTION="$1"
shift

runtime_session_file() {
  printf '%s' "${ORQUESTA_RUNTIME_SESSION_FILE:-$HOME/.orquesta/runtime-session-id}"
}

example_detect() {
  local runtime_cmd="$1"
  local _cwd="$2"
  local _start_ts="$3"
  local _bootstrap_token="$4"
  local _branch="$5"
  local fallback_id="${6:-}"
  local session_file
  session_file="$(runtime_session_file)"

  if [[ -f "$session_file" ]]; then
    head -n 1 "$session_file"
    return 0
  fi
  printf '%s' "$fallback_id"
}

example_launch() {
  local runtime_cmd="$1"
  local cwd="$2"
  local external_session_id="${3:-}"
  local bootstrap_prompt="${4:-}"

  case "$runtime_cmd" in
    codex)
      if [[ -n "$external_session_id" ]]; then
        exec codex resume -C "$cwd" "$external_session_id"
      fi
      exec codex -C "$cwd" "$bootstrap_prompt"
      ;;
    *)
      exec bash -lc "$runtime_cmd"
      ;;
  esac
}

example_resume_hint() {
  local runtime_cmd="$1"
  local external_session_id="${2:-}"
  if [[ -z "$external_session_id" ]]; then
    return 1
  fi
  case "$runtime_cmd" in
    codex)
      printf 'codex resume %s' "$external_session_id"
      ;;
    *)
      printf '%s --resume %s' "$runtime_cmd" "$external_session_id"
      ;;
  esac
}

case "$ACTION" in
  detect)
    if [[ $# -lt 5 ]]; then
      usage
      exit 1
    fi
    example_detect "$@"
    ;;
  launch)
    if [[ $# -lt 2 ]]; then
      usage
      exit 1
    fi
    example_launch "$@"
    ;;
  resume-hint)
    if [[ $# -lt 1 ]]; then
      usage
      exit 1
    fi
    example_resume_hint "$@"
    ;;
  *)
    usage
    exit 1
    ;;
esac
