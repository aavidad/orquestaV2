package orquestaruntime

import (
	"encoding/json"
	"testing"
)

func TestValidateAgentLauncherInboundV0AceptaOrdenCompacta(t *testing.T) {
	inbound := agentLauncherInboundValidoV0()

	issues := ValidateAgentLauncherInboundV0(inbound)
	if len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestAgentLauncherInboundV0JSONRoundTripMantieneContrato(t *testing.T) {
	raw, err := json.Marshal(agentLauncherInboundValidoV0())
	if err != nil {
		t.Fatalf("marshal inbound: %v", err)
	}

	var inbound AgentLauncherInboundV0
	if err := json.Unmarshal(raw, &inbound); err != nil {
		t.Fatalf("unmarshal inbound: %v", err)
	}

	if issues := ValidateAgentLauncherInboundV0(inbound); len(issues) != 0 {
		t.Fatalf("errores inesperados tras JSON round-trip: %#v", issues)
	}
	if inbound.TargetPort != AgentLauncherTargetPortV0 {
		t.Fatalf("target_port = %q", inbound.TargetPort)
	}
	if inbound.MessageType != AgentLauncherMessageTypeV0 {
		t.Fatalf("message_type = %q", inbound.MessageType)
	}
}

func TestValidateAgentLauncherInboundV0RechazaTargetInvalido(t *testing.T) {
	inbound := agentLauncherInboundValidoV0()
	inbound.TargetPort = "otro_target"

	requireAgentLauncherCodeV0(t, ValidateAgentLauncherInboundV0(inbound), AgentLauncherTargetInvalidoV0)
}

func TestValidateAgentLauncherInboundV0RechazaOperacionInvalida(t *testing.T) {
	inbound := agentLauncherInboundValidoV0()
	inbound.MessageType = "OtraOperacion"

	requireAgentLauncherCodeV0(t, ValidateAgentLauncherInboundV0(inbound), AgentLauncherOperacionInvalidaV0)
}

func TestValidateAgentLauncherInboundV0RechazaPayloadAusente(t *testing.T) {
	inbound := agentLauncherInboundValidoV0()
	inbound.Payload = nil

	requireAgentLauncherCodeV0(t, ValidateAgentLauncherInboundV0(inbound), AgentLauncherPayloadInvalidoV0)
}

func TestValidateLaunchRuntimeAgentRequestV0RechazaFunctionContractRefAusente(t *testing.T) {
	req := launchRuntimeAgentRequestValidaV0()
	req.TaskRef = ""

	requireAgentLauncherCodeV0(t, ValidateLaunchRuntimeAgentRequestV0(req), AgentLauncherFunctionRefRequeridaV0)
}

func TestValidateLaunchRuntimeAgentRequestV0RechazaCapacityDecisionRefAusente(t *testing.T) {
	req := launchRuntimeAgentRequestValidaV0()
	req.CapacityRequestRef = ""

	requireAgentLauncherCodeV0(t, ValidateLaunchRuntimeAgentRequestV0(req), AgentLauncherCapacityRefRequeridaV0)
}

func TestValidateLaunchRuntimeAgentRequestV0RechazaReferenciasNoOpacas(t *testing.T) {
	req := launchRuntimeAgentRequestValidaV0()
	req.EvidenceRefs = []string{"evidence/ref"}

	requireAgentLauncherCodeV0(t, ValidateLaunchRuntimeAgentRequestV0(req), AgentLauncherReferenciaNoOpacaV0)
}

func TestValidateLaunchRuntimeAgentRequestV0RechazaSecretos(t *testing.T) {
	req := launchRuntimeAgentRequestValidaV0()
	req.AgentRequestID = "sk-test-token"

	requireAgentLauncherCodeV0(t, ValidateLaunchRuntimeAgentRequestV0(req), AgentLauncherSecretoDetectadoV0)
}

func TestValidateAgentLauncherResolvedDependenciesV0DeclaraBloqueosDeEnriquecimiento(t *testing.T) {
	issues := ValidateAgentLauncherResolvedDependenciesV0(AgentLauncherResolvedDependenciesV0{})

	requireAgentLauncherCodeV0(t, issues, AgentLauncherFunctionNoResueltaV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherCapacityNoResueltaV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherRuntimeBindingRefsV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherEvidenceRefsV0)
	requireAgentLauncherCodeV0(t, issues, AgentLauncherContextBundleNoResueltoV0)
}

func requireAgentLauncherCodeV0(t *testing.T, issues []AgentLauncherInboundErrorV0, code AgentLauncherInboundErrorCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func agentLauncherInboundValidoV0() AgentLauncherInboundV0 {
	req := launchRuntimeAgentRequestValidaV0()
	return AgentLauncherInboundV0{
		TargetPort:     AgentLauncherTargetPortV0,
		MessageType:    AgentLauncherMessageTypeV0,
		CorrelationID:  "corr-agent-launcher-001",
		IdempotencyKey: "idem-agent-launcher-001",
		Payload:        &req,
	}
}

func launchRuntimeAgentRequestValidaV0() LaunchRuntimeAgentRequestV0 {
	return LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-ref-001",
		RunID:              "run-ref-001",
		PhaseID:            "phase-ref-implementacion",
		TaskRef:            "task-ref-001",
		CapacityRequestRef: "capacity-request-ref-001",
		Role:               "implementador-runtime",
		Summary:            "Orden compacta para preparar lanzamiento runtime sin transporte real.",
		EvidenceRefs: []string{
			"business-evidence-ref-001",
			"workflow-evidence-ref-001",
		},
	}
}
