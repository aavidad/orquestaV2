package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
)

// This proof is deliberately stricter than merely finding a receipt for the
// historical attempt. It is the sole exception that permits an agent-launch
// receipt after the original physical-attempt lease elapsed.
const terminalAgentLaunchReconciliationReceiptProof = `EXISTS (
 SELECT 1 FROM agent_launch_reconciliation_receipts terminal_receipt
 JOIN agent_launch_reconciliation_authorities terminal_authority
   ON terminal_authority.ref=terminal_receipt.authority_ref
 JOIN agent_launch_reconciliation_jobs terminal_job
   ON terminal_job.authority_ref=terminal_authority.ref
  AND terminal_job.ref=terminal_authority.job_ref
 JOIN agent_launch_reconciliation_attempts terminal_attempt
   ON terminal_attempt.ref=terminal_receipt.reconciliation_attempt_ref
  AND terminal_attempt.authority_ref=terminal_authority.ref
 WHERE terminal_receipt.outcome='completed'
  AND terminal_receipt.error_code=''
  AND terminal_receipt.effect_receipt_ref=receipt.ref
  AND terminal_job.state='completed'
  AND terminal_authority.effect_attempt_ref=attempt.ref
  AND terminal_authority.effect_intent_ref=intent.ref
  AND terminal_authority.effect_intent_digest=attempt.intent_digest
  AND terminal_authority.action_ref=attempt.action_ref
  AND terminal_authority.action_fence=attempt.action_fence
)`

