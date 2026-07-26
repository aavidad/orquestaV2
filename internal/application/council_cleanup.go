package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const councilCleanupReason = "council.round_cleanup"

func IsCouncilCleanupControl(control ControlRecord) bool {
	return strings.HasPrefix(control.RequestRef, "request:council-cleanup:") &&
		strings.HasPrefix(control.Ref, "control:council-cleanup:") &&
		control.Operation == ControlStop && control.Target == ControlTargetExecution &&
		control.Mode == ports.AgentStopCooperative && control.Reason == councilCleanupReason
}

func IsRoundCleanupControl(control ControlRecord) bool {
	return IsReviewCleanupControl(control) || IsCouncilCleanupControl(control)
}

func pendingCouncilCleanupControl(record GoalRecord, execution ExecutionRecord) (ControlRecord, bool) {
	for _, control := range record.Controls {
		if IsCouncilCleanupControl(control) && control.Status == ControlRequested &&
			control.GoalRef == execution.GoalRef && control.WorkItemRef == execution.WorkItemRef &&
			control.ExecutionRef == execution.Ref && control.ExecutionAttempt == execution.AttemptNo {
			return control, true
		}
	}
	return ControlRecord{}, false
}

func councilCleanupAuthority(record GoalRecord, author ExecutionRecord) (EffectIntent, error) {
	for _, intent := range record.EffectIntents {
		if intent.ActionKind == ActionLaunchAgent && intent.Kind == EffectKindAgentLaunch &&
			intent.Subject.GoalRef == author.GoalRef && intent.Subject.WorkItemRef == author.WorkItemRef &&
			intent.Subject.ExecutionRef == author.Ref && intent.Permission == identity.PermissionGoalsCreate {
			return intent, nil
		}
	}
	return EffectIntent{}, errors.New("council.cleanup_authority_missing")
}

func (orchestrator *Orchestrator) councilCleanupPlan(record GoalRecord, item goal.WorkItem,
	exclude goal.ExecutionRef, subjectDigest CouncilSubjectDigest, triggerRef string, at time.Time,
) ([]CouncilParticipantRetirement, []string, []ControlRecord, []ActionRecord, []EventRecord, error) {
	author, found := authorBound(record, item)
	if !found {
		return nil, nil, nil, nil, nil, errors.New("council.cleanup_author_missing")
	}
	authority, err := councilCleanupAuthority(record, author)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	var retired []CouncilParticipantRetirement
	var retireActions []string
	var controls []ControlRecord
	var actions []ActionRecord
	var events []EventRecord
	for _, participant := range record.Executions {
		if participant.Ref == exclude || !isCouncilExecution(participant) ||
			participant.GoalRef != record.Goal.Ref() || participant.WorkItemRef != item.Ref() ||
			participant.CouncilSubjectDigest != subjectDigest {
			continue
		}
		switch participant.State {
		case ExecutionQueued:
			participant.State, participant.FailureCode, participant.FinishedAt =
				ExecutionFailed, "council.round_aborted", at.UTC()
			retired = append(retired, CouncilParticipantRetirement{
				Execution: participant, ExpectedState: ExecutionQueued,
			})
			retireActions = append(retireActions, "action:launch:"+participant.Ref.String())
			events = append(events, EventRecord{
				Ref: "event:council-round-retired:" + participant.Ref.String(), Kind: "council.round_participant_retired",
				GoalRef: participant.GoalRef, WorkItemRef: participant.WorkItemRef,
				ExecutionRef: participant.Ref, OccurredAt: at.UTC(),
			})
		case ExecutionDispatching, ExecutionRunning:
			if _, pending := pendingCouncilCleanupControl(record, participant); pending {
				continue
			}
			control := newCouncilCleanupControl(record, item, participant, authority, triggerRef, at)
			controls = append(controls, control)
			if participant.State == ExecutionRunning {
				action, actionErr := orchestrator.councilCleanupStopAction(
					policy, control, record.Goal, item, participant, at,
				)
				if actionErr != nil {
					return nil, nil, nil, nil, nil, actionErr
				}
				actions = append(actions, action)
				retireActions = append(retireActions, "action:observe:"+participant.Ref.String())
			}
			events = append(events, EventRecord{
				Ref: "event:council-cleanup-requested:" + participant.Ref.String(), Kind: "council.cleanup_requested",
				GoalRef: participant.GoalRef, WorkItemRef: participant.WorkItemRef,
				ExecutionRef: participant.Ref, OccurredAt: at.UTC(),
			})
		}
	}
	return retired, retireActions, controls, actions, events, nil
}

