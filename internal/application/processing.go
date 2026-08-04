package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type ProcessResult struct {
	Processed bool
	GoalRef   goal.GoalRef
	Action    ActionKind
}

type ActionClaimSelection struct {
	ExcludeLaunch bool
}

func (orchestrator *Orchestrator) ProcessNext(ctx context.Context, workerRef string) (ProcessResult, error) {
	claim, found, err := orchestrator.ClaimNextAction(ctx, workerRef, ActionClaimSelection{})
	if err != nil || !found {
		return ProcessResult{}, err
	}
	return orchestrator.ProcessClaim(ctx, claim)
}

func (orchestrator *Orchestrator) ClaimNextAction(
	ctx context.Context,
	workerRef string,
	selection ActionClaimSelection,
) (ActionClaim, bool, error) {
	if orchestrator == nil {
		return ActionClaim{}, false, errors.New("application.unavailable")
	}
	if strings.TrimSpace(workerRef) == "" {
		return ActionClaim{}, false, errors.New("application.worker_ref_required")
	}
	token, err := orchestrator.ids.NewID(ctx, "claim")
	if err != nil {
		return ActionClaim{}, false, err
	}
	candidatos, err := orchestrator.obtenerCandidatosCapacidad(ctx, selection.ExcludeLaunch)
	if err != nil {
		return ActionClaim{}, false, err
	}
	excluirLanzamiento := selection.ExcludeLaunch || len(candidatos) == 0 && orchestrator.capacitySources != nil
	return orchestrator.state.ClaimNextAction(ctx, ClaimRequest{
		WorkerRef: workerRef, Token: token, LeaseDuration: orchestrator.claimLease,
		AttestTestLeaseDuration: orchestrator.attestTestClaimLease,
		Capabilities:            orchestrator.agentCapabilities,
		BudgetPolicy:            orchestrator.budgetPolicy,
		ExcludeLaunch:           excluirLanzamiento,
		CapacityCandidates:      candidatos,
	})
}

func (orchestrator *Orchestrator) ProcessClaim(
	ctx context.Context,
	claim ActionClaim,
) (ProcessResult, error) {
	if orchestrator == nil {
		return ProcessResult{}, errors.New("application.unavailable")
	}
	return orchestrator.processClaim(ctx, claim)
}

func (orchestrator *Orchestrator) processClaim(
	ctx context.Context,
	claim ActionClaim,
) (ProcessResult, error) {
	result := ProcessResult{Processed: true, GoalRef: claim.Action.GoalRef, Action: claim.Action.Kind}
	if claim.Disposition == ActionClaimDispositionRetryBudgetIrreversible {
		return result, orchestrator.processIrreversibleRetryBudget(ctx, claim)
	}
	if claim.Disposition != ActionClaimDispositionNormal {
		return result, &StateError{Code: StateConflict}
	}
	var err error
	switch claim.Action.Kind {
	case ActionPrepareWorkspace:
		err = orchestrator.processPrepareWorkspace(ctx, claim)
	case ActionLaunchAgent:
		err = orchestrator.processLaunch(ctx, claim)
	case ActionObserveAgent:
		err = orchestrator.processObservation(ctx, claim)
	case ActionStopAgent:
		err = orchestrator.processStop(ctx, claim)
	case ActionCommitChange:
		err = orchestrator.processCommitChange(ctx, claim)
	case ActionAttestTest:
		err = orchestrator.processAttestTest(ctx, claim)
	case ActionIntegrateChange:
		err = orchestrator.processIntegrateChange(ctx, claim)
	case ActionAdmitMailbox:
		err = orchestrator.processPostArtifactMailboxAdmission(ctx, claim)
	case ActionRevokeSession:
		err = orchestrator.processExecutionSessionRevocation(ctx, claim)
	default:
		err = orchestrator.quarantine(ctx, claim, fmt.Sprintf("application.action_kind_invalid:%s", claim.Action.Kind))
	}
	return result, err
}

