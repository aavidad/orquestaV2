package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type CodexStackAgentUsageSourceV0 struct {
	Store           CodexReceiptStorePortV0
	ModelAlias      string
	ReasoningEffort string
	UsageMetrics    CodexStackAgentUsageMetricsProviderPortV0
}

var _ orquestacionnucleoapp.AgentUsageStatsProviderPortV0 = CodexStackAgentUsageSourceV0{}

type CodexStackAgentUsageMetricsProviderPortV0 interface {
	BuildCodexStackAgentUsageMetricsV0(
		context.Context,
		CodexStackAgentUsageMetricsRequestV0,
	) ([]CodexStackAgentUsageMetricV0, error)
}

type CodexStackAgentUsageMetricsRequestV0 struct {
	RunID         string
	AgentRefs     []string
	CorrelationID string
	EvidenceRefs  []string
}

type CodexStackAgentUsageMetricV0 struct {
	AgentRequestID   string
	QuotaStatus      string
	QuotaRemaining   int64
	QuotaLimit       int64
	QuotaResetAt     string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CostMicros       int64
	EvidenceRefs     []string
}

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
	metrics, err := source.metricsByAgentV0(ctx, request, descriptors)
	if err != nil {
		return nil, err
	}
	out := make([]orquestacionnucleoapp.AgentUsageStatsObservationV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, source.observationFromDescriptorV0(
			descriptor,
			metrics[strings.TrimSpace(descriptor.AgentRef)],
		))
	}
	return out, nil
}

func (source CodexStackAgentUsageSourceV0) observationFromDescriptorV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	metric CodexStackAgentUsageMetricV0,
) orquestacionnucleoapp.AgentUsageStatsObservationV0 {
	spec := descriptor.Spec
	quotaStatus := strings.TrimSpace(metric.QuotaStatus)
	if quotaStatus == "" {
		quotaStatus = orquestacionnucleoapp.DirectorAgentUsageQuotaNotConfiguredV0
	}
	return orquestacionnucleoapp.AgentUsageStatsObservationV0{
		AgentRequestID:   strings.TrimSpace(descriptor.AgentRef),
		RuntimeKind:      strings.TrimSpace(spec.RuntimeKind),
		ConnectorRef:     strings.TrimSpace(spec.ConnectorRef),
		ProfileRef:       strings.TrimSpace(spec.ProfileRef),
		ModelAlias:       strings.TrimSpace(source.ModelAlias),
		CapacityLevel:    strings.TrimSpace(spec.AgentPacket.CapacityLevel),
		ReasoningEffort:  strings.TrimSpace(source.ReasoningEffort),
		QuotaStatus:      quotaStatus,
		QuotaRemaining:   metric.QuotaRemaining,
		QuotaLimit:       metric.QuotaLimit,
		QuotaResetAt:     strings.TrimSpace(metric.QuotaResetAt),
		PromptTokens:     metric.PromptTokens,
		CompletionTokens: metric.CompletionTokens,
		TotalTokens:      metric.TotalTokens,
		CostMicros:       metric.CostMicros,
		EvidenceRefs: compactStringsV0(append([]string{
			strings.TrimSpace(descriptor.DescriptorRef),
			strings.TrimSpace(spec.RequestID),
			strings.TrimSpace(spec.AgentPacket.WorkOrderRef),
		}, metric.EvidenceRefs...)),
	}
}

func (source CodexStackAgentUsageSourceV0) metricsByAgentV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentUsageStatsRequestV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (map[string]CodexStackAgentUsageMetricV0, error) {
	out := map[string]CodexStackAgentUsageMetricV0{}
	if source.UsageMetrics == nil || len(descriptors) == 0 {
		return out, nil
	}
	metrics, err := source.UsageMetrics.BuildCodexStackAgentUsageMetricsV0(
		ctx,
		CodexStackAgentUsageMetricsRequestV0{
			RunID:         strings.TrimSpace(request.Run.RunID),
			AgentRefs:     codexStackUsageAgentRefsV0(descriptors),
			CorrelationID: strings.TrimSpace(request.CorrelationID),
			EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
		},
	)
	if err != nil {
		return nil, err
	}
	for _, metric := range metrics {
		agentRef := strings.TrimSpace(metric.AgentRequestID)
		if agentRef != "" {
			metric.AgentRequestID = agentRef
			out[agentRef] = metric
		}
	}
	return out, nil
}

func codexStackUsageAgentRefsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	refs := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		refs = append(refs, descriptor.AgentRef)
	}
	return compactStringsV0(refs)
}
