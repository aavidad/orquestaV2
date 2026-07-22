package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type childHandoffGateVersionControl struct {
	*scriptedVersionControl

	mu             sync.Mutex
	previewCalls   int
	integrateCalls int
	targetOID      string
}

func (control *childHandoffGateVersionControl) PreviewIntegration(
	ctx context.Context,
	request ports.IntegrationPreviewRequest,
) (ports.IntegrationPreview, error) {
	control.mu.Lock()
	control.previewCalls++
	control.mu.Unlock()
	return control.scriptedVersionControl.PreviewIntegration(ctx, request)
}

func (control *childHandoffGateVersionControl) Integrate(
	ctx context.Context,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, error) {
	control.mu.Lock()
	control.integrateCalls++
	control.mu.Unlock()
	result, err := control.scriptedVersionControl.Integrate(ctx, request)
	if err == nil {
		control.mu.Lock()
		control.targetOID = result.TargetAfterOID
		control.mu.Unlock()
	}
	return result, err
}

func (control *childHandoffGateVersionControl) facts() (int, int, string) {
	control.mu.Lock()
	defer control.mu.Unlock()
	return control.previewCalls, control.integrateCalls, control.targetOID
}

type childHandoffGateState struct {
	StateRepository

	mu       sync.Mutex
	requeues []ActionRequeuedState
}

func (state *childHandoffGateState) RequeueAction(ctx context.Context, input ActionRequeuedState) error {
	if err := state.StateRepository.RequeueAction(ctx, input); err != nil {
		return err
	}
	state.mu.Lock()
	state.requeues = append(state.requeues, input)
	state.mu.Unlock()
	return nil
}

func (state *childHandoffGateState) requeueFacts() []ActionRequeuedState {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]ActionRequeuedState(nil), state.requeues...)
}

