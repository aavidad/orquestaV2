package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
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
	principalRef string
	method       string
	projectRef   string
	permission   identity.Permission
	resourceRef  string
}

func validateRecoveryV21ExecutionAuthorizationScope(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT receipt.principal_ref,principal.authentication_method,receipt.project_ref,
       receipt.permission,receipt.resource_ref
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
		if err := rows.Scan(
			&authorization.principalRef, &authorization.method, &authorization.projectRef,
			&authorization.permission, &authorization.resourceRef,
		); err != nil {
			_ = rows.Close()
			return err
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
	rows, err := tx.QueryContext(ctx, `
SELECT execution.goal_ref,execution.work_item_ref,execution.ref,
       COALESCE(execution.replaces_execution_ref,''),execution.attempt_no,
       execution.plan_generation,execution.app_spec_generation,execution.spec_hash
FROM executions execution
JOIN goals goal ON goal.ref=execution.goal_ref
WHERE goal.project_ref=?`,
		authorization.projectRef,
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	projectRef, err := goal.NewProjectRef(authorization.projectRef)
	if err != nil {
		return false, err
	}
	for rows.Next() {
		var goalValue, itemValue, executionValue, replacementValue, specHash string
		var attempt, planGeneration, appSpecGeneration int64
		if err := rows.Scan(
			&goalValue, &itemValue, &executionValue, &replacementValue, &attempt,
			&planGeneration, &appSpecGeneration, &specHash,
		); err != nil {
			return false, err
		}
		goalRef, goalErr := goal.NewGoalRef(goalValue)
		itemRef, itemErr := goal.NewWorkItemRef(itemValue)
		executionRef, executionErr := goal.NewExecutionRef(executionValue)
		var replacement goal.ExecutionRef
		var replacementErr error
		if replacementValue != "" {
			replacement, replacementErr = goal.NewExecutionRef(replacementValue)
		}
		if goalErr != nil || itemErr != nil || executionErr != nil || replacementErr != nil ||
			attempt <= 0 || planGeneration <= 0 || appSpecGeneration <= 0 {
			return false, errors.New("sqlite.recovery_v21_execution_authority_invalid")
		}
		authority, deriveErr := application.DeriveExecutionSessionAuthority(
			ports.ExecutionSessionEnsureRequest{
				ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: itemRef,
				ExecutionRef: executionRef, ExecutionAttempt: uint64(attempt),
				ReplacesExecutionRef: replacement, PlanGeneration: goal.PlanGeneration(planGeneration),
				AppSpecGeneration: goal.AppSpecGeneration(appSpecGeneration), SpecHash: specHash,
			},
			authorization.method,
		)
		if deriveErr != nil {
			return false, deriveErr
		}
		if authority.ServicePrincipal.Ref.String() != authorization.principalRef {
			continue
		}
		scoped, scopeErr := validateRecoveryV21ExecutionResourceScope(
			ctx, tx, authorization, authority,
		)
		return scoped, scopeErr
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func validateRecoveryV21ExecutionResourceScope(
	ctx context.Context,
	tx *sql.Tx,
	authorization recoveryV21ExecutionAuthorization,
	authority ports.ExecutionSessionAuthority,
) (bool, error) {
	switch authorization.permission {
	case identity.PermissionGoalsDirect:
		return authorization.resourceRef == authority.Request.GoalRef.String(), nil
	case identity.PermissionGoalsGet:
		if authorization.resourceRef == authority.Request.GoalRef.String() {
			return true, nil
		}
		var count int
		err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM mailbox_envelopes envelope
WHERE envelope.ref=? AND envelope.project_ref=? AND envelope.goal_ref=?
 AND ((envelope.source_principal_ref=? AND envelope.source_execution_ref=?) OR
      (envelope.recipient_principal_ref=? AND envelope.recipient_execution_ref=?))`,
			authorization.resourceRef, authorization.projectRef,
			authority.Request.GoalRef.String(),
			authorization.principalRef, authority.Request.ExecutionRef.String(),
			authorization.principalRef, authority.Request.ExecutionRef.String(),
		).Scan(&count)
		return count > 0, err
	case identity.PermissionArtifactsRead:
		var count int
		err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM mailbox_artifact_refs artifact
JOIN mailbox_envelopes envelope
 ON envelope.ref=artifact.mailbox_message_ref AND envelope.goal_ref=artifact.goal_ref
WHERE artifact.artifact_ref=? AND envelope.project_ref=? AND envelope.goal_ref=?
 AND envelope.recipient_principal_ref=? AND envelope.recipient_execution_ref=?`,
			authorization.resourceRef, authorization.projectRef,
			authority.Request.GoalRef.String(), authorization.principalRef,
			authority.Request.ExecutionRef.String(),
		).Scan(&count)
		return count > 0, err
	default:
		return false, nil
	}
}
