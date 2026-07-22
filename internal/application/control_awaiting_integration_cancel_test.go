package application

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestCancelAwaitingIntegrationRetiresExactAdmittedAction(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	principal, projectRef, err := system.access.values()
	appTestNoError(t, err)
	accessStore, ok := system.orchestrator.access.(*memoryAccessRepository)
	if !ok {
		t.Fatalf("access repository=%T, want memory", system.orchestrator.access)
	}
	seedDirectorMembership(t, accessStore, principal, projectRef, identity.RoleProjectOwner, system.orchestrator.clock.Now())
	system.processCommit(t)
	system.process(t, ActionAttestTest)

	awaiting := system.record(t)
	item := awaiting.Goal.WorkItems()[0]
	execution := onlyExecution(t, awaiting)
	change := awaiting.ChangeSets[0]
	admitted, err := system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:cancel-awaiting", GoalRef: awaiting.Goal.Ref(),
		ChangeRef: change.Ref, ExpectedTargetOID: awaiting.WorkspaceBindings[0].BaseOID,
	})
	if err != nil || !admitted.Created || admitted.Action.Kind != ActionIntegrateChange {
		t.Fatalf("admit integration: result=%+v err=%v", admitted, err)
	}

	admittedRecord := system.record(t)
	request := ControlRequest{
		RequestRef: "control:cancel-awaiting-integration", Operation: ControlCancel,
		Target: ControlTargetWorkItem, GoalRef: admittedRecord.Goal.Ref(),
		ExpectedGoalRevision: admittedRecord.Goal.Revision(), ExpectedPlanGeneration: admittedRecord.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: admittedRecord.Goal.AppSpec().Generation(), ExpectedSpecHash: admittedRecord.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		Reason: "retire exact admitted integration before claim",
	}
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !result.Created || result.Control.Status != ControlConfirmed {
		t.Fatalf("cancel awaiting integration: result=%+v err=%v", result, err)
	}

	canceled := system.record(t)
	canceledExecution, found := executionByRef(canceled.Executions, execution.Ref)
	if !found || canceledExecution.State != ExecutionCanceled {
		t.Fatalf("cancel lifecycle: execution=%s", canceledExecution.State)
	}
	var retired *ActionConsumptionReceipt
	for index := range canceled.ConsumptionReceipts {
		candidate := &canceled.ConsumptionReceipts[index]
		if candidate.ActionRef == admitted.Action.Ref {
			retired = candidate
			break
		}
	}
	if retired == nil || retired.Kind != ActionIntegrateChange || retired.GoalRef != canceled.Goal.Ref() ||
		retired.WorkItemRef != item.Ref() || retired.ExecutionRef != execution.Ref || retired.ChangeRef != change.Ref ||
		retired.PlanGeneration != execution.PlanGeneration || retired.Outcome != ActionConsumedCompleted {
		t.Fatalf("exact durable integration retirement=%+v", retired)
	}
	system.repository.mu.Lock()
	pendingActions := len(system.repository.actions)
	system.repository.mu.Unlock()
	if pendingActions != 0 {
		t.Fatalf("orphan outbox actions=%d", pendingActions)
	}
	processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:after-integration-cancel")
	if err != nil || processed.Processed {
		t.Fatalf("retired integration remained claimable: result=%+v err=%v", processed, err)
	}
	system.control.mu.Lock()
	integrationCalls := len(system.control.integrationRequests)
	system.control.mu.Unlock()
	if integrationCalls != 0 {
		t.Fatalf("VCS integration calls=%d, want zero", integrationCalls)
	}

	beforeReplay := system.record(t)
	replay, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || replay.Created || replay.Control.Ref != result.Control.Ref {
		t.Fatalf("control replay: result=%+v err=%v", replay, err)
	}
	afterReplay := system.record(t)
	if !reflect.DeepEqual(afterReplay, beforeReplay) {
		t.Fatalf("control replay mutated durable state")
	}
	system.repository.mu.Lock()
	pendingActions = len(system.repository.actions)
	system.repository.mu.Unlock()
	system.control.mu.Lock()
	integrationCalls = len(system.control.integrationRequests)
	system.control.mu.Unlock()
	if pendingActions != 0 || integrationCalls != 0 {
		t.Fatalf("control replay leaked effects: actions=%d VCS=%d", pendingActions, integrationCalls)
	}
}

func TestIntegrationActionRefsForCancellationAreCausalAndNonterminal(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	execution := onlyExecution(t, record)
	admitted, err := system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:cancel-ref-selection", GoalRef: record.Goal.Ref(),
		ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
	})
	if err != nil || !admitted.Created {
		t.Fatalf("admit integration: result=%+v err=%v", admitted, err)
	}
	record = system.record(t)

	foreignExecutionRef, err := goal.NewExecutionRef("execution:foreign-integration")
	appTestNoError(t, err)
	foreign := admitted.Action.EffectIntent
	foreign.Ref = "effect-intent:foreign-integration"
	foreign.ActionRef = "action:foreign-integration"
	foreign.Subject.ExecutionRef = foreignExecutionRef
	foreign.Digest = EffectIntentDigest(foreign)
	withForeign := record
	withForeign.EffectIntents = append(append([]EffectIntent(nil), record.EffectIntents...), foreign)
	refs, err := integrationActionRefsForCancellation(withForeign, execution)
	if err != nil || !reflect.DeepEqual(refs, []string{admitted.Action.Ref}) {
		t.Fatalf("causal refs=%v err=%v", refs, err)
	}

	terminal := withForeign
	terminal.ConsumptionReceipts = append(append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...), ActionConsumptionReceipt{
		ActionRef: admitted.Action.Ref, Kind: ActionIntegrateChange,
		GoalRef: record.Goal.Ref(), WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref,
		ChangeRef: admitted.Action.ChangeRef, PlanGeneration: execution.PlanGeneration,
		Outcome: ActionConsumedCompleted,
	})
	refs, err = integrationActionRefsForCancellation(terminal, execution)
	if err != nil || len(refs) != 0 {
		t.Fatalf("terminal action selected: refs=%v err=%v", refs, err)
	}

	otherGeneration := execution
	otherGeneration.PlanGeneration++
	refs, err = integrationActionRefsForCancellation(record, otherGeneration)
	if err != nil || len(refs) != 0 {
		t.Fatalf("other generation selected: refs=%v err=%v", refs, err)
	}
}
