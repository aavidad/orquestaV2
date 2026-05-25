package orquestamcp

import (
	"context"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPDirectorStatsToolExecutorV0ReportaUsoOptInSinFuente(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-usage-missing-001")

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:         "request-ref-mcp-director-stats-usage-missing-001",
		CorrelationID:     "corr-mcp-director-stats-usage-missing-001",
		RunRef:            run.RunID,
		IncludeAgentUsage: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Stats == nil ||
		len(result.Stats.Progress.Issues) != 1 ||
		result.Stats.Progress.Issues[0].Code != "agent_usage_source_not_configured" {
		t.Fatalf("result=%+v", result)
	}
}
