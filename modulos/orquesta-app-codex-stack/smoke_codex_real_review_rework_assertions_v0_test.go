package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func codexStackRealSmokeAssertReworkProjectedV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
	reworkAgentRef string,
) {
	t.Helper()
	rework, reworkOK := codexStackRealSmokeReworkForDeliveryV0(run, deliveryRef)
	if !codexStackRefsContainPartV0(run.Reviews, deliveryRef) ||
		!reworkOK ||
		!codexStackRealSmokeReviewResultNeedsReworkV0(run, rework) ||
		!codexStackRealSmokeReplanForReworkAgentV0(run, rework.ReworkRequestRef, reworkAgentRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, reworkAgentRef) {
		t.Fatalf("rework real no proyectado delivery=%s reviews=%v results=%v reworks=%v replan=%v started=%v",
			deliveryRef,
			run.Reviews,
			run.ReviewResults,
			run.ReworkRequests,
			run.ReplanDecisions,
			run.StartedAgents,
		)
	}
}

func codexStackRealSmokeReworkDescriptorV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	original orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	taskRef := strings.TrimSpace(original.Spec.AgentPacket.Task.TaskRef)
	originalAckRef := strings.TrimSpace(original.Spec.AgentPacket.DeliveryRefs.AckRef)
	for _, descriptor := range descriptors {
		if !codexStackRealSmokeIsReworkDescriptorV0(run, descriptor, taskRef, originalAckRef) {
			continue
		}
		return descriptor
	}
	t.Fatalf("descriptor de rework no encontrado: %v", codexStackRealSmokeDescriptorAgentsV0(descriptors))
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

func codexStackRealSmokeIsReworkDescriptorV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	taskRef string,
	originalAckRef string,
) bool {
	if strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) != taskRef {
		return false
	}
	if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) == originalAckRef {
		return false
	}
	if strings.TrimSpace(descriptor.Spec.AgentPacket.Phase) != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		return false
	}
	rework, ok := codexStackRealSmokeReworkForDeliveryV0(run, originalAckRef)
	if !ok || !codexStackRealSmokeReviewResultNeedsReworkV0(run, rework) {
		return false
	}
	agentRef := codexStackRealSmokeDescriptorAgentRefV0(descriptor)
	return codexStackStringInSetForTestV0(run.StartedAgents, agentRef) &&
		codexStackRealSmokeReplanForReworkTaskAgentV0(run, rework.ReworkRequestRef, taskRef, agentRef)
}

func codexStackRealSmokeReworkForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) (reviewReworkProjectionV0, bool) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, raw := range run.ReworkRequests {
		rework, ok := parseReviewReworkProjectionV0(raw)
		if ok && rework.DeliveryRef == deliveryRef {
			return rework, true
		}
	}
	return reviewReworkProjectionV0{}, false
}

func codexStackRealSmokeReviewResultNeedsReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	rework reviewReworkProjectionV0,
) bool {
	result, ok := reviewResultForReworkV0(run, rework)
	return ok && result.Status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0
}

func codexStackRealSmokeReplanForReworkAgentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reworkRef string,
	agentRef string,
) bool {
	for _, raw := range run.ReplanDecisions {
		replan, ok := codexStackRealSmokeParseReplanProjectionV0(raw)
		if ok && replan.SourceRef == reworkRef &&
			replan.Action == string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) &&
			codexStackStringInSetForTestV0(replan.FollowupRefs, agentRef) {
			return true
		}
	}
	return false
}

func codexStackRealSmokeReplanForReworkTaskAgentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reworkRef string,
	taskRef string,
	agentRef string,
) bool {
	for _, raw := range run.ReplanDecisions {
		replan, ok := codexStackRealSmokeParseReplanProjectionV0(raw)
		if ok && replan.SourceRef == reworkRef &&
			replan.TaskRef == taskRef &&
			replan.Action == string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) &&
			codexStackStringInSetForTestV0(replan.FollowupRefs, agentRef) {
			return true
		}
	}
	return false
}

type codexStackRealSmokeReplanProjectionV0 struct {
	SourceRef    string
	TaskRef      string
	Action       string
	FollowupRefs []string
}

func codexStackRealSmokeParseReplanProjectionV0(
	value string,
) (codexStackRealSmokeReplanProjectionV0, bool) {
	_, tail, ok := strings.Cut(strings.TrimSpace(value), "#source:")
	if !ok {
		return codexStackRealSmokeReplanProjectionV0{}, false
	}
	sourceRef, tail, ok := strings.Cut(tail, "#task:")
	if !ok {
		return codexStackRealSmokeReplanProjectionV0{}, false
	}
	taskRef, tail, ok := strings.Cut(tail, "#action:")
	if !ok {
		return codexStackRealSmokeReplanProjectionV0{}, false
	}
	action, followups, ok := strings.Cut(tail, "#followups:")
	if !ok {
		return codexStackRealSmokeReplanProjectionV0{}, false
	}
	replan := codexStackRealSmokeReplanProjectionV0{
		SourceRef:    strings.TrimSpace(sourceRef),
		TaskRef:      strings.TrimSpace(taskRef),
		Action:       strings.TrimSpace(action),
		FollowupRefs: compactStringsV0(strings.Split(followups, "+")),
	}
	return replan, replan.SourceRef != "" && replan.TaskRef != "" &&
		replan.Action != "" && len(replan.FollowupRefs) > 0
}

func codexStackRealSmokeDescriptorAgentRefV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	if value := strings.TrimSpace(descriptor.AgentRef); value != "" {
		return value
	}
	return strings.TrimSpace(descriptor.Spec.RequestID)
}

func codexStackRealSmokeAssertReviewGateAcceptedV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) {
	t.Helper()
	if codexStackRefsContainPartV0(run.AcceptedReviews, deliveryRef) {
		return
	}
	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		ctx,
		orquestacionnucleoapp.ReviewGateObservationRequestV0{
			Run:           run,
			CorrelationID: "corr-app-stack-real-review-rework-quality",
			EvidenceRefs:  []string{"evidence-ref-app-stack-real-review-rework-quality"},
		},
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0 rework: %v", err)
	}
	for _, observation := range observations {
		if observation.DeliveryRef != deliveryRef {
			continue
		}
		if observation.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			t.Fatalf("review gate rework no aceptado delivery=%s observation=%+v", deliveryRef, observation)
		}
		return
	}
	t.Fatalf("review gate no evaluo rework delivery=%s observations=%+v", deliveryRef, observations)
}
