#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
catalog="${repo_root}/product/knowledge/tooling_adoption_v1.json"
roadmap="${repo_root}/product/roadmap.json"

status=""
capability=""
term=""
show_all=false
json=false

usage() {
  printf '%s\n' \
    "Uso: scripts/consultar_herramientas.sh [filtro]" \
    "  --all" \
    "  --status accredited|partial|candidate|conditional|debt" \
    "  --capability ID" \
    "  --term TEXTO" \
    "  --json"
}

while (($# > 0)); do
  case "$1" in
    --all)
      show_all=true
      shift
      ;;
    --status)
      (($# >= 2)) || { usage >&2; exit 2; }
      status="$2"
      shift 2
      ;;
    --capability)
      (($# >= 2)) || { usage >&2; exit 2; }
      capability="$2"
      shift 2
      ;;
    --term)
      (($# >= 2)) || { usage >&2; exit 2; }
      term="$2"
      shift 2
      ;;
    --json)
      json=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "${show_all}" != true && -z "${status}" && -z "${capability}" && -z "${term}" ]]; then
  usage >&2
  exit 2
fi

query='
  $roadmap[0] as $roadmap_document |
  ([
    $roadmap_document |
    .. |
    objects |
    select(.id? != null and .status? != null) |
    {key: .id, value: .status}
  ] | from_entries) as $live_status |
  [
    .entries[] |
    . + {
      live_capabilities: [
        .capability_ids[] as $id |
        {
          id: $id,
          roadmap_status: ($live_status[$id] // "missing")
        }
      ]
    } |
    select(
      ($status == "" or .status == $status) and
      ($capability == "" or (.capability_ids | index($capability)) != null) and
      (
        $term == "" or
        (
          [
            .key,
            .name,
            .owner_vertical,
            .decision,
            .next_gate,
            (.capability_ids | join(" "))
          ] |
          join(" ") |
          ascii_downcase |
          contains($term | ascii_downcase)
        )
      )
    )
  ]
'

if [[ "${json}" == true ]]; then
  output="$(
    jq --arg status "${status}" \
      --arg capability "${capability}" \
      --arg term "${term}" \
      --slurpfile roadmap "${roadmap}" \
      "${query}" "${catalog}"
  )"
else
  output="$(
    jq -r --arg status "${status}" \
      --arg capability "${capability}" \
      --arg term "${term}" \
      --slurpfile roadmap "${roadmap}" \
      "${query} |
      .[] |
      [
        .key,
        .status,
        .owner_vertical,
        (
          .live_capabilities |
          map(.id + \"=\" + .roadmap_status) |
          join(\",\")
        ),
        .decision,
        .next_gate
      ] |
      @tsv" "${catalog}"
  )"
fi

if [[ -z "${output}" || "${output}" == "[]" ]]; then
  printf 'status=no_matches\n' >&2
  exit 1
fi

printf '%s\n' "${output}"
