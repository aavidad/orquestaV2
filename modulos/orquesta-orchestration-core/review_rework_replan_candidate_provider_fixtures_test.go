package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type staticReviewReworkReplanPlanSourceV0 struct {
	Plans []ReviewReworkReplanPlanV0
}

func (source staticReviewReworkReplanPlanSourceV0) BuildReviewReworkReplanPlansV0(
	context.Context,
	ReviewReworkReplanPlanRequestV0,
) ([]ReviewReworkReplanPlanV0, error) {
	return append([]ReviewReworkReplanPlanV0(nil), source.Plans...), nil
}

type reviewReworkLoopPlanSourceV0 struct{}

func (source reviewReworkLoopPlanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request ReviewReworkReplanPlanRequestV0,
) ([]ReviewReworkReplanPlanV0, error) {
	if !reviewReworkRunHasRefV0(request.Run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") ||
		reviewReworkRunContainsRefV0(request.Run.Agents, "agent-ref-nucleo-review-retry-001") {
		return nil, nil
	}
	return []ReviewReworkReplanPlanV0{reviewReworkRetryPlanV0(request.Run.RunID)}, nil
}

type reviewReworkLoopGateSourceV0 struct{}

func (source reviewReworkLoopGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	request ReviewGateObservationRequestV0,
) ([]ReviewGateObservationV0, error) {
	if reviewReworkRunHasRefV0(request.Run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") {
		return nil, nil
	}
	return []ReviewGateObservationV0{reviewReworkChangesRequestedGateObservationV0()}, nil
}

func mustReviewReworkReadyRunV0(
	t *testing.T,
	runRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustReviewGateReadyRunV0(t, runRef)
	run = mustApplyCommandV0(t, run, mustReviewReworkRequestReviewCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustReviewReworkResultCommandV0(t, runRef, status))
	return mustApplyCommandV0(t, run, mustReviewReworkRequestCommandV0(t, runRef, status))
}

func reviewReworkChangesRequestedGateObservationV0() ReviewGateObservationV0 {
	observation := reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0)
	observation.ReworkRequestRef = "rework-request-ref-nucleo-review-001"
	return observation
}

func assertReviewReworkProgressiveEventsV0(
	t *testing.T,
	sink *InMemoryEventSinkV0,
	want []string,
) {
	t.Helper()
	events := sink.EventsV0()
	next := 0
	for _, event := range events {
		if next < len(want) && event.EventType == want[next] {
			next++
		}
	}
	if next == len(want) {
		return
	}
	t.Fatalf("eventos progresivos incompletos: want=%v got=%v", want, reviewReworkEventTypesV0(events))
}

func reviewReworkEventTypesV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func mustReviewReworkRequestReviewCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRequestReviewCommandV0(
		commandMetaV0(runRef, "cmd-request-review-rework-001", "idem-request-review-rework-001"),
		orquestacoreworkflow.RequestReviewCommandPayloadV0{
			ReviewRequestID: "review-request-ref-nucleo-review-001",
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     "delivery-ref-nucleo-review-001",
			Summary:         "Revision compacta de entrega.",
			EvidenceRefs:    []string{"evidence-ref-review-request-rework-001"},
		},
	)
	if err != nil {
		t.Fatalf("request review command: %v", err)
	}
	return command
}

func mustReviewReworkResultCommandV0(
	t *testing.T,
	runRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRecordReviewResultCommandV0(
		commandMetaV0(runRef, "cmd-record-review-rework-001", "idem-record-review-rework-001"),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-ref-nucleo-review-001",
			ReviewRequestID: "review-request-ref-nucleo-review-001",
			DeliveryRef:     "delivery-ref-nucleo-review-001",
			Status:          status,
			Summary:         "Revision compacta solicita cambios.",
			EvidenceRefs:    []string{"evidence-ref-review-result-rework-001"},
			QualityGateRef:  "quality-gate-ref-nucleo-review-rework-001",
		},
	)
	if err != nil {
		t.Fatalf("review result command: %v", err)
	}
	return command
}

func mustReviewReworkRequestCommandV0(
	t *testing.T,
	runRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	if status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("accepted review cannot request rework")
	}
	command, err := orquestacoreworkflow.NewRequestReworkCommandV0(
		commandMetaV0(runRef, "cmd-request-rework-review-001", "idem-request-rework-review-001"),
		orquestacoreworkflow.RequestReworkCommandPayloadV0{
			ReworkRequestRef: "rework-request-ref-nucleo-review-001",
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  "review-result-ref-nucleo-review-001",
			ReviewRequestID:  "review-request-ref-nucleo-review-001",
			DeliveryRef:      "delivery-ref-nucleo-review-001",
			Summary:          "Retrabajo compacto solicitado.",
			EvidenceRefs:     []string{"evidence-ref-rework-request-review-001"},
		},
	)
	if err != nil {
		t.Fatalf("request rework command: %v", err)
	}
	return command
}

func reviewReworkRetryPlanV0(runRef string) ReviewReworkReplanPlanV0 {
	return ReviewReworkReplanPlanV0{
		CandidateRef:               "review-rework-replan-candidate-ref-nucleo-review-001",
		ReplanRef:                  "replan-ref-nucleo-review-001",
		SignalRef:                  "review-rework-signal-ref-nucleo-review-001",
		ReworkRequestRef:           "rework-request-ref-nucleo-review-001",
		TaskRef:                    "task-ref-nucleo-001",
		RequestedAction:            orquestacorereplanner.ReplanActionRetryTaskV0,
		CapacityRequestRef:         "capacity-ref-nucleo-review-retry-001",
		AgentRequestID:             "agent-ref-nucleo-review-retry-001",
		AgentRole:                  "implementacion",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		Summary:                    "Repetir tarea tras revision no aceptada.",
		EvidenceRefs:               []string{"evidence-ref-review-rework-plan-001"},
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-ref-nucleo-review-001",
			ReviewRequestID: "review-request-ref-nucleo-review-001",
			DeliveryRef:     "delivery-ref-nucleo-review-001",
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "Revision compacta solicita cambios.",
			EvidenceRefs:    []string{"evidence-ref-review-result-rework-001"},
			QualityGateRef:  "quality-gate-ref-nucleo-review-rework-001",
		},
	}
}
