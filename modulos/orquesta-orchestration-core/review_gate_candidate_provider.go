package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type ReviewGateCandidateProviderV0 struct {
	Base        CandidateProviderPortV0
	GateSource  ReviewGateObservationProviderPortV0
	RequestedBy string
}

var _ CandidateProviderPortV0 = ReviewGateCandidateProviderV0{}

func (provider ReviewGateCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.GateSource == nil ||
		request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		return candidates, nil
	}
	observations, err := provider.GateSource.BuildReviewGateObservationsV0(
		ctx,
		reviewGateObservationRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	for _, observation := range observations {
		candidate, ok, err := provider.reviewGateCandidateV0(request, observation)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		if !ok {
			continue
		}
		candidates.ReviewGateCandidates = append(candidates.ReviewGateCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
		return candidates, nil
	}
	return candidates, nil
}

func (provider ReviewGateCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func reviewGateObservationRequestV0(request SchedulerCandidateRequestV0) ReviewGateObservationRequestV0 {
	return ReviewGateObservationRequestV0{
		Run:              request.Run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       request.OccurredAt,
		PreviousStep:     request.PreviousStep,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: request.PreviousDecision,
		WaitAgentRefs:    append([]string(nil), request.WaitAgentRefs...),
	}
}

func (provider ReviewGateCandidateProviderV0) reviewGateCandidateV0(
	request SchedulerCandidateRequestV0,
	observation ReviewGateObservationV0,
) (orquestadirectorscheduler.SchedulableReviewGateCandidateV0, bool, error) {
	observation = normalizeReviewGateObservationV0(request, observation)
	if err := validateReviewGateObservationV0(request, observation); err != nil {
		return orquestadirectorscheduler.SchedulableReviewGateCandidateV0{}, false, err
	}
	candidate := orquestadirectorscheduler.SchedulableReviewGateCandidateV0{
		CandidateRef: observation.CandidateRef,
		RequestReview: &orquestadirectorscheduler.SchedulerRequestReviewCandidateV0{
			CommandMeta: reviewGateCommandMetaV0(request, provider.RequestedBy, "request-review", observation.ReviewRequestID),
			Payload: orquestacoreworkflow.RequestReviewCommandPayloadV0{
				ReviewRequestID: observation.ReviewRequestID,
				PhaseID:         observation.PhaseID,
				DeliveryRef:     observation.DeliveryRef,
				Summary:         observation.Summary,
				EvidenceRefs:    observation.EvidenceRefs,
			},
		},
		RecordReviewResult: &orquestadirectorscheduler.SchedulerRecordReviewResultCandidateV0{
			CommandMeta: reviewGateCommandMetaV0(request, provider.RequestedBy, "record-review-result", observation.ReviewResultRef),
			Payload: orquestacoreworkflow.ReviewResultV0{
				ReviewResultRef: observation.ReviewResultRef,
				ReviewRequestID: observation.ReviewRequestID,
				DeliveryRef:     observation.DeliveryRef,
				Status:          observation.Status,
				Summary:         observation.Summary,
				EvidenceRefs:    observation.EvidenceRefs,
				QualityGateRef:  observation.QualityGateRef,
			},
		},
		EvidenceRefs: compactStringsV0(append(request.EvidenceRefs, observation.EvidenceRefs...)),
	}
	if observation.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		candidate.AcceptReview = &orquestadirectorscheduler.SchedulerAcceptReviewCandidateV0{
			CommandMeta: reviewGateCommandMetaV0(request, provider.RequestedBy, "accept-review", observation.AcceptedReviewRef),
			Payload: orquestacoreworkflow.AcceptReviewCommandPayloadV0{
				AcceptedReviewRef: observation.AcceptedReviewRef,
				PhaseID:           observation.PhaseID,
				ReviewRequestID:   observation.ReviewRequestID,
				DeliveryRef:       observation.DeliveryRef,
				Summary:           observation.Summary,
				EvidenceRefs:      observation.EvidenceRefs,
			},
		}
	}
	if reviewGateStatusSupportsReworkV0(observation.Status) {
		candidate.RequestRework = &orquestadirectorscheduler.SchedulerRequestReworkCandidateV0{
			CommandMeta: reviewGateCommandMetaV0(request, provider.RequestedBy, "request-rework", observation.ReworkRequestRef),
			Payload: orquestacoreworkflow.RequestReworkCommandPayloadV0{
				ReworkRequestRef: observation.ReworkRequestRef,
				PhaseID:          observation.PhaseID,
				ReviewResultRef:  observation.ReviewResultRef,
				ReviewRequestID:  observation.ReviewRequestID,
				DeliveryRef:      observation.DeliveryRef,
				Summary:          observation.Summary,
				EvidenceRefs:     observation.EvidenceRefs,
			},
		}
	}
	next, ok := nextReviewGateCandidateV0(request.Run, candidate)
	return next, ok, nil
}

func normalizeReviewGateObservationV0(
	request SchedulerCandidateRequestV0,
	observation ReviewGateObservationV0,
) ReviewGateObservationV0 {
	observation.CandidateRef = strings.TrimSpace(observation.CandidateRef)
	observation.ReviewRequestID = strings.TrimSpace(observation.ReviewRequestID)
	observation.ReviewResultRef = strings.TrimSpace(observation.ReviewResultRef)
	observation.AcceptedReviewRef = strings.TrimSpace(observation.AcceptedReviewRef)
	observation.ReworkRequestRef = strings.TrimSpace(observation.ReworkRequestRef)
	observation.DeliveryRef = strings.TrimSpace(observation.DeliveryRef)
	observation.PhaseID = strings.TrimSpace(observation.PhaseID)
	observation.Status = orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(observation.Status)))
	observation.Summary = strings.TrimSpace(observation.Summary)
	observation.QualityGateRef = strings.TrimSpace(observation.QualityGateRef)
	observation.EvidenceRefs = compactStringsV0(observation.EvidenceRefs)
	if observation.CandidateRef == "" {
		observation.CandidateRef = "review-gate-candidate-ref-" + observation.DeliveryRef
	}
	if observation.ReworkRequestRef == "" && reviewGateStatusSupportsReworkV0(observation.Status) {
		observation.ReworkRequestRef = "rework-request-ref-" + observation.ReviewResultRef
	}
	if observation.PhaseID == "" {
		observation.PhaseID = string(request.Run.CurrentPhase)
	}
	return compactReviewGateObservationPayloadV0(observation)
}

