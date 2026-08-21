package sqlite

import (
	"context"
	"database/sql"
)

func validateRecoveryV37EffectNonApplication(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{{
		"sqlite.recovery_v37_effect_non_application_invalid", `SELECT COUNT(*) FROM effect_non_application_evidence evidence
LEFT JOIN effect_attempts attempt ON attempt.ref=evidence.attempt_ref
LEFT JOIN effect_intents intent ON intent.ref=attempt.intent_ref
WHERE attempt.ref IS NULL OR intent.ref IS NULL OR intent.kind<>'agent_stop'
 OR evidence.ref<>'effect-attempt-outcome:' || attempt.ref || ':definitely-not-applied'
 OR evidence.intent_ref<>attempt.intent_ref OR evidence.intent_digest<>attempt.intent_digest
 OR evidence.approval_ref<>attempt.approval_ref OR evidence.project_ref<>attempt.project_ref OR evidence.goal_ref<>attempt.goal_ref
 OR evidence.work_item_ref<>attempt.work_item_ref OR evidence.execution_ref<>attempt.execution_ref
 OR evidence.plan_generation<>attempt.plan_generation OR evidence.app_spec_generation<>attempt.app_spec_generation
 OR evidence.spec_hash<>attempt.spec_hash OR evidence.actor_ref<>attempt.actor_ref
 OR evidence.action_ref<>attempt.action_ref OR evidence.action_fence<>attempt.action_fence
 OR evidence.idempotency_key<>attempt.idempotency_key OR evidence.outcome<>'definitely_not_applied'
 OR evidence.observed_at<attempt.started_at OR evidence.observed_at>attempt.claim_lease_until
 OR EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
 OR EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=attempt.ref)`}, {
		"sqlite.recovery_v37_unknown_applied_repeated", `SELECT COUNT(*) FROM effect_attempts later
JOIN effect_attempts prior ON prior.action_ref=later.action_ref
 AND prior.intent_ref=later.intent_ref AND prior.action_fence<later.action_fence
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=prior.ref
WHERE receipt.ref IS NULL
 AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=prior.ref)
 AND NOT EXISTS (SELECT 1 FROM effect_non_application_evidence evidence
                 WHERE evidence.attempt_ref=prior.ref
                   AND evidence.outcome='definitely_not_applied')`}})
}
