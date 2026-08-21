package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func validateRecoveryV16TestAttestorMigrationSource(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{"sqlite.migration_v17_artifact_provenance_ambiguous", `SELECT COUNT(*) FROM artifacts artifact WHERE (SELECT COUNT(*) FROM attestations attestation WHERE attestation.goal_ref=artifact.goal_ref AND attestation.artifact_ref=artifact.ref)<>1`},
		{"sqlite.migration_v17_attestation_scope_invalid", `SELECT COUNT(*) FROM attestations attestation LEFT JOIN artifacts artifact ON artifact.goal_ref=attestation.goal_ref AND artifact.ref=attestation.artifact_ref LEFT JOIN executions execution ON execution.goal_ref=attestation.goal_ref AND execution.work_item_ref=attestation.work_item_ref AND execution.ref=attestation.execution_ref LEFT JOIN work_items item ON item.goal_ref=attestation.goal_ref AND item.ref=attestation.work_item_ref WHERE artifact.ref IS NULL OR execution.ref IS NULL OR item.ref IS NULL OR artifact.work_item_ref<>attestation.work_item_ref`},
	})
}

func validateRecoveryV17TestAttestor(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryV17Checks(ctx, tx, recoveryV17TestAttestorChecks())
}

func recoveryV17TestAttestorChecks() []recoveryV17Check {
	return []recoveryV17Check{
		{"sqlite.recovery_v17_required_test_order_invalid", `SELECT COUNT(*) FROM work_item_required_tests required WHERE EXISTS (SELECT 1 FROM work_item_required_tests sibling WHERE sibling.goal_ref=required.goal_ref AND sibling.work_item_ref=required.work_item_ref GROUP BY sibling.goal_ref,sibling.work_item_ref HAVING MIN(sibling.position)<>0 OR MAX(sibling.position)+1<>COUNT(*))`},
		{"sqlite.recovery_v17_required_test_argument_order_invalid", `SELECT COUNT(*) FROM work_item_required_test_arguments argument WHERE EXISTS (SELECT 1 FROM work_item_required_test_arguments sibling WHERE sibling.goal_ref=argument.goal_ref AND sibling.work_item_ref=argument.work_item_ref AND sibling.required_test_ref=argument.required_test_ref GROUP BY sibling.goal_ref,sibling.work_item_ref,sibling.required_test_ref HAVING MIN(sibling.position)<>0 OR MAX(sibling.position)+1<>COUNT(*))`},
		{"sqlite.recovery_v17_artifact_cas_invalid", `SELECT COUNT(*) FROM artifact_occurrences occurrence JOIN artifacts artifact ON artifact.goal_ref=occurrence.goal_ref AND artifact.ref=occurrence.artifact_ref WHERE (length(artifact.digest)<>64 OR artifact.digest GLOB '*[^0-9a-f]*' OR artifact.ref<>'artifact:sha256:' || artifact.digest) AND NOT (occurrence.occurrence_ref='artifact-occurrence:migrated-v16:' || artifact.ref AND occurrence.kind='agent_output' AND EXISTS (SELECT 1 FROM attestations attestation WHERE attestation.kind='artifact_provenance' AND attestation.verdict='observed' AND attestation.goal_ref=occurrence.goal_ref AND attestation.work_item_ref=occurrence.work_item_ref AND attestation.execution_ref=occurrence.execution_ref AND attestation.artifact_ref=occurrence.artifact_ref))`},
		{"sqlite.recovery_v17_artifact_occurrence_duplicate", `SELECT COUNT(*) FROM artifact_occurrences occurrence WHERE EXISTS (SELECT 1 FROM artifact_occurrences sibling WHERE sibling.occurrence_ref<>occurrence.occurrence_ref AND sibling.kind=occurrence.kind AND sibling.goal_ref=occurrence.goal_ref AND sibling.work_item_ref=occurrence.work_item_ref AND sibling.execution_ref=occurrence.execution_ref AND sibling.artifact_ref=occurrence.artifact_ref AND sibling.execution_attempt=occurrence.execution_attempt AND sibling.plan_generation=occurrence.plan_generation AND sibling.work_item_generation=occurrence.work_item_generation AND sibling.app_spec_generation=occurrence.app_spec_generation AND sibling.spec_hash=occurrence.spec_hash)`},
		{"sqlite.recovery_v17_artifact_occurrence_invalid", `SELECT COUNT(*) FROM artifact_occurrences occurrence LEFT JOIN artifacts artifact ON artifact.goal_ref=occurrence.goal_ref AND artifact.ref=occurrence.artifact_ref LEFT JOIN executions execution ON execution.goal_ref=occurrence.goal_ref AND execution.work_item_ref=occurrence.work_item_ref AND execution.ref=occurrence.execution_ref LEFT JOIN work_items item ON item.goal_ref=occurrence.goal_ref AND item.ref=occurrence.work_item_ref WHERE artifact.ref IS NULL OR execution.ref IS NULL OR item.ref IS NULL OR occurrence.execution_attempt<>execution.attempt_no OR occurrence.plan_generation<>execution.plan_generation OR occurrence.app_spec_generation<>execution.app_spec_generation OR occurrence.spec_hash<>execution.spec_hash OR occurrence.work_item_generation>item.revision`},
		{"sqlite.recovery_v17_attestation_scope_invalid", `SELECT COUNT(*) FROM attestations attestation LEFT JOIN executions execution ON execution.goal_ref=attestation.goal_ref AND execution.work_item_ref=attestation.work_item_ref AND execution.ref=attestation.execution_ref LEFT JOIN work_items item ON item.goal_ref=attestation.goal_ref AND item.ref=attestation.work_item_ref LEFT JOIN artifacts artifact ON artifact.goal_ref=attestation.goal_ref AND artifact.ref=attestation.artifact_ref WHERE execution.ref IS NULL OR item.ref IS NULL OR artifact.ref IS NULL OR attestation.execution_attempt<>execution.attempt_no OR attestation.plan_generation<>execution.plan_generation OR attestation.app_spec_generation<>execution.app_spec_generation OR attestation.spec_hash<>execution.spec_hash OR attestation.work_item_generation>item.revision`},
		{"sqlite.recovery_v17_attestation_outcomes_invalid", `SELECT COUNT(*) FROM attestations attestation WHERE attestation.kind='required_tests' AND ((SELECT COUNT(*) FROM attestation_test_outcomes outcome WHERE outcome.attestation_ref=attestation.ref)<>(SELECT COUNT(*) FROM work_item_required_tests required WHERE required.goal_ref=attestation.goal_ref AND required.work_item_ref=attestation.work_item_ref) OR NOT EXISTS (SELECT 1 FROM work_item_required_tests required WHERE required.goal_ref=attestation.goal_ref AND required.work_item_ref=attestation.work_item_ref) OR EXISTS (SELECT 1 FROM attestation_test_outcomes outcome LEFT JOIN work_item_required_tests required ON required.goal_ref=outcome.goal_ref AND required.work_item_ref=outcome.work_item_ref AND required.ref=outcome.required_test_ref AND required.position=outcome.position WHERE outcome.attestation_ref=attestation.ref AND required.ref IS NULL) OR (attestation.verdict='passed' AND EXISTS (SELECT 1 FROM attestation_test_outcomes outcome WHERE outcome.attestation_ref=attestation.ref AND outcome.exit_code<>0)) OR (attestation.verdict='failed' AND NOT EXISTS (SELECT 1 FROM attestation_test_outcomes outcome WHERE outcome.attestation_ref=attestation.ref AND outcome.exit_code<>0)))`},
		{"sqlite.recovery_v17_attestation_ledger_invalid", `SELECT COUNT(*) FROM attestations attestation LEFT JOIN change_sets change_set ON change_set.ref=attestation.change_set_ref LEFT JOIN effect_intents intent ON intent.ref=attestation.effect_intent_ref LEFT JOIN effect_attempts attempt ON attempt.ref=attestation.effect_attempt_ref LEFT JOIN effect_receipts receipt ON receipt.ref=attestation.effect_receipt_ref LEFT JOIN action_consumption_receipts consumed ON consumed.kind='attest_test' AND consumed.goal_ref=attestation.goal_ref AND consumed.work_item_ref=attestation.work_item_ref AND consumed.execution_ref=attestation.execution_ref AND consumed.change_ref=attestation.change_set_ref AND consumed.fence=attestation.effect_fence WHERE attestation.kind='required_tests' AND (change_set.ref IS NULL OR change_set.goal_ref<>attestation.goal_ref OR change_set.work_item_ref<>attestation.work_item_ref OR change_set.execution_ref<>attestation.execution_ref OR change_set.execution_attempt<>attestation.execution_attempt OR change_set.plan_generation<>attestation.plan_generation OR change_set.app_spec_generation<>attestation.app_spec_generation OR change_set.app_spec_hash<>attestation.spec_hash OR intent.ref IS NULL OR intent.kind<>'attest_test' OR intent.action_kind<>'attest_test' OR intent.goal_ref<>attestation.goal_ref OR intent.work_item_ref<>attestation.work_item_ref OR intent.execution_ref<>attestation.execution_ref OR intent.plan_generation<>attestation.plan_generation OR intent.app_spec_generation<>attestation.app_spec_generation OR intent.spec_hash<>attestation.spec_hash OR attempt.ref IS NULL OR attempt.intent_ref<>intent.ref OR attempt.action_fence<>attestation.effect_fence OR receipt.ref IS NULL OR receipt.intent_ref<>intent.ref OR receipt.attempt_ref<>attempt.ref OR receipt.action_fence<>attestation.effect_fence OR receipt.status<>CASE attestation.verdict WHEN 'passed' THEN 'attested_passed' ELSE 'attested_failed' END OR consumed.action_ref IS NULL OR consumed.effect_receipt_ref<>receipt.ref OR consumed.plan_generation<>attestation.plan_generation OR consumed.work_item_generation<>attestation.work_item_generation OR consumed.outcome<>'completed' OR consumed.error_code<>'')`},
		{"sqlite.recovery_v17_integration_without_pass", `SELECT COUNT(*) FROM executions execution WHERE execution.state IN ('awaiting_integration','succeeded') AND execution.execution_workspace_ref<>'' AND EXISTS (SELECT 1 FROM change_sets change_set WHERE change_set.goal_ref=execution.goal_ref AND change_set.work_item_ref=execution.work_item_ref AND change_set.execution_ref=execution.ref) AND EXISTS (SELECT 1 FROM work_item_required_tests required WHERE required.goal_ref=execution.goal_ref AND required.work_item_ref=execution.work_item_ref) AND NOT EXISTS (SELECT 1 FROM attestations attestation WHERE attestation.kind='required_tests' AND attestation.verdict='passed' AND attestation.goal_ref=execution.goal_ref AND attestation.work_item_ref=execution.work_item_ref AND attestation.execution_ref=execution.ref AND attestation.execution_attempt=execution.attempt_no AND attestation.plan_generation=execution.plan_generation AND attestation.app_spec_generation=execution.app_spec_generation AND attestation.spec_hash=execution.spec_hash)`},
		{"sqlite.recovery_v17_failed_attestation_missing", `SELECT COUNT(*) FROM executions execution WHERE execution.state='failed' AND execution.failure_code='test_attestor.required_tests_failed' AND NOT EXISTS (SELECT 1 FROM attestations attestation WHERE attestation.kind='required_tests' AND attestation.verdict='failed' AND attestation.goal_ref=execution.goal_ref AND attestation.work_item_ref=execution.work_item_ref AND attestation.execution_ref=execution.ref)`},
		{"sqlite.recovery_v17_attest_action_scope_invalid", `SELECT COUNT(*) FROM outbox action LEFT JOIN change_sets change_set ON change_set.ref=action.change_ref LEFT JOIN executions execution ON execution.goal_ref=action.goal_ref AND execution.work_item_ref=action.work_item_ref AND execution.ref=action.execution_ref WHERE action.kind='attest_test' AND (change_set.ref IS NULL OR change_set.goal_ref<>action.goal_ref OR change_set.work_item_ref<>action.work_item_ref OR change_set.execution_ref<>action.execution_ref OR change_set.plan_generation<>action.plan_generation OR execution.ref IS NULL OR NOT EXISTS (SELECT 1 FROM work_item_required_tests required WHERE required.goal_ref=action.goal_ref AND required.work_item_ref=action.work_item_ref) OR (action.completed_at IS NULL AND execution.state<>'awaiting_attestation'))`},
	}
}

