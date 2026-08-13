#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
module_index="${repository_root}/product/knowledge/legacy_go_function_snapshot_v1.jsonl"
module_manifest="${repository_root}/product/knowledge/legacy_go_function_snapshot_v1.manifest.json"
cmd_index="${repository_root}/product/knowledge/legacy_cmd_go_function_snapshot_v1.jsonl"
cmd_manifest="${repository_root}/product/knowledge/legacy_cmd_go_function_snapshot_v1.manifest.json"
assessments="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.jsonl"
assessments_manifest="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.manifest.json"
observations="${repository_root}/product/knowledge/legacy_test_observations_v1.jsonl"
guard="${script_dir}/lib/legacy_read_model_guard.sh"
fixtures_dir="${repository_root}/product/traceability/fixtures"

name=""
text_filter=""
path_filter=""
package_filter=""
occurrence_ref=""
symbol_kind=""
limit=20
summary=0
output_json=0

usage() {
  printf '%s\n' \
    "Uso: /ruta/a/orquestaV2/scripts/consultar_catalogo_funciones_legacy_v1.sh [filtro]" \
    "" \
    "Catálogo general de V1. Puede invocarse desde cualquier directorio o app." \
    "No recibe capabilities, rutas V2 ni decisiones de una aplicación consumidora." \
    "" \
    "  --name NOMBRE          Nombre exacto de función o método" \
    "  --text TEXTO           Subcadena en nombre, firma, receptor, paquete o ruta" \
    "  --path PREFIJO         Prefijo de source_path V1" \
    "  --package PAQUETE      package_name exacto" \
    "  --occurrence-ref REF   Aparición AST exacta" \
    "  --kind func|method     Clase de símbolo" \
    "  --limit N              Máximo de resultados (1–200; defecto 20)" \
    "  --summary              Resume el catálogo general" \
    "  --json                 Array JSON; por defecto JSONL compacto" \
    "  --help                 Muestra esta ayuda"
}

while (($#)); do
  case "$1" in
    --name) name="${2:?falta valor para --name}"; shift 2 ;;
    --text) text_filter="${2:?falta valor para --text}"; shift 2 ;;
    --path) path_filter="${2:?falta valor para --path}"; shift 2 ;;
    --package) package_filter="${2:?falta valor para --package}"; shift 2 ;;
    --occurrence-ref) occurrence_ref="${2:?falta valor para --occurrence-ref}"; shift 2 ;;
    --kind) symbol_kind="${2:?falta valor para --kind}"; shift 2 ;;
    --limit) limit="${2:?falta valor para --limit}"; shift 2 ;;
    --summary) summary=1; shift ;;
    --json) output_json=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *)
      printf 'Argumento desconocido: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! "$limit" =~ ^[0-9]+$ ]] || ((limit < 1 || limit > 200)); then
  printf 'El límite debe estar entre 1 y 200.\n' >&2
  exit 2
fi
if [[ -n "$symbol_kind" && "$symbol_kind" != "func" && "$symbol_kind" != "method" ]]; then
  printf 'La clase debe ser func o method.\n' >&2
  exit 2
fi
if ((summary == 0)) && [[ -z "$name" && -z "$text_filter" && -z "$path_filter" &&
  -z "$package_filter" && -z "$occurrence_ref" && -z "$symbol_kind" ]]; then
  printf 'Indica al menos un filtro o --summary.\n' >&2
  usage >&2
  exit 2
fi
for command in jq sha256sum stat; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'Falta %s; el catálogo no usa IA, red ni runtime V1.\n' "$command" >&2
    exit 2
  fi
done
for required in "$module_index" "$module_manifest" "$cmd_index" "$cmd_manifest" \
  "$assessments" "$assessments_manifest" "$observations" "$guard"; do
  if [[ ! -r "$required" || ! -f "$required" ]]; then
    printf 'Catálogo o manifiesto no legible: %s\n' "$required" >&2
    exit 2
  fi
done

# shellcheck source=scripts/lib/legacy_read_model_guard.sh
source "$guard"
legacy_verify_file_manifest \
  "$module_index" "$module_manifest" "legacy_go_function_snapshot_index"
legacy_verify_file_manifest \
  "$cmd_index" "$cmd_manifest" "legacy_cmd_go_function_snapshot_index"
legacy_validate_reuse_assessments "$assessments" "$assessments_manifest"
if ! jq -e '.contains_bodies == false and .claims_accreditation == false' \
  "$module_manifest" >/dev/null ||
  ! jq -e '.contains_bodies == false and .claims_accreditation == false' \
  "$cmd_manifest" >/dev/null; then
  printf 'El catálogo general rechaza snapshots con cuerpos o autoridad de acreditación.\n' >&2
  exit 2
fi

