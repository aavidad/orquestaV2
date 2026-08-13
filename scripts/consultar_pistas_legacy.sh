#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
canonical_task_entries_path="${repository_root}/product/traceability/task_entries.jsonl"
task_entries_path="$canonical_task_entries_path"
roadmap_path="${repository_root}/product/roadmap.json"
canonical_mode=1

# Solo los tests pueden sustituir el ledger. La doble opt-in evita que una variable
# heredada cambie silenciosamente la autoridad consultada por un agente.
if [[ -n "${ORQUESTA_LEGACY_TASK_ENTRIES_PATH:-}" ]]; then
  if [[ "${ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE:-}" != "1" ]]; then
    printf '%s\n' \
      'ORQUESTA_LEGACY_TASK_ENTRIES_PATH solo se admite con ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE=1.' >&2
    exit 2
  fi
  task_entries_path="$ORQUESTA_LEGACY_TASK_ENTRIES_PATH"
  canonical_mode=0
fi

capability=""
text_filter=""
source_filter=""
entry_ref=""
show_all=0
summary=0
output_json=0
output_jsonl=0
limit=20
offset=0

usage() {
  printf '%s\n' \
    "Uso: scripts/consultar_pistas_legacy.sh [filtros]" \
    "" \
    "  --capability ID    Capability exacta, por ejemplo ORC-01" \
    "  --text TEXTO       Busca texto literal sin distinguir mayúsculas" \
    "  --source TEXTO     Filtra source_ref por subcadena literal" \
    "  --entry-ref REF    TASKENTRY exacta" \
    "  --limit N          Máximo de fichas (1–200; defecto 20)" \
    "  --offset N         Desplazamiento para paginar (defecto 0)" \
    "  --all              Omite paginación e incluye toda la selección" \
    "  --summary          Resume revisión y cobertura semántica del ledger" \
    "  --json             Devuelve fichas compactas como array JSON" \
    "  --jsonl            Devuelve fichas compactas como JSONL" \
    "  --help             Muestra esta ayuda" \
    "" \
    "Cada ficha es una pista semántica advisory: no evalúa verde histórico," \
    "conducta profunda ni correspondencia con funciones legacy."
}

require_value() {
  local option="$1"
  local remaining="$2"
  local value="${3:-}"
  if ((remaining < 2)) || [[ -z "$value" || "$value" == --* ]]; then
    printf 'Falta valor para %s.\n' "$option" >&2
    exit 2
  fi
}

while (($#)); do
  case "$1" in
    --capability)
      require_value "$1" "$#" "${2:-}"
      capability="$2"
      shift 2
      ;;
    --text)
      require_value "$1" "$#" "${2:-}"
      text_filter="$2"
      shift 2
      ;;
    --source)
      require_value "$1" "$#" "${2:-}"
      source_filter="$2"
      shift 2
      ;;
    --entry-ref)
      require_value "$1" "$#" "${2:-}"
      entry_ref="$2"
      shift 2
      ;;
    --limit)
      require_value "$1" "$#" "${2:-}"
      limit="$2"
      shift 2
      ;;
    --offset)
      require_value "$1" "$#" "${2:-}"
      offset="$2"
      shift 2
      ;;
    --all)
      show_all=1
      shift
      ;;
    --summary)
      summary=1
      shift
      ;;
    --json)
      output_json=1
      shift
      ;;
    --jsonl)
      output_jsonl=1
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

if [[ -n "$capability" && ! "$capability" =~ ^[A-Z]+-[0-9]+$ ]]; then
  printf 'Capability inválida: %s\n' "$capability" >&2
  exit 2
fi
if [[ -n "$entry_ref" && ! "$entry_ref" =~ ^TASKENTRY-[0-9a-f]{24}$ ]]; then
  printf 'entry_ref inválida: %s\n' "$entry_ref" >&2
  exit 2
fi
if [[ ! "$limit" =~ ^[0-9]+$ ]] || ((limit < 1 || limit > 200)); then
  printf '%s\n' 'El límite debe estar entre 1 y 200.' >&2
  exit 2
fi
if [[ ! "$offset" =~ ^[0-9]+$ ]]; then
  printf '%s\n' 'El offset debe ser un entero no negativo.' >&2
  exit 2
fi
if ((output_json == 1 && output_jsonl == 1)); then
  printf '%s\n' '--json y --jsonl son incompatibles.' >&2
  exit 2
