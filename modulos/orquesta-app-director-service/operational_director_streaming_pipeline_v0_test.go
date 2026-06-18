package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// Escenario end-to-end (a traves del flujo real updateOperationalDirectorPlanStateAfterLoopV0):
// una ola con una tarea PEQUENA (entrega pronto) y una GRANDE (entrega tarde).
// Con streaming activado, la pequena debe avanzar a review en cuanto entrega,
// mientras la grande sigue en wait; cuando la grande entrega, tambien avanza.
// Sin streaming (off por defecto), la pequena NO avanza hasta que la grande
// entrega (barrera de ola).
func TestStreamingSubwavePipelineSmallAdvancesBeforeLargeV0(t *testing.T) {
	runRef := "run-streaming-pipeline"
	planRef := "plan-streaming-pipeline"
	small := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-small-fast"}
	large := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-large-slow"}
	smallAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(small.TaskRef)
	largeAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(large.TaskRef)

	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalDirectorWideWaveWaitStateForTestV0(
			runRef, planRef, "parent-pipeline", "wave-pipeline", "cohort-pipeline", small, large,
		),
	)
	ports := StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	baseRequest := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		StreamingSubwaveEnabled:    true,
	}

	// t0: solo la tarea PEQUENA entrega (la grande sigue trabajando).
	runT0 := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{smallAgent},
		DeliveredTasks:  []string{small.TaskRef},
	}
	reqT0 := baseRequest
	reqT0.OccurredAt = "2026-05-22T14:00:00Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), reqT0, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    runT0,
	}); err != nil {
		t.Fatalf("after-loop t0: %v", err)
	}
	stateT0, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("load t0: %v", err)
	}
	if stateT0.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("t0 active=%q want review (la pequena debe avanzar sola)", stateT0.ActiveStepID)
	}
	reviewT0 := serviceOperationalDirectorPlanStateStepForTestV0(t, stateT0, "step-review-deliveries")
	if !sameStringSetForSubsetTestV0(reviewT0.AgentRefs, []string{smallAgent}) {
		t.Fatalf("t0 review agents=%v want solo small", reviewT0.AgentRefs)
	}
	waitT0 := serviceOperationalDirectorPlanStateStepForTestV0(t, stateT0, "step-wait-subagents")
	if waitT0.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!sameStringSetForSubsetTestV0(waitT0.PendingAgentRefs, []string{largeAgent}) {
		t.Fatalf("t0 wait debe seguir running con large pendiente: %+v", waitT0)
	}

	// t1: ahora tambien entrega la GRANDE. Debe sumarse al review.
	runT1 := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{smallAgent, largeAgent},
		DeliveredTasks:  []string{small.TaskRef, large.TaskRef},
	}
	reqT1 := baseRequest
	reqT1.OccurredAt = "2026-05-22T14:05:00Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), reqT1, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    runT1,
	}); err != nil {
		t.Fatalf("after-loop t1: %v", err)
	}
	stateT1, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("load t1: %v", err)
	}
	reviewT1 := serviceOperationalDirectorPlanStateStepForTestV0(t, stateT1, "step-review-deliveries")
	if !sameStringSetForSubsetTestV0(reviewT1.AgentRefs, []string{smallAgent, largeAgent}) {
		t.Fatalf("t1 review agents=%v want small+large", reviewT1.AgentRefs)
	}
}

// Control: con streaming OFF (defecto), la pequena NO avanza con entrega parcial;
// la barrera de ola completa mantiene el wait hasta que entregan todas.
func TestStreamingSubwavePipelineDisabledKeepsBarrierV0(t *testing.T) {
	runRef := "run-pipeline-barrier"
	planRef := "plan-pipeline-barrier"
	small := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-barrier-small"}
	large := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-barrier-large"}
	smallAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(small.TaskRef)

	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalDirectorWideWaveWaitStateForTestV0(
			runRef, planRef, "parent-barrier", "wave-barrier", "cohort-barrier", small, large,
		),
	)
	ports := StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	runPartial := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{smallAgent},
		DeliveredTasks:  []string{small.TaskRef},
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-22T14:00:00Z",
		// StreamingSubwaveEnabled queda en false (defecto).
	}, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    runPartial,
	}); err != nil {
		t.Fatalf("after-loop barrier: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("load barrier: %v", err)
	}
	if state.ActiveStepID != "step-wait-subagents" {
		t.Fatalf("sin streaming la entrega parcial NO debe avanzar: active=%q", state.ActiveStepID)
	}
}
