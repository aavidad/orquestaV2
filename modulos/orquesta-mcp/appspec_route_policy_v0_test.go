package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPPrepararOrquestacionAppToolV0DeclaraPreviewCompatibilidad(t *testing.T) {
	result := ExecuteMCPPrepararOrquestacionAppToolV0(MCPPrepararOrquestacionAppToolInputV0{
		AppSpec: validMCPPrepareLargeAppSpecForTestV0(t),
	})
	if result.RoutePolicy.Mode != orquestaapprunner.AppRunnerRouteModePreviewCompatV0 ||
		result.RoutePolicy.PreferredEntrypoint != MCPArrancarDirectorAppToolNameV0 ||
		result.RoutePolicy.LegacyEntrypoint != MCPPrepararOrquestacionAppToolNameV0 {
		t.Fatalf("route_policy=%+v", result.RoutePolicy)
	}
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0BloqueaDirectorV2Requerido(t *testing.T) {
	result, err := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{},
	).Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:             "request-ref-mcp-run-app-director-v2-required-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:            "2026-05-25T16:20:00Z",
		RequireDirectorV2:     true,
		AppSpec:               validMCPPrepareLargeAppSpecForTestV0(t),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != orquestaapprunner.AppRunnerDirectorV2RequiredFieldV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.RoutePolicy.PreferredEntrypoint != MCPArrancarDirectorAppToolNameV0 ||
		result.RoutePolicy.LegacyEntrypoint != MCPEjecutarOrquestacionAppToolNameV0 {
		t.Fatalf("route_policy=%+v", result.RoutePolicy)
	}
}

func TestMCPArrancarDirectorAppToolV0DeclaraRutaOperativaPreferente(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	result, err := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	).Execute(context.Background(), validMCPDirectorAppInputForTestV0())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.RoutePolicy.Mode != "operativo_preferente" ||
		result.RoutePolicy.PreferredEntrypoint != MCPArrancarDirectorAppToolNameV0 {
		t.Fatalf("route_policy=%+v", result.RoutePolicy)
	}
}

func TestMCPAppSpecDescriptorsV0PublicanRoutePolicy(t *testing.T) {
	cases := []struct {
		name             string
		output           string
		input            string
		requireV2        bool
		requireGoalRefs  bool
		requireEvidences bool
	}{
		{
			name:             MCPPrepararOrquestacionAppToolNameV0,
			output:           MCPPrepararOrquestacionAppDescriptorV0().Output,
			requireEvidences: true,
		},
		{
			name:             MCPEjecutarOrquestacionAppToolNameV0,
			output:           MCPEjecutarOrquestacionAppDescriptorV0().Output,
			input:            MCPEjecutarOrquestacionAppDescriptorV0().InputSchema,
			requireV2:        true,
			requireEvidences: true,
		},
		{
			name:             MCPArrancarDirectorAppToolNameV0,
			output:           MCPArrancarDirectorAppDescriptorV0().Output,
			requireGoalRefs:  true,
			requireEvidences: true,
		},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.output, "route_policy") {
			t.Fatalf("%s output=%q", tc.name, tc.output)
		}
		if tc.requireV2 && !strings.Contains(tc.input, "require_director_v2") {
			t.Fatalf("%s input_schema=%q", tc.name, tc.input)
		}
		if tc.requireGoalRefs && !strings.Contains(tc.output, "external_goal_ref?") {
			t.Fatalf("%s output no declara external_goal_ref: %q", tc.name, tc.output)
		}
		if tc.requireEvidences && !strings.Contains(tc.output, "evidence_refs?") {
			t.Fatalf("%s output no declara evidence_refs: %q", tc.name, tc.output)
		}
	}
}
