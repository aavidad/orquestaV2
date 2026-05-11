package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPTransportV0DirectorStatsInvocaPuerto(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-transport-001")
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		DirectorStats: MCPDirectorStatsToolExecutorV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPDirectorStatsToolNameV0,
		MCPDirectorStatsToolInputV0{RunRef: run.RunID},
	)
	if err != nil {
		t.Fatalf("call director stats: %v", err)
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Counts.TasksOpen != 2 ||
		!result.Stats.Closure.Blocked {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPTransportV0DirectorStatsIncluyeProgressObservadoEnJSON(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-progress-001")
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		DirectorStats: MCPDirectorStatsToolExecutorV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			ProgressSource: mcpDirectorStatsProgressSourceForTestV0{
				Observations: []orquestacionnucleoapp.AgentProgressObservationV0{
					mcpDirectorStatsProgressObservationForTestV0(run.RunID),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPDirectorStatsToolNameV0,
		MCPDirectorStatsToolInputV0{
			CorrelationID:        "corr-mcp-director-stats-progress-001",
			RunRef:               run.RunID,
			OccurredAt:           "2026-05-10T09:00:00Z",
			IncludeAgentProgress: true,
		},
	)
	if err != nil {
		t.Fatalf("call director stats progress: %v", err)
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Stats == nil || result.DecisionContext == nil {
		t.Fatalf("result incompleto=%+v", result)
	}
	if result.Stats.Progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceLoadedV0 ||
		result.Stats.Progress.TasksObserved != 1 ||
		result.Stats.Progress.ProgressingAgents != 1 {
		t.Fatalf("progress=%+v result=%+v", result.Stats.Progress, result)
	}
	assertDirectorStatsJSONHasProgressClosureContextV0(t, output)
}

func assertDirectorStatsJSONHasProgressClosureContextV0(t *testing.T, output json.RawMessage) {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	var stats map[string]json.RawMessage
	if err := json.Unmarshal(envelope["stats"], &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if _, ok := stats["progress"]; !ok {
		t.Fatalf("json sin stats.progress: %s", string(output))
	}
	if _, ok := stats["closure"]; !ok {
		t.Fatalf("json sin stats.closure: %s", string(output))
	}
	if _, ok := envelope["decision_context"]; !ok {
		t.Fatalf("json sin decision_context: %s", string(output))
	}
}
