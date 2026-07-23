package application

import (
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func (orchestrator *Orchestrator) applyDirectorProposal(
	current GoalRecord,
	request ProposeDirectorPlanRequest,
	plan goal.Plan,
	at time.Time,
) (goal.Goal, []ExecutionRecord, []string, []EventRecord, error) {
	if request.Cause == "" {
		updated, err := current.Goal.ApplyPlan(request.ExpectedGoalRevision, plan)
		return updated, nil, nil, nil, err
	}
	source, found := current.Goal.WorkItem(request.SourceWorkItemRef)
	if !found || source.Revision() != request.ExpectedWorkItemRevision {
		return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
	}
	execution, found := executionByRef(current.Executions, request.SourceExecutionRef)
	if !found || execution.WorkItemRef != source.Ref() ||
		execution.AttemptNo != request.SourceExecutionAttempt {
		return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
	}
	switch request.Cause {
	case goal.ReplanCauseSplitPending:
		if source.State() != goal.WorkItemStatePending || execution.State != ExecutionQueued {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
	case goal.ReplanCauseExecutionStopped:
		if execution.State != ExecutionStopped {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
	case goal.ReplanCauseExecutionFailed:
		if execution.State != ExecutionFailed {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
	case goal.ReplanCauseReviewChangesRequested:
		change, changeFound := changeForAuthor(current, execution)
		interrupt, hasInterrupt := source.InterruptCause()
		if execution.State != ExecutionFailed ||
			execution.FailureCode != string(goal.ReplanCauseReviewChangesRequested) || !changeFound ||
			!FailedReviewPreservesCandidate(current, source, execution, change) || !hasInterrupt ||
			interrupt != goal.WorkItemInterruptExecutionFailed {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
	case goal.ReplanCauseGovernanceDecision:
		if !councilGovernanceReplanValid(current, source, execution, request) {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
	default:
		return goal.Goal{}, nil, nil, nil, &StateError{Code: StateInvalid}
	}
	allItems := plan.WorkItems()
	existingCount := current.Goal.WorkItemCount()
	if len(allItems) <= existingCount {
		return goal.Goal{}, nil, nil, nil, &StateError{Code: StateInvalid}
	}
	updated, err := current.Goal.ApplyReplan(request.ExpectedGoalRevision, goal.ReplanInput{
		ExpectedPlanGeneration: request.ExpectedPlanGeneration,
		Source:                 request.SourceWorkItemRef, ExpectedSourceRevision: request.ExpectedWorkItemRevision,
		Cause: request.Cause, CausalExecution: request.SourceExecutionRef,
		Successors: allItems[existingCount:], At: at,
	})
	if err != nil {
		return goal.Goal{}, nil, nil, nil, err
	}
	if request.Cause == goal.ReplanCauseGovernanceDecision {
		change, changeFound := changeForAuthor(current, execution)
		if !changeFound {
			return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
		}
		for _, intent := range current.EffectIntents {
			if intent.ActionKind == ActionIntegrateChange && intent.Subject.ExecutionRef == execution.Ref {
				return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
			}
		}
		for _, receipt := range current.IntegrationReceipts {
			if receipt.ChangeRef == change.Ref {
				return goal.Goal{}, nil, nil, nil, &StateError{Code: StateConflict}
			}
		}
		execution.State, execution.FailureCode, execution.FinishedAt = ExecutionCanceled, "application.execution_superseded", at.UTC()
		return updated, []ExecutionRecord{execution}, nil, []EventRecord{{
			Ref: "event:execution-canceled:" + execution.Ref.String(), Kind: "execution.canceled",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
		}}, nil
	}
	if request.Cause != goal.ReplanCauseSplitPending {
		return updated, nil, nil, nil, nil
	}
	execution.State = ExecutionCanceled
	execution.FailureCode = "application.execution_superseded"
	execution.FinishedAt = at.UTC()
	event := EventRecord{
		Ref: "event:execution-canceled:" + execution.Ref.String(), Kind: "execution.canceled",
		GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, OccurredAt: at,
	}
	actionRef := "action:launch:" + execution.Ref.String()
	if len(source.WriteSet()) != 0 {
		actionRef = "action:prepare-workspace:" + execution.Ref.String()
	}
	return updated, []ExecutionRecord{execution}, []string{actionRef}, []EventRecord{event}, nil
}

func councilGovernanceReplanValid(record GoalRecord, source goal.WorkItem, author ExecutionRecord,
	request ProposeDirectorPlanRequest,
) bool {
	bound, boundOK := source.Execution()
	if source.State() != goal.WorkItemStateRunning || len(source.WriteSet()) == 0 || author.Purpose != ExecutionPurposeAuthor ||
		!boundOK || bound != author.Ref || author.State != ExecutionAwaitingIntegration || author.Ref != request.SourceExecutionRef ||
		!validCouncilDigest(string(request.CouncilSubjectDigest)) || !validCouncilDigest(string(request.CouncilDecisionDigest)) {
		return false
	}
	round, found := councilRoundFor(record, request.CouncilSubjectDigest)
	if !found || round.SubjectDigest != request.CouncilSubjectDigest || round.Subject.GoalRef != record.Goal.Ref().String() ||
		round.Subject.WorkItemRef != source.Ref().String() || round.Subject.SpecHash != author.SpecHash {
		return false
	}
	change, changeFound := changeForAuthor(record, author)
	if !changeFound || round.Subject.ChangeSetRef != change.Ref.String() {
		return false
	}
	for _, decision := range record.CouncilDecisions {
		if decision.Ref != request.CouncilDecisionRef || decision.SubjectDigest != request.CouncilSubjectDigest ||
			decision.DecisionDigest != request.CouncilDecisionDigest || decision.Decision.SubjectDigest != string(request.CouncilSubjectDigest) ||
			decision.Decision.Digest != string(request.CouncilDecisionDigest) {
			continue
		}
		switch decision.Decision.Outcome {
		case council.OutcomeRejected, council.OutcomeNoConsensus, council.OutcomeBlockedSecurity:
			return true
		}
	}
	return false
}

func executionByRef(records []ExecutionRecord, ref goal.ExecutionRef) (ExecutionRecord, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return ExecutionRecord{}, false
}
