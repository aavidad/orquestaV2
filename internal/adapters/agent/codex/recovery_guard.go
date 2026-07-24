package codex

import (
	"context"
	"errors"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (record launchRecord) recoveryRequest(receipt ports.AgentLaunchReceipt) ports.AgentLaunchRequest {
	actorRef, _ := goal.NewActorRef(record.ActorRef)
	projectRef, _ := goal.NewProjectRef(record.ProjectRef)
	sessionRef, _ := ports.NewExecutionSessionRef(record.ExecutionSessionRef)
	return ports.AgentLaunchRequest{
		SessionRef: sessionRef, ProjectRef: projectRef, ActorRef: actorRef,
		GoalRef: receipt.GoalRef, WorkItemRef: receipt.WorkItemRef, ExecutionRef: receipt.ExecutionRef,
		ExecutionAttempt: receipt.ExecutionAttempt, PlanGeneration: receipt.PlanGeneration,
		AppSpecGeneration: receipt.AppSpecGeneration,
		SpecHash:          record.SpecHash,
	}
}

func (adapter *Adapter) recoverExecutionGuards(
	ctx context.Context,
	record launchRecord,
	state *executionState,
) error {
	needsCredential := adapter.config.CredentialStore != nil
	if !needsCredential && record.ExecutionSessionRef == "" {
		return nil
	}
	if needsCredential && (record.ActorRef == "" || record.ProjectRef == "") {
		return &Error{Code: CodeCredentialUnavailable}
	}
	request := record.recoveryRequest(state.receipt)
	session, err := adapter.recoverSession(ctx, request)
	if err != nil {
		return err
	}
	defer session.destroy()
	var credentialGuard *credentials.LeakGuard
	if needsCredential {
		var guardErr error
		_, err = adapter.config.CredentialStore.Use(ctx, adapter.credentialUseRequest(request), func(secret credentials.Secret) error {
			credentialGuard, guardErr = credentials.NewLeakGuard(secret)
			return guardErr
		})
		if guardErr != nil || err != nil {
			credentialGuard.Destroy()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return &Error{Code: CodeCredentialUnavailable, Cause: errors.Join(guardErr, err)}
		}
	}
	state.credentialGuard = credentialGuard
	if session != nil {
		state.sessionGuard = session.guard
		session.guard = nil
	}
	return nil
}

func destroyExecutionGuards(state *executionState) {
	if state != nil {
		for _, guard := range []*credentials.LeakGuard{state.credentialGuard, state.sessionGuard} {
			guard.Destroy()
		}
		state.credentialGuard, state.sessionGuard = nil, nil
	}
}

func (adapter *Adapter) scrubRecoveryFailure(runPath string, cause error) error {
	if err := adapter.credentialOutputScrub(runPath); err != nil {
		return err
	}
	return cause
}
