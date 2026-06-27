package orquestamcp

import (
	"context"
	"encoding/json"
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
		MCPWorkspaceTimelineResourceNameV0,
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
		MCPObserveAppDirectorGoalToolNameV0,
		MCPDirectorAgentDecisionToolNameV0,
		MCPDirectorSupervisorBriefingToolNameV0,
		MCPDirectorStatsToolNameV0,
		MCPPrepararOrquestacionAppToolNameV0,
		MCPEjecutarOrquestacionAppToolNameV0,
		MCPAutoprogrammingValidateRequestToolNameV0,
		MCPHumanDirectorWorkReviewPlanToolNameV0,
		MCPAutoprogrammingSelfImprovementToolNameV0,
		MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingObserveGoalToolNameV0,
		MCPAutoprogrammingObserveActiveGoalsToolNameV0,
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingSuperviseToolNameV0,
		MCPBootstrapToolNameV0,
		MCPCoreWorkflowCommandToolNameV0,
		MCPRunControlToolNameV0,
		MCPRuntimeModelsToolNameV0,
		MCPRunQueuePriorityToolNameV0,
		MCPRunSupervisorToolNameV0,
		MCPWorkspaceTimelineToolNameV0,
		MCPServerShutdownToolNameV0,
		MCPDomainWorkToolNameV0,
		MCPExternalWorkRunToolNameV0,
		MCPAppVCSToolNameV0,
		operator.OperatorMCPStatusToolNameV0,
		operator.OperatorMCPBurstToolNameV0,
		operator.OperatorMCPOutboxToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
		MCPOperatorFriendlyStatusToolNameV0,
		MCPOperatorFriendlyTasksToolNameV0,
		MCPOperatorFriendlyProjectsToolNameV0,
		MCPOperatorFriendlyAgentsToolNameV0,
		MCPOperatorFriendlyCommandToolNameV0,
	} {
		if _, ok := transport.tools[name]; !ok {
			t.Fatalf("tool no registrado: %s", name)
		}
	}
	assertTransportPayloadSaneadoMCPTestV0(t, transport.resources, 9000)
	assertTransportPayloadSaneadoMCPTestV0(t, transport.tools, 28000)
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

func TestMCPTransportV0ExponeHerramientasOperadorSinRefsInternas(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: queue,
		DirectorStats:    stats,
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyStatusToolNameV0, MCPOperatorFriendlyQueryV0{
		IncludeAgents: true,
	})
	if err != nil {
		t.Fatalf("call friendly status: %v", err)
	}
	var result MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != "ok" ||
		result.Counts.Tasks != 1 ||
		result.Counts.Projects != 1 ||
		result.Counts.Agents != 1 ||
		len(result.Tasks) != 1 ||
		len(result.Projects) != 1 {
		t.Fatalf("friendly status=%+v", result)
	}
	if queue.input.Action != MCPRunQueuePriorityActionRankV0 {
		t.Fatalf("queue input=%+v", queue.input)
	}
	if stats.input.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("stats input=%+v", stats.input)
	}
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

func TestMCPTransportV0ObserveAppDirectorGoalQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPObserveAppDirectorGoalToolNameV0, MCPObserveAppDirectorGoalToolInputV0{
		RunRef: "run-ref-goal-unbound-001",
	})
	if err != nil {
		t.Fatalf("call observe goal unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPObserveAppDirectorGoalToolNameV0 || result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("observe director goal debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0AutoprogrammingObserveGoalQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingObserveGoalToolNameV0, MCPAutoprogrammingObserveGoalToolInputV0{
		RunRef: "run-ref-autoprogramming-goal-unbound-001",
	})
	if err != nil {
		t.Fatalf("call autoprogramming observe goal unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPAutoprogrammingObserveGoalToolNameV0 || result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("autoprogramming observe goal debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0AutoprogrammingObserveActiveGoalsQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingObserveActiveGoalsToolNameV0, MCPAutoprogrammingObserveActiveGoalsToolInputV0{})
	if err != nil {
		t.Fatalf("call autoprogramming observe active goals unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPAutoprogrammingObserveActiveGoalsToolNameV0 || result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("autoprogramming observe active goals debe ser opt-in: %+v", result)
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

func TestMCPTransportV0AutoprogrammingPrepareRunQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingPrepareRunToolNameV0, MCPAutoprogrammingPrepareRunToolInputV0{})
	if err != nil {
		t.Fatalf("call autoprogramming prepare run unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPAutoprogrammingPrepareRunToolNameV0 || result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("autoprogramming prepare run debe ser opt-in: %+v", result)
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
