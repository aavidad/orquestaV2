package orquestaruntime

import (
	"strings"
	"time"
)

type AgentLauncherRuntimeLaunchOptionsV0 struct {
	Locale                  string
	AdapterRef              string
	RequestedAt             string
	ReadinessTimeoutSeconds int
	MaxStartupSeconds       int
}

func LaunchRuntimeAgentToRuntimeLaunchRequestV0(
	inbound AgentLauncherInboundV0,
	resolved AgentLauncherResolvedDependenciesV0,
	options AgentLauncherRuntimeLaunchOptionsV0,
) (RuntimeLaunchRequestV0, []AgentLauncherInboundErrorV0) {
	if issues := agentLauncherTransformPreflightV0(inbound, resolved); len(issues) > 0 {
		return RuntimeLaunchRequestV0{}, issues
	}
	payload := *inbound.Payload
	request := RuntimeLaunchRequestV0{
		SchemaVersion:  RuntimeLaunchRequestSchemaVersionV0,
		RequestID:      strings.TrimSpace(payload.AgentRequestID),
		CorrelationID:  strings.TrimSpace(inbound.CorrelationID),
		IdempotencyKey: strings.TrimSpace(inbound.IdempotencyKey),
		RequestedAt:    runtimeLaunchRequestedAtV0(options.RequestedAt),
		Source: &RuntimeLaunchSourceV0{
			Module:     RuntimeLaunchSourceModuleCoreV0,
			AdapterRef: runtimeLaunchAdapterRefV0(options.AdapterRef),
		},
		Locale:     runtimeLaunchLocaleV0(options.Locale),
		LaunchMode: RuntimeLaunchModeNewSessionV0,
		Task: &RuntimeLaunchTaskV0{
			TaskRef:  strings.TrimSpace(payload.TaskRef),
			PhaseRef: strings.TrimSpace(payload.PhaseID),
			Priority: "normal",
		},
		FunctionContract: cloneRuntimeFunctionContractV0(resolved.FunctionContract),
		CapacityDecision: cloneRuntimeCapacityDecisionV0(resolved.CapacityDecision),
		RuntimeBinding:   cloneRuntimeBindingV0(resolved.RuntimeBinding),
		EvidenceRefs:     cloneRuntimeEvidenceRefsV0(resolved.EvidenceRefs),
		ContextBundle:    cloneRuntimeContextBundleV0(resolved.ContextBundle),
		Delivery: &RuntimeDeliveryV0{
			MailboxProtocol:         DeliveryMailboxProtocolV0,
			AckRequired:             true,
			ReadinessTimeoutSeconds: runtimeLaunchPositiveDefaultV0(options.ReadinessTimeoutSeconds, 120),
			MaxStartupSeconds:       runtimeLaunchPositiveDefaultV0(options.MaxStartupSeconds, 180),
		},
		Safety: &RuntimeSafetyV0{
			SecretsPolicy:   SafetyPolicyReferencesOnlyV0,
			HomePathsPolicy: SafetyPolicyOpaqueRefsOnlyV0,
			ProviderPolicy:  SafetyPolicyOpaqueRefsOnlyV0,
			WriteSetPolicy:  SafetyPolicyWriteSetClosedV0,
		},
	}
	if issues := ValidateRuntimeLaunchRequestV0(request); len(issues) > 0 {
		return request, runtimeLaunchIssuesAsAgentLauncherIssuesV0(request.CorrelationID, issues)
	}
	return request, nil
}

func agentLauncherTransformPreflightV0(
	inbound AgentLauncherInboundV0,
	resolved AgentLauncherResolvedDependenciesV0,
) []AgentLauncherInboundErrorV0 {
	issues := append([]AgentLauncherInboundErrorV0{}, ValidateAgentLauncherInboundV0(inbound)...)
	issues = append(issues, ValidateAgentLauncherResolvedDependenciesV0(resolved)...)
	return correlateAgentLauncherIssuesV0(inbound.CorrelationID, issues)
}

func runtimeLaunchRequestedAtV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func runtimeLaunchLocaleV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "es-ES"
	}
	return value
}

func runtimeLaunchAdapterRefV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "agent-launcher-adapter-v0"
	}
	return value
}

func runtimeLaunchPositiveDefaultV0(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