func newCouncilCleanupControl(record GoalRecord, item goal.WorkItem, participant ExecutionRecord,
	authority EffectIntent, triggerRef string, at time.Time,
) ControlRecord {
	digest := fingerprintFields("orquesta.council-cleanup.v1", triggerRef, participant.Ref.String(),
		string(participant.CouncilSubjectDigest))
	request := ControlRequest{
		RequestRef: "request:council-cleanup:" + digest, Operation: ControlStop,
		Target: ControlTargetExecution, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: participant.Ref, ExpectedExecutionAttempt: participant.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: councilCleanupReason,
	}
	return ControlRecord{
		Ref: "control:council-cleanup:" + digest, RequestRef: request.RequestRef,
		RequestFingerprint: controlFingerprint(authority.ProposedBy, record.Goal.Project(), request),
		PrincipalRef:       authority.ProposedBy, ProjectRef: record.Goal.Project(), GoalRef: record.Goal.Ref(),
		WorkItemRef: item.Ref(), WorkItemRevision: item.Revision(), ExecutionRef: participant.Ref,
		ExecutionAttempt: participant.AttemptNo, Operation: ControlStop, Target: ControlTargetExecution,
		Mode: ports.AgentStopCooperative, Reason: councilCleanupReason,
		GoalRevision: record.Goal.Revision(), PlanGeneration: record.Goal.PlanGeneration(),
		AppSpecGeneration: record.Goal.AppSpec().Generation(), SpecHash: record.Goal.SpecHash(),
		Status: ControlRequested, RequestedAt: at.UTC(), AuthorizationReceipt: authority.Authority,
	}
}

func (orchestrator *Orchestrator) councilCleanupStopAction(policy effectPolicySnapshot,
	control ControlRecord, aggregate goal.Goal, item goal.WorkItem, execution ExecutionRecord, at time.Time,
) (ActionRecord, error) {
	actionRef := "action:stop:" + control.Ref + ":" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: control.RequestRef,
		RequestFingerprint: effectAdmissionFingerprint(actionRef, control.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionStopAgent, Kind: EffectKindAgentStop,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: control.PrincipalRef,
		Permission: identity.PermissionGoalsCreate, Authority: control.AuthorizationReceipt,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortLow, PolicyHash: policy.PolicyHash,
		PolicyRevision: policy.PolicyRevision, QuotaRetryDelay: policy.QuotaRetryDelay,
		ApprovalTTL: policy.ApprovalTTL, TargetDigest: stopTargetDigest(control, stopRequest(control, execution)),
		IdempotencyKey: "stop:" + control.Ref + ":" + execution.Ref.String(), CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionStopAgent, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ControlRef: control.Ref, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AvailableAt: at.UTC(),
	}, EffectApprovalSourceGoalConfirmation, at)
}

func (orchestrator *Orchestrator) interruptAuthorForCouncilFailure(record GoalRecord, item goal.WorkItem, at time.Time) (
	goal.Goal, ExecutionRecord, error,
) {
	author, found := authorBound(record, item)
	if !found || author.State != ExecutionAwaitingIntegration {
		return goal.Goal{}, ExecutionRecord{}, errors.New("council.author_unavailable")
	}
	aggregate, err := record.Goal.InterruptWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), author.Ref,
		goal.WorkItemInterruptExecutionFailed, lifecycleTime(at, record.Goal, item),
	)
	if err != nil {
		return goal.Goal{}, ExecutionRecord{}, err
	}
	author.State, author.FailureCode, author.FinishedAt = ExecutionFailed, "council.unavailable", at.UTC()
	return aggregate, author, nil
}

func activeCouncilCleanupParticipant(executions []ExecutionRecord, stopped ExecutionRecord) bool {
	for _, participant := range executions {
		if participant.Ref == stopped.Ref || !isCouncilExecution(participant) ||
			participant.GoalRef != stopped.GoalRef || participant.WorkItemRef != stopped.WorkItemRef ||
			participant.CouncilSubjectDigest != stopped.CouncilSubjectDigest {
			continue
		}
		if participant.State == ExecutionDispatching || participant.State == ExecutionRunning {
			return true
		}
	}
	return false
}

func councilCleanupStillActive(record GoalRecord, current ExecutionRecord,
	retired []CouncilParticipantRetirement,
) bool {
	candidate := replaceExecution(record.Executions, current)
	for _, participant := range retired {
		candidate = replaceExecution(candidate, participant.Execution)
	}
	return activeCouncilCleanupParticipant(candidate, current)
}

