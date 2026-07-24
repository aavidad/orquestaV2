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
	return validateRecoveryV21ExecutionAuthorizationScope(ctx, tx)
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
