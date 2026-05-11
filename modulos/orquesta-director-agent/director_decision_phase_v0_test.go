package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaOpenPhaseCompacto(t *testing.T) {
	decision := validDirectorAgentOpenPhaseDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaVotacionCompacta(t *testing.T) {
	decision := validDirectorAgentVoteDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaDecisionCompacta(t *testing.T) {
	decision := validDirectorAgentAcceptDecisionDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func validDirectorAgentOpenPhaseDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-open-vote-001",
		RunID:         "run-ref-001",
		PhaseID:       "brainstorming_arquitectura",
		CommandType:   DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-director-open-vote-001",
		Summary:       "Abrir fase de decision.",
		EvidenceRefs:  []string{"evidence-ref-open-vote-001"},
		OpenPhase: &DirectorAgentOpenPhaseCommandV0{
			PhaseID: "votacion_y_decision",
			Reason:  "Preparar votacion compacta.",
		},
	}
}

func validDirectorAgentVoteDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-vote-001",
		RunID:         "run-ref-001",
		PhaseID:       "votacion_y_decision",
		CommandType:   DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-director-vote-001",
		Summary:       "Solicitar votacion compacta.",
		EvidenceRefs:  []string{"evidence-ref-vote-001"},
		RequestVote: &DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-ref-architecture-001",
			PhaseID:                    "votacion_y_decision",
			DecisionTopicRef:           "topic-ref-architecture-001",
			BrainstormRef:              "brainstorm-ref-director-001",
			Summary:                    "Elegir arquitectura hexagonal e i18n.",
			MinimumRecommendedCapacity: DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-vote-topic-001"},
		},
	}
}

func validDirectorAgentAcceptDecisionDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-accept-001",
		RunID:         "run-ref-001",
		PhaseID:       "votacion_y_decision",
		CommandType:   DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-director-accept-001",
		Summary:       "Aceptar decision compacta.",
		EvidenceRefs:  []string{"evidence-ref-accept-001"},
		AcceptDecision: &DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       "decision-ref-architecture-001",
			PhaseID:           "votacion_y_decision",
			VoteRef:           "vote-ref-architecture-001",
			AcceptedOptionRef: "option:hexagonal_i18n",
			Summary:           "Aceptar arquitectura hexagonal con i18n.",
			EvidenceRefs:      []string{"evidence-ref-accepted-option-001"},
		},
	}
}
