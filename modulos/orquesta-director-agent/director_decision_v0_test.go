package orquestadirectoragent

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateDirectorAgentDecisionV0AceptaBrainstormCompacto(t *testing.T) {
	decision := validDirectorAgentDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaContratoCompacto(t *testing.T) {
	decision := validDirectorAgentPublishContractDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaMicrotareaCompacta(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaMicrotareaConWriteSetAmplio(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.WriteSet = []string{
		"go.mod",
		"cmd/server/main.go",
		"internal/domain",
		"internal/application",
		"internal/ports",
		"internal/httpapi",
		"internal/webadmin",
		"internal/i18n",
		"internal/memory",
		"README.md",
		"docs",
		"tests",
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaWriteSetYTestsDeConector(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.WriteSet = []string{
		"modulos/orquesta-runtime-codex-delivery",
		"modulos/orquesta-app-codex-stack",
	}
	decision.CreateMicrotask.Task.RequiredTests = []string{
		"go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack",
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaMicrotareaDuranteProgramacion(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.PhaseID = decision.CreateMicrotask.Task.PhaseID

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaMicrotareaConLinajeRecursivo(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.ParentTaskRef = "task-ref-parent-001"
	decision.CreateMicrotask.Task.CohortRef = "cohort-ref-recursive-001"
	decision.CreateMicrotask.Task.WaveRef = "wave-ref-recursive-001"
	decision.CreateMicrotask.Task.DelegationDepth = 2
	decision.CreateMicrotask.Task.MaxChildAgents = 4
	decision.CreateMicrotask.Task.ChildTaskRefs = []string{"task-ref-child-001", "task-ref-child-002"}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaMicrotareaConContextRefs(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.ContextRefs = []string{
		"context-ref-scope-001",
		"context-ref-policy-001",
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0AceptaChildRefsHastaLimiteRecursivo(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.MaxChildAgents = maxDirectorAgentRecursionLimitV0
	decision.CreateMicrotask.Task.ChildTaskRefs = make([]string, 0, maxDirectorAgentRecursionLimitV0)
	for index := 0; index < maxDirectorAgentRecursionLimitV0; index++ {
		decision.CreateMicrotask.Task.ChildTaskRefs = append(
			decision.CreateMicrotask.Task.ChildTaskRefs,
			fmt.Sprintf("task-ref-child-%03d", index+1),
		)
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0RechazaProveedorYModelo(t *testing.T) {
	decision := validDirectorAgentDecisionV0()
	decision.Summary = "usar provider y modelo concretos"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_texto_invalido",
	)
}

func TestValidateDirectorAgentDecisionV0NoRechazaModelarComoModelo(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.Summary = "Modelar eventos, rangos horarios y errores de negocio."

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0RechazaComandoSinPayload(t *testing.T) {
	decision := validDirectorAgentDecisionV0()
	decision.RequestBrainstorm = nil

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_payload_requerido",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaCapacidadNoSoportada(t *testing.T) {
	decision := validDirectorAgentDecisionV0()
	decision.RequestBrainstorm.MinimumRecommendedCapacity = "premium"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_capacidad_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaAnalisisLargo(t *testing.T) {
	decision := validDirectorAgentDecisionV0()
	decision.Summary = strings.Repeat("x", maxDirectorAgentStringV0+1)

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_texto_invalido",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaMicrotareaSinContrato(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.FunctionContractRefs = nil

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_lista_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaMicrotareaProgramacionSinRequiredTests(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.RequiredTests = nil

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_lista_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaDependenciaNoCompacta(t *testing.T) {
	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.CreateMicrotask.Task.DependsOn = []string{"../task-bootstrap"}

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_ref_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaContextRefsNoCompactas(t *testing.T) {
	for _, refs := range [][]string{
		{"context ref invalid"},
		{"external/context-ref-001"},
		{"token-ref-context-001"},
	} {
		decision := validDirectorAgentCreateMicrotaskDecisionV0()
		decision.CreateMicrotask.Task.ContextRefs = refs

		requireDirectorAgentIssueV0(t,
			ValidateDirectorAgentDecisionV0(decision),
			"director_agent_ref_invalida",
		)
	}
}

func TestValidateDirectorAgentDecisionV0RechazaLinajeRecursivoInvalido(t *testing.T) {
	for _, mutate := range []func(*DirectorAgentDecisionV0){
		func(decision *DirectorAgentDecisionV0) {
			decision.CreateMicrotask.Task.ParentTaskRef = decision.CreateMicrotask.Task.TaskID
		},
		func(decision *DirectorAgentDecisionV0) {
			decision.CreateMicrotask.Task.ParentTaskRef = "../task-parent"
		},
		func(decision *DirectorAgentDecisionV0) {
			decision.CreateMicrotask.Task.ChildTaskRefs = []string{decision.CreateMicrotask.Task.TaskID}
		},
		func(decision *DirectorAgentDecisionV0) { decision.CreateMicrotask.Task.DelegationDepth = -1 },
		func(decision *DirectorAgentDecisionV0) {
			decision.CreateMicrotask.Task.MaxChildAgents = maxDirectorAgentRecursionLimitV0 + 1
		},
	} {
		decision := validDirectorAgentCreateMicrotaskDecisionV0()
		mutate(&decision)
		requireDirectorAgentIssueV0(t,
			ValidateDirectorAgentDecisionV0(decision),
			"director_agent_ref_invalida",
			"director_agent_numero_invalido",
		)
	}
}

func validDirectorAgentDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-ref-001",
		RunID:         "run-ref-001",
		PhaseID:       "brainstorming_arquitectura",
		CommandType:   DirectorAgentCommandRequestBrainstormV0,
		CommandRef:    "command-ref-director-brainstorm-001",
		Summary:       "Proponer inicio de analisis de arquitectura.",
		EvidenceRefs:  []string{"evidence-ref-director-001"},
		RequestBrainstorm: &DirectorAgentBrainstormCommandV0{
			BrainstormRequestID:        "brainstorm-ref-director-001",
			PhaseID:                    "brainstorming_arquitectura",
			TopicRef:                   "topic-ref-director-001",
			Summary:                    "Evaluar arquitectura compacta y segura.",
			MinimumRecommendedCapacity: DirectorAgentCapacityXHighV0,
			EvidenceRefs:               []string{"evidence-ref-brainstorm-001"},
		},
	}
}

func validDirectorAgentPublishContractDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-contract-001",
		RunID:         "run-ref-001",
		PhaseID:       "planificacion_microtareas",
		CommandType:   DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-director-contract-001",
		Summary:       "Publicar contrato funcional compacto.",
		EvidenceRefs:  []string{"evidence-ref-contract-001"},
		PublishContract: &DirectorAgentPublishContractCommandV0{
			ContractRef:   "contract:function:agenda:v0",
			PhaseID:       "planificacion_microtareas",
			DecisionRef:   "decision-ref-architecture-001",
			Summary:       "Contrato funcional para agenda.",
			FunctionNames: []string{"AgendaUseCases"},
			EvidenceRefs:  []string{"evidence-ref-decision-001"},
		},
	}
}

func validDirectorAgentCreateMicrotaskDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-task-001",
		RunID:         "run-ref-001",
		PhaseID:       "planificacion_microtareas",
		CommandType:   DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-director-task-001",
		Summary:       "Crear microtarea compacta.",
		EvidenceRefs:  []string{"evidence-ref-task-001"},
		CreateMicrotask: &DirectorAgentCreateMicrotaskCommandV0{
			Task: DirectorAgentMicrotaskV0{
				SchemaVersion: DirectorAgentMicrotaskSchemaVersionV0,
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
				DependsOn:     []string{"task-ref-bootstrap-001"},
				FunctionContractRefs: []DirectorAgentFunctionContractRefV0{
					{ContractRef: "contract:function:agenda:v0", FunctionName: "AgendaUseCases"},
				},
			},
		},
	}
}

func requireDirectorAgentIssueV0(t *testing.T, issues []DirectorAgentDecisionIssueV0, codes ...string) {
	t.Helper()
	for _, issue := range issues {
		for _, code := range codes {
			if issue.Code == code {
				return
			}
		}
	}
	t.Fatalf("no se encontro ninguna issue de %+v en %+v", codes, issues)
}
