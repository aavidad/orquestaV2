#!/usr/bin/env bash
set -euo pipefail

# Lectura MEJ-106 2026-07-04: base env=511; actual temporal=512 por
# env_vars_orquesta_allow_increase_to=512 documentado. status=16,
# interfaces=65, director=17.

usage() {
  echo "usage: scripts/orquesta_metricas_deuda.sh [--json]" >&2
}

fail_not_repo_root() {
  echo "orquesta_metricas_deuda: run from a repo root containing modulos/ and cmd/" >&2
  exit 2
}

json=false
case "$#" in
  0)
    ;;
  1)
    case "$1" in
      --json)
        json=true
        ;;
      *)
        usage
        exit 2
        ;;
    esac
    ;;
  *)
    usage
    exit 2
    ;;
esac

if [ ! -d modulos ] || [ ! -d cmd ]; then
  fail_not_repo_root
fi

go_grep() {
  local pattern="$1"
  find modulos cmd -type f -name '*.go' -exec grep -hEo "$pattern" {} + 2>/dev/null || true
}

count_lines() {
  wc -l | tr -d '[:space:]'
}

env_vars_orquesta="$(
  go_grep 'ORQUESTA_[A-Z0-9_]+' |
    sort -u |
    count_lines
)"

endpoints_status="$(
  go_grep '"/api/v0/[^"]*"' |
    sed 's/^"//; s/"$//' |
    awk '{ path=tolower($0); if (path ~ /(status|stats|readiness|observe|health|cockpit|control)/) print $0 }' |
    sort -u |
    count_lines
)"

interfaces_estado="$(
  {
    find modulos cmd -type f -name '*.go' -exec grep -hE '^[[:space:]]*type[[:space:]]+[A-Za-z_][A-Za-z0-9_]*(\[[^]]+\])?[[:space:]]+interface([[:space:]]|\{)' {} + 2>/dev/null ||
      true
  } |
    awk '{
      for (i = 1; i <= NF; i++) {
        if ($i == "type") {
          name = $(i + 1)
          sub(/\[.*/, "", name)
          lower = tolower(name)
          if (lower ~ /(store|registry|ledger|outbox|state|marker|snapshot|queue)/) {
            print name
          }
          break
        }
      }
    }' |
    sort -u |
    count_lines
)"

modulos_director="$(
  find modulos -mindepth 1 -maxdepth 1 -type d -name '*director*' |
    count_lines
)"

if [ "$json" = true ]; then
  printf '{"env_vars_orquesta":%s,"endpoints_status":%s,"interfaces_estado":%s,"modulos_director":%s}\n' \
    "$env_vars_orquesta" \
    "$endpoints_status" \
    "$interfaces_estado" \
    "$modulos_director"
else
  printf 'env_vars_orquesta=%s\n' "$env_vars_orquesta"
  printf 'endpoints_status=%s\n' "$endpoints_status"
  printf 'interfaces_estado=%s\n' "$interfaces_estado"
  printf 'modulos_director=%s\n' "$modulos_director"
fi