fi
if ((summary == 1 && (output_json == 1 || output_jsonl == 1))); then
  printf '%s\n' '--summary ya devuelve JSON y no admite --json ni --jsonl.' >&2
  exit 2
fi
if ((show_all == 0 && summary == 0)) &&
  [[ -z "$capability" && -z "$text_filter" && -z "$source_filter" && -z "$entry_ref" ]]; then
  printf '%s\n' 'Indica al menos un filtro, --all o --summary.' >&2
  usage >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  printf '%s\n' 'Falta jq; la consulta es local y no usa IA ni red.' >&2
  exit 2
fi
if [[ ! -r "$task_entries_path" || ! -f "$task_entries_path" ]]; then
  printf 'Ledger de pistas no legible: %s\n' "$task_entries_path" >&2
  exit 2
fi
if [[ ! -r "$roadmap_path" || ! -f "$roadmap_path" ]]; then
  printf 'Roadmap V2 no legible: %s\n' "$roadmap_path" >&2
  exit 2
fi

# Validación fail-closed: JSONL, schema, campos que proyectamos y unicidad.
if ! jq -e -s '
  length > 0 and
  (map(.entry_ref) | unique | length) == length and
  all(.[];
    .schema_version == 2 and
    (.entry_ref | type) == "string" and
    (.entry_ref | test("^TASKENTRY-[0-9a-f]{24}$")) and
    (.capability_id | type) == "string" and
    (.capability_id | test("^[A-Z]+-[0-9]+$")) and
    (.semantic_review_state == "reviewed") and
    (.semantic_reason | type) == "string" and
    (.semantic_reason | length) > 0 and
    (.source_ref | type) == "string" and
    (.source_ref | length) > 0 and
    (.source_sha256 | type) == "string" and
    (.source_sha256 | test("^sha256:[0-9a-f]{64}$")) and
    (.subject_first_line | type) == "number" and
    .subject_first_line >= 1 and
    (.subject_last_line | type) == "number" and
    .subject_last_line >= .subject_first_line and
    (.subject_sha256 | type) == "string" and
    (.subject_sha256 | test("^sha256:[0-9a-f]{64}$")) and
    (.disposition | type) == "string" and
    (.disposition | length) > 0 and
    (.original_state | type) == "string" and
    (.original_state | length) > 0 and
    (.closure_evidence | type) == "string" and
    (.closure_evidence | length) > 0
  )
' "$task_entries_path" >/dev/null; then
  printf 'Ledger de pistas inválido, duplicado o con shape no reconocido: %s\n' \
    "$task_entries_path" >&2
  exit 2
fi

# El snapshot canónico tiene ratchet explícito. Un cambio exige revisar esta
# herramienta para no anunciar cobertura que ya no corresponde al ledger.
if ((canonical_mode == 1)); then
  if ! jq -e -s '
    length == 2108 and
    (map(.entry_ref) | unique | length) == 2108 and
    (map(select(.semantic_review_state == "reviewed")) | length) == 2108 and
    (map(select((.semantic_reason | length) > 0)) | length) == 2108 and
    (map(.capability_id) | unique | length) == 184
  ' "$task_entries_path" >/dev/null; then
    printf '%s\n' \
      'El ledger canónico ya no satisface el ratchet 2108/2108 reviewed, razones completas y 184 capabilities.' >&2
    exit 2
  fi
fi

