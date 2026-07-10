#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_dir=""
min_age_days="${ORQUESTA_RUNTIME_RETENTION_MIN_AGE_DAYS:-14}"
delete_mode=0
confirm_delete=""

usage() {
  cat <<'USAGE'
Uso: scripts/orquesta_runtime_retention.sh [--repo PATH] [--runtime-dir PATH] [--min-age-days N] [--delete --confirm-delete orquesta-runtime-retention]

Inventaria candidatos de retencion local de Orquesta. Por defecto solo informa:
- hijos antiguos de .orquesta-runtime;
- hijos antiguos de contenedores .orquesta-runtime/codex-waves y .orquesta-runtime/waves;
- directorios .orquesta-purged-* en la raiz del repo.

No borra nada salvo con --delete y confirmacion literal. No borra la raiz
.orquesta-runtime ni contenedores globales como codex-waves/waves.
USAGE
}

fail() {
  printf 'orquesta_runtime_retention_failed: %s\n' "$1" >&2
  exit 2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo)
      [ "$#" -ge 2 ] || fail "repo_sin_valor"
      repo_root="$2"
      shift 2
      ;;
    --runtime-dir)
      [ "$#" -ge 2 ] || fail "runtime_dir_sin_valor"
      runtime_dir="$2"
      shift 2
      ;;
    --min-age-days)
      [ "$#" -ge 2 ] || fail "min_age_days_sin_valor"
      min_age_days="$2"
      shift 2
      ;;
    --delete)
      delete_mode=1
      shift
      ;;
    --confirm-delete)
      [ "$#" -ge 2 ] || fail "confirm_delete_sin_valor"
      confirm_delete="$2"
      shift 2
      ;;
    --dry-run)
      delete_mode=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "opcion_incompatible:$1"
      ;;
  esac
done

case "$min_age_days" in
  ''|*[!0-9]*)
    fail "min_age_days_invalido"
    ;;
esac

repo_root="$(cd "$repo_root" && pwd -P)"
if [ -z "$runtime_dir" ]; then
  runtime_dir="$repo_root/.orquesta-runtime"
fi

if [ "$delete_mode" -eq 1 ] && [ "$confirm_delete" != "orquesta-runtime-retention" ]; then
  fail "confirm_delete_requerido"
fi

runtime_abs=""
if [ -d "$runtime_dir" ]; then
  runtime_abs="$(cd "$runtime_dir" && pwd -P)"
fi

