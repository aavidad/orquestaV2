package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaCierreCompacto(t *testing.T) {
	cases := []DirectorAgentDecisionV0{
		validDirectorAgentFinalValidationDecisionV0(),
		validDirectorAgentCloseRunDecisionV0(),
	}
	for _, decision := range cases {
		if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", decision.CommandType, issues)
		}
	}
}

func TestValidateDirectorAgentDecisionV0RechazaCierreConReferenciaNoCompacta(t *testing.T) {
	decision := validDirectorAgentCloseRunDecisionV0()
	decision.CloseRun.EvidenceRefs = []string{"docs/cierre.md"}

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_ref_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaCierreConFaseIncoherente(t *testing.T) {
	decision := validDirectorAgentFinalValidationDecisionV0()
	decision.RegisterFinalValidation.PhaseID = "cierre"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_phase_mismatch",
	)
}

func validDirectorAgentFinalValidationDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-final-validation-001",
		RunID:         "run-ref-001",
		PhaseID:       "validacion_final",
		CommandType:   DirectorAgentCommandRegisterFinalValidationV0,
		CommandRef:    "command-ref-final-validation-001",
		Summary:       "Registrar validacion final.",
		EvidenceRefs:  []string{"evidence-ref-final-validation-001"},
		RegisterFinalValidation: &DirectorAgentFinalValidationCommandV0{
			ValidationRef:       "validation-ref-001",
			PhaseID:             "validacion_final",
			ClosedTaskRef:       "task-ref-agenda-001",
			Summary:             "Solicitud validada.",
			RequestKind:         "crear_app_completa",
			ExecutionMode:       "normal",
			MinimumDeliverables: []string{"programacion", "pruebas", "revision_final"},
			EvidenceRefs:        []string{"evidence-ref-final-validation-002"},
		},
	}
}

func validDirectorAgentCloseRunDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-close-run-001",
		RunID:         "run-ref-001",
		PhaseID:       "cierre",
		CommandType:   DirectorAgentCommandCloseRunV0,
		CommandRef:    "command-ref-close-run-001",
		Summary:       "Cerrar solicitud.",
		EvidenceRefs:  []string{"evidence-ref-close-run-001"},
		CloseRun: &DirectorAgentCloseRunCommandV0{
			ClosureRef:          "closure-ref-001",
			PhaseID:             "cierre",
			ValidationRef:       "validation-ref-001",
			Summary:             "Solicitud cerrada.",
			RequestKind:         "crear_app_completa",
			ExecutionMode:       "normal",
			MinimumDeliverables: []string{"programacion", "pruebas", "revision_final"},
			EvidenceRefs:        []string{"evidence-ref-close-run-002"},
		},
	}
}
