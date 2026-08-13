#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_catalogo_funciones_legacy_v1.sh"

bash -n "$query"

# El consumidor simula otra aplicación: la consulta parte de un cwd que no
# pertenece al repositorio y no recibe capability, ruta V2 ni operación V2.
summary="$(cd /tmp && "$query" --summary)"
jq -e '
  .document_kind == "legacy_v1_general_function_catalog_summary" and
  .audience == "any_application" and
  .function_count == 12782 and
  .method_count == 2761 and
  .declaration_count == 15543 and
  .unique_occurrence_count == 15543 and
  .behavior_assessment_count == 235 and
  .function_behavior_link_count == 347 and
  .unique_semantically_linked_functions == 306 and
  .contains_bodies == false and
  .code_reuse_authorized == false and
  .consumer_application_decisions_included == false and
  .claims_accreditation == false and
  (has("capability_id") | not) and
  (has("v2") | not)
' <<<"$summary" >/dev/null

linked="$(cd /tmp && "$query" --name HandleCommandV0 --json)"
jq -e '
  length == 1 and
  .[0].function.name == "HandleCommandV0" and
  (.[0].function.signature | startswith("func HandleCommandV0")) and
  .[0].semantic_state == "linked_to_reviewed_behavior_characterization" and
  (.[0].behavior_assessments | length) >= 1 and
  all(.[0].behavior_assessments[];
    (.problem | length) > 20 and
    (.result_reason | length) > 20 and
    (.mechanism | length) > 20 and
    (.worked_claims | type) == "array" and
    (.failures_and_limitations | type) == "array" and
    (.snapshot_test_observations | type) == "array" and
    (.relation | length) > 0 and
    (.semantic_rationale | length) > 20 and
    (has("capability_id") | not) and
    (has("v2") | not) and
    (has("reuse_kind") | not)) and
  .[0].license.state ==
    "not_declared_per_function_requires_source_review" and
  .[0].license.permits_code_reuse == false and
  (.[0].reuse_conditions | length) == 5 and
  .[0].contains_body == false and
  .[0].code_reuse_authorized == false and
  .[0].consumer_application_decision == null and
  (.[0].function | has("body") | not) and
  (.[0] | has("capability_id") | not) and
  (.[0] | has("v2") | not)
' <<<"$linked" >/dev/null

structural_ref="sha256:874b04f5c2fb252933257660c4b73a862cfca75cdd70c097dfc2f3182c33de73"
structural="$(cd /tmp && "$query" --occurrence-ref "$structural_ref" --json)"
jq -e --arg ref "$structural_ref" '
  length == 1 and
  .[0].function.occurrence_ref == $ref and
  .[0].semantic_state == "structural_only_not_semantically_assessed" and
  .[0].behavior_assessments == [] and
  .[0].code_reuse_authorized == false and
  .[0].contains_body == false
' <<<"$structural" >/dev/null

if (cd /tmp && "$query" --name FunctionThatNeverExistedV0 >/dev/null 2>&1); then
  printf 'Una función ausente no debe devolver éxito.\n' >&2
  exit 1
fi

printf 'status=passed portable_catalog=15543 cwd_independent=true v2_neutral=true\n'
