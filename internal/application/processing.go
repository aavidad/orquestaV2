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
	claim, found, err := orchestrator.state.ClaimNextAction(ctx, ClaimRequest{
		WorkerRef: workerRef, Token: token, LeaseDuration: orchestrator.claimLease,
		Capabilities: orchestrator.agentCapabilities,
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
	execution, found := executionForAction(record, claim.Action)
	if !ok || !found {
		return &StateError{Code: StateConflict}
	}
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !phaseFound {
		return &StateError{Code: StateConflict}
	}
	if execution.State == ExecutionQueued {
		transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
		aggregate, startErr := record.Goal.StartWorkItem(
			record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref, transitionAt,
		)
		if goal.ErrorCodeOf(startErr) == goal.ErrorWorkItemNotReady {
			return orchestrator.requeue(ctx, claim, execution, "application.work_item_not_ready")
		}
		if startErr != nil {
			return startErr
		}
		execution.State = ExecutionDispatching
		if err := orchestrator.state.RecordLaunchPrepared(ctx, LaunchPreparedState{
			Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: aggregate,
			Execution: execution,
			Event: EventRecord{
				Ref: "event:execution-dispatching:" + execution.Ref.String(), Kind: "execution.dispatching",
				GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
				OccurredAt: transitionAt,
			},
			OperationAt: transitionAt,
		}); err != nil {
			if IsStateError(err, StateConflict) {
				return orchestrator.requeue(ctx, claim, executionWithState(execution, ExecutionQueued), "state.conflict")
			}
			return err
		}
		record.Goal = aggregate
		record.Executions = replaceExecution(record.Executions, execution)
		item, _ = aggregate.WorkItem(item.Ref())
	}
	request := ports.AgentLaunchRequest{
		ExecutionRef: execution.Ref, GoalRef: record.Goal.Ref(),
		WorkItemRef: item.Ref(), SpecHash: record.Goal.SpecHash(), ActorRef: record.Goal.Actor(),
		ProjectRef: record.Goal.Project(), Objective: item.Objective(),
		PlanGeneration: record.Goal.PlanGeneration(), AppSpecGeneration: record.Goal.AppSpec().Generation(),
		ExecutionAttempt: execution.AttemptNo,
		PhaseRef:         phase.Ref().String(), PhaseKey: item.Phase().String(),
		PhaseTemplateRef: phase.TemplateRef().String(),
		PhaseInputRefs:   workItemRefs(phase.InputRefs()), PhaseCriterionRefs: workItemRefs(phase.CriterionRefs()),
		RoleKey:   item.Role().String(),
		SkillRefs: workItemRefs(item.SkillRefs()), ToolRefs: workItemRefs(item.ToolRefs()),
		CapabilityRefs: workItemRefs(item.CapabilityRefs()),
		WriteSet:       workItemWriteSet(item), OutputContract: string(item.OutputContract().Kind()),
		ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey:    execution.IdempotencyKey,
		MaxOutputBytes:    execution.MaxOutputBytes,
	}
	receipt, launchErr := orchestrator.launcher.Launch(ctx, request)
	if launchErr != nil {
		if ctx.Err() != nil {
			return orchestrator.requeue(ctx, claim, execution, ctx.Err().Error())
		}
		if isTemporaryAgentError(launchErr) {
			return orchestrator.requeue(ctx, claim, execution, "agent.temporarily_unavailable")
		}
		return orchestrator.replaceExecutionAttempt(ctx, claim, record, "agent.launch_failed", orchestrator.clock.Now())
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		code := ports.AgentContractErrorCode(err)
		if isSpecHashFenceCode(code) {
			return orchestrator.quarantine(ctx, claim, code)
		}
		return orchestrator.failGoal(ctx, claim, record, code)
	}
	if receipt.ProviderRef != orchestrator.agentCapabilities.ProviderRef ||
		receipt.ModelRef != orchestrator.agentCapabilities.ModelRef ||
		receipt.AgentRef != orchestrator.agentCapabilities.AgentRef {
		return orchestrator.failGoal(ctx, claim, record, "agent.receipt_identity_mismatch")
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.State = ExecutionRunning
	execution.ProviderRef = receipt.ProviderRef
	execution.ModelRef = receipt.ModelRef
	execution.AgentRef = receipt.AgentRef
	execution.ExternalRef = receipt.ExternalRef
	execution.StartedAt = transitionAt
	execution.DeadlineAt = transitionAt.Add(orchestrator.executionTimeout)
	execution.ProviderAcceptedAt = receipt.AcceptedAt.UTC()
	next := ActionRecord{
		Ref: "action:observe:" + execution.Ref.String(), Kind: ActionObserveAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: item.Revision(),
		AvailableAt: orchestrator.clock.Now(),
	}
	return orchestrator.state.RecordLaunchAccepted(ctx, LaunchAcceptedState{
		Claim: claim, Execution: execution, NextAction: next,
		Event: EventRecord{
			Ref: "event:execution-accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			OccurredAt: transitionAt,
		},
		OperationAt: transitionAt,
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
	execution, found := executionForAction(record, claim.Action)
	if !ok || !found || item.State() != goal.WorkItemStateRunning || execution.State != ExecutionRunning {
		return &StateError{Code: StateConflict}
	}
	observation, observeErr := orchestrator.observer.Observe(ctx, execution.Ref)
	if observeErr != nil {
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.replaceExecutionAttempt(ctx, claim, record, "application.execution_expired", orchestrator.clock.Now())
		}
		return orchestrator.requeue(ctx, claim, execution, "agent.observe_failed")
	}
	if observation.ExecutionRef != execution.Ref {
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_execution_mismatch")
	}
	if err := ports.ValidateAgentObservation(observation, execution.MaxOutputBytes); err != nil {
		code := ports.AgentContractErrorCode(err)
		if isSpecHashFenceCode(code) {
			return orchestrator.quarantine(ctx, claim, code)
		}
		return orchestrator.failGoal(ctx, claim, record, code)
	}
	if observation.SpecHash != record.Goal.SpecHash() {
		return orchestrator.quarantine(ctx, claim, "agent.observation_spec_hash_mismatch")
	}
	if observation.Status == ports.AgentCompleted && !compatibleMediaType(execution.ArtifactMediaType, observation.MediaType) {
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_media_type_mismatch")
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.LastObservedAt = transitionAt
	execution.ProviderObservedAt = observation.ObservedAt.UTC()
	switch observation.Status {
	case ports.AgentPending, ports.AgentRunning:
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.replaceExecutionAttempt(ctx, claim, record, "application.execution_expired", transitionAt)
		}
		return orchestrator.requeue(ctx, claim, execution, "")
	case ports.AgentFailed:
		return orchestrator.replaceExecutionAttempt(ctx, claim, record, observation.ErrorCode, transitionAt)
	case ports.AgentCompleted:
		return orchestrator.succeedGoal(ctx, claim, record, execution, observation, transitionAt)
	default:
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_status_invalid")
	}
}

func isSpecHashFenceCode(code string) bool {
	switch code {
	case "agent.receipt_spec_hash_required",
		"agent.receipt_spec_hash_invalid",
		"agent.receipt_spec_hash_mismatch",
		"agent.observation_spec_hash_required",
		"agent.observation_spec_hash_invalid",
		"agent.observation_spec_hash_mismatch":
		return true
	default:
		return false
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
	// Artifact persistence may be slow. Lifecycle time and the repository's
	// exclusive lease fence must observe the time after that external effect.
	transitionAt = lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
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
	if outcome, closable := aggregate.ClosableOutcome(); closable {
		aggregate, err = aggregate.Close(aggregate.Revision(), outcome, transitionAt)
		if err != nil {
			return err
		}
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
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleReady(ctx, aggregate, existing, transitionAt)
	if err != nil {
		return err
	}
	events := []EventRecord{{Ref: "event:work-succeeded:" + execution.Ref.String(), Kind: "work_item.succeeded", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt}}
	if aggregate.State() == goal.GoalStateSucceeded {
		events = append(events, EventRecord{Ref: "event:goal-succeeded:" + aggregate.Ref().String(), Kind: "goal.succeeded", GoalRef: aggregate.Ref(), OccurredAt: transitionAt})
	} else if aggregate.State() == goal.GoalStateFailed {
		events = append(events, EventRecord{Ref: "event:goal-failed:" + aggregate.Ref().String(), Kind: "goal.failed", GoalRef: aggregate.Ref(), OccurredAt: transitionAt})
	}
	events = append(events, scheduledEvents...)
	return orchestrator.state.RecordGoalSucceeded(ctx, GoalSucceededState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Goal: aggregate, Execution: execution,
		Artifact: artifact, Attestation: attestation,
		NewExecutions: newExecutions, NewActions: newActions, Events: events,
		OperationAt: transitionAt,
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
	execution, found := executionForAction(record, claim.Action)
	if !found || item.State() != goal.WorkItemStateRunning {
		return &StateError{Code: StateConflict}
	}
	aggregate, err := aggregate.FailWorkItem(
		aggregate.Revision(), item.Revision(), item.Ref(), at,
	)
	if err != nil {
		return err
	}
	if outcome, closable := aggregate.ClosableOutcome(); closable {
		aggregate, err = aggregate.Close(aggregate.Revision(), outcome, at)
		if err != nil {
			return err
		}
	}
	execution.State = ExecutionFailed
	execution.FailureCode = stableFailureCode(code)
	execution.FinishedAt = at.UTC()
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleReady(ctx, aggregate, existing, at)
	if err != nil {
		return err
	}
	events := []EventRecord{{Ref: "event:work-failed:" + execution.Ref.String(), Kind: "work_item.failed", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at}}
	for _, after := range aggregate.WorkItems() {
		before, existed := record.Goal.WorkItem(after.Ref())
		if existed && before.State() != goal.WorkItemStateSkipped && after.State() == goal.WorkItemStateSkipped {
			events = append(events, EventRecord{Ref: "event:work-skipped:" + after.Ref().String(), Kind: "work_item.skipped", GoalRef: aggregate.Ref(), WorkItemRef: after.Ref(), OccurredAt: at})
		}
	}
	if aggregate.State() == goal.GoalStateFailed {
		events = append(events, EventRecord{Ref: "event:goal-failed:" + aggregate.Ref().String(), Kind: "goal.failed", GoalRef: aggregate.Ref(), OccurredAt: at})
	}
	events = append(events, scheduledEvents...)
	return orchestrator.state.RecordGoalFailed(ctx, GoalFailedState{
		Claim: claim, ExpectedGoalRevision: expectedGoal,
		ExpectedItemRevision: expectedItem, Goal: aggregate, Execution: execution,
		NewExecutions: newExecutions, NewActions: newActions, Events: events,
		OperationAt: at,
	})
}

func (orchestrator *Orchestrator) replaceExecutionAttempt(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	code string,
	at time.Time,
) error {
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, found := executionForAction(record, claim.Action)
	if !ok || !found || item.State() != goal.WorkItemStateRunning {
		return &StateError{Code: StateConflict}
	}
	if execution.AttemptNo >= execution.MaxExecutionAttempts {
		return orchestrator.failGoalAt(ctx, claim, record, code, at)
	}
	replacementRef, err := newExecutionRef(ctx, orchestrator.ids)
	if err != nil {
		return err
	}
	at = lifecycleTime(at, record.Goal, item)
	aggregate, err := record.Goal.ReplaceWorkItemExecution(
		record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref, replacementRef, at,
	)
	if err != nil {
		return err
	}
	execution.State = ExecutionFailed
	execution.FailureCode = stableFailureCode(code)
	execution.FinishedAt = at.UTC()
	replacement := ExecutionRecord{
		Ref: replacementRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		State: ExecutionDispatching, ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey: "execution:" + replacementRef.String(),
		MaxOutputBytes: execution.MaxOutputBytes,
		CreatedAt:      at.UTC(),
	}
	updatedItem, _ := aggregate.WorkItem(item.Ref())
	next := ActionRecord{
		Ref: "action:launch:" + replacementRef.String(), Kind: ActionLaunchAgent,
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacementRef,
		PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: updatedItem.Revision(),
		AvailableAt: at.Add(executionRetryBackoff(orchestrator.observationDelay, execution.AttemptNo, orchestrator.executionTimeout)),
	}
	events := []EventRecord{
		{Ref: "event:execution-failed:" + execution.Ref.String(), Kind: "execution.failed", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
		{Ref: "event:execution-dispatching:" + replacement.Ref.String(), Kind: "execution.dispatching", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacement.Ref, OccurredAt: at},
	}
	return orchestrator.state.RecordExecutionReplaced(ctx, ExecutionReplacedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: aggregate, FailedExecution: execution, ReplacementExecution: replacement,
		NextAction: next, Events: events, ErrorCode: execution.FailureCode, OperationAt: at,
	})
}

func executionRetryBackoff(base time.Duration, completedAttempt uint64, ceiling time.Duration) time.Duration {
	if base >= ceiling {
		return ceiling
	}
	delay := base
	for attempt := uint64(1); attempt < completedAttempt; attempt++ {
		if delay > ceiling-delay {
			return ceiling
		}
		delay += delay
	}
	return delay
}

func replaceExecution(records []ExecutionRecord, updated ExecutionRecord) []ExecutionRecord {
	result := append([]ExecutionRecord(nil), records...)
	for index := range result {
		if result[index].Ref == updated.Ref {
			result[index] = updated
			return result
		}
	}
	return append(result, updated)
}

func executionWithState(execution ExecutionRecord, state ExecutionState) ExecutionRecord {
	execution.State = state
	if state == ExecutionQueued {
		execution.StartedAt = time.Time{}
		execution.DeadlineAt = time.Time{}
	}
	return execution
}

func workItemWriteSet(item goal.WorkItem) []string {
	scopes := item.WriteSet()
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, scope.String())
	}
	return result
}

func phaseForWorkItem(aggregate goal.Goal, item goal.WorkItem) (goal.PhaseInstance, bool) {
	for _, phase := range aggregate.Phases() {
		if phase.Key() == item.Phase() {
			return phase, true
		}
	}
	return goal.PhaseInstance{}, false
}

func workItemRefs[T interface{ String() string }](refs []T) []string {
	if len(refs) == 0 {
		return nil
	}
	result := make([]string, len(refs))
	for index, ref := range refs {
		result[index] = ref.String()
	}
	return result
}

func (orchestrator *Orchestrator) requeue(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	code string,
) error {
	now := orchestrator.clock.Now()
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution,
		AvailableAt: now.Add(orchestrator.observationDelay), OperationAt: now,
		ErrorCode: stableFailureCode(code),
	})
}

func (orchestrator *Orchestrator) executionExpired(execution ExecutionRecord, claim ActionClaim) bool {
	_ = claim // delivery retries never consume provider execution attempts.
	return !orchestrator.clock.Now().Before(execution.DeadlineAt)
}

func (orchestrator *Orchestrator) quarantine(ctx context.Context, claim ActionClaim, code string) error {
	code = stableFailureCode(code)
	now := orchestrator.clock.Now()
	err := orchestrator.state.QuarantineAction(ctx, ActionQuarantinedState{
		Claim: claim, ErrorCode: code, OperationAt: now,
		Event: EventRecord{
			Ref:  fmt.Sprintf("event:action-quarantined:%s:%d", claim.Action.Ref, claim.DeliveryAttempt),
			Kind: "action.quarantined", GoalRef: claim.Action.GoalRef,
			WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
			OccurredAt: now,
		},
	})
	if err != nil {
		return err
	}
	return errors.New(code)
}
