#!/usr/bin/env bash

smoke_require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta herramienta requerida: $1" >&2
    exit 2
  fi
}

smoke_require_tools() {
  local tool
  for tool in "$@"; do
    smoke_require_tool "$tool"
  done
}

smoke_require_confirm() {
  local var_name="$1"
  local expected="${2:-1}"
  local message="$3"
  local current="${!var_name:-}"
  if [[ "$current" != "$expected" ]]; then
    echo "$message" >&2
    exit 2
  fi
}

smoke_is_local_url() {
  case "$1" in
    http://127.0.0.1|http://127.0.0.1:*|http://localhost|http://localhost:*|http://[::1]|http://[::1]:*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

smoke_require_opes_temporal_destination() {
  local url="$1"
  local label="${2:-OPES_BASE_URL}"
  if [[ "${ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM:-0}" == "1" ||
    "${ORQUESTA_OPES_PRODUCTIVE_CONFIRM:-0}" == "1" ]]; then
    echo "smoke OPES bloqueado: no ejecutar contra OPES productivo" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_TEMPORAL_CONFIRM:-0}" != "1" ]]; then
    echo "smoke OPES bloqueado: confirma instancia temporal con ORQUESTA_OPES_TEMPORAL_CONFIRM=1" >&2
    exit 2
  fi
  if [[ -z "$url" ]]; then
    echo "smoke OPES bloqueado: define $label apuntando a OPES temporal" >&2
    exit 2
  fi
  if ! smoke_is_local_url "$url" &&
    [[ "${ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL:-0}" != "1" ]]; then
    echo "smoke OPES bloqueado: $label no parece loopback: $(smoke_public_url_ref "$url")" >&2
    echo "si es temporal no local, exporta ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL=1" >&2
    exit 2
  fi
}

smoke_public_url_ref() {
  printf '%s' "$1" |
    sed -E 's#(https?://)[^/@[:space:]]+@#\1[redacted]@#; s#\?.*$#?[redacted]#'
}

smoke_print_file_excerpt() {
  local file="$1"
  local max_bytes="${SMOKE_ERROR_BYTES:-2000}"
  if [[ -s "$file" ]]; then
    head -c "$max_bytes" "$file" >&2 || true
    echo >&2
  fi
}

smoke_post_json() {
  local url="$1"
  local payload="$2"
  local output="$3"
  local status
  status="$(curl -sS -o "$output" -w "%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "$payload")"
  if [[ "$status" -lt 200 || "$status" -gt 299 ]]; then
    echo "POST $(smoke_public_url_ref "$url") devolvio HTTP $status" >&2
    smoke_print_file_excerpt "$output"
    return 1
  fi
}

smoke_get_json() {
  local url="$1"
  local output="$2"
  curl -sS -f "$url" >"$output"
}

smoke_json_id() {
  jq -r '.id // .ID // empty' "$1"
}

smoke_json_count() {
  jq 'if type == "array" then length else (.artifacts // .blocks // [] | length) end' \
    "$1" 2>/dev/null || echo 0
}

smoke_wait_orquesta_readiness_from_state_file() {
  local state_file="$1"
  local polls="${2:-80}"
  local sleep_seconds="${3:-0.5}"
  local out_var="$4"
  local addr=""
  local url=""
  local _
  for _ in $(seq 1 "$polls"); do
    if [[ -s "$state_file" ]]; then
      addr="$(jq -r '.addr // empty' "$state_file")"
      if [[ -n "$addr" ]]; then
        url="http://$addr"
        if curl -fsS -m 2 "$url/api/v0/server/readiness" >/dev/null; then
          printf -v "$out_var" '%s' "$url"
          return 0
        fi
      fi
    fi
    sleep "$sleep_seconds"
  done
  return 1
}

