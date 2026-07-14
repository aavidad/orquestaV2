package goal_test

import (
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestReplaceWorkItemExecutionUsesDualCASAndPreservesLifecycle(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "replace execution")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first := mustRef(t, "execution:first", domain.NewExecutionRef)
	running = startGoalItem(t, running, refs[0], first.String(), baseTime().Add(4*time.Minute))
	item, _ := running.WorkItem(refs[0])
	startedAt, _ := item.StartedAt()
	second := mustRef(t, "execution:second", domain.NewExecutionRef)

	_, err = running.ReplaceWorkItemExecution(
		running.Revision()-1, item.Revision(), refs[0], first, second, baseTime().Add(5*time.Minute),
	)
	requireCode(t, err, domain.ErrorRevisionConflict)
	_, err = running.ReplaceWorkItemExecution(
		running.Revision(), item.Revision()-1, refs[0], first, second, baseTime().Add(5*time.Minute),
	)
	requireCode(t, err, domain.ErrorRevisionConflict)
	_, err = running.ReplaceWorkItemExecution(
		running.Revision(), item.Revision(), refs[0],
		mustRef(t, "execution:stale", domain.NewExecutionRef), second, baseTime().Add(5*time.Minute),
	)
	requireCode(t, err, domain.ErrorRevisionConflict)

	replaced, err := running.ReplaceWorkItemExecution(
		running.Revision(), item.Revision(), refs[0], first, second, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("ReplaceWorkItemExecution() error = %v", err)
	}
	replacedItem, _ := replaced.WorkItem(refs[0])
	gotExecution, _ := replacedItem.Execution()
	gotStartedAt, _ := replacedItem.StartedAt()
	if replaced.State() != domain.GoalStateRunning || replaced.Revision() != running.Revision()+1 ||
		replacedItem.State() != domain.WorkItemStateRunning || replacedItem.Revision() != item.Revision()+1 ||
		gotExecution != second || !gotStartedAt.Equal(startedAt) {
		t.Fatalf("replacement changed lifecycle: goal=%s/%d item=%s/%d execution=%q started=%s", replaced.State(), replaced.Revision(), replacedItem.State(), replacedItem.Revision(), gotExecution, gotStartedAt)
	}
	originalExecution, _ := item.Execution()
	if originalExecution != first || item.Revision() != 2 {
		t.Fatalf("replacement mutated source item: execution=%q revision=%d", originalExecution, item.Revision())
	}
}

func TestReplaceWorkItemExecutionRejectsInvalidStateIdentityAndTime(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "replace execution safely")
	first := mustRef(t, "execution:first", domain.NewExecutionRef)
	second := mustRef(t, "execution:second", domain.NewExecutionRef)
	pendingItem, _ := aggregate.WorkItem(refs[0])
	_, err := aggregate.ReplaceWorkItemExecution(
		aggregate.Revision(), pendingItem.Revision(), refs[0], first, second, baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidTransition)

	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running = startGoalItem(t, running, refs[0], first.String(), baseTime().Add(4*time.Minute))
	item, _ := running.WorkItem(refs[0])
	tests := []struct {
		name        string
		current     domain.ExecutionRef
		replacement domain.ExecutionRef
		at          time.Time
		code        domain.ErrorCode
	}{
		{name: "current missing", replacement: second, at: baseTime().Add(5 * time.Minute), code: domain.ErrorInvalidRef},
		{name: "replacement missing", current: first, at: baseTime().Add(5 * time.Minute), code: domain.ErrorInvalidRef},
		{name: "same execution", current: first, replacement: first, at: baseTime().Add(5 * time.Minute), code: domain.ErrorInvalidArgument},
		{name: "time before start", current: first, replacement: second, at: baseTime().Add(3 * time.Minute), code: domain.ErrorInvalidArgument},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, replaceErr := running.ReplaceWorkItemExecution(
				running.Revision(), item.Revision(), refs[0], testCase.current, testCase.replacement, testCase.at,
			)
			requireCode(t, replaceErr, testCase.code)
		})
	}

	succeeded := succeedGoalItem(t, running, refs[0], "replacement-terminal", baseTime().Add(5*time.Minute))
	terminalItem, _ := succeeded.WorkItem(refs[0])
	_, err = succeeded.ReplaceWorkItemExecution(
		succeeded.Revision(), terminalItem.Revision(), refs[0], first, second, baseTime().Add(6*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidTransition)
}

func TestReplacementRevisionsRestoreForRunningAndTerminalWork(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "restore replacements")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first := mustRef(t, "execution:restore-1", domain.NewExecutionRef)
	second := mustRef(t, "execution:restore-2", domain.NewExecutionRef)
	third := mustRef(t, "execution:restore-3", domain.NewExecutionRef)
	running = startGoalItem(t, running, refs[0], first.String(), baseTime().Add(4*time.Minute))
	item, _ := running.WorkItem(refs[0])
	running, err = running.ReplaceWorkItemExecution(
		running.Revision(), item.Revision(), refs[0], first, second, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("first replacement: %v", err)
	}
	item, _ = running.WorkItem(refs[0])
	running, err = running.ReplaceWorkItemExecution(
		running.Revision(), item.Revision(), refs[0], second, third, baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatalf("second replacement: %v", err)
	}

	restoredRunning, err := domain.RestoreGoal(running.Snapshot())
	if err != nil {
		t.Fatalf("RestoreGoal(running replacements) error = %v", err)
	}
	restoredItem, _ := restoredRunning.WorkItem(refs[0])
	restoredExecution, _ := restoredItem.Execution()
	if restoredItem.Revision() != 4 || restoredExecution != third || restoredRunning.Revision() != running.Revision() {
		t.Fatalf("running replacement restore = goal revision %d/%d item %d execution %q", restoredRunning.Revision(), running.Revision(), restoredItem.Revision(), restoredExecution)
	}

	succeeded := succeedGoalItem(t, running, refs[0], "replacement-restored", baseTime().Add(7*time.Minute))
	restoredTerminal, err := domain.RestoreGoal(succeeded.Snapshot())
	if err != nil {
		t.Fatalf("RestoreGoal(terminal replacements) error = %v", err)
	}
	terminalItem, _ := restoredTerminal.WorkItem(refs[0])
	if terminalItem.Revision() != 5 || restoredTerminal.Revision() != succeeded.Revision() {
		t.Fatalf("terminal replacement restore = goal revision %d/%d item %d", restoredTerminal.Revision(), succeeded.Revision(), terminalItem.Revision())
	}

	failedItem, _ := running.WorkItem(refs[0])
	failed, err := running.FailWorkItem(
		running.Revision(), failedItem.Revision(), refs[0], baseTime().Add(7*time.Minute),
	)
	if err != nil {
		t.Fatalf("FailWorkItem(after replacements) error = %v", err)
	}
	restoredFailed, err := domain.RestoreGoal(failed.Snapshot())
	if err != nil {
		t.Fatalf("RestoreGoal(failed replacements) error = %v", err)
	}
	restoredFailedItem, _ := restoredFailed.WorkItem(refs[0])
	if restoredFailedItem.Revision() != 5 || restoredFailed.Revision() != failed.Revision() {
		t.Fatalf("failed replacement restore = goal revision %d/%d item %d", restoredFailed.Revision(), failed.Revision(), restoredFailedItem.Revision())
	}

	tampered := running.Snapshot()
	tampered.WorkItems[0].Revision++
	if _, err := domain.RestoreGoal(tampered); domain.ErrorCodeOf(err) != domain.ErrorSnapshotInvalid {
		t.Fatalf("unbalanced replacement revision restored: %v", err)
	}
}
