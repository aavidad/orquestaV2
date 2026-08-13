#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_reutilizacion_legacy.sh"
assessments="${script_dir}/../product/knowledge/legacy_reuse_assessments_v1.jsonl"
observations="${script_dir}/../product/knowledge/legacy_test_observations_v1.jsonl"
assessment_manifest="${assessments%.jsonl}.manifest.json"

bash -n "$query"
guard="${script_dir}/lib/legacy_read_model_guard.sh"
# shellcheck source=scripts/lib/legacy_read_model_guard.sh
source "$guard"
legacy_validate_reuse_assessments "$assessments" "$assessment_manifest"
expected_behaviors="$(jq -er '.assessment_count' "$assessment_manifest")"
expected_links="$(jq -er '.function_behavior_link_count' "$assessment_manifest")"
expected_occurrences="$(jq -er '.unique_function_occurrence_count' "$assessment_manifest")"
jq -e -s --argjson expected_behaviors "$expected_behaviors" '
  length == $expected_behaviors and length >= 53 and
  (map(.characterization_ref) | unique | length) == $expected_behaviors and
  all(.[ ];
    .schema_version == 1 and
    .authority == "advisory_not_capability_state" and
    .review_state == "independent_bootstrap_counterreview_completed_advisory" and
    (.historical_green_state == "documented_green_unverified" or
      .historical_green_state == "partial_or_mixed" or
      .historical_green_state == "failed_or_negative" or
      .historical_green_state == "unknown") and
    (.reuse_kind == "v2_existing" or .reuse_kind == "contract" or
      .reuse_kind == "idea" or .reuse_kind == "negative_lesson") and
    (.exact_v2_fit == "exact_or_primary" or
      .exact_v2_fit == "partial_overlap" or
      .exact_v2_fit == "not_implemented") and
    (.green_rationale | length) > 30 and
    (.agent_recommendation | length) > 30 and
    (.function_mapping.state == "pending" or
      .function_mapping.state == "linked_exact" or
      .function_mapping.state == "reviewed_no_direct_function" or
      .function_mapping.state == "blocked_historical_snapshot") and
    (.function_mapping.rationale | length) > 30 and
    .function_mapping.snapshot_census_sha256 ==
      "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" and
    (if .function_mapping.state == "linked_exact"
      then (.function_mapping.links | length) > 0 and
        all(.function_mapping.links[];
          .schema_version == 1 and
          (.occurrence_ref | startswith("sha256:")) and
          (.declaration_ref | startswith("sha256:")) and
          (.variant_ref | startswith("go-function-variant:sha256:")) and
          (.source_path | startswith("modulos/orquesta-")) and
          (.git_blob_oid | test("^[0-9a-f]{40}$")) and
          (.blob_digest | startswith("sha256:")) and
          (.signature | startswith("func ")) and
          (.semantic_rationale | length) > 30 and
          (.evidence_refs | length) > 0 and
          .review_state == "bootstrap_symbol_mapping_reviewed_advisory" and
          (has("body") | not) and
          (has("canonical_source") | not))
      else (.function_mapping.links | length) == 0 end)
  )
' "$assessments" >/dev/null

jq -e -s '
  length == 3 and
  (map(.observation_ref) | unique | length) == 3 and
  all(.[ ];
    .schema_version == 1 and
    .repository_commit == "82644ed80f4eb826099d437a643df59a9c10315b" and
    .repository_tracked_status == "clean" and
    .result == "passed" and
    .verification_kind == "scoped_unit_test_current_snapshot_not_product_attestation" and
    (.module_tree_oid | test("^[0-9a-f]{40}$")) and
    (.command | startswith("go test -mod=vendor -count=1 ./modulos/")) and
    (.test_refs | length) > 0 and
    (.behavior_refs | length) > 0 and
    (.limitations | length) > 0 and
    .authority == "historical_test_observation_advisory" and
    .claims_accreditation == false
  )
' "$observations" >/dev/null

guard_dir="$(mktemp -d)"
guard_data="${guard_dir}/assessments.jsonl"
guard_manifest="${guard_dir}/manifest.json"
cp -- "$assessments" "$guard_data"
cp -- "${assessments%.jsonl}.manifest.json" "$guard_manifest"
legacy_validate_reuse_assessments "$guard_data" "$guard_manifest"
printf '%s\n' '{"schema_version":1}' >"$guard_data"
if legacy_validate_reuse_assessments "$guard_data" "$guard_manifest" >/dev/null 2>&1; then
  printf '%s\n' 'El guard aceptó un assessment truncado.' >&2
  exit 1
