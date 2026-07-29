package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func validateRecoveryV21PostArtifactMailbox(ctx context.Context, tx *sql.Tx) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM outbox action
LEFT JOIN goals goal ON goal.ref=action.goal_ref
LEFT JOIN work_items item
 ON item.goal_ref=action.goal_ref AND item.ref=action.work_item_ref
LEFT JOIN executions execution
 ON execution.goal_ref=action.goal_ref AND execution.work_item_ref=action.work_item_ref
 AND execution.ref=action.execution_ref
WHERE action.kind='admit_mailbox' AND (
 action.ref<>'action:admit-mailbox:'||action.execution_ref OR
 goal.ref IS NULL OR
 (action.completed_at IS NULL AND goal.state<>'running') OR
 item.ref IS NULL OR item.state<>'succeeded' OR item.handoff_required<>1 OR
 item.execution_ref<>action.execution_ref OR item.revision<>action.work_item_generation OR
 execution.ref IS NULL OR execution.state<>'succeeded' OR
 execution.plan_generation<>action.plan_generation OR
 action.mailbox_message_ref IS NOT NULL OR action.control_ref IS NOT NULL OR
 action.change_ref<>'' OR action.expected_target_oid<>'' OR
 action.admission_request_ref<>'' OR action.admission_request_fingerprint<>'' OR
 action.governance_version<>0 OR action.effect_intent_ref IS NOT NULL OR
 (action.completed_at IS NOT NULL AND action.quarantined_at IS NULL AND NOT EXISTS (
  SELECT 1
  FROM mailbox_admission_receipts admission
  JOIN mailbox_envelopes envelope ON envelope.ref=admission.mailbox_message_ref
  WHERE envelope.goal_ref=action.goal_ref
   AND envelope.child_work_item_ref=action.work_item_ref
   AND envelope.source_execution_ref=action.execution_ref
 )))
`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v21_post_artifact_mailbox_invalid")
	}
	if err := validateRecoveryV21ExecutionSessionRevocations(ctx, tx); err != nil {
		return err
	}
	return validateRecoveryV21ExecutionAuthorizationScope(ctx, tx)
}

func validateRecoveryV21ExecutionSessionRevocations(ctx context.Context, tx *sql.Tx) error {
	var invalid, missing int
	if err := tx.QueryRowContext(ctx, `
WITH active_admission AS (
 SELECT goal_ref,work_item_ref,execution_ref FROM outbox
 WHERE kind='admit_mailbox' AND completed_at IS NULL
  AND retired_at IS NULL AND quarantined_at IS NULL
)
SELECT
(SELECT COUNT(*) FROM outbox action
LEFT JOIN work_items item
 ON item.goal_ref=action.goal_ref AND item.ref=action.work_item_ref
LEFT JOIN executions execution
 ON execution.goal_ref=action.goal_ref AND execution.work_item_ref=action.work_item_ref
 AND execution.ref=action.execution_ref
LEFT JOIN work_item_fences item_fence
 ON item_fence.goal_ref=action.goal_ref AND item_fence.work_item_ref=action.work_item_ref
LEFT JOIN action_consumption_receipts receipt ON receipt.action_ref=action.ref
LEFT JOIN active_admission admission
 ON admission.goal_ref=action.goal_ref AND admission.work_item_ref=action.work_item_ref
 AND admission.execution_ref=action.execution_ref
WHERE action.kind='revoke_execution_session' AND (
 action.ref<>'action:revoke-execution-session:'||action.execution_ref OR item.ref IS NULL OR
 execution.ref IS NULL OR execution.execution_session_ref='' OR
 execution.state NOT IN ('succeeded','failed','canceled','stopped') OR execution.finished_at IS NULL OR
 action.plan_generation<>execution.plan_generation OR action.work_item_generation>item.revision OR
 (action.work_item_generation<item.revision AND
  (action.completed_at IS NULL OR receipt.action_ref IS NULL)) OR
 action.mailbox_message_ref IS NOT NULL OR action.control_ref IS NOT NULL OR action.change_ref<>'' OR
 action.expected_target_oid<>'' OR action.admission_request_ref<>'' OR action.admission_request_fingerprint<>'' OR
 action.governance_version<>0 OR action.effect_intent_ref IS NOT NULL OR action.review_gate_digest<>'' OR
 action.council_subject_digest<>'' OR action.council_resolution_kind<>'' OR action.council_decision_ref IS NOT NULL OR
 action.council_decision_digest IS NOT NULL OR action.council_skip_ref IS NOT NULL OR action.council_skip_digest IS NOT NULL OR
 action.retired_at IS NOT NULL OR action.quarantined_at IS NOT NULL OR
 action.last_error_code NOT IN ('','application.execution_session_revoke_unavailable') OR
 (action.claim_token IS NULL)<>(action.claimed_by IS NULL) OR (action.claim_token IS NULL)<>(action.claimed_until IS NULL) OR
 (action.claim_token IS NULL AND ((action.delivery_attempt=0)<>(action.fence=0) OR action.delivery_attempt<0 OR action.fence<0)) OR
 (action.fence>0 AND (item_fence.fence IS NULL OR item_fence.fence<action.fence)) OR
 (action.claim_token IS NOT NULL AND (length(trim(action.claim_token))=0 OR length(trim(action.claimed_by))=0 OR
   action.delivery_attempt<=0 OR action.fence<=0 OR
   (action.completed_at IS NULL AND item_fence.fence<>action.fence))) OR
 admission.execution_ref IS NOT NULL OR
 (action.completed_at IS NULL AND receipt.action_ref IS NOT NULL) OR
 (action.completed_at IS NOT NULL AND (
   receipt.action_ref IS NULL OR action.last_error_code<>'' OR receipt.governance_version<>0 OR receipt.kind<>action.kind OR
   receipt.goal_ref<>action.goal_ref OR receipt.work_item_ref<>action.work_item_ref OR receipt.execution_ref<>action.execution_ref OR
   receipt.plan_generation<>action.plan_generation OR receipt.work_item_generation<>action.work_item_generation OR
   receipt.fence<>action.fence OR receipt.delivery_attempt<>action.delivery_attempt OR
   receipt.claim_token<>action.claim_token OR receipt.worker_ref<>action.claimed_by OR receipt.outcome<>'completed' OR
   receipt.error_code<>'' OR receipt.change_ref<>'' OR receipt.mailbox_message_ref IS NOT NULL OR
   receipt.consumed_at<>action.completed_at OR receipt.consumed_at>=action.claimed_until OR
   receipt.effect_receipt_ref IS NOT NULL OR receipt.legacy_effect_status IS NOT NULL OR
   receipt.legacy_effect_confirmed_at IS NOT NULL
 ))
)),
(SELECT COUNT(*) FROM executions execution
 LEFT JOIN active_admission admission
  ON admission.goal_ref=execution.goal_ref AND admission.work_item_ref=execution.work_item_ref
  AND admission.execution_ref=execution.ref
 WHERE execution.execution_session_ref<>'' AND execution.state IN ('succeeded','failed','canceled','stopped')
 AND NOT EXISTS (
  SELECT 1 FROM outbox action WHERE action.kind='revoke_execution_session'
   AND action.goal_ref=execution.goal_ref AND action.work_item_ref=execution.work_item_ref
   AND action.execution_ref=execution.ref
 )
 AND admission.execution_ref IS NULL)
