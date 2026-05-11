package orquestadirector

import (
	"encoding/json"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildConcurrencyGateAgentRequestsV0AllowConstruyeGateYRequestAgent(t *testing.T) {
	input := validConcurrencyGateAgentRequestsInputV0()

	result, err := BuildConcurrencyGateAgentRequestsV0(input)
	if err != nil {
		t.Fatalf("BuildConcurrencyGateAgentRequestsV0: %v %+v", err, result.Issues)
	}
	if result.Evaluation.Decision != orquestacoreconcurrency.ConcurrencyGateDecisionAllowRequestAgentV0 {
		t.Fatalf("decision=%q, want allow", result.Evaluation.Decision)
	}
	if result.GateCommand.CommandType != orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0 {
		t.Fatalf("gate command=%+v", result.GateCommand)
	}
	if len(result.AgentCommands) != 2 {
		t.Fatalf("agent commands=%d, want 2", len(result.AgentCommands))
	}
	for _, command := range result.AgentCommands {
		if command.CommandType != orquestacoreworkflow.OrchestrationCommandRequestAgentV0 {
			t.Fatalf("agent command type=%q", command.CommandType)
		}
	}

	var gatePayload orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0
	mustDecodeConcurrencyGatePayloadV0(t, result.GateCommand.Payload, &gatePayload)
	if gatePayload.Decision != orquestacoreworkflow.ConcurrencyGateDecisionAllowRequestAgentV0 ||
		gatePayload.GateRef == "" ||
		len(gatePayload.SubjectClaimRefs) != 2 {
		t.Fatalf("gate payload inesperado: %+v", gatePayload)
	}
}

func TestBuildConcurrencyGateAgentRequestsV0BlockNoConstruyeRequestAgent(t *testing.T) {
	input := validConcurrencyGateAgentRequestsInputV0()
	input.Claims[1].WriteSet = []orquestacoreconcurrency.ScopeRefV0{scopeRefForGateTestV0(t, "modulos/orquesta-director/docs")}

	result, err := BuildConcurrencyGateAgentRequestsV0(input)
	if err != nil {
		t.Fatalf("BuildConcurrencyGateAgentRequestsV0: %v %+v", err, result.Issues)
	}
	if result.Evaluation.Decision != orquestacoreconcurrency.ConcurrencyGateDecisionBlockRequestAgentV0 {
		t.Fatalf("decision=%q, want block", result.Evaluation.Decision)
	}
	if len(result.AgentCommands) != 0 {
		t.Fatalf("agent commands=%d, want 0", len(result.AgentCommands))
	}
	if len(result.BlockedClaimRefs) == 0 || len(result.ConflictRefs) == 0 {
		t.Fatalf("blocked=%v conflicts=%v", result.BlockedClaimRefs, result.ConflictRefs)
	}
}

func TestBuildConcurrencyGateAgentRequestsV0AllowRequiereCandidatoPorSubject(t *testing.T) {
	input := validConcurrencyGateAgentRequestsInputV0()
	input.CandidateRequests = input.CandidateRequests[:1]

	result, err := BuildConcurrencyGateAgentRequestsV0(input)
	if err == nil {
		t.Fatalf("expected error, result=%+v", result)
	}
	requireConcurrencyGateIssueV0(t, result.Issues, ErrDirectorConcurrencyGateAgentRequestsInvalidoV0)
	if len(result.AgentCommands) != 0 {
		t.Fatalf("agent commands=%d, want 0", len(result.AgentCommands))
	}
}

func validConcurrencyGateAgentRequestsInputV0() ConcurrencyGateAgentRequestsInputV0 {
	return ConcurrencyGateAgentRequestsInputV0{
		GateCommandMeta: concurrencyGateMetaV0("cmd-gate-001", "idem-gate-001"),
		Claims: []orquestacoreconcurrency.WorksetClaimV0{
			worksetClaimForGateTestV0("claim:worker-a", "agent-request-gate-a", "modulos/orquesta-director/docs/contratos.md"),
			worksetClaimForGateTestV0("claim:worker-b", "agent-request-gate-b", "modulos/orquesta-director/docs/pruebas.md"),
		},
		SubjectClaimRefs: []string{"claim:worker-b", "claim:worker-a"},
		EvidenceRefs:     []string{"evidence-gate-001"},
		CandidateRequests: []CandidateAgentRequestV0{
			candidateAgentForGateTestV0("claim:worker-a", "agent-request-gate-a", "builder"),
			candidateAgentForGateTestV0("claim:worker-b", "agent-request-gate-b", "reviewer"),
		},
	}
}

func worksetClaimForGateTestV0(
	claimRef string,
	agentRequestID string,
	writeRef string,
) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         "run-gate-agent-requests-001",
		TaskRef:        "task-gate-agent-requests-001",
		AgentRequestID: agentRequestID,
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{scopeRefForGateTestMustV0(writeRef)},
	}
}

func candidateAgentForGateTestV0(
	claimRef string,
	agentRequestID string,
	role string,
) CandidateAgentRequestV0 {
	return CandidateAgentRequestV0{
		ClaimRef:    claimRef,
		CommandMeta: concurrencyGateMetaV0("cmd-"+agentRequestID, "idem-"+agentRequestID),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     agentRequestID,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-gate-agent-requests-001",
			CapacityRequestRef: "capacity-request-gate-001",
			Role:               role,
			Summary:            "Solicitar agente candidato tras gate de concurrencia.",
			EvidenceRefs:       []string{"evidence-" + agentRequestID},
		},
	}
}

func concurrencyGateMetaV0(commandID string, idempotencyKey string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          "run-gate-agent-requests-001",
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-gate-agent-requests-001",
		RequestedBy:    "director-test",
		OccurredAt:     "2026-05-06T10:00:00Z",
	}
}

func scopeRefForGateTestV0(t *testing.T, raw string) orquestacoreconcurrency.ScopeRefV0 {
	t.Helper()
	ref, issues := orquestacoreconcurrency.NewScopeRefV0(raw)
	if len(issues) > 0 {
		t.Fatalf("scope ref %q invalido: %+v", raw, issues)
	}
	return ref
}

func scopeRefForGateTestMustV0(raw string) orquestacoreconcurrency.ScopeRefV0 {
	ref, issues := orquestacoreconcurrency.NewScopeRefV0(raw)
	if len(issues) > 0 {
		panic(issues)
	}
	return ref
}

func mustDecodeConcurrencyGatePayloadV0(t *testing.T, raw json.RawMessage, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
}

func requireConcurrencyGateIssueV0(t *testing.T, issues []ConcurrencyGateAgentRequestsIssueV0, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrada en %+v", code, issues)
}
