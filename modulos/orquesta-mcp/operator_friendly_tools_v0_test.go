package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPOperatorFriendlyStatusTransportV0AceptaIncludesComoListas(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyStatusToolNameV0, map[string]any{
		"limit":             1,
		"include_agents":    []string{"summary"},
		"include_run_stats": []string{"progress"},
		"include_usage":     "true",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != "ok" || result.Counts.Runs == 0 || result.Counts.Agents == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPOperatorFriendlyAgentsTransportV0FiltraPorProyecto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: fakeMCPFriendlyQueueV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyAgentsToolNameV0, map[string]any{
		"project_ref": "app-ref-beta",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != "ok" || result.Counts.Agents != 1 || len(result.Agents) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Agents[0].ProjectRef != "app-ref-beta" || result.Agents[0].RunRef != "run-ref-beta" {
		t.Fatalf("agent=%+v", result.Agents[0])
	}
}

func TestMCPOperatorFriendlyTasksTransportV0ProyectaReadyConAgenteVivoComoRunning(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: fakeMCPFriendlyReadyQueueV0{},
		DirectorStats:    fakeMCPFriendlyLiveStatsV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	allOutput, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyTasksToolNameV0, map[string]any{})
	if err != nil {
		t.Fatalf("call all tasks: %v", err)
	}
	var allResult MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(allOutput, &allResult); err != nil {
		t.Fatalf("decode all tasks: %v", err)
	}
	if len(allResult.Tasks) != 2 ||
		allResult.Tasks[0].Status != "running" ||
		allResult.Tasks[1].Status != "ready" ||
		allResult.Counts.ByStatus["running"] != 1 ||
		allResult.Counts.ByStatus["ready"] != 1 {
		t.Fatalf("all result=%+v", allResult)
	}

	output, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyTasksToolNameV0, map[string]any{
		"status": "running",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != "ok" || len(result.Tasks) != 1 || result.Tasks[0].RunRef != "run-ref-live" {
		t.Fatalf("result=%+v", result)
	}
	if result.Tasks[0].Status != "running" || len(result.Runs) != 0 || len(result.Agents) != 0 {
		t.Fatalf("task/status projection=%+v runs=%+v agents=%+v", result.Tasks, result.Runs, result.Agents)
	}
	if result.Counts.ByStatus["running"] != 1 || result.Counts.ByStatus["ready"] != 0 {
		t.Fatalf("counts=%+v", result.Counts.ByStatus)
	}
}

func TestMCPOperatorFriendlyTasksTransportV0UsaRunRefDeStatsEnProyeccionLive(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: fakeMCPFriendlyReadyQueueV0{},
		DirectorStats:    fakeMCPFriendlyStatsRunRefOnlyV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPOperatorFriendlyTasksToolNameV0, map[string]any{
		"status": "running",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPOperatorFriendlyStatusResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Tasks) != 1 || result.Tasks[0].RunRef != "run-ref-live" || result.Tasks[0].Status != "running" {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPFriendlyQueueV0 struct{}

func (fakeMCPFriendlyQueueV0) Execute(
	_ context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	return MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoOKV0,
		CorrelationID: input.CorrelationID,
		Action:        MCPRunQueuePriorityActionRankV0,
		QueueRef:      input.QueueRef,
		Count:         2,
		Ranked: []MCPRunQueueRankedCandidateCompactV0{
			{Rank: 1, RunRef: "run-ref-alpha", AppRef: "app-ref-alpha", Status: "running", PriorityScore: 50},
			{Rank: 2, RunRef: "run-ref-beta", AppRef: "app-ref-beta", Status: "running", PriorityScore: 40},
		},
		Errores: []MCPValidationIssueV0{},
	}, nil
}

type fakeMCPFriendlyReadyQueueV0 struct{}

func (fakeMCPFriendlyReadyQueueV0) Execute(
	_ context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	return MCPRunQueuePriorityToolResultV0{
		Estado:   MCPRunQueuePriorityEstadoOKV0,
		Action:   MCPRunQueuePriorityActionRankV0,
		QueueRef: input.QueueRef,
		Count:    2,
		Ranked: []MCPRunQueueRankedCandidateCompactV0{
			{Rank: 1, RunRef: "run-ref-live", AppRef: "app-ref-alpha", Status: "ready", PriorityScore: 50},
			{Rank: 2, RunRef: "run-ref-pending", AppRef: "app-ref-beta", Status: "ready", PriorityScore: 40},
		},
		Errores: []MCPValidationIssueV0{},
	}, nil
}

type fakeMCPFriendlyLiveStatsV0 struct{}

func (fakeMCPFriendlyLiveStatsV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	stats := &orquestacionnucleoapp.DirectorRunStatsV0{
		RunRef:     input.RunRef,
		ProjectRef: "app-ref-beta",
	}
	if input.RunRef == "run-ref-live" {
		stats.ProjectRef = "app-ref-alpha"
		stats.Counts.AgentsInFlight = 1
		stats.Agents = []orquestacionnucleoapp.DirectorAgentStatsV0{{
			AgentRequestID: "agent-ref-live",
			Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
			InFlight:       true,
		}}
	}
	return MCPDirectorStatsToolResultV0{
		Estado:  MCPDirectorStatsEstadoOKV0,
		RunRef:  input.RunRef,
		Stats:   stats,
		Errores: []MCPValidationIssueV0{},
	}, nil
}

type fakeMCPFriendlyStatsRunRefOnlyV0 struct{}

func (fakeMCPFriendlyStatsRunRefOnlyV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	stats := &orquestacionnucleoapp.DirectorRunStatsV0{RunRef: input.RunRef}
	if input.RunRef == "run-ref-live" {
		stats.Counts.AgentsInFlight = 1
	}
	return MCPDirectorStatsToolResultV0{
		Estado:  MCPDirectorStatsEstadoOKV0,
		Stats:   stats,
		Errores: []MCPValidationIssueV0{},
	}, nil
}