smoke_shutdown_orquesta_server() {
  local server_pid="$1"
  local base_url="${2:-}"
  local shutdown_timeout="${3:-5}"
  local grace_polls="${4:-25}"
  if [[ -z "$server_pid" ]] || ! kill -0 "$server_pid" >/dev/null 2>&1; then
    return 0
  fi
  if [[ -n "$base_url" ]] && command -v curl >/dev/null 2>&1; then
    curl -sS -m "$shutdown_timeout" -X POST "$base_url/api/v0/server/shutdown" >/dev/null 2>&1 || true
    local _
    for _ in $(seq 1 "$grace_polls"); do
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        wait "$server_pid" >/dev/null 2>&1 || true
        return 0
      fi
      sleep 0.2
    done
  fi
  if kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -INT "$server_pid" >/dev/null 2>&1 || true
    local _
    for _ in $(seq 1 "$grace_polls"); do
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        wait "$server_pid" >/dev/null 2>&1 || true
        return 0
      fi
      sleep 0.2
    done
  fi
  if kill -0 "$server_pid" >/dev/null 2>&1; then
    kill -TERM "$server_pid" >/dev/null 2>&1 || true
  fi
  wait "$server_pid" >/dev/null 2>&1 || true
}

smoke_temp_root_abs() {
  local path="$1"
  if command -v realpath >/dev/null 2>&1; then
    realpath -m "$path"
    return
  fi
  if [[ "$path" == "/" ]]; then
    printf '/\n'
    return
  fi
  case "$path" in
    /*)
      printf '%s\n' "${path%/}"
      ;;
    *)
      printf '%s/%s\n' "$(pwd -P)" "${path#./}"
      ;;
  esac
}

smoke_temp_root_basename() {
  local path="${1%/}"
  printf '%s' "${path##*/}"
}

smoke_temp_root_public_ref() {
  local root="$1"
  local abs
  abs="$(smoke_temp_root_abs "$root")"
  printf 'temp-root:%s' "$(smoke_temp_root_basename "$abs")"
}

