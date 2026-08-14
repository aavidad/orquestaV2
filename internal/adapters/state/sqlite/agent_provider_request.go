package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"math"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const agentProviderRequestSelect = `SELECT
 execution_ref,action_fence,stage,effect_attempt_ref,provider_ref,idempotency_key,
 target_ref,expected_revision,body,body_sha256,launch_binding_ref,launch_binding_revision
FROM agent_provider_requests
WHERE execution_ref=? AND action_fence=? AND stage=?`

var _ ports.AgentProviderRequestJournal = (*Repository)(nil)

func (repository *Repository) RecordAgentProviderRequest(
	ctx context.Context,
	request ports.AgentProviderRequest,
) (ports.AgentProviderRequest, error) {
	if ctx == nil || request.Key.ActionFence > math.MaxInt64 ||
		request.ExpectedRevision > math.MaxInt64 ||
		ports.ValidatePreparedAgentProviderRequest(request) != nil {
		return ports.AgentProviderRequest{}, invalid(errors.New("sqlite.agent_provider_request_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderRequest{}, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
	}
	defer transaction.Rollback()

	persisted, found, err := readAgentProviderRequest(ctx, transaction, request.Key)
	if err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if found {
		if !sameAgentProviderRequestExceptLaunchBinding(persisted, request) {
			return ports.AgentProviderRequest{}, conflict(errors.New("sqlite.agent_provider_request_conflict"))
		}
		if err := commit(transaction); err != nil {
			return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
		}
		return ports.CloneAgentProviderRequest(persisted), nil
	}

	result, err := transaction.ExecContext(ctx, `
INSERT INTO agent_provider_requests(
 execution_ref,action_fence,stage,effect_attempt_ref,provider_ref,idempotency_key,
 target_ref,expected_revision,body,body_sha256
)
SELECT ?,?,?,?,?,?,?,?,?,?
FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN executions execution ON execution.ref=attempt.execution_ref
WHERE attempt.ref=? AND attempt.execution_ref=? AND attempt.action_fence=?
 AND intent.kind='agent_launch' AND intent.execution_ref=attempt.execution_ref
 AND execution.state='dispatching'
 AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
 AND (?='launch' OR EXISTS (
  SELECT 1 FROM agent_provider_requests launch
  WHERE launch.execution_ref=? AND launch.action_fence=? AND launch.stage='launch'
   AND launch.effect_attempt_ref=? AND launch.provider_ref=?
   AND launch.launch_binding_ref=? AND launch.launch_binding_revision=?
 ))
 AND (?<>'session_input' OR EXISTS (
  SELECT 1 FROM agent_provider_requests started
  WHERE started.execution_ref=? AND started.action_fence=? AND started.stage='session_start'
   AND started.effect_attempt_ref=? AND started.provider_ref=?
   AND started.target_ref=? AND started.expected_revision=?
 ))`,
		request.Key.ExecutionRef.String(), request.Key.ActionFence, string(request.Key.Stage),
		request.EffectAttemptRef, request.ProviderRef, request.IdempotencyKey,
		request.TargetRef, request.ExpectedRevision, request.Body, request.BodySHA256,
		request.EffectAttemptRef, request.Key.ExecutionRef.String(), request.Key.ActionFence,
		string(request.Key.Stage), request.Key.ExecutionRef.String(), request.Key.ActionFence,
		request.EffectAttemptRef, request.ProviderRef, request.TargetRef, request.ExpectedRevision,
		string(request.Key.Stage), request.Key.ExecutionRef.String(), request.Key.ActionFence,
		request.EffectAttemptRef, request.ProviderRef, request.TargetRef, request.ExpectedRevision,
	)
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	if rows != 1 {
		return ports.AgentProviderRequest{}, conflict(errors.New("sqlite.agent_provider_request_causality_conflict"))
	}
	persisted, found, err = readAgentProviderRequest(ctx, transaction, request.Key)
	if err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if !found || !sameAgentProviderRequestExceptLaunchBinding(persisted, request) {
		return ports.AgentProviderRequest{}, invalid(errors.New("sqlite.agent_provider_request_insert_invalid"))
	}
	if err := commit(transaction); err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
	}
	return ports.CloneAgentProviderRequest(persisted), nil
}

func (repository *Repository) BindAgentProviderLaunch(
	ctx context.Context,
	key ports.AgentProviderRequestKey,
	externalRef string,
	revision uint64,
) (ports.AgentProviderRequest, error) {
	if ctx == nil || key.Stage != ports.AgentProviderRequestLaunch || key.ActionFence > math.MaxInt64 ||
		revision == 0 || revision > math.MaxInt64 || ports.ValidateAgentProviderRequestKey(key) != nil {
		return ports.AgentProviderRequest{}, invalid(errors.New("sqlite.agent_provider_request_binding_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderRequest{}, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
	}
	defer transaction.Rollback()
	persisted, found, err := readAgentProviderRequest(ctx, transaction, key)
	if err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if !found {
		return ports.AgentProviderRequest{}, stateError(application.StateNotFound, errors.New("sqlite.agent_provider_request_not_found"))
	}
	if persisted.LaunchBindingRef != "" {
		if persisted.LaunchBindingRef != externalRef || persisted.LaunchBindingRevision != revision {
			return ports.AgentProviderRequest{}, conflict(errors.New("sqlite.agent_provider_request_binding_conflict"))
		}
		if err := commit(transaction); err != nil {
			return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
		}
		return ports.CloneAgentProviderRequest(persisted), nil
	}
	candidate := ports.CloneAgentProviderRequest(persisted)
	candidate.LaunchBindingRef, candidate.LaunchBindingRevision = externalRef, revision
	if ports.ValidateAgentProviderRequest(candidate) != nil {
		return ports.AgentProviderRequest{}, invalid(errors.New("sqlite.agent_provider_request_binding_invalid"))
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE agent_provider_requests
SET launch_binding_ref=?,launch_binding_revision=?
WHERE execution_ref=? AND action_fence=? AND stage='launch'
 AND launch_binding_ref IS NULL AND launch_binding_revision IS NULL
 AND EXISTS (
  SELECT 1 FROM effect_attempts attempt
  JOIN executions execution ON execution.ref=attempt.execution_ref
  WHERE attempt.ref=agent_provider_requests.effect_attempt_ref
   AND execution.state='dispatching'
   AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
 )`, externalRef, revision, key.ExecutionRef.String(), key.ActionFence)
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	if rows != 1 {
		return ports.AgentProviderRequest{}, conflict(errors.New("sqlite.agent_provider_request_binding_conflict"))
	}
	persisted, found, err = readAgentProviderRequest(ctx, transaction, key)
	if err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if !found || !sameAgentProviderRequest(persisted, candidate) {
		return ports.AgentProviderRequest{}, invalid(errors.New("sqlite.agent_provider_request_binding_write_invalid"))
	}
	if err := commit(transaction); err != nil {
		return ports.AgentProviderRequest{}, agentProviderRequestContextError(ctx, err)
	}
	return ports.CloneAgentProviderRequest(persisted), nil
}

func (repository *Repository) ResolveAgentProviderRequest(
	ctx context.Context,
	key ports.AgentProviderRequestKey,
) (ports.AgentProviderRequest, bool, error) {
	if ctx == nil || key.ActionFence > math.MaxInt64 || ports.ValidateAgentProviderRequestKey(key) != nil {
		return ports.AgentProviderRequest{}, false, invalid(errors.New("sqlite.agent_provider_request_key_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderRequest{}, false, err
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return ports.AgentProviderRequest{}, false, agentProviderRequestContextError(ctx, err)
	}
	defer transaction.Rollback()
	persisted, found, err := readAgentProviderRequest(ctx, transaction, key)
	if err != nil {
		return ports.AgentProviderRequest{}, false, err
	}
	if !found {
		return ports.AgentProviderRequest{}, false, nil
	}
	if err := commit(transaction); err != nil {
		return ports.AgentProviderRequest{}, false, agentProviderRequestContextError(ctx, err)
	}
	return ports.CloneAgentProviderRequest(persisted), true, nil
}

func readAgentProviderRequest(
	ctx context.Context,
	source queryer,
	key ports.AgentProviderRequestKey,
) (ports.AgentProviderRequest, bool, error) {
	var executionRef, stage, attemptRef, providerRef, idempotencyKey, targetRef, bodySHA string
	var actionFence, expectedRevision int64
	var body []byte
	var bindingRef sql.NullString
	var bindingRevision sql.NullInt64
	err := source.QueryRowContext(ctx, agentProviderRequestSelect,
		key.ExecutionRef.String(), key.ActionFence, string(key.Stage)).Scan(
		&executionRef, &actionFence, &stage, &attemptRef, &providerRef, &idempotencyKey,
		&targetRef, &expectedRevision, &body, &bodySHA, &bindingRef, &bindingRevision,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AgentProviderRequest{}, false, nil
	}
	if err != nil {
		return ports.AgentProviderRequest{}, false, agentProviderRequestDatabaseError(ctx, err)
	}
	execution, refErr := goal.NewExecutionRef(executionRef)
	if refErr != nil || actionFence <= 0 || expectedRevision < 0 ||
		bindingRevision.Valid && bindingRevision.Int64 <= 0 {
		return ports.AgentProviderRequest{}, false, invalid(errors.New("sqlite.agent_provider_request_corrupt"))
	}
	request := ports.AgentProviderRequest{
		Key:              ports.AgentProviderRequestKey{ExecutionRef: execution, ActionFence: uint64(actionFence), Stage: ports.AgentProviderRequestStage(stage)},
		EffectAttemptRef: attemptRef, ProviderRef: providerRef, IdempotencyKey: idempotencyKey,
		TargetRef: targetRef, ExpectedRevision: uint64(expectedRevision), Body: append([]byte(nil), body...), BodySHA256: bodySHA,
	}
	if bindingRef.Valid {
		request.LaunchBindingRef = bindingRef.String
	}
	if bindingRevision.Valid {
		request.LaunchBindingRevision = uint64(bindingRevision.Int64)
	}
	if request.Key != key || ports.ValidateAgentProviderRequest(request) != nil {
		return ports.AgentProviderRequest{}, false, invalid(errors.New("sqlite.agent_provider_request_corrupt"))
	}
	return ports.CloneAgentProviderRequest(request), true, nil
}

func sameAgentProviderRequest(left, right ports.AgentProviderRequest) bool {
	return left.Key == right.Key && left.EffectAttemptRef == right.EffectAttemptRef &&
		left.ProviderRef == right.ProviderRef && left.IdempotencyKey == right.IdempotencyKey &&
		left.TargetRef == right.TargetRef && left.ExpectedRevision == right.ExpectedRevision &&
		bytes.Equal(left.Body, right.Body) && left.BodySHA256 == right.BodySHA256 &&
		left.LaunchBindingRef == right.LaunchBindingRef &&
		left.LaunchBindingRevision == right.LaunchBindingRevision
}

func sameAgentProviderRequestExceptLaunchBinding(left, right ports.AgentProviderRequest) bool {
	left.LaunchBindingRef, right.LaunchBindingRef = "", ""
	left.LaunchBindingRevision, right.LaunchBindingRevision = 0, 0
	return sameAgentProviderRequest(left, right)
}

func agentProviderRequestContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
	}
	return err
}

func agentProviderRequestDatabaseError(ctx context.Context, err error) error {
	if ctx != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return mapDatabaseError(err)
}
