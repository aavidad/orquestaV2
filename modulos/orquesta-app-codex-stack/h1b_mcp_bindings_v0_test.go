package orquestaappcodexstack

import (
	"context"
	"reflect"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestBuildStackV0CableaSolicitarNuevaAppMCPInProcessV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	result, err := stack.MCPTransportBindings.NuevaApp.Execute(context.Background(), orquestamcp.MCPNuevaAppToolInputV0{
		RequestID:      "request-ref-h1b-nueva-app-001",
		CorrelationID:  "correlation-ref-h1b-nueva-app-001",
		AppSpecRequest: codexStackAppSpecRequestV0(),
	})
	if err != nil || result.Estado != orquestamcp.MCPNuevaAppEstadoOKV0 ||
		result.AppSpec.SpecID == "" || result.Backlog.Microtareas == 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestBuildStackV0CableaCompatibilidadLegacySinActivarlaEnGoalFirstV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	result, err := stack.MCPTransportBindings.EjecutarOrquestacion.Execute(
		context.Background(),
		orquestamcp.MCPEjecutarOrquestacionAppToolInputV0{
			RequestID:             "request-ref-h1b-ejecutar-001",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		},
	)
	if err != nil || result.Estado != orquestamcp.MCPEjecutarOrquestacionAppEstadoErrorV0 ||
		result.RoutePolicy.PreferredEntrypoint != orquestamcp.MCPArrancarDirectorAppToolNameV0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestBuildStackV0CableaDecisionYCapabilitiesSinTransportUnboundV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.MCPTransportBindings.DirectorDecision == nil || stack.MCPTransportBindings.ToolCapabilities == nil {
		t.Fatalf("bindings=%+v", stack.MCPTransportBindings)
	}
	decision, err := stack.MCPTransportBindings.DirectorDecision.Execute(
		context.Background(), orquestamcp.MCPDirectorAgentDecisionToolInputV0{},
	)
	if err != nil || decision.Estado != orquestamcp.MCPDirectorAgentDecisionEstadoErrorV0 {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	capabilities, err := stack.MCPTransportBindings.ToolCapabilities.Execute(
		context.Background(), orquestamcp.MCPToolCapabilitiesListToolInputV0{},
	)
	if err != nil || capabilities.Estado != orquestamcp.MCPToolCapabilitiesListEstadoErrorV0 ||
		len(capabilities.Errores) != 1 || capabilities.Errores[0].Code != orquestamcp.MCPToolCapabilitiesListErrCatalogUnavailableV0 {
		t.Fatalf("capabilities=%+v err=%v", capabilities, err)
	}
}

func TestBuildStackV0ApplyDecisionUsaPuertosGobernadosDelStackV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	outer, ok := stack.MCPTransportBindings.DirectorDecision.(codexStackDirectorDecisionExecutorV0)
	if !ok {
		t.Fatalf("executor=%T", stack.MCPTransportBindings.DirectorDecision)
	}
	inner, ok := outer.Inner.(orquestamcp.MCPDirectorAgentDecisionToolExecutorV0)
	if !ok {
		t.Fatalf("inner=%T", outer.Inner)
	}
	if reflect.TypeOf(inner.Ports.EventSink) != reflect.TypeOf(stack.Ports.EventSink) {
		t.Fatalf("apply_decision salta EventSink gobernado: inner=%T stack=%T", inner.Ports.EventSink, stack.Ports.EventSink)
	}
	if inner.Ports.RunStore == nil || inner.Ports.TaskStore == nil {
		t.Fatalf("ports incompletos: %+v", inner.Ports)
	}
}
