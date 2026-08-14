package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"math"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const agentProviderStopRequestSelect = `SELECT
 execution_ref,launch_action_fence,stop_action_fence,stop_effect_attempt_ref,
 provider_ref,idempotency_key,target_ref,expected_revision,body,body_sha256
FROM agent_provider_stop_requests
WHERE execution_ref=? AND launch_action_fence=? AND stop_action_fence=?`

var _ ports.AgentProviderStopRequestJournal = (*Repository)(nil)

func (repository *Repository) RecordAgentProviderStopRequest(
	ctx context.Context,
	request ports.AgentProviderStopRequest,
) (ports.AgentProviderStopRequest, error) {
	if ctx == nil || !agentProviderStopRequestFitsSQLite(request) {
		return ports.AgentProviderStopRequest{}, invalid(errors.New("sqlite.agent_provider_stop_request_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderStopRequest{}, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.AgentProviderStopRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	defer transaction.Rollback()
	persisted, found, err := readAgentProviderStopRequest(ctx, transaction, request.Key)
	if err != nil {
		return ports.AgentProviderStopRequest{}, err
	}
	if found {
		return replayAgentProviderStopRequest(ctx, transaction, persisted, request)
	}
	persisted, err = insertAgentProviderStopRequest(ctx, transaction, request)
	if err != nil {
		return ports.AgentProviderStopRequest{}, err
	}
	if err := commit(transaction); err != nil {
		return ports.AgentProviderStopRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	return ports.CloneAgentProviderStopRequest(persisted), nil
}

func insertAgentProviderStopRequest(
	ctx context.Context,
	transaction *sql.Tx,
	request ports.AgentProviderStopRequest,
) (ports.AgentProviderStopRequest, error) {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO agent_provider_stop_requests(
 execution_ref,launch_action_fence,stop_action_fence,stop_effect_attempt_ref,
 provider_ref,idempotency_key,target_ref,expected_revision,body,body_sha256
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		request.Key.ExecutionRef.String(), request.Key.LaunchActionFence, request.Key.StopActionFence,
		request.StopEffectAttemptRef, request.ProviderRef, request.IdempotencyKey,
		request.TargetRef, request.ExpectedRevision, request.Body, request.BodySHA256,
	)
	if err != nil {
		return ports.AgentProviderStopRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	persisted, found, err := readAgentProviderStopRequest(ctx, transaction, request.Key)
	if err != nil {
		return ports.AgentProviderStopRequest{}, err
	}
	if !found || !sameAgentProviderStopRequest(persisted, request) {
		return ports.AgentProviderStopRequest{}, invalid(errors.New("sqlite.agent_provider_stop_request_insert_invalid"))
	}
	return persisted, nil
}

func replayAgentProviderStopRequest(
	ctx context.Context,
	transaction *sql.Tx,
	persisted ports.AgentProviderStopRequest,
	request ports.AgentProviderStopRequest,
) (ports.AgentProviderStopRequest, error) {
	if !sameAgentProviderStopRequest(persisted, request) {
		return ports.AgentProviderStopRequest{}, conflict(errors.New("sqlite.agent_provider_stop_request_conflict"))
	}
	if err := commit(transaction); err != nil {
		return ports.AgentProviderStopRequest{}, agentProviderRequestDatabaseError(ctx, err)
	}
	return ports.CloneAgentProviderStopRequest(persisted), nil
}

func agentProviderStopRequestFitsSQLite(request ports.AgentProviderStopRequest) bool {
	return request.Key.LaunchActionFence <= math.MaxInt64 &&
		request.Key.StopActionFence <= math.MaxInt64 && request.ExpectedRevision <= math.MaxInt64 &&
		ports.ValidateAgentProviderStopRequest(request) == nil
}

func (repository *Repository) ResolveAgentProviderStopRequest(
	ctx context.Context,
	key ports.AgentProviderStopRequestKey,
) (ports.AgentProviderStopRequest, bool, error) {
	if ctx == nil || key.LaunchActionFence > math.MaxInt64 || key.StopActionFence > math.MaxInt64 ||
		ports.ValidateAgentProviderStopRequestKey(key) != nil {
		return ports.AgentProviderStopRequest{}, false, invalid(errors.New("sqlite.agent_provider_stop_request_key_invalid"))
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderStopRequest{}, false, err
	}
	database, err := repository.database()
	if err != nil {
		return ports.AgentProviderStopRequest{}, false, agentProviderRequestDatabaseError(ctx, err)
	}
	request, found, err := readAgentProviderStopRequest(ctx, database, key)
	if err != nil || !found {
		return ports.AgentProviderStopRequest{}, found, err
	}
	return ports.CloneAgentProviderStopRequest(request), true, nil
}

func readAgentProviderStopRequest(
	ctx context.Context,
	source queryer,
	key ports.AgentProviderStopRequestKey,
) (ports.AgentProviderStopRequest, bool, error) {
	var executionRef, attemptRef, providerRef, idempotencyKey, targetRef, bodySHA string
	var launchFence, stopFence, expectedRevision int64
	var body []byte
	err := source.QueryRowContext(ctx, agentProviderStopRequestSelect,
		key.ExecutionRef.String(), key.LaunchActionFence, key.StopActionFence,
	).Scan(
		&executionRef, &launchFence, &stopFence, &attemptRef, &providerRef,
		&idempotencyKey, &targetRef, &expectedRevision, &body, &bodySHA,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AgentProviderStopRequest{}, false, nil
	}
	if err != nil {
		return ports.AgentProviderStopRequest{}, false, agentProviderRequestDatabaseError(ctx, err)
	}
	execution, refErr := goal.NewExecutionRef(executionRef)
	if refErr != nil || launchFence <= 0 || stopFence <= 0 || expectedRevision <= 0 {
		return ports.AgentProviderStopRequest{}, false, invalid(errors.New("sqlite.agent_provider_stop_request_corrupt"))
	}
	request := ports.AgentProviderStopRequest{
		Key: ports.AgentProviderStopRequestKey{
			ExecutionRef: execution, LaunchActionFence: uint64(launchFence), StopActionFence: uint64(stopFence),
		},
		StopEffectAttemptRef: attemptRef, ProviderRef: providerRef, IdempotencyKey: idempotencyKey,
		TargetRef: targetRef, ExpectedRevision: uint64(expectedRevision),
		Body: append([]byte(nil), body...), BodySHA256: bodySHA,
	}
	if request.Key != key || ports.ValidateAgentProviderStopRequest(request) != nil {
		return ports.AgentProviderStopRequest{}, false, invalid(errors.New("sqlite.agent_provider_stop_request_corrupt"))
	}
	return ports.CloneAgentProviderStopRequest(request), true, nil
}

func sameAgentProviderStopRequest(left, right ports.AgentProviderStopRequest) bool {
	return left.Key == right.Key && left.StopEffectAttemptRef == right.StopEffectAttemptRef &&
		left.ProviderRef == right.ProviderRef && left.IdempotencyKey == right.IdempotencyKey &&
		left.TargetRef == right.TargetRef && left.ExpectedRevision == right.ExpectedRevision &&
		bytes.Equal(left.Body, right.Body) && left.BodySHA256 == right.BodySHA256
}
