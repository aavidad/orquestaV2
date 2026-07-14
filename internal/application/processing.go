package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type ProcessResult struct {
	Processed bool
	GoalRef   goal.GoalRef
	Action    ActionKind
}

func (orchestrator *Orchestrator) ProcessNext(ctx context.Context, workerRef string) (ProcessResult, error) {
	if orchestrator == nil {
		return ProcessResult{}, errors.New("application.unavailable")
	}
	if strings.TrimSpace(workerRef) == "" {
		return ProcessResult{}, errors.New("application.worker_ref_required")
	}
	token, err := orchestrator.ids.NewID(ctx, "claim")
	if err != nil {
		return ProcessResult{}, err
	}
	now := orchestrator.clock.Now()
	claim, found, err := orchestrator.state.ClaimNextAction(ctx, ClaimRequest{
		WorkerRef: workerRef, Token: token, Now: now, LeaseDuration: orchestrator.claimLease,
	})
	if err != nil || !found {
		return ProcessResult{}, err
	}
	result := ProcessResult{Processed: true, GoalRef: claim.Action.GoalRef, Action: claim.Action.Kind}
	switch claim.Action.Kind {
	case ActionLaunchAgent:
		err = orchestrator.processLaunch(ctx, claim)
	case ActionObserveAgent:
		err = orchestrator.processObservation(ctx, claim)
	default:
		err = orchestrator.quarantine(ctx, claim, fmt.Sprintf("application.action_kind_invalid:%s", claim.Action.Kind))
	}
	return result, err
}

func (orchestrator *Orchestrator) processLaunch(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			return orchestrator.quarantine(ctx, claim, "application.action_goal_not_found")
		}
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionLaunchAgent); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.State() != goal.WorkItemStatePending || record.Execution.State != ExecutionQueued {
		return &StateError{Code: StateConflict}
	}
	request := ports.AgentLaunchRequest{
		ExecutionRef: record.Execution.Ref, GoalRef: record.Goal.Ref(),
		WorkItemRef: item.Ref(), ActorRef: record.Goal.Actor(),
		ProjectRef: record.Goal.Project(), Objective: item.Objective(),
		ArtifactMediaType: record.Execution.ArtifactMediaType,
		IdempotencyKey:    record.Execution.IdempotencyKey,
		MaxOutputBytes:    record.Execution.MaxOutputBytes,
	}
	receipt, launchErr := orchestrator.launcher.Launch(ctx, request)
	if launchErr != nil {
		if ctx.Err() != nil {
			return orchestrator.requeue(ctx, claim, record.Execution, ctx.Err().Error())
		}
		if isTemporaryAgentError(launchErr) {
			return orchestrator.requeue(ctx, claim, record.Execution, "agent.temporarily_unavailable")
		}
		return orchestrator.failGoal(ctx, claim, record, "agent.launch_failed")
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return orchestrator.failGoal(ctx, claim, record, ports.AgentContractErrorCode(err))
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	aggregate, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Execution.Ref, transitionAt,
	)
	if err != nil {
		return err
	}
	execution := record.Execution
	execution.State = ExecutionRunning
	execution.ProviderRef = receipt.ProviderRef
	execution.ExternalRef = receipt.ExternalRef
	execution.StartedAt = transitionAt
	execution.DeadlineAt = transitionAt.Add(orchestrator.executionTimeout)
	execution.ProviderAcceptedAt = receipt.AcceptedAt.UTC()
	next := ActionRecord{
		Ref: "action:observe:" + execution.Ref.String(), Kind: ActionObserveAgent,
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		AvailableAt: orchestrator.clock.Now(),
	}
	return orchestrator.state.RecordLaunchAccepted(ctx, LaunchAcceptedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Goal: aggregate, Execution: execution,
		NextAction: next,
		Event: EventRecord{
			Ref: "event:execution-accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			OccurredAt: transitionAt,
		},
	})
}

func (orchestrator *Orchestrator) processObservation(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			return orchestrator.quarantine(ctx, claim, "application.action_goal_not_found")
		}
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionObserveAgent); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.State() != goal.WorkItemStateRunning || record.Execution.State != ExecutionRunning {
		return &StateError{Code: StateConflict}
	}
	observation, observeErr := orchestrator.observer.Observe(ctx, record.Execution.Ref)
	if observeErr != nil {
		if orchestrator.executionExpired(record.Execution, claim) {
			return orchestrator.failGoal(ctx, claim, record, "application.execution_expired")
		}
		return orchestrator.requeue(ctx, claim, record.Execution, "agent.observe_failed")
	}
	if observation.ExecutionRef != record.Execution.Ref {
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_execution_mismatch")
	}
	if err := ports.ValidateAgentObservation(observation, record.Execution.MaxOutputBytes); err != nil {
		return orchestrator.failGoal(ctx, claim, record, ports.AgentContractErrorCode(err))
	}
	if observation.Status == ports.AgentCompleted && !compatibleMediaType(record.Execution.ArtifactMediaType, observation.MediaType) {
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_media_type_mismatch")
	}
	execution := record.Execution
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.LastObservedAt = transitionAt
	execution.ProviderObservedAt = observation.ObservedAt.UTC()
	switch observation.Status {
	case ports.AgentPending, ports.AgentRunning:
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.failGoalAt(ctx, claim, record, "application.execution_expired", transitionAt)
		}
		return orchestrator.requeue(ctx, claim, execution, "")
	case ports.AgentFailed:
		return orchestrator.failGoalAt(ctx, claim, record, observation.ErrorCode, transitionAt)
	case ports.AgentCompleted:
		return orchestrator.succeedGoal(ctx, claim, record, execution, observation, transitionAt)
	default:
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_status_invalid")
	}
}

