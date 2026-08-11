package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
)

type recoveryV15Check struct{ code, query string }

func validateRecoveryV15Governance(ctx context.Context, tx *sql.Tx) error {
	for _, check := range recoveryV15Checks {
		var invalid int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&invalid); err != nil {
			return err
		}
		if invalid != 0 {
			return errors.New(check.code)
		}
	}
	if err := validateRecoveryV15IntentSemantics(ctx, tx); err != nil {
		return err
	}
	if err := validateRecoveryV15SettlementSemantics(ctx, tx); err != nil {
		return err
	}
	if err := validateRecoveryV15ReceiptSemantics(ctx, tx); err != nil {
		return err
	}
	return validateRecoveryV15ApprovalSemantics(ctx, tx)
}

var recoveryV15Checks = []recoveryV15Check{
	{"sqlite.recovery_v15_goal_envelope_cardinality_invalid", `
SELECT COUNT(*) FROM goals goal WHERE
 ((EXISTS(SELECT 1 FROM work_items item WHERE item.goal_ref=goal.ref AND item.governance_version=1))
   <> ((SELECT COUNT(*) FROM budget_envelopes envelope
        WHERE envelope.scope='goal' AND envelope.subject_ref=goal.ref)=1))
 OR (NOT EXISTS(SELECT 1 FROM work_items item WHERE item.goal_ref=goal.ref AND item.governance_version=1)
     AND EXISTS(SELECT 1 FROM budget_envelopes envelope
                WHERE envelope.scope='goal' AND envelope.subject_ref=goal.ref))`},
	{"sqlite.recovery_v15_work_item_authority_invalid", `
SELECT COUNT(*) FROM work_items item JOIN goals goal ON goal.ref=item.goal_ref
LEFT JOIN work_item_authorities authority ON authority.goal_ref=item.goal_ref AND authority.work_item_ref=item.ref
LEFT JOIN authorization_receipts receipt ON receipt.ref=authority.authorization_receipt_ref
WHERE (item.governance_version=1)<>(authority.work_item_ref IS NOT NULL) OR (authority.work_item_ref IS NOT NULL AND
 (authority.recorded_at<item.created_at OR receipt.ref IS NULL OR receipt.principal_ref<>authority.principal_ref
  OR receipt.permission<>authority.permission OR receipt.project_ref<>goal.project_ref OR receipt.outcome<>'allowed'
  OR (authority.source='goal_confirmation')<>(authority.permission='goals.create')
  OR receipt.resource_ref<>CASE authority.permission WHEN 'goals.create' THEN goal.project_ref ELSE goal.ref END))`},
	{"sqlite.recovery_v15_action_intent_invalid", `
SELECT COUNT(*) FROM outbox action LEFT JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
LEFT JOIN work_items item ON item.goal_ref=action.goal_ref AND item.ref=action.work_item_ref
LEFT JOIN executions execution ON execution.goal_ref=action.goal_ref AND execution.ref=action.execution_ref
WHERE (action.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
       AND (action.governance_version<>1 OR action.effect_intent_ref IS NULL))
 OR (action.governance_version=0 AND action.effect_intent_ref IS NOT NULL) OR (action.governance_version=1 AND
 (action.kind NOT IN ('launch_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                      'stop_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  OR intent.ref IS NULL OR intent.action_ref<>action.ref
  OR intent.action_kind<>action.kind OR intent.goal_ref<>action.goal_ref OR intent.work_item_ref<>action.work_item_ref
  OR intent.execution_ref<>action.execution_ref OR intent.plan_generation<>action.plan_generation
  OR intent.app_spec_generation<>execution.app_spec_generation OR intent.spec_hash<>execution.spec_hash
  OR (action.kind='launch_agent' AND (intent.demand_ref<>item.budget_demand_ref
   OR intent.demand_tokens<>item.budget_tokens OR intent.demand_money_micros<>item.budget_money_micros
   OR intent.demand_currency<>item.budget_currency OR intent.demand_active_time_ns<>item.budget_active_time_ns
   OR intent.demand_process_slots<>item.budget_process_slots OR intent.demand_disk_bytes<>item.budget_disk_bytes
   OR intent.security_criticality<>item.security_criticality OR intent.reasoning_effort<>item.reasoning_effort))))`},
	{"sqlite.recovery_v15_budget_envelope_set_invalid", `
SELECT COUNT(*) FROM budget_envelopes envelope WHERE
 (envelope.scope='goal' AND (envelope.ref<>'budget-envelope:goal:'||envelope.subject_ref||':'||envelope.policy_hash
  OR NOT EXISTS(SELECT 1 FROM goals goal WHERE goal.ref=envelope.subject_ref)
  OR (SELECT COUNT(*) FROM budget_envelopes peer JOIN goals goal ON goal.ref=envelope.subject_ref
      WHERE peer.policy_hash=envelope.policy_hash AND peer.revision=envelope.revision
       AND ((peer.scope='project' AND peer.subject_ref=goal.project_ref) OR peer.scope='deployment'))<>2))
 OR (envelope.scope='project' AND (envelope.ref<>'budget-envelope:project:'||envelope.subject_ref||':'||envelope.policy_hash
  OR NOT EXISTS(SELECT 1 FROM goals goal WHERE goal.project_ref=envelope.subject_ref)))
 OR (envelope.scope<>'goal' AND NOT EXISTS(SELECT 1 FROM budget_envelopes goal_envelope
      WHERE goal_envelope.scope='goal' AND goal_envelope.policy_hash=envelope.policy_hash
       AND goal_envelope.revision=envelope.revision))`},
	{"sqlite.recovery_v15_budget_reservation_invalid", `
SELECT COUNT(*) FROM budget_reservations reservation
LEFT JOIN effect_intents intent ON intent.ref=reservation.effect_intent_ref
LEFT JOIN outbox action ON action.ref=reservation.action_ref
WHERE intent.ref IS NULL OR action.kind<>'launch_agent' OR reservation.action_ref<>intent.action_ref
 OR reservation.demand_ref<>intent.demand_ref OR reservation.project_ref<>intent.project_ref
 OR reservation.goal_ref<>intent.goal_ref OR reservation.work_item_ref<>intent.work_item_ref
 OR reservation.execution_ref<>intent.execution_ref OR reservation.plan_generation<>intent.plan_generation
 OR reservation.app_spec_generation<>intent.app_spec_generation OR reservation.work_item_generation<>action.work_item_generation
 OR reservation.fence>action.fence OR reservation.spec_hash<>intent.spec_hash OR reservation.policy_hash<>intent.policy_hash
 OR (reservation.fence=action.fence AND action.claimed_until IS NOT NULL AND
    (reservation.reserved_at<action.available_at OR reservation.reserved_at>=action.claimed_until))
 OR reservation.tokens<>intent.demand_tokens OR reservation.money_micros<>intent.demand_money_micros
 OR reservation.currency<>intent.demand_currency OR reservation.active_time_ns<>intent.demand_active_time_ns
 OR reservation.process_slots<>intent.demand_process_slots OR reservation.disk_bytes<>intent.demand_disk_bytes
 OR reservation.reserved_at<intent.created_at OR (SELECT COUNT(*) FROM budget_envelopes envelope
    WHERE envelope.policy_hash=reservation.policy_hash AND envelope.revision=intent.policy_revision
     AND (envelope.scope='deployment'
      OR (envelope.scope='project' AND envelope.subject_ref=reservation.project_ref)
      OR (envelope.scope='goal' AND envelope.subject_ref=reservation.goal_ref))
    AND envelope.tokens>=reservation.tokens AND envelope.money_micros>=reservation.money_micros
    AND envelope.active_time_ns>=reservation.active_time_ns AND envelope.process_slots>=reservation.process_slots
     AND envelope.disk_bytes>=reservation.disk_bytes AND (envelope.currency='' OR envelope.currency=reservation.currency))<>3`},
	{"sqlite.recovery_v15_budget_reservation_cardinality_invalid", `
SELECT COUNT(*) FROM (
 SELECT reservation.action_ref FROM budget_reservations reservation
 LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
 GROUP BY reservation.action_ref
 HAVING SUM(CASE WHEN settlement.ref IS NULL THEN 1 ELSE 0 END)>1
)`},
	{"sqlite.recovery_v15_budget_settlement_invalid", `
SELECT COUNT(*) FROM budget_settlements settlement JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref
WHERE settlement.settled_at<reservation.reserved_at OR settlement.reserved_tokens<>reservation.tokens
 OR settlement.reserved_money_micros<>reservation.money_micros OR settlement.reserved_currency<>reservation.currency
 OR settlement.reserved_active_time_ns<>reservation.active_time_ns OR settlement.reserved_process_slots<>reservation.process_slots
 OR settlement.reserved_disk_bytes<>reservation.disk_bytes`},
	{"sqlite.recovery_v15_effect_attempt_invalid", `
SELECT COUNT(*) FROM effect_attempts attempt
LEFT JOIN effect_intents intent ON intent.ref=attempt.intent_ref
LEFT JOIN effect_approvals approval ON approval.ref=attempt.approval_ref
LEFT JOIN outbox action ON action.ref=attempt.action_ref
WHERE intent.ref IS NULL OR approval.ref IS NULL OR action.ref IS NULL OR approval.decision<>'approved'
 OR approval.intent_ref<>intent.ref OR approval.intent_digest<>intent.digest OR attempt.intent_digest<>intent.digest
 OR attempt.project_ref<>intent.project_ref OR attempt.goal_ref<>intent.goal_ref OR attempt.work_item_ref<>intent.work_item_ref
 OR attempt.execution_ref<>intent.execution_ref OR attempt.plan_generation<>intent.plan_generation
 OR attempt.app_spec_generation<>intent.app_spec_generation OR attempt.spec_hash<>intent.spec_hash
 OR attempt.actor_ref<>intent.actor_ref OR attempt.action_ref<>intent.action_ref OR attempt.action_fence>action.fence
 OR attempt.idempotency_key<>intent.idempotency_key OR attempt.started_at<intent.created_at
 OR attempt.started_at<approval.decided_at OR attempt.started_at>=action.claimed_until
 OR (intent.kind='agent_launch' AND NOT EXISTS(SELECT 1 FROM budget_reservations reservation
    JOIN events event ON event.goal_ref=attempt.goal_ref AND event.work_item_ref=attempt.work_item_ref
     AND event.execution_ref=attempt.execution_ref AND event.kind='execution.dispatching'
    WHERE reservation.action_ref=attempt.action_ref AND reservation.fence<=attempt.action_fence
     AND event.occurred_at>=reservation.reserved_at AND event.occurred_at<=attempt.started_at))
 OR (intent.kind='agent_stop' AND NOT EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=attempt.goal_ref
    AND event.work_item_ref=attempt.work_item_ref AND event.execution_ref=attempt.execution_ref
    AND event.kind='execution.accepted' AND event.occurred_at<=attempt.started_at))
 OR (intent.kind='prepare_workspace' AND NOT EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=attempt.goal_ref
    AND event.work_item_ref=attempt.work_item_ref AND event.execution_ref=attempt.execution_ref
    AND event.kind='execution.queued' AND event.occurred_at<=attempt.started_at))
 OR (intent.kind='commit_change' AND NOT EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=attempt.goal_ref
    AND event.work_item_ref=attempt.work_item_ref AND event.execution_ref=attempt.execution_ref
    AND event.kind='execution.output_ready' AND event.occurred_at<=attempt.started_at))
 OR (intent.kind='attest_test' AND NOT EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=attempt.goal_ref
    AND event.work_item_ref=attempt.work_item_ref AND event.execution_ref=attempt.execution_ref
    AND event.kind='change.committed' AND event.occurred_at<=attempt.started_at))
 OR (intent.kind='integrate_change' AND NOT EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=attempt.goal_ref
    AND event.work_item_ref=attempt.work_item_ref AND event.execution_ref=attempt.execution_ref
    AND event.kind='test_attestation.passed' AND event.occurred_at<=attempt.started_at))
 OR (approval.source='explicit_decision' AND attempt.started_at>=approval.expires_at)
 OR approval.ref IS NOT (SELECT latest.ref FROM effect_approvals latest WHERE latest.intent_ref=intent.ref
    AND latest.decided_at<=attempt.started_at ORDER BY latest.decided_at DESC,
    CASE latest.source WHEN 'explicit_decision' THEN 0 ELSE 1 END,
    CASE latest.decision WHEN 'denied' THEN 0 ELSE 1 END, latest.ref DESC LIMIT 1)`},
	{"sqlite.recovery_v15_unknown_applied_repeated", `
SELECT COUNT(*) FROM effect_attempts later
JOIN effect_attempts prior ON prior.action_ref=later.action_ref AND prior.intent_ref=later.intent_ref
 AND prior.action_fence<later.action_fence
JOIN effect_intents intent ON intent.ref=prior.intent_ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=prior.ref
WHERE receipt.ref IS NULL AND intent.kind IN ('agent_launch','attest_test')
 AND (intent.kind<>'agent_launch' OR NOT EXISTS(
  SELECT 1 FROM budget_reservations reservation
  JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
  WHERE reservation.action_ref=prior.action_ref AND reservation.effect_intent_ref=prior.intent_ref
   AND reservation.fence<=prior.action_fence AND reservation.reserved_at<=prior.started_at
   AND (SELECT COUNT(*) FROM budget_reservations candidate
       WHERE candidate.action_ref=prior.action_ref AND candidate.effect_intent_ref=prior.intent_ref
        AND candidate.fence<=prior.action_fence AND candidate.reserved_at<=prior.started_at
        AND NOT EXISTS(SELECT 1 FROM budget_settlements candidate_settlement
            WHERE candidate_settlement.reservation_ref=candidate.ref
             AND candidate_settlement.settled_at<prior.started_at))=1
   AND NOT EXISTS(SELECT 1 FROM effect_attempts peer
       WHERE peer.ref<>prior.ref AND peer.action_ref=prior.action_ref AND peer.intent_ref=prior.intent_ref
        AND reservation.fence<=peer.action_fence AND reservation.reserved_at<=peer.started_at
        AND NOT EXISTS(SELECT 1 FROM budget_settlements peer_settlement
            WHERE peer_settlement.reservation_ref=reservation.ref
             AND peer_settlement.settled_at<peer.started_at))
   AND settlement.settled_at>prior.started_at
   AND settlement.observed_known=31 AND settlement.observed_quality='exact'
   AND settlement.observed_tokens=0 AND settlement.observed_money_micros=0
   AND settlement.observed_currency=reservation.currency AND settlement.observed_active_time_ns=0
   AND settlement.observed_process_slots=0 AND settlement.observed_disk_bytes=0
   AND settlement.charged_tokens=0 AND settlement.charged_money_micros=0
   AND settlement.charged_currency=reservation.currency AND settlement.charged_active_time_ns=0
   AND settlement.charged_process_slots=0 AND settlement.charged_disk_bytes=0
   AND settlement.released_tokens=reservation.tokens
   AND settlement.released_money_micros=reservation.money_micros
   AND settlement.released_currency=reservation.currency
   AND settlement.released_active_time_ns=reservation.active_time_ns
   AND settlement.released_process_slots=reservation.process_slots
   AND settlement.released_disk_bytes=reservation.disk_bytes
   AND settlement.overrun_tokens=0 AND settlement.overrun_money_micros=0
   AND settlement.overrun_currency=reservation.currency AND settlement.overrun_active_time_ns=0
   AND settlement.overrun_process_slots=0 AND settlement.overrun_disk_bytes=0
 ))`},
	{"sqlite.recovery_v15_effect_receipt_invalid", `
SELECT COUNT(*) FROM effect_receipts receipt LEFT JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
LEFT JOIN effect_intents intent ON intent.ref=receipt.intent_ref
LEFT JOIN outbox action ON action.ref=receipt.action_ref
WHERE attempt.ref IS NULL OR intent.ref IS NULL OR action.ref IS NULL
 OR receipt.intent_ref<>attempt.intent_ref OR receipt.intent_digest<>attempt.intent_digest
 OR receipt.approval_ref<>attempt.approval_ref OR receipt.project_ref<>attempt.project_ref
 OR receipt.goal_ref<>attempt.goal_ref OR receipt.work_item_ref<>attempt.work_item_ref
 OR receipt.execution_ref<>attempt.execution_ref OR receipt.plan_generation<>attempt.plan_generation
 OR receipt.app_spec_generation<>attempt.app_spec_generation OR receipt.spec_hash<>attempt.spec_hash
 OR receipt.actor_ref<>attempt.actor_ref OR receipt.action_ref<>attempt.action_ref
 OR receipt.action_fence<>attempt.action_fence OR receipt.idempotency_key<>attempt.idempotency_key
 OR receipt.confirmed_at<attempt.started_at
 OR (receipt.confirmed_at>=action.claimed_until
     AND intent.kind NOT IN ('agent_quiesce','agent_environment_preserve','agent_environment_close'))
 OR (intent.kind='agent_launch' AND receipt.status<>'accepted')
 OR (intent.kind='agent_quiesce' AND receipt.status<>'quiesced')
 OR (intent.kind='agent_environment_preserve' AND receipt.status<>'preserved')
 OR (intent.kind='agent_environment_close' AND receipt.status<>'closed')
 OR (intent.kind='agent_stop' AND receipt.status NOT IN ('stopped','already_stopped','already_completed','already_failed'))
 OR (intent.kind='prepare_workspace' AND receipt.status<>'prepared')
 OR (intent.kind='commit_change' AND receipt.status<>'committed')
 OR (intent.kind='attest_test' AND receipt.status NOT IN ('attested_passed','attested_failed'))
 OR (intent.kind='integrate_change' AND receipt.status NOT IN ('integrated','conflicted','stale'))`},
	{"sqlite.recovery_v15_effect_binding_invalid", `
SELECT (SELECT COUNT(*) FROM executions execution LEFT JOIN effect_intents intent ON intent.ref=execution.effect_intent_ref
 LEFT JOIN budget_reservations reservation ON reservation.ref=execution.budget_reservation_ref
 LEFT JOIN effect_receipts receipt ON receipt.ref=execution.launch_receipt_ref
 WHERE (execution.governance_version=0 AND (reservation.ref IS NOT NULL OR intent.ref IS NOT NULL OR receipt.ref IS NOT NULL))
 OR (execution.governance_version=1 AND (intent.ref IS NULL OR reservation.ref IS NULL
  OR intent.execution_ref<>execution.ref OR reservation.execution_ref<>execution.ref
  OR reservation.effect_intent_ref<>intent.ref OR (receipt.ref IS NOT NULL AND receipt.intent_ref<>intent.ref)
  OR (execution.state IN ('running','succeeded','stopped','canceled') AND receipt.ref IS NULL))))
+(SELECT COUNT(*) FROM action_consumption_receipts consumed JOIN outbox action ON action.ref=consumed.action_ref
 LEFT JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
 WHERE consumed.governance_version<>action.governance_version
 OR (consumed.consumed_at>=action.claimed_until
     AND consumed.kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment'))
 OR (receipt.ref IS NOT NULL AND
  (receipt.action_ref<>consumed.action_ref OR receipt.action_fence<>consumed.fence))
 OR (consumed.governance_version=1 AND consumed.kind='launch_agent' AND consumed.outcome='completed'
  AND consumed.error_code='' AND receipt.ref IS NULL)
 OR (consumed.governance_version=1 AND consumed.kind IN (
      'quiesce_agent','preserve_agent_environment','close_agent_environment',
      'prepare_workspace','commit_change','attest_test','integrate_change')
  AND consumed.outcome='completed' AND consumed.error_code='' AND receipt.ref IS NULL)
 OR (consumed.governance_version=1 AND consumed.kind='stop_agent' AND consumed.outcome='completed'
  AND consumed.error_code='' AND receipt.ref IS NULL AND EXISTS(SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=consumed.action_ref AND attempt.action_fence=consumed.fence)))
+(SELECT COUNT(*) FROM effect_receipts receipt LEFT JOIN action_consumption_receipts consumed
 ON consumed.effect_receipt_ref=receipt.ref AND consumed.action_ref=receipt.action_ref AND consumed.fence=receipt.action_fence
 WHERE consumed.action_ref IS NULL)`},
	{"sqlite.recovery_v15_terminal_budget_invalid", `
SELECT COUNT(*) FROM executions execution LEFT JOIN budget_settlements settlement
 ON settlement.reservation_ref=execution.budget_reservation_ref WHERE execution.governance_version=1 AND
 (execution.state IN ('succeeded','failed','canceled','stopped') AND settlement.ref IS NULL
  OR execution.state IN ('queued','dispatching','running') AND settlement.ref IS NOT NULL)`},
	{"sqlite.recovery_v15_fairness_invalid", `
SELECT COUNT(*) FROM fairness_cursors cursor WHERE (cursor.scope='project' AND NOT EXISTS
 (SELECT 1 FROM goals goal WHERE goal.project_ref=cursor.subject_ref)) OR (cursor.scope='goal' AND NOT EXISTS
 (SELECT 1 FROM goals goal WHERE goal.ref=cursor.subject_ref))`},
}

