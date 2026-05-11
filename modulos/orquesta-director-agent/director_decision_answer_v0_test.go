package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaRespuestaDirectorCompacta(t *testing.T) {
	decision := DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-answer-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   DirectorAgentCommandAnswerQuestionV0,
		CommandRef:    "command-ref-director-answer-001",
		Summary:       "Responder consulta del usuario.",
		EvidenceRefs:  []string{"evidence-ref-answer-001"},
		AnswerQuestion: &DirectorAgentAnswerQuestionCommandV0{
			AnswerID:     "answer-ref-question-001",
			QuestionID:   "question-ref-app-change-001",
			Decision:     DirectorAgentAnswerReplanV0,
			Summary:      "Aceptar replanificacion compacta.",
			EvidenceRefs: []string{"evidence-ref-answer-001"},
		},
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}