func TestIntegrateChangeWaitsForChildHandoffBeforeVersionControl(t *testing.T) {
	ctx := context.Background()
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	clock, ok := system.orchestrator.clock.(*mutableClock)
	if !ok {
		t.Fatalf("clock type = %T", system.orchestrator.clock)
	}
	control := &childHandoffGateVersionControl{scriptedVersionControl: &scriptedVersionControl{}}
	system.orchestrator.versionControl = control
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.approveReviews(t)

	record := system.record(t)
	parent := record.Goal.WorkItems()[0]
	childDemand := parent.BudgetDemand()
	childDemand.Ref = "budget-demand:child-handoff-gate"
	policy, err := historicalEffectPolicy(record)
	appTestNoError(t, err)
	extension, err := system.orchestrator.compilePlanExtension(ctx, record.Goal, PlanSpec{
		WorkItems: []WorkItemSpec{{
			Key: "handoff-child", Objective: "unresolved integration child",
			Phase: parent.Phase().String(), Role: "role:worker", Parent: parent.Ref().String(),
			HandoffRequired: true, OutputContract: goal.OutputContractArtifact,
			BudgetDemand: childDemand,
		}},
	}, policy, clock.Now())
	appTestNoError(t, err)
	aggregate, err := record.Goal.ApplyPlan(record.Goal.Revision(), extension)
	appTestNoError(t, err)
	system.repository.mu.Lock()
	current := system.repository.records[system.goalRef]
	current.Goal = aggregate
	system.repository.records[system.goalRef] = current
	system.repository.mu.Unlock()

	record = system.record(t)
	parent, _ = record.Goal.WorkItem(parent.Ref())
	child := record.Goal.WorkItems()[1]
	change := record.ChangeSets[0]
	execution := record.Executions[0]
	if _, passed := requiredTestsPassForChange(
		record, parent, execution, change, system.orchestrator.testAttestationPolicy,
	); !passed || record.Goal.ChildHandoffsResolved(parent.Ref()) {
		t.Fatalf("precondition exact_pass=%v child_resolved=%v", passed,
			record.Goal.ChildHandoffsResolved(parent.Ref()))
	}
	control.mu.Lock()
	control.targetOID = record.WorkspaceBindings[0].BaseOID
	targetBefore := control.targetOID
	control.mu.Unlock()
	admitted, err := system.orchestrator.IntegrateChange(ctx, system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:child-handoff-gate", GoalRef: record.Goal.Ref(),
		ChangeRef: change.Ref, ExpectedTargetOID: targetBefore,
	})
	if err != nil || !admitted.Created || admitted.Action.Kind != ActionIntegrateChange {
		t.Fatalf("admit=%+v err=%v", admitted, err)
	}

	trackedState := &childHandoffGateState{StateRepository: system.repository}
	system.orchestrator.state = trackedState
	attemptsBefore := len(record.EffectAttempts)
	processed, err := system.orchestrator.ProcessNext(ctx, "worker:child-handoff-gate")
	if err != nil || processed.Action != ActionIntegrateChange {
		t.Fatalf("pending process=%+v err=%v", processed, err)
	}
	previewCalls, integrateCalls, targetAfterGate := control.facts()
	if previewCalls != 0 || integrateCalls != 0 || targetAfterGate != targetBefore {
		t.Fatalf("version control ran before handoff: preview=%d integrate=%d target=%q want=%q",
			previewCalls, integrateCalls, targetAfterGate, targetBefore)
	}
	requeues := trackedState.requeueFacts()
	if len(requeues) != 1 || requeues[0].ErrorCode != "application.child_handoffs_pending" {
		t.Fatalf("requeues=%+v", requeues)
	}
	pending := system.record(t)
	if len(pending.EffectAttempts) != attemptsBefore || len(pending.IntegrationReceipts) != 0 {
		t.Fatalf("pending effect facts: attempts=%d want=%d receipts=%d",
			len(pending.EffectAttempts), attemptsBefore, len(pending.IntegrationReceipts))
	}
	system.repository.mu.Lock()
	pendingAction, durable := system.repository.actions[admitted.Action.Ref]
	system.repository.mu.Unlock()
	if !durable || pendingAction.token != "" || !pendingAction.record.AvailableAt.After(clock.Now()) {
		t.Fatalf("integration action not retryable: durable=%v action=%+v", durable, pendingAction)
	}

	childExecution, err := goal.NewExecutionRef("execution:child-handoff-gate")
	appTestNoError(t, err)
	childArtifact, err := goal.NewArtifactRef("artifact:child-handoff-gate")
	appTestNoError(t, err)
	resolved, err := pending.Goal.StartWorkItem(
		pending.Goal.Revision(), child.Revision(), child.Ref(), childExecution, clock.Now(),
	)
	appTestNoError(t, err)
	child, _ = resolved.WorkItem(child.Ref())
	resolved, err = resolved.SucceedWorkItem(
		resolved.Revision(), child.Revision(), child.Ref(), []goal.ArtifactRef{childArtifact}, nil, clock.Now(),
	)
	appTestNoError(t, err)
	resolved, err = resolved.ResolveChildHandoff(
		resolved.Revision(), parent.Ref(), child.Ref(), "message:child-handoff-gate",
		goal.ChildHandoffAcknowledged, "receipt:child-handoff-gate", clock.Now(),
	)
	appTestNoError(t, err)
	system.repository.mu.Lock()
	current = system.repository.records[system.goalRef]
	current.Goal = resolved
	system.repository.records[system.goalRef] = current
	system.repository.mu.Unlock()

	clock.Advance(time.Second)
	processed, err = system.orchestrator.ProcessNext(ctx, "worker:child-handoff-gate")
	if err != nil || processed.Action != ActionIntegrateChange {
		t.Fatalf("resolved process=%+v err=%v", processed, err)
	}
	previewCalls, integrateCalls, targetAfterIntegration := control.facts()
	closed := system.record(t)
	if previewCalls != 1 || integrateCalls != 1 || targetAfterIntegration == targetBefore ||
		closed.Goal.State() != goal.GoalStateSucceeded || len(closed.IntegrationReceipts) != 1 {
		t.Fatalf("resolved integration: preview=%d integrate=%d target=%q goal=%s receipts=%d",
			previewCalls, integrateCalls, targetAfterIntegration, closed.Goal.State(), len(closed.IntegrationReceipts))
	}
}
