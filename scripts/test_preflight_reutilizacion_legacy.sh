#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "$0")" && pwd -P)"
preflight="$script_dir/preflight_reutilizacion_legacy.sh"

bash -n "$preflight"

outbox="$("$preflight" --capability ORC-13 --path internal/application/processing.go --operation outbox --task 'entregar outbox sin duplicar el efecto' --function ClaimNextAction)"
jq -e '
  .purpose == "agent_preflight_before_design_or_implementation" and
  .authority == "advisory_no_state_change" and
  .coverage.state == "capability_candidates_with_reviewed_legacy_function_links" and
  .coverage.candidate_count == 5 and
  .coverage.candidates_returned == 3 and
  .coverage.candidates_truncated == true and
  .coverage.ledger_semantic_lead_count >= 0 and
  .coverage.structural_function_candidate_count == 0 and
  .coverage.exact_function_match == false and
  (.cross_cutting_lessons | map(.pattern_id) | index("LEGACY-004")) != null and
  (.reuse_candidates | length) == 3 and
  (.decision_summary.primary |
    IN("reuse", "reimplement", "characterize", "reject")) and
  all(.reuse_candidates[];
    .v2.status == "accredited" and
    .v1_green_verified == false and
    (.problem | length) > 20 and
    (.result_reason | length) > 20 and
    (.snapshot_test_observation_refs | type) == "array" and
    (.mechanism | length) > 20 and
    (.preserve | length) > 0 and
    (.avoid | length) > 0 and
    (.decision | IN("reuse", "reimplement", "characterize", "reject")) and
    (.action | length) > 20 and
    (.exact_legacy_source_links | length) > 0 and
    .function_mapping_state == "linked_exact"
  ) and
  .structural_function_candidates == [] and
  .canonical_state_change == false and
  .creates_work_item == false and
  .closes_capability == false and
  .claims_accreditation == false
' <<<"$outbox" >/dev/null

exact_ref="sha256:dcc8f47db4bb4273329ca49c4926e6a846cf9e8fe5e18699f0eddcd2c3733bf5"
exact="$($preflight --capability ORC-13 --path internal/application/processing.go --operation outbox --task 'cerrar entrega confirmada' --legacy-function-ref "$exact_ref")"
jq -e --arg exact_ref "$exact_ref" '
  .query.legacy_function_ref == $exact_ref and
  .coverage.state == "exact_legacy_function_linked" and
  .coverage.candidate_count == 1 and
  .coverage.structural_function_candidate_count == 1 and
  .coverage.exact_function_match == true and
  .reuse_candidates[0].behavior == "claim_lease_ack_correlacionados" and
  .reuse_candidates[0].match_authority == "exact_legacy_function_ref" and
  any(.reuse_candidates[0].exact_legacy_source_links[];
    .occurrence_ref == $exact_ref) and
  .structural_function_candidates[0].occurrence_ref == $exact_ref
' <<<"$exact" >/dev/null

named="$($preflight --capability ORC-13 --path internal/application/processing.go --operation command --task 'aplicar comando durable' --function HandleCommandV0)"
jq -e '
  .coverage.structural_function_candidate_count == 1 and
  .structural_function_candidates[0].name == "HandleCommandV0" and
  .structural_function_candidates[0].module_context.rule_id ==
    "lifecycle_state_and_scheduler" and
  .structural_function_candidates[0].semantic_state ==
    "linked_to_characterized_behavior" and
  (.structural_function_candidates[0].semantic_links | length) >= 1
' <<<"$named" >/dev/null

# ORC-12 supera cien pistas. El preflight debe paginar sus entradas internas en
# vez de convertir todo el ledger en un único argumento de proceso.
large_family="$($preflight --capability ORC-12 --path internal/application/processing.go --operation retry --task 'reintentar efecto externo sin duplicarlo' --function Retry)"
jq -e '
  .coverage.candidate_count >= 7 and
  .coverage.ledger_semantic_lead_count >= 0 and
  .coverage.ledger_semantic_lead_count <= 83 and
  .coverage.ledger_semantic_leads_returned ==
    ([.coverage.ledger_semantic_lead_count, 5] | min) and
  .coverage.structural_function_candidate_count == 91 and
  .coverage.structural_functions_returned == 3 and
  .coverage.structural_functions_truncated == true and
  (.ledger_semantic_leads | length) == .coverage.ledger_semantic_leads_returned and
  all(.ledger_semantic_leads[];
    .capability == "ORC-12" and
    .depth == "ledger_semantic_lead_not_deep_behavior_assessment" and
    .historical_green == "not_assessed" and
    .function_mapping == "not_assessed" and
    .claims_historical_success == false and
    .claims_reuse_decision == false) and
  (.structural_function_candidates | length) == 3
' <<<"$large_family" >/dev/null

gap="$("$preflight" --capability WIZ-99 --path internal/application/example.go --operation plan --task 'conducta aún no inventariada')"
jq -e '
  .coverage.state == "no_semantic_candidate_for_capability" and
  .coverage.candidate_count == 0 and
  .cross_cutting_lessons == [] and
  .reuse_candidates == [] and
  .ledger_semantic_leads == [] and
  .source_disposition_leads == [] and
  .structural_function_candidates == [] and
  .claims_accreditation == false
' <<<"$gap" >/dev/null

# E2E de las cuatro decisiones del protocolo. Cada recorrido usa capability,
# ruta, operación, tarea y función; la decisión nunca autoriza copiar V1.
reuse_case="$($preflight --capability ORC-13 --path internal/application/processing.go --operation outbox --task 'mutacion y outbox coordinadas' --function RecordDirectorCycleOutboxV0)"
reimplement_case="$($preflight --capability AGT-04 --path internal/adapters/agents/hermes.go --operation external-agent --task 'integrar Hermes por protocolo externo' --function ExecuteDirectorCycleStepV0)"
characterize_case="$($preflight --capability ORC-28 --path internal/application/processing.go --operation placement --task 'reservar capacidad antes del lanzamiento' --function ClaimNextAction)"
reject_case="$($preflight --capability APP-08 --path internal/application/errors.go --operation diagnostics --task 'evitar diagnósticos inseguros y secretos en logs' --function ValidateError)"
jq -e '.decision_summary.primary == "reuse" and any(.reuse_candidates[]; .decision == "reuse")' <<<"$reuse_case" >/dev/null
jq -e '.decision_summary.primary == "reimplement" and any(.reuse_candidates[]; .decision == "reimplement")' <<<"$reimplement_case" >/dev/null
jq -e '.decision_summary.primary == "characterize" and any(.reuse_candidates[]; .decision == "characterize")' <<<"$characterize_case" >/dev/null
jq -e '.decision_summary.primary == "reject" and any(.reuse_candidates[]; .decision == "reject")' <<<"$reject_case" >/dev/null
jq -e '
  all(.reuse_candidates[];
    (.result_reason | length) > 20 and
    (.mechanism | length) > 20 and
    (.preserve | type) == "array" and
    (.avoid | type) == "array" and
    (.snapshot_test_observation_refs | type) == "array" and
    (.exact_legacy_source_links | type) == "array") and
  .claims_accreditation == false
' <<<"$reuse_case" >/dev/null

if "$preflight" --capability ORC-13 --path internal/application/processing.go --operation outbox >/dev/null 2>&1; then
  printf 'El preflight no debe aceptar una intención sin tarea o función.\n' >&2
  exit 1
fi

printf 'status=passed agent_preflight=outbox decisions=4 gap=explicit\n'
