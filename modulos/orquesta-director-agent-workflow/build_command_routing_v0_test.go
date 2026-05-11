package orquestadirectoragentworkflow

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceConsultasCapacidadAgenteReworkYReplan(t *testing.T) {
	tests := []struct {
		name     string
		decision orquestadirectoragent.DirectorAgentDecisionV0
		wantType string
	}{
		{
			name:     "ask_user",
			decision: workflowAskUserDecisionV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
		},
		{
			name:     "request_capacity",
			decision: workflowRequestCapacityDecisionV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
		},
		{
			name:     "request_agent",
			decision: workflowRequestAgentDecisionV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
		},
		{
			name:     "request_rework",
			decision: workflowRequestReworkDecisionV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandRequestReworkV0,
		},
		{
			name:     "record_replan_decision",
			decision: workflowReplanDecisionV0(),
			wantType: orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		},
	}
	for _, tt := range tests {
		request := validDirectorAgentWorkflowRequestForTestV0()
		request.Decision = tt.decision

		command, issues := BuildDirectorAgentWorkflowCommandV0(request)

		if len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", tt.name, issues)
		}
		if command.CommandType != tt.wantType {
			t.Fatalf("%s command_type=%s want %s", tt.name, command.CommandType, tt.wantType)
		}
		if !json.Valid(command.Payload) {
			t.Fatalf("%s payload invalido: %s", tt.name, string(command.Payload))
		}
	}
}

func TestBuildDirectorAgentWorkflowCommandV0AskUserMantieneTargetUser(t *testing.T) {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = workflowAskUserDecisionV0()

	command, issues := BuildDirectorAgentWorkflowCommandV0(request)

	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	var payload orquestacoreworkflow.AskDirectorCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.TargetGroup != orquestadirectoragent.DirectorAgentTargetUserV0 || !payload.Blocking {
		t.Fatalf("payload=%+v", payload)
	}
}

func workflowAskUserDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-ask-user-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   orquestadirectoragent.DirectorAgentCommandAskUserV0,
		CommandRef:    "command-ref-ask-user-001",
		Summary:       "Pedir dato faltante al usuario.",
		EvidenceRefs:  []string{"evidence-ref-ask-user-001"},
		AskUser: &orquestadirectoragent.DirectorAgentAskQuestionCommandV0{
			QuestionID:   "question-ref-user-001",
			SourceGroup:  "director",
			TargetGroup:  orquestadirectoragent.DirectorAgentTargetUserV0,
			Summary:      "Confirmar alcance antes de seguir.",
			Options:      []string{"continuar_minimo", "ampliar_alcance"},
			EvidenceRefs: []string{"evidence-ref-ask-user-002"},
			Blocking:     true,
		},
	}
}

func workflowRequestCapacityDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-capacity-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestCapacityV0,
		CommandRef:    "command-ref-capacity-001",
		Summary:       "Pedir capacidad para microtarea.",
		EvidenceRefs:  []string{"evidence-ref-capacity-001"},
		RequestCapacity: &orquestadirectoragent.DirectorAgentRequestCapacityCommandV0{
			CapacityRequestID:          "capacity-request-ref-001",
			PhaseID:                    "programacion",
			TaskRef:                    "task-ref-001",
			ReasonCode:                 "programming_task",
			Summary:                    "Resolver una unidad pequena.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-capacity-002"},
		},
	}
}

func workflowRequestAgentDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-agent-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestAgentV0,
		CommandRef:    "command-ref-agent-001",
		Summary:       "Pedir agente especializado.",
		EvidenceRefs:  []string{"evidence-ref-agent-001"},
		RequestAgent: &orquestadirectoragent.DirectorAgentRequestAgentCommandV0{
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

func workflowRequestReworkDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-rework-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestReworkV0,
		CommandRef:    "command-ref-rework-001",
		Summary:       "Solicitar retrabajo compacto.",
		EvidenceRefs:  []string{"evidence-ref-rework-001"},
		RequestRework: &orquestadirectoragent.DirectorAgentRequestReworkCommandV0{
			ReworkRequestRef: "rework-request-ref-001",
			PhaseID:          "revision",
			ReviewResultRef:  "review-result-ref-001",
			ReviewRequestID:  "review-request-ref-001",
			DeliveryRef:      "delivery-ref-001",
			Summary:          "Corregir criterio incumplido.",
			EvidenceRefs:     []string{"evidence-ref-rework-002"},
		},
	}
}

func workflowReplanDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-replan-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRecordReplanDecisionV0,
		CommandRef:    "command-ref-replan-001",
		Summary:       "Registrar replan compacto.",
		EvidenceRefs:  []string{"evidence-ref-replan-001"},
		RecordReplanDecision: &orquestadirectoragent.DirectorAgentReplanDecisionCommandV0{
			ReplanRef:      "replan-ref-001",
			RunRef:         "run-ref-001",
			TaskRef:        "task-ref-001",
			SourceRef:      "rework-request-ref-001",
			AcceptedAction: orquestadirectoragent.DirectorAgentReplanActionRetryTaskV0,
			FollowupRefs:   []string{"capacity-request-ref-002"},
			Summary:        "Reintentar tarea con alcance acotado.",
			EvidenceRefs:   []string{"evidence-ref-replan-002"},
		},
	}
}
