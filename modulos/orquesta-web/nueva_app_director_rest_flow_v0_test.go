package orquestaweb

import (
	"context"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestRESTArrancarDirectorAppClientV0FlujoVerticalMCP(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	executor := orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				webDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				webDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	server := httptestNewDirectorServerV0(t, executor)
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	vm, err := client.ArrancarDirectorApp(context.Background(), WebNuevaAppFormV0{
		RequestID:             "req-web-director-flow-001",
		Locale:                "es-ES",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		Nombre:                "Agenda",
		Objetivo:              "Gestionar citas con API y web",
		TipoApp:               "mixed",
		Agentes: WebNuevaAppAgentesFormV0{
			Autonomia: "media",
		},
	})
	if err != nil {
		t.Fatalf("ArrancarDirectorApp: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoDirector ||
		vm.Director == nil ||
		vm.Director.RunRef == "" ||
		len(vm.Director.StartedAgents) != 1 {
		t.Fatalf("vm=%+v", vm)
	}
	if vm.Director.DirectorTasks[0].Capacity == "" {
		t.Fatalf("director task sin capacidad: %+v", vm.Director.DirectorTasks)
	}
}

func httptestNewDirectorServerV0(
	t *testing.T,
	executor orquestamcp.MCPArrancarDirectorAppToolExecutorV0,
) *webHTTPClientTestServerV0 {
	t.Helper()
	return newWebHTTPTestServerV0(t, orquestamcp.NewMCPArrancarDirectorAppHTTPHandlerV0(executor))
}

func webDirectorCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-10T03:00:00Z",
			CorrelationID:   "corr-web-director-flow-capacity",
			RequestedBy:     "orquesta-web-test",
		},
		Acker: ledger,
	}
}

func webDirectorAgentLauncherDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-10T03:01:00Z",
			CorrelationID: "corr-web-director-flow-agent",
			RequestedBy:   "orquesta-web-test",
		},
		Acker: ledger,
	}
}
