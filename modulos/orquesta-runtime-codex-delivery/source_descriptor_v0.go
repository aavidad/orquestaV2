package orquestaruntimecodexdelivery

import (
	"context"
	"strings"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexReceiptDescriptorStorePortV0 interface {
	ListCodexReceiptDescriptorsV0(
		context.Context,
		CodexReceiptDescriptorRequestV0,
	) ([]CodexReceiptDescriptorV0, error)
}

type CodexReceiptDescriptorRecorderPortV0 interface {
	RecordCodexReceiptDescriptorV0(context.Context, CodexReceiptDescriptorV0) error
}

type CodexReceiptDescriptorRequestV0 struct {
	RunID          string
	StartedAgents  []string
	Deliveries     []string
	PhaseArtifacts []string
	CorrelationID  string
	EvidenceRefs   []string
}

type CodexReceiptDescriptorV0 struct {
	DescriptorRef                  string
	RunID                          string
	AgentRef                       string
	Spec                           orquestaruntime.ExternalAgentLaunchSpecV0
	AckPath                        string
	ProjectWorkDir                 string
	WorktreeBaselineRef            string
	DirectorDecisionSidecarReceipt *orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0
}

func codexReceiptDescriptorAlreadyReflectedV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
) bool {
	ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	if ackRef == "" {
		return false
	}
	return stringInCodexDeliverySetV0(request.Run.Deliveries, ackRef) ||
		codexReceiptArtifactRefRegisteredV0(request.Run.PhaseArtifacts, ackRef)
}

func codexReceiptDescriptorAgentRefV0(descriptor CodexReceiptDescriptorV0) string {
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	if agentRef != "" {
		return agentRef
	}
	agentRef = strings.TrimSpace(descriptor.Spec.RequestID)
	if agentRef != "" {
		return agentRef
	}
	return strings.TrimSpace(descriptor.Spec.AgentPacket.RequestID)
}

func codexReceiptDescriptorRequestFromNucleoV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) CodexReceiptDescriptorRequestV0 {
	return CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  codexDeliveryObservationStartedAgentsForDescriptorRequestV0(request),
		Deliveries:     compactCodexDeliveryRefsV0(request.Run.Deliveries),
		PhaseArtifacts: compactCodexDeliveryRefsV0(request.Run.PhaseArtifacts),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactCodexDeliveryRefsV0(request.EvidenceRefs),
	}
}

func codexDeliveryObservationStartedAgentsForDescriptorRequestV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) []string {
	startedAgents := compactCodexDeliveryRefsV0(request.Run.StartedAgents)
	waitAgentRefs := compactCodexDeliveryRefsV0(request.WaitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return startedAgents
	}
	scoped := make([]string, 0, len(startedAgents))
	for _, agentRef := range startedAgents {
		if stringInCodexDeliverySetV0(waitAgentRefs, agentRef) {
			scoped = append(scoped, agentRef)
		}
	}
	return scoped
}
