package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaRevisionCompacta(t *testing.T) {
	cases := []DirectorAgentDecisionV0{
		validDirectorAgentRequestReviewDecisionV0(),
		validDirectorAgentRecordReviewResultDecisionV0(),
		validDirectorAgentAcceptReviewDecisionV0(),
		validDirectorAgentCloseTaskDecisionV0(),
	}
	for _, decision := range cases {
		if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", decision.CommandType, issues)
		}
	}
}

func TestValidateDirectorAgentDecisionV0RechazaCloseTaskSinEvidencia(t *testing.T) {
	decision := validDirectorAgentCloseTaskDecisionV0()
	decision.CloseTask.EvidenceRefs = nil

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_evidence_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaCloseTaskFueraDeRevision(t *testing.T) {
	decision := validDirectorAgentCloseTaskDecisionV0()
	decision.CloseTask.PhaseID = "validacion_final"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_phase_invalida",
	)
}

func validDirectorAgentRequestReviewDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-review-request-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandRequestReviewV0,
		CommandRef:    "command-ref-review-request-001",
		Summary:       "Solicitar revision de entrega.",
		EvidenceRefs:  []string{"evidence-ref-review-request-001"},
		RequestReview: &DirectorAgentRequestReviewCommandV0{
			ReviewRequestID: "review-request-ref-001",
			PhaseID:         "revision",
			DeliveryRef:     "delivery-ref-001",
			Summary:         "Revisar entrega compacta.",
			EvidenceRefs:    []string{"evidence-ref-review-request-002"},
		},
	}
}

func validDirectorAgentRecordReviewResultDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-review-result-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandRecordReviewResultV0,
		CommandRef:    "command-ref-review-result-001",
		Summary:       "Registrar revision aceptada.",
		EvidenceRefs:  []string{"evidence-ref-review-result-001"},
		RecordReviewResult: &DirectorAgentReviewResultCommandV0{
			ReviewResultRef: "review-result-ref-001",
			ReviewRequestID: "review-request-ref-001",
			DeliveryRef:     "delivery-ref-001",
			Status:          "accepted",
			Summary:         "Entrega revisada sin cambios bloqueantes.",
			EvidenceRefs:    []string{"evidence-ref-review-result-002"},
			QualityGateRef:  "quality-gate-ref-001",
		},
	}
}

func validDirectorAgentAcceptReviewDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-accept-review-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandAcceptReviewV0,
		CommandRef:    "command-ref-accept-review-001",
		Summary:       "Aceptar revision.",
		EvidenceRefs:  []string{"evidence-ref-accept-review-001"},
		AcceptReview: &DirectorAgentAcceptReviewCommandV0{
			AcceptedReviewRef: "accepted-review-ref-001",
			PhaseID:           "revision",
			ReviewRequestID:   "review-request-ref-001",
			DeliveryRef:       "delivery-ref-001",
			Summary:           "Revision aceptada.",
			EvidenceRefs:      []string{"evidence-ref-accept-review-002"},
		},
	}
}

func validDirectorAgentCloseTaskDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-close-task-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandCloseTaskV0,
		CommandRef:    "command-ref-close-task-001",
		Summary:       "Cerrar microtarea revisada.",
		EvidenceRefs:  []string{"evidence-ref-close-task-001"},
		CloseTask: &DirectorAgentCloseTaskCommandV0{
			TaskID:            "task-ref-agenda-001",
			PhaseID:           "revision",
			DeliveryRef:       "delivery-ref-001",
			AcceptedReviewRef: "accepted-review-ref-001",
			Summary:           "Microtarea cerrada tras revision aceptada.",
			EvidenceRefs:      []string{"evidence-ref-close-task-002"},
		},
	}
}