fi
rm -f -- "$guard_data" "$guard_manifest"
rmdir -- "$guard_dir"

all="$($query --all --json)"
expected_task_refs="$(jq '[.[].source_task_entry_refs[]] | unique | length' <<<"$all")"
expected_capabilities="$(jq '[.[].capability_id] | unique | length' <<<"$all")"
deep_capabilities="$(jq '[.[].capability_id] | unique' <<<"$all")"
expected_coverage="$(jq -n --argjson refs "$expected_task_refs" \
  '(($refs * 10000 / 2108 | floor) / 100)')"
expected_green_states="$(jq 'group_by(.historical_green.state) | map({key:.[0].historical_green.state,value:length}) | from_entries' <<<"$all")"
expected_reuse_kinds="$(jq 'group_by(.reuse.kind) | map({key:.[0].reuse.kind,value:length}) | from_entries' <<<"$all")"
expected_v2_statuses="$(jq 'group_by(.v2.status) | map({key:.[0].v2.status,value:length}) | from_entries' <<<"$all")"
expected_v2_capability_statuses="$(jq 'group_by(.capability_id) | map(.[0]) | group_by(.v2.status) | map({key:.[0].v2.status,value:length}) | from_entries' <<<"$all")"
expected_exact_behaviors="$(jq '[.[]|select(.function_mapping.state=="linked_exact")]|length' <<<"$all")"
jq -e --argjson expected_behaviors "$expected_behaviors" \
  --argjson expected_task_refs "$expected_task_refs" '
  length == $expected_behaviors and
  (map(.id) | unique | length) == $expected_behaviors and
  (map(.source_task_entry_refs[]) | unique | length) == $expected_task_refs and
  all(.[ ];
    .schema_version == 1 and
    .authority == "derived_advisory_read_model" and
    .canonical_state_change == false and
    .closes_capability == false and
    .claims_accreditation == false and
    .historical_green.independently_verified == false and
    (.historical_green.claims | length) > 0 and
    (.historical_green.limitations | length) > 0 and
    (.historical_green.test_observations | type) == "array" and
    (.mechanism.decision_authority | length) > 10 and
    (.mechanism.preserve | length) > 0 and
    (.mechanism.avoid | length) > 0 and
    (.reuse.code_reuse_state ==
      "not_assessed_requires_function_provenance_license_and_architecture_fit") and
    (.function_mapping.state | length) > 0 and
    (.v2.decision == "accept" or .v2.decision == "conditional" or
      (.v2.decision == "reject" and .reuse.kind == "negative_lesson"))
  )
' <<<"$all" >/dev/null

jsonl="$($query --all --jsonl)"
jq -e -s --argjson expected_behaviors "$expected_behaviors" '
  length == $expected_behaviors and
  all(.[ ];
    (.id | startswith("LEGACY-REUSE::BEHAVIOR-AGENT-BATCH-")) and
    (.title | length) > 5 and
    (.summary | length) > 20 and
    (.family | test("^[A-Z]+-[0-9]+$")) and
    (.sources | length) > 0 and
    (.attempts | length) > 0 and
    (.uncertainties | length) > 0
  )
' <<<"$jsonl" >/dev/null

agent_cards="$($query --capability ORC-13)"
jq -e -s '
  length == 5 and
  all(.[ ];
    (.ref | startswith("BEHAVIOR-AGENT-BATCH-07-")) and
    .capability == "ORC-13" and
    .v2_status == "accredited" and
    .historical_green_verified == false and
    (.problem | length) > 20 and
    (.result_reason | length) > 20 and
    (.historical_test_observations | type) == "array" and
    (.mechanism | length) > 20 and
    (.preserve | length) > 0 and
    (.avoid | length) > 0 and
    (.action | length) > 20 and
    .function_mapping.state == "linked_exact" and
    (.function_mapping.links | length) > 0 and
    (.source_refs | length) > 0 and
    (.uncertainties | length) > 0
  )
' <<<"$agent_cards" >/dev/null

orc13="$($query --capability ORC-13 --json)"
jq -e '
  length == 5 and
  all(.[ ];
    .v2.status == "accredited" and
    (.v2.evidence_refs | index("product/evidence/v06_atomic_state_outbox.json")) != null
  ) and
  (map(select(.behavior_key == "outbox_pendiente_bloquea_avance" and
    .reuse.kind == "contract" and .reuse.exact_v2_fit == "partial_overlap")) | length) == 1
' <<<"$orc13" >/dev/null

