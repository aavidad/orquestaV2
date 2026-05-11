package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildReplanFollowupsV0QualityGateBlockedRetryPideCapacidadAplicable(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0, "quality-gate")
	input.SourceKind = ReplanFollowupSourceQualityGateBlockedV0
	input.DecisionPayload.SourceRef = "quality-gate-ref-replan-blocked"
	input.DecisionPayload.FollowupRefs = []string{
		"capacity-request-ref-replan-quality-gate",
	}
	input.CapacityCandidate = validReplanCapacityCandidateV0("quality-gate", input.DecisionCommandMeta.RunID)
	input.CapacityCandidate.Payload.TaskRef = "task-ref-replan-quality-gate"
	input.AskDirectorCandidate = validReplanAskDirectorCandidateV0("quality-gate", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0 quality gate: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusCapacityRequestedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertCommandTypeV0(t, result.RecordReplanDecisionCommand, orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0)
	assertOptionalCommandTypeV0(t, result.RequestCapacityCommand, orquestacoreworkflow.OrchestrationCommandRequestCapacityV0)
	if result.AskDirectorCommand != nil || result.RequestAgentCommand != nil {
		t.Fatalf("followups inesperados: ask=%v agent=%v", result.AskDirectorCommand, result.RequestAgentCommand)
	}

	var decision orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0
	mustDecodeCommandPayloadV0(t, result.RecordReplanDecisionCommand, &decision)
	if decision.SourceRef != "quality-gate-ref-replan-blocked" {
		t.Fatalf("source_ref=%q", decision.SourceRef)
	}
	if !containsStringV0(decision.FollowupRefs, "capacity-request-ref-replan-quality-gate") {
		t.Fatalf("followup_refs sin capacidad opaca: %v", decision.FollowupRefs)
	}
	assertNoDocumentacionFollowupV0(t, result)
}

func TestBuildReplanFollowupsV0QualityGateBlockedSplitPreguntaAlDirector(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0, "quality-gate-ask")
	input.SourceKind = ReplanFollowupSourceQualityGateBlockedV0
	input.DecisionPayload.SourceRef = "quality-gate-ref-replan-ask"
	input.AskDirectorCandidate = validReplanAskDirectorCandidateV0("quality-gate-ask", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0 quality gate ask: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusAskDirectorV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertOptionalCommandTypeV0(t, result.AskDirectorCommand, orquestacoreworkflow.OrchestrationCommandAskDirectorV0)
	if result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil {
		t.Fatalf("split_task no debe crear followups automaticos en programacion: %+v", result)
	}
}

func assertNoDocumentacionFollowupV0(t *testing.T, result ReplanFollowupsResultV0) {
	t.Helper()
	for _, command := range replanResultCommandsV0(result) {
		if command.CommandType == orquestacoreworkflow.OrchestrationCommandOpenPhaseV0 {
			t.Fatalf("no esperaba OpenPhase: %+v", command)
		}
		if commandHasDocumentacionPhaseV0(t, command) {
			t.Fatalf("comando avanza a documentacion: %+v", command)
		}
	}
}

func replanResultCommandsV0(result ReplanFollowupsResultV0) []orquestacoreworkflow.OrchestrationCommandV0 {
	commands := []orquestacoreworkflow.OrchestrationCommandV0{result.RecordReplanDecisionCommand}
	for _, optional := range []*orquestacoreworkflow.OrchestrationCommandV0{
		result.RequestCapacityCommand,
		result.RequestAgentCommand,
		result.AskDirectorCommand,
	} {
		if optional != nil {
			commands = append(commands, *optional)
		}
	}
	return commands
}

func commandHasDocumentacionPhaseV0(t *testing.T, command orquestacoreworkflow.OrchestrationCommandV0) bool {
	t.Helper()
	var payload struct {
		PhaseID string `json:"phase_id"`
		Task    struct {
			PhaseID string `json:"phase_id"`
		} `json:"task"`
	}
	mustDecodeCommandPayloadV0(t, command, &payload)
	return payload.PhaseID == string(orquestacoreworkflow.OrchestrationPhaseDocumentacionV0) ||
		payload.Task.PhaseID == string(orquestacoreworkflow.OrchestrationPhaseDocumentacionV0)
}

func containsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
