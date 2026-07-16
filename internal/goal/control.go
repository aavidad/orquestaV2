package goal

import (
	"math"
	"time"
)

type WorkItemLogicalOutcome string

const (
	WorkItemLogicalSucceeded WorkItemLogicalOutcome = "succeeded"
	WorkItemLogicalFailed    WorkItemLogicalOutcome = "failed"
)

func (goal Goal) Paused() bool            { return goal.paused }
func (goal Goal) CancelRequested() bool   { return goal.cancelRequested }
func (goal Goal) ControlSequence() uint64 { return goal.controlSequence }

func (goal Goal) EffectivePause(ref WorkItemRef) (bool, bool) {
	item, found := goal.items[ref]
	return goal.paused || item.paused, found
}

// AdvanceControl records execution-only control. Other operations advance it themselves.
func (goal Goal) AdvanceControl(expected Revision, at time.Time) (Goal, error) {
	return goal.beginControl(expected, at, false)
}

func (goal Goal) SetPaused(expected Revision, paused bool, at time.Time) (Goal, error) {
	if goal.cancelRequested || goal.paused == paused {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_pause")
	}
	updated, err := goal.beginControl(expected, at, true)
	if err == nil {
		updated.paused = paused
	}
	return updated, err
}

func (goal Goal) SetWorkItemPaused(expectedGoal, expectedItem Revision, ref WorkItemRef, paused bool, at time.Time) (Goal, error) {
	item, err := goal.controlWorkItem(expectedGoal, expectedItem, ref)
	if err != nil {
		return Goal{}, err
	}
	if goal.cancelRequested || item.cancelRequested || item.IsTerminal() || item.paused == paused || item.controlSequence == math.MaxUint64 {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_pause")
	}
	updated, err := goal.beginControl(expectedGoal, at, true)
	if err != nil {
		return Goal{}, err
	}
	item.paused, item.revision, item.controlSequence = paused, item.revision+1, item.controlSequence+1
	updated.items[ref] = item
	return updated, nil
}

// RequestCancel retires local work; running executions remain in-flight until settled.
func (goal Goal) RequestCancel(expected Revision, at time.Time) (Goal, error) {
	if goal.cancelRequested {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_cancel_requested")
	}
	for _, item := range goal.items {
		if item.cancelRequested && !item.IsTerminal() {
			return Goal{}, domainError(ErrorInvalidTransition, "work_item_cancel_pending")
		}
	}
	updated, err := goal.beginControl(expected, at, true)
	if err != nil {
		return Goal{}, err
	}
	updated.cancelRequested, updated.paused = true, false
	for _, ref := range updated.itemOrder {
		item := updated.items[ref]
		if item.IsTerminal() {
			continue
		}
		if item, err = item.requestCancel(at); err != nil {
			return Goal{}, err
		}
		updated.items[ref] = item
	}
	return updated, nil
}

func (goal Goal) CompleteCancel(expected Revision, at time.Time) (Goal, error) {
	if !goal.cancelRequested || goal.IsTerminal() {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_cancel_requested")
	}
	for _, item := range goal.items {
		if !item.IsTerminal() {
			return Goal{}, domainError(ErrorWorkItemsNotTerminal, "work_items")
		}
		if item.finishedAt.After(at) {
			return Goal{}, domainError(ErrorInvalidArgument, "closed_at")
		}
	}
	updated, err := goal.beginControl(expected, at, true)
	if err == nil {
		updated.state, updated.closedAt = GoalStateCanceled, canonicalTime(at)
	}
	return updated, err
}

func (goal Goal) RequestWorkItemCancel(expectedGoal, expectedItem Revision, ref WorkItemRef, at time.Time) (Goal, error) {
	item, err := goal.controlWorkItem(expectedGoal, expectedItem, ref)
	if err != nil {
		return Goal{}, err
	}
	if goal.cancelRequested || item.cancelRequested || item.IsTerminal() {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_cancel_requested")
	}
	updated, err := goal.beginControl(expectedGoal, at, true)
	if err != nil {
		return Goal{}, err
	}
	if item, err = item.requestCancel(at); err != nil {
		return Goal{}, err
	}
	updated.items[ref] = item
	if item.state == WorkItemStateCanceled {
		err = updated.cascadeDependencySkips(at)
	}
	return updated, err
}

