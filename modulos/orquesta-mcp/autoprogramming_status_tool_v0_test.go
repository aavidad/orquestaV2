package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAutoprogrammingStatusDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingStatusToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingStatusResourceURIV0 ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor incompleto: %+v", descriptor)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DelegaEnColaYRun(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: queue,
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RequestID:     "request-ref-autop-status-001",
		CorrelationID: "corr-autop-status-001",
		RunRef:        "run-ref-autop-status-001",
		QueueRef:      "queue-ref-autop-status-001",
		QueueLimit:    3,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.Queue == nil ||
		result.Run == nil ||
		result.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("result=%+v", result)
	}
	if queue.input.Action != MCPRunQueuePriorityActionRankV0 ||
		queue.input.Limit != 3 ||
		stats.input.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("queue=%+v stats=%+v", queue.input, stats.input)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaPuertosNoConfigurados(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{}).Execute(
		context.Background(),
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-missing-001"},
	)

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Diagnostics) < 2 ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-transport-001"},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 || result.Run == nil || result.Queue == nil {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPAutoprogrammingQueueStatusV0 struct {
	input MCPRunQueuePriorityToolInputV0
}

func (fake *fakeMCPAutoprogrammingQueueStatusV0) Execute(
	_ context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	fake.input = input
	return MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoOKV0,
		CorrelationID: input.CorrelationID,
		Action:        MCPRunQueuePriorityActionRankV0,
		QueueRef:      input.QueueRef,
		Count:         1,
		Ranked: []MCPRunQueueRankedCandidateCompactV0{{
			Rank:          1,
			RunRef:        "run-ref-autop-status-001",
			AppRef:        "app-ref-autop-status-001",
			PriorityScore: 50,
		}},
		Errores: []MCPValidationIssueV0{},
	}, nil
}

type fakeMCPAutoprogrammingRunStatusV0 struct {
	input MCPDirectorStatsToolInputV0
}

func (fake *fakeMCPAutoprogrammingRunStatusV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	fake.input = input
	return MCPDirectorStatsToolResultV0{
		Estado:        MCPDirectorStatsEstadoOKV0,
		CorrelationID: input.CorrelationID,
		RunRef:        input.RunRef,
		Errores:       []MCPValidationIssueV0{},
	}, nil
}
