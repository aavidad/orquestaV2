#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
index_path="${repository_root}/product/traceability/source_dispositions.jsonl"
roles_path="${repository_root}/product/traceability/markdown_source_roles.jsonl"

capability=""
kind=""
text_filter=""
source_ref=""
show_all=0
summary=0
output_json=0

usage() {
  printf '%s\n' \
    "Uso: scripts/consultar_fuentes_legacy.sh [filtros]" \
    "" \
    "  --capability ID   Capability V2 exacta" \
    "  --kind TIPO       bug, skill, task o context" \
    "  --text TEXTO      Subcadena en ruta, razón, decisión o evidence_state" \
    "  --source RUTA     source_ref exacta" \
    "  --all             Devuelve las 840 fuentes Markdown clasificadas" \
    "  --summary         Resume el inventario de fuentes" \
    "  --json            Array JSON; por defecto JSONL compacto" \
    "  --help            Muestra esta ayuda"
}

while (($#)); do
  case "$1" in
    --capability) capability="${2:?falta valor para --capability}"; shift 2 ;;
    --kind) kind="${2:?falta valor para --kind}"; shift 2 ;;
    --text) text_filter="${2:?falta valor para --text}"; shift 2 ;;
    --source) source_ref="${2:?falta valor para --source}"; shift 2 ;;
    --all) show_all=1; shift ;;
    --summary) summary=1; shift ;;
    --json) output_json=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) printf 'Argumento desconocido: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -n "$capability" && ! "$capability" =~ ^[A-Z]+-[0-9]+$ ]]; then
  printf 'Capability inválida: %s\n' "$capability" >&2
  exit 2
fi
if [[ -n "$kind" && "$kind" != "bug" && "$kind" != "skill" &&
  "$kind" != "task" && "$kind" != "context" ]]; then
  printf 'Kind inválido: %s\n' "$kind" >&2
  exit 2
fi
if ((show_all == 0 && summary == 0)) &&
  [[ -z "$capability" && -z "$kind" && -z "$text_filter" && -z "$source_ref" ]]; then
  printf '%s\n' 'Indica al menos un filtro, --all o --summary.' >&2
  exit 2
fi
if [[ ! -r "$index_path" || ! -r "$roles_path" ]] || ! command -v jq >/dev/null 2>&1; then
  printf '%s\n' 'Falta el índice de fuentes o jq.' >&2
  exit 2
fi

if ! jq -e -s '
  length == 339 and
  (map(.source_ref) | unique | length) == 339 and
  all(.[];
    .schema_version == 1 and
    (.kind == "bug" or .kind == "skill" or .kind == "task") and
    (.source_ref | length) > 0 and
    (.source_sha256 | test("^sha256:[0-9a-f]{64}$")) and
    (.capability_ids | length) > 0 and
    all(.capability_ids[]; test("^[A-Z]+-[0-9]+$")) and
    (.reason_code | length) > 0 and
    (.decision == "accepted" or .decision == "conditional" or
      .decision == "historical") and
    (if .kind == "bug" then
      (.lesson_id | length) > 0 and (.invariant_test_ref | length) > 0 and
      .lesson_state == "pending_invariant_test"
    else true end)
  )
' "$index_path" >/dev/null; then
  printf '%s\n' 'El inventario de fuentes no satisface su contrato 339/339.' >&2
  exit 2
fi

if ! jq -e -s --slurpfile dispositions "$index_path" '
  (map(.source_ref) | unique) as $markdown_refs |
  length == 840 and
  (map(.source_ref) | unique | length) == 840 and
  all(.[];
    .schema_version == 1 and
    (.source_ref | length) > 0 and
    (.source_sha256 | test("^sha256:[0-9a-f]{64}$")) and
    (.classification | length) > 0 and
    (.classification_basis | length) > 0 and
    (.roles | length) > 0) and
  all($dispositions[]; .source_ref as $ref | ($markdown_refs | index($ref)) != null)
' "$roles_path" >/dev/null; then
  printf '%s\n' 'El censo Markdown no satisface su contrato 840/840.' >&2
  exit 2
