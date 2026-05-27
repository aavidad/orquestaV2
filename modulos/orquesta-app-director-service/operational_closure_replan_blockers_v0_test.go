package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0ReplanCausalSiFaltaValidationRefConReviewAceptada(t *testing.T) {
	runRef := "run-service-operational-closure-validation-ref-replan"
	planRef := "plan-ref-service-operational-closure-validation-ref-replan"
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
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ClosureRef:               "closure-ref-service-operational-closure-validation-ref-replan",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-validation-ref-replan"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T21:10:00Z",
			CorrelationID:              "corr-service-operational-closure-validation-ref-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
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
	if len(issues) == 0 || issues[0].Field != "validation_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin validation_ref: %+v", loop.Run)
	}

	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	for _, event := range eventSink.EventsV0() {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("QualityGateRecorded payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
			if err := json.Unmarshal(event.Payload, &replanPayload); err != nil {
				t.Fatalf("ReplanDecisionRecorded payload: %v", err)
			}
		}
	}
	if gateEvents != 1 || replanEvents != 1 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, eventSink.EventsV0())
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != taskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "validation_ref") ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "operational_closure_insufficient") {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != taskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		len(replanPayload.FollowupRefs) != 2 ||
		!strings.HasPrefix(replanPayload.FollowupRefs[0], "capacity-ref-app-director-operational-closure-retry-") ||
		!strings.HasPrefix(replanPayload.FollowupRefs[1], "agent-ref-app-director-operational-closure-retry-") {
		t.Fatalf("replanPayload=%+v gatePayload=%+v", replanPayload, gatePayload)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(state.BlockerRefs, "validation_ref") ||
		!serviceStringInSetV0(state.BlockerRefs, "operational_closure_insufficient") ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 {
		t.Fatalf("state=%+v step=%+v", state, step)
	}
}

func TestOperationalDirectorPlanEmitClosureIssueReplanDecisionV0NoDuplicaSiRunYaReflejaDecision(t *testing.T) {
	runRef := "run-service-operational-closure-replan-reflected"
	taskRef := "task-ref-service-operational-closure-001"
	match := operationalDirectorPlanAcceptedReviewMatchV0{
		TaskRef:           taskRef,
		DeliveryRef:       "delivery-ref-service-operational-closure-reflected",
		ReviewRequestID:   "review-request-ref-service-operational-closure-reflected",
		ReviewResultRef:   "review-result-ref-service-operational-closure-reflected",
		AcceptedReviewRef: "accepted-review-ref-service-operational-closure-reflected",
	}
	issueRefs := []string{"validation_ref", "operational_closure_insufficient"}
	request := ContinueAppDirectorRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-22T22:05:00Z",
		CorrelationID: "corr-service-operational-closure-replan-reflected",
		RequestedBy:   "test",
	}
	refs := operationalDirectorClosureIssueAutoReplanRefsV0(request, match, issueRefs)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.QualityGates = []string{
		refs.GateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) + "#subject:" + taskRef,
	}
	run.ReplanDecisions = []string{
		refs.ReplanRef + "#source:" + refs.GateRef +
			"#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + refs.CapacityRef + "+" + refs.AgentRef,
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	replanRefs, err := operationalDirectorPlanEmitClosureIssueReplanDecisionV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:  runStore,
			EventSink: eventSink,
		},
		run,
		match,
		issueRefs,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanEmitClosureIssueReplanDecisionV0: %v", err)
	}
	if len(eventSink.EventsV0()) != 0 {
		t.Fatalf("emitio eventos duplicados: %+v", eventSink.EventsV0())
	}
	for _, ref := range []string{refs.GateRef, refs.ReplanRef, refs.CapacityRef, refs.AgentRef} {
		if !serviceStringInSetV0(replanRefs, ref) {
			t.Fatalf("replanRefs=%+v sin %s", replanRefs, ref)
		}
	}
}