func (goal Goal) CompleteWorkItemCancel(expectedGoal, expectedItem Revision, ref WorkItemRef, at time.Time) (Goal, error) {
	item, err := goal.controlWorkItem(expectedGoal, expectedItem, ref)
	if err != nil {
		return Goal{}, err
	}
	if !item.cancelRequested || item.state != WorkItemStateRunning {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_cancel_requested")
	}
	updated, err := goal.beginControl(expectedGoal, at, true)
	if err != nil {
		return Goal{}, err
	}
	if item, err = item.completeCancel(at); err != nil {
		return Goal{}, err
	}
	updated.items[ref] = item
	err = updated.cascadeDependencySkips(at)
	return updated, err
}

func (goal Goal) InterruptWorkItem(expectedGoal, expectedItem Revision, ref WorkItemRef, execution ExecutionRef, cause WorkItemInterruptCause, at time.Time) (Goal, error) {
	item, err := goal.controlWorkItem(expectedGoal, expectedItem, ref)
	if err != nil {
		return Goal{}, err
	}
	if goal.cancelRequested || item.cancelRequested || item.state != WorkItemStateRunning || item.execution != execution ||
		!validInterruptCause(cause) || item.controlSequence == math.MaxUint64 {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_interrupt")
	}
	updated, err := goal.beginControl(expectedGoal, at, false)
	if err != nil {
		return Goal{}, err
	}
	item.state, item.interruptCause, item.interruptedAt = WorkItemStateInterrupted, cause, canonicalTime(at)
	item.revision, item.controlSequence = item.revision+1, item.controlSequence+1
	updated.items[ref] = item
	return updated, nil
}

func (goal Goal) RetryWorkItem(expectedGoal, expectedItem Revision, ref WorkItemRef, current, replacement ExecutionRef, at time.Time) (Goal, error) {
	item, err := goal.controlWorkItem(expectedGoal, expectedItem, ref)
	if err != nil {
		return Goal{}, err
	}
	if goal.cancelRequested || item.cancelRequested || item.state != WorkItemStateInterrupted || item.execution != current ||
		!validExecutionRef(replacement) || replacement == current || item.controlSequence == math.MaxUint64 {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_retry")
	}
	updated, err := goal.beginControl(expectedGoal, at, false)
	if err != nil {
		return Goal{}, err
	}
	item.state, item.execution, item.interruptCause, item.interruptedAt = WorkItemStateRunning, replacement, "", time.Time{}
	item.revision, item.controlSequence = item.revision+1, item.controlSequence+1
	updated.items[ref] = item
	return updated, nil
}

func (goal Goal) LogicalWorkItemOutcome(ref WorkItemRef) (WorkItemLogicalOutcome, bool) {
	if _, found := goal.items[ref]; !found {
		return "", false
	}
	return logicalWorkItemOutcomeIn(goal.items, ref, map[WorkItemRef]bool{})
}

