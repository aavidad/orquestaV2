package orquestamcp

import "testing"

type statusScopeTypedPayloadV0 struct {
	RunRef string                      `json:"run_ref"`
	Nested []statusScopeTypedPayloadV0 `json:"nested"`
}

type statusScopeTypedContainersV0 struct {
	Array      [1]statusScopeTypedPayloadV0         `json:"array"`
	Slice      []statusScopeTypedPayloadV0          `json:"slice"`
	Map        map[string]statusScopeTypedPayloadV0 `json:"map"`
	NumericMap map[int]statusScopeTypedPayloadV0    `json:"numeric_map"`
}

type statusScopePrivatePayloadV0 struct {
	runRef string `json:"run_ref"`
}

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

func TestMCPAutoprogrammingStatusScopeV0SanitizesNestedSemanticRefsAndAppAuthority(t *testing.T) {
	allowed, foreign, app := "run-opaque-allowed", "run-opaque-foreign", "app-opaque-allowed"
	result := MCPAutoprogrammingStatusToolResultV0{
		Queue: &MCPRunQueuePriorityToolResultV0{QueueRef: "queue-opaque", Ranked: []MCPRunQueueRankedCandidateCompactV0{
			{RunRef: allowed, AppRef: app, ParentRunRef: foreign, SupersedesRunRef: allowed, EvidenceRefs: []string{"evidence", foreign, allowed}},
			{RunRef: foreign, AppRef: "app-opaque-foreign"},
		}},
		Tasks:        []MCPAutoprogrammingTaskV0{{RunRef: allowed, EvidenceRefs: []string{foreign, "task-evidence"}}},
		Agents:       []MCPAutoprogrammingAgentV0{{RunRef: allowed, EvidenceRefs: []string{foreign, "agent-evidence"}}},
		StaleRunning: []MCPAutoprogrammingActionableRunV0{{RunRef: allowed, EvidenceRefs: []string{foreign, "action-evidence"}}},
		Diagnostics:  []MCPAutoprogrammingDiagnosticV0{{Scope: "run:" + allowed, SampleRefs: []string{foreign, "diagnostic"}, EvidenceRefs: []string{foreign, "diagnostic-evidence"}}},
		Operator: &MCPAutoprogrammingOperatorV0{
			ClosureBlockers: []MCPAutoprogrammingClosureBlockerV0{{RunRef: allowed, Evidence: []string{foreign, "blocker-evidence"}}},
			SafeActions: []MCPAutoprogrammingSafeActionV0{
				{RunRef: allowed, Payload: map[string]any{"typed": statusScopeTypedPayloadV0{RunRef: allowed, Nested: []statusScopeTypedPayloadV0{{RunRef: foreign}}}}},
				{RunRef: allowed, Payload: map[string]any{"run_ref": foreign}},
			},
		},
	}
	scoped := scopeMCPAutoprogrammingStatusResultV0(result, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeAppV0, Scope: app})
	if scoped.Queue == nil || len(scoped.Queue.Ranked) != 1 || scoped.Queue.Ranked[0].ParentRunRef != "" || scoped.Queue.Ranked[0].SupersedesRunRef != allowed {
		t.Fatalf("candidate no saneado: %+v", scoped.Queue)
	}
	if len(scoped.Tasks) != 1 || len(scoped.Tasks[0].EvidenceRefs) != 1 || len(scoped.Agents[0].EvidenceRefs) != 1 || len(scoped.StaleRunning[0].EvidenceRefs) != 1 || len(scoped.Diagnostics[0].EvidenceRefs) != 1 || len(scoped.Operator.ClosureBlockers[0].Evidence) != 1 {
		t.Fatalf("evidencias ajenas filtradas de forma incompleta: %+v", scoped)
	}
	if len(scoped.Operator.SafeActions) != 0 {
		t.Fatalf("payload recursivo extranjero debe descartar accion: %+v", scoped.Operator.SafeActions)
	}
}

