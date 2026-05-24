package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOrchestrationPhaseCatalogV0Cerrado(t *testing.T) {
	expected := []OrchestrationPhaseIDV0{
		OrchestrationPhaseDescubrimientoV0,
		OrchestrationPhaseBrainstormingArquitecturaV0,
		OrchestrationPhaseVotacionYDecisionV0,
		OrchestrationPhasePlanificacionMicrotareasV0,
		OrchestrationPhaseProgramacionV0,
		OrchestrationPhaseDocumentacionV0,
		OrchestrationPhaseIntegracionV0,
		OrchestrationPhaseRevisionV0,
		OrchestrationPhaseValidacionFinalV0,
		OrchestrationPhaseCierreV0,
	}

	got := SupportedOrchestrationPhaseIDsV0()
	if len(got) != len(expected) {
		t.Fatalf("fases esperadas=%d obtenidas=%d", len(expected), len(got))
	}

	for index, phase := range expected {
		if got[index] != phase {
			t.Fatalf("fase %d esperada=%q obtenida=%q", index, phase, got[index])
		}
		if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
			t.Fatalf("fase v0 soportada rechazada %q: %v", phase, err)
		}
	}
}

func TestOrchestrationPhaseCatalogV0TieneMetadataPura(t *testing.T) {
	for _, phase := range OrchestrationPhaseCatalogV0() {
		if phase.Status != OrchestrationPhaseStatusPendingV0 {
			t.Fatalf("fase %s con estado inicial no canonico: %s", phase.ID, phase.Status)
		}
		if len(phase.EntryCriteria) == 0 || len(phase.ExitCriteria) == 0 || len(phase.EvidenceRequired) == 0 {
			t.Fatalf("fase %s sin criterios o evidencia minima", phase.ID)
		}
		if issues := ValidateOrchestrationPhaseV0(phase); len(issues) > 0 {
			t.Fatalf("fase %s invalida: %+v", phase.ID, issues)
		}
	}
}

func TestValidateOrchestrationPhaseIDV0RechazaDesconocida(t *testing.T) {
	err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0("despliegue_produccion"))
	if err == nil {
		t.Fatal("fase desconocida aceptada")
	}

	issue, ok := err.(OrchestrationValidationIssueV0)
	if !ok {
		t.Fatalf("error publico inesperado: %T", err)
	}
	if issue.Code != OrchestrationFaseNoSoportadaV0 {
		t.Fatalf("codigo esperado=%s obtenido=%s", OrchestrationFaseNoSoportadaV0, issue.Code)
	}
	if IsSupportedOrchestrationPhaseV0(OrchestrationPhaseIDV0("despliegue_produccion")) {
		t.Fatal("helper de catalogo marco soportada una fase desconocida")
	}
}

func TestValidateOrchestrationStatusV0RechazaEstadoDesconocido(t *testing.T) {
	if err := ValidateOrchestrationRunStatusV0(OrchestrationRunStatusV0("ejecutando_en_runtime")); err == nil {
		t.Fatal("estado de run desconocido aceptado")
	}
	if err := ValidateOrchestrationPhaseStatusV0(OrchestrationPhaseStatusV0("en_terminal")); err == nil {
		t.Fatal("estado de fase desconocido aceptado")
	}
}

func TestOrchestrationRunV0SerializableSinAdaptadores(t *testing.T) {
	run := validRunForSerializationTestV0()
	if issues := ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run valido rechazado: %+v", issues)
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("serializacion fallida: %v", err)
	}

	serialized := strings.ToLower(string(data))
	for _, forbidden := range forbiddenSerializedDetailsV0() {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serializacion contiene detalle prohibido %q: %s", forbidden, serialized)
		}
	}
}

func TestValidateOrchestrationRunV0RejectsForbiddenRefs(t *testing.T) {
	run := validRunForSerializationTestV0()
	run.Tasks = []string{"task_ref_001", "api_key=valor"}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatal("run con detalle prohibido aceptado")
	}
	if issues[0].Code != OrchestrationDetalleProhibidoV0 || issues[0].Field != "tasks" {
		t.Fatalf("issue=%+v, want detalle_prohibido tasks", issues[0])
	}
}

func TestValidateOrchestrationRunV0RejectsAnsweredQuestionWithoutQuestion(t *testing.T) {
	run := validRunForSerializationTestV0()
	run.DirectorAnsweredQuestions = []string{"question_ref_missing"}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatal("run con pregunta respondida inexistente aceptado")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "director_answered_questions" {
		t.Fatalf("issue=%+v, want estado_inconsistente director_answered_questions", issues[0])
	}
}

func validRunForSerializationTestV0() OrchestrationRunV0 {
	phase := OrchestrationPhaseCatalogV0()[0]
	phase.Status = OrchestrationPhaseStatusActiveV0
	phase.OpenedAt = "2026-05-04T10:00:00Z"

	return OrchestrationRunV0{
		SchemaVersion:     OrchestrationRunSchemaVersionV0,
		RunID:             "run_ref_001",
		ProjectRef:        "project_ref_001",
		AppSpecRef:        "app_spec_ref_001",
		Status:            OrchestrationRunStatusActiveV0,
		CurrentPhase:      phase.ID,
		Phases:            []OrchestrationPhaseV0{phase},
		Brainstorms:       []string{"brainstorm_ref_001"},
		Votes:             []string{"vote_ref_001"},
		Tasks:             []string{"task_ref_001"},
		FunctionContracts: []string{"function_contract_ref_001"},
		Decisions:         []string{"decision_ref_001"},
		CapacityRequests:  []string{"capacity_request_ref_001"},
		Agents:            []string{"agent_ref_001"},
		Deliveries:        []string{"delivery_ref_001"},
		DeliveredTasks:    []string{"task_ref_001"},
		DeliveredAgents:   []string{"agent_ref_001"},
		Reviews:           []string{"review_ref_001"},
		AcceptedReviews:   []string{"accepted_review_ref_001"},
		ClosedTasks:       []string{"task_ref_001"},
		Validations:       []string{"validation_ref_001"},
		Closures:          []string{"closure_ref_001"},
		DirectorQuestions: []string{"director_question_ref_001"},
		Blockers:          []string{},
		LastEventID:       "event_ref_001",
	}
}

func forbiddenSerializedDetailsV0() []string {
	return operationalSensitiveFragmentsForTestV0()
}
