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

func TestMCPTransportV0OperadorConectorSimuladoSirveTodasLasOperaciones(t *testing.T) {
	connector := operator.NewOperatorMCPSimulatedConnectorV0(operator.OperatorMCPSimulatedConnectorConfigV0{})
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{OperatorConnector: connector}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	cases := []struct {
		name  string
		tool  string
		input any
		check func(MCPOperatorToolResultV0) bool
	}{
		{
			name: "status",
			tool: operator.OperatorMCPStatusToolNameV0,
			input: operator.OperatorStatusQueryV0{
				RequestRef:         "req-1",
				SubjectRef:         "run-1",
				StatusConnectorRef: "operator-state-connector-ref-simulated",
				IncludeSections:    []string{"health"},
			},
			check: func(result MCPOperatorToolResultV0) bool {
				return result.Status != nil && result.Status.Status == "simulated"
			},
		},
		{
			name: "outbox",
			tool: operator.OperatorMCPOutboxToolNameV0,
			input: operator.OperatorPendingOutboxQueryV0{
				RequestRef:         "req-1",
				SubjectRef:         "run-1",
				OutboxConnectorRef: "outbox-pending-connector-ref-simulated",
				Limit:              5,
				IncludeKinds:       []string{"question"},
			},
			check: func(result MCPOperatorToolResultV0) bool {
				return result.Outbox != nil && result.Outbox.PendingCount == 1
			},
		},
		{
			name: "query",
			tool: operator.OperatorMCPDirectedQueryToolV0,
			input: operator.OperatorDirectedQueryV0{
				QueryRef:          "query-1",
				TargetRef:         "director-1",
				QueryConnectorRef: "consulta-dirigida-connector-ref-simulated",
				Question:          "Que accion publica falta?",
			},
			check: func(result MCPOperatorToolResultV0) bool {
				return result.DirectedQuery != nil && result.DirectedQuery.Accepted
			},
		},
	}

	for _, tc := range cases {
		output, err := transport.CallToolV0(context.Background(), tc.tool, tc.input)
		if err != nil {
			t.Fatalf("%s call: %v", tc.name, err)
		}
		var result MCPOperatorToolResultV0
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("%s decode: %v", tc.name, err)
		}
		if result.Estado != MCPOperatorToolEstadoOKV0 || !tc.check(result) {
			t.Fatalf("%s result inesperado: %+v", tc.name, result)
		}
	}
}

func TestMCPTransportV0OperadorPropagaErrorPublicoDeConector(t *testing.T) {
	connector := operator.NewOperatorMCPSimulatedConnectorV0(operator.OperatorMCPSimulatedConnectorConfigV0{})
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{OperatorConnector: connector}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), operator.OperatorMCPOutboxToolNameV0, operator.OperatorPendingOutboxQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-connector-ref-externo",
		Limit:              5,
	})
	if err != nil {
		t.Fatalf("call outbox: %v", err)
	}
	var result MCPOperatorToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode outbox: %v", err)
	}
	if result.Estado != MCPOperatorToolEstadoErrorV0 ||
		result.ErrorCode != operator.ErrOperatorMCPConnectorUnavailableV0 {
		t.Fatalf("error publico esperado: %+v", result)
	}
}
