package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
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
