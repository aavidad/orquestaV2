package goal

import (
	"math"
	"time"
)

type ReplanCause string

const (
	ReplanCauseSplitPending           ReplanCause = "split_pending"
	ReplanCauseExecutionStopped       ReplanCause = "execution_stopped"
	ReplanCauseExecutionFailed        ReplanCause = "execution_failed"
	ReplanCauseReviewChangesRequested ReplanCause = "review_changes_requested"
)

type ReplanInput struct {
	ExpectedPlanGeneration PlanGeneration
	Source                 WorkItemRef
	ExpectedSourceRevision Revision
	Cause                  ReplanCause
	CausalExecution        ExecutionRef
	Successors             []WorkItem
	At                     time.Time
}

// ApplyReplan is append-only 1:N. Application verifies Director lease/fence;
// domain additionally fences Goal, plan generation and source revision.
func (goal Goal) ApplyReplan(expected Revision, input ReplanInput) (Goal, error) {
	if goal.state != GoalStateRunning || goal.cancelRequested || input.ExpectedPlanGeneration != goal.planGeneration {
		return Goal{}, domainError(ErrorInvalidTransition, "replan_fence")
	}
	source, err := goal.controlWorkItem(expected, input.ExpectedSourceRevision, input.Source)
	if err != nil {
		return Goal{}, err
	}
	if !goal.validReplanSource(source, input) || reworkSourceTouchesHandoffIn(source.ref, goal.items) ||
		goal.hasSkippedDependencyDescendant(source.ref) || len(input.Successors) == 0 {
		return Goal{}, domainError(ErrorInvalidPlan, "replan_source")
	}
	next, err := nextPlanGeneration(goal.planGeneration)
	if err != nil {
		return Goal{}, err
	}
	updated, err := goal.beginControl(expected, input.At, false)
	if err != nil {
		return Goal{}, err
	}
	updated.planGeneration, updated.items[source.ref] = next, source.superseded(input.At)
	runningScopes := make([][]WriteScope, 0)
	for _, item := range goal.items {
		if item.state == WorkItemStateRunning {
			runningScopes = append(runningScopes, item.writeSet)
		}
	}
	seen := map[WorkItemRef]struct{}{}
	successorScopes := make([][]WriteScope, 0, len(input.Successors))
	for _, successor := range input.Successors {
		if err := goal.validateReplanSuccessor(successor, input.At, seen, runningScopes, successorScopes); err != nil {
			return Goal{}, err
		}
		successor = successor.clone()
		successor.reworkOf = source.ref
		updated.items[successor.ref] = successor
		updated.itemOrder = append(updated.itemOrder, successor.ref)
		seen[successor.ref] = struct{}{}
		successorScopes = append(successorScopes, successor.writeSet)
	}
	if err := validateRestoredPlan(Plan{generation: next, phases: updated.phases, items: updated.WorkItems()}); err != nil {
		return Goal{}, err
	}
	return updated, nil
}

func (goal Goal) validReplanSource(source WorkItem, input ReplanInput) bool {
	if source.controlSequence == math.MaxUint64 {
		return false
	}
	switch input.Cause {
	case ReplanCauseSplitPending:
		// The aggregate has no binding for a queued execution. Application owns
		// and validates that exact projection fence when one exists.
		return source.state == WorkItemStatePending
	case ReplanCauseExecutionStopped:
		return source.state == WorkItemStateInterrupted && source.interruptCause == WorkItemInterruptExecutionStopped &&
			source.execution == input.CausalExecution && validExecutionRef(input.CausalExecution)
	case ReplanCauseExecutionFailed:
		return source.state == WorkItemStateInterrupted && source.interruptCause == WorkItemInterruptExecutionFailed &&
			source.execution == input.CausalExecution && validExecutionRef(input.CausalExecution)
	case ReplanCauseReviewChangesRequested:
		return source.state == WorkItemStateInterrupted && source.execution == input.CausalExecution && validExecutionRef(input.CausalExecution)
	}
	return false
}

func (goal Goal) validateReplanSuccessor(item WorkItem, at time.Time, seen map[WorkItemRef]struct{}, running, successors [][]WriteScope) error {
	_, duplicate := seen[item.ref]
	_, exists := goal.items[item.ref]
	if !validWorkItemRef(item.ref) || duplicate || exists || item.state != WorkItemStatePending || item.revision != 1 ||
		item.goal != goal.ref || item.actor != goal.actor || item.project != goal.project || item.paused || item.cancelRequested ||
		item.controlSequence != 0 || item.interruptCause != "" || validWorkItemRef(item.reworkOf) || validExecutionRef(item.execution) ||
		!item.startedAt.IsZero() || !item.interruptedAt.IsZero() || !item.finishedAt.IsZero() ||
		len(item.artifacts) != 0 || len(item.attestations) != 0 || item.createdAt.Before(goal.createdAt) || item.createdAt.After(canonicalTime(at)) {
		return domainError(ErrorInvalidPlan, "replan_successor")
	}
	phaseFound := false
	for _, phase := range goal.phases {
		phaseFound = phaseFound || phase.key == item.phase
	}
	if !phaseFound || conflictsWithWriteSets(item.writeSet, running) || conflictsWithWriteSets(item.writeSet, successors) {
		return domainError(ErrorInvalidPlan, "replan_write_set")
	}
	return validateWorkItemPlanMetadata(item)
}

func (goal Goal) hasSkippedDependencyDescendant(source WorkItemRef) bool {
	seen, queue := map[WorkItemRef]bool{source: true}, []WorkItemRef{source}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, item := range goal.items {
			if seen[item.ref] || !containsWorkItemRef(item.dependencies, current) {
				continue
			}
			if item.state == WorkItemStateSkipped {
				return true
			}
			seen[item.ref], queue = true, append(queue, item.ref)
		}
	}
	return false
}

func (item WorkItem) superseded(at time.Time) WorkItem {
	item.state, item.paused, item.finishedAt = WorkItemStateSuperseded, false, canonicalTime(at)
	item.revision, item.controlSequence = item.revision+1, item.controlSequence+1
	return item
}

func containsWorkItemRef(refs []WorkItemRef, candidate WorkItemRef) bool {
	for _, ref := range refs {
		if ref == candidate {
			return true
		}
	}
	return false
}
