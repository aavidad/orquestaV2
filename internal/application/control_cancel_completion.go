package application

import (
	"time"

	"orquesta/internal/goal"
)

// completeCanceledWorkItem terminalizes a staged author only after every
// provider-owned execution in the WorkItem has reached a terminal state.
// Passive actions are retired when cancel is requested, so this transition
// cannot commit, attest or integrate the preserved candidate afterwards.
func completeCanceledWorkItem(
	aggregate goal.Goal,
	item goal.WorkItem,
	executions []ExecutionRecord,
	at time.Time,
) (goal.Goal, []ExecutionRecord, error) {
	if activeExecutionInItem(executions, item.Ref()) {
		return aggregate, nil, nil
	}
	current, found := aggregate.WorkItem(item.Ref())
	if !found || current.State() != goal.WorkItemStateRunning || !current.CancelRequested() {
		return aggregate, nil, nil
	}

	var updates []ExecutionRecord
	if bound, hasBinding := current.Execution(); hasBinding {
		author, authorFound := executionByRef(executions, bound)
		if authorFound &&
			(author.State == ExecutionAwaitingCommit ||
				author.State == ExecutionAwaitingAttestation ||
				author.State == ExecutionAwaitingIntegration) {
			author.State = ExecutionCanceled
			author.FailureCode = "application.execution_canceled"
			author.FinishedAt = at.UTC()
			updates = append(updates, author)
		}
	}

	completed, err := aggregate.CompleteWorkItemCancel(
		aggregate.Revision(), current.Revision(), current.Ref(), at.UTC(),
	)
	return completed, updates, err
}

func canceledExecutionEvents(executions []ExecutionRecord, at time.Time) []EventRecord {
	events := make([]EventRecord, 0, len(executions))
	for _, execution := range executions {
		if execution.State != ExecutionCanceled {
			continue
		}
		events = append(events, EventRecord{
			Ref: "event:execution-canceled:" + execution.Ref.String(), Kind: "execution.canceled",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
		})
	}
	return events
}
