package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) interruptAuthorForReviewFailure(record GoalRecord, item goal.WorkItem, at time.Time, code string) (
	goal.Goal, ExecutionRecord, error,
) {
	author, found := authorBound(record, item)
	if !found || author.State != ExecutionAwaitingIntegration {
		return goal.Goal{}, ExecutionRecord{}, errors.New("review.author_unavailable")
	}
	aggregate, err := record.Goal.InterruptWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), author.Ref,
		goal.WorkItemInterruptExecutionFailed, lifecycleTime(at, record.Goal, item),
	)
	if err != nil {
		return goal.Goal{}, ExecutionRecord{}, err
	}
	author.State, author.FailureCode, author.FinishedAt = ExecutionFailed, code, at.UTC()
	return aggregate, author, nil
}

func (orchestrator *Orchestrator) abortReviewerLaunch(ctx context.Context, claim ActionClaim,
	record GoalRecord, execution ExecutionRecord, attempt EffectAttempt, receipt ports.AgentLaunchReceipt,
) error {
	item, found := record.Goal.WorkItem(execution.WorkItemRef)
	if !found || execution.State != ExecutionDispatching || !isReviewerExecution(execution) {
		return &StateError{Code: StateConflict}
	}
	at := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.State, execution.FailureCode, execution.FinishedAt =
		ExecutionFailed, "review.external_ref_reused", at.UTC()
	retired, actionRefs, cleanupControls, cleanupActions, cleanupEvents, err :=
		orchestrator.reviewCleanupPlan(record, item, execution.Ref, execution.ReviewSubjectDigest,
			claim.Action.Ref, at)
	if err != nil {
		return err
	}
	var resolvedCleanup *ControlRecord
	if pending, found := pendingReviewCleanupControl(record, execution); found {
		resolved := confirmedLocalReviewCleanup(pending, execution, at, attempt.Ref)
		resolvedCleanup = &resolved
	}
	aggregate, author := record.Goal, ExecutionRecord{}
	if !reviewCleanupStillActive(record, execution, retired) {
		aggregate, author, err = orchestrator.interruptAuthorForReviewFailure(
			record, item, at, "review.unavailable",
		)
		if err != nil {
			return err
		}
	}
	settlement, err := settlementForExecutionAttempt(
		record, claim, execution, unknownUsage(), 0, at, false,
	)
	if err != nil {
		return err
	}
	externalReceipt, err := effectReceipt(
		claim, attempt, receipt.ReceiptRef, EffectStatusAccepted, unknownUsage(), at,
	)
	if err != nil {
		return err
	}
	events := append([]EventRecord{{
		Ref: "event:review-external-ref-reused:" + execution.Ref.String(), Kind: "review.external_ref_reused",
		GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
	}}, cleanupEvents...)
	return orchestrator.state.RecordReviewExecutionFailed(ctx, ReviewExecutionFailedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Execution: execution, Goal: aggregate, AuthorExecution: author,
		RetiredReviewers: retired, RetireActionRefs: actionRefs,
		CleanupControls: cleanupControls, CleanupActions: cleanupActions,
		ResolvedCleanup: resolvedCleanup,
		EffectReceipt:   &externalReceipt, QuarantineClaim: true, BudgetSettlement: settlement,
		Events: events, OperationAt: at.UTC(),
	})
}