fi

inventory="$(jq -n \
  --slurpfile dispositions "$index_path" \
  --slurpfile roles "$roles_path" '
    ([$dispositions[].source_ref] | unique) as $actionable_refs |
    ($dispositions | map(. + {
      actionable: true,
      classification: null,
      roles: []
    })) +
    ($roles | map(select(.source_ref as $ref | ($actionable_refs | index($ref)) == null) |
      {
        schema_version,
        kind: "context",
        source_ref,
        source_sha256,
        capability_ids: [],
        decision: "context_only",
        reason_code: .classification,
        evidence_state: "classified_context_not_actionable_source",
        classification,
        roles,
        origin,
        classification_basis,
        actionable: false
      }))
  ')"

selection="$(jq \
  --arg capability "$capability" \
  --arg kind "$kind" \
  --arg text_filter "$text_filter" \
  --arg source_ref "$source_ref" '
    map(select(
      ($capability == "" or (.capability_ids | index($capability)) != null) and
      ($kind == "" or .kind == $kind) and
      ($source_ref == "" or .source_ref == $source_ref) and
      ($text_filter == "" or
        (([.source_ref, .kind, .decision, .reason_code, .evidence_state] |
          join(" ") | ascii_downcase) | contains($text_filter | ascii_downcase)))
    )) | sort_by(.kind, .source_ref)
  ' <<<"$inventory")"

if ((summary == 1)); then
  jq '
    {
      schema_version: 1,
      markdown_sources: length,
      actionable_sources: (map(select(.actionable == true)) | length),
      context_only_sources: (map(select(.actionable == false)) | length),
      kinds: (group_by(.kind) | map({key:.[0].kind,value:length}) | from_entries),
      decisions: (group_by(.decision) |
        map({key:.[0].decision,value:length}) | from_entries),
      sources_with_capabilities: (map(select((.capability_ids | length) > 0)) | length),
      bug_sources_with_pending_invariant_test:
        (map(select(.kind == "bug" and .lesson_state == "pending_invariant_test")) | length),
      depth: "classified_markdown_source_inventory",
      authority: "derived_advisory_from_source_dispositions",
      claims_historical_success: false,
      claims_current_bug: false,
      claims_reuse_decision: false
    }
  ' <<<"$selection"
  exit 0
fi

if [[ "$(jq 'length' <<<"$selection")" == "0" ]]; then
  printf '%s\n' 'Sin fuentes legacy para los filtros dados.' >&2
  exit 1
fi

cards="$(jq '
  map({
    schema_version,
    kind,
    source_ref,
    source_sha256,
    capability_ids,
    decision,
    reason_code,
    evidence_state,
    classification: (.classification // null),
    roles: (.roles // []),
    actionable,
    lesson_id: (.lesson_id // null),
    invariant_test_ref: (.invariant_test_ref // null),
    lesson_state: (.lesson_state // null),
    depth:
      (if .actionable then "source_disposition_lead_not_content_assessment"
      else "classified_context_not_actionable_source" end),
    recommended_next_action:
      (if .kind == "bug" then
        "Abrir la fuente y convertir solo el fallo aún aplicable en negativo o test; no asumir que el bug siga vivo en V2."
      elif .kind == "skill" then
        "Comparar la idea o herramienta con el registro de adopción V2 antes de conservarla; no importar runtime legacy."
      elif .kind == "task" then
        "Usar consultar_pistas_legacy.sh para bajar de esta fuente a TASKENTRY revisadas y después a una ficha profunda."
      else
        "Usar esta fuente solo como contexto o referencia; no inferir una conducta reutilizable sin una pista accionable y revisión adicional."
      end),
    authority: "derived_advisory_from_source_dispositions",
    claims_historical_success: false,
    claims_current_bug: false,
    claims_reuse_decision: false
  })
' <<<"$selection")"

if ((output_json == 1)); then
  jq '.' <<<"$cards"
else
  jq -c '.[]' <<<"$cards"
fi