func (orchestrator *Orchestrator) processIrreversibleRetryBudget(
	ctx context.Context,
	claim ActionClaim,
) error {
	if claim.Action.Kind != ActionLaunchAgent ||
		claim.Disposition != ActionClaimDispositionRetryBudgetIrreversible {
		return &StateError{Code: StateConflict}
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionLaunchAgent); err != nil {
		return &StateError{Code: StateConflict}
	}
	item, itemFound := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	predecessor, predecessorFound := executionByRef(record.Executions, execution.ReplacesExecutionRef)
	intent, intentFound := effectIntentByRef(record.EffectIntents, claim.Action.EffectIntentRef)
	localState := execution.State == ExecutionQueued || execution.State == ExecutionDispatching
	if !itemFound || !executionFound || !predecessorFound || !intentFound ||
		!localState || execution.AttemptNo <= 1 ||
		predecessor.Ref != execution.ReplacesExecutionRef || predecessor.State != ExecutionFailed ||
		predecessor.GoalRef != execution.GoalRef || predecessor.WorkItemRef != execution.WorkItemRef ||
		execution.BudgetReservationRef != "" || execution.EffectIntentRef != "" ||
		execution.LaunchReceiptRef != "" ||
		!execution.StartedAt.IsZero() || !execution.ProviderAcceptedAt.IsZero() ||
		claim.BudgetReservationRef != "" || claim.BudgetReservation != (governance.BudgetReservation{}) ||
		intent.Ref != claim.Action.EffectIntent.Ref ||
		EffectIntentDigest(intent) != EffectIntentDigest(claim.Action.EffectIntent) ||
		intent.ActionRef != claim.Action.Ref || intent.ActionKind != ActionLaunchAgent ||
		intent.Subject.GoalRef != execution.GoalRef || intent.Subject.WorkItemRef != execution.WorkItemRef ||
		intent.Subject.ExecutionRef != execution.Ref || intent.Demand != item.BudgetDemand() ||
		priorEffectAttemptBlocksDispatch(record, claim.Action) {
		return &StateError{Code: StateConflict}
	}
	disposition, evidence, err := retryBudgetExhaustionForRecord(record, item.BudgetDemand())
	if err != nil {
		return err
	}
	if disposition != RetryBudgetIrreversible || evidence != claim.RetryBudgetExhaustion {
		// A new durable settlement changed the frontier after claim. Never
		// consume a decision proved against a different causal snapshot.
		return &StateError{Code: StateConflict}
	}
	code := predecessor.FailureCode
	if code == "" {
		code = "governance.retry_budget_irreversible"
	}
	at := orchestrator.clock.Now().UTC()
	switch {
	case isReviewerExecution(execution):
		return orchestrator.replaceReviewerExecution(
			ctx, claim, record, execution, code, failedExecutionMustTerminate,
			at, unknownUsage(), 0, true,
		)
	case isCouncilExecution(execution):
		return orchestrator.replaceCouncilExecution(
			ctx, claim, record, execution, code, failedExecutionMustTerminate,
			at, unknownUsage(), 0, true,
		)
	case execution.Purpose == ExecutionPurposeWork || execution.Purpose == ExecutionPurposeAuthor:
		return orchestrator.interruptExhaustedExecution(
			ctx, claim, record, execution, item, code, at, unknownUsage(), 0, true,
		)
	default:
		return &StateError{Code: StateConflict}
	}
}

func (orchestrator *Orchestrator) processExecutionSessionRevocation(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	execution, found := executionForAction(record, claim.Action)
	if !found || claim.Action.Kind != ActionRevokeSession ||
		claim.Action.Ref != "action:revoke-execution-session:"+execution.Ref.String() ||
		!terminalExecutionState(execution.State) {
		return &StateError{Code: StateConflict}
	}
	if orchestrator.executionSessions == nil {
		return orchestrator.requeueExecutionSessionRevocation(ctx, claim, execution)
	}
	if err := orchestrator.executionSessions.Revoke(ctx, ExecutionSessionRequest(record.Goal, execution)); err != nil {
		return orchestrator.requeueExecutionSessionRevocation(ctx, claim, execution)
	}
	return orchestrator.state.RecordExecutionSessionRevoked(ctx, ExecutionSessionRevokedState{
		Claim: claim, OperationAt: orchestrator.clock.Now().UTC(),
	})
}

func (orchestrator *Orchestrator) requeueExecutionSessionRevocation(
	ctx context.Context, claim ActionClaim, execution ExecutionRecord,
) error {
	now := orchestrator.clock.Now().UTC()
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution, ErrorCode: "application.execution_session_revoke_unavailable",
		AvailableAt: now.Add(orchestrator.observationDelay), OperationAt: now,
	})
}

func terminalExecutionState(state ExecutionState) bool {
	return state == ExecutionSucceeded || state == ExecutionFailed ||
		state == ExecutionCanceled || state == ExecutionStopped
}

