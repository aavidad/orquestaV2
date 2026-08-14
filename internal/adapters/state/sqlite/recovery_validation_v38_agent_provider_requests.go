package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const recoveryV38AgentProviderRequestInvalid = "sqlite.recovery_v38_agent_provider_request_invalid"

func validateRecoveryV38AgentProviderRequests(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT execution_ref,action_fence,stage
FROM agent_provider_requests
ORDER BY execution_ref,action_fence,stage`)
	if err != nil {
		return invalidRecoveryV38AgentProviderRequest(err)
	}
	var keys []ports.AgentProviderRequestKey
	for rows.Next() {
		var executionRef, stage string
		var actionFence int64
		if err := rows.Scan(&executionRef, &actionFence, &stage); err != nil {
			_ = rows.Close()
			return invalidRecoveryV38AgentProviderRequest(err)
		}
		execution, refErr := goal.NewExecutionRef(executionRef)
		key := ports.AgentProviderRequestKey{
			ExecutionRef: execution, ActionFence: uint64(actionFence), Stage: ports.AgentProviderRequestStage(stage),
		}
		if refErr != nil || actionFence <= 0 || ports.ValidateAgentProviderRequestKey(key) != nil {
			_ = rows.Close()
			return invalidRecoveryV38AgentProviderRequest(nil)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return invalidRecoveryV38AgentProviderRequest(err)
	}
	if err := rows.Err(); err != nil {
		return invalidRecoveryV38AgentProviderRequest(err)
	}
	for _, key := range keys {
		request, found, err := readAgentProviderRequest(ctx, tx, key)
		if err != nil || !found {
			return invalidRecoveryV38AgentProviderRequest(err)
		}
		if err := validateRecoveryV38AgentProviderRequestCausality(ctx, tx, request); err != nil {
			return err
		}
	}
	return nil
}

func validateRecoveryV38AgentProviderRequestCausality(
	ctx context.Context,
	tx *sql.Tx,
	request ports.AgentProviderRequest,
) error {
	var attemptExecution, intentKind, intentExecution, executionProvider, executionExternal string
	var attemptFence int64
	var receiptExternal sql.NullString
	if err := tx.QueryRowContext(ctx, `
SELECT attempt.execution_ref,attempt.action_fence,intent.kind,intent.execution_ref,
 execution.provider_ref,execution.external_ref,receipt.external_ref
FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN executions execution ON execution.ref=attempt.execution_ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
WHERE attempt.ref=?`, request.EffectAttemptRef).Scan(
		&attemptExecution, &attemptFence, &intentKind, &intentExecution,
		&executionProvider, &executionExternal, &receiptExternal,
	); err != nil {
		return invalidRecoveryV38AgentProviderRequest(err)
	}
	if attemptFence <= 0 || attemptExecution != request.Key.ExecutionRef.String() ||
		uint64(attemptFence) != request.Key.ActionFence || intentKind != "agent_launch" ||
		intentExecution != attemptExecution ||
		executionProvider != "" && executionProvider != request.ProviderRef {
		return invalidRecoveryV38AgentProviderRequest(nil)
	}
	launchKey := request.Key
	launchKey.Stage = ports.AgentProviderRequestLaunch
	launch, found, err := readAgentProviderRequest(ctx, tx, launchKey)
	if err != nil || !found {
		return invalidRecoveryV38AgentProviderRequest(err)
	}
	if launch.EffectAttemptRef != request.EffectAttemptRef || launch.ProviderRef != request.ProviderRef {
		return invalidRecoveryV38AgentProviderRequest(nil)
	}
	if request.Key.Stage != ports.AgentProviderRequestLaunch &&
		(launch.LaunchBindingRef != request.TargetRef ||
			launch.LaunchBindingRevision != request.ExpectedRevision) {
		return invalidRecoveryV38AgentProviderRequest(nil)
	}
	if request.Key.Stage == ports.AgentProviderRequestSessionInput {
		startKey := request.Key
		startKey.Stage = ports.AgentProviderRequestSessionStart
		started, startFound, startErr := readAgentProviderRequest(ctx, tx, startKey)
		if startErr != nil || !startFound || started.EffectAttemptRef != request.EffectAttemptRef ||
			started.ProviderRef != request.ProviderRef || started.TargetRef != request.TargetRef ||
			started.ExpectedRevision != request.ExpectedRevision {
			return invalidRecoveryV38AgentProviderRequest(startErr)
		}
	}
	if receiptExternal.Valid &&
		(launch.LaunchBindingRef == "" || receiptExternal.String != launch.LaunchBindingRef) {
		return invalidRecoveryV38AgentProviderRequest(nil)
	}
	if executionExternal != "" &&
		(launch.LaunchBindingRef == "" || executionExternal != launch.LaunchBindingRef) {
		return invalidRecoveryV38AgentProviderRequest(nil)
	}
	return nil
}

func invalidRecoveryV38AgentProviderRequest(cause error) error {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}
	return invalid(errors.Join(cause, errors.New(recoveryV38AgentProviderRequestInvalid)))
}