func validateReviewGateObservationV0(
	request SchedulerCandidateRequestV0,
	observation ReviewGateObservationV0,
) error {
	requestCandidate := orquestacoreworkflow.RequestReviewCommandPayloadV0{
		ReviewRequestID: observation.ReviewRequestID,
		PhaseID:         observation.PhaseID,
		DeliveryRef:     observation.DeliveryRef,
		Summary:         observation.Summary,
		EvidenceRefs:    observation.EvidenceRefs,
	}
	if _, err := orquestacoreworkflow.NewRequestReviewCommandV0(
		reviewGateCommandMetaV0(request, "validation", "request-review", observation.ReviewRequestID),
		requestCandidate,
	); err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "review_gate_observation", err.Error())
	}
	result := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: observation.ReviewResultRef,
		ReviewRequestID: observation.ReviewRequestID,
		DeliveryRef:     observation.DeliveryRef,
		Status:          observation.Status,
		Summary:         observation.Summary,
		EvidenceRefs:    observation.EvidenceRefs,
		QualityGateRef:  observation.QualityGateRef,
	}
	if _, err := orquestacoreworkflow.NewRecordReviewResultCommandV0(
		reviewGateCommandMetaV0(request, "validation", "record-review-result", observation.ReviewResultRef),
		result,
	); err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "review_gate_observation", err.Error())
	}
	if observation.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		return validateAcceptedReviewGateObservationV0(request, observation)
	}
	if reviewGateStatusSupportsReworkV0(observation.Status) {
		return validateReworkReviewGateObservationV0(request, observation)
	}
	return nil
}

func validateAcceptedReviewGateObservationV0(
	request SchedulerCandidateRequestV0,
	observation ReviewGateObservationV0,
) error {
	payload := orquestacoreworkflow.AcceptReviewCommandPayloadV0{
		AcceptedReviewRef: observation.AcceptedReviewRef,
		PhaseID:           observation.PhaseID,
		ReviewRequestID:   observation.ReviewRequestID,
		DeliveryRef:       observation.DeliveryRef,
		Summary:           observation.Summary,
		EvidenceRefs:      observation.EvidenceRefs,
	}
	if _, err := orquestacoreworkflow.NewAcceptReviewCommandV0(
		reviewGateCommandMetaV0(request, "validation", "accept-review", observation.AcceptedReviewRef),
		payload,
	); err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "review_gate_observation", err.Error())
	}
	return nil
}

func validateReworkReviewGateObservationV0(
	request SchedulerCandidateRequestV0,
	observation ReviewGateObservationV0,
) error {
	payload := orquestacoreworkflow.RequestReworkCommandPayloadV0{
		ReworkRequestRef: observation.ReworkRequestRef,
		PhaseID:          observation.PhaseID,
		ReviewResultRef:  observation.ReviewResultRef,
		ReviewRequestID:  observation.ReviewRequestID,
		DeliveryRef:      observation.DeliveryRef,
		Summary:          observation.Summary,
		EvidenceRefs:     observation.EvidenceRefs,
	}
	if _, err := orquestacoreworkflow.NewRequestReworkCommandV0(
		reviewGateCommandMetaV0(request, "validation", "request-rework", observation.ReworkRequestRef),
		payload,
	); err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "review_gate_observation", err.Error())
	}
	return nil
}

func reviewGateStatusSupportsReworkV0(status orquestacoreworkflow.ReviewResultStatusV0) bool {
	return status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		status == orquestacoreworkflow.ReviewResultStatusRejectedV0
}

func reviewGateCommandMetaV0(
	request SchedulerCandidateRequestV0,
	requestedBy string,
	kind string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-" + kind + "-" + ref,
		RunID:          request.Run.RunID,
		IdempotencyKey: "idem-" + kind + "-" + ref,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    reviewGateRequestedByV0(requestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
}

func reviewGateRequestedByV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "orquesta-nucleo-review-gate"
}
