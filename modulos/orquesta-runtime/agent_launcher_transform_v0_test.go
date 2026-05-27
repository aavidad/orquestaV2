package orquestaruntime

import (
	"testing"
	"time"
)

func TestLaunchRuntimeAgentToRuntimeLaunchRequestV0ConstruyeRequestValida(t *testing.T) {
	base := runtimeLaunchRequestValidaV0()
	inbound := agentLauncherInboundValidoV0()
	resolved := AgentLauncherResolvedDependenciesV0{
		FunctionContract: base.FunctionContract,
		CapacityDecision: base.CapacityDecision,
		RuntimeBinding:   base.RuntimeBinding,
		EvidenceRefs:     base.EvidenceRefs,
		ContextBundle:    base.ContextBundle,
	}

	req, issues := LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		inbound,
		resolved,
		AgentLauncherRuntimeLaunchOptionsV0{
			Locale:                  "es-ES",
			AdapterRef:              "adapter-ref-agent-launcher-001",
			RequestedAt:             "2026-05-08T10:00:00Z",
			ReadinessTimeoutSeconds: 45,
			MaxStartupSeconds:       90,
		},
	)
	if len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
	if !req.Valid() {
		t.Fatalf("request runtime invalido: %#v", ValidateRuntimeLaunchRequestV0(req))
	}
	if req.RequestID != inbound.Payload.AgentRequestID ||
		req.CorrelationID != inbound.CorrelationID ||
		req.IdempotencyKey != inbound.IdempotencyKey {
		t.Fatalf("trazabilidad inesperada: %+v", req)
	}
	if req.Source == nil || req.Source.Module != RuntimeLaunchSourceModuleCoreV0 {
		t.Fatalf("source inesperado: %+v", req.Source)
	}
	if req.Task == nil || req.Task.TaskRef != inbound.Payload.TaskRef || req.Task.Priority != "normal" {
		t.Fatalf("task inesperada: %+v", req.Task)
	}
	if req.Delivery == nil || !req.Delivery.AckRequired || req.Delivery.ReadinessTimeoutSeconds != 45 {
		t.Fatalf("delivery inesperado: %+v", req.Delivery)
	}
	if req.Safety == nil || req.Safety.WriteSetPolicy != SafetyPolicyWriteSetClosedV0 {
		t.Fatalf("safety inesperada: %+v", req.Safety)
	}
}

func TestLaunchRuntimeAgentToRuntimeLaunchRequestV0BloqueaDependenciasIncompletas(t *testing.T) {
	_, issues := LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		agentLauncherInboundValidoV0(),
		AgentLauncherResolvedDependenciesV0{},
		AgentLauncherRuntimeLaunchOptionsV0{RequestedAt: "2026-05-08T10:00:00Z"},
	)

	requireAgentLauncherCodeV0(t, issues, AgentLauncherFunctionNoResueltaV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherCapacityNoResueltaV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherRuntimeBindingRefsV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherEvidenceRefsV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherContextBundleNoResueltoV0)
}

func TestLaunchRuntimeAgentToRuntimeLaunchRequestV0UsaRelojInyectadoV0(t *testing.T) {
	base := runtimeLaunchRequestValidaV0()
	resolved := AgentLauncherResolvedDependenciesV0{
		FunctionContract: base.FunctionContract,
		CapacityDecision: base.CapacityDecision,
		RuntimeBinding:   base.RuntimeBinding,
		EvidenceRefs:     base.EvidenceRefs,
		ContextBundle:    base.ContextBundle,
	}

	req, issues := LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		agentLauncherInboundValidoV0(),
		resolved,
		AgentLauncherRuntimeLaunchOptionsV0{
			Clock: ClockFuncV0(func() time.Time {
				return time.Date(2026, 5, 24, 10, 11, 12, 0, time.UTC)
			}),
		},
	)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if req.RequestedAt != "2026-05-24T10:11:12Z" {
		t.Fatalf("requested_at=%q", req.RequestedAt)
	}
}

func TestLaunchRuntimeAgentToRuntimeLaunchRequestV0PropagaValidacionRuntime(t *testing.T) {
	base := runtimeLaunchRequestValidaV0()
	binding := *base.RuntimeBinding
	binding.ModelRef = "model-ref-distinto"
	resolved := AgentLauncherResolvedDependenciesV0{
		FunctionContract: base.FunctionContract,
		CapacityDecision: base.CapacityDecision,
		RuntimeBinding:   &binding,
		EvidenceRefs:     base.EvidenceRefs,
		ContextBundle:    base.ContextBundle,
	}

	_, issues := LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		agentLauncherInboundValidoV0(),
		resolved,
		AgentLauncherRuntimeLaunchOptionsV0{RequestedAt: "2026-05-08T10:00:00Z"},
	)

	requireAgentLauncherCodeV0(t, issues, AgentLauncherRuntimeLaunchInvalidaV0)
	if len(issues) == 0 || issues[0].CorrelationID == "" {
		t.Fatalf("la validacion debe conservar correlation_id: %#v", issues)
	}
}