func (orchestrator *Orchestrator) processLaunch(ctx context.Context, claim ActionClaim) error {
	record, item, execution, phase, proceed, err := orchestrator.loadClaimedLaunch(ctx, claim)
	if err != nil || !proceed {
		return err
	}
	sessionRef, proceed, err := orchestrator.ensureExecutionSession(ctx, claim, record, execution)
	if err != nil || !proceed {
		return err
	}
	execution.ExecutionSessionRef = sessionRef
	record, item, execution, proceed, err = orchestrator.prepareLaunchDispatch(ctx, claim, record, item, execution)
	if err != nil || !proceed {
		return err
	}
	request := agentLaunchRequest(record.Goal, item, execution, phase)
	if isReviewerExecution(execution) {
		request, err = reviewerAgentLaunchRequest(record, item, execution, phase, orchestrator.testAttestationPolicy)
		if err != nil {
			return orchestrator.quarantineUnapplied(ctx, claim, err.Error())
		}
	} else if isCouncilExecution(execution) {
		request, err = councilAgentLaunchRequest(record, item, execution, phase)
		if err != nil {
			return orchestrator.quarantineUnapplied(ctx, claim, err.Error())
		}
	}
	request.SessionRef, request.ReferenciaColocacion, request.RequierePreservacionEntorno =
		sessionRef, claim.ReferenciaColocacion, orchestrator.agentCapabilities.RequierePreservacionEntorno
	targetDigest := authorLaunchTargetDigest(request)
	if isReviewerExecution(execution) || isCouncilExecution(execution) {
		targetDigest = reviewerLaunchTargetDigest(request)
	}
	if targetDigest != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantineUnapplied(ctx, claim, "application.effect_target_mismatch")
	}
	attempt, receipt, proceed, err := orchestrator.dispatchLaunchEffect(ctx, claim, record, execution, request)
	if err != nil || !proceed {
		return err
	}
	latest, latestItem, latestExecution, err := orchestrator.reloadPreparedLaunch(ctx, claim)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	record, item, execution = latest, latestItem, latestExecution
	if isReviewerExecution(execution) && !reviewExternalRefAvailable(record, execution, receipt.ExternalRef) {
		return orchestrator.abortReviewerLaunch(ctx, claim, record, execution, attempt, receipt)
	}
	if isCouncilExecution(execution) && !councilExternalRefAvailable(record, execution, receipt.ExternalRef) {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.State = ExecutionRunning
	execution.ProviderRef, execution.ModelRef = receipt.ProviderRef, receipt.ModelRef
	execution.AgentRef, execution.ExternalRef, execution.RequierePreservacionEntorno =
		receipt.AgentRef, receipt.ExternalRef, receipt.RequierePreservacionEntorno
	execution.StartedAt = transitionAt
	execution.DeadlineAt = transitionAt.Add(orchestrator.executionTimeout)
	execution.ProviderAcceptedAt = receipt.AcceptedAt.UTC()
	externalReceipt, err := effectReceipt(
		claim, attempt, receipt.ReceiptRef, EffectStatusAccepted, unknownUsage(), transitionAt,
	)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	execution.LaunchReceiptRef = externalReceipt.Ref
	next := ActionRecord{
		Ref: "action:observe:" + execution.Ref.String(), Kind: ActionObserveAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(),
		AvailableAt: orchestrator.clock.Now(),
	}
	if cleanup, pending := pendingReviewCleanupControl(record, execution); pending {
		policy, policyErr := historicalEffectPolicy(record)
		if policyErr != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		next, err = orchestrator.reviewCleanupStopAction(policy, cleanup, record.Goal, item, execution, transitionAt)
		if err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
	} else if cleanup, pending := pendingCouncilCleanupControl(record, execution); pending {
		policy, policyErr := historicalEffectPolicy(record)
		if policyErr != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		next, err = orchestrator.councilCleanupStopAction(policy, cleanup, record.Goal, item, execution, transitionAt)
		if err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
	}
	err = orchestrator.state.RecordLaunchAccepted(ctx, LaunchAcceptedState{
		Claim: claim, Execution: execution, NextAction: next, EffectReceipt: externalReceipt,
		Event: EventRecord{
			Ref: "event:execution-accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			OccurredAt: transitionAt,
		},
		OperationAt: transitionAt,
	})
	if err != nil {
		if isReviewerExecution(execution) {
			latest, _, latestExecution, reloadErr := orchestrator.reloadPreparedLaunch(ctx, claim)
			if reloadErr == nil && !reviewExternalRefAvailable(latest, latestExecution, receipt.ExternalRef) {
				return orchestrator.abortReviewerLaunch(ctx, claim, latest, latestExecution, attempt, receipt)
			}
		}
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	return nil
}

func (orchestrator *Orchestrator) ensureExecutionSession(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	execution ExecutionRecord,
) (ports.ExecutionSessionRef, bool, error) {
	if orchestrator.executionSessions == nil {
		return "", true, nil
	}
	request := ExecutionSessionRequest(record.Goal, execution)
	receipt, err := orchestrator.executionSessions.Ensure(ctx, request)
	if err != nil {
		requeueErr := orchestrator.requeueUnappliedEffect(
			ctx, claim, execution, "", "application.execution_session_unavailable",
		)
		return "", false, requeueErr
	}
	method := receipt.Authority.ServicePrincipal.Method
	expected, deriveErr := DeriveExecutionSessionAuthority(request, method)
	if deriveErr != nil || expected != receipt.Authority ||
		receipt.EnsuredAt.IsZero() {
		err = orchestrator.quarantineUnapplied(ctx, claim, "application.execution_session_invalid")
		return "", false, err
	}
	return receipt.Authority.SessionRef, true, nil
}

func (orchestrator *Orchestrator) loadClaimedLaunch(
	ctx context.Context,
	claim ActionClaim,
) (GoalRecord, goal.WorkItem, ExecutionRecord, goal.PhaseInstance, bool, error) {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			err = orchestrator.quarantineUnapplied(ctx, claim, "application.action_goal_not_found")
		}
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	if priorEffectAttemptBlocksDispatch(record, claim.Action) {
		err = orchestrator.quarantineUnknownApplied(ctx, claim)
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	if err := validateClaimedRecord(claim, record, ActionLaunchAgent); err != nil {
		err = orchestrator.quarantineUnapplied(ctx, claim, err.Error())
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		err = orchestrator.quarantineUnapplied(ctx, claim, err.Error())
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, found := executionForAction(record, claim.Action)
	if !ok || !found {
		err = orchestrator.quarantineUnapplied(ctx, claim, "state.conflict")
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		err = orchestrator.requeueUnappliedEffect(ctx, claim, execution, "", err.Error())
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	execution.BudgetReservationRef = claim.BudgetReservation.Ref
	execution.EffectIntentRef = claim.Action.EffectIntent.Ref
	if execution.State == ExecutionQueued && launchBlockedByLifecycleControl(record.Goal, item) {
		err = orchestrator.requeueUnappliedEffect(ctx, claim, execution, "", "application.launch_control_pending")
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !phaseFound {
		err = orchestrator.quarantineUnapplied(ctx, claim, "state.conflict")
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, goal.PhaseInstance{}, false, err
	}
	return record, item, execution, phase, true, nil
}

func (orchestrator *Orchestrator) prepareLaunchDispatch(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
) (GoalRecord, goal.WorkItem, ExecutionRecord, bool, error) {
	if execution.State != ExecutionQueued {
		return record, item, execution, true, nil
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	aggregate := record.Goal
	if isReviewerExecution(execution) {
		if err := validateReviewerLaunch(record, item, execution, orchestrator.testAttestationPolicy); err != nil {
			err = orchestrator.quarantineUnapplied(ctx, claim, err.Error())
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
		execution.State = ExecutionDispatching
		err := orchestrator.state.RecordLaunchPrepared(ctx, LaunchPreparedState{
			Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: aggregate, Execution: execution,
			Event: EventRecord{Ref: "event:execution-dispatching:" + execution.Ref.String(), Kind: "execution.dispatching",
				GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt},
			OperationAt: transitionAt,
		})
		if err != nil {
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
		record.Executions = replaceExecution(record.Executions, execution)
		return record, item, execution, true, nil
	}
	if isCouncilExecution(execution) {
		if err := validateCouncilLaunch(record, item, execution); err != nil {
			err = orchestrator.quarantineUnapplied(ctx, claim, err.Error())
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
		execution.State = ExecutionDispatching
		err := orchestrator.state.RecordLaunchPrepared(ctx, LaunchPreparedState{
			Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: aggregate, Execution: execution,
			Event: EventRecord{Ref: "event:execution-dispatching:" + execution.Ref.String(), Kind: "execution.dispatching",
				GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt},
			OperationAt: transitionAt,
		})
		if err != nil {
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
		record.Executions = replaceExecution(record.Executions, execution)
		return record, item, execution, true, nil
	}
	switch item.State() {
	case goal.WorkItemStatePending:
		var startErr error
		aggregate, startErr = record.Goal.StartWorkItem(
			record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref, transitionAt,
		)
		if goal.ErrorCodeOf(startErr) == goal.ErrorWorkItemNotReady {
			err := orchestrator.requeueUnappliedEffect(ctx, claim, execution, "", "application.work_item_not_ready")
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
		if startErr != nil {
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, startErr
		}
	case goal.WorkItemStateRunning:
		bound, found := item.Execution()
		if !found || bound != execution.Ref {
			err := orchestrator.quarantineUnapplied(ctx, claim, "state.conflict")
			return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
		}
	default:
		err := orchestrator.quarantineUnapplied(ctx, claim, "state.conflict")
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
	}
	execution.State = ExecutionDispatching
	err := orchestrator.state.RecordLaunchPrepared(ctx, LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: aggregate, Execution: execution,
		Event: EventRecord{
			Ref: "event:execution-dispatching:" + execution.Ref.String(), Kind: "execution.dispatching",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt,
		},
		OperationAt: transitionAt,
	})
	if IsStateError(err, StateConflict) {
		err = orchestrator.requeueUnappliedEffect(
			ctx, claim, executionWithState(execution, ExecutionQueued), "", "state.conflict",
		)
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
	}
	if err != nil {
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, false, err
	}
	record.Goal = aggregate
	record.Executions = replaceExecution(record.Executions, execution)
	item, _ = aggregate.WorkItem(item.Ref())
	return record, item, execution, true, nil
}

func agentLaunchRequest(
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	phase goal.PhaseInstance,
) ports.AgentLaunchRequest {
	request := launchEffectTargetRequest(aggregate, item, execution)
	if isReviewerExecution(execution) {
		return ports.AgentLaunchRequest{}
	}
	request.Objective = item.Objective()
	request.PhaseRef, request.PhaseKey = phase.Ref().String(), item.Phase().String()
	request.PhaseTemplateRef = phase.TemplateRef().String()
	request.PhaseInputRefs, request.PhaseCriterionRefs = workItemRefs(phase.InputRefs()), workItemRefs(phase.CriterionRefs())
	request.RoleKey = item.Role().String()
	request.SkillRefs, request.ToolRefs = workItemRefs(item.SkillRefs()), workItemRefs(item.ToolRefs())
	request.CapabilityRefs, request.WriteSet = workItemRefs(item.CapabilityRefs()), workItemWriteSet(item)
	request.OutputContract, request.ArtifactMediaType = string(item.OutputContract().Kind()), execution.ArtifactMediaType
	request.ExecutionWorkspaceRef = execution.ExecutionWorkspaceRef
	request.MaxOutputBytes, request.BudgetDemand = execution.MaxOutputBytes, item.BudgetDemand()
	request.SecurityCriticality, request.ReasoningEffort = item.SecurityCriticality(), item.ReasoningEffort()
	return request
}

func validateReviewerLaunch(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	policy TestAttestationPolicy,
) error {
	if item.State() != goal.WorkItemStateRunning {
		return errors.New("review.subject_mismatch")
	}
	attachment, err := reviewAttached(record, item, execution, policy)
	if err != nil || attachment.Author.State != ExecutionAwaitingIntegration {
		return errors.New("review.subject_mismatch")
	}
	return nil
}

func (orchestrator *Orchestrator) dispatchLaunchEffect(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	execution ExecutionRecord,
	request ports.AgentLaunchRequest,
) (EffectAttempt, ports.AgentLaunchReceipt, bool, error) {
	attempt, err := orchestrator.beginNewEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, err
	}
	request.EffectAuthority = ports.AgentLaunchEffectAuthority{
		AuthorizationReceiptRef: claim.Action.EffectIntent.Authority.Ref(),
		EffectApprovalRef:       attempt.ApprovalRef,
		EffectAttemptRef:        attempt.Ref,
		ActionFence:             attempt.ActionFence,
		StartedAt:               attempt.StartedAt,
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false,
			orchestrator.quarantineUnappliedAttempt(ctx, claim, err.Error(), attempt.Ref)
	}
	receipt, launchErr := orchestrator.launcher.Launch(ctx, request)
	if launchErr != nil {
		var handled error
		if isDefinitelyNotAppliedAgentError(launchErr) {
			latest, latestItem, latestExecution, reloadErr := orchestrator.reloadPreparedLaunch(ctx, claim)
			if reloadErr != nil {
				handled = reloadErr
			} else if cleanup, pending := pendingReviewCleanupControl(latest, latestExecution); isReviewerExecution(latestExecution) && pending {
				handled = orchestrator.resolveUnappliedReviewCleanupLaunch(
					ctx, claim, latest, latestItem, latestExecution, cleanup, attempt.Ref, orchestrator.clock.Now(),
				)
			} else if cleanup, pending := pendingCouncilCleanupControl(latest, latestExecution); isCouncilExecution(latestExecution) && pending {
				handled = orchestrator.resolveUnappliedCouncilCleanupLaunch(
					ctx, claim, latest, latestItem, latestExecution, cleanup, attempt.Ref, orchestrator.clock.Now(),
				)
			} else {
				switch {
				case latestItem.CancelRequested():
					handled = orchestrator.settleCanceledLaunchRejection(
						ctx, claim, latest, latestItem, latestExecution, attempt.Ref, "agent.launch_failed",
					)
				case isTemporaryAgentError(launchErr):
					handled = orchestrator.requeueUnappliedEffect(
						ctx, claim, latestExecution, attempt.Ref, "agent.temporarily_unavailable",
					)
				case isCouncilExecution(latestExecution):
					handled = orchestrator.replaceCouncilExecution(
						ctx, claim, latest, latestExecution, "agent.launch_failed", failedExecutionMayRetry,
						orchestrator.clock.Now(), unknownUsage(), 0, true,
					)
				default:
					handled = orchestrator.replaceExecutionAttempt(
						ctx, claim, latest, "agent.launch_failed", failedExecutionMayRetry,
						orchestrator.clock.Now(), unknownUsage(), 0, true,
					)
				}
			}
			if handled != nil {
				handled = orchestrator.quarantineUnknownApplied(ctx, claim)
			}
		} else {
			handled = orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, handled
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		err = orchestrator.quarantineUnknownApplied(ctx, claim)
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, err
	}
	if receipt.ProviderRef != orchestrator.agentCapabilities.ProviderRef ||
		receipt.ModelRef != orchestrator.agentCapabilities.ModelRef ||
		receipt.AgentRef != orchestrator.agentCapabilities.AgentRef {
		err = orchestrator.quarantineUnknownApplied(ctx, claim)
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, err
	}
	if isReviewerExecution(execution) && !reviewExternalRefAvailable(record, execution, receipt.ExternalRef) {
		err = orchestrator.abortReviewerLaunch(ctx, claim, record, execution, attempt, receipt)
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, err
	}
	if isCouncilExecution(execution) && !councilExternalRefAvailable(record, execution, receipt.ExternalRef) {
		err = orchestrator.quarantineUnknownApplied(ctx, claim)
		return EffectAttempt{}, ports.AgentLaunchReceipt{}, false, err
	}
	return attempt, receipt, true, nil
}

func launchBlockedByLifecycleControl(aggregate goal.Goal, item goal.WorkItem) bool {
	paused, _ := aggregate.EffectivePause(item.Ref())
	return paused || aggregate.CancelRequested() || item.CancelRequested()
}

func (orchestrator *Orchestrator) reloadPreparedLaunch(
	ctx context.Context,
	claim ActionClaim,
) (GoalRecord, goal.WorkItem, ExecutionRecord, error) {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, err
	}
	item, itemFound := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	if !itemFound || !executionFound || execution.State != ExecutionDispatching {
		return GoalRecord{}, goal.WorkItem{}, ExecutionRecord{}, &StateError{Code: StateConflict}
	}
	return record, item, execution, nil
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
	if isCouncilExecution(execution) {
		return orchestrator.processCouncilObservation(ctx, claim, record, item, execution)
	}
	reviewer := isReviewerExecution(execution)
	observation, observeErr := orchestrator.observer.Observe(ctx, execution.Ref)
	if observeErr != nil {
		if orchestrator.executionExpired(execution, claim) {
			if reviewer {
				return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
					"application.execution_expired", failedExecutionMayRetry,
					orchestrator.clock.Now(), unknownUsage(), 0, false)
			}
			return orchestrator.replaceExecutionAttempt(
				ctx, claim, record, "application.execution_expired", failedExecutionMayRetry, orchestrator.clock.Now(),
				unknownUsage(), 0, false,
			)
		}
		return orchestrator.requeue(ctx, claim, execution, "agent.observe_failed")
	}
	if observation.ExecutionRef != execution.Ref {
		if reviewer {
			return orchestrator.quarantine(ctx, claim, "agent.observation_execution_mismatch")
		}
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_execution_mismatch")
	}
	if err := ports.ValidateAgentObservation(observation, execution.MaxOutputBytes); err != nil {
		code := ports.AgentContractErrorCode(err)
		if isSpecHashFenceCode(code) {
			return orchestrator.quarantine(ctx, claim, code)
		}
		if reviewer {
			return orchestrator.replaceReviewerExecution(ctx, claim, record, execution, code, failedExecutionMayRetry,
				orchestrator.clock.Now(), observation.Usage, int64(len(observation.Content)), false)
		}
		return orchestrator.failGoal(ctx, claim, record, code)
	}
	if observation.SpecHash != record.Goal.SpecHash() {
		return orchestrator.quarantine(ctx, claim, "agent.observation_spec_hash_mismatch")
	}
	if item.CancelRequested() &&
		(observation.Status == ports.AgentCompleted || observation.Status == ports.AgentFailed) {
		return orchestrator.settleCanceledObservation(ctx, claim, record, item, execution, observation)
	}
	if observation.Status == ports.AgentCompleted && !compatibleMediaType(execution.ArtifactMediaType, observation.MediaType) {
		if reviewer {
			return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
				"agent.observation_media_type_mismatch", failedExecutionMayRetry, orchestrator.clock.Now(), observation.Usage,
				int64(len(observation.Content)), false)
		}
		return orchestrator.failGoal(ctx, claim, record, "agent.observation_media_type_mismatch")
	}
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.LastObservedAt = transitionAt
	execution.ProviderObservedAt = observation.ObservedAt.UTC()
	switch observation.Status {
	case ports.AgentPending, ports.AgentRunning:
		if orchestrator.executionExpired(execution, claim) {
			if reviewer {
				return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
					"application.execution_expired", failedExecutionMayRetry, transitionAt, observation.Usage,
					int64(len(observation.Content)), false)
			}
			return orchestrator.replaceExecutionAttempt(
				ctx, claim, record, "application.execution_expired", failedExecutionMayRetry, transitionAt,
				observation.Usage, int64(len(observation.Content)), false,
			)
		}
		return orchestrator.requeue(ctx, claim, execution, "")
	case ports.AgentFailed:
		retryPolicy := failedExecutionRetryPolicyFor(observation)
		if reviewer {
			return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
				observation.ErrorCode, retryPolicy, transitionAt, observation.Usage, int64(len(observation.Content)), false)
		}
		return orchestrator.replaceExecutionAttempt(
			ctx, claim, record, observation.ErrorCode, retryPolicy, transitionAt,
			observation.Usage, int64(len(observation.Content)), false,
		)
	case ports.AgentCompleted:
		if reviewer {
			return orchestrator.recordReviewerObservation(ctx, claim, record, item, execution, observation, transitionAt)
		}
		if len(item.WriteSet()) != 0 {
			return orchestrator.stageExecutionOutput(ctx, claim, record, execution, observation, transitionAt)
		}
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

func (orchestrator *Orchestrator) succeedGoal(ctx context.Context, claim ActionClaim, record GoalRecord, execution ExecutionRecord, observation ports.AgentObservation, transitionAt time.Time) error {
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		return &StateError{Code: StateConflict}
	}
	if !record.Goal.ChildHandoffsResolved(item.Ref()) {
		return orchestrator.requeue(ctx, claim, execution, "application.child_handoffs_pending")
	}
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
	settlement, err := settlementFor(record, execution, observation.Usage, int64(len(observation.Content)), transitionAt)
	if err != nil {
		return err
	}
	artifact := artifactProvenanceRecord(stored, aggregate, item, execution, transitionAt)
	attestation := artifactProvenanceAttestation(
		attestationRef, stored.Ref, aggregate, item, execution, transitionAt,
	)
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, existing, transitionAt,
	)
	if err != nil {
		return err
	}
	var postArtifactAction *ActionRecord
	if orchestrator.postArtifactMailbox != nil && orchestrator.executionSessions != nil {
		succeededItem, _ := aggregate.WorkItem(item.Ref())
		postArtifactAction, err = postArtifactMailboxAction(aggregate, succeededItem, execution, transitionAt)
		if err != nil {
			return err
		}
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
		PostArtifactAction: postArtifactAction,
		BudgetSettlement:   settlement,
		OperationAt:        transitionAt,
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
	settlement, err := settlementFor(record, execution, unknownUsage(), 0, at)
	if err != nil {
		return err
	}
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, existing, at,
	)
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
		BudgetSettlement: settlement,
		OperationAt:      at,
	})
}

type failedExecutionRetryPolicy uint8

const (
	failedExecutionMayRetry failedExecutionRetryPolicy = iota
	failedExecutionMustTerminate
)

func failedExecutionRetryPolicyFor(observation ports.AgentObservation) failedExecutionRetryPolicy {
	if observation.FailureDisposition == ports.AgentFailureDispositionTerminalSecurity {
		return failedExecutionMustTerminate
	}
	return failedExecutionMayRetry
}

func (orchestrator *Orchestrator) replaceExecutionAttempt(ctx context.Context, claim ActionClaim, record GoalRecord,
	code string, retryPolicy failedExecutionRetryPolicy, at time.Time, usage governance.ResourceUsage,
	diskBytes int64, definitelyUnapplied bool,
) error {
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, found := executionForAction(record, claim.Action)
	if !ok || !found || item.State() != goal.WorkItemStateRunning {
		return &StateError{Code: StateConflict}
	}
	switch retryPolicy {
	case failedExecutionMustTerminate:
		return orchestrator.interruptExhaustedExecution(
			ctx, claim, record, execution, item, code, at, usage, diskBytes, definitelyUnapplied,
		)
	case failedExecutionMayRetry:
	default:
		return errors.New("application.execution_retry_policy_invalid")
	}
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !authorityFound {
		if len(record.WorkItemAuthorities) == 0 {
			return orchestrator.interruptExhaustedExecution(
				ctx, claim, record, execution, item, "governance.legacy_reauthorization_required",
				at, usage, diskBytes, definitelyUnapplied,
			)
		}
		return errors.New("application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	if execution.AttemptNo >= execution.MaxExecutionAttempts {
		return orchestrator.interruptExhaustedExecution(
			ctx, claim, record, execution, item, code, at, usage, diskBytes, definitelyUnapplied,
		)
	}
	settlement, err := settlementForExecutionAttempt(
		record, claim, execution, usage, diskBytes, at, definitelyUnapplied,
	)
	if err != nil {
		return err
	}
	retryFits, err := retryFitsIrreversibleGoalBudget(record, settlement, item.BudgetDemand())
	if err != nil {
		return err
	}
	if !retryFits {
		return orchestrator.interruptExhaustedExecution(
			ctx, claim, record, execution, item, code, at, usage, diskBytes, definitelyUnapplied,
		)
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
	replacement, err := orchestrator.buildReplacementExecution(ctx, aggregate, item, execution, replacementRef, at)
	if err != nil {
		return err
	}
	updatedItem, _ := aggregate.WorkItem(item.Ref())
	availableAt := at.Add(executionRetryBackoff(orchestrator.observationDelay, execution.AttemptNo, orchestrator.executionTimeout))
	var next ActionRecord
	if len(updatedItem.WriteSet()) != 0 {
		next, err = orchestrator.prepareWorkspaceAction(policy, aggregate, updatedItem, replacement, authority, at, availableAt)
	} else {
		next, err = orchestrator.launchAction(policy, aggregate, updatedItem, replacement, authority, at, availableAt)
	}
	if err != nil {
		return err
	}
	events := []EventRecord{
		{Ref: "event:execution-failed:" + execution.Ref.String(), Kind: "execution.failed", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
		{Ref: "event:execution-queued:" + replacement.Ref.String(), Kind: "execution.queued", GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacement.Ref, OccurredAt: at},
	}
	err = orchestrator.state.RecordExecutionReplaced(ctx, ExecutionReplacedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: aggregate, FailedExecution: execution, ReplacementExecution: replacement,
		NextAction: next, Events: events, ErrorCode: execution.FailureCode,
		BudgetSettlement: settlement, OperationAt: at,
	})
	if IsStateError(err, StateRecipientMailboxActive) {
		return orchestrator.failGoalAt(ctx, claim, record, code, at)
	}
	return err
}

func (orchestrator *Orchestrator) buildReplacementExecution(ctx context.Context, aggregate goal.Goal, item goal.WorkItem,
	execution ExecutionRecord, replacementRef goal.ExecutionRef, at time.Time,
) (ExecutionRecord, error) {
	replacement := ExecutionRecord{
		Ref: replacementRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		State: ExecutionQueued, Purpose: execution.Purpose,
		ReviewSubjectDigest: execution.ReviewSubjectDigest,
		ArtifactMediaType:   execution.ArtifactMediaType,
		IdempotencyKey:      "execution:" + replacementRef.String(),
		MaxOutputBytes:      execution.MaxOutputBytes,
		CreatedAt:           at.UTC(),
	}
	if len(item.WriteSet()) == 0 {
		return replacement, nil
	}
	workspaceRef, err := newExecutionWorkspaceRef(ctx, orchestrator.ids)
	if err != nil {
		return ExecutionRecord{}, err
	}
	replacement.RepositoryRef = execution.RepositoryRef
	if replacement.RepositoryRef.String() == "" {
		replacement.RepositoryRef, err = orchestrator.state.ProjectRepository(ctx, aggregate.Project())
		if err != nil {
			return ExecutionRecord{}, err
		}
	}
	replacement.ExecutionWorkspaceRef = workspaceRef
	return replacement, nil
}

func (orchestrator *Orchestrator) interruptExhaustedExecution(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	execution ExecutionRecord,
	item goal.WorkItem,
	code string,
	at time.Time,
	usage governance.ResourceUsage,
	diskBytes int64,
	definitelyUnapplied bool,
) error {
	at = lifecycleTime(at, record.Goal, item)
	expectedExecutionState := execution.State
	aggregate, err := record.Goal.InterruptWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref,
		goal.WorkItemInterruptExecutionFailed, at,
	)
	if err != nil {
		return err
	}
	execution.State = ExecutionFailed
	execution.FailureCode = stableFailureCode(code)
	execution.FinishedAt = at
	settlement, err := settlementForExecutionAttempt(
		record, claim, execution, usage, diskBytes, at, definitelyUnapplied,
	)
	if err != nil {
		return err
	}
	events := []EventRecord{
		{
			Ref: "event:execution-failed:" + execution.Ref.String(), Kind: "execution.failed",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at,
		},
		{
			Ref: "event:work-interrupted:" + execution.Ref.String(), Kind: "work_item.interrupted",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, existing, at,
	)
	if err != nil {
		return err
	}
	events = append(events, scheduledEvents...)
	return orchestrator.state.RecordExecutionInterrupted(ctx, ExecutionInterruptedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		ExpectedExecutionState: expectedExecutionState,
		Goal:                   aggregate, Execution: execution, NewExecutions: newExecutions, NewActions: newActions,
		Events: events, BudgetSettlement: settlement, OperationAt: at,
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
	return orchestrator.requeueAfter(ctx, claim, execution, code, orchestrator.observationDelay)
}

func (orchestrator *Orchestrator) requeueStop(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	code string,
) error {
	base := claim.Action.EffectIntent.QuotaRetryDelay
	if claim.Action.Kind != ActionStopAgent || base <= 0 {
		return errors.New("application.stop_retry_policy_invalid")
	}
	delay := executionRetryBackoff(base, claim.DeliveryAttempt, orchestrator.executionTimeout)
	return orchestrator.requeueAfter(ctx, claim, execution, code, delay)
}

func (orchestrator *Orchestrator) requeueAfter(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	code string,
	delay time.Duration,
) error {
	now := orchestrator.clock.Now()
	errorCode := ""
	if strings.TrimSpace(code) != "" {
		errorCode = stableFailureCode(code)
	}
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution,
		AvailableAt: now.Add(delay), OperationAt: now,
		ErrorCode: errorCode,
	})
}

func (orchestrator *Orchestrator) requeueUnappliedEffect(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	causalAttemptRef string,
	code string,
) error {
	now := orchestrator.clock.Now()
	var settlement governance.BudgetSettlement
	var err error
	if causalAttemptRef == "" {
		settlement, err = releaseSettlement(claim, now)
	} else {
		settlement, err = releaseSettlementForAttempt(claim, causalAttemptRef, now)
	}
	if err != nil {
		return err
	}
	execution.BudgetReservationRef = ""
	execution.EffectIntentRef = ""
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: now.Add(orchestrator.observationDelay),
		ErrorCode: stableFailureCode(code), OperationAt: now, BudgetSettlement: &settlement,
		ClearEffectBinding: true,
	})
}

func (orchestrator *Orchestrator) executionExpired(execution ExecutionRecord, claim ActionClaim) bool {
	_ = claim // delivery retries never consume provider execution attempts.
	return !orchestrator.clock.Now().Before(execution.DeadlineAt)
}

func (orchestrator *Orchestrator) quarantine(ctx context.Context, claim ActionClaim, code string) error {
	return orchestrator.quarantineEffect(ctx, claim, code, false)
}

func (orchestrator *Orchestrator) quarantineUnapplied(ctx context.Context, claim ActionClaim, code string) error {
	return orchestrator.quarantineEffect(ctx, claim, code, true)
}

func (orchestrator *Orchestrator) quarantineUnappliedAttempt(
	ctx context.Context,
	claim ActionClaim,
	code string,
	attemptRef string,
) error {
	return orchestrator.quarantineEffectWithAttempt(ctx, claim, code, true, attemptRef)
}

func (orchestrator *Orchestrator) quarantineUnknownApplied(ctx context.Context, claim ActionClaim) error {
	return orchestrator.quarantineUnknownAppliedWithCause(ctx, claim, "")
}

func (orchestrator *Orchestrator) quarantineUnknownAppliedWithCause(
	ctx context.Context,
	claim ActionClaim,
	causeCode string,
) error {
	now := orchestrator.clock.Now()
	err := orchestrator.state.QuarantineAction(ctx, ActionQuarantinedState{
		Claim: claim, ErrorCode: effectUnknownAppliedCode, OperationAt: now,
		BudgetSettlement: nil, ClearEffectBinding: false,
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
	if causeCode != "" {
		return &effectUnknownAppliedError{causeCode: causeCode}
	}
	return errors.New(effectUnknownAppliedCode)
}

func (orchestrator *Orchestrator) quarantineEffect(
	ctx context.Context,
	claim ActionClaim,
	code string,
	definitelyUnapplied bool,
) error {
	return orchestrator.quarantineEffectWithAttempt(ctx, claim, code, definitelyUnapplied, "")
}

func (orchestrator *Orchestrator) quarantineEffectWithAttempt(
	ctx context.Context,
	claim ActionClaim,
	code string,
	definitelyUnapplied bool,
	attemptRef string,
) error {
	code = stableFailureCode(code)
	now := orchestrator.clock.Now()
	settlement, settlementErr := claimBudgetSettlement(claim, now, definitelyUnapplied)
	if settlementErr == nil && definitelyUnapplied && attemptRef != "" &&
		claim.Action.Kind == ActionLaunchAgent && claim.BudgetReservationRef != "" {
		causal, err := releaseSettlementForAttempt(claim, attemptRef, now)
		settlement, settlementErr = &causal, err
	}
	if settlementErr != nil {
		return settlementErr
	}
	err := orchestrator.state.QuarantineAction(ctx, ActionQuarantinedState{
		Claim: claim, ErrorCode: code, OperationAt: now, BudgetSettlement: settlement,
		ClearEffectBinding: definitelyUnapplied,
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
