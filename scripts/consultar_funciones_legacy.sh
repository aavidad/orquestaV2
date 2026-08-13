#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_dir}/.." && pwd -P)"
index_path="${repository_root}/product/knowledge/legacy_go_function_snapshot_v1.jsonl"
manifest_path="${repository_root}/product/knowledge/legacy_go_function_snapshot_v1.manifest.json"
cmd_index_path="${repository_root}/product/knowledge/legacy_cmd_go_function_snapshot_v1.jsonl"
cmd_manifest_path="${repository_root}/product/knowledge/legacy_cmd_go_function_snapshot_v1.manifest.json"
assessments_path="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.jsonl"
assessments_manifest_path="${repository_root}/product/knowledge/legacy_reuse_assessments_v1.manifest.json"
task_entries_path="${repository_root}/product/traceability/task_entries.jsonl"
legacy_go_path="${repository_root}/product/traceability/legacy_go.json"
guard_path="${script_dir}/lib/legacy_read_model_guard.sh"

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
    "Uso: scripts/consultar_funciones_legacy.sh [filtro]" \
    "" \
    "  --name NOMBRE          Nombre exacto de función o método" \
    "  --text TEXTO           Subcadena en nombre, firma, receptor, paquete o ruta" \
    "  --path PREFIJO         Prefijo de source_path" \
    "  --package PAQUETE      package_name exacto" \
    "  --occurrence-ref REF   Aparición AST exacta" \
    "  --kind func|method     Clase de símbolo" \
    "  --limit N              Máximo de resultados (1–200; defecto 20)" \
    "  --summary              Cobertura estructural y enlaces semánticos" \
    "  --json                 Array JSON; por defecto JSONL compacto" \
    "  --help                 Muestra esta ayuda"
}

while (($#)); do
  case "$1" in
    --name)
      name="${2:?falta valor para --name}"
      shift 2
      ;;
    --text)
      text_filter="${2:?falta valor para --text}"
      shift 2
      ;;
    --path)
      path_filter="${2:?falta valor para --path}"
      shift 2
      ;;
    --package)
      package_filter="${2:?falta valor para --package}"
      shift 2
      ;;
    --occurrence-ref)
      occurrence_ref="${2:?falta valor para --occurrence-ref}"
      shift 2
      ;;
    --kind)
      symbol_kind="${2:?falta valor para --kind}"
      shift 2
      ;;
    --limit)
      limit="${2:?falta valor para --limit}"
      shift 2
      ;;
    --summary)
      summary=1
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
for required in "$index_path" "$manifest_path" "$cmd_index_path" "$cmd_manifest_path" "$assessments_path" \
  "$assessments_manifest_path" "$task_entries_path" "$legacy_go_path"; do
  if [[ ! -r "$required" ]]; then
    printf 'Read-model no legible: %s\n' "$required" >&2
    exit 2
  fi
done
for command in jq sha256sum stat; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'Falta %s; la consulta no usa IA ni red.\n' "$command" >&2
    exit 2
  fi
done

# shellcheck source=scripts/lib/legacy_read_model_guard.sh
source "$guard_path"
legacy_validate_reuse_assessments "$assessments_path" "$assessments_manifest_path"

for pair in "$index_path|$manifest_path" "$cmd_index_path|$cmd_manifest_path"; do
  data_path="${pair%%|*}"
  data_manifest="${pair#*|}"
  expected_sha="$(jq -er '.jsonl_sha256 | sub("^sha256:"; "")' "$data_manifest")"
  expected_bytes="$(jq -er '.jsonl_bytes' "$data_manifest")"
  actual_sha="$(sha256sum "$data_path" | cut -d ' ' -f 1)"
  actual_bytes="$(stat -c '%s' "$data_path")"
  if [[ "$actual_sha" != "$expected_sha" || "$actual_bytes" != "$expected_bytes" ]]; then
    printf 'El índice estructural no coincide con su manifiesto: %s\n' "$data_path" >&2
    exit 2
  fi
done

if ((summary == 1)); then
  jq --slurpfile assessments "$assessments_path" --slurpfile task_entries "$task_entries_path" \
    --slurpfile cmd_manifest "$cmd_manifest_path" '
    ([ $assessments[].function_mapping.links[].occurrence_ref ] | unique) as $linked |
    ($task_entries | map(select(.semantic_review_state == "reviewed" and
      (.semantic_reason | length) > 0)) | length) as $reviewed_leads |
    . as $modules |
    ($cmd_manifest[0]) as $cmd |
    {
      document_kind: "legacy_go_function_snapshot_index_union",
      schema_version: 1,
      scopes: [$modules, $cmd],
      production_file_count: ($modules.production_file_count + $cmd.production_file_count),
      function_count: ($modules.function_count + $cmd.function_count),
      method_count: ($modules.method_count + $cmd.method_count),
      declaration_count: ($modules.declaration_count + $cmd.declaration_count),
      unique_occurrence_count: ($modules.unique_occurrence_count + $cmd.unique_occurrence_count),
      contains_bodies: false,
      semantic_behavior_count: ($assessments | length),
      semantic_function_behavior_links:
        ([$assessments[].function_mapping.links[]] | length),
      structurally_indexed_occurrences_with_semantic_links: ($linked | length),
      structurally_indexed_occurrences_without_semantic_links:
        (($modules.declaration_count + $cmd.declaration_count) - ($linked | length)),
      ledger_semantic_leads_reviewed: $reviewed_leads,
      ledger_semantic_lead_total: ($task_entries | length),
      ledger_semantic_lead_coverage_percent:
        (($reviewed_leads * 10000 / ($task_entries | length) | floor) / 100),
      git_snapshot_ast_census_coverage_percent:
        ((($modules.unique_occurrence_count + $cmd.unique_occurrence_count) * 10000 /
          ($modules.declaration_count + $cmd.declaration_count) | floor) / 100),
      authority: "derived_advisory_read_model",
      claims_accreditation: false,
      note: "Estructural 100 % de dos snapshots separados: modulos y cmd legacy fuera de cmd/orquesta; la valoración semántica se mide aparte."
    }
  ' "$manifest_path"
  exit 0
