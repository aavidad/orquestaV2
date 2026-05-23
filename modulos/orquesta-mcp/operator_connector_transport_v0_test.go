package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPTransportV0OperadorSinConectorDevuelveErrorPublico(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), operator.OperatorMCPStatusToolNameV0, operator.OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		StatusConnectorRef: "status-connector-ref-simulated",
	})
	if err != nil {
		t.Fatalf("call status: %v", err)
	}
	var result MCPOperatorToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if result.Estado != MCPOperatorToolEstadoErrorV0 || result.ErrorCode != "operator_mcp_port_unavailable" {
		t.Fatalf("sin conector inesperado: %+v", result)
	}
}

func TestMCPTransportV0OperadorUsaConectorSimuladoAgregado(t *testing.T) {
	connector := operator.NewOperatorMCPSimulatedConnectorV0(operator.OperatorMCPSimulatedConnectorConfigV0{})
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{OperatorConnector: connector}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), operator.OperatorMCPBurstToolNameV0, operator.OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-1",
		RunRef:            "run-1",
		BurstConnectorRef: "burst-connector-ref-simulated",
		SupervisionRef:    "supervision-1",
		MaxSteps:          3,
	})
	if err != nil {
		t.Fatalf("call burst: %v", err)
	}
	var result MCPOperatorToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode burst: %v", err)
	}
	if result.Estado != MCPOperatorToolEstadoOKV0 || result.Burst == nil ||
		result.Burst.ExecutedSteps != 2 {
		t.Fatalf("conector simulado inesperado: %+v", result)
	}
}