func validateRecoveryV40TerminalAgentLaunchReconciliation(ctx context.Context, tx *sql.Tx) error {
	if err := validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{
			"sqlite.recovery_v40_agent_launch_reconciliation_authority_invalid",
			`SELECT COUNT(*)
FROM agent_launch_reconciliation_authorities authority
LEFT JOIN agent_launch_reconciliation_jobs job ON job.authority_ref=authority.ref
LEFT JOIN outbox action ON action.ref=authority.action_ref
LEFT JOIN action_consumption_receipts consumed ON consumed.action_ref=authority.action_ref
LEFT JOIN effect_intents intent ON intent.ref=authority.effect_intent_ref
LEFT JOIN effect_attempts attempt ON attempt.ref=authority.effect_attempt_ref
LEFT JOIN executions execution ON execution.ref=authority.execution_ref
WHERE job.ref IS NULL OR job.ref<>authority.job_ref
 OR action.ref IS NULL OR action.kind<>'launch_agent'
 OR action.goal_ref<>authority.goal_ref OR action.work_item_ref<>authority.work_item_ref
 OR action.execution_ref<>authority.execution_ref OR action.effect_intent_ref<>authority.effect_intent_ref
 OR action.plan_generation<>authority.plan_generation
 OR action.work_item_generation<>authority.work_item_generation
 OR action.fence<>authority.action_fence OR action.completed_at IS NULL
 OR action.quarantined_at<>action.completed_at
 OR action.last_error_code<>'application.effect_unknown_applied'
 OR consumed.action_ref IS NULL OR consumed.kind<>'launch_agent'
 OR consumed.goal_ref<>authority.goal_ref OR consumed.work_item_ref<>authority.work_item_ref
 OR consumed.execution_ref<>authority.execution_ref OR consumed.plan_generation<>authority.plan_generation
 OR consumed.work_item_generation<>authority.work_item_generation
 OR consumed.fence<>authority.action_fence OR consumed.outcome<>'quarantined'
 OR consumed.error_code<>'application.effect_unknown_applied' OR consumed.effect_receipt_ref IS NOT NULL
 OR intent.ref IS NULL OR intent.kind<>'agent_launch' OR intent.digest<>authority.effect_intent_digest
 OR attempt.ref IS NULL OR attempt.action_ref<>authority.action_ref
 OR attempt.intent_ref<>authority.effect_intent_ref OR attempt.intent_digest<>authority.effect_intent_digest
 OR attempt.action_fence<>authority.action_fence
 OR execution.ref IS NULL OR execution.goal_ref<>authority.goal_ref
 OR execution.work_item_ref<>authority.work_item_ref`,
		},
		{
			"sqlite.recovery_v40_agent_launch_reconciliation_job_invalid",
			`SELECT COUNT(*)
FROM agent_launch_reconciliation_jobs job
JOIN agent_launch_reconciliation_authorities authority ON authority.ref=job.authority_ref
LEFT JOIN agent_launch_reconciliation_receipts receipt ON receipt.authority_ref=authority.ref
WHERE (job.state='pending' AND receipt.ref IS NOT NULL)
 OR (job.state='completed' AND (receipt.ref IS NULL OR receipt.outcome<>'completed'
     OR receipt.error_code<>'' OR receipt.effect_receipt_ref IS NULL
     OR receipt.reconciliation_attempt_ref IS NULL))
 OR (job.state='quarantined' AND (receipt.ref IS NULL OR receipt.outcome<>'quarantined'
     OR length(trim(receipt.error_code))=0 OR receipt.effect_receipt_ref IS NOT NULL))`,
		},
		{
			"sqlite.recovery_v40_agent_launch_reconciliation_attempt_invalid",
			`SELECT COUNT(*)
FROM agent_launch_reconciliation_attempts reconciliation_attempt
LEFT JOIN agent_launch_reconciliation_authorities authority
 ON authority.ref=reconciliation_attempt.authority_ref
LEFT JOIN agent_launch_reconciliation_jobs job ON job.authority_ref=authority.ref
WHERE authority.ref IS NULL OR job.ref IS NULL
 OR reconciliation_attempt.original_effect_attempt_ref<>authority.effect_attempt_ref
 OR reconciliation_attempt.request_fingerprint<>authority.request_fingerprint
 OR reconciliation_attempt.job_fence<=authority.action_fence
 OR reconciliation_attempt.delivery_attempt<=0
 OR reconciliation_attempt.claim_lease_until<=reconciliation_attempt.started_at
 OR reconciliation_attempt.job_fence>job.fence
 OR reconciliation_attempt.delivery_attempt>job.delivery_attempt`,
		},
		{
			"sqlite.recovery_v40_agent_launch_reconciliation_completion_invalid",
			`SELECT COUNT(*)
FROM agent_launch_reconciliation_receipts reconciliation_receipt
JOIN agent_launch_reconciliation_authorities authority
 ON authority.ref=reconciliation_receipt.authority_ref
LEFT JOIN agent_launch_reconciliation_attempts reconciliation_attempt
 ON reconciliation_attempt.ref=reconciliation_receipt.reconciliation_attempt_ref
LEFT JOIN effect_receipts receipt ON receipt.ref=reconciliation_receipt.effect_receipt_ref
LEFT JOIN executions execution ON execution.ref=authority.execution_ref
LEFT JOIN agent_capacity_reservations capacity ON capacity.action_ref=authority.action_ref
LEFT JOIN agent_capacity_transitions capacity_transition ON capacity_transition.ref=capacity.last_transition_ref
WHERE reconciliation_receipt.outcome='completed' AND (
 reconciliation_attempt.ref IS NULL OR reconciliation_attempt.authority_ref<>authority.ref
 OR receipt.ref IS NULL OR receipt.attempt_ref<>authority.effect_attempt_ref
 OR receipt.intent_ref<>authority.effect_intent_ref
 OR receipt.intent_digest<>authority.effect_intent_digest
 OR receipt.action_ref<>authority.action_ref OR receipt.action_fence<>authority.action_fence
 OR receipt.status<>'accepted' OR execution.state<>'running'
 OR execution.launch_receipt_ref<>receipt.ref OR capacity.state<>'consumed'
 OR capacity_transition.cause_kind<>'reconciliation'
 OR capacity_transition.cause_ref<>authority.ref
 OR capacity_transition.effect_attempt_ref<>authority.effect_attempt_ref
 OR capacity_transition.effect_receipt_ref<>receipt.ref)`,
		},
		{
			"sqlite.recovery_v40_agent_launch_reconciliation_unresolved_invalid",
			`SELECT COUNT(*)
FROM agent_launch_reconciliation_jobs job
JOIN agent_launch_reconciliation_authorities authority ON authority.ref=job.authority_ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=authority.effect_attempt_ref
LEFT JOIN executions execution ON execution.ref=authority.execution_ref
LEFT JOIN agent_capacity_reservations capacity ON capacity.action_ref=authority.action_ref
WHERE job.state IN ('pending','quarantined')
 AND (receipt.ref IS NOT NULL OR execution.state<>'dispatching' OR capacity.state<>'quarantined')`,
		},
	}); err != nil {
		return err
	}
	type authorityKey struct {
		ref         string
		availableAt int64
	}
	rows, err := tx.QueryContext(ctx, `SELECT authority.ref,job.available_at
FROM agent_launch_reconciliation_authorities authority
JOIN agent_launch_reconciliation_jobs job ON job.authority_ref=authority.ref
ORDER BY authority.ref`)
	if err != nil {
		return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
	}
	var keys []authorityKey
	for rows.Next() {
		var key authorityKey
		if err := rows.Scan(&key.ref, &key.availableAt); err != nil {
			_ = rows.Close()
			return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
	}
	if err := rows.Close(); err != nil {
		return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
	}
	for _, key := range keys {
		authority, found, err := readTerminalReconciliationAuthorityByRef(ctx, tx, key.ref)
		if err != nil || !found || validateTerminalReconciliationAuthority(
			authority, time.Unix(0, key.availableAt).UTC(),
		) != nil {
			return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
		}
		receipts, err := readConsumptionReceipts(ctx, tx, authority.GoalRef.String())
		if err != nil {
			return invalidRecoveryV40TerminalAgentLaunchReconciliation(err)
		}
		matches := 0
		for _, receipt := range receipts {
			if receipt.ActionRef == authority.ActionRef {
				matches++
				if application.TerminalAgentLaunchReceiptFingerprint(receipt) != authority.OriginalReceiptFingerprint {
					return invalidRecoveryV40TerminalAgentLaunchReconciliation(nil)
				}
			}
		}
		if matches != 1 {
			return invalidRecoveryV40TerminalAgentLaunchReconciliation(nil)
		}
	}
	return nil
}

func invalidRecoveryV40TerminalAgentLaunchReconciliation(cause error) error {
	return invalid(errors.Join(cause, errors.New("sqlite.recovery_v40_agent_launch_reconciliation_invalid")))
}
