package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const recoveryV38AgentProviderStopRequestInvalid = "sqlite.recovery_v38_agent_provider_stop_request_invalid"

func validateRecoveryV38AgentProviderStopRequests(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT execution_ref,launch_action_fence,stop_action_fence
FROM agent_provider_stop_requests
ORDER BY execution_ref,stop_action_fence`)
	if err != nil {
		return invalidRecoveryV38AgentProviderStopRequest(err)
	}
	var keys []ports.AgentProviderStopRequestKey
	for rows.Next() {
		var executionRef string
		var launchFence, stopFence int64
		if err := rows.Scan(&executionRef, &launchFence, &stopFence); err != nil {
			_ = rows.Close()
			return invalidRecoveryV38AgentProviderStopRequest(err)
		}
		execution, refErr := goal.NewExecutionRef(executionRef)
		key := ports.AgentProviderStopRequestKey{
			ExecutionRef: execution, LaunchActionFence: uint64(launchFence), StopActionFence: uint64(stopFence),
		}
		if refErr != nil || launchFence <= 0 || stopFence <= 0 ||
			ports.ValidateAgentProviderStopRequestKey(key) != nil {
			_ = rows.Close()
			return invalidRecoveryV38AgentProviderStopRequest(nil)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return invalidRecoveryV38AgentProviderStopRequest(err)
	}
	if err := rows.Err(); err != nil {
		return invalidRecoveryV38AgentProviderStopRequest(err)
	}
	return validateRecoveryV38AgentProviderStopRequestKeys(ctx, tx, keys)
}

func validateRecoveryV38AgentProviderStopRequestKeys(ctx context.Context, tx *sql.Tx, keys []ports.AgentProviderStopRequestKey) error {
	for _, key := range keys {
		request, found, err := readAgentProviderStopRequest(ctx, tx, key)
		if err != nil || !found {
			return invalidRecoveryV38AgentProviderStopRequest(err)
		}
		if err := validateRecoveryV38AgentProviderStopRequestCausality(ctx, tx, request); err != nil {
			return err
		}
	}
	return nil
}

func validateRecoveryV38AgentProviderStopRequestCausality(
	ctx context.Context,
	tx *sql.Tx,
	request ports.AgentProviderStopRequest,
) error {
	var attemptExecution, intentKind, intentExecution string
	var executionProvider, executionExternal string
	var attemptFence int64
	var receiptStatus sql.NullString
	if err := tx.QueryRowContext(ctx, `
SELECT attempt.execution_ref,attempt.action_fence,intent.kind,intent.execution_ref,
 execution.provider_ref,execution.external_ref,receipt.status
FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN executions execution ON execution.ref=attempt.execution_ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
WHERE attempt.ref=?`, request.StopEffectAttemptRef).Scan(
		&attemptExecution, &attemptFence, &intentKind, &intentExecution,
		&executionProvider, &executionExternal, &receiptStatus,
	); err != nil {
		return invalidRecoveryV38AgentProviderStopRequest(err)
	}
	attemptIdentity := [5]string{attemptExecution, intentKind, intentExecution, executionProvider, executionExternal}
	wantIdentity := [5]string{request.Key.ExecutionRef.String(), "agent_stop",
		request.Key.ExecutionRef.String(), request.ProviderRef, request.TargetRef}
	if attemptFence <= 0 || uint64(attemptFence) != request.Key.StopActionFence || attemptIdentity != wantIdentity {
		return invalidRecoveryV38AgentProviderStopRequest(nil)
	}
	if receiptStatus.Valid {
		switch receiptStatus.String {
		case "stopped", "already_stopped", "already_completed", "already_failed":
		default:
			return invalidRecoveryV38AgentProviderStopRequest(nil)
		}
	}
	return validateRecoveryV38AgentProviderStopRequestLaunch(ctx, tx, request)
}

func validateRecoveryV38AgentProviderStopRequestLaunch(ctx context.Context, tx *sql.Tx, request ports.AgentProviderStopRequest) error {
	launchKey := ports.AgentProviderRequestKey{
		ExecutionRef: request.Key.ExecutionRef,
		ActionFence:  request.Key.LaunchActionFence,
		Stage:        ports.AgentProviderRequestLaunch,
	}
	launch, found, err := readAgentProviderRequest(ctx, tx, launchKey)
	if err != nil || !found {
		return invalidRecoveryV38AgentProviderStopRequest(err)
	}
	if launch.ProviderRef != request.ProviderRef || launch.LaunchBindingRef != request.TargetRef ||
		launch.LaunchBindingRevision != request.ExpectedRevision {
		return invalidRecoveryV38AgentProviderStopRequest(nil)
	}
	return nil
}

func invalidRecoveryV38AgentProviderStopRequest(cause error) error {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}
	return invalid(errors.Join(cause, errors.New(recoveryV38AgentProviderStopRequestInvalid)))
}
