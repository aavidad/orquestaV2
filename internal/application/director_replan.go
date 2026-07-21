package application

import (
	"time"

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

func executionByRef(records []ExecutionRecord, ref goal.ExecutionRef) (ExecutionRecord, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return ExecutionRecord{}, false
}
