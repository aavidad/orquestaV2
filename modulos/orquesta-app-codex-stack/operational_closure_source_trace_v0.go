package orquestaappcodexstack

import (
	"encoding/json"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func codexStackOperationalClosureTraceFromEventsAndRunV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) codexStackOperationalClosureTraceV0 {
	return codexStackOperationalClosureTraceWithRunProjectionsV0(
		codexStackOperationalClosureTraceFromEventsV0(events),
		run,
	)
}

func codexStackOperationalClosureTraceWithRunProjectionsV0(
	trace codexStackOperationalClosureTraceV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) codexStackOperationalClosureTraceV0 {
	trace = codexStackOperationalClosureTraceEnsureMapsV0(trace)
	for _, deliveryRef := range codexStackOperationalClosureCompactRefsV0(run.Deliveries) {
		if _, ok := trace.Deliveries[deliveryRef]; ok {
			continue
		}
		taskRef := codexStackOperationalClosureInferDeliveryTaskFromRunV0(run, deliveryRef)
		if taskRef == "" {
			continue
		}
		trace.Deliveries[deliveryRef] = orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  deliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       taskRef,
			AgentRef:     codexStackOperationalClosureInferDeliveryAgentFromRunV0(run, deliveryRef, taskRef),
			Summary:      "Entrega reconstruida desde proyeccion durable del run.",
			EvidenceRefs: []string{deliveryRef, "evidence-ref-operational-closure-run-projection-delivery"},
		}
		if !codexStackOperationalClosureContainsV0(trace.DeliveryRefs, deliveryRef) {
			trace.DeliveryRefs = append(trace.DeliveryRefs, deliveryRef)
		}
	}
	for _, reviewRequestID := range codexStackOperationalClosureCompactRefsV0(run.Reviews) {
		if _, ok := trace.ReviewRequests[reviewRequestID]; ok {
			continue
		}
		deliveryRef := codexStackOperationalClosureInferReviewDeliveryFromRunV0(run, reviewRequestID)
		if deliveryRef == "" {
			continue
		}
		trace.ReviewRequests[reviewRequestID] = orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestID,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Review reconstruida desde proyeccion durable del run.",
			EvidenceRefs:    []string{reviewRequestID, "evidence-ref-operational-closure-run-projection-review-request"},
		}
	}
	for _, rawResult := range codexStackOperationalClosureCompactRefsV0(run.ReviewResults) {
		result, ok := codexStackOperationalClosureReviewResultFromRunProjectionV0(run, rawResult)
		if !ok {
			continue
		}
		if _, exists := trace.ReviewResults[strings.TrimSpace(result.ReviewResultRef)]; exists {
			continue
		}
		trace.ReviewResults[strings.TrimSpace(result.ReviewResultRef)] = result
		if _, exists := trace.ReviewRequests[strings.TrimSpace(result.ReviewRequestID)]; !exists {
			trace.ReviewRequests[strings.TrimSpace(result.ReviewRequestID)] = orquestacoreworkflow.ReviewRequestedPayloadV0{
				ReviewRequestID: result.ReviewRequestID,
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     result.DeliveryRef,
				Summary:         "Review reconstruida desde resultado durable del run.",
				EvidenceRefs: []string{
					result.ReviewRequestID,
					"evidence-ref-operational-closure-run-projection-review-request-from-result",
				},
			}
		}
	}
	for _, acceptedReviewRef := range codexStackOperationalClosureCompactRefsV0(run.AcceptedReviews) {
		if _, ok := trace.AcceptedReviews[acceptedReviewRef]; ok {
			continue
		}
		deliveryRef := codexStackOperationalClosureInferAcceptedReviewDeliveryFromRunV0(run, acceptedReviewRef)
		reviewRequestID := codexStackOperationalClosureInferAcceptedReviewRequestFromRunV0(trace, run, deliveryRef)
		if deliveryRef == "" || reviewRequestID == "" {
			continue
		}
		trace.AcceptedReviews[acceptedReviewRef] = orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestID,
			DeliveryRef:       deliveryRef,
			Summary:           "Review aceptada reconstruida desde proyeccion durable del run.",
			EvidenceRefs:      []string{acceptedReviewRef, "evidence-ref-operational-closure-run-projection-accepted-review"},
		}
	}
	trace.DeliveryRefs = codexStackOperationalClosureCompactRefsV0(trace.DeliveryRefs)
	return trace
}

func codexStackOperationalClosureTraceEnsureMapsV0(
	trace codexStackOperationalClosureTraceV0,
) codexStackOperationalClosureTraceV0 {
	if trace.Deliveries == nil {
		trace.Deliveries = map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{}
	}
	if trace.ReviewRequests == nil {
		trace.ReviewRequests = map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{}
	}
	if trace.ReviewResults == nil {
		trace.ReviewResults = map[string]orquestacoreworkflow.ReviewResultV0{}
	}
	if trace.AcceptedReviews == nil {
		trace.AcceptedReviews = map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{}
	}
	return trace
}

