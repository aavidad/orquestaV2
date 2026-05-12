package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestReviewGateCandidateProviderV0BuildsReworkForChangesRequested(t *testing.T) {
	run := mustReviewGateReadyRunV0(t, "run-nucleo-review-gate-rework-001")
	run = mustApplyCommandV0(t, run, mustReviewReworkRequestReviewCommandV0(t, run.RunID))
	run = mustApplyCommandV0(t, run, mustReviewReworkResultCommandV0(
		t,
		run.RunID,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	))
	provider := ReviewGateCandidateProviderV0{
		GateSource: staticReviewGateObservationSourceV0{Observations: []ReviewGateObservationV0{
			reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0),
		}},
		RequestedBy: "orquesta-nucleo-test",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T09:30:00Z",
		CorrelationID: "corr-review-gate-rework-001",
		EvidenceRefs:  []string{"evidence-ref-review-gate-request-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReviewGateCandidates) != 1 {
		t.Fatalf("review_gate_candidates=%d", len(candidates.ReviewGateCandidates))
	}
	candidate := candidates.ReviewGateCandidates[0]
	if candidate.AcceptReview != nil {
		t.Fatalf("accept_review no debe existir: %+v", candidate.AcceptReview)
	}
	if candidate.RequestReview != nil || candidate.RecordReviewResult != nil {
		t.Fatalf("candidate debe contener solo request_rework: %+v", candidate)
	}
	if candidate.RequestRework == nil {
		t.Fatalf("request_rework requerido: %+v", candidate)
	}
	payload := candidate.RequestRework.Payload
	if payload.ReworkRequestRef != "rework-request-ref-review-result-ref-nucleo-review-001" ||
		payload.ReviewResultRef != "review-result-ref-nucleo-review-001" ||
		payload.ReviewRequestID != "review-request-ref-nucleo-review-001" ||
		payload.DeliveryRef != "delivery-ref-nucleo-review-001" {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestReviewGateCandidateProviderV0BuildsAcceptForAccepted(t *testing.T) {
	run := mustReviewGateReadyRunV0(t, "run-nucleo-review-gate-accept-001")
	run = mustApplyCommandV0(t, run, mustReviewReworkRequestReviewCommandV0(t, run.RunID))
	run = mustApplyCommandV0(t, run, mustReviewReworkResultCommandV0(
		t,
		run.RunID,
		orquestacoreworkflow.ReviewResultStatusAcceptedV0,
	))
	observation := reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusAcceptedV0)
	observation.AcceptedReviewRef = "accepted-review-ref-nucleo-review-001"
	provider := ReviewGateCandidateProviderV0{
		GateSource: staticReviewGateObservationSourceV0{Observations: []ReviewGateObservationV0{observation}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T09:31:00Z",
		CorrelationID: "corr-review-gate-accept-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReviewGateCandidates) != 1 {
		t.Fatalf("review_gate_candidates=%d", len(candidates.ReviewGateCandidates))
	}
	candidate := candidates.ReviewGateCandidates[0]
	if candidate.AcceptReview == nil {
		t.Fatalf("accept_review requerido: %+v", candidate)
	}
	if candidate.RequestRework != nil {
		t.Fatalf("request_rework no debe existir: %+v", candidate.RequestRework)
	}
	if candidate.RequestReview != nil || candidate.RecordReviewResult != nil {
		t.Fatalf("candidate debe contener solo accept_review: %+v", candidate)
	}
	if candidate.AcceptReview.CommandMeta.RequestedBy != "orquesta-nucleo-review-gate" {
		t.Fatalf("requested_by=%q", candidate.AcceptReview.CommandMeta.RequestedBy)
	}
}

func TestReviewGateCandidateProviderV0LimitaUnCandidatoPorTick(t *testing.T) {
	run := mustReviewGateReadyRunV0(t, "run-nucleo-review-gate-single-tick-001")
	second := reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0)
	second.CandidateRef = "review-gate-candidate-ref-nucleo-review-002"
	second.ReviewRequestID = "review-request-ref-nucleo-review-002"
	second.ReviewResultRef = "review-result-ref-nucleo-review-002"
	second.DeliveryRef = "delivery-ref-nucleo-review-002"
	provider := ReviewGateCandidateProviderV0{
		GateSource: staticReviewGateObservationSourceV0{Observations: []ReviewGateObservationV0{
			reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0),
			second,
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-10T09:32:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReviewGateCandidates) != 1 {
		t.Fatalf("review_gate_candidates=%d", len(candidates.ReviewGateCandidates))
	}
	candidate := candidates.ReviewGateCandidates[0]
	if candidate.RequestReview == nil ||
		candidate.RequestReview.Payload.ReviewRequestID != "review-request-ref-nucleo-review-001" {
		t.Fatalf("request_review=%+v", candidate.RequestReview)
	}
	if candidate.RecordReviewResult != nil || candidate.RequestRework != nil || candidate.AcceptReview != nil {
		t.Fatalf("candidate debe contener solo request_review: %+v", candidate)
	}
}

func TestReviewGateCandidateProviderV0EmiteSoloResultadoTrasReview(t *testing.T) {
	run := mustReviewGateReadyRunV0(t, "run-nucleo-review-gate-result-001")
	run = mustApplyCommandV0(t, run, mustReviewReworkRequestReviewCommandV0(t, run.RunID))
	provider := ReviewGateCandidateProviderV0{
		GateSource: staticReviewGateObservationSourceV0{Observations: []ReviewGateObservationV0{
			reviewGateObservationForTestV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0),
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-10T09:33:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	candidate := candidates.ReviewGateCandidates[0]
	if candidate.RecordReviewResult == nil ||
		candidate.RequestReview != nil ||
		candidate.RequestRework != nil ||
		candidate.AcceptReview != nil {
		t.Fatalf("candidate debe contener solo record_review_result: %+v", candidate)
	}
}

type staticReviewGateObservationSourceV0 struct {
	Observations []ReviewGateObservationV0
}

func (source staticReviewGateObservationSourceV0) BuildReviewGateObservationsV0(
	context.Context,
	ReviewGateObservationRequestV0,
) ([]ReviewGateObservationV0, error) {
	return append([]ReviewGateObservationV0(nil), source.Observations...), nil
}

func mustReviewGateReadyRunV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryReadyRunV0(t, runRef)
	run = mustApplyCommandV0(t, run, mustReviewGateDeliveryCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustOpenRevisionCommandV0(t, runRef))
	return run
}

func mustReviewGateDeliveryCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRegisterDeliveryCommandV0(
		commandMetaV0(runRef, "cmd-register-delivery-review-001", "idem-register-delivery-review-001"),
		orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  "delivery-ref-nucleo-review-001",
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       "task-ref-nucleo-001",
			AgentRef:     "agent-ref-nucleo-001",
			Summary:      "Entrega compacta para revision.",
			EvidenceRefs: []string{"evidence-ref-delivery-review-001"},
		},
	)
	if err != nil {
		t.Fatalf("delivery command: %v", err)
	}
	return command
}

func mustOpenRevisionCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		commandMetaV0(runRef, "cmd-open-revision-review-gate-001", "idem-open-revision-review-gate-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "Preparar revision.",
		},
	)
	if err != nil {
		t.Fatalf("open revision command: %v", err)
	}
	return command
}

func reviewGateObservationForTestV0(
	status orquestacoreworkflow.ReviewResultStatusV0,
) ReviewGateObservationV0 {
	return ReviewGateObservationV0{
		CandidateRef:    "review-gate-candidate-ref-nucleo-review-001",
		ReviewRequestID: "review-request-ref-nucleo-review-001",
		ReviewResultRef: "review-result-ref-nucleo-review-001",
		DeliveryRef:     "delivery-ref-nucleo-review-001",
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:          status,
		Summary:         "Revision compacta con evidencia trazable.",
		QualityGateRef:  "quality-gate-ref-nucleo-review-001",
		EvidenceRefs:    []string{"evidence-ref-review-gate-nucleo-001"},
	}
}
