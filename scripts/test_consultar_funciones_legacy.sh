#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_funciones_legacy.sh"
assessment_manifest="${script_dir}/../product/knowledge/legacy_reuse_assessments_v1.manifest.json"

bash -n "$query"
expected_behaviors="$(jq -er '.assessment_count' "$assessment_manifest")"
expected_links="$(jq -er '.function_behavior_link_count' "$assessment_manifest")"
expected_occurrences="$(jq -er '.unique_function_occurrence_count' "$assessment_manifest")"

summary="$($query --summary)"
jq -e --argjson expected_behaviors "$expected_behaviors" \
  --argjson expected_links "$expected_links" \
  --argjson expected_occurrences "$expected_occurrences" '
  .document_kind == "legacy_go_function_snapshot_index_union" and
  .production_file_count == 2258 and
  .function_count == 12782 and
  .method_count == 2761 and
  .declaration_count == 15543 and
  .unique_occurrence_count == 15543 and
  (.scopes | length) == 2 and
  (.scopes | map(.source_root) | sort) == ["cmd","modulos"] and
  .contains_bodies == false and
  .semantic_behavior_count == $expected_behaviors and
  .semantic_behavior_count >= 53 and
  .semantic_function_behavior_links == $expected_links and
  .semantic_function_behavior_links >= 145 and
  .structurally_indexed_occurrences_with_semantic_links == $expected_occurrences and
  .structurally_indexed_occurrences_with_semantic_links >= 125 and
  .structurally_indexed_occurrences_without_semantic_links ==
    (15543 - $expected_occurrences) and
  .ledger_semantic_leads_reviewed == 2108 and
  .ledger_semantic_lead_total == 2108 and
  .ledger_semantic_lead_coverage_percent == 100 and
  .git_snapshot_ast_census_coverage_percent == 100 and
  .claims_accreditation == false
' <<<"$summary" >/dev/null

handler="$($query --name HandleCommandV0 --json)"
jq -e '
  length == 1 and
  .[0].name == "HandleCommandV0" and
  .[0].source_path == "modulos/orquesta-core-workflow/handler_v0.go" and
  .[0].semantic_state == "linked_to_characterized_behavior" and
  (.[0].semantic_links | length) >= 1 and
  .[0].module_context.rule_id == "lifecycle_state_and_scheduler" and
  (.[0].module_context.capability_ids | index("ORC-13")) != null and
  .[0].module_context.relation_to_function ==
    "module_scope_not_exact_semantic_assessment" and
  .[0].module_semantic_lead_count > 0 and
  (.[0].module_semantic_leads | length) > 0 and
  all(.[0].module_semantic_leads[];
    .relation_to_function == "module_context_not_exact_function_mapping" and
    .historical_green == "not_assessed" and
    .reuse_decision == "not_assessed") and
  .[0].contains_body == false and
  .[0].claims_accreditation == false and
  .[0].query_total_matches == 1 and
  .[0].query_truncated == false and
  (.[0] | has("canonical_source") | not) and
  (.[0] | has("body") | not)
' <<<"$handler" >/dev/null

ref="$(jq -r '.[0].occurrence_ref' <<<"$handler")"
exact="$($query --occurrence-ref "$ref" --json)"
jq -e --arg ref "$ref" '
  length == 1 and .[0].occurrence_ref == $ref
' <<<"$exact" >/dev/null

methods="$($query --name ObserveGoalWorkV0 --kind method --json)"
jq -e '
  length == 10 and
  (map(.receiver) | unique | length) == 10 and
  (map(select(.snapshot_scope == "modulos")) | length) == 7 and
  (map(select(.snapshot_scope == "cmd")) | length) == 3 and
  all(.[]; .symbol_kind == "method")
' <<<"$methods" >/dev/null

ranked="$($query --text director --limit 20 --json)"
jq -e '
  length == 20 and .[0].query_total_matches == 2867 and
  .[0].query_truncated == true and
  any(.[]; .semantic_state == "linked_to_characterized_behavior")
' <<<"$ranked" >/dev/null

cmd_legacy="$($query --name codexServerWorktreeSnapshotBudgetSettingsV0 --json)"
jq -e '
  length == 1 and
  .[0].source_path == "cmd/orquesta-server/worktree_snapshot_budget_env_v0.go" and
  .[0].snapshot_scope == "cmd" and
  .[0].semantic_state == "structural_only_not_semantically_assessed" and
  .[0].module_context.relation_to_function ==
    "cmd_legacy_bootstrap_scope_not_semantically_assessed" and
  .[0].contains_body == false and
  (.[0] | has("body") | not) and
  (.[0] | has("canonical_source") | not)
' <<<"$cmd_legacy" >/dev/null

if "$query" --name FunctionThatNeverExistedV0 >/dev/null 2>&1; then
  printf 'Una función ausente no debe devolver éxito.\n' >&2
  exit 1
fi
if "$query" --limit 0 --name HandleCommandV0 >/dev/null 2>&1; then
  printf 'Un límite inválido no debe aceptarse.\n' >&2
  exit 1
fi

printf 'status=passed structural_functions=15543 semantic_occurrences=%s\n' \
  "$expected_occurrences"
