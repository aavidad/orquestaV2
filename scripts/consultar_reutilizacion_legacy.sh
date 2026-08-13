#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
roadmap_path="${repository_root}/product/roadmap.json"
assessments_path="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.jsonl"
assessments_manifest_path="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.manifest.json"
observations_path="${repository_root}/product/knowledge/legacy_test_observations_v1.jsonl"
task_entries_path="${repository_root}/product/traceability/task_entries.jsonl"
markdown_sources_path="${repository_root}/product/traceability/markdown_source_roles.jsonl"
source_dispositions_path="${repository_root}/product/traceability/source_dispositions.jsonl"
function_manifest_path="${repository_root}/product/knowledge/legacy_go_function_snapshot_v1.manifest.json"
cmd_function_manifest_path="${repository_root}/product/knowledge/legacy_cmd_go_function_snapshot_v1.manifest.json"
guard_path="${script_dir}/lib/legacy_read_model_guard.sh"
fixtures_dir="${repository_root}/product/traceability/fixtures"

capability=""
behavior=""
text_filter=""
green_state=""
v2_status=""
reuse_kind=""
legacy_function_ref=""
show_all=0
output_json=0
output_jsonl=0
output_detail=0
summary=0

usage() {
  printf '%s\n' \
    "Uso: scripts/consultar_reutilizacion_legacy.sh [filtros]" \
    "" \
    "  --capability ID    Capability exacta, por ejemplo ORC-13" \
    "  --behavior REF     characterization_ref o behavior_key exactos" \
    "  --text TEXTO       Busca texto literal sin distinguir mayúsculas" \
    "  --green ESTADO     documented_green_unverified, partial_or_mixed," \
    "                     failed_or_negative o unknown" \
    "  --v2-status ESTADO declared, implemented, wired, exercised o accredited" \
    "  --reuse-kind TIPO  v2_existing, contract, idea o negative_lesson" \
    "  --legacy-function-ref REF" \
    "                     occurrence_ref exacta de una función legacy enlazada" \
    "  --all              Incluye todas las caracterizaciones disponibles" \
    "  --summary          Devuelve métricas de la selección" \
    "  --json             Devuelve el detalle completo como array JSON" \
    "  --jsonl            Devuelve el detalle completo como JSONL" \
    "  --detail           Devuelve texto extenso; por defecto emite fichas JSONL compactas" \
    "  --help             Muestra esta ayuda"
}

while (($#)); do
  case "$1" in
    --capability)
      capability="${2:?falta valor para --capability}"
      shift 2
      ;;
    --behavior)
      behavior="${2:?falta valor para --behavior}"
      shift 2
      ;;
    --text)
      text_filter="${2:?falta valor para --text}"
      shift 2
      ;;
    --green)
      green_state="${2:?falta valor para --green}"
      shift 2
      ;;
    --v2-status)
      v2_status="${2:?falta valor para --v2-status}"
      shift 2
      ;;
    --reuse-kind)
      reuse_kind="${2:?falta valor para --reuse-kind}"
      shift 2
      ;;
    --legacy-function-ref)
      legacy_function_ref="${2:?falta valor para --legacy-function-ref}"
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
    --detail)
      output_detail=1
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

if ((show_all == 0 && summary == 0)) &&
  [[ -z "$capability" && -z "$behavior" && -z "$text_filter" &&
    -z "$green_state" && -z "$v2_status" && -z "$reuse_kind" &&
    -z "$legacy_function_ref" ]]; then
  printf 'Indica al menos un filtro, --all o --summary.\n' >&2
  usage >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  printf 'Falta jq; la consulta no usa IA ni red.\n' >&2
  exit 2
fi
for required_path in "$roadmap_path" "$assessments_path" "$assessments_manifest_path" "$observations_path" \
  "$task_entries_path" "$markdown_sources_path" "$source_dispositions_path" \
  "$function_manifest_path" "$cmd_function_manifest_path"; do
  if [[ ! -r "$required_path" ]]; then
    printf 'Autoridad o read-model no legible: %s\n' "$required_path" >&2
    exit 2
  fi
done
# shellcheck source=scripts/lib/legacy_read_model_guard.sh
source "$guard_path"
legacy_validate_reuse_assessments "$assessments_path" "$assessments_manifest_path"

