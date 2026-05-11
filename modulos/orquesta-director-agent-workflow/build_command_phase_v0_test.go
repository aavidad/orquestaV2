package orquestadirectoragentworkflow

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceOpenPhase(t *testing.T) {
	command, issues := BuildDirectorAgentWorkflowCommandV0(
		validDirectorAgentWorkflowOpenVoteRequestForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandOpenPhaseV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0TraduceVote(t *testing.T) {
	command, issues := BuildDirectorAgentWorkflowCommandV0(
		validDirectorAgentWorkflowVoteRequestForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandRequestVoteV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0TraduceAcceptDecision(t *testing.T) {
	command, issues := BuildDirectorAgentWorkflowCommandV0(
		validDirectorAgentWorkflowAcceptRequestForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandAcceptDecisionV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
}

func validDirectorAgentWorkflowOpenVoteRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-open-vote-001",
		RunID:         "run-ref-001",
		PhaseID:       "brainstorming_arquitectura",
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-director-open-vote-001",
		Summary:       "Abrir fase de decision.",
		EvidenceRefs:  []string{"evidence-ref-open-vote-001"},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: "votacion_y_decision",
			Reason:  "Preparar votacion compacta.",
		},
	}
	return request
}

func validDirectorAgentWorkflowVoteRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-vote-001",
		RunID:         "run-ref-001",
		PhaseID:       "votacion_y_decision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-director-vote-001",
		Summary:       "Solicitar votacion compacta.",
		EvidenceRefs:  []string{"evidence-ref-vote-001"},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-ref-architecture-001",
			PhaseID:                    "votacion_y_decision",
			DecisionTopicRef:           "topic-ref-architecture-001",
			BrainstormRef:              "brainstorm-ref-director-001",
			Summary:                    "Elegir arquitectura hexagonal e i18n.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-vote-topic-001"},
		},
	}
	return request
}

func validDirectorAgentWorkflowAcceptRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-accept-001",
		RunID:         "run-ref-001",
		PhaseID:       "votacion_y_decision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-director-accept-001",
		Summary:       "Aceptar decision compacta.",
		EvidenceRefs:  []string{"evidence-ref-accept-001"},
		AcceptDecision: &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       "decision-ref-architecture-001",
			PhaseID:           "votacion_y_decision",
			VoteRef:           "vote-ref-architecture-001",
			AcceptedOptionRef: "option:hexagonal_i18n",
			Summary:           "Aceptar arquitectura hexagonal con i18n.",
			EvidenceRefs:      []string{"evidence-ref-accepted-option-001"},
		},
	}
	return request
}