func (orchestrator *Orchestrator) settleCouncilCleanupStopped(ctx context.Context, claim ActionClaim,
	record GoalRecord, item goal.WorkItem, execution ExecutionRecord, control ControlRecord,
	receipt ports.AgentStopReceipt, effectReceipt EffectReceipt,
) error {
	at := lifecycleTime(effectReceipt.ConfirmedAt, record.Goal, item)
	previous := execution.State
	execution.State, execution.FailureCode, execution.FinishedAt =
		ExecutionStopped, "council.round_aborted", at.UTC()
	aggregate, author := record.Goal, ExecutionRecord{}
	candidate := replaceExecution(record.Executions, execution)
	if !activeCouncilCleanupParticipant(candidate, execution) {
		var err error
		aggregate, author, err = orchestrator.interruptAuthorForCouncilFailure(record, item, at)
		if err != nil {
			return err
		}
	}
	control.Status, control.ConfirmedAt = ControlConfirmed, at
	control.ReceiptRef = receipt.ReceiptRef
	settlement, err := settlementFor(record, execution, unknownUsage(), 0, at)
	if err != nil {
		return err
	}
	updates := []ExecutionRecord{execution}
	if author.Ref.String() != "" {
		updates = append(updates, author)
	}
	events := []EventRecord{{
		Ref: "event:council-cleanup-stopped:" + execution.Ref.String(), Kind: "council.cleanup_stopped",
		GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
	}}
	if author.Ref.String() != "" {
		events = append(events, EventRecord{
			Ref: "event:council.unavailable:" + author.Ref.String(), Kind: "council.unavailable",
			GoalRef: author.GoalRef, WorkItemRef: author.WorkItemRef,
			ExecutionRef: author.Ref, OccurredAt: at.UTC(),
		})
	}
	_, _, err = orchestrator.state.ApplyControl(ctx, ApplyControlState{
		RequestRef: control.RequestRef, RequestFingerprint: control.RequestFingerprint,
		AuthorizationReceipt: control.AuthorizationReceipt, PrincipalRef: control.PrincipalRef,
		ProjectRef: control.ProjectRef, GoalRef: control.GoalRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedWorkItemRevision: item.Revision(), ExpectedExecutionState: previous,
		ExpectedControlStatus: ControlRequested, Claim: claim, Goal: aggregate,
		Executions: updates, RetireMailboxForExecutionRef: execution.Ref,
		EffectReceipt: &effectReceipt, BudgetSettlement: settlement,
		Events: events, Control: control, OperationAt: at,
	})
	return err
}

func (orchestrator *Orchestrator) resolveUnappliedCouncilCleanupLaunch(ctx context.Context,
	claim ActionClaim, record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	control ControlRecord, attemptRef string, at time.Time,
) error {
	at = lifecycleTime(at, record.Goal, item)
	expectedExecutionState := execution.State
	execution.State, execution.FailureCode, execution.FinishedAt =
		ExecutionFailed, "council.round_aborted", at.UTC()
	aggregate, author := record.Goal, ExecutionRecord{}
	if !councilCleanupStillActive(record, execution, nil) {
		var err error
		aggregate, author, err = orchestrator.interruptAuthorForCouncilFailure(record, item, at)
		if err != nil {
			return err
		}
	}
	resolved := control
	resolved.Status, resolved.ConfirmedAt = ControlConfirmed, at.UTC()
	resolved.ReceiptRef = "receipt:council-cleanup-local:" + fingerprintFields(
		"orquesta.council-cleanup-local.v1", control.Ref, execution.Ref.String(), attemptRef,
	)
	settlement, err := settlementForExecutionAttempt(
		record, claim, execution, unknownUsage(), 0, at, true,
	)
	if err != nil {
		return err
	}
	events := []EventRecord{{
		Ref:  "event:council-cleanup-launch-unapplied:" + execution.Ref.String(),
		Kind: "council.cleanup_launch_unapplied", GoalRef: execution.GoalRef,
		WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
	}}
	if author.Ref.String() != "" {
		events = append(events, EventRecord{
			Ref: "event:council.unavailable:" + author.Ref.String(), Kind: "council.unavailable",
			GoalRef: author.GoalRef, WorkItemRef: author.WorkItemRef,
			ExecutionRef: author.Ref, OccurredAt: at.UTC(),
		})
	}
	return orchestrator.state.RecordCouncilExecutionFailed(ctx, CouncilExecutionFailedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		ExpectedExecutionState: expectedExecutionState,
		Execution:              execution, Goal: aggregate, AuthorExecution: author, ResolvedCleanup: &resolved,
		BudgetSettlement: settlement, Events: events, OperationAt: at.UTC(),
	})
}
