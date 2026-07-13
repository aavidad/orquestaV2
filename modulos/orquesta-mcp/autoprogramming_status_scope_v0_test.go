package orquestamcp

import "testing"

func TestMCPAutoprogrammingStatusScopeV0UsesExactOpaqueRunMembership(t *testing.T) {
	allowed := "request-ref-opaque-allowed-001"
	foreign := "request-ref-opaque-allowed-001-shadow"
	result := MCPAutoprogrammingStatusToolResultV0{
		RunRef:           foreign,
		CausalVerdict:    "foreign_causal",
		CausalReasonCode: "foreign_reason",
		QueueRef:         "queue-ref-opaque-001",
		Queue:            &MCPRunQueuePriorityToolResultV0{QueueRef: "queue-ref-opaque-001", Ranked: []MCPRunQueueRankedCandidateCompactV0{{RunRef: allowed}, {RunRef: foreign}}},
		Projects:         []MCPAutoprogrammingProjectV0{{RunRefs: []string{allowed, foreign}}},
		Tasks:            []MCPAutoprogrammingTaskV0{{RunRef: allowed}, {RunRef: foreign}},
		Agents:           []MCPAutoprogrammingAgentV0{{RunRef: allowed}, {RunRef: foreign}},
		StaleRunning:     []MCPAutoprogrammingActionableRunV0{{RunRef: allowed}, {RunRef: foreign}},
		Diagnostics: []MCPAutoprogrammingDiagnosticV0{
			{Code: "allowed", Scope: "run:" + allowed + " field:status"},
			{Code: "foreign", Scope: "run:" + foreign + " field:status"},
		},
		Operator: &MCPAutoprogrammingOperatorV0{SafeActions: []MCPAutoprogrammingSafeActionV0{
			{Action: "observe_goal", Scope: "run:" + allowed, RunRef: allowed, Payload: map[string]any{"nested": map[string]any{"run_refs": []any{allowed}}}},
			{Action: "observe_goal", Scope: "run:" + allowed, RunRef: allowed, Payload: map[string]any{"nested": []any{map[string]any{"run_ref": foreign}}}},
		}},
	}

	scoped := scopeMCPAutoprogrammingStatusResultV0(result, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeQueueV0, Scope: "queue-ref-opaque-001"})
	if scoped.ScopeMode != MCPAutoprogrammingStatusScopeQueueV0 || scoped.Scope != "queue-ref-opaque-001" || scoped.Run != nil || scoped.RunRef != "" || scoped.CausalVerdict != "" || scoped.CausalReasonCode != "" {
		t.Fatalf("scope/causal=%+v", scoped)
	}
	if scoped.Queue == nil || len(scoped.Queue.Ranked) != 2 || len(scoped.Projects) != 1 || len(scoped.Projects[0].RunRefs) != 2 || len(scoped.Tasks) != 2 || len(scoped.Agents) != 2 || len(scoped.StaleRunning) != 2 {
		t.Fatalf("queue scope debe usar exactamente la identidad de todos los candidatos: %+v", scoped)
	}

	scoped = scopeMCPAutoprogrammingStatusResultV0(result, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeRunV0, Scope: allowed})
	if scoped.Queue == nil || len(scoped.Queue.Ranked) != 1 || scoped.Queue.Ranked[0].RunRef != allowed || len(scoped.Projects) != 1 || len(scoped.Projects[0].RunRefs) != 1 || scoped.Projects[0].RunRefs[0] != allowed || len(scoped.Tasks) != 1 || len(scoped.Agents) != 1 || len(scoped.StaleRunning) != 1 || len(scoped.Diagnostics) != 1 || scoped.Diagnostics[0].Code != "allowed" || scoped.Operator == nil || len(scoped.Operator.SafeActions) != 1 {
		t.Fatalf("scope run no filtro todo por igualdad exacta: %+v", scoped)
	}
	if scoped.Operator.SafeActions[0].Payload["nested"] == nil {
		t.Fatalf("payload permitido perdido: %+v", scoped.Operator.SafeActions)
	}
}

func TestMCPAutoprogrammingStatusScopeV0FailClosedClearsDiscardedQueue(t *testing.T) {
	result := scopeMCPAutoprogrammingStatusResultV0(MCPAutoprogrammingStatusToolResultV0{
		QueueRef: "queue-ref-visible-001",
		Queue:    &MCPRunQueuePriorityToolResultV0{QueueRef: "queue-ref-visible-001", Ranked: []MCPRunQueueRankedCandidateCompactV0{{RunRef: "request-ref-visible-001"}}},
	}, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeQueueV0, Scope: "queue-ref-other-001"})
	if result.Queue != nil || result.QueueRef != "" {
		t.Fatalf("cola fuera de scope debe desaparecer: %+v", result)
	}
}
