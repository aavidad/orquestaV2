package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func (source CodexReviewGateObservationSourceV0) reviewGateObservationV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	deliveryRef := strings.TrimSpace(ack.AckRef)
	observation := orquestacionnucleoapp.ReviewGateObservationV0{
		CandidateRef:    "review-gate-candidate-ref-" + deliveryRef,
		ReviewRequestID: codexReviewGateReviewRequestIDV0(deliveryRef),
		ReviewResultRef: codexReviewGateReviewResultRefV0(deliveryRef),
		DeliveryRef:     deliveryRef,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:          source.reviewGateStatusV0(result),
		Summary:         codexReviewGateSummaryV0(result),
		QualityGateRef:  source.qualityGateRefV0(deliveryRef),
		EvidenceRefs:    codexReviewGateEvidenceRefsV0(ack, result),
	}
	if result.Accepted {
		observation.AcceptedReviewRef = codexReviewGateAcceptedReviewRefV0(deliveryRef)
	}
	return observation
}

func (source CodexReviewGateObservationSourceV0) reviewGateStatusV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) orquestacoreworkflow.ReviewResultStatusV0 {
	if result.Accepted {
		return orquestacoreworkflow.ReviewResultStatusAcceptedV0
	}
	if source.FailureStatus != "" {
		return source.FailureStatus
	}
	return orquestacoreworkflow.ReviewResultStatusChangesRequestedV0
}

func (source CodexReviewGateObservationSourceV0) qualityGateRefV0(deliveryRef string) string {
	if source.QualityGateRefFn != nil {
		if ref := strings.TrimSpace(source.QualityGateRefFn(deliveryRef)); ref != "" {
			return ref
		}
	}
	return "quality-gate-ref-" + strings.TrimSpace(deliveryRef)
}
