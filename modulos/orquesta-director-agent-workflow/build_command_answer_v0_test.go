package orquestadirectoragentworkflow

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceAnswerQuestion(t *testing.T) {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-answer-001",
		RunID:         "run-ref-001",
		PhaseID:       "programacion",
		CommandType:   orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0,
		CommandRef:    "command-ref-director-answer-001",
		Summary:       "Responder consulta.",
		EvidenceRefs:  []string{"evidence-ref-answer-001"},
		AnswerQuestion: &orquestadirectoragent.DirectorAgentAnswerQuestionCommandV0{
			AnswerID:     "answer-ref-question-001",
			QuestionID:   "question-ref-app-change-001",
			Decision:     orquestadirectoragent.DirectorAgentAnswerReplanV0,
			Summary:      "Aceptar replanificacion.",
			EvidenceRefs: []string{"evidence-ref-answer-001"},
		},
	}

	command, issues := BuildDirectorAgentWorkflowCommandV0(request)

	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandAnswerDirectorQuestionV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
	var payload orquestacoreworkflow.AnswerDirectorQuestionCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.QuestionID != "question-ref-app-change-001" ||
		payload.Decision != orquestacoreworkflow.DirectorAnswerDecisionReplanV0 {
		t.Fatalf("payload=%+v", payload)
	}
}
