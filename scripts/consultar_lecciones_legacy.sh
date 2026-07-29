#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
index_path="${repository_root}/product/knowledge/legacy_preflight_patterns.jsonl"

capability=""
target_path=""
operation=""
state=""
output_json=0
show_all=0

usage() {
  printf '%s\n' \
    "Uso: scripts/consultar_lecciones_legacy.sh [filtros]" \
    "" \
    "  --capability ID   Capacidad exacta, por ejemplo ORC-03" \
    "  --path RUTA       Ruta del cambio, relativa al repositorio" \
    "  --operation OP    Operación exacta: plan, test, shutdown..." \
    "  --state ESTADO    enforced, partial" \
    "  --all             Muestra todos los patrones" \
    "  --json            Devuelve un array JSON" \
    "  --help            Muestra esta ayuda"
}

while (($#)); do
  case "$1" in
    --capability)
      capability="${2:?falta valor para --capability}"
      shift 2
      ;;
    --path)
      target_path="${2:?falta valor para --path}"
      shift 2
      ;;
    --operation)
      operation="${2:?falta valor para --operation}"
      shift 2
      ;;
    --state)
      state="${2:?falta valor para --state}"
      shift 2
      ;;
    --all)
      show_all=1
      shift
      ;;
    --json)
      output_json=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'Argumento desconocido: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ((show_all == 0)) && [[ -z "$capability" && -z "$target_path" && -z "$operation" && -z "$state" ]]; then
  printf 'Indica al menos un filtro o usa --all.\n' >&2
  usage >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  printf 'Falta jq; la consulta no usa IA ni red.\n' >&2
  exit 2
fi
if [[ ! -r "$index_path" ]]; then
  printf 'Índice no legible: %s\n' "$index_path" >&2
  exit 2
fi

result="$(
  jq -s \
    --arg capability "$capability" \
    --arg target_path "$target_path" \
    --arg operation "$operation" \
    --arg state "$state" \
    '
      def has_capability($value):
        ($value == "") or any(.capability_ids[]; . == $value);
      def has_operation($value):
        ($value == "") or any(.operations[]; . == $value);
      def has_state($value):
        ($value == "") or (.current_state == $value);
      def has_path($value):
        ($value == "") or any(
          .path_prefixes[];
          . as $prefix |
          $prefix == "*" or $prefix == $value or
          ($value | startswith($prefix + "/")) or
          ($prefix | startswith($value + "/"))
        );
      map(select(
        has_capability($capability) and
        has_path($target_path) and
        has_operation($operation) and
        has_state($state)
      )) | sort_by(.pattern_id)
    ' "$index_path"
)"

count="$(jq 'length' <<<"$result")"
if ((count == 0)); then
  printf 'Sin patrones para filtros dados.\n' >&2
  exit 1
fi

if ((output_json == 1)); then
  jq '.' <<<"$result"
  exit 0
fi

jq -r '
  .[] |
  "[\(.pattern_id)] \(.title)\n" +
  "estado: \(.current_state)\n" +
  "reusar: \(.reuse)\n" +
  "evitar: \(.avoid)\n" +
  (if .gap == "" then "" else "hueco: \(.gap)\n" end) +
  "evidencia: \(.evidence_refs | join(", "))\n"
' <<<"$result"