`).Scan(&invalid, &missing); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v21_execution_session_revocation_invalid")
	}
	if missing != 0 {
		return errors.New("sqlite.recovery_v21_execution_session_revocation_missing")
	}
	return nil
}

type recoveryV21ExecutionAuthorization struct {
	principal   identity.Principal
	projectRef  goal.ProjectRef
	permission  identity.Permission
	resourceRef string
}

func validateRecoveryV21ExecutionAuthorizationScope(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT receipt.principal_ref,principal.actor_ref,principal.kind,
       principal.authentication_method,receipt.project_ref,receipt.permission,receipt.resource_ref
FROM authorization_receipts receipt
JOIN principals principal ON principal.ref=receipt.principal_ref
WHERE receipt.reason_code=?`,
		authorizationReasonExecutionBound,
	)
	if err != nil {
		return err
	}
	var authorizations []recoveryV21ExecutionAuthorization
	for rows.Next() {
		var authorization recoveryV21ExecutionAuthorization
		var principalRef, actorRef, kind, projectRef string
		if err := rows.Scan(&principalRef, &actorRef, &kind, &authorization.principal.Method,
			&projectRef, &authorization.permission, &authorization.resourceRef); err != nil {
			_ = rows.Close()
			return err
		}
		var parseErr error
		authorization.principal.Ref, parseErr = identity.NewPrincipalRef(principalRef)
		if parseErr == nil {
			authorization.principal.ActorRef, parseErr = goal.NewActorRef(actorRef)
		}
		authorization.principal.Kind = identity.PrincipalKind(kind)
		if parseErr == nil {
			authorization.projectRef, parseErr = goal.NewProjectRef(projectRef)
		}
		if parseErr != nil || identity.ValidatePrincipal(authorization.principal) != nil {
			_ = rows.Close()
			return errors.New("sqlite.recovery_v21_execution_authority_invalid")
		}
		authorizations = append(authorizations, authorization)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, authorization := range authorizations {
		valid, err := validateRecoveryV21ExecutionAuthorization(ctx, tx, authorization)
		if err != nil {
			return err
		}
		if !valid {
			return errors.New("sqlite.recovery_v21_execution_authorization_scope_invalid")
		}
	}
	return nil
}

func validateRecoveryV21ExecutionAuthorization(
	ctx context.Context,
	tx *sql.Tx,
	authorization recoveryV21ExecutionAuthorization,
) (bool, error) {
	match, found, err := findExecutionAuthority(
		ctx, tx, goal.ExecutionRef{}, authorization.principal,
	)
	if err != nil || !found {
		return false, err
	}
	if match.authority.Request.ProjectRef != authorization.projectRef {
		return false, nil
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "recovery.execution.scope", Principal: authorization.principal,
		ProjectRef: authorization.projectRef, Permission: authorization.permission,
		ResourceRef: authorization.resourceRef, RequestedAt: time.Unix(1, 0),
	})
	if err != nil {
		return false, err
	}
	return executionAuthorizationScope(ctx, tx, request, match.authority)
}