path_under() {
  local root="$1"
  local path="$2"
  [ "$path" != "$root" ] && [[ "$path" == "$root"/* ]]
}

safe_candidate() {
  local candidate="$1"
  [ -n "$candidate" ] || return 1
  [ "$candidate" != "/" ] || return 1
  [ "$candidate" != "." ] || return 1
  if [ -n "$runtime_abs" ] && path_under "$runtime_abs" "$candidate"; then
    case "$(basename "$candidate")" in
      codex-waves|waves)
        return 1
        ;;
    esac
    return 0
  fi
  case "$candidate" in
    "$repo_root"/.orquesta-purged-*)
      [ "$(dirname "$candidate")" = "$repo_root" ] && return 0
      ;;
  esac
  return 1
}

entry_size() {
  du -sh "$1" 2>/dev/null | awk '{print $1}'
}

entry_mtime() {
  date -r "$1" '+%Y-%m-%dT%H:%M:%S%z'
}

emit_candidate() {
  local action="$1"
  local reason="$2"
  local candidate="$3"
  printf 'candidate\taction=%s\treason=%s\tsize=%s\tmtime=%s\tpath=%s\n' \
    "$action" "$reason" "$(entry_size "$candidate")" "$(entry_mtime "$candidate")" "$candidate"
}

candidate_blockers() {
  local candidate="$1"
  local blockers=()
  while IFS= read -r -d '' path; do
    name="$(basename "$path")"
    dir="$(dirname "$path")"
    case "$name" in
      agent_ack.json)
        blockers+=("agent_ack_unreconciled")
        ;;
      director_decisions.json)
        blockers+=("director_decisions_pending")
        ;;
      plan.json|artifacts_manifest.json|artifact_manifest.json)
        blockers+=("durable_plan_or_artifact_manifest")
        ;;
      orquesta_shutdown_request.json)
        if [ ! -e "$dir/agent_shutdown_checkpoint_ack.json" ]; then
          blockers+=("checkpoint_pending")
        fi
        ;;
      *)
        low="$(printf '%s' "$name" | tr '[:upper:]' '[:lower:]')"
        case "$low" in
          *outbox*|*plan_state*)
            blockers+=("open_state_or_outbox_pending")
            ;;
          *artifact*manifest*|*artifacts*manifest*)
            blockers+=("durable_plan_or_artifact_manifest")
            ;;
        esac
        ;;
    esac
  done < <(find "$candidate" -xdev -type f \( \
    -name 'agent_ack.json' -o \
    -name 'director_decisions.json' -o \
    -name 'plan.json' -o \
    -name 'artifacts_manifest.json' -o \
    -name 'artifact_manifest.json' -o \
    -name 'orquesta_shutdown_request.json' -o \
    -iname '*outbox*' -o \
    -iname '*plan_state*' -o \
    -iname '*artifact*manifest*' -o \
    -iname '*artifacts*manifest*' \
  \) -print0 2>/dev/null)
  while IFS= read -r -d '' registry; do
    if grep -Eq '"status"[[:space:]]*:[[:space:]]*"(running|stop_requested)"' "$registry" 2>/dev/null; then
      blockers+=("agent_live")
    fi
    while IFS= read -r pid; do
      case "$pid" in
        ''|*[!0-9]*)
          continue
          ;;
      esac
      if [ "$pid" -gt 0 ] && kill -0 "$pid" 2>/dev/null; then
        blockers+=("agent_live")
      fi
    done < <(grep -Eo '"pid"[[:space:]]*:[[:space:]]*[0-9]+' "$registry" 2>/dev/null | awk -F: '{gsub(/[[:space:]]/, "", $2); print $2}')
  done < <(find "$candidate" -xdev -type f -name 'codex_wave_registry_v0.json' -print0 2>/dev/null)
  if [ "${#blockers[@]}" -eq 0 ]; then
    return 0
  fi
  printf '%s\n' "${blockers[@]}" | sort -u | paste -sd, -
}

handle_candidate() {
  local candidate="$1"
  local reason="$2"
  local abs
  local blockers
  abs="$(cd "$candidate" && pwd -P)"
  if ! safe_candidate "$abs"; then
    printf 'candidate\taction=blocked\treason=unsafe_path\tpath=%s\n' "$abs"
    return 0
  fi
  if { [ "$reason" = "old_runtime_wave" ] || [ "$reason" = "old_runtime_nested_wave" ]; } &&
    [ ! -f "$abs/codex_wave_registry_v0.json" ]; then
    printf 'candidate\taction=blocked\treason=wave_registry_missing\tsize=%s\tmtime=%s\tpath=%s\n' \
      "$(entry_size "$abs")" "$(entry_mtime "$abs")" "$abs"
    return 0
  fi
  blockers="$(candidate_blockers "$abs")"
  if [ -n "$blockers" ]; then
    printf 'candidate\taction=blocked\treason=%s\tsize=%s\tmtime=%s\tpath=%s\n' \
      "$blockers" "$(entry_size "$abs")" "$(entry_mtime "$abs")" "$abs"
    return 0
  fi
  if [ "$delete_mode" -eq 1 ]; then
    emit_candidate "delete" "$reason" "$abs"
    rm -rf -- "$abs"
    return 0
  fi
  emit_candidate "report" "$reason" "$abs"
}

printf 'schema_version=orquesta_runtime_retention_report.v0\n'
printf 'repo=%s\n' "$repo_root"
printf 'runtime_dir=%s\n' "${runtime_abs:-$runtime_dir}"
printf 'min_age_days=%s\n' "$min_age_days"
if [ "$delete_mode" -eq 1 ]; then
  printf 'mode=delete\n'
else
  printf 'mode=dry-run\n'
fi

if [ -n "$runtime_abs" ]; then
  for container in "$runtime_abs/codex-waves" "$runtime_abs/waves"; do
    [ -d "$container" ] || continue
    while IFS= read -r -d '' nested; do
      handle_candidate "$nested" "old_runtime_wave"
    done < <(find "$container" -mindepth 1 -maxdepth 1 -type d -mtime +"$min_age_days" -print0 2>/dev/null)
    # Legacy domain containers (for example codex-waves/<domain>/<wave>) have
    # no registry themselves. Inspect only their immediate wave children: a
    # deeper registry belongs to that wave and must be retained or purged with
    # its parent, never independently.
    while IFS= read -r -d '' registry; do
      nested="$(dirname "$registry")"
      handle_candidate "$nested" "old_runtime_nested_wave"
    done < <(find "$container" -mindepth 3 -maxdepth 3 -type f -name 'codex_wave_registry_v0.json' -mtime +"$min_age_days" -print0 2>/dev/null)
  done
  while IFS= read -r -d '' child; do
    name="$(basename "$child")"
    case "$name" in
      codex-waves|waves)
        continue
        ;;
      *)
        handle_candidate "$child" "old_runtime_entry"
        ;;
    esac
  done < <(find "$runtime_abs" -mindepth 1 -maxdepth 1 -type d -mtime +"$min_age_days" -print0 2>/dev/null)
fi

shopt -s nullglob
for purged in "$repo_root"/.orquesta-purged-*; do
  [ -d "$purged" ] || continue
  if find "$purged" -maxdepth 0 -type d -mtime +"$min_age_days" | grep -q .; then
    handle_candidate "$purged" "old_purged_dir"
  fi
done
