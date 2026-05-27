package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0MarcaWaitStateClearedAlCerrarPlanState(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-wait-cleared"
	planRef := "plan-ref-service-operational-closure-plan-state-wait-cleared"
	waitRef := "wait-ref-service-operational-closure-plan-state-wait-cleared"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	waitStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		StepID:           "step-wait-subagents",
		Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:           orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:          state.ActiveWaveRef,
		CohortRef:        state.ActiveCohortRef,
		ParentTaskRef:    state.ActiveParentTaskRef,
		TaskRefs:         []string{taskRef},
		WaitRefs:         []string{waitRef},
		AgentRefs:        []string{agentRef},
		PendingAgentRefs: []string{agentRef},
	}
	state.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{waitStep}, state.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	waitStateStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(
		orquestacionnucleoapp.WorkflowTaskWaitStateV0{
			SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
			WaitRef:          waitRef,
			RunRef:           runRef,
			ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
			CohortRef:        state.ActiveCohortRef,
			WaveRef:          state.ActiveWaveRef,
			ParentTaskRef:    state.ActiveParentTaskRef,
			TaskRefs:         []string{taskRef},
			AgentRefs:        []string{agentRef},
			PendingAgentRefs: []string{agentRef},
			Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
			Attempt:          1,
			ObservedAt:       "2026-05-22T13:00:00Z",
		},
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-wait-cleared",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-wait-cleared",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T13:10:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStateStore,
		WaitStateWriter:            waitStateStore,
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
	waitState, err := waitStateStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 first: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusClearedV0 ||
		len(waitState.PendingAgentRefs) != 0 ||
		waitState.ObservedAt != "2026-05-22T13:10:00Z" ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-cleared-v0") != 1 {
		t.Fatalf("waitState no cleared: %+v", waitState)
	}
	request.OccurredAt = "2026-05-22T13:10:01Z"
	_, issues, err = maybeCloseOperationalDirectorV0(
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
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	replayedWaitState, err := waitStateStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 replay: %v", err)
	}
	if replayedWaitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusClearedV0 ||
		serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-cleared-v0") != 1 {
		t.Fatalf("waitState replay no idempotente: %+v", replayedWaitState)
	}
}

func TestMaybeCloseOperationalDirectorV0ReplayNoDuplicaCierreNiPlanState(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-closed-replay"
	planRef := "plan-ref-service-operational-closure-plan-state-closed-replay"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-closed-replay",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-closed-replay",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T10:10:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	}

	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		loop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	request.OccurredAt = "2026-05-22T10:10:01Z"
	replay, issues, err := maybeCloseOperationalDirectorV0(
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
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(replay.Run.Closures) != 1 ||
		replay.Run.Closures[0] != "closure-ref-service-operational-closure-plan-state-closed-replay" ||
		len(replay.Run.Validations) != 1 ||
		len(replay.Run.ClosedTasks) != 1 {
		t.Fatalf("run replay no idempotente: %+v", replay.Run)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 1 {
		t.Fatalf("RunClosed duplicado: got=%d events=%+v", got, eventSink.EventsV0())
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		len(state.BlockerRefs) != 0 ||
		serviceCountStringV0(state.EvidenceRefs, "closure-ref-service-operational-closure-plan-state-closed-replay") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-succeeded-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(step.RequiredTestEvidenceRefs, "test-evidence-ref-service-operational-closure-001") != 1 {
		t.Fatalf("plan state replay no idempotente: state=%+v step=%+v", state, step)
	}
}