func (goal Goal) beginControl(expected Revision, at time.Time, allowPending bool) (Goal, error) {
	if err := goal.expectRevision(expected); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStateRunning && !(allowPending && goal.state == GoalStatePending) {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	notBefore := goal.startedAt
	if notBefore.IsZero() {
		notBefore = goal.createdAt
	}
	if !validTransitionTime(at, notBefore) || goal.controlSequence == math.MaxUint64 {
		return Goal{}, domainError(ErrorInvalidArgument, "control_at")
	}
	updated := goal.clone()
	updated.revision, updated.controlSequence = updated.revision+1, updated.controlSequence+1
	return updated, nil
}

func (goal Goal) controlWorkItem(expectedGoal, expectedItem Revision, ref WorkItemRef) (WorkItem, error) {
	if err := goal.expectRevision(expectedGoal); err != nil {
		return WorkItem{}, err
	}
	item, found := goal.items[ref]
	if !found {
		return WorkItem{}, domainError(ErrorWorkItemNotFound, "work_item_ref")
	}
	if err := item.expectRevision(expectedItem); err != nil {
		return WorkItem{}, err
	}
	return item.clone(), nil
}

func (item WorkItem) requestCancel(at time.Time) (WorkItem, error) {
	notBefore := item.startedAt
	if notBefore.IsZero() {
		notBefore = item.createdAt
	}
	if !validTransitionTime(at, notBefore) || item.controlSequence == math.MaxUint64 {
		return WorkItem{}, domainError(ErrorInvalidArgument, "canceled_at")
	}
	item.cancelRequested, item.paused = true, false
	item.revision, item.controlSequence = item.revision+1, item.controlSequence+1
	if item.state == WorkItemStatePending || item.state == WorkItemStateInterrupted {
		item.state, item.finishedAt = WorkItemStateCanceled, canonicalTime(at)
	}
	return item, nil
}

func (item WorkItem) completeCancel(at time.Time) (WorkItem, error) {
	if !validTransitionTime(at, item.startedAt) || item.controlSequence == math.MaxUint64 {
		return WorkItem{}, domainError(ErrorInvalidArgument, "canceled_at")
	}
	item.state, item.finishedAt, item.paused = WorkItemStateCanceled, canonicalTime(at), false
	item.revision, item.controlSequence = item.revision+1, item.controlSequence+1
	return item, nil
}

func (goal *Goal) cascadeDependencySkips(at time.Time) error {
	for changed := true; changed; {
		changed = false
		for _, ref := range goal.itemOrder {
			item := goal.items[ref]
			reason, blocked := dependencySkipReasonIn(goal.items, item)
			if item.state != WorkItemStatePending || !blocked {
				continue
			}
			updated, err := item.skipDependency(item.revision, reason, at)
			if err != nil {
				return err
			}
			goal.items[ref], changed = updated, true
		}
	}
	return nil
}

func dependencySkipReasonIn(items map[WorkItemRef]WorkItem, item WorkItem) (WorkItemSkipReason, bool) {
	failed := false
	for _, ref := range item.dependencies {
		outcome, resolved := logicalWorkItemOutcomeIn(items, ref, map[WorkItemRef]bool{})
		if !resolved || outcome != WorkItemLogicalFailed {
			continue
		}
		dependency := items[ref]
		if dependency.state == WorkItemStateCanceled ||
			(dependency.state == WorkItemStateSkipped && dependency.skipReason == WorkItemSkipReasonDependencyCanceled) {
			return WorkItemSkipReasonDependencyCanceled, true
		}
		failed = true
	}
	return WorkItemSkipReasonDependencyFailed, failed
}

func logicalWorkItemOutcomeIn(items map[WorkItemRef]WorkItem, ref WorkItemRef, visiting map[WorkItemRef]bool) (WorkItemLogicalOutcome, bool) {
	item, found := items[ref]
	if !found || visiting[ref] {
		return "", false
	}
	switch item.state {
	case WorkItemStateSucceeded:
		return WorkItemLogicalSucceeded, true
	case WorkItemStateFailed, WorkItemStateCanceled, WorkItemStateSkipped:
		return WorkItemLogicalFailed, true
	case WorkItemStateSuperseded:
		visiting[ref] = true
		defer delete(visiting, ref)
		found, unresolved := false, false
		for next, successor := range items {
			if successor.reworkOf != ref {
				continue
			}
			found = true
			outcome, resolved := logicalWorkItemOutcomeIn(items, next, visiting)
			if resolved && outcome == WorkItemLogicalFailed {
				return outcome, true
			}
			unresolved = unresolved || !resolved
		}
		if found && !unresolved {
			return WorkItemLogicalSucceeded, true
		}
	}
	return "", false
}

func validInterruptCause(cause WorkItemInterruptCause) bool {
	return cause == WorkItemInterruptExecutionStopped || cause == WorkItemInterruptExecutionFailed
}