shopt -s nullglob
behavior_files=("${fixtures_dir}"/behavior_characterization_agent_batch_*.jsonl)
shopt -u nullglob
if ((${#behavior_files[@]} == 0)); then
  printf 'No hay caracterizaciones de conducta disponibles.\n' >&2
  exit 2
fi

if ((summary == 1)); then
  jq -n \
    --slurpfile modules "$module_manifest" \
    --slurpfile cmd "$cmd_manifest" \
    --slurpfile reuse "$assessments_manifest" '
      {
        schema_version: 1,
        document_kind: "legacy_v1_general_function_catalog_summary",
        audience: "any_application",
        source_scopes: [
          {source_root: $modules[0].source_root,
            declarations: $modules[0].declaration_count},
          {source_root: $cmd[0].source_root,
            declarations: $cmd[0].declaration_count}
        ],
        function_count: ($modules[0].function_count + $cmd[0].function_count),
        method_count: ($modules[0].method_count + $cmd[0].method_count),
        declaration_count:
          ($modules[0].declaration_count + $cmd[0].declaration_count),
        unique_occurrence_count:
          ($modules[0].unique_occurrence_count + $cmd[0].unique_occurrence_count),
        behavior_assessment_count: $reuse[0].assessment_count,
        function_behavior_link_count: $reuse[0].function_behavior_link_count,
        unique_semantically_linked_functions: $reuse[0].unique_function_occurrence_count,
        contains_bodies: false,
        license_state: "not_declared_per_function_requires_source_review",
        code_reuse_authorized: false,
        consumer_application_decisions_included: false,
        authority: "portable_advisory_catalog_not_runtime_or_product_state",
        claims_accreditation: false
      }
    '
  exit 0
fi

result="$(jq -s \
  --slurpfile assessments "$assessments" \
  --slurpfile observations "$observations" \
  --slurpfile behaviors <(jq -s '.' "${behavior_files[@]}") \
  --arg name "$name" \
  --arg text_filter "$text_filter" \
  --arg path_filter "$path_filter" \
  --arg package_filter "$package_filter" \
  --arg occurrence_ref "$occurrence_ref" \
  --arg symbol_kind "$symbol_kind" \
  --argjson limit "$limit" '
    def behavior_by_ref($ref):
      first($behaviors[0][] | select(.characterization_ref == $ref));
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
    def general_assessments($ref):
      [$assessments[] as $assessment |
        $assessment.function_mapping.links[] |
        select(.occurrence_ref == $ref) |
        . as $link |
        (behavior_by_ref($assessment.characterization_ref)) as $behavior |
        {
          behavior_ref: $assessment.characterization_ref,
          behavior_key: $behavior.behavior_key,
          problem: $behavior.problem,
          historical_result: $assessment.historical_green_state,
          historical_result_independently_verified: false,
          result_reason: $assessment.green_rationale,
          mechanism: $behavior.decision_authority,
          mechanism_scope:
            "characterized_historical_problem_not_consumer_architecture_decision",
          worked_claims: ($behavior.worked // []),
          failures_and_limitations: ($behavior.did_not_work // []),
          preserve: ($behavior.preserve // []),
          avoid: ($behavior.avoid // []),
          relation: $link.relation,
          semantic_rationale: $link.semantic_rationale,
          source_evidence: ($behavior.evidence // []),
          snapshot_test_observations:
            observations_by_ref($assessment.characterization_ref),
          authority: "portable_advisory_characterization"
        }
      ];
    (map(select(
      ($name == "" or .name == $name) and
      ($path_filter == "" or (.source_path | startswith($path_filter))) and
      ($package_filter == "" or .package_name == $package_filter) and
      ($occurrence_ref == "" or .occurrence_ref == $occurrence_ref) and
      ($symbol_kind == "" or .symbol_kind == $symbol_kind) and
      ($text_filter == "" or
        (([.name, .receiver, .signature, .package_name, .source_path] |
          join(" ") | ascii_downcase) |
          contains($text_filter | ascii_downcase)))
    ))) as $matched |
    ($matched | sort_by(.name, .receiver, .source_path, .start_line)) as $ranked |
    ($ranked | length) as $total |
    $ranked[0:$limit] |
    map(
      . as $function |
      (general_assessments($function.occurrence_ref)) as $semantic |
      {
        schema_version: 1,
        catalog_kind: "legacy_v1_general_function_catalog_entry",
        function: ($function | {
          occurrence_ref,
          declaration_ref,
          variant_ref,
          source_root,
          source_path,
          git_blob_oid,
          blob_digest,
          package_path,
          package_name,
          symbol_kind,
          name,
          receiver,
          signature,
          source_sha256,
          ast_sha256,
          start_line,
          end_line
        }),
        semantic_state:
          (if ($semantic | length) > 0 then
            "linked_to_reviewed_behavior_characterization"
          else "structural_only_not_semantically_assessed" end),
        behavior_assessments: $semantic,
        license: {
          state: "not_declared_per_function_requires_source_review",
          declared_license: null,
          author: null,
          permits_code_reuse: false
        },
        reuse_conditions: [
          "verify exact source provenance and ownership",
          "establish a compatible license before copying code",
          "compare behavior and negative cases with the target application",
          "review dependencies, security and target architecture fit",
          "exercise target-specific tests before adoption"
        ],
        contains_body: false,
        code_reuse_authorized: false,
        consumer_application_decision: null,
        authority: "portable_advisory_catalog_not_runtime_or_product_state",
        query_total_matches: $total,
        query_truncated: ($total > $limit),
        claims_accreditation: false
      }
    )
  ' "$module_index" "$cmd_index")"

if [[ "$(jq 'length' <<<"$result")" == "0" ]]; then
  printf 'Sin funciones legacy para los filtros dados.\n' >&2
  exit 1
fi
if ((output_json == 1)); then
  jq '.' <<<"$result"
else
  jq -c '.[]' <<<"$result"
fi
