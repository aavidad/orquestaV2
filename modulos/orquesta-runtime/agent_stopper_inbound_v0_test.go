package orquestaruntime

import (
	"encoding/json"
	"testing"
)

func TestValidateAgentStopperInboundV0AceptaOrdenCompacta(t *testing.T) {
	inbound := agentStopperInboundValidoV0()

	issues := ValidateAgentStopperInboundV0(inbound)
	if len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestAgentStopperInboundV0JSONRoundTripMantieneContrato(t *testing.T) {
	raw, err := json.Marshal(agentStopperInboundValidoV0())
	if err != nil {
		t.Fatalf("marshal inbound: %v", err)
	}

	var inbound AgentStopperInboundV0
	if err := json.Unmarshal(raw, &inbound); err != nil {
		t.Fatalf("unmarshal inbound: %v", err)
	}

	if issues := ValidateAgentStopperInboundV0(inbound); len(issues) != 0 {
		t.Fatalf("errores inesperados tras JSON round-trip: %#v", issues)
	}
	if inbound.TargetPort != AgentStopperTargetPortV0 {
		t.Fatalf("target_port = %q", inbound.TargetPort)
	}
	if inbound.MessageType != AgentStopperMessageTypeV0 {
		t.Fatalf("message_type = %q", inbound.MessageType)
	}
}

func TestValidateAgentStopperInboundV0RechazaTargetInvalido(t *testing.T) {
	inbound := agentStopperInboundValidoV0()
	inbound.TargetPort = "otro_target"

	requireAgentStopperCodeV0(t, ValidateAgentStopperInboundV0(inbound), AgentStopperTargetInvalidoV0)
}

func TestValidateAgentStopperInboundV0RechazaOperacionInvalida(t *testing.T) {
	inbound := agentStopperInboundValidoV0()
	inbound.MessageType = "OtraOperacion"

	requireAgentStopperCodeV0(t, ValidateAgentStopperInboundV0(inbound), AgentStopperOperacionInvalidaV0)
}

func TestValidateAgentStopperInboundV0RechazaPayloadAusente(t *testing.T) {
	inbound := agentStopperInboundValidoV0()
	inbound.Payload = nil

	requireAgentStopperCodeV0(t, ValidateAgentStopperInboundV0(inbound), AgentStopperPayloadInvalidoV0)
}

func TestValidateStopRuntimeAgentRequestV0RechazaReferenciasNoOpacas(t *testing.T) {
	req := stopRuntimeAgentRequestValidaV0()
	req.EvidenceRefs = []string{"evidence/ref"}

	requireAgentStopperCodeV0(t, ValidateStopRuntimeAgentRequestV0(req), AgentStopperReferenciaNoOpacaV0)
}

func TestValidateStopRuntimeAgentRequestV0RechazaSecretos(t *testing.T) {
	req := stopRuntimeAgentRequestValidaV0()
	req.AgentRequestID = "sk-test-token"

	requireAgentStopperCodeV0(t, ValidateStopRuntimeAgentRequestV0(req), AgentStopperSecretoDetectadoV0)
}

func requireAgentStopperCodeV0(t *testing.T, issues []AgentStopperInboundErrorV0, code AgentStopperInboundErrorCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func agentStopperInboundValidoV0() AgentStopperInboundV0 {
	req := stopRuntimeAgentRequestValidaV0()
	return AgentStopperInboundV0{
		TargetPort:     AgentStopperTargetPortV0,
		MessageType:    AgentStopperMessageTypeV0,
		CorrelationID:  "corr-agent-stopper-001",
		IdempotencyKey: "idem-agent-stopper-001",
		Payload:        &req,
	}
}

func stopRuntimeAgentRequestValidaV0() StopRuntimeAgentRequestV0 {
	return StopRuntimeAgentRequestV0{
		AgentRequestID: "agent-request-ref-001",
		RunID:          "run-ref-001",
		ReasonCode:     "operator.logical_stop",
		Summary:        "Orden compacta para solicitar parada logica sin tocar procesos reales.",
		EvidenceRefs: []string{
			"operator-evidence-ref-001",
			"workflow-evidence-ref-001",
		},
	}
}
