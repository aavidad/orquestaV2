package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestCancelAwaitingCommitWithDeniedEffectRetiresAction(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	principal, projectRef, err := system.access.values()
	appTestNoError(t, err)
	accessStore, ok := system.orchestrator.access.(*memoryAccessRepository)
	if !ok {
		t.Fatalf("access repository=%T, want memory", system.orchestrator.access)
	}
	seedDirectorMembership(
		t, accessStore, principal, projectRef, identity.RoleProjectOwner, system.orchestrator.clock.Now(),
	)
	system.process(t, ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent)

	before := system.record(t)
	item := before.Goal.WorkItems()[0]
	execution := onlyExecution(t, before)
	if execution.State != ExecutionAwaitingCommit {
		t.Fatalf("execution state=%s want=%s", execution.State, ExecutionAwaitingCommit)
	}
	var commitIntent EffectIntent
	for _, candidate := range before.EffectIntents {
		if candidate.ActionKind == ActionCommitChange && candidate.Subject.ExecutionRef == execution.Ref {
			commitIntent = candidate
			break
		}
	}
	if commitIntent.Ref == "" {
		t.Fatal("commit effect intent missing")
	}
	denied, err := system.orchestrator.DecideEffect(context.Background(), system.access, DecideEffectRequest{
		RequestRef:           "request:deny-empty-commit",
		GoalRef:              before.Goal.Ref(),
		IntentRef:            commitIntent.Ref,
		ExpectedIntentDigest: commitIntent.Digest,
		Decision:             EffectDenied,
		Reason:               "candidate is blocked and workspace is clean",
	})
	if err != nil || !denied.Created {
		t.Fatalf("deny commit effect: result=%+v err=%v", denied, err)
	}

	request := ControlRequest{
		RequestRef:                "request:cancel-denied-commit",
		Operation:                 ControlCancel,
		Target:                    ControlTargetWorkItem,
		GoalRef:                   before.Goal.Ref(),
		ExpectedGoalRevision:      before.Goal.Revision(),
		ExpectedPlanGeneration:    before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(),
		ExpectedSpecHash:          before.Goal.SpecHash(),
		WorkItemRef:               item.Ref(),
		ExpectedWorkItemRevision:  item.Revision(),
		Reason:                    "retire blocked commit without false evidence",
	}
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !result.Created || result.Control.Status != ControlConfirmed {
		t.Fatalf("cancel awaiting commit: result=%+v err=%v", result, err)
	}

	after := system.record(t)
	canceled, found := executionByRef(after.Executions, execution.Ref)
	if !found || canceled.State != ExecutionCanceled {
		t.Fatalf("execution after cancel=%+v found=%v", canceled, found)
	}
	system.repository.mu.Lock()
	_, actionStillPending := system.repository.actions["action:commit-change:"+execution.Ref.String()]
	system.repository.mu.Unlock()
	if actionStillPending {
		t.Fatal("denied commit action remains pending")
	}
	retirementReceipts := 0
	for _, receipt := range after.ConsumptionReceipts {
		if receipt.ActionRef != "action:commit-change:"+execution.Ref.String() {
			continue
		}
		retirementReceipts++
		if receipt.Kind != ActionCommitChange || receipt.ChangeRef.String() == "" ||
			receipt.Outcome != ActionConsumedCompleted || receipt.EffectReceiptRef != "" {
			t.Fatalf("commit retirement receipt=%+v", receipt)
		}
	}
	if retirementReceipts != 1 {
		t.Fatalf("commit retirement receipts=%d", retirementReceipts)
	}
	if current, found := after.Goal.WorkItem(goal.WorkItemRef(item.Ref())); !found || !current.CancelRequested() {
		t.Fatalf("work item cancel state=%+v found=%v", current, found)
	}
}