func TestMCPAutoprogrammingStatusScopeV0RejectsIncompleteAndInvalidSelector(t *testing.T) {
	for _, input := range []MCPAutoprogrammingStatusToolInputV0{
		{ScopeMode: MCPAutoprogrammingStatusScopeRunV0}, {ScopeMode: "unknown", Scope: "opaque"}, {Scope: "opaque"}, {ScopeMode: MCPAutoprogrammingStatusScopeLegacyV0, Scope: "opaque"},
	} {
		result, err := (MCPAutoprogrammingStatusToolExecutorV0{}).Execute(nil, input)
		if err != nil || result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 || len(result.Errores) != 1 || result.Errores[0].Code != "autoprogramming_status_scope_invalid" {
			t.Fatalf("input=%+v result=%+v err=%v", input, result, err)
		}
	}
}

func TestMCPAutoprogrammingStatusScopeV0FiltersKnownBlockerRefOnly(t *testing.T) {
	allowed, foreign, opaque := "run-opaque-allowed", "run-opaque-foreign", "blocker-opaque-external"
	result := MCPAutoprogrammingStatusToolResultV0{
		Queue: &MCPRunQueuePriorityToolResultV0{QueueRef: "queue-opaque", Ranked: []MCPRunQueueRankedCandidateCompactV0{{RunRef: allowed}, {RunRef: foreign}}},
		Operator: &MCPAutoprogrammingOperatorV0{ClosureBlockers: []MCPAutoprogrammingClosureBlockerV0{
			{RunRef: allowed, BlockerRef: foreign},
			{RunRef: allowed, BlockerRef: opaque},
		}},
	}
	scoped := scopeMCPAutoprogrammingStatusResultV0(result, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeRunV0, Scope: allowed})
	if got := scoped.Operator.ClosureBlockers; len(got) != 2 || got[0].BlockerRef != "" || got[1].BlockerRef != opaque {
		t.Fatalf("blocker refs deben borrar solo run conocido extranjero: %+v", got)
	}
}

func TestMCPAutoprogrammingStatusScopeV0RecursesTypedContainersAndFailsClosedPrivateFields(t *testing.T) {
	allowed, foreign := "run-opaque-allowed", "run-opaque-foreign"
	allowedContainers := statusScopeTypedContainersV0{
		Array:      [1]statusScopeTypedPayloadV0{{RunRef: allowed}},
		Slice:      []statusScopeTypedPayloadV0{{RunRef: allowed}},
		Map:        map[string]statusScopeTypedPayloadV0{"item": {RunRef: allowed}},
		NumericMap: map[int]statusScopeTypedPayloadV0{1: {RunRef: allowed}},
	}
	foreignArray, foreignSlice, foreignMap := allowedContainers, allowedContainers, allowedContainers
	foreignArray.Array = [1]statusScopeTypedPayloadV0{{RunRef: foreign}}
	foreignSlice.Slice = []statusScopeTypedPayloadV0{{RunRef: foreign}}
	foreignMap.NumericMap = map[int]statusScopeTypedPayloadV0{1: {RunRef: foreign}}
	result := MCPAutoprogrammingStatusToolResultV0{
		Queue: &MCPRunQueuePriorityToolResultV0{QueueRef: "queue-opaque", Ranked: []MCPRunQueueRankedCandidateCompactV0{{RunRef: allowed}, {RunRef: foreign}}},
		Operator: &MCPAutoprogrammingOperatorV0{SafeActions: []MCPAutoprogrammingSafeActionV0{
			{RunRef: allowed, Payload: map[string]any{"typed": allowedContainers}},
			{RunRef: allowed, Payload: map[string]any{"typed": foreignArray}},
			{RunRef: allowed, Payload: map[string]any{"typed": foreignSlice}},
			{RunRef: allowed, Payload: map[string]any{"typed": foreignMap}},
			{RunRef: allowed, Payload: map[string]any{"private": statusScopePrivatePayloadV0{runRef: foreign}}},
		}},
	}
	scoped := scopeMCPAutoprogrammingStatusResultV0(result, MCPAutoprogrammingStatusToolInputV0{ScopeMode: MCPAutoprogrammingStatusScopeRunV0, Scope: allowed})
	if scoped.Operator == nil || len(scoped.Operator.SafeActions) != 1 {
		t.Fatalf("contenedores extranjeros o campos privados deben descartar sin panic: %+v", scoped.Operator)
	}
	if _, ok := scoped.Operator.SafeActions[0].Payload["typed"].(statusScopeTypedContainersV0); !ok {
		t.Fatalf("payload tipado permitido debe conservar su tipo: %+v", scoped.Operator.SafeActions[0].Payload)
	}
}