func (orchestrator *Orchestrator) succeedGoal(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	execution ExecutionRecord,
	observation ports.AgentObservation,
	transitionAt time.Time,
) error {
	putRequest := ports.PutArtifactRequest{
		MediaType: observation.MediaType, Content: observation.Content,
	}
	stored, err := orchestrator.artifacts.Put(ctx, putRequest)
	if err != nil {
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.failGoalAt(ctx, claim, record, "artifact.store_failed", transitionAt)
		}
		return orchestrator.requeue(ctx, claim, execution, "artifact.store_failed")
	}
	if err := ports.ValidateStoredArtifact(putRequest, stored); err != nil {
		return orchestrator.failGoalAt(ctx, claim, record, ports.ArtifactContractErrorCode(err), transitionAt)
	}
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	attestationRef, err := goal.NewAttestationRef("attestation:execution:" + execution.Ref.String())
	if err != nil {
		return err
	}
	aggregate, err := record.Goal.SucceedWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(),
		[]goal.ArtifactRef{stored.Ref}, []goal.AttestationRef{attestationRef}, transitionAt,
	)
	if err != nil {
		return err
	}
	aggregate, err = aggregate.Close(aggregate.Revision(), goal.GoalOutcomeSucceeded, transitionAt)
	if err != nil {
		return err
	}
	execution.State = ExecutionSucceeded
	execution.FinishedAt = transitionAt
	artifact := ArtifactRecord{
		Stored: stored, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		CreatedAt: transitionAt,
	}
	attestation := AttestationRecord{
		Ref: attestationRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ArtifactRef: stored.Ref,
		Policy: outputAttestationPolicy, AcceptedAt: transitionAt,
	}
	return orchestrator.state.RecordGoalSucceeded(ctx, GoalSucceededState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Goal: aggregate, Execution: execution,
		Artifact: artifact, Attestation: attestation,
		Events: []EventRecord{
			{Ref: "event:work-succeeded:" + execution.Ref.String(), Kind: "work_item.succeeded", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt},
			{Ref: "event:goal-succeeded:" + aggregate.Ref().String(), Kind: "goal.succeeded", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt},
		},
	})
}

func (orchestrator *Orchestrator) failGoal(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	code string,
) error {
	return orchestrator.failGoalAt(ctx, claim, record, code, orchestrator.clock.Now())
}

func (orchestrator *Orchestrator) failGoalAt(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	code string,
	at time.Time,
) error {
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok {
		return &StateError{Code: StateConflict}
	}
	at = lifecycleTime(at, record.Goal, item)
	aggregate := record.Goal
	expectedGoal := aggregate.Revision()
	expectedItem := item.Revision()
	var err error
	if item.State() == goal.WorkItemStatePending {
		aggregate, err = aggregate.StartWorkItem(
			aggregate.Revision(), item.Revision(), item.Ref(), record.Execution.Ref, at,
		)
		if err != nil {
			return err
		}
		item, _ = aggregate.WorkItem(item.Ref())
	}
	aggregate, err = aggregate.FailWorkItem(
		aggregate.Revision(), item.Revision(), item.Ref(), at,
	)
	if err != nil {
		return err
	}
	aggregate, err = aggregate.Close(aggregate.Revision(), goal.GoalOutcomeFailed, at)
	if err != nil {
		return err
	}
	execution := record.Execution
	execution.State = ExecutionFailed
	if execution.StartedAt.IsZero() {
		execution.StartedAt = at
	}
	execution.FailureCode = stableFailureCode(code)
	execution.FinishedAt = at.UTC()
	return orchestrator.state.RecordGoalFailed(ctx, GoalFailedState{
		Claim: claim, ExpectedGoalRevision: expectedGoal,
		ExpectedItemRevision: expectedItem, Goal: aggregate, Execution: execution,
		Events: []EventRecord{
			{Ref: "event:work-failed:" + execution.Ref.String(), Kind: "work_item.failed", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
			{Ref: "event:goal-failed:" + aggregate.Ref().String(), Kind: "goal.failed", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
		},
	})
}

func (orchestrator *Orchestrator) requeue(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	code string,
) error {
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution,
		AvailableAt: orchestrator.clock.Now().Add(orchestrator.observationDelay),
		ErrorCode:   stableFailureCode(code),
	})
}

func (orchestrator *Orchestrator) executionExpired(execution ExecutionRecord, claim ActionClaim) bool {
	return claim.Attempt >= execution.MaxAttempts || !orchestrator.clock.Now().Before(execution.DeadlineAt)
}

func (orchestrator *Orchestrator) quarantine(ctx context.Context, claim ActionClaim, code string) error {
	code = stableFailureCode(code)
	err := orchestrator.state.QuarantineAction(ctx, ActionQuarantinedState{
		Claim: claim, ErrorCode: code,
		Event: EventRecord{
			Ref:  fmt.Sprintf("event:action-quarantined:%s:%d", claim.Action.Ref, claim.Attempt),
			Kind: "action.quarantined", GoalRef: claim.Action.GoalRef,
			WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
			OccurredAt: orchestrator.clock.Now(),
		},
	})
	if err != nil {
		return err
	}
	return errors.New(code)
}
