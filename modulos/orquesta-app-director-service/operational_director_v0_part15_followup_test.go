package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func serviceContinueRequiredTestsFailedReplanFollowupCloseForTestV0(
	t *testing.T,
	ctx context.Context,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	executor *fakeServiceRequiredTestCommandExecutorV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	closureSource *serviceOperationalDirectorClosureSourceRefsForTestV0,
	followupAgentRef string,
) {
	deliveredRun, err := ports.RunStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 delivered: %v", err)
	}
	followupDeliveryRef := closureSource.DeliveryRef
	followupReviewRequestRef := "review-request-ref-app-director-required-tests-replan-followup"
	followupReviewResultRef := "review-result-ref-app-director-required-tests-replan-followup"
	followupAcceptedReviewRef := closureSource.AcceptedReviewRef
	deliveredRun.DeliveredAgents = compactServiceRefsV0(append(deliveredRun.DeliveredAgents, followupAgentRef))
	deliveredRun.Deliveries = compactServiceRefsV0(append(deliveredRun.Deliveries, followupDeliveryRef))
	deliveredRun.DeliveredTasks = compactServiceRefsV0(append(deliveredRun.DeliveredTasks, fixture.TaskRef))
	deliveredRun.Reviews = compactServiceRefsV0(append(deliveredRun.Reviews, followupReviewRequestRef))
	deliveredRun.ReviewResults = compactServiceRefsV0(append(deliveredRun.ReviewResults,
		followupReviewResultRef+"#review_result:accepted#review_request:"+followupReviewRequestRef+"#delivery:"+followupDeliveryRef,
	))
	deliveredRun.AcceptedReviews = compactServiceRefsV0(append(deliveredRun.AcceptedReviews, followupAcceptedReviewRef))
	deliveredRun.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	for index := range deliveredRun.Phases {
		switch deliveredRun.Phases[index].ID {
		case orquestacoreworkflow.OrchestrationPhaseProgramacionV0:
			deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusClosedV0
		case orquestacoreworkflow.OrchestrationPhaseRevisionV0:
			deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		default:
			if deliveredRun.Phases[index].Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 {
				deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
			}
		}
	}
	if err := ports.RunStore.SaveRunV0(ctx, deliveredRun); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	followupEvents := []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  followupDeliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       fixture.TaskRef,
			AgentRef:     followupAgentRef,
			Summary:      "Entrega followup causal tras replan por tests fallidos.",
			EvidenceRefs: []string{"evidence-ref-delivery-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: followupReviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     followupDeliveryRef,
			Summary:         "Review followup causal.",
			EvidenceRefs:    []string{"evidence-ref-review-requested-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 7, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: followupReviewResultRef,
			ReviewRequestID: followupReviewRequestRef,
			DeliveryRef:     followupDeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review followup aceptada.",
			EvidenceRefs:    []string{"evidence-ref-review-result-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 8, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: followupAcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   followupReviewRequestRef,
			DeliveryRef:       followupDeliveryRef,
			Summary:           "Review followup aceptada con cadena causal.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted-required-tests-replan-followup"},
		}),
	}
	if err := ports.EventSink.AppendRunEventsV0(ctx, fixture.RunRef, followupEvents); err != nil {
		t.Fatalf("AppendRunEventsV0 followup: %v", err)
	}

	request.OccurredAt = "2026-05-22T23:40:02Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-close"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0,
		Run:    deliveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 followup: %v", err)
	}
	stateAfterTests, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 after tests: %v", err)
	}
	testsStepAfterFollowup := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterTests, "step-run-required-tests")
	replanStepAfterFollowup := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterTests, "step-replan-or-close")
	if stateAfterTests.ActiveStepID != "step-replan-or-close" ||
		testsStepAfterFollowup.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStepAfterFollowup.Reason != "required-tests-passed" ||
		replanStepAfterFollowup.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(replanStepAfterFollowup.RequiredTestEvidenceRefs) != 1 ||
		serviceStringInSetV0(replanStepAfterFollowup.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state tras followup no avanza con evidencia passed nueva: state=%+v tests=%+v replan=%+v failed=%s", stateAfterTests, testsStepAfterFollowup, replanStepAfterFollowup, fixture.RequiredTestEvidenceRef)
	}
	postTestsRun, err := ports.RunStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 post tests: %v", err)
	}
	closed, issues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    postTestsRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef:           fixture.RunRef,
			WaitAgentRefs:    []string{followupAgentRef},
			WaitScopeApplied: true,
		},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 close: err=%v issues=%+v", err, issues)
	}
	if closed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		state, _ := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
		t.Fatalf("run no cerrado: loop=%s run=%+v state=%+v", closed.Status, closed.Run, state)
	}
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("runner commands=%+v", executor.commands)
	}
	if !closureSource.Called || len(closureSource.LastRequest.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("closure source request=%+v called=%v", closureSource.LastRequest, closureSource.Called)
	}
	if !closureSource.LastRequest.WaitScopeApplied ||
		!serviceStringInSetV0(closureSource.LastRequest.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(closureSource.LastRequest.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(closureSource.LastRequest.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("closure source recibio scope/evidencia no causal: request=%+v old_agent=%s failed_evidence=%s", closureSource.LastRequest, fixture.AgentRef, fixture.RequiredTestEvidenceRef)
	}
	evidence, err := ports.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, fixture.RunRef, closureSource.LastRequest.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TaskRef != fixture.TaskRef ||
		evidence[0].TestCommand != fixture.RequiredTest ||
		evidence[0].DeliveryRef != followupDeliveryRef ||
		evidence[0].ReviewRequestID != followupReviewRequestRef ||
		evidence[0].ReviewResultRef != followupReviewResultRef ||
		evidence[0].AcceptedReviewRef != followupAcceptedReviewRef {
		t.Fatalf("evidence no causal del followup: %+v", evidence)
	}
	eventsAfterClose := eventSink.EventsV0()
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 2 {
		t.Fatalf("QualityGateRecorded esperado fallo+aceptado: got=%d events=%+v", got, eventsAfterClose)
	}
	postCloseRun, err := ports.RunStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 post close: %v", err)
	}
	if blockers := orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(postCloseRun, fixture.TaskRef); len(blockers) != 0 {
		t.Fatalf("quality gate requerido no resuelto: blockers=%v gates=%v", blockers, postCloseRun.QualityGates)
	}
	stateAfterClose, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 post close: %v", err)
	}
	closedReplanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterClose, "step-replan-or-close")
	if stateAfterClose.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		stateAfterClose.ClosureReason != "operational-closure-succeeded" ||
		closedReplanStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(closedReplanStep.RequiredTestEvidenceRefs, evidence[0].EvidenceRef) != 1 ||
		serviceStringInSetV0(closedReplanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("plan state cerrado sin evidencia passed causal: state=%+v step=%+v passed=%s failed=%s", stateAfterClose, closedReplanStep, evidence[0].EvidenceRef, fixture.RequiredTestEvidenceRef)
	}
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded duplicado: got=%d events=%+v", got, eventsAfterClose)
	}
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 1 {
		t.Fatalf("RunClosed esperado una vez: got=%d events=%+v", got, eventsAfterClose)
	}

	replayClosureSource := &serviceOperationalDirectorClosureSourceRefsForTestV0{
		TaskRef:           fixture.TaskRef,
		ValidationRef:     closureSource.ValidationRef,
		ClosureRef:        closureSource.ClosureRef,
		DeliveryRef:       followupDeliveryRef,
		AcceptedReviewRef: followupAcceptedReviewRef,
	}
	replayPorts := ports
	replayPorts.OperationalClosureSource = replayClosureSource
	request.OccurredAt = "2026-05-22T23:40:03Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-replay"
	replay, err := ContinueAppDirectorV0(ctx, request, replayPorts)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 replay: %v", err)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		replayClosureSource.Called ||
		len(eventSink.EventsV0()) != len(eventsAfterClose) {
		t.Fatalf("replay duplico o llamo cierre: run=%+v called=%v before=%d after=%d", replay.Run, replayClosureSource.Called, len(eventsAfterClose), len(eventSink.EventsV0()))
	}
}
