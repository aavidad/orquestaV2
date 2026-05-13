package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestRegisterMCPTransportV0ExponeOperacionesExistentes(t *testing.T) {
	fakeStatus := &fakeTransportStatusPortMCPV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{OperatorStatus: fakeStatus})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	for _, name := range []string{
		MCPSharedContractsResourceNameV0,
		MCPProjectRoadmapResourceNameV0,
		MCPOperationalStatusResourceNameV0,
		MCPBootstrapResourceNameV0,
		MCPCoreWorkflowContractsResourceNameV0,
		MCPOperatorOperationsResourceNameV0,
	} {
		if _, ok := transport.resources[name]; !ok {
			t.Fatalf("resource no registrado: %s", name)
		}
	}
	for _, name := range []string{
		MCPNuevaAppToolNameV0,
		MCPArrancarDirectorAppToolNameV0,
		MCPDirectorAgentDecisionToolNameV0,
		MCPDirectorStatsToolNameV0,
		MCPPrepararOrquestacionAppToolNameV0,
		MCPEjecutarOrquestacionAppToolNameV0,
		MCPAutoprogrammingValidateRequestToolNameV0,
		MCPBootstrapToolNameV0,
		MCPCoreWorkflowCommandToolNameV0,
		MCPRunControlToolNameV0,
		MCPRunQueuePriorityToolNameV0,
		MCPServerShutdownToolNameV0,
		MCPDomainWorkToolNameV0,
		operator.OperatorMCPStatusToolNameV0,
		operator.OperatorMCPBurstToolNameV0,
		operator.OperatorMCPOutboxToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
	} {
		if _, ok := transport.tools[name]; !ok {
			t.Fatalf("tool no registrado: %s", name)
		}
	}
	assertTransportPayloadSaneadoMCPTestV0(t, transport.resources, 5000)
	assertTransportPayloadSaneadoMCPTestV0(t, transport.tools, 8000)
}

func TestMCPTransportV0SirveResourceYToolConFakeEnMemoria(t *testing.T) {
	fakeStatus := &fakeTransportStatusPortMCPV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{OperatorStatus: fakeStatus})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	resourcePayload, err := transport.ReadResourceV0(context.Background(), MCPOperatorOperationsResourceNameV0)
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(resourcePayload), 2500)

	input := operator.OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		StatusConnectorRef: "status-connector-1",
		IncludeSections:    []string{"summary"},
	}
	output, err := transport.CallToolV0(context.Background(), operator.OperatorMCPStatusToolNameV0, input)
	if err != nil {
		t.Fatalf("call status tool: %v", err)
	}
	if fakeStatus.called != 1 {
		t.Fatalf("puerto fake no invocado: %d", fakeStatus.called)
	}
	var result MCPOperatorToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPOperatorToolEstadoOKV0 || result.Status == nil || result.Status.Status != "healthy" {
		t.Fatalf("resultado status inesperado: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1000)
}

func TestMCPTransportV0NuevaAppQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPNuevaAppToolNameV0, MCPNuevaAppToolInputV0{})
	if err != nil {
		t.Fatalf("call nueva app unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("nueva app debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0ArrancarDirectorQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPArrancarDirectorAppToolNameV0, MCPArrancarDirectorAppToolInputV0{})
	if err != nil {
		t.Fatalf("call arrancar director unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("arrancar director debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0DirectorAgentDecisionQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPDirectorAgentDecisionToolNameV0, MCPDirectorAgentDecisionToolInputV0{})
	if err != nil {
		t.Fatalf("call director decision unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("director decision debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0EjecutarOrquestacionQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPEjecutarOrquestacionAppToolNameV0, MCPEjecutarOrquestacionAppToolInputV0{})
	if err != nil {
		t.Fatalf("call ejecutar orquestacion unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("ejecutar orquestacion debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0DomainWorkQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	tool, ok := transport.tools[MCPDomainWorkToolNameV0]
	if !ok {
		t.Fatalf("domain work no registrado: %s", MCPDomainWorkToolNameV0)
	}
	descriptor := MCPDomainWorkDescriptorV0()
	if tool.ResourceURI != descriptor.ResourceURI ||
		tool.InputShape != descriptor.InputSchema ||
		tool.OutputShape != descriptor.Output ||
		tool.Mode != MCPTransportModeOptInV0 {
		t.Fatalf("domain work envelope inesperado: %+v", tool)
	}

	output, err := transport.CallToolV0(context.Background(), MCPDomainWorkToolNameV0, MCPDomainWorkToolInputV0{})
	if err != nil {
		t.Fatalf("call domain work unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPDomainWorkToolNameV0 || result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("domain work debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

type fakeMCPTransportV0 struct {
	resources map[string]MCPTransportResourceEnvelopeV0
	tools     map[string]MCPTransportToolEnvelopeV0
}

func newFakeMCPTransportV0() *fakeMCPTransportV0 {
	return &fakeMCPTransportV0{
		resources: map[string]MCPTransportResourceEnvelopeV0{},
		tools:     map[string]MCPTransportToolEnvelopeV0{},
	}
}

func (f *fakeMCPTransportV0) RegisterResourceV0(resource MCPTransportResourceEnvelopeV0) error {
	f.resources[resource.Name] = resource
	return nil
}

func (f *fakeMCPTransportV0) RegisterToolV0(tool MCPTransportToolEnvelopeV0) error {
	f.tools[tool.Name] = tool
	return nil
}

func (f *fakeMCPTransportV0) ReadResourceV0(ctx context.Context, name string) (json.RawMessage, error) {
	return f.resources[name].Handler(ctx)
}

func (f *fakeMCPTransportV0) CallToolV0(ctx context.Context, name string, input any) (json.RawMessage, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return f.tools[name].Handler(ctx, raw)
}

type fakeTransportStatusPortMCPV0 struct{ called int }

func (f *fakeTransportStatusPortMCPV0) QueryOperatorStatusV0(
	operator.OperatorStatusQueryV0,
) (operator.OperatorMCPStatusResultV0, error) {
	f.called++
	return operator.OperatorMCPStatusResultV0{
		Status:       "healthy",
		Summary:      "status compacto",
		Sections:     []string{"summary"},
		EvidenceRefs: []string{"evidence-ref-1"},
	}, nil
}

func assertTransportPayloadSaneadoMCPTestV0(t *testing.T, value any, maxBytes int) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(payload) > maxBytes {
		t.Fatalf("payload demasiado grande: got=%d max=%d payload=%s", len(payload), maxBytes, payload)
	}
	text := strings.ToLower(string(payload))
	for _, allowedFalseFlag := range []string{
		`"contains_secret":false`,
		`"contains_transcript":false`,
		`"contains_prompt":false`,
		`"contains_completion":false`,
		`"contains_connection_detail":false`,
	} {
		text = strings.ReplaceAll(text, allowedFalseFlag, "")
	}
	for _, forbidden := range []string{"password", "oauth", "provider", "model", "/home/", "home=", "event-store", "internal/", "transcript", "secret", "dsn", "sql"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("payload contiene %q: %s", forbidden, text)
		}
	}
}
