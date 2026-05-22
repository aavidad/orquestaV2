package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestReviewReworkReplanCandidateProviderV0BuildsRetryPlanFromRework(t *testing.T) {
	runRef := "run-nucleo-review-rework-replan-001"
	run := mustReviewReworkReadyRunV0(t, runRef, orquestacoreworkflow.ReviewResultStatusChangesRequestedV0)
	provider := ReviewReworkReplanCandidateProviderV0{
		PlanSource: staticReviewReworkReplanPlanSourceV0{Plans: []ReviewReworkReplanPlanV0{
			reviewReworkRetryPlanV0(runRef),
		}},
		RequestedBy: "orquesta-nucleo-test",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T10:00:00Z",
		CorrelationID: "corr-review-rework-replan-001",
		EvidenceRefs:  []string{"evidence-ref-review-rework-request-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan candidates=%d", len(candidates.ReplanFollowupCandidates))
	}
	input := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput
	if input.SourceKind != orquestadirector.ReplanFollowupSourceReviewReworkV0 {
		t.Fatalf("source_kind=%q", input.SourceKind)
	}
	if input.DecisionPayload.SourceRef != "rework-request-ref-nucleo-review-001" ||
		input.DecisionPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("decision payload=%+v", input.DecisionPayload)
	}
	if input.OpenPhaseCandidate == nil ||
		input.OpenPhaseCandidate.Payload.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		t.Fatalf("open phase candidate=%+v", input.OpenPhaseCandidate)
	}
	if input.CapacityCandidate == nil || input.AgentCandidate == nil {
		t.Fatalf("followups incompletos: capacity=%+v agent=%+v", input.CapacityCandidate, input.AgentCandidate)
	}
	if input.AgentCandidate.Payload.Summary != "Repetir tarea tras revision no aceptada." ||
		input.CapacityCandidate.Payload.Summary != "Repetir tarea tras revision no aceptada." {
		t.Fatalf("summary de plan no propagado: capacity=%q agent=%q",
			input.CapacityCandidate.Payload.Summary,
			input.AgentCandidate.Payload.Summary,
		)
	}
}

func TestReviewReworkReplanCandidateProviderV0NoPropagaEvidenciaAmbientalDelRuntime(t *testing.T) {
	runRef := "run-nucleo-review-rework-replan-evidence-001"
	run := mustReviewReworkReadyRunV0(t, runRef, orquestacoreworkflow.ReviewResultStatusChangesRequestedV0)
	provider := ReviewReworkReplanCandidateProviderV0{
		PlanSource: staticReviewReworkReplanPlanSourceV0{Plans: []ReviewReworkReplanPlanV0{
			reviewReworkRetryPlanV0(runRef),
		}},
		RequestedBy: "orquesta-nucleo-test",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T10:00:00Z",
		CorrelationID: "corr-review-rework-replan-evidence-001",
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-stack-drain",
			"evidence-ref-runtime-drain",
		},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0 con evidencia ambiental: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan candidates=%d", len(candidates.ReplanFollowupCandidates))
	}
	evidence := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.EvidenceRefs
	if reviewReworkRunContainsRefV0(evidence, "evidence-ref-codex-supervisor-stack-drain") ||
		reviewReworkRunContainsRefV0(evidence, "evidence-ref-runtime-drain") {
		t.Fatalf("evidence_refs filtran detalles de runtime: %v", evidence)
	}
	if !reviewReworkRunContainsRefV0(evidence, "evidence-ref-review-rework-plan-001") {
		t.Fatalf("evidence_refs perdio evidencia causal del plan: %v", evidence)
	}
}

func TestReviewReworkReplanCandidateProviderV0ProgressiveLoopRoutesChangesRequestedToNewAgent(t *testing.T) {
	runRef := "run-nucleo-review-rework-loop-001"
	run := mustReviewGateReadyRunV0(t, runRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		!reviewReworkRunContainsRefV0(run.Deliveries, "delivery-ref-nucleo-review-001") {
		t.Fatalf("precondicion de revision invalida: %+v", run)
	}
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: ReviewReworkReplanCandidateProviderV0{
			Base: ReviewGateCandidateProviderV0{
				GateSource:  reviewReworkLoopGateSourceV0{},
				RequestedBy: "orquesta-nucleo-test",
			},
			PlanSource:  reviewReworkLoopPlanSourceV0{},
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 1,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-10T10:01:00Z",
		MaxBursts:            3,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-review-rework-loop-001",
		EvidenceRefs:         []string{"evidence-ref-review-rework-loop-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s result=%+v events=%v",
			result.Run.CurrentPhase, result, reviewReworkEventTypesV0(sink.EventsV0()))
	}
	if !reviewReworkRunContainsRefV0(result.Run.Reviews, "review-request-ref-nucleo-review-001") ||
		!reviewReworkRunHasRefV0(result.Run.ReviewResults, "review-result-ref-nucleo-review-001", "#review_result:") ||
		!reviewReworkRunHasRefV0(result.Run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") {
		t.Fatalf("revision no aceptada incompleta: reviews=%v results=%v reworks=%v",
			result.Run.Reviews, result.Run.ReviewResults, result.Run.ReworkRequests)
	}
	if !reviewReworkRunHasRefV0(result.Run.ReplanDecisions, "replan-ref-nucleo-review-001", "#source:") {
		t.Fatalf("replan_decisions=%v", result.Run.ReplanDecisions)
	}
	if !reviewReworkRunContainsRefV0(result.Run.CapacityRequests, "capacity-ref-nucleo-review-retry-001") ||
		!reviewReworkRunHasRefV0(result.Run.CapacityDecisions, "capacity-ref-nucleo-review-retry-001", "#capacity_decision:") {
		t.Fatalf("capacity requests=%v decisions=%v", result.Run.CapacityRequests, result.Run.CapacityDecisions)
	}
	if !reviewReworkRunContainsRefV0(result.Run.Agents, "agent-ref-nucleo-review-retry-001") {
		t.Fatalf("agents=%v result=%+v", result.Run.Agents, result)
	}
	if reviewReworkRunContainsRefV0(result.Run.AcceptedReviews, "accepted-review-ref-nucleo-review-001") ||
		sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0) ||
		sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventRunClosedV0) {
		t.Fatalf("changes_requested no debe aceptar revision ni cerrar run: %+v", result.Run)
	}
	assertReviewReworkProgressiveEventsV0(t, sink, []string{
		orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReworkRequestedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventPhaseOpenedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
	})
}