smoke_temp_root_allowed_prefix() {
  local abs="$1"
  local prefix
  local tmp_abs
  for prefix in "${TMPDIR:-/tmp}" /tmp /var/tmp; do
    tmp_abs="$(smoke_temp_root_abs "$prefix")"
    if [[ "$abs" == "$tmp_abs"/* && "$abs" != "$tmp_abs" ]]; then
      return 0
    fi
  done
  if [[ -n "${ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES:-}" ]]; then
    local old_ifs="$IFS"
    IFS=:
    for prefix in $ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES; do
      IFS="$old_ifs"
      [[ -z "$prefix" ]] && continue
      tmp_abs="$(smoke_temp_root_abs "$prefix")"
      if [[ "$abs" == "$tmp_abs"/* && "$abs" != "$tmp_abs" ]]; then
        return 0
      fi
      IFS=:
    done
    IFS="$old_ifs"
  fi
  return 1
}

smoke_temp_root_validate() {
  local root="$1"
  SMOKE_TEMP_ROOT_REASON=""
  if [[ -z "$root" ]]; then
    SMOKE_TEMP_ROOT_REASON="empty-root"
    return 1
  fi
  local abs
  abs="$(smoke_temp_root_abs "$root")"
  if [[ "$abs" == "/" || "$abs" == "." ]]; then
    SMOKE_TEMP_ROOT_REASON="forbidden-root"
    return 1
  fi
  case "$abs" in
    */.orquesta-runtime|*/.orquesta-runtime/*)
      SMOKE_TEMP_ROOT_REASON="forbidden-orquesta-runtime"
      return 1
      ;;
  esac
  local project_root="${ORQUESTA_SMOKE_PROJECT_ROOT:-${repo_root:-${ROOT_DIR:-${ROOT:-$(pwd -P)}}}}"
  local project_abs
  project_abs="$(smoke_temp_root_abs "$project_root")"
  if [[ "$abs" == "$project_abs" || "$abs" == "$project_abs"/* ]]; then
    SMOKE_TEMP_ROOT_REASON="forbidden-project-root"
    return 1
  fi
  if [[ -n "${HOME:-}" ]]; then
    local home_abs
    home_abs="$(smoke_temp_root_abs "$HOME")"
    if [[ "$abs" == "$home_abs" || "$abs" == "$home_abs"/* ]]; then
      SMOKE_TEMP_ROOT_REASON="forbidden-home-root"
      return 1
    fi
  fi
  if ! smoke_temp_root_allowed_prefix "$abs"; then
    SMOKE_TEMP_ROOT_REASON="outside-allowed-temp-prefix"
    return 1
  fi
}

smoke_temp_root_marker_path() {
  printf '%s/.orquesta-smoke-root.v0' "$1"
}

smoke_temp_root_has_marker() {
  local root="$1"
  local marker
  marker="$(smoke_temp_root_marker_path "$root")"
  local first_line=""
  [[ -f "$marker" ]] || return 1
  IFS= read -r first_line <"$marker" || true
  [[ "$first_line" == "schema_version=orquesta_smoke_temp_root.v0" ]]
}

smoke_temp_root_prepare() {
  local root="$1"
  local source_ref="${2:-generated}"
  if ! smoke_temp_root_validate "$root"; then
    echo "smoke_temp_root_blocked reason=$SMOKE_TEMP_ROOT_REASON root_ref=$(smoke_temp_root_public_ref "$root") source=$source_ref" >&2
    return 1
  fi
  if [[ -d "$root" ]] && ! smoke_temp_root_has_marker "$root"; then
    local old_nullglob
    local old_dotglob
    old_nullglob="$(shopt -p nullglob || true)"
    old_dotglob="$(shopt -p dotglob || true)"
    shopt -s nullglob dotglob
    local children=("$root"/*)
    eval "$old_nullglob"
    eval "$old_dotglob"
    if ((${#children[@]} > 0)); then
      echo "smoke_temp_root_blocked reason=unmarked-nonempty-root root_ref=$(smoke_temp_root_public_ref "$root") source=$source_ref" >&2
      return 1
    fi
  fi
  mkdir -p "$root"
  if ! smoke_temp_root_has_marker "$root"; then
    {
      echo "schema_version=orquesta_smoke_temp_root.v0"
      echo "source=$source_ref"
    } >"$(smoke_temp_root_marker_path "$root")"
  fi
}

smoke_temp_root_cleanup() {
  local root="$1"
  local keep="${2:-0}"
  if [[ "$keep" == "1" ]]; then
    echo "directorio conservado: $(smoke_temp_root_public_ref "$root") reason=keep-smoke-dir" >&2
    return 0
  fi
  if ! smoke_temp_root_validate "$root"; then
    echo "smoke_temp_root_preserved reason=$SMOKE_TEMP_ROOT_REASON root_ref=$(smoke_temp_root_public_ref "$root")" >&2
    return 0
  fi
  if ! smoke_temp_root_has_marker "$root"; then
    echo "smoke_temp_root_preserved reason=missing-marker root_ref=$(smoke_temp_root_public_ref "$root")" >&2
    return 0
  fi
  local rm_bin="${ORQUESTA_SMOKE_RM_BIN:-}"
  if [[ -z "$rm_bin" ]]; then
    rm_bin="$(command -v rm || true)"
  fi
  if [[ -z "$rm_bin" && -x /bin/rm ]]; then
    rm_bin="/bin/rm"
  fi
  if [[ -z "$rm_bin" ]]; then
    echo "smoke_temp_root_preserved reason=rm-unavailable root_ref=$(smoke_temp_root_public_ref "$root")" >&2
    return 0
  fi
  if command -v chmod >/dev/null 2>&1; then
    chmod -R u+w -- "$root" 2>/dev/null || true
  fi
  "$rm_bin" -rf -- "$root"
}
