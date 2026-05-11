package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type CodexStackAgentUsageSourceV0 struct {
	Store           CodexReceiptStorePortV0
	ModelAlias      string
	ReasoningEffort string
}

var _ orquestacionnucleoapp.AgentUsageStatsProviderPortV0 = CodexStackAgentUsageSourceV0{}

func (source CodexStackAgentUsageSourceV0) BuildAgentUsageStatsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentUsageStatsRequestV0,
) ([]orquestacionnucleoapp.AgentUsageStatsObservationV0, error) {
	if source.Store == nil {
		return nil, nil
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(ctx, orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  compactStringsV0(request.Run.StartedAgents),
		Deliveries:     compactStringsV0(request.Run.Deliveries),
		PhaseArtifacts: compactStringsV0(request.Run.PhaseArtifacts),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactStringsV0(request.EvidenceRefs),
	})
	if err != nil {
		return nil, err
	}
	out := make([]orquestacionnucleoapp.AgentUsageStatsObservationV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, source.observationFromDescriptorV0(descriptor))
	}
	return out, nil
}

func (source CodexStackAgentUsageSourceV0) observationFromDescriptorV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.AgentUsageStatsObservationV0 {
	spec := descriptor.Spec
	return orquestacionnucleoapp.AgentUsageStatsObservationV0{
		AgentRequestID:  strings.TrimSpace(descriptor.AgentRef),
		RuntimeKind:     strings.TrimSpace(spec.RuntimeKind),
		ConnectorRef:    strings.TrimSpace(spec.ConnectorRef),
		ProfileRef:      strings.TrimSpace(spec.ProfileRef),
		ModelAlias:      strings.TrimSpace(source.ModelAlias),
		CapacityLevel:   strings.TrimSpace(spec.AgentPacket.CapacityLevel),
		ReasoningEffort: strings.TrimSpace(source.ReasoningEffort),
		QuotaStatus:     orquestacionnucleoapp.DirectorAgentUsageQuotaNotConfiguredV0,
		EvidenceRefs: compactStringsV0([]string{
			strings.TrimSpace(descriptor.DescriptorRef),
			strings.TrimSpace(spec.RequestID),
			strings.TrimSpace(spec.AgentPacket.WorkOrderRef),
		}),
	}
}