shopt -s nullglob
behavior_files=("${fixtures_dir}"/behavior_characterization_agent_batch_*.jsonl)
shopt -u nullglob
if ((${#behavior_files[@]} == 0)); then
  printf 'No hay fixtures de caracterización disponibles.\n' >&2
  exit 2
fi

projection="$({
  jq -s \
    --slurpfile roadmap "$roadmap_path" \
    --slurpfile assessments "$assessments_path" \
    --slurpfile observations "$observations_path" \
    --arg capability "$capability" \
    --arg behavior "$behavior" \
    --arg text_filter "$text_filter" \
    --arg green_state "$green_state" \
    --arg v2_status "$v2_status" \
    --arg reuse_kind "$reuse_kind" \
    --arg legacy_function_ref "$legacy_function_ref" \
    '
      def capability_by_id($id):
        first($roadmap[0].capability_entries[] | select(.id == $id));
      def assessment_by_ref($ref):
        first($assessments[] | select(.characterization_ref == $ref));
      def observations_by_ref($ref):
        [$observations[] |
          select((.behavior_refs | index($ref)) != null) |
          {
            observation_ref,
            repository_commit,
            module_path,
            module_tree_oid,
            command,
            result,
            verification_kind,
            test_refs,
            limitations
          }
        ];
      map(
        . as $behavior_item |
        (assessment_by_ref($behavior_item.characterization_ref)) as $assessment |
        (capability_by_id($behavior_item.capability_id)) as $v2 |
        {
          schema_version: 1,
          id: ("LEGACY-REUSE::" + $behavior_item.characterization_ref),
          behavior_ref: $behavior_item.characterization_ref,
          characterization_ref: $behavior_item.characterization_ref,
          behavior_key: $behavior_item.behavior_key,
          title: ($behavior_item.behavior_key | gsub("_"; " ")),
          summary: $behavior_item.problem,
          family: $behavior_item.capability_id,
          capability_id: $behavior_item.capability_id,
          problem: $behavior_item.problem,
          historical_green: {
            state: $assessment.historical_green_state,
            independently_verified: false,
            rationale: $assessment.green_rationale,
            claims: ($behavior_item.worked // []),
            limitations: ($behavior_item.did_not_work // []),
            test_observations: observations_by_ref($behavior_item.characterization_ref)
          },
          mechanism: {
            decision_authority: $behavior_item.decision_authority,
            inputs: ($behavior_item.inputs // []),
            outputs: ($behavior_item.outputs // []),
            state_read: ($behavior_item.state_read // []),
            state_written: ($behavior_item.state_written // []),
            failure_retry_concurrency_restart: $behavior_item.failure_retry_concurrency_restart,
            preserve: ($behavior_item.preserve // []),
            avoid: ($behavior_item.avoid // [])
          },
          reuse: {
            kind: $assessment.reuse_kind,
            exact_v2_fit: $assessment.exact_v2_fit,
            code_reuse_state: "not_assessed_requires_function_provenance_license_and_architecture_fit",
            recommendation: $assessment.agent_recommendation
          },
          function_mapping: $assessment.function_mapping,
          v2: {
            title: $v2.title,
            decision: $v2.decision,
            status: $v2.status,
            acceptance_contracts: ($v2.acceptance_contracts // []),
            evidence_refs: ($v2.evidence_refs // [])
          },
          sources: ([$behavior_item.evidence[].source_ref] | unique),
          source_evidence: $behavior_item.evidence,
          source_task_entry_refs: $behavior_item.task_entry_refs,
          attempts: ($behavior_item.attempts // []),
          uncertainties: ($behavior_item.uncertainties // []),
          characterization_review_state: $behavior_item.review_state,
          reuse_assessment_review_state: $assessment.review_state,
          authority: "derived_advisory_read_model",
          canonical_state_change: false,
          closes_capability: false,
          claims_accreditation: false
        }
      ) |
      map(select(
        ($capability == "" or .capability_id == $capability) and
        ($behavior == "" or .characterization_ref == $behavior or .behavior_key == $behavior) and
        ($green_state == "" or .historical_green.state == $green_state) and
        ($v2_status == "" or .v2.status == $v2_status) and
        ($reuse_kind == "" or .reuse.kind == $reuse_kind) and
        ($legacy_function_ref == "" or
          any(.function_mapping.links[]; .occurrence_ref == $legacy_function_ref)) and
        ($text_filter == "" or ((tojson | ascii_downcase) | contains($text_filter | ascii_downcase)))
      )) |
      sort_by(.capability_id, .characterization_ref)
    ' "${behavior_files[@]}"
})"

count="$(jq 'length' <<<"$projection")"
if ((count == 0)); then
  printf 'Sin reutilizaciones para los filtros dados.\n' >&2
  exit 1
fi

if ((summary == 1)); then
  task_total="$(jq -s 'map(.entry_ref) | unique | length' "$task_entries_path")"
  jq --slurpfile roadmap "$roadmap_path" \
    --slurpfile task_entries "$task_entries_path" \
    --slurpfile markdown_sources "$markdown_sources_path" \
    --slurpfile source_dispositions "$source_dispositions_path" \
    --slurpfile function_manifest "$function_manifest_path" \
    --slurpfile cmd_function_manifest "$cmd_function_manifest_path" \
    --argjson task_total "$task_total" \
    '
      (map(.source_task_entry_refs[]) | unique | length) as $characterized |
      (map(.source_task_entry_refs[]) | unique) as $characterized_refs |
      (map(.capability_id) | unique) as $deep_capability_ids |
      ($deep_capability_ids | length) as $deep_capabilities |
      ($task_entries | map(.capability_id) | unique | length) as $total_capabilities |
      ($roadmap[0].capability_entries | length) as $roadmap_catalog_size |
      ($task_entries |
        map(.entry_ref as $ref |
          select(($characterized_refs | index($ref)) == null))) as $remaining |
      ($task_entries |
        map(.capability_id as $capability |
          select(($deep_capability_ids | index($capability)) == null))) as
        $capabilities_without_deep_assessment |
      {
        schema_version: 1,
        authority: "derived_advisory_read_model",
        inventory_layers: {
          markdown_sources: {
            classified: ($markdown_sources | length),
            total: ($markdown_sources | length),
            coverage_percent:
              (if ($markdown_sources | length) == 0 then 0 else
                (((($markdown_sources | length) * 10000) /
                  ($markdown_sources | length) | floor) / 100)
              end),
            actionable: ($source_dispositions | length),
            context_only: (($markdown_sources | length) - ($source_dispositions | length))
          },
          task_semantic_leads: {
            reviewed: ($task_entries | map(select(.semantic_review_state == "reviewed" and
              (.semantic_reason | length) > 0)) | length),
            total: $task_total,
            coverage_percent:
              (if $task_total == 0 then 0 else
                ((($task_entries | map(select(.semantic_review_state == "reviewed" and
                  (.semantic_reason | length) > 0)) | length) * 10000 /
                  $task_total | floor) / 100)
              end)
          },
          go_function_snapshot: {
            indexed: ($function_manifest[0].declaration_count +
              $cmd_function_manifest[0].declaration_count),
            total: ($function_manifest[0].declaration_count +
              $cmd_function_manifest[0].declaration_count),
            coverage_percent:
              (if ($function_manifest[0].declaration_count +
                $cmd_function_manifest[0].declaration_count) == 0 then 0 else 100
              end),
            functions: ($function_manifest[0].function_count +
              $cmd_function_manifest[0].function_count),
            methods: ($function_manifest[0].method_count +
              $cmd_function_manifest[0].method_count),
            scopes: [
              {source_root:$function_manifest[0].source_root,
                declarations:$function_manifest[0].declaration_count},
              {source_root:$cmd_function_manifest[0].source_root,
                declarations:$cmd_function_manifest[0].declaration_count}
            ]
          },
          deep_behavior_assessment: {
            task_entry_refs: $characterized,
            total_task_entry_refs: $task_total,
            coverage_percent: (($characterized * 10000 / $task_total | floor) / 100),
            behaviors: length,
            capabilities: $deep_capabilities,
            total_capabilities: $total_capabilities,
            capability_coverage_percent:
              (($deep_capabilities * 10000 / $total_capabilities | floor) / 100)
          }
        },
        behaviors: length,
        capabilities: (map(.capability_id) | unique | length),
        coverage_scope: "legacy_task_ledger_capabilities_not_product_roadmap",
        capability_ledger_total: $total_capabilities,
        roadmap_catalog_size: $roadmap_catalog_size,
        capability_deep_assessment_coverage_percent:
          (($deep_capabilities * 10000 / $total_capabilities | floor) / 100),
        task_entry_refs_characterized: $characterized,
        task_entry_ledger_total: $task_total,
        task_entry_ledger_semantic_leads_reviewed:
          ($task_entries | map(select(.semantic_review_state == "reviewed" and
            (.semantic_reason | length) > 0)) | length),
        ledger_semantic_lead_coverage_percent:
          ((($task_entries | map(select(.semantic_review_state == "reviewed" and
            (.semantic_reason | length) > 0)) | length) * 10000 /
            $task_total | floor) / 100),
        task_entry_refs_uncharacterized: ($remaining | length),
        task_entry_reference_coverage_percent:
          (($characterized * 10000 / $task_total | floor) / 100),
        deep_behavior_assessment_coverage_percent:
          (($characterized * 10000 / $task_total | floor) / 100),
        historical_green_states:
          (group_by(.historical_green.state) |
            map({key: .[0].historical_green.state, value: length}) |
            from_entries),
        reuse_kinds:
          (group_by(.reuse.kind) |
            map({key: .[0].reuse.kind, value: length}) |
            from_entries),
        behavior_v2_statuses:
          (group_by(.v2.status) |
            map({key: .[0].v2.status, value: length}) |
            from_entries),
        v2_capability_statuses:
          (group_by(.capability_id) |
            map(.[0]) |
            group_by(.v2.status) |
            map({key: .[0].v2.status, value: length}) |
            from_entries),
        reuse_assessments_counterreviewed:
          (map(select(.reuse_assessment_review_state ==
            "independent_bootstrap_counterreview_completed_advisory")) | length),
        source_characterizations_still_marked_pending:
          (map(select(.characterization_review_state ==
            "bootstrap_first_review_pending_independent_counterreview")) | length),
        verified_historical_greens:
          (map(select(.historical_green.independently_verified == true)) | length),
        behaviors_with_passing_snapshot_tests:
          (map(select(any(.historical_green.test_observations[]; .result == "passed"))) | length),
        function_behavior_links:
          ([.[].function_mapping.links[]] | length),
        behaviors_with_exact_function_links:
          (map(select(.function_mapping.state == "linked_exact")) | length),
        unique_exact_function_occurrences:
          ([.[].function_mapping.links[].occurrence_ref] | unique | length),
        next_uncharacterized_capabilities:
          ($capabilities_without_deep_assessment |
            group_by(.capability_id) |
            map({capability_id: .[0].capability_id, task_entry_refs: length}) |
            sort_by(-.task_entry_refs, .capability_id) | .[0:10]),
        note: "Las 2.108 pistas tienen capability, razón y procedencia revisadas; 184/184 mide solo familias presentes en el ledger legacy, no las 257 capacidades del producto. La cobertura profunda añade problema, aciertos, fallos, mecanismo, valor V2 y funciones exactas."
      }
    ' <<<"$projection"
  exit 0
fi

if ((output_json == 1)); then
  jq '.' <<<"$projection"
  exit 0
fi
if ((output_jsonl == 1)); then
  jq -c '.[]' <<<"$projection"
  exit 0
fi

if ((output_detail == 0)); then
  jq -c '
    .[] | {
      ref: .characterization_ref,
      capability: .capability_id,
      v2_status: .v2.status,
      problem,
      historical_result: .historical_green.state,
      historical_green_verified: .historical_green.independently_verified,
      result_reason: .historical_green.rationale,
      historical_test_observations: .historical_green.test_observations,
      mechanism: .mechanism.decision_authority,
      preserve: .mechanism.preserve,
      avoid: .mechanism.avoid,
      reuse: .reuse.kind,
      v2_fit: .reuse.exact_v2_fit,
      action: .reuse.recommendation,
      v2_acceptance: .v2.acceptance_contracts,
      v2_evidence: .v2.evidence_refs,
      function_mapping,
      source_refs: .sources,
      uncertainties
    }
  ' <<<"$projection"
  exit 0
fi

jq -r '
  .[] |
  "[\(.characterization_ref)] \(.title)\n" +
  "capability V2: \(.capability_id) (\(.v2.status))\n" +
  "problema: \(.problem)\n" +
  "verde histórico: \(.historical_green.state); verificado independientemente: no\n" +
  "dictamen: \(.historical_green.rationale)\n" +
  "afirmaciones o intentos favorables declarados, no verificados: \(.historical_green.claims | join("; "))\n" +
  "límites observados: \(.historical_green.limitations | join("; "))\n" +
  "mecanismo/autoridad: \(.mechanism.decision_authority)\n" +
  "conservar: \(.mechanism.preserve | join("; "))\n" +
  "evitar: \(.mechanism.avoid | join("; "))\n" +
  "reutilización: \(.reuse.kind); encaje V2: \(.reuse.exact_v2_fit)\n" +
  "acción: \(.reuse.recommendation)\n" +
  "código legacy: \(.reuse.code_reuse_state)\n" +
  "evidencia V2: \(if (.v2.evidence_refs | length) == 0 then "ninguna todavía" else (.v2.evidence_refs | join(", ")) end)\n" +
  "fuentes V1: \(.sources | join(", "))\n"
' <<<"$projection"
