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

const reviewCleanupReason = "review.round_cleanup"

func IsReviewCleanupControl(control ControlRecord) bool {
	return strings.HasPrefix(control.RequestRef, "request:review-cleanup:") &&
		strings.HasPrefix(control.Ref, "control:review-cleanup:") &&
		control.Operation == ControlStop && control.Target == ControlTargetExecution &&
		control.Mode == ports.AgentStopCooperative && control.Reason == reviewCleanupReason
}

func pendingReviewCleanupControl(record GoalRecord, execution ExecutionRecord) (ControlRecord, bool) {
	for _, control := range record.Controls {
		if IsReviewCleanupControl(control) && control.Status == ControlRequested &&
			control.GoalRef == execution.GoalRef && control.WorkItemRef == execution.WorkItemRef &&
			control.ExecutionRef == execution.Ref && control.ExecutionAttempt == execution.AttemptNo {
			return control, true
		}
	}
	return ControlRecord{}, false
}

func isReviewCleanupStopAction(action ActionRecord) bool {
	return action.Kind == ActionStopAgent && strings.HasPrefix(action.ControlRef, "control:review-cleanup:") &&
		action.EffectIntent.ActionKind == ActionStopAgent && action.EffectIntent.Kind == EffectKindAgentStop &&
		action.EffectIntent.Permission == identity.PermissionGoalsCreate
}

func reviewCleanupAuthority(record GoalRecord, author ExecutionRecord) (EffectIntent, error) {
	for _, intent := range record.EffectIntents {
		if intent.ActionKind == ActionLaunchAgent && intent.Kind == EffectKindAgentLaunch &&
			intent.Subject.GoalRef == author.GoalRef && intent.Subject.WorkItemRef == author.WorkItemRef &&
			intent.Subject.ExecutionRef == author.Ref && intent.Permission == identity.PermissionGoalsCreate {
			return intent, nil
		}
	}
	return EffectIntent{}, errors.New("review.cleanup_authority_missing")
}

func (orchestrator *Orchestrator) reviewCleanupPlan(record GoalRecord, item goal.WorkItem,
	exclude goal.ExecutionRef, subjectDigest, triggerRef string, at time.Time,
) ([]ReviewParticipantRetirement, []string, []ControlRecord, []ActionRecord, []EventRecord, error) {
	author, found := authorBound(record, item)
	if !found {
		return nil, nil, nil, nil, nil, errors.New("review.cleanup_author_missing")
	}
	authority, err := reviewCleanupAuthority(record, author)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	retired := make([]ReviewParticipantRetirement, 0, 1)
	retireActions := make([]string, 0, 1)
	controls := make([]ControlRecord, 0, 1)
	actions := make([]ActionRecord, 0, 1)
	events := make([]EventRecord, 0, 2)
	for _, participant := range record.Executions {
		if participant.Ref == exclude || !isReviewerExecution(participant) ||
			participant.GoalRef != record.Goal.Ref() || participant.WorkItemRef != item.Ref() ||
			(subjectDigest != "" && participant.ReviewSubjectDigest != subjectDigest) {
			continue
		}
		switch participant.State {
		case ExecutionQueued:
			participant.State, participant.FailureCode, participant.FinishedAt =
				ExecutionFailed, "review.round_aborted", at.UTC()
			retired = append(retired, ReviewParticipantRetirement{
				Execution: participant, ExpectedState: ExecutionQueued,
			})
			retireActions = append(retireActions, "action:launch:"+participant.Ref.String())
			events = append(events, EventRecord{
				Ref:  "event:review-round-retired:" + participant.Ref.String(),
				Kind: "review.round_participant_retired", GoalRef: participant.GoalRef,
				WorkItemRef: participant.WorkItemRef, ExecutionRef: participant.Ref, OccurredAt: at.UTC(),
			})
		case ExecutionDispatching, ExecutionRunning:
			if _, pending := pendingReviewCleanupControl(record, participant); pending {
				continue
			}
			control := newReviewCleanupControl(record, item, participant, authority, triggerRef, at)
			controls = append(controls, control)
			if participant.State == ExecutionRunning {
				action, actionErr := orchestrator.reviewCleanupStopAction(
					policy, control, record.Goal, item, participant, at,
				)
				if actionErr != nil {
					return nil, nil, nil, nil, nil, actionErr
				}
				actions = append(actions, action)
				retireActions = append(retireActions, "action:observe:"+participant.Ref.String())
			}
			events = append(events, EventRecord{
				Ref:  "event:review-cleanup-requested:" + participant.Ref.String(),
				Kind: "review.cleanup_requested", GoalRef: participant.GoalRef,
				WorkItemRef: participant.WorkItemRef, ExecutionRef: participant.Ref, OccurredAt: at.UTC(),
			})
		}
	}
	return retired, retireActions, controls, actions, events, nil
}

