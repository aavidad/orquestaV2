package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreClosureIssuesConFollowupTardio(t *testing.T) {
	runRef := "run-service-operational-closure-late-followup"
	planRef := "plan-ref-service-operational-closure-late-followup"
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
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:        "step-wait-subagents",
			Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:       planState.ActiveWaveRef,
			CohortRef:     planState.ActiveCohortRef,
			ParentTaskRef: planState.ActiveParentTaskRef,
			TaskRefs:      []string{taskRef},
			AgentRefs:     []string{agentRef},
			WaitRefs:      []string{"wait-ref-service-operational-closure-late-followup"},
		},
	}, planState.Steps...)
	for i := range planState.Steps {
		planState.Steps[i].RequiredTestEvidenceRefs = nil
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            taskRef,
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-late-followup",
			ClosureRef:        "closure-ref-service-operational-closure-late-followup",
			EvidenceRefs:      []string{"evidence-ref-service-operational-closure-late-followup"},
		},
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T18:30:00Z",
			CorrelationID:              "corr-service-operational-closure-late-followup",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "required_test_evidence_refs" {
		t.Fatalf("issues=%+v", issues)
	}

	blockedState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-replan-or-close")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues" ||
		!serviceStringInSetV0(closureStep.BlockerRefs, "required_test_evidence_refs") ||
		blockedState.ReplanAttempts != 0 {
		t.Fatalf("blockedState=%+v closureStep=%+v", blockedState, closureStep)
	}

	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	events := append(closureEvents, eventSink.EventsV0()...)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T18:30:01Z",
		CorrelationID:              "corr-service-operational-closure-late-followup",
		OperationalDirectorPlanRef: planRef,
		WaitAgentRefs:              []string{agentRef},
		WaitWaveRef:                planState.ActiveWaveRef,
		WaitCohortRef:              planState.ActiveCohortRef,
		WaitParentTaskRef:          planState.ActiveParentTaskRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, agentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	closureStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ClosureReason != "" ||
		len(state.BlockerRefs) != 0 ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, agentRef) {
		t.Fatalf("state=%+v", state)
	}
	if closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues-replan-recorded" ||
		!serviceStringInSetV0(closureStep.BlockerRefs, "required_test_evidence_refs") ||
		len(closureStep.ReplanDecisionRefs) != 1 {
		t.Fatalf("closureStep=%+v", closureStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "operational-closure-issues-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, agentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	replayed, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T18:30:02Z",
		CorrelationID:              "corr-service-operational-closure-late-followup",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if !serviceStringInSetV0(replayed.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(replayed.WaitAgentRefs, agentRef) {
		t.Fatalf("replayed=%+v followup=%s old=%s", replayed, followupAgentRef, agentRef)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if state.ReplanAttempts != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-issues-replan-v0") != 1 {
		t.Fatalf("state replay=%+v", state)
	}
}
