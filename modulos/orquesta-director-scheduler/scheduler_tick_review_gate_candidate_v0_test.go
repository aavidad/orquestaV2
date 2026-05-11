package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0ReviewGateStagesReviewCommands(t *testing.T) {
	input := validSchedulerTickInputWithReviewGateV0()

	plan := mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRequestReviewV0)

	input.Snapshot.Reviews = []string{"review-request-ref-scheduler-001"}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRecordReviewResultV0)

	input.Snapshot.ReviewResults = []string{
		"review-result-ref-scheduler-001#review_result:accepted#review_request:review-request-ref-scheduler-001#delivery:delivery-ref-scheduler-001",
	}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandAcceptReviewV0)

	input.Snapshot.AcceptedReviews = []string{"accepted-review-ref-scheduler-001"}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0ReviewGateStagesReworkCommand(t *testing.T) {
	input := validSchedulerTickInputWithReviewGateV0()
	input.ReviewGateCandidates[0] = schedulableReviewGateReworkCandidateV0()

	plan := mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRequestReviewV0)

	input.Snapshot.Reviews = []string{"review-request-ref-scheduler-001"}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRecordReviewResultV0)

	input.Snapshot.ReviewResults = []string{
		"review-result-ref-scheduler-001#review_result:changes_requested#review_request:review-request-ref-scheduler-001#delivery:delivery-ref-scheduler-001",
	}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRequestReworkV0)

	input.Snapshot.ReworkRequests = []string{
		"rework-request-ref-scheduler-001#review_result:review-result-ref-scheduler-001#review_request:review-request-ref-scheduler-001#delivery:delivery-ref-scheduler-001",
	}
	plan = mustSchedulerTickPlanV0(t, input)
	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0ReviewGateWaitsForDelivery(t *testing.T) {
	input := validSchedulerTickInputWithReviewGateV0()
	input.Snapshot.Deliveries = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusNeedsDirectorV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingCandidateMissingV0)
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunReviewGateCandidate(t *testing.T) {
	input := validSchedulerTickInputWithReviewGateV0()
	input.ReviewGateCandidates[0].RequestReview.CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)

	assertSchedulerTickErrorV0(t, err, "review_gate_candidates.request_review.command_meta.run_id")
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunReworkCandidate(t *testing.T) {
	input := validSchedulerTickInputWithReviewGateV0()
	input.ReviewGateCandidates[0] = schedulableReviewGateReworkCandidateV0()
	input.ReviewGateCandidates[0].RequestRework.CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)

	assertSchedulerTickErrorV0(t, err, "review_gate_candidates.request_rework.command_meta.run_id")
}

func validSchedulerTickInputWithReviewGateV0() DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.Snapshot.CurrentPhaseID = string(orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	input.Snapshot.Deliveries = []string{"delivery-ref-scheduler-001"}
	input.ReviewGateCandidates = []SchedulableReviewGateCandidateV0{
		validSchedulableReviewGateCandidateV0(),
	}
	return input
}

func validSchedulableReviewGateCandidateV0() SchedulableReviewGateCandidateV0 {
	return SchedulableReviewGateCandidateV0{
		CandidateRef: "review-gate-candidate-ref-scheduler-001",
		RequestReview: &SchedulerRequestReviewCandidateV0{
			CommandMeta: schedulerMetaV0("cmd-review-request-scheduler-001", "review-request"),
			Payload: orquestacoreworkflow.RequestReviewCommandPayloadV0{
				ReviewRequestID: "review-request-ref-scheduler-001",
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     "delivery-ref-scheduler-001",
				Summary:         "Revision compacta de entrega.",
				EvidenceRefs:    []string{"evidence-ref-review-request-scheduler-001"},
			},
		},
		RecordReviewResult: &SchedulerRecordReviewResultCandidateV0{
			CommandMeta: schedulerMetaV0("cmd-review-result-scheduler-001", "review-result"),
			Payload: orquestacoreworkflow.ReviewResultV0{
				ReviewResultRef: "review-result-ref-scheduler-001",
				ReviewRequestID: "review-request-ref-scheduler-001",
				DeliveryRef:     "delivery-ref-scheduler-001",
				Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
				Summary:         "Revision compacta aceptada.",
				EvidenceRefs:    []string{"evidence-ref-review-result-scheduler-001"},
				QualityGateRef:  "quality-gate-ref-scheduler-001",
			},
		},
		AcceptReview: &SchedulerAcceptReviewCandidateV0{
			CommandMeta: schedulerMetaV0("cmd-accept-review-scheduler-001", "accept-review"),
			Payload: orquestacoreworkflow.AcceptReviewCommandPayloadV0{
				AcceptedReviewRef: "accepted-review-ref-scheduler-001",
				PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				ReviewRequestID:   "review-request-ref-scheduler-001",
				DeliveryRef:       "delivery-ref-scheduler-001",
				Summary:           "Revision compacta aceptada.",
				EvidenceRefs:      []string{"evidence-ref-accept-review-scheduler-001"},
			},
		},
		EvidenceRefs: []string{"evidence-ref-review-gate-scheduler-001"},
	}
}

func schedulableReviewGateReworkCandidateV0() SchedulableReviewGateCandidateV0 {
	candidate := validSchedulableReviewGateCandidateV0()
	candidate.RecordReviewResult.Payload.Status = orquestacoreworkflow.ReviewResultStatusChangesRequestedV0
	candidate.RecordReviewResult.Payload.Summary = "Revision compacta solicita cambios."
	candidate.AcceptReview = nil
	candidate.RequestRework = &SchedulerRequestReworkCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-rework-request-scheduler-001", "rework-request"),
		Payload: orquestacoreworkflow.RequestReworkCommandPayloadV0{
			ReworkRequestRef: "rework-request-ref-scheduler-001",
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  "review-result-ref-scheduler-001",
			ReviewRequestID:  "review-request-ref-scheduler-001",
			DeliveryRef:      "delivery-ref-scheduler-001",
			Summary:          "Solicitar retrabajo compacto de revision.",
			EvidenceRefs:     []string{"evidence-ref-rework-scheduler-001"},
		},
	}
	return candidate
}
