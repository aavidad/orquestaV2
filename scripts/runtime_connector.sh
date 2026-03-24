#!/usr/bin/env bash

runtime_connector_script() {
  local script="${ORQUESTA_RUNTIME_CONNECTOR_SCRIPT:-}"
  if [[ -n "$script" && -x "$script" ]]; then
    printf '%s' "$script"
    return 0
  fi
  return 1
}

runtime_connector_hook() {
  local action="$1"
  shift
  local script
  if ! script="$(runtime_connector_script)"; then
    return 1
  fi
  "$script" "$action" "$@"
}

runtime_detect_external_session_id() {
  local runtime_cmd="$1"
  local cwd_trabajo="$2"
  local start_ts="$3"
  local bootstrap_token="$4"
  local branch="$5"
  local fallback_id="${6:-}"
  local detected_id=""

  if detected_id="$(runtime_connector_hook detect "$runtime_cmd" "$cwd_trabajo" "$start_ts" "$bootstrap_token" "$branch" "$fallback_id" 2>/dev/null)"; then
    if [[ -n "$detected_id" ]]; then
      printf '%s' "$detected_id"
      return 0
    fi
  fi
  printf '%s' "$fallback_id"
}

runtime_launch() {
  local runtime_cmd="$1"
  local cwd_trabajo="$2"
  local external_session_id="${3:-}"
  local bootstrap_prompt="${4:-}"

  if runtime_connector_hook launch "$runtime_cmd" "$cwd_trabajo" "$external_session_id" "$bootstrap_prompt"; then
    return 0
  fi

  case "$runtime_cmd" in
    codex)
      if [[ -n "$external_session_id" ]]; then
        codex resume -C "$cwd_trabajo" "$external_session_id"
      else
        codex -C "$cwd_trabajo" "$bootstrap_prompt"
      fi
      ;;
    *)
      bash -lc "$runtime_cmd"
      ;;
  esac
}

runtime_resume_hint() {
  local runtime_cmd="$1"
  local external_session_id="${2:-}"
  local hint=""

  if hint="$(runtime_connector_hook resume-hint "$runtime_cmd" "$external_session_id" 2>/dev/null)"; then
    if [[ -n "$hint" ]]; then
      printf '%s' "$hint"
      return 0
    fi
  fi
  if [[ -z "$external_session_id" ]]; then
    return 1
  fi

  case "$runtime_cmd" in
    codex)
      printf 'codex resume %s' "$external_session_id"
      ;;
    *)
      return 1
      ;;
  esac
}
