#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "$0")" && pwd -P)"
lessons_query="$script_dir/consultar_lecciones_legacy.sh"
reuse_query="$script_dir/consultar_reutilizacion_legacy.sh"
function_query="$script_dir/consultar_funciones_legacy.sh"
ledger_leads_query="$script_dir/consultar_pistas_legacy.sh"
source_leads_query="$script_dir/consultar_fuentes_legacy.sh"

capability=""
target_path=""
operation=""
task_description=""
function_description=""
legacy_function_ref=""

usage() {
  printf '%s\n' "Uso: scripts/preflight_reutilizacion_legacy.sh [campos obligatorios]" "" "  --capability ID    Capability del write-set" "  --path RUTA        Ruta V2 que se va a modificar" "  --operation OP     Operación del cambio" "  --task TEXTO       Tarea o conducta que se va a implementar" "  --function TEXTO   Nombre o firma prevista de la función (pista léxica)" "  --legacy-function-ref REF" "                     occurrence_ref legacy exacta ya conocida" "  --help             Muestra esta ayuda" "" "Indica --task, --function o ambos. La salida siempre es un único JSON compacto."
}

while (($#)); do
  case "$1" in
    --capability)
      capability="$2"
      shift 2
      ;;
    --path)
      target_path="$2"
      shift 2
      ;;
    --operation)
      operation="$2"
      shift 2
      ;;
    --task)
      task_description="$2"
      shift 2
      ;;
    --function)
      function_description="$2"
      shift 2
      ;;
    --legacy-function-ref)
      legacy_function_ref="$2"
      shift 2
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

if [[ -z "$capability" || -z "$target_path" || -z "$operation" || ( -z "$task_description" && -z "$function_description" ) ]]; then
  printf 'Faltan capability, path, operation o la descripción de tarea/función.\n' >&2
  usage >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  printf 'Falta jq; el preflight no usa IA ni red.\n' >&2
  exit 2
fi

lessons='[]'
set +e
matched_lessons="$("$lessons_query" --capability "$capability" --path "$target_path" --operation "$operation" --json 2>&1)"
lessons_status=$?
set -e
if ((lessons_status == 0)); then
  lessons="$matched_lessons"
elif ((lessons_status != 1)); then
  printf '%s\n' "$matched_lessons" >&2
  exit "$lessons_status"
fi

reuse_candidates='[]'
reuse_args=(--capability "$capability" --json)
if [[ -n "$legacy_function_ref" ]]; then
  reuse_args+=(--legacy-function-ref "$legacy_function_ref")
fi
set +e
matched_reuse="$("$reuse_query" "${reuse_args[@]}" 2>&1)"
reuse_status=$?
set -e
if ((reuse_status == 0)); then
  reuse_candidates="$matched_reuse"
elif ((reuse_status != 1)); then
  printf '%s\n' "$matched_reuse" >&2
  exit "$reuse_status"
fi

structural_functions='[]'
if [[ -n "$legacy_function_ref" ]]; then
  set +e
  matched_functions="$("$function_query" --occurrence-ref "$legacy_function_ref" --json 2>&1)"
  function_status=$?
  set -e
  if ((function_status == 0)); then
    structural_functions="$matched_functions"
  elif ((function_status != 1)); then
    printf '%s\n' "$matched_functions" >&2
    exit "$function_status"
  fi
elif [[ -n "$function_description" ]]; then
  set +e
  matched_functions="$("$function_query" --name "$function_description" --json 2>&1)"
  function_status=$?
  set -e
  if ((function_status == 0)); then
    structural_functions="$matched_functions"
  elif ((function_status == 1)); then
    set +e
    matched_functions="$("$function_query" --text "$function_description" --json 2>&1)"
    function_status=$?
    set -e
    if ((function_status == 0)); then
      structural_functions="$matched_functions"
    elif ((function_status != 1)); then
      printf '%s\n' "$matched_functions" >&2
      exit "$function_status"
    fi
  else
    printf '%s\n' "$matched_functions" >&2
    exit "$function_status"
  fi
fi

ledger_leads='[]'
set +e
matched_ledger_leads="$("$ledger_leads_query" --capability "$capability" --limit 50 --json 2>&1)"
ledger_status=$?
set -e
if ((ledger_status == 0)); then
  ledger_leads="$matched_ledger_leads"
elif ((ledger_status != 1)); then
  printf '%s\n' "$matched_ledger_leads" >&2
  exit "$ledger_status"
fi

source_leads='[]'
set +e
matched_source_leads="$("$source_leads_query" --capability "$capability" --json 2>&1)"
source_status=$?
set -e
if ((source_status == 0)); then
  source_leads="$matched_source_leads"
elif ((source_status != 1)); then
  printf '%s\n' "$matched_source_leads" >&2
  exit "$source_status"
fi

