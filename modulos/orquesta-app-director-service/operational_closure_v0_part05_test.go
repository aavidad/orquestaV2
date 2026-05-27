package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0ReplanCausalCierreInsuficienteEnOlaMultitarea(t *testing.T) {
	runRef := "run-service-operational-closure-multitask-replan"
	planRef := "plan-ref-service-operational-closure-multitask-replan"
	parentTaskRef := "parent-task-service-operational-closure-multitask-replan"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-multitask-replan-a",
		DeliveryRef:     "delivery-ref-service-operational-closure-multitask-replan-a",
		ReviewRequestID: "review-request-ref-service-operational-closure-multitask-replan-a",
		ReviewResultRef: "review-result-ref-service-operational-closure-multitask-replan-a",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-multitask-replan-a",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-multitask-replan-a",
		ValidationRef:   "validation-ref-service-operational-closure-multitask-replan-a",
		ClosureRef:      "closure-ref-service-operational-closure-multitask-replan-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-multitask-replan-b",
		DeliveryRef:     "delivery-ref-service-operational-closure-multitask-replan-b",
		ReviewRequestID: "review-request-ref-service-operational-closure-multitask-replan-b",
		ReviewResultRef: "review-result-ref-service-operational-closure-multitask-replan-b",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-multitask-replan-b",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-multitask-replan-b",
		ValidationRef:   "validation-ref-service-operational-closure-multitask-replan-b",
	}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.ClosedTasks = []string{childA.TaskRef}
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
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB)
	planState := serviceOperationalClosurePlanStateReadyForCloseTasksV0(runRef, planRef, parentTaskRef, childA, childB)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
		StepID:        "step-wait-subagents",
		Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:       planState.ActiveWaveRef,
		CohortRef:     planState.ActiveCohortRef,
		ParentTaskRef: parentTaskRef,
		TaskRefs:      []string{childA.TaskRef, childB.TaskRef},
		AgentRefs: []string{
			orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef),
			orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef),
		},
		WaitRefs: []string{"wait-ref-service-operational-closure-multitask-replan"},
	}}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childB.TaskRef: {
				TaskID:                   childB.TaskRef,
				DeliveryRef:              childB.DeliveryRef,
				AcceptedReviewRef:        childB.AcceptedRef,
				ValidationRef:            childB.ValidationRef,
				RequiredTestEvidenceRefs: []string{childB.TestEvidenceRef},
				EvidenceRefs:             []string{"evidence-ref-service-operational-closure-multitask-replan-b"},
			},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T20:10:00Z",
		CorrelationID:              "corr-service-operational-closure-multitask-replan",
		RequestedBy:                "test",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:  runStore,
		EventSink: eventSink,
		EventReader: serviceOperationalClosureEventReaderForTestV0{
			Events: closureEvents,
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

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
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
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(updatedRun.QualityGates) != 1 ||
		!strings.Contains(updatedRun.QualityGates[0], "#subject:"+childB.TaskRef) ||
		len(updatedRun.ReplanDecisions) != 1 ||
		!strings.Contains(updatedRun.ReplanDecisions[0], "#task:"+childB.TaskRef) {
		t.Fatalf("run sin replan causal de childB: %+v", updatedRun)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T20:10:01Z",
		CorrelationID:              "corr-service-operational-closure-multitask-replan",
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
	oldAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef)
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, oldAgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, oldAgentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		closureStep.Reason != "operational-closure-issues-replan-recorded" {
		t.Fatalf("state=%+v waitStep=%+v closureStep=%+v", state, waitStep, closureStep)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateConIssuesDeCierre(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-issues"
	planRef := "plan-ref-service-operational-closure-plan-state-issues"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-issues",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-issues",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-17T16:20:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
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
	if len(issues) == 0 || issues[0].Field != "required_test_evidence_store" {
		t.Fatalf("issues=%+v", issues)
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar con issues: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		t.Fatalf("state sin active step: %+v", state)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(state.BlockerRefs, "operational-closure-issues") ||
		!serviceStringInSetV0(state.BlockerRefs, "required_test_evidence_store") ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-issues" {
		t.Fatalf("plan state no bloqueado por issues: state=%+v step=%+v", state, step)
	}
}
