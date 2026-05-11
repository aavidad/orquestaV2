package orquestadirectoragentworkflow

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceCierre(t *testing.T) {
	tests := []struct {
		name     string
		decision orquestadirectoragent.DirectorAgentDecisionV0
		wantType string
	}{
		{
			name:     "close_task",
			decision: closeTaskDecisionForWorkflowTestV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandCloseTaskV0,
		},
		{
			name:     "validacion_final",
			decision: finalValidationDecisionForWorkflowTestV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandRegisterFinalValidationV0,
		},
		{
			name:     "cierre",
			decision: closeRunDecisionForWorkflowTestV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandCloseRunV0,
		},
	}
	for _, tt := range tests {
		command, issues := BuildDirectorAgentWorkflowCommandV0(
			DirectorAgentWorkflowCommandRequestV0{
				Decision:      tt.decision,
				OccurredAt:    "2026-05-10T12:00:00Z",
				CorrelationID: "corr-close-workflow-001",
				RequestedBy:   "orquesta-director-agent-workflow-test",
			},
		)
		if len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", tt.name, issues)
		}
		if command.CommandType != tt.wantType || !json.Valid(command.Payload) {
			t.Fatalf("%s command=%s payload=%s", tt.name, command.CommandType, string(command.Payload))
		}
	}
}

func closeTaskDecisionForWorkflowTestV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-close-task-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandCloseTaskV0,
		CommandRef:    "command-ref-close-task-001",
		Summary:       "Cerrar microtarea revisada.",
		EvidenceRefs:  []string{"evidence-ref-close-task-001"},
		CloseTask: &orquestadirectoragent.DirectorAgentCloseTaskCommandV0{
			TaskID:            "task-ref-agenda-001",
			PhaseID:           "revision",
			DeliveryRef:       "delivery-ref-001",
			AcceptedReviewRef: "accepted-review-ref-001",
			Summary:           "Microtarea cerrada tras revision aceptada.",
			EvidenceRefs:      []string{"evidence-ref-close-task-002"},
		},
	}
}

func finalValidationDecisionForWorkflowTestV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-final-validation-001",
		RunID:         "run-ref-001",
		PhaseID:       "validacion_final",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRegisterFinalValidationV0,
		CommandRef:    "command-ref-final-validation-001",
		Summary:       "Registrar validacion final.",
		EvidenceRefs:  []string{"evidence-ref-final-validation-001"},
		RegisterFinalValidation: &orquestadirectoragent.DirectorAgentFinalValidationCommandV0{
			ValidationRef:       "validation-ref-001",
			PhaseID:             "validacion_final",
			ClosedTaskRef:       "task-ref-001",
			Summary:             "Solicitud validada.",
			RequestKind:         "crear_app_completa",
			ExecutionMode:       "normal",
			MinimumDeliverables: []string{"programacion", "pruebas", "revision_final"},
			EvidenceRefs:        []string{"evidence-ref-final-validation-002"},
		},
	}
}

func closeRunDecisionForWorkflowTestV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-close-run-001",
		RunID:         "run-ref-001",
		PhaseID:       "cierre",
		CommandType:   orquestadirectoragent.DirectorAgentCommandCloseRunV0,
		CommandRef:    "command-ref-close-run-001",
		Summary:       "Cerrar solicitud.",
		EvidenceRefs:  []string{"evidence-ref-close-run-001"},
		CloseRun: &orquestadirectoragent.DirectorAgentCloseRunCommandV0{
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