jq -nc --arg capability "$capability" --arg target_path "$target_path" --arg operation "$operation" --arg task "$task_description" --arg function "$function_description" --arg legacy_function_ref "$legacy_function_ref" \
  --slurpfile lessons_input <(printf '%s\n' "$lessons") \
  --slurpfile candidates_input <(printf '%s\n' "$reuse_candidates") \
  --slurpfile structural_functions_input <(printf '%s\n' "$structural_functions") \
  --slurpfile ledger_leads_input <(printf '%s\n' "$ledger_leads") \
  --slurpfile source_leads_input <(printf '%s\n' "$source_leads") '
  ($lessons_input[0] // []) as $lessons |
  ($candidates_input[0] // []) as $candidates |
  ($structural_functions_input[0] // []) as $structural_functions |
  ($ledger_leads_input[0] // []) as $ledger_leads |
  ($source_leads_input[0] // []) as $source_leads |
  def tokens($value):
    [$value | ascii_downcase | splits("[^[:alnum:]_áéíóúüñ]+") |
      select(length >= 4)] | unique;
  def haystack($item):
    ([$item.behavior_key, $item.problem, $item.historical_green.rationale,
      $item.mechanism.decision_authority, $item.reuse.recommendation] +
      $item.mechanism.preserve + $item.mechanism.avoid +
      $item.attempts + $item.uncertainties +
      [$item.function_mapping.links[] |
        (.source_path + " " + .name + " " + .receiver + " " + .signature)]) |
    join(" ") | ascii_downcase;
  def relevance($item):
    (tokens($task + " " + $function)) as $tokens |
    if ($tokens | length) == 0 then 0
    else ([$tokens[] as $token | select(haystack($item) | contains($token))] | length)
    end;
  def compact_link($link):
    $link | {
      occurrence_ref,
      source_path,
      symbol_kind,
      name,
      receiver,
      signature,
      relation,
      semantic_rationale,
      evidence_refs
    };
  def implementation_decision($item):
    if $item.reuse.kind == "negative_lesson" then "reject"
    elif $item.v2.status == "accredited" and
      $item.reuse.exact_v2_fit == "exact_or_primary" then "reuse"
    elif $item.reuse.exact_v2_fit == "not_implemented" and
      ($item.reuse.kind == "contract" or $item.reuse.kind == "idea") then
      "reimplement"
    else "characterize"
    end;
  def card($item):
    ($item.function_mapping.links |
      if $legacy_function_ref != "" then . else .[0:2] end) as $selected_links |
    {
      ref: $item.characterization_ref,
      behavior: $item.behavior_key,
      relevance_tokens: relevance($item),
      match_authority:
        (if $legacy_function_ref != "" then "exact_legacy_function_ref"
        elif relevance($item) > 0 then "lexical_hint_only" else "capability_only" end),
      problem: $item.problem,
      v1_result: $item.historical_green.state,
      v1_green_verified: $item.historical_green.independently_verified,
      result_reason: $item.historical_green.rationale,
      snapshot_test_observation_refs:
        [$item.historical_green.test_observations[].observation_ref],
      mechanism: $item.mechanism.decision_authority,
      preserve: $item.mechanism.preserve,
      avoid: $item.mechanism.avoid,
      decision: implementation_decision($item),
      reuse: $item.reuse.kind,
      v2_fit: $item.reuse.exact_v2_fit,
      action: $item.reuse.recommendation,
      v2: {
        status: $item.v2.status,
        acceptance: $item.v2.acceptance_contracts,
        evidence: $item.v2.evidence_refs
      },
      exact_legacy_source_link_count: ($item.function_mapping.links | length),
      exact_legacy_source_links_truncated:
        (($selected_links | length) < ($item.function_mapping.links | length)),
      exact_legacy_source_links:
        [$selected_links[] | compact_link(.)],
      function_mapping_state: $item.function_mapping.state,
      source_refs: $item.sources,
      uncertainties: $item.uncertainties
    };
  def lead_relevance($item):
    (tokens($task + " " + $function)) as $tokens |
    (($item.semantic_reason + " " + $item.source_ref) | ascii_downcase) as $lead_text |
    if ($tokens | length) == 0 then 0
    else ([$tokens[] as $token | select($lead_text | contains($token))] | length)
    end;
  ($candidates | [.[].source_task_entry_refs[]] | unique) as $deep_task_refs |
  (($ledger_leads[0].query_total_matches // 0) - ($deep_task_refs | length)) as $unassessed_lead_count |
  ($ledger_leads |
    map(select(.entry_ref as $ref | ($deep_task_refs | index($ref)) == null)) |
    map(. + {relevance_tokens: lead_relevance(.)}) |
    sort_by([-(.relevance_tokens), .entry_ref])) as $unassessed_leads |
  ($source_leads | map(select(.kind != "task")) |
    sort_by(.kind, .source_ref)) as $non_task_source_leads |
  ($candidates | map(card(.)) | sort_by([-(.relevance_tokens), .ref])) as $all_cards |
  (if $legacy_function_ref != "" then $all_cards else $all_cards[0:3] end) as $cards |
  ($unassessed_leads[0:5] | map({
    entry_ref,
    capability,
    semantic_reason,
    source_ref,
    subject_first_line,
    subject_last_line,
    disposition,
    v2: {title: .v2.title, status: .v2.status,
      acceptance_contracts: .v2.acceptance_contracts,
      evidence_refs: .v2.evidence_refs},
    recommended_next_action,
    depth,
    historical_green,
    function_mapping,
    claims_historical_success,
    claims_reuse_decision,
    relevance_tokens
  })) as $compact_leads |
  ($non_task_source_leads[0:5] | map({
    kind,
    source_ref,
    capability_ids,
    decision,
    reason_code,
    evidence_state,
    lesson_id,
    invariant_test_ref,
    depth,
    recommended_next_action,
    claims_current_bug,
    claims_reuse_decision
  })) as $compact_source_leads |
  ($structural_functions | map({
    occurrence_ref,
    source_path,
    package_name,
    symbol_kind,
    name,
    receiver,
    signature,
    module_context,
    semantic_state,
    semantic_links: [.semantic_links[] | {
      behavior_ref,
      historical_green_state,
      reuse_kind,
      exact_v2_fit,
      relation,
      semantic_rationale,
      recommendation
    }]
  }) | .[0:3]) as $compact_functions |
  {
    schema_version: 1,
    purpose: "agent_preflight_before_design_or_implementation",
    authority: "advisory_no_state_change",
    query: {
      capability: $capability,
      path: $target_path,
      operation: $operation,
      task: $task,
      function: $function,
      legacy_function_ref: $legacy_function_ref
    },
    coverage: {
      state:
        (if ($cards | length) == 0 and $legacy_function_ref != "" then
          "exact_legacy_function_not_linked_to_capability"
        elif ($cards | length) == 0 and ($unassessed_leads | length) > 0 then
          "ledger_semantic_leads_require_deep_assessment"
        elif ($cards | length) == 0 and ($non_task_source_leads | length) > 0 then
          "source_disposition_leads_require_deep_assessment"
        elif ($cards | length) == 0 then "no_semantic_candidate_for_capability"
        elif $legacy_function_ref != "" then "exact_legacy_function_linked"
        elif any($cards[]; (.exact_legacy_source_links | length) > 0) then
          "capability_candidates_with_reviewed_legacy_function_links"
        else "capability_candidates_without_exact_function_mapping" end),
      candidate_count: ($all_cards | length),
      candidates_returned: ($cards | length),
      candidates_truncated: (($all_cards | length) > ($cards | length)),
      ledger_semantic_lead_count:
        (if $unassessed_lead_count < 0 then 0 else $unassessed_lead_count end),
      ledger_semantic_leads_returned: ($compact_leads | length),
      source_disposition_lead_count: ($non_task_source_leads | length),
      source_disposition_leads_returned: ($compact_source_leads | length),
      structural_function_candidate_count:
        (if ($structural_functions | length) == 0 then 0
        else $structural_functions[0].query_total_matches end),
      structural_functions_returned: ($compact_functions | length),
      structural_functions_truncated:
        (if ($structural_functions | length) == 0 then false
        else $structural_functions[0].query_total_matches > ($compact_functions | length) end),
      exact_function_match: ($legacy_function_ref != "" and ($cards | length) > 0),
      warning:
        "La coincidencia textual solo ordena candidatos. No demuestra equivalencia, verde histórico ni permiso para copiar código."
    },
    cross_cutting_lessons: $lessons,
    decision_summary:
      (if ($cards | length) == 0 then
        {
          primary: "characterize",
          candidates: {characterize: 1},
          reason: "No hay una ficha profunda aplicable; caracterizar el hueco antes de diseñar."
        }
      else
        {
          primary: $cards[0].decision,
          candidates:
            ($cards | group_by(.decision) |
              map({key: .[0].decision, value: length}) | from_entries),
          reason:
            "La decisión es advisory y se deriva del estado V2, el encaje y el resultado histórico; no autoriza copiar código."
        }
      end),
    reuse_candidates: $cards,
    ledger_semantic_leads: $compact_leads,
    source_disposition_leads: $compact_source_leads,
    structural_function_candidates: $compact_functions,
    agent_protocol: [
      "Si V2 está acreditada y el encaje es exact_or_primary, abrir primero acceptance/evidence y reutilizar el owner V2.",
      "Si el encaje es partial_overlap, comparar contrato y negativos antes de diseñar la brecha.",
      "Si V2 no lo implementa, reutilizar solo contrato, idea o negativo detrás de la arquitectura V2.",
      "No copiar código V1 hasta enlazar la función exacta y validar autoría, licencia, dependencias, seguridad y tests.",
      "Una función structural_only está censada pero no valorada: caracterizar problema, aciertos y fallos antes de reutilizarla.",
      "Una pista ledger_semantic_lead ya tiene capability, razón y procedencia revisadas, pero no aciertos, fallos ni función: úsala para abrir la fuente y profundizar, no para afirmar reutilización.",
      "Una source_disposition_lead localiza documentos de bugs o skills; no afirma que el bug siga vivo ni que la herramienta deba adoptarse.",
      "Si no hay candidato exacto, registrar el hueco; no inferir equivalencia por nombre o keywords."
    ],
    canonical_state_change: false,
    creates_work_item: false,
    closes_capability: false,
    claims_accreditation: false
  }
'
