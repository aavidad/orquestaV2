package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0CierraOlaMultitareaPorReentradasSinBloquear(t *testing.T) {
	runRef := "run-service-operational-closure-multitask-wave"
	planRef := "plan-ref-service-operational-closure-multitask-wave"
	parentTaskRef := "parent-task-service-operational-closure-wave"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-child-a",
		DeliveryRef:     "delivery-ref-service-operational-closure-child-a",
		ReviewRequestID: "review-request-ref-service-operational-closure-child-a",
		ReviewResultRef: "review-result-ref-service-operational-closure-child-a",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-child-a",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-child-a",
		ValidationRef:   "validation-ref-service-operational-closure-child-a",
		ClosureRef:      "closure-ref-service-operational-closure-child-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-child-b",
		DeliveryRef:     "delivery-ref-service-operational-closure-child-b",
		ReviewRequestID: "review-request-ref-service-operational-closure-child-b",
		ReviewResultRef: "review-result-ref-service-operational-closure-child-b",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-child-b",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-child-b",
		ValidationRef:   "validation-ref-service-operational-closure-child-b",
		ClosureRef:      "closure-ref-service-operational-closure-child-b",
	}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef, childB.DeliveryRef}
	run.DeliveredTasks = []string{childA.TaskRef, childB.TaskRef}
	run.Reviews = []string{childA.ReviewRequestID, childB.ReviewRequestID}
	run.ReviewResults = []string{
		serviceOperationalClosureReviewResultProjectionForTestV0(childA),
		serviceOperationalClosureReviewResultProjectionForTestV0(childB),
	}
	run.AcceptedReviews = []string{childA.AcceptedRef, childB.AcceptedRef}
	run.Agents = []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef),
	}
	run.StartedAgents = append([]string(nil), run.Agents...)
	run.DeliveredAgents = append([]string(nil), run.Agents...)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseTasksV0(runRef, planRef, parentTaskRef, childA, childB),
	)
	source := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childA.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childA),
			childB.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childB),
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T12:00:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:  runStore,
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader: serviceOperationalClosureEventReaderForTestV0{
			Events: serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB),
		},
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childA),
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childB),
		),
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childA),
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childB),
		),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	if first.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		!serviceStringInSetV0(first.Run.ClosedTasks, childA.TaskRef) ||
		serviceStringInSetV0(first.Run.ClosedTasks, childB.TaskRef) {
		t.Fatalf("primer cierre parcial incorrecto: %+v", first.Run)
	}
	if first.Status != orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0 {
		t.Fatalf("primer cierre parcial debe reentrar al director, got=%s", first.Status)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 first: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 {
		t.Fatalf("plan state no debe bloquearse tras cierre parcial: %+v", state)
	}

	request.OccurredAt = "2026-05-22T12:00:01Z"
	second, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 second err=%v issues=%+v", err, issues)
	}
	if second.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childA.TaskRef) ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childB.TaskRef) ||
		!serviceStringInSetV0(second.Run.Closures, childB.ClosureRef) {
		t.Fatalf("segundo cierre no completo la ola: %+v", second.Run)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 second: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" {
		t.Fatalf("plan state no cerrado al final de la ola: %+v", state)
	}
	if len(source.TaskCalls) != 2 ||
		source.TaskCalls[0] != childA.TaskRef ||
		source.TaskCalls[1] != childB.TaskRef {
		t.Fatalf("source no avanzo por tareas abiertas: %+v", source.TaskCalls)
	}
}