func codexStackOperationalClosureInferDeliveryTaskFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, taskRef := range codexStackOperationalClosureCompactRefsV0(run.DeliveredTasks) {
		if strings.Contains(deliveryRef, taskRef) {
			return taskRef
		}
	}
	if delivered := codexStackOperationalClosureCompactRefsV0(run.DeliveredTasks); len(delivered) == 1 {
		return delivered[0]
	}
	if tasks := codexStackOperationalClosureCompactRefsV0(run.Tasks); len(tasks) == 1 {
		return tasks[0]
	}
	return ""
}

func codexStackOperationalClosureInferDeliveryAgentFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
	taskRef string,
) string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	taskAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	for _, agentRef := range codexStackOperationalClosureCompactRefsV0(run.DeliveredAgents) {
		if strings.Contains(deliveryRef, agentRef) || agentRef == taskAgentRef {
			return agentRef
		}
	}
	if delivered := codexStackOperationalClosureCompactRefsV0(run.DeliveredAgents); len(delivered) == 1 {
		return delivered[0]
	}
	return taskAgentRef
}

func codexStackOperationalClosureInferReviewDeliveryFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRequestID string,
) string {
	reviewRequestID = strings.TrimSpace(reviewRequestID)
	if candidate := strings.TrimPrefix(reviewRequestID, "review-request-ref-"); candidate != reviewRequestID &&
		codexStackOperationalClosureContainsV0(run.Deliveries, candidate) {
		return candidate
	}
	for _, deliveryRef := range codexStackOperationalClosureCompactRefsV0(run.Deliveries) {
		if strings.Contains(reviewRequestID, deliveryRef) {
			return deliveryRef
		}
	}
	if deliveries := codexStackOperationalClosureCompactRefsV0(run.Deliveries); len(deliveries) == 1 {
		return deliveries[0]
	}
	return ""
}

func codexStackOperationalClosureReviewResultFromRunProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	raw string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	parts := strings.Split(strings.TrimSpace(raw), "#")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return orquestacoreworkflow.ReviewResultV0{}, false
	}
	result := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: strings.TrimSpace(parts[0]),
		Summary:         "Resultado de review reconstruido desde proyeccion durable del run.",
		EvidenceRefs:    []string{strings.TrimSpace(parts[0]), "evidence-ref-operational-closure-run-projection-review-result"},
	}
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "review_result":
			result.Status = orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(value))
		case "review_request":
			result.ReviewRequestID = strings.TrimSpace(value)
		case "delivery":
			result.DeliveryRef = strings.TrimSpace(value)
		}
	}
	if result.Status == "" || result.DeliveryRef == "" {
		return orquestacoreworkflow.ReviewResultV0{}, false
	}
	if result.ReviewRequestID == "" {
		result.ReviewRequestID = codexStackOperationalClosureInferReviewRequestForDeliveryFromRunV0(run, result.DeliveryRef)
	}
	if result.ReviewRequestID == "" {
		return orquestacoreworkflow.ReviewResultV0{}, false
	}
	return result, true
}

func codexStackOperationalClosureInferReviewRequestForDeliveryFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	candidate := "review-request-ref-" + deliveryRef
	if codexStackOperationalClosureContainsV0(run.Reviews, candidate) {
		return candidate
	}
	for _, reviewRef := range codexStackOperationalClosureCompactRefsV0(run.Reviews) {
		if strings.Contains(reviewRef, deliveryRef) {
			return reviewRef
		}
	}
	if reviews := codexStackOperationalClosureCompactRefsV0(run.Reviews); len(reviews) == 1 {
		return reviews[0]
	}
	return ""
}

func codexStackOperationalClosureInferAcceptedReviewDeliveryFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	acceptedReviewRef string,
) string {
	acceptedReviewRef = strings.TrimSpace(acceptedReviewRef)
	if candidate := strings.TrimPrefix(acceptedReviewRef, "accepted-review-ref-"); candidate != acceptedReviewRef &&
		codexStackOperationalClosureContainsV0(run.Deliveries, candidate) {
		return candidate
	}
	for _, deliveryRef := range codexStackOperationalClosureCompactRefsV0(run.Deliveries) {
		if strings.Contains(acceptedReviewRef, deliveryRef) {
			return deliveryRef
		}
	}
	if deliveries := codexStackOperationalClosureCompactRefsV0(run.Deliveries); len(deliveries) == 1 {
		return deliveries[0]
	}
	return ""
}

func codexStackOperationalClosureInferAcceptedReviewRequestFromRunV0(
	trace codexStackOperationalClosureTraceV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, reviewRequest := range trace.ReviewRequests {
		if strings.TrimSpace(reviewRequest.DeliveryRef) == deliveryRef &&
			codexStackOperationalClosureContainsV0(run.Reviews, reviewRequest.ReviewRequestID) {
			return strings.TrimSpace(reviewRequest.ReviewRequestID)
		}
	}
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.DeliveryRef) == deliveryRef && strings.TrimSpace(result.ReviewRequestID) != "" {
			return strings.TrimSpace(result.ReviewRequestID)
		}
	}
	return codexStackOperationalClosureInferReviewRequestForDeliveryFromRunV0(run, deliveryRef)
}

func codexStackOperationalClosureDecodeEventPayloadV0(event orquestacoreworkflow.OrchestrationEventV0, out any) bool {
	return json.Unmarshal(event.Payload, out) == nil
}
