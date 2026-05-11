package orquestadirectoragentworkflow

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceBrainstorm(t *testing.T) {
	command, issues := BuildDirectorAgentWorkflowCommandV0(
		validDirectorAgentWorkflowRequestForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandRequestBrainstormV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
	if command.IdempotencyKey != "idem-command-ref-director-brainstorm-001" ||
		command.RequestedBy != "orquesta-director-agent-workflow-test" {
		t.Fatalf("meta=%+v", command)
	}
	var payload orquestacoreworkflow.RequestBrainstormCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.BrainstormRequestID != "brainstorm-ref-director-001" ||
		payload.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0TraduceContrato(t *testing.T) {
	command, issues := BuildDirectorAgentWorkflowCommandV0(
		validDirectorAgentWorkflowContractRequestForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandPublishFunctionContractV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
	var payload orquestacoreworkflow.PublishFunctionContractCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.ContractRef != "contract:function:agenda:v0" ||
		payload.DecisionRef != "decision-ref-architecture-001" {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0TraduceMicrotarea(t *testing.T) {
	request := validDirectorAgentWorkflowMicrotaskRequestForTestV0()
	request.Decision.CreateMicrotask.Task.DependsOn = []string{"task-ref-bootstrap-001"}
	command, issues := BuildDirectorAgentWorkflowCommandV0(request)
	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0 {
		t.Fatalf("command_type=%s", command.CommandType)
	}
	var payload orquestacoreworkflow.CreateMicrotaskCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.Task.TaskID != "task-ref-agenda-001" ||
		payload.Task.FunctionContractRefs[0].ContractRef != "contract:function:agenda:v0" {
		t.Fatalf("payload=%+v", payload)
	}
	if len(payload.Task.RequiredTests) != 1 || payload.Task.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%v", payload.Task.RequiredTests)
	}
	if len(payload.Task.DependsOn) != 1 || payload.Task.DependsOn[0] != "task-ref-bootstrap-001" {
		t.Fatalf("depends_on=%v", payload.Task.DependsOn)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0IdempotenciaUsaCommandRef(t *testing.T) {
	request := validDirectorAgentWorkflowContractRequestForTestV0()
	request.Decision.DecisionRef = "decision-ref-architecture-accepted-001"
	request.Decision.CommandRef = "command-ref-publish-contract-001"
	request.Decision.PublishContract.DecisionRef = "decision-ref-architecture-accepted-001"

	command, issues := BuildDirectorAgentWorkflowCommandV0(request)

	if len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
	if command.IdempotencyKey != "idem-command-ref-publish-contract-001" {
		t.Fatalf("idempotency_key=%q", command.IdempotencyKey)
	}
}

func TestBuildDirectorAgentWorkflowCommandV0RechazaDecisionInvalida(t *testing.T) {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision.Summary = "usar provider concreto"

	_, issues := BuildDirectorAgentWorkflowCommandV0(request)

	requireDirectorAgentWorkflowIssueV0(t, issues, "director_agent_texto_invalido")
}

func TestBuildDirectorAgentWorkflowCommandV0RequiereOccurredAt(t *testing.T) {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.OccurredAt = ""

	_, issues := BuildDirectorAgentWorkflowCommandV0(request)

	requireDirectorAgentWorkflowIssueV0(t, issues, "director_agent_workflow_required")
}

func validDirectorAgentWorkflowRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	return DirectorAgentWorkflowCommandRequestV0{
		OccurredAt:    "2026-05-09T23:20:00Z",
		CorrelationID: "corr-director-agent-workflow-001",
		RequestedBy:   "orquesta-director-agent-workflow-test",
		Decision: orquestadirectoragent.DirectorAgentDecisionV0{
			SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
			DecisionRef:   "director-decision-ref-001",
			RunID:         "run-ref-001",
			PhaseID:       "brainstorming_arquitectura",
			CommandType:   orquestadirectoragent.DirectorAgentCommandRequestBrainstormV0,
			CommandRef:    "command-ref-director-brainstorm-001",
			Summary:       "Proponer inicio de analisis de arquitectura.",
			EvidenceRefs:  []string{"evidence-ref-director-001"},
			RequestBrainstorm: &orquestadirectoragent.DirectorAgentBrainstormCommandV0{
				BrainstormRequestID:        "brainstorm-ref-director-001",
				PhaseID:                    "brainstorming_arquitectura",
				TopicRef:                   "topic-ref-director-001",
				Summary:                    "Evaluar arquitectura compacta y segura.",
				MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityXHighV0,
				EvidenceRefs:               []string{"evidence-ref-brainstorm-001"},
			},
		},
	}
}

func validDirectorAgentWorkflowContractRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-contract-001",
		RunID:         "run-ref-001",
		PhaseID:       "planificacion_microtareas",
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-director-contract-001",
		Summary:       "Publicar contrato funcional compacto.",
		EvidenceRefs:  []string{"evidence-ref-contract-001"},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef:   "contract:function:agenda:v0",
			PhaseID:       "planificacion_microtareas",
			DecisionRef:   "decision-ref-architecture-001",
			Summary:       "Contrato funcional para agenda.",
			FunctionNames: []string{"AgendaUseCases"},
			EvidenceRefs:  []string{"evidence-ref-decision-001"},
		},
	}
	return request
}

func validDirectorAgentWorkflowMicrotaskRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-task-001",
		RunID:         "run-ref-001",
		PhaseID:       "planificacion_microtareas",
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-director-task-001",
		Summary:       "Crear microtarea compacta.",
		EvidenceRefs:  []string{"evidence-ref-task-001"},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:        "task-ref-agenda-001",
				RunID:         "run-ref-001",
				PhaseID:       "programacion",
				Title:         "Crear caso de uso de agenda",
				Summary:       "Implementar caso de uso principal.",
				WriteSet:      []string{"internal/agenda"},
				AcceptanceCriteria: []string{
					"Compila con pruebas unitarias.",
					"Expone contratos internos pequenos.",
				},
				RequiredTests: []string{"go test ./..."},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
					{ContractRef: "contract:function:agenda:v0", FunctionName: "AgendaUseCases"},
				},
			},
		},
	}
	return request
}

func requireDirectorAgentWorkflowIssueV0(
	t *testing.T,
	issues []DirectorAgentWorkflowIssueV0,
	code string,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro issue %q en %+v", code, issues)
}
