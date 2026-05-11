package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaConsultasCapacidadAgenteYReplanCompactos(t *testing.T) {
	tests := []DirectorAgentDecisionV0{
		validDirectorAgentAskDirectorDecisionV0(),
		validDirectorAgentAskUserDecisionV0(),
		validDirectorAgentRequestCapacityDecisionV0(),
		validDirectorAgentRequestAgentDecisionV0(),
		validDirectorAgentRequestReworkDecisionV0(),
		validDirectorAgentReplanDecisionV0(),
	}
	for _, decision := range tests {
		if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", decision.CommandType, issues)
		}
	}
}

func TestValidateDirectorAgentDecisionV0RechazaReviewStatusNoSoportado(t *testing.T) {
	decision := reviewResultDecisionForRoutingV0()
	decision.RecordReviewResult.Status = "approved"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_review_status_invalido",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaAskUserSinDestinoUsuario(t *testing.T) {
	decision := validDirectorAgentAskUserDecisionV0()
	decision.AskUser.TargetGroup = DirectorAgentTargetDirectorV0

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_target_invalido",
	)
}

func validDirectorAgentAskDirectorDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-ask-director-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   DirectorAgentCommandAskDirectorV0,
		CommandRef:    "command-ref-ask-director-001",
		Summary:       "Consultar al director por bloqueo interno.",
		EvidenceRefs:  []string{"evidence-ref-ask-director-001"},
		AskDirector: &DirectorAgentAskQuestionCommandV0{
			QuestionID:   "question-ref-director-001",
			SourceGroup:  "scheduler",
			TargetGroup:  DirectorAgentTargetDirectorV0,
			Summary:      "Elegir siguiente paso compacto.",
			Options:      []string{"replan", "stop_agent"},
			EvidenceRefs: []string{"evidence-ref-ask-director-002"},
		},
	}
}

func validDirectorAgentAskUserDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-ask-user-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   DirectorAgentCommandAskUserV0,
		CommandRef:    "command-ref-ask-user-001",
		Summary:       "Pedir dato faltante al usuario.",
		EvidenceRefs:  []string{"evidence-ref-ask-user-001"},
		AskUser: &DirectorAgentAskQuestionCommandV0{
			QuestionID:   "question-ref-user-001",
			SourceGroup:  "director",
			TargetGroup:  DirectorAgentTargetUserV0,
			Summary:      "Confirmar alcance antes de seguir.",
			Options:      []string{"continuar_minimo", "ampliar_alcance"},
			EvidenceRefs: []string{"evidence-ref-ask-user-002"},
			Blocking:     true,
		},
	}
}

func validDirectorAgentRequestCapacityDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-capacity-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   DirectorAgentCommandRequestCapacityV0,
		CommandRef:    "command-ref-capacity-001",
		Summary:       "Pedir capacidad para microtarea.",
		EvidenceRefs:  []string{"evidence-ref-capacity-001"},
		RequestCapacity: &DirectorAgentRequestCapacityCommandV0{
			CapacityRequestID:          "capacity-request-ref-001",
			PhaseID:                    "programacion",
			TaskRef:                    "task-ref-001",
			ReasonCode:                 "programming_task",
			Summary:                    "Resolver una unidad pequena.",
			MinimumRecommendedCapacity: DirectorAgentCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-capacity-002"},
		},
	}
}

func validDirectorAgentRequestAgentDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agent-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   DirectorAgentCommandRequestAgentV0,
		CommandRef:    "command-ref-agent-001",
		Summary:       "Pedir agente especializado.",
		EvidenceRefs:  []string{"evidence-ref-agent-001"},
		RequestAgent: &DirectorAgentRequestAgentCommandV0{
			AgentRequestID:     "agent-request-ref-001",
			PhaseID:            "programacion",
			TaskRef:            "task-ref-001",
			CapacityRequestRef: "capacity-request-ref-001",
			Role:               "programador",
			Summary:            "Ejecutar solo la microtarea asignada.",
			EvidenceRefs:       []string{"evidence-ref-agent-002"},
		},
	}
}

func validDirectorAgentRequestReworkDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-rework-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandRequestReworkV0,
		CommandRef:    "command-ref-rework-001",
		Summary:       "Solicitar retrabajo de entrega rechazada.",
		EvidenceRefs:  []string{"evidence-ref-rework-001"},
		RequestRework: &DirectorAgentRequestReworkCommandV0{
			ReworkRequestRef: "rework-request-ref-001",
			PhaseID:          "revision",
			ReviewResultRef:  "review-result-ref-001",
			ReviewRequestID:  "review-request-ref-001",
			DeliveryRef:      "delivery-ref-001",
			Summary:          "Corregir los criterios incumplidos.",
			EvidenceRefs:     []string{"evidence-ref-rework-002"},
		},
	}
}

func validDirectorAgentReplanDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-replan-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   DirectorAgentCommandRecordReplanDecisionV0,
		CommandRef:    "command-ref-replan-001",
		Summary:       "Registrar replan compacto.",
		EvidenceRefs:  []string{"evidence-ref-replan-001"},
		RecordReplanDecision: &DirectorAgentReplanDecisionCommandV0{
			ReplanRef:      "replan-ref-001",
			RunRef:         "run-ref-001",
			TaskRef:        "task-ref-001",
			SourceRef:      "rework-request-ref-001",
			AcceptedAction: DirectorAgentReplanActionRetryTaskV0,
			FollowupRefs:   []string{"capacity-request-ref-002"},
			Summary:        "Reintentar tarea con alcance acotado.",
			EvidenceRefs:   []string{"evidence-ref-replan-002"},
		},
	}
}

func reviewResultDecisionForRoutingV0() DirectorAgentDecisionV0 {
	decision := validDirectorAgentRequestReworkDecisionV0()
	decision.DecisionRef = "director-decision-review-result-routing-001"
	decision.CommandType = DirectorAgentCommandRecordReviewResultV0
	decision.CommandRef = "command-ref-review-result-routing-001"
	decision.RecordReviewResult = &DirectorAgentReviewResultCommandV0{
		ReviewResultRef: "review-result-ref-001",
		ReviewRequestID: "review-request-ref-001",
		DeliveryRef:     "delivery-ref-001",
		Status:          DirectorAgentReviewStatusRejectedV0,
		Summary:         "Entrega rechazada por criterio incumplido.",
		EvidenceRefs:    []string{"evidence-ref-review-result-001"},
	}
	decision.RequestRework = nil
	return decision
}
