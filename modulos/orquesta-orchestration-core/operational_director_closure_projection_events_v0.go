package orquestacionnucleoapp

import (
	"encoding/json"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func operationalDirectorClosureProjectionEventsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	request OperationalDirectorClosureRequestV0,
) ([]orquestacoreworkflow.OrchestrationEventV0, bool) {
	if !operationalDirectorClosureReflectedV0(run.Deliveries, request.DeliveryRef) ||
		!operationalDirectorClosureReflectedV0(run.AcceptedReviews, request.AcceptedReviewRef) {
		return nil, false
	}
	result, ok := operationalDirectorClosureReviewResultFromRunProjectionV0(run, request)
	if !ok {
		return nil, false
	}
	deliveryTaskRef := operationalDirectorClosureDeliveryTaskFromRunProjectionV0(run, request)
	if deliveryTaskRef == "" {
		return nil, false
	}
	delivery := orquestacoreworkflow.DeliveryRegisteredPayloadV0{
		DeliveryRef: request.DeliveryRef,
		PhaseID:     string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:      deliveryTaskRef,
		AgentRef:    operationalDirectorClosureDeliveryAgentFromRunProjectionV0(run, deliveryTaskRef),
		Summary:     "Entrega reconstruida desde proyeccion durable del run.",
		EvidenceRefs: []string{
			request.DeliveryRef,
			"evidence-ref-operational-closure-run-projection-delivery",
		},
	}
	if !operationalDirectorClosureDeliveryMatchesTaskOrDescendantV0(delivery, tasks, request.TaskID) {
		return nil, false
	}
	reviewRequest := orquestacoreworkflow.ReviewRequestedPayloadV0{
		ReviewRequestID: result.ReviewRequestID,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		DeliveryRef:     request.DeliveryRef,
		Summary:         "Review reconstruida desde proyeccion durable del run.",
		EvidenceRefs: []string{
			result.ReviewRequestID,
			"evidence-ref-operational-closure-run-projection-review-request",
		},
	}
	accepted := orquestacoreworkflow.ReviewAcceptedPayloadV0{
		AcceptedReviewRef: request.AcceptedReviewRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		ReviewRequestID:   result.ReviewRequestID,
		DeliveryRef:       request.DeliveryRef,
		Summary:           "Review aceptada reconstruida desde proyeccion durable del run.",
		EvidenceRefs: []string{
			request.AcceptedReviewRef,
			"evidence-ref-operational-closure-run-projection-accepted-review",
		},
	}
	events, ok := operationalDirectorClosureProjectionEventPayloadsV0(run.RunID, request, []struct {
		eventType string
		ref       string
		payload   any
	}{
		{orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, request.DeliveryRef, delivery},
		{orquestacoreworkflow.OrchestrationEventReviewRequestedV0, result.ReviewRequestID, reviewRequest},
		{orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, result.ReviewResultRef, result},
		{orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, request.AcceptedReviewRef, accepted},
	})
	return events, ok
}

func operationalDirectorClosureReviewResultFromRunProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	for _, raw := range compactStringsV0(run.ReviewResults) {
		result, ok := operationalDirectorClosureParseReviewResultProjectionV0(run, raw)
		if !ok ||
			result.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
			strings.TrimSpace(result.DeliveryRef) != strings.TrimSpace(request.DeliveryRef) ||
			strings.TrimSpace(result.ReviewRequestID) == "" {
			continue
		}
		return result, true
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorClosureParseReviewResultProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	raw string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	parts := strings.Split(strings.TrimSpace(raw), "#")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return orquestacoreworkflow.ReviewResultV0{}, false
	}
	result := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: strings.TrimSpace(parts[0]),
		Status:          "",
		Summary:         "Resultado de review reconstruido desde proyeccion durable del run.",
		EvidenceRefs: []string{
			strings.TrimSpace(parts[0]),
			"evidence-ref-operational-closure-run-projection-review-result",
		},
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
	if result.DeliveryRef == "" {
		result.DeliveryRef = operationalDirectorClosureSingleProjectionRefV0(run.Deliveries)
	}
	if result.ReviewRequestID == "" {
		result.ReviewRequestID = operationalDirectorClosureReviewRequestForDeliveryProjectionV0(run, result.DeliveryRef)
	}
	if result.Status == "" || result.DeliveryRef == "" || result.ReviewRequestID == "" {
		return orquestacoreworkflow.ReviewResultV0{}, false
	}
	return result, true
}

func operationalDirectorClosureReviewRequestForDeliveryProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	candidate := "review-request-ref-" + deliveryRef
	if operationalDirectorClosureReflectedV0(run.Reviews, candidate) {
		return candidate
	}
	for _, reviewRef := range compactStringsV0(run.Reviews) {
		if strings.Contains(reviewRef, deliveryRef) {
			return reviewRef
		}
	}
	return operationalDirectorClosureSingleProjectionRefV0(run.Reviews)
}

func operationalDirectorClosureDeliveryTaskFromRunProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) string {
	for _, taskRef := range compactStringsV0(run.DeliveredTasks) {
		if strings.Contains(request.DeliveryRef, taskRef) {
			return taskRef
		}
	}
	if operationalDirectorClosureReflectedV0(run.DeliveredTasks, request.TaskID) {
		return request.TaskID
	}
	if taskRef := operationalDirectorClosureSingleProjectionRefV0(run.DeliveredTasks); taskRef != "" {
		return taskRef
	}
	if operationalDirectorClosureReflectedV0(run.Tasks, request.TaskID) {
		return request.TaskID
	}
	return ""
}

func operationalDirectorClosureDeliveryAgentFromRunProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) string {
	for _, agentRef := range compactStringsV0(run.DeliveredAgents) {
		if strings.Contains(agentRef, taskRef) {
			return agentRef
		}
	}
	if agentRef := operationalDirectorClosureSingleProjectionRefV0(run.DeliveredAgents); agentRef != "" {
		return agentRef
	}
	return "agent-ref-" + strings.TrimSpace(taskRef)
}

func operationalDirectorClosureSingleProjectionRefV0(values []string) string {
	values = compactStringsV0(values)
	if len(values) != 1 {
		return ""
	}
	return values[0]
}

func operationalDirectorClosureProjectionEventPayloadsV0(
	runRef string,
	request OperationalDirectorClosureRequestV0,
	payloads []struct {
		eventType string
		ref       string
		payload   any
	},
) ([]orquestacoreworkflow.OrchestrationEventV0, bool) {
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(payloads))
	for index, item := range payloads {
		raw, err := json.Marshal(item.payload)
		if err != nil {
			return nil, false
		}
		events = append(events, orquestacoreworkflow.OrchestrationEventV0{
			EventID:        fmt.Sprintf("evt-operational-closure-projection-%s-%d", operationalDirectorClosureSafeRefPartV0(item.ref), index+1),
			EventType:      item.eventType,
			RunID:          strings.TrimSpace(runRef),
			Sequence:       int64(index + 1),
			IdempotencyKey: "idem-operational-closure-projection-" + operationalDirectorClosureSafeRefPartV0(item.ref),
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			OccurredAt:     strings.TrimSpace(request.OccurredAt),
			PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
			Payload:        raw,
		})
	}
	return events, true
}

func operationalDirectorClosureSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "empty"
	}
	return value
}
