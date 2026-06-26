package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func assessmentReplanProjectionAlreadyCoveredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef != "" && stringInSetV0(run.ClosedTasks, taskRef) {
		return true
	}
	deliveryRef := assessmentReplanDeliveryRefV0(projection, taskRef, descriptors)
	if deliveryRef == "" || !stringInSetV0(run.Deliveries, deliveryRef) {
		return false
	}
	return stringInSetV0(run.AcceptedReviews, "accepted-review-ref-"+deliveryRef) ||
		codexStackReviewGateReworkAcceptanceAlreadyDoneV0(run, deliveryRef)
}

func assessmentReplanDeliveryRefV0(
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	if deliveryRef := strings.TrimSpace(projection.DeliveryRef); deliveryRef != "" {
		return deliveryRef
	}
	agentRef := strings.TrimSpace(projection.AgentRequestID)
	taskRef = strings.TrimSpace(taskRef)
	for _, descriptor := range descriptors {
		if agentRef != "" && assessmentReplanDescriptorAgentRefV0(descriptor) != agentRef {
			continue
		}
		if taskRef != "" &&
			strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) != taskRef {
			continue
		}
		if deliveryRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef); deliveryRef != "" {
			return deliveryRef
		}
	}
	return ""
}

func assessmentReplanDescriptorAgentRefV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	for _, candidate := range []string{
		descriptor.AgentRef,
		descriptor.Spec.RequestID,
		descriptor.Spec.AgentPacket.RequestID,
	} {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			return candidate
		}
	}
	return ""
}
