package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func serviceReopenAndCloseOperationalClosurePrerequisiteForTestV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	ports := fixture.fullPortsV0(t)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-22T22:40:01Z",
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		ports,
	)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")
	if !serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v falta agent=%s", reentered, agentRef)
	}
	serviceAssertOperationalClosurePrerequisiteReopenedV0(t, fixture)

	run, err := fixture.RunStore.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		reentered,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 close err=%v issues=%+v", err, issues)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(loop.Run.ClosedTasks) != 1 ||
		len(loop.Run.Validations) != 1 ||
		len(loop.Run.Closures) != 1 {
		t.Fatalf("run no cerrado tras retry: %+v", loop.Run)
	}
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 closed: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" {
		t.Fatalf("state no cerrado tras retry: %+v", state)
	}
}

func serviceAssertOperationalClosurePrerequisiteBlockedV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
	reason string,
) {
	t.Helper()
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != reason ||
		!serviceStringInSetV0(state.BlockerRefs, reason) ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != reason ||
		!serviceStringInSetV0(step.BlockerRefs, reason) {
		t.Fatalf("state no bloqueado por %s: state=%+v step=%+v", reason, state, step)
	}
}

func serviceAssertOperationalClosurePrerequisiteReopenedV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reopened: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ClosureReason != "" ||
		len(state.BlockerRefs) != 0 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		step.Reason != "operational-closure-prerequisite-ready" ||
		len(step.BlockerRefs) != 0 ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0") {
		t.Fatalf("state no reabierto: state=%+v step=%+v", state, step)
	}
}

func serviceOperationalClosurePrerequisiteStateFilePortsV0(
	store *orquestastatefile.StoreV0,
	source *serviceOperationalClosureSourceForTestV0,
) StartAppDirectorPortsV0 {
	ports := StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		DirectorTaskStore:          store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
	}
	if source != nil {
		ports.OperationalClosureSource = source
	}
	return ports
}

func serviceAssertOperationalClosurePrerequisiteStateFileBlockedV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
	reason string,
) {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != reason ||
		serviceCountStringV0(state.BlockerRefs, reason) != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != reason ||
		serviceCountStringV0(step.BlockerRefs, reason) != 1 {
		t.Fatalf("state bloqueado invalido: state=%+v step=%+v", state, step)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileEventCountsV0(t, store, fixture.RunRef, 0, 0, 0)
}

func serviceAssertOperationalClosurePrerequisiteStateFileReadyV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 ready: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ClosureReason != "" ||
		len(state.BlockerRefs) != 0 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		step.Reason != "operational-closure-prerequisite-ready" ||
		len(step.BlockerRefs) != 0 ||
		serviceCountStringV0(step.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0") != 1 {
		t.Fatalf("state ready invalido: state=%+v step=%+v", state, step)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileEventCountsV0(t, store, fixture.RunRef, 0, 0, 0)
}

func serviceAssertOperationalClosurePrerequisiteStateFileClosedV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 closed: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		len(state.BlockerRefs) != 0 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-succeeded-v0") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, fixture.Source.Request.ClosureRef) != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(step.RequiredTestEvidenceRefs, "test-evidence-ref-service-operational-closure-001") != 1 {
		t.Fatalf("state closed invalido: state=%+v step=%+v", state, step)
	}
	run, err := store.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 closed: %v", err)
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(run.ClosedTasks) != 1 ||
		len(run.Validations) != 1 ||
		len(run.Closures) != 1 {
		t.Fatalf("run closed invalido: %+v", run)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileEventCountsV0(t, store, fixture.RunRef, 1, 1, 1)
}

func serviceAssertOperationalClosurePrerequisiteStateFileEventCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	wantTaskClosed int,
	wantValidation int,
	wantRunClosed int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, runRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventTaskClosedV0); got != wantTaskClosed {
		t.Fatalf("TaskClosed got=%d want=%d events=%+v", got, wantTaskClosed, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0); got != wantValidation {
		t.Fatalf("FinalValidationRegistered got=%d want=%d events=%+v", got, wantValidation, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventRunClosedV0); got != wantRunClosed {
		t.Fatalf("RunClosed got=%d want=%d events=%+v", got, wantRunClosed, events)
	}
}
