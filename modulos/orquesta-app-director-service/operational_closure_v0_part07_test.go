package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0ReplanCausalSiCierreInsuficienteConReviewAceptada(t *testing.T) {
	runRef := "run-service-operational-closure-insufficient-replan"
	planRef := "plan-ref-service-operational-closure-insufficient-replan"
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
			WaitRefs:      []string{"wait-ref-service-operational-closure-insufficient-replan"},
		},
	}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-insufficient-replan",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-insufficient-replan"},
		},
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:10:00Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
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
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 0 {
		t.Fatalf("RunClosed=%d events=%+v", got, eventSink.EventsV0())
	}
	_, replayIssues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:10:01Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
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
		t.Fatalf("maybeCloseOperationalDirectorV0 replay: %v", err)
	}
	if len(replayIssues) != 0 {
		t.Fatalf("replayIssues=%+v", replayIssues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded replay=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded replay=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 0 {
		t.Fatalf("RunClosed replay=%d events=%+v", got, eventSink.EventsV0())
	}
	blockedState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-replan-or-close")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		blockedState.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(blockedState.BlockerRefs, "closure_ref") ||
		!serviceStringInSetV0(blockedState.BlockerRefs, "operational_closure_insufficient") ||
		closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues" {
		t.Fatalf("blockedState=%+v closureStep=%+v", blockedState, closureStep)
	}

	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T19:10:02Z",
		CorrelationID:              "corr-service-operational-closure-insufficient-replan",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, agentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, agentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
}
