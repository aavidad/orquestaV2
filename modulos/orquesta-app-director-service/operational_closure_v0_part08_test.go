package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0MantieneClosureIssuesBloqueadoHastaFollowup(t *testing.T) {
	runRef := "run-service-operational-closure-insufficient-replan-wait-followup"
	planRef := "plan-ref-service-operational-closure-insufficient-replan-wait-followup"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events
	planState := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
		StepID:        "step-wait-subagents",
		Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:       planState.ActiveWaveRef,
		CohortRef:     planState.ActiveCohortRef,
		ParentTaskRef: planState.ActiveParentTaskRef,
		TaskRefs:      []string{taskRef},
		AgentRefs:     []string{agentRef},
		WaitRefs:      []string{"wait-ref-service-operational-closure-insufficient-replan-wait-followup"},
	}}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-insufficient-replan-wait-followup",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-insufficient-replan-wait-followup"},
		},
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T23:10:00Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan-wait-followup",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T23:10:01Z",
		CorrelationID:              "corr-service-operational-closure-insufficient-replan-wait-followup",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 no debe fallar esperando followup: %v", err)
	}
	if len(reentered.WaitAgentRefs) != 0 {
		t.Fatalf("reentered no debe esperar hasta que followup exista: %+v", reentered)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		state.ReplanAttempts != 0 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-issues" {
		t.Fatalf("state debe seguir bloqueado hasta followup: state=%+v step=%+v", state, step)
	}
}