type recoveryV17Check struct{ code, query string }

func validateRecoveryV17Checks(ctx context.Context, tx *sql.Tx, checks []recoveryV17Check) error {
	for _, check := range checks {
		var count int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("%s:%d", check.code, count)
		}
	}
	return nil
}

func validateRecoveryV17Governance(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryV17GovernanceWithChecks(ctx, tx, recoveryV15Checks)
}

func validateRecoveryV17GovernanceWithChecks(ctx context.Context, tx *sql.Tx, checks []recoveryV15Check) error {
	return validateRecoveryV17GovernanceWithOutcome(ctx, tx, checks, false)
}

func validateRecoveryV17GovernanceWithOutcome(
	ctx context.Context, tx *sql.Tx, checks []recoveryV15Check, allowNonApplication bool,
) error {
	for _, check := range checks {
		if check.code == "sqlite.recovery_v15_unknown_applied_repeated" {
			continue
		}
		var invalid int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&invalid); err != nil {
			return err
		} else if invalid != 0 {
			return errors.New(check.code)
		}
	}
	for _, validate := range []func(context.Context, queryer) error{validateRecoveryV15IntentSemantics, validateRecoveryV15SettlementSemantics, validateRecoveryV15ReceiptSemantics, validateRecoveryV15ApprovalSemantics} {
		if err := validate(ctx, tx); err != nil {
			return err
		}
	}
	unknownAppliedQuery := `SELECT COUNT(*) FROM effect_attempts later JOIN effect_attempts prior ON prior.action_ref=later.action_ref AND prior.intent_ref=later.intent_ref AND prior.action_fence<later.action_fence LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=prior.ref WHERE receipt.ref IS NULL AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=prior.ref)`
	if allowNonApplication {
		unknownAppliedQuery = `SELECT COUNT(*) FROM effect_attempts later JOIN effect_attempts prior ON prior.action_ref=later.action_ref AND prior.intent_ref=later.intent_ref AND prior.action_fence<later.action_fence LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=prior.ref WHERE receipt.ref IS NULL AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=prior.ref) AND NOT EXISTS (SELECT 1 FROM effect_non_application_evidence evidence WHERE evidence.attempt_ref=prior.ref AND evidence.outcome='definitely_not_applied')`
	}
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{"sqlite.recovery_v17_causal_settlement_invalid", `SELECT COUNT(*) FROM budget_settlements settlement JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref LEFT JOIN effect_attempts attempt ON attempt.ref=settlement.causal_attempt_ref WHERE settlement.causal_attempt_ref IS NOT NULL AND (attempt.ref IS NULL OR attempt.action_ref<>reservation.action_ref OR attempt.intent_ref<>reservation.effect_intent_ref OR reservation.fence>attempt.action_fence OR settlement.observed_known<>31 OR settlement.observed_quality<>'exact' OR settlement.observed_tokens<>0 OR settlement.observed_money_micros<>0 OR settlement.observed_currency<>reservation.currency OR settlement.observed_active_time_ns<>0 OR settlement.observed_process_slots<>0 OR settlement.observed_disk_bytes<>0 OR settlement.charged_tokens<>0 OR settlement.charged_money_micros<>0 OR settlement.charged_currency<>reservation.currency OR settlement.charged_active_time_ns<>0 OR settlement.charged_process_slots<>0 OR settlement.charged_disk_bytes<>0 OR settlement.released_tokens<>reservation.tokens OR settlement.released_money_micros<>reservation.money_micros OR settlement.released_currency<>reservation.currency OR settlement.released_active_time_ns<>reservation.active_time_ns OR settlement.released_process_slots<>reservation.process_slots OR settlement.released_disk_bytes<>reservation.disk_bytes OR settlement.overrun_tokens<>0 OR settlement.overrun_money_micros<>0 OR settlement.overrun_currency<>reservation.currency OR settlement.overrun_active_time_ns<>0 OR settlement.overrun_process_slots<>0 OR settlement.overrun_disk_bytes<>0)`},
		{"sqlite.recovery_v17_unknown_applied_repeated", unknownAppliedQuery},
	})
}
