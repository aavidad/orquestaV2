package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/governance"
)

func (orchestrator *Orchestrator) replaceCouncilExecution(ctx context.Context, claim ActionClaim, record GoalRecord,
	execution ExecutionRecord, code string, retryPolicy failedExecutionRetryPolicy, at time.Time,
	usage governance.ResourceUsage, diskBytes int64, definitelyUnapplied bool,
) error {
	item, found := record.Goal.WorkItem(execution.WorkItemRef)
	role, councilExecution := councilRole(execution)
	if !found || !councilExecution {
		return &StateError{Code: StateConflict}
	}
	at = lifecycleTime(at, record.Goal, item)
	execution.State, execution.FailureCode, execution.FinishedAt = ExecutionFailed, stableFailureCode(code), at.UTC()
	settlement, err := settlementForExecutionAttempt(record, claim, execution, usage, diskBytes, at, definitelyUnapplied)
	if err != nil {
		return err
	}
	switch retryPolicy {
	case failedExecutionMayRetry, failedExecutionMustTerminate:
	default:
		return errors.New("application.council_retry_policy_invalid")
	}
	if retryPolicy == failedExecutionMustTerminate || execution.AttemptNo >= execution.MaxExecutionAttempts {
		retired, actionRefs, cleanupControls, cleanupActions, cleanupEvents, cleanupErr :=
			orchestrator.councilCleanupPlan(record, item, execution.Ref, execution.CouncilSubjectDigest,
				claim.Action.Ref, at)
		if cleanupErr != nil {
			return cleanupErr
		}
		aggregate, author := record.Goal, ExecutionRecord{}
		if !councilCleanupStillActive(record, execution, retired) {
			aggregate, author, cleanupErr = orchestrator.interruptAuthorForCouncilFailure(record, item, at)
			if cleanupErr != nil {
				return cleanupErr
			}
		}
		events := append([]EventRecord{{Ref: "event:council-failed:" + execution.Ref.String(), Kind: "council.failed",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()}},
			cleanupEvents...)
		return orchestrator.state.RecordCouncilExecutionFailed(ctx, CouncilExecutionFailedState{Claim: claim,
			ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), Execution: execution,
			Goal: aggregate, AuthorExecution: author, RetiredPeers: retired, RetireActionRefs: actionRefs,
			CleanupControls: cleanupControls, CleanupActions: cleanupActions,
			BudgetSettlement: settlement, Events: events, OperationAt: at.UTC()})
	}
	replacement, err := councilReplacementExecution(execution, role, at)
	if err != nil {
		return err
	}
	authority, found := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !found {
		return errors.New("application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	round, found := councilRoundFor(record, execution.CouncilSubjectDigest)
	if !found {
		return &StateError{Code: StateConflict}
	}
	retryRecord := record
	retryRecord.Executions = append(append([]ExecutionRecord(nil), record.Executions...), replacement)
	next, err := orchestrator.councilLaunchAction(policy, retryRecord, item, replacement, authority, round.Subject, at,
		at.Add(executionRetryBackoff(orchestrator.observationDelay, execution.AttemptNo, orchestrator.executionTimeout)))
	if err != nil {
		return err
	}
	events := []EventRecord{
		{Ref: "event:council-attempt-failed:" + execution.Ref.String(), Kind: "council.attempt_failed", GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()},
		{Ref: "event:council-queued:" + replacement.Ref.String(), Kind: "council.queued", GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacement.Ref, OccurredAt: at.UTC()},
	}
	return orchestrator.state.RecordCouncilExecutionReplaced(ctx, CouncilExecutionReplacedState{Claim: claim,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), FailedExecution: execution,
		ReplacementExecution: replacement, NextAction: next, BudgetSettlement: settlement, Events: events, OperationAt: at.UTC()})
}

func councilReplacementExecution(execution ExecutionRecord, role council.Role, at time.Time) (ExecutionRecord, error) {
	ref, err := councilRetryRef(execution, role)
	if err != nil {
		return ExecutionRecord{}, err
	}
	return ExecutionRecord{Ref: ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, RepositoryRef: execution.RepositoryRef, State: ExecutionQueued, Purpose: execution.Purpose,
		CouncilSubjectDigest: execution.CouncilSubjectDigest, ExecutionWorkspaceRef: execution.ExecutionWorkspaceRef,
		ArtifactMediaType: council.ContributionMediaType, IdempotencyKey: "execution:" + ref.String(),
		MaxOutputBytes: execution.MaxOutputBytes, CreatedAt: at.UTC()}, nil
}
