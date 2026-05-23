package orquestamcp

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPOperatorOperationsResourceV0ListaCapacidades(t *testing.T) {
	resource := NewMCPOperatorOperationsResourceV0()
	if resource.URI != MCPOperatorOperationsResourceURIV0 || resource.SchemaVersion != operator.OperatorMCPSchemaVersionV0 {
		t.Fatalf("resource identidad: %+v", resource)
	}
	for _, name := range []string{
		operator.OperatorMCPStatusToolNameV0,
		operator.OperatorMCPBurstToolNameV0,
		operator.OperatorMCPOutboxToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
	} {
		if !operatorToolListedMCPTestV0(resource.Tools, name) {
			t.Fatalf("capacidad no listada: %s en %+v", name, resource.Tools)
		}
	}
	assertOperatorPayloadSaneadoMCPTestV0(t, resource)
}

func TestExecuteMCPOperatorBurstToolV0ValidaMaxStepsYRefs(t *testing.T) {
	fake := &fakeOperatorBurstPortMCPTestV0{}
	result := ExecuteMCPOperatorBurstToolV0(fake, operator.OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-1",
		RunRef:            "run..1",
		BurstConnectorRef: "burst-connector-1",
		SupervisionRef:    "supervision-1",
		MaxSteps:          operator.OperatorMCPMaxBurstStepsLimitV0 + 1,
	})
	if result.Estado != MCPOperatorToolEstadoErrorV0 || len(result.Issues) < 2 || fake.called != 0 {
		t.Fatalf("validacion esperada sin ejecutar puerto: result=%+v called=%d", result, fake.called)
	}
	assertOperatorPayloadSaneadoMCPTestV0(t, result)
}

func TestExecuteMCPOperatorToolsV0DeleganSoloConPuertoFake(t *testing.T) {
	input := validOperatorBurstInputMCPTestV0()
	withoutPort := ExecuteMCPOperatorBurstToolV0(nil, input)
	if withoutPort.Estado != MCPOperatorToolEstadoErrorV0 || withoutPort.ErrorCode != "operator_mcp_port_unavailable" {
		t.Fatalf("sin puerto debe fallar: %+v", withoutPort)
	}

	fake := &fakeOperatorBurstPortMCPTestV0{}
	withPort := ExecuteMCPOperatorBurstToolV0(fake, input)
	if withPort.Estado != MCPOperatorToolEstadoOKV0 || fake.called != 1 ||
		withPort.Burst == nil || withPort.Burst.BurstRef != "burst-ref-1" {
		t.Fatalf("delegacion esperada: result=%+v called=%d", withPort, fake.called)
	}
	assertOperatorPayloadSaneadoMCPTestV0(t, withPort)
}

func TestExecuteMCPOperatorOutboxToolV0ListaPendientesPorPuerto(t *testing.T) {
	result := ExecuteMCPOperatorOutboxToolV0(&fakeOperatorOutboxPortMCPTestV0{}, operator.OperatorPendingOutboxQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-connector-1",
		Limit:              5,
	})
	if result.Estado != MCPOperatorToolEstadoOKV0 || result.Outbox == nil ||
		result.Outbox.PendingCount != 1 || len(result.Outbox.Items) != 1 {
		t.Fatalf("outbox compacto esperado: %+v", result)
	}
	assertOperatorPayloadSaneadoMCPTestV0(t, result)
}

func TestExecuteMCPOperatorToolV0PreservaErrorPublicoDeConector(t *testing.T) {
	result := ExecuteMCPOperatorBurstToolV0(fakeOperatorErrorPortMCPTestV0{}, validOperatorBurstInputMCPTestV0())
	if result.Estado != MCPOperatorToolEstadoErrorV0 ||
		result.ErrorCode != operator.ErrOperatorMCPConnectorUnavailableV0 {
		t.Fatalf("error publico esperado: %+v", result)
	}

	result = ExecuteMCPOperatorBurstToolV0(fakeOperatorGenericErrorPortMCPTestV0{}, validOperatorBurstInputMCPTestV0())
	if result.Estado != MCPOperatorToolEstadoErrorV0 ||
		result.ErrorCode != operator.ErrOperatorMCPPortErrorV0 {
		t.Fatalf("error generico esperado: %+v", result)
	}
}

type fakeOperatorBurstPortMCPTestV0 struct{ called int }

func (f *fakeOperatorBurstPortMCPTestV0) RequestOperatorSupervisedBurstV0(
	operator.OperatorSupervisedBurstRequestV0,
) (operator.OperatorMCPBurstResultV0, error) {
	f.called++
	return operator.OperatorMCPBurstResultV0{BurstRef: "burst-ref-1", ExecutedSteps: 2, TraceRefs: []string{"trace-ref-1"}}, nil
}

type fakeOperatorErrorPortMCPTestV0 struct{}

func (fakeOperatorErrorPortMCPTestV0) RequestOperatorSupervisedBurstV0(
	operator.OperatorSupervisedBurstRequestV0,
) (operator.OperatorMCPBurstResultV0, error) {
	return operator.OperatorMCPBurstResultV0{},
		operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
}

type fakeOperatorGenericErrorPortMCPTestV0 struct{}

func (fakeOperatorGenericErrorPortMCPTestV0) RequestOperatorSupervisedBurstV0(
	operator.OperatorSupervisedBurstRequestV0,
) (operator.OperatorMCPBurstResultV0, error) {
	return operator.OperatorMCPBurstResultV0{}, errors.New("local adapter failed")
}

type fakeOperatorOutboxPortMCPTestV0 struct{}

func (fakeOperatorOutboxPortMCPTestV0) ListOperatorPendingOutboxV0(
	operator.OperatorPendingOutboxQueryV0,
) (operator.OperatorMCPOutboxResultV0, error) {
	return operator.OperatorMCPOutboxResultV0{
		WatermarkRef: "watermark-ref-1",
		PendingCount: 1,
		Items: []operator.OperatorMCPOutboxItemV0{{
			MessageRef: "message-ref-1", Kind: "director_question", TargetRef: "director-ref-1",
		}},
	}, nil
}

func validOperatorBurstInputMCPTestV0() operator.OperatorSupervisedBurstRequestV0 {
	return operator.OperatorSupervisedBurstRequestV0{
		RequestRef: "req-1", RunRef: "run-1", BurstConnectorRef: "burst-connector-1",
		SupervisionRef: "supervision-1", MaxSteps: 2,
	}
}

func operatorToolListedMCPTestV0(tools []operator.OperatorMCPToolDescriptorV0, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func assertOperatorPayloadSaneadoMCPTestV0(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{"provider", "model", "/home/", "home=", "token", "password"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("payload contiene %q: %s", forbidden, text)
		}
	}
}
