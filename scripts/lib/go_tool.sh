#!/usr/bin/env bash

orquesta_go_tool_ensure_path() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi

  local home_go=""
  if [[ -n "${HOME:-}" ]]; then
    home_go=":${HOME}/go/bin/go"
  fi
  local candidates_raw="${ORQUESTA_GO_TOOL_CANDIDATES:-/usr/local/go/bin/go${home_go}:/usr/bin/go}"
  local old_ifs="$IFS"
  IFS=':'
  local candidate
  for candidate in $candidates_raw; do
    if [[ -x "$candidate" ]]; then
      PATH="${candidate%/*}:$PATH"
      export PATH
      IFS="$old_ifs"
      return 0
    fi
  done
  IFS="$old_ifs"

  echo "go: orden no encontrada; instala Go o define ORQUESTA_GO_TOOL_CANDIDATES con rutas absolutas separadas por ':'" >&2
  return 127
}
