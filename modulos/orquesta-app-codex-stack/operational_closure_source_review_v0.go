package orquestaappcodexstack

import (
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func codexStackOperationalClosureDeliveriesForTaskV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.DeliveryRegisteredPayloadV0 {
	deliveries := make([]orquestacoreworkflow.DeliveryRegisteredPayloadV0, 0)
	for _, deliveryRef := range request.Run.Deliveries {
		delivery, ok := trace.Deliveries[strings.TrimSpace(deliveryRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(delivery.TaskID) != strings.TrimSpace(task.TaskID) ||
			!codexStackOperationalClosureContainsV0(request.Run.Deliveries, delivery.DeliveryRef) {
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	closedTasks := codexStackOperationalClosureSetV0(request.Run.ClosedTasks)
	for _, childRef := range codexStackOperationalClosureChildTaskRefsV0(request.Run, task) {
		if !closedTasks[childRef] {
			continue
		}
		for _, deliveryRef := range request.Run.Deliveries {
			delivery, ok := trace.Deliveries[strings.TrimSpace(deliveryRef)]
			if !ok || strings.TrimSpace(delivery.TaskID) != childRef ||
				!codexStackOperationalClosureContainsV0(request.Run.Deliveries, delivery.DeliveryRef) {
				continue
			}
			deliveries = append(deliveries, delivery)
		}
	}
	return deliveries
}

func codexStackOperationalClosureAcceptedReviewForDeliveryV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	deliveryRef string,
) (
	orquestacoreworkflow.ReviewRequestedPayloadV0,
	orquestacoreworkflow.ReviewAcceptedPayloadV0,
	orquestacoreworkflow.ReviewResultV0,
	bool,
) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, acceptedReviewRef := range request.Run.AcceptedReviews {
		accepted, ok := trace.AcceptedReviews[strings.TrimSpace(acceptedReviewRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(accepted.DeliveryRef) != deliveryRef ||
			!codexStackOperationalClosureContainsV0(request.Run.AcceptedReviews, accepted.AcceptedReviewRef) {
			continue
		}
		reviewRequest, ok := trace.ReviewRequests[strings.TrimSpace(accepted.ReviewRequestID)]
		if !ok || strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef {
			continue
		}
		result, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
		if ok {
			return reviewRequest, accepted, result, true
		}
	}
	return orquestacoreworkflow.ReviewRequestedPayloadV0{}, orquestacoreworkflow.ReviewAcceptedPayloadV0{}, orquestacoreworkflow.ReviewResultV0{}, false
}

func codexStackOperationalClosureAcceptedResultV0(
	trace codexStackOperationalClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == strings.TrimSpace(accepted.ReviewRequestID) &&
			strings.TrimSpace(result.DeliveryRef) == strings.TrimSpace(accepted.DeliveryRef) &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}
