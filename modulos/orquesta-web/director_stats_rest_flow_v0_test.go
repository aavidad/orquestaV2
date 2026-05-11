package orquestaweb

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestRESTDirectorStatsClientV0ConsumeMCPDirectorStatsHTTPBridgeV0(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-web-mcp-stats-flow-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-web-mcp-stats-001", "task-web-mcp-stats-002"},
		ClosedTasks:   []string{"task-web-mcp-stats-001"},
		Agents:        []string{"agent-web-mcp-stats-001"},
		StartedAgents: []string{"agent-web-mcp-stats-001"},
		Deliveries:    []string{"delivery-web-mcp-stats-001"},
	}
	server := httptest.NewServer(orquestamcp.NewMCPDirectorStatsHTTPHandlerV0(
		orquestamcp.MCPDirectorStatsToolExecutorV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		},
	))
	defer server.Close()

	client := NewRESTDirectorStatsClientV0(server.URL, time.Second)
	panel, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RequestID:            "request-ref-web-mcp-stats-flow-001",
		CorrelationID:        "corr-web-mcp-stats-flow-001",
		Locale:               "es",
		RunRef:               run.RunID,
		IncludeProcessRefs:   true,
		IncludeAgentProgress: true,
	})
	if err != nil {
		t.Fatalf("ConsultarDirectorStats bridge: %v", err)
	}

	if panel.RunRef != run.RunID ||
		panel.Estado != WebDirectorStatsEstadoOKV0 ||
		panel.FaseActual != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) ||
		panel.Counts.TasksTotal != 2 ||
		panel.Counts.TasksOpen != 1 ||
		panel.Counts.AgentsStarted != 1 ||
		panel.Progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceNotConfiguredV0 {
		t.Fatalf("panel=%+v", panel)
	}
}