# Es un programa jq: los $nombres se resuelven mediante --arg, no por Bash.
# shellcheck disable=SC2016
jq_filter='
  def matches:
    ($capability == "" or .capability_id == $capability) and
    ($entry_ref == "" or .entry_ref == $entry_ref) and
    ($source_filter == "" or
      ((.source_ref | ascii_downcase) | contains($source_filter | ascii_downcase))) and
    ($text_filter == "" or
      (([
        .entry_ref,
        .capability_id,
        .semantic_reason,
        .source_ref,
        (.source_entry_id // ""),
        .disposition,
        .original_state,
        .closure_evidence
      ] | join(" ") | ascii_downcase) |
        contains($text_filter | ascii_downcase)));
  map(select(matches)) |
  sort_by([
    (if $text_filter != "" and
      ((.semantic_reason | ascii_downcase) |
        contains($text_filter | ascii_downcase)) then 0 else 1 end),
    .entry_ref
  ])
'

selection="$(jq -s \
  --arg capability "$capability" \
  --arg text_filter "$text_filter" \
  --arg source_filter "$source_filter" \
  --arg entry_ref "$entry_ref" \
  "${jq_filter}" \
  "$task_entries_path")"

selection_count="$(jq 'length' <<<"$selection")"
if ((summary == 0 && selection_count == 0)); then
  printf '%s\n' 'Sin pistas legacy para los filtros dados.' >&2
  exit 1
fi

page="$selection"
if ((show_all == 0)); then
  page="$(jq --argjson offset "$offset" --argjson limit "$limit" \
    '.[ $offset : ($offset + $limit) ]' <<<"$selection")"
fi

if ((summary == 1)); then
  jq -s \
    --argjson matched_entries "$selection_count" '
      {
        schema_version: 1,
        authority: "derived_advisory_from_task_entry_ledger",
        entries_total: length,
        unique_entry_refs: (map(.entry_ref) | unique | length),
        entries_reviewed:
          (map(select(.semantic_review_state == "reviewed")) | length),
        semantic_reasons_non_empty:
          (map(select((.semantic_reason | length) > 0)) | length),
        capabilities: (map(.capability_id) | unique | length),
        matched_entries: $matched_entries,
        review_coverage_percent:
          (((map(select(.semantic_review_state == "reviewed")) | length) * 10000 /
            length | floor) / 100),
        semantic_reason_coverage_percent:
          (((map(select((.semantic_reason | length) > 0)) | length) * 10000 /
            length | floor) / 100),
        dispositions:
          (group_by(.disposition) |
            map({key: .[0].disposition, value: length}) |
            from_entries),
        closure_evidence_states:
          (group_by(.closure_evidence) |
            map({key: .[0].closure_evidence, value: length}) |
            from_entries),
        depth: "ledger_semantic_lead_not_deep_behavior_assessment",
        historical_green: "not_assessed",
        function_mapping: "not_assessed",
        note: "Reviewed significa asignación semántica en el ledger; no prueba acierto, fallo, verde histórico, reutilización ni acreditación V2.",
        canonical_state_change: false,
        claims_historical_success: false,
        claims_reuse_decision: false
      }
    ' "$task_entries_path"
  exit 0
fi

# La proyección se repite aquí de forma explícita para que JSON y JSONL tengan
# exactamente el mismo shape advisory y nunca filtren la fila canónica cruda.
cards="$(jq --slurpfile roadmap "$roadmap_path" \
  --argjson total_matches "$selection_count" \
  --argjson offset "$offset" \
  --argjson limit "$limit" \
  --argjson unpaginated "$show_all" '
  def v2_capability($id):
    first($roadmap[0].capability_entries[] | select(.id == $id));
  (length) as $page_count |
  map({
    schema_version: 1,
    entry_ref,
    capability: .capability_id,
    semantic_reason,
    source_ref,
    subject_first_line,
    subject_last_line,
    source_sha256,
    subject_sha256,
    disposition,
    original_state,
    closure_evidence,
    v2: (v2_capability(.capability_id) | {
      title,
      decision,
      status,
      acceptance_contracts: (.acceptance_contracts // []),
      evidence_refs: (.evidence_refs // [])
    }),
    recommended_next_action:
      (if .disposition == "rejected_by_operator" then
        "No reutilizar esta pista salvo cambio explícito del roadmap; conservarla como decisión negativa con procedencia."
      elif .disposition == "historical_superseded_by_capability" then
        "Abrir primero el owner, acceptance_contracts y evidence_refs V2; usar la fuente legacy solo para comparar contratos y negativos."
      elif (v2_capability(.capability_id).status // "") == "accredited" then
        "Abrir primero la solución acreditada V2; caracterizar la brecha exacta antes de rescatar concepto o contrato legacy."
      else
        "Abrir source_ref y su rango, caracterizar aciertos, fallos y mecanismo, y reimplementar solo el valor útil detrás de la arquitectura V2."
      end),
    depth: "ledger_semantic_lead_not_deep_behavior_assessment",
    historical_green: "not_assessed",
    function_mapping: "not_assessed",
    authority: "derived_advisory_from_task_entry_ledger",
    canonical_state_change: false,
    claims_historical_success: false,
    claims_reuse_decision: false,
    query_total_matches: $total_matches,
    query_offset: (if $unpaginated == 1 then 0 else $offset end),
    query_limit: (if $unpaginated == 1 then $total_matches else $limit end),
    query_truncated: ($unpaginated == 0 and
      ($offset + $page_count) < $total_matches)
  })
' <<<"$page")"

if ((output_json == 1)); then
  jq '.' <<<"$cards"
else
  jq -c '.[]' <<<"$cards"
fi