func validateRecoveryV15IntentSemantics(ctx context.Context, source queryer) error {
	refs, err := readSingleColumn(ctx, source, `SELECT ref FROM effect_intents ORDER BY ref`)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		intent, err := readEffectIntent(ctx, source, ref)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v15_effect_intent_semantic_invalid:%s", ref)
		}
		var linked int
		if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM outbox action WHERE action.ref=? AND action.effect_intent_ref=?
 AND action.kind=? AND action.goal_ref=? AND action.work_item_ref=? AND action.execution_ref=?
 AND action.plan_generation=?`, intent.ActionRef, intent.Ref, string(intent.ActionKind),
			intent.Subject.GoalRef.String(), intent.Subject.WorkItemRef.String(),
			intent.Subject.ExecutionRef.String(), int64(intent.Subject.PlanGeneration),
		).Scan(&linked); err != nil || linked != 1 {
			return fmt.Errorf("sqlite.recovery_v15_effect_intent_action_invalid:%s", ref)
		}
	}
	return nil
}

func validateRecoveryV15SettlementSemantics(ctx context.Context, source queryer) error {
	return validateRecoveryV15GoalLedger(ctx, source, `
SELECT DISTINCT reservation.goal_ref FROM budget_settlements settlement
JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref
ORDER BY reservation.goal_ref`, "sqlite.recovery_v15_budget_settlement_semantic_invalid",
		func(goalRef string) error {
			if _, err := readBudgetSettlementsForGoal(ctx, source, goalRef); err != nil {
				return err
			}
			return nil
		})
}

func validateRecoveryV15ReceiptSemantics(ctx context.Context, source queryer) error {
	return validateRecoveryV15GoalLedger(ctx, source,
		`SELECT DISTINCT goal_ref FROM effect_receipts ORDER BY goal_ref`,
		"sqlite.recovery_v15_effect_receipt_semantic_invalid", func(goalRef string) error {
			_, err := readEffectReceiptsForGoal(ctx, source, goalRef)
			return err
		})
}

func validateRecoveryV15GoalLedger(
	ctx context.Context, source queryer, query, code string, load func(string) error,
) error {
	goals, err := readSingleColumn(ctx, source, query)
	if err != nil {
		return err
	}
	for _, goalRef := range goals {
		if err := load(goalRef); err != nil {
			return fmt.Errorf("%s:%s", code, goalRef)
		}
	}
	return nil
}

func validateRecoveryV15ApprovalSemantics(ctx context.Context, source queryer) error {
	var duplicateRequests int
	if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM (SELECT decided_by_ref,request_ref FROM effect_approvals WHERE source='explicit_decision'
GROUP BY decided_by_ref,request_ref HAVING COUNT(*)>1)`).Scan(&duplicateRequests); err != nil {
		return err
	}
	if duplicateRequests != 0 {
		return errors.New("sqlite.recovery_v15_effect_approval_request_duplicate")
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,intent_ref FROM effect_approvals ORDER BY ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type pair struct{ approval, intent string }
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.approval, &p.intent); err != nil {
			return err
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pairs {
		intent, err := readEffectIntent(ctx, source, p.intent)
		if err != nil {
			return err
		}
		approval, found, err := readEffectApprovalByRef(ctx, source, p.approval)
		if err != nil {
			return err
		}
		if !found || application.ValidateEffectApproval(intent, approval) != nil ||
			(approval.Source == application.EffectApprovalSourceExplicitDecision && application.ValidatePersistedEffectApproval(approval) != nil) {
			return fmt.Errorf("sqlite.recovery_v15_effect_approval_invalid:%s", p.approval)
		}
	}
	return nil
}