exact_ref="sha256:dcc8f47db4bb4273329ca49c4926e6a846cf9e8fe5e18699f0eddcd2c3733bf5"
exact="$($query --capability ORC-13 --legacy-function-ref "$exact_ref" --json)"
jq -e --arg exact_ref "$exact_ref" '
  length == 1 and
  .[0].behavior_key == "claim_lease_ack_correlacionados" and
  any(.[0].function_mapping.links[]; .occurrence_ref == $exact_ref)
' <<<"$exact" >/dev/null

agt12="$($query --capability AGT-12 --json)"
jq -e '
  length == 1 and
  .[0].v2.status == "declared" and
  .[0].historical_green.state == "failed_or_negative" and
  .[0].reuse.kind == "contract"
' <<<"$agt12" >/dev/null

summary="$($query --summary)"
jq -e \
  --argjson expected_behaviors "$expected_behaviors" \
  --argjson expected_task_refs "$expected_task_refs" \
  --argjson expected_capabilities "$expected_capabilities" \
  --argjson expected_coverage "$expected_coverage" \
  --argjson expected_green_states "$expected_green_states" \
  --argjson expected_reuse_kinds "$expected_reuse_kinds" \
  --argjson expected_v2_statuses "$expected_v2_statuses" \
  --argjson expected_v2_capability_statuses "$expected_v2_capability_statuses" \
  --argjson expected_links "$expected_links" \
  --argjson expected_exact_behaviors "$expected_exact_behaviors" \
  --argjson expected_occurrences "$expected_occurrences" \
  --argjson deep_capabilities "$deep_capabilities" '
  .behaviors == $expected_behaviors and
  .inventory_layers.markdown_sources ==
    {"classified":840,"total":840,"coverage_percent":100,"actionable":339,"context_only":501} and
  .inventory_layers.task_semantic_leads ==
    {"reviewed":2108,"total":2108,"coverage_percent":100} and
  .inventory_layers.go_function_snapshot ==
    {"indexed":15543,"total":15543,"coverage_percent":100,"functions":12782,"methods":2761,
      "scopes":[{"source_root":"modulos","declarations":13204},
                {"source_root":"cmd","declarations":2339}]} and
  .inventory_layers.deep_behavior_assessment ==
    {"task_entry_refs":$expected_task_refs,"total_task_entry_refs":2108,
      "coverage_percent":$expected_coverage,"behaviors":$expected_behaviors,
      "capabilities":$expected_capabilities,"total_capabilities":184,
      "capability_coverage_percent":
        (($expected_capabilities * 10000 / 184 | floor) / 100)} and
  .capabilities == $expected_capabilities and
  .coverage_scope == "legacy_task_ledger_capabilities_not_product_roadmap" and
  .capability_ledger_total == 184 and
  .roadmap_catalog_size == 257 and
  .capability_deep_assessment_coverage_percent ==
    (($expected_capabilities * 10000 / 184 | floor) / 100) and
  .task_entry_refs_characterized == $expected_task_refs and
  .task_entry_ledger_total == 2108 and
  .task_entry_ledger_semantic_leads_reviewed == 2108 and
  .ledger_semantic_lead_coverage_percent == 100 and
  .task_entry_refs_uncharacterized == (2108 - $expected_task_refs) and
  .task_entry_reference_coverage_percent == $expected_coverage and
  .deep_behavior_assessment_coverage_percent == $expected_coverage and
  .historical_green_states == $expected_green_states and
  .reuse_kinds == $expected_reuse_kinds and
  .behavior_v2_statuses == $expected_v2_statuses and
  .v2_capability_statuses == $expected_v2_capability_statuses and
  (has("v2_statuses") | not) and
  .reuse_assessments_counterreviewed == $expected_behaviors and
  .source_characterizations_still_marked_pending == $expected_behaviors and
  .verified_historical_greens == 0 and
  .behaviors_with_passing_snapshot_tests == 5 and
  .function_behavior_links == $expected_links and
  .behaviors_with_exact_function_links == $expected_exact_behaviors and
  .unique_exact_function_occurrences == $expected_occurrences and
  (.next_uncharacterized_capabilities | type) == "array" and
  all(.next_uncharacterized_capabilities[];
    (.capability_id as $candidate |
      ($deep_capabilities | index($candidate)) == null))
' <<<"$summary" >/dev/null

if "$query" --capability WIZ-99 >/dev/null 2>&1; then
  printf 'Una consulta sin coincidencias no debe devolver éxito.\n' >&2
  exit 1
fi

printf 'status=passed behaviors=%s capabilities=%s task_refs=%s coverage=%s%% exact_behavior_links=%s verified_green=0\n' \
  "$expected_behaviors" "$expected_capabilities" "$expected_task_refs" \
  "$expected_coverage" "$expected_links"