fi

result="$(jq -s \
  --slurpfile assessments "$assessments_path" \
  --slurpfile task_entries "$task_entries_path" \
  --slurpfile legacy_go "$legacy_go_path" \
  --arg name "$name" \
  --arg text_filter "$text_filter" \
  --arg path_filter "$path_filter" \
  --arg package_filter "$package_filter" \
  --arg occurrence_ref "$occurrence_ref" \
  --arg symbol_kind "$symbol_kind" \
  --argjson limit "$limit" '
    def semantic_links($ref):
      [$assessments[] as $assessment |
        $assessment.function_mapping.links[] |
        select(.occurrence_ref == $ref) |
        {
          behavior_ref: $assessment.characterization_ref,
          historical_green_state: $assessment.historical_green_state,
          reuse_kind: $assessment.reuse_kind,
          exact_v2_fit: $assessment.exact_v2_fit,
          relation,
          semantic_rationale,
          evidence_refs,
          recommendation: $assessment.agent_recommendation
        }
      ];
    def module_leads($module_path):
      [$task_entries[] |
        select(.source_ref | startswith($module_path + "/")) |
        {
          entry_ref,
          capability: .capability_id,
          semantic_reason,
          disposition,
          source_ref,
          subject_first_line,
          subject_last_line,
          relation_to_function: "module_context_not_exact_function_mapping",
          historical_green: "not_assessed",
          reuse_decision: "not_assessed"
        }
      ] | sort_by(.entry_ref);
    def module_context($module_path; $source_root):
      if $source_root == "cmd" then
        {
          rule_id: "cmd_legacy_bootstrap",
          disposition: "supersede",
          capability_ids: [],
          reason: "Bootstrap y runtime legacy: consultar la mecánica, nunca portar su autoridad o lifecycle.",
          characterization_required: true,
          relation_to_function: "cmd_legacy_bootstrap_scope_not_semantically_assessed"
        }
      else
        (first($legacy_go[0].module_rules[] |
          select(any(.paths[]; . == $module_path))) |
        {
          rule_id: .id,
          disposition,
          capability_ids,
          reason,
          characterization_required,
          relation_to_function: "module_scope_not_exact_semantic_assessment"
        })
      end;
    (map(select(
      ($name == "" or .name == $name) and
      ($path_filter == "" or (.source_path | startswith($path_filter))) and
      ($package_filter == "" or .package_name == $package_filter) and
      ($occurrence_ref == "" or .occurrence_ref == $occurrence_ref) and
      ($symbol_kind == "" or .symbol_kind == $symbol_kind) and
      ($text_filter == "" or
        (([.name, .receiver, .signature, .package_name, .source_path] | join(" ") |
          ascii_downcase) | contains($text_filter | ascii_downcase)))
    ))) as $matched |
    ($matched | map(
      . as $function |
      (semantic_links($function.occurrence_ref)) as $semantic |
      . + {
        semantic_state:
          (if ($semantic | length) > 0 then "linked_to_characterized_behavior"
          else "structural_only_not_semantically_assessed" end),
        semantic_links: $semantic
      }
    ) | sort_by([
      (if .name == $name and $name != "" then 0 else 1 end),
      (if (.semantic_links | length) > 0 then 0 else 1 end),
      .name, .receiver, .source_path, .start_line
    ])) as $ranked |
    ($ranked | length) as $total_matches |
    ($ranked[0:$limit]) |
    map(
      . as $function |
      (($function.source_path | split("/") | .[0:2] | join("/"))) as $module_path |
      (module_leads($module_path)) as $module_leads |
      . + {
        snapshot_scope: $function.source_root,
        module_context: module_context($module_path; $function.source_root),
        module_semantic_lead_count: ($module_leads | length),
        module_semantic_leads: $module_leads[0:5],
        query_total_matches: $total_matches,
        query_truncated: ($total_matches > $limit),
        authority: "derived_advisory_read_model",
        contains_body: false,
        claims_accreditation: false
      }
    )
  ' "$index_path" "$cmd_index_path")"

if [[ "$(jq 'length' <<<"$result")" == "0" ]]; then
  printf 'Sin funciones legacy para los filtros dados.\n' >&2
  exit 1
fi
if ((output_json == 1)); then
  jq '.' <<<"$result"
else
  jq -c '.[]' <<<"$result"
fi