func newReviewCleanupControl(record GoalRecord, item goal.WorkItem, participant ExecutionRecord,
	authority EffectIntent, triggerRef string, at time.Time,
) ControlRecord {
	digest := fingerprintFields("orquesta.review-cleanup.v1", triggerRef, participant.Ref.String(),
		participant.ReviewSubjectDigest)
	request := ControlRequest{
		RequestRef: "request:review-cleanup:" + digest, Operation: ControlStop,
		Target: ControlTargetExecution, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: participant.Ref, ExpectedExecutionAttempt: participant.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: reviewCleanupReason,
	}
	return ControlRecord{
		Ref: "control:review-cleanup:" + digest, RequestRef: request.RequestRef,
		RequestFingerprint: controlFingerprint(authority.ProposedBy, record.Goal.Project(), request),
		PrincipalRef:       authority.ProposedBy, ProjectRef: record.Goal.Project(), GoalRef: record.Goal.Ref(),
		WorkItemRef: item.Ref(), WorkItemRevision: item.Revision(), ExecutionRef: participant.Ref,
		ExecutionAttempt: participant.AttemptNo, Operation: ControlStop, Target: ControlTargetExecution,
		Mode: ports.AgentStopCooperative, Reason: reviewCleanupReason,
		GoalRevision: record.Goal.Revision(), PlanGeneration: record.Goal.PlanGeneration(),
		AppSpecGeneration: record.Goal.AppSpec().Generation(), SpecHash: record.Goal.SpecHash(),
		Status: ControlRequested, RequestedAt: at.UTC(), AuthorizationReceipt: authority.Authority,
	}
}

func (orchestrator *Orchestrator) reviewCleanupStopAction(policy effectPolicySnapshot,
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

func (orchestrator *Orchestrator) settleReviewCleanupStopped(ctx context.Context, claim ActionClaim,
	record GoalRecord, item goal.WorkItem, execution ExecutionRecord, control ControlRecord,
	receipt ports.AgentStopReceipt, effectReceipt EffectReceipt,
) error {
	at := lifecycleTime(effectReceipt.ConfirmedAt, record.Goal, item)
	previous := execution.State
	execution.State, execution.FailureCode, execution.FinishedAt =
		ExecutionStopped, "review.round_aborted", at.UTC()
	aggregate, author := record.Goal, ExecutionRecord{}
	candidate := replaceExecution(record.Executions, execution)
	if !activeReviewCleanupParticipant(candidate, execution) {
		var err error
		aggregate, author, err = orchestrator.interruptAuthorForReviewFailure(
			record, item, at, "review.unavailable",
		)
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
		Ref: "event:review-cleanup-stopped:" + execution.Ref.String(), Kind: "review.cleanup_stopped",
		GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
	}}
	if author.Ref.String() != "" {
		events = append(events, EventRecord{
			Ref: "event:" + author.FailureCode + ":" + author.Ref.String(), Kind: author.FailureCode,
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

func activeReviewCleanupParticipant(executions []ExecutionRecord, stopped ExecutionRecord) bool {
	for _, participant := range executions {
		if participant.Ref == stopped.Ref || !isReviewerExecution(participant) ||
			participant.GoalRef != stopped.GoalRef || participant.WorkItemRef != stopped.WorkItemRef ||
			participant.ReviewSubjectDigest != stopped.ReviewSubjectDigest {
			continue
		}
		if participant.State == ExecutionDispatching || participant.State == ExecutionRunning {
			return true
		}
	}
	return false
}

func reviewCleanupStillActive(record GoalRecord, current ExecutionRecord,
	retired []ReviewParticipantRetirement,
) bool {
	candidate := replaceExecution(record.Executions, current)
	for _, participant := range retired {
		candidate = replaceExecution(candidate, participant.Execution)
	}
	return activeReviewCleanupParticipant(candidate, current)
}

func confirmedLocalReviewCleanup(control ControlRecord, execution ExecutionRecord,
	at time.Time, suffix string,
) ControlRecord {
	control.Status, control.ConfirmedAt = ControlConfirmed, at.UTC()
	control.ReceiptRef = "receipt:review-cleanup-local:" + fingerprintFields(
		"orquesta.review-cleanup-local.v1", control.Ref, execution.Ref.String(), suffix,
	)
	return control
}

func (orchestrator *Orchestrator) resolveUnappliedReviewCleanupLaunch(ctx context.Context,
	claim ActionClaim, record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	control ControlRecord, attemptRef string, at time.Time,
) error {
	at = lifecycleTime(at, record.Goal, item)
	execution.State, execution.FailureCode, execution.FinishedAt =
		ExecutionFailed, "review.round_aborted", at.UTC()
	aggregate, author := record.Goal, ExecutionRecord{}
	if !reviewCleanupStillActive(record, execution, nil) {
		var err error
		aggregate, author, err = orchestrator.interruptAuthorForReviewFailure(record, item, at, "review.unavailable")
		if err != nil {
			return err
		}
	}
	resolved := confirmedLocalReviewCleanup(control, execution, at, attemptRef)
	settlement, err := settlementForExecutionAttempt(
		record, claim, execution, unknownUsage(), 0, at, true,
	)
	if err != nil {
		return err
	}
	events := []EventRecord{{
		Ref:  "event:review-cleanup-launch-unapplied:" + execution.Ref.String(),
		Kind: "review.cleanup_launch_unapplied", GoalRef: execution.GoalRef,
		WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
	}}
	if author.Ref.String() != "" {
		events = append(events, EventRecord{
			Ref: "event:review.unavailable:" + author.Ref.String(), Kind: "review.unavailable",
			GoalRef: author.GoalRef, WorkItemRef: author.WorkItemRef,
			ExecutionRef: author.Ref, OccurredAt: at.UTC(),
		})
	}
	return orchestrator.state.RecordReviewExecutionFailed(ctx, ReviewExecutionFailedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Execution: execution, Goal: aggregate, AuthorExecution: author, ResolvedCleanup: &resolved,
		BudgetSettlement: settlement, Events: events, OperationAt: at.UTC(),
	})
}
