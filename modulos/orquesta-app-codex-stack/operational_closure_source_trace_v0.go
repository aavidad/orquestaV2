package orquestaappcodexstack

import (
	"encoding/json"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func codexStackOperationalClosureEvidenceRefsV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) []string {
	refs := append([]string(nil), request.EvidenceRefs...)
	refs = append(refs, delivery.EvidenceRefs...)
	refs = append(refs, reviewRequest.EvidenceRefs...)
	refs = append(refs, result.EvidenceRefs...)
	refs = append(refs, accepted.EvidenceRefs...)
	refs = append(refs, "evidence-ref-operational-director-closure-source-"+codexStackOperationalClosureSafeRefV0(delivery.DeliveryRef))
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosureTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) codexStackOperationalClosureTraceV0 {
	trace := codexStackOperationalClosureTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
	}
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				deliveryRef := strings.TrimSpace(payload.DeliveryRef)
				trace.Deliveries[deliveryRef] = payload
				trace.DeliveryRefs = append(trace.DeliveryRefs, deliveryRef)
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.ReviewRequests[strings.TrimSpace(payload.ReviewRequestID)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.ReviewResults[strings.TrimSpace(payload.ReviewResultRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.AcceptedReviews[strings.TrimSpace(payload.AcceptedReviewRef)] = payload
			}
		}
	}
	return trace
}

func codexStackOperationalClosureDecodeEventPayloadV0(event orquestacoreworkflow.OrchestrationEventV0, out any) bool {
	return json.Unmarshal(event.Payload, out) == nil
}
