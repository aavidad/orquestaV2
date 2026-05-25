package orquestacionnucleoapp

import (
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type operationalDirectorClosureTraceV0 struct {
	Deliveries      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	ReviewRequests  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults   map[string]orquestacoreworkflow.ReviewResultV0
	AcceptedReviews map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
}

func operationalDirectorClosureCausalIssuesV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	trace := operationalDirectorClosureTraceFromEventsV0(events)
	issues := make([]ErrorV0, 0)
	delivery, ok := trace.Deliveries[request.DeliveryRef]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no encontrada en eventos"))
	} else if !operationalDirectorClosureDeliveryMatchesTaskOrDescendantV0(delivery, tasks, request.TaskID) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no corresponde a la microtarea"))
	}
	accepted, ok := trace.AcceptedReviews[request.AcceptedReviewRef]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no encontrada en eventos"))
		return issues
	}
	if strings.TrimSpace(accepted.DeliveryRef) != request.DeliveryRef {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no corresponde a la entrega"))
	}
	reviewRequest, ok := trace.ReviewRequests[strings.TrimSpace(accepted.ReviewRequestID)]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_request_id", "solicitud de review no encontrada"))
	} else if strings.TrimSpace(reviewRequest.DeliveryRef) != request.DeliveryRef {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_request_id", "solicitud de review no corresponde a la entrega"))
	}
	if !operationalDirectorClosureHasAcceptedResultV0(trace, accepted, request.DeliveryRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_result_ref", "resultado aceptado no encontrado para la entrega"))
	}
	return issues
}

func operationalDirectorClosureDeliveryMatchesTaskOrDescendantV0(
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) bool {
	deliveryTaskID := strings.TrimSpace(delivery.TaskID)
	taskID = strings.TrimSpace(taskID)
	if deliveryTaskID == "" || taskID == "" {
		return false
	}
	if deliveryTaskID == taskID {
		return true
	}
	childrenByParent := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		parentRef := strings.TrimSpace(task.ParentTaskRef)
		childRef := strings.TrimSpace(task.TaskID)
		if parentRef != "" && childRef != "" {
			childrenByParent[parentRef] = append(childrenByParent[parentRef], childRef)
		}
		for _, childRef := range task.ChildTaskRefs {
			childRef = strings.TrimSpace(childRef)
			if childRef != "" && strings.TrimSpace(task.TaskID) != "" {
				childrenByParent[strings.TrimSpace(task.TaskID)] = append(childrenByParent[strings.TrimSpace(task.TaskID)], childRef)
			}
		}
	}
	queue := append([]string(nil), childrenByParent[taskID]...)
	seen := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		if current == deliveryTaskID {
			return true
		}
		queue = append(queue, childrenByParent[current]...)
	}
	return false
}

func operationalDirectorClosureTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) operationalDirectorClosureTraceV0 {
	trace := operationalDirectorClosureTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
	}
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.Deliveries[strings.TrimSpace(payload.DeliveryRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.ReviewRequests[strings.TrimSpace(payload.ReviewRequestID)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.ReviewResults[strings.TrimSpace(payload.ReviewResultRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.AcceptedReviews[strings.TrimSpace(payload.AcceptedReviewRef)] = payload
			}
		}
	}
	return trace
}

func operationalDirectorDecodeEventPayloadV0(event orquestacoreworkflow.OrchestrationEventV0, out any) bool {
	return json.Unmarshal(event.Payload, out) == nil
}

func operationalDirectorClosureHasAcceptedResultV0(
	trace operationalDirectorClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	deliveryRef string,
) bool {
	_, ok := operationalDirectorClosureAcceptedResultV0(trace, accepted, deliveryRef)
	return ok
}

func operationalDirectorClosureAcceptedResultV0(
	trace operationalDirectorClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(accepted.ReviewRequestID)
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(result.DeliveryRef) == deliveryRef &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}
