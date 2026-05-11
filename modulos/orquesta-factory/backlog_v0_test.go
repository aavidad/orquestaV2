package orquestafactory

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGenerarBacklogInicialPropuestoV0DesdeSpecValida(t *testing.T) {
	spec := validBacklogSpecV0(t)

	backlog, issues := GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if backlog.SchemaVersion != BacklogInicialPropuestoSchemaV0 {
		t.Fatalf("schema=%q, want %q", backlog.SchemaVersion, BacklogInicialPropuestoSchemaV0)
	}
	if backlog.SpecID != spec.SpecID {
		t.Fatalf("spec_id=%q, want %q", backlog.SpecID, spec.SpecID)
	}
	if len(backlog.Fases) != 5 {
		t.Fatalf("fases=%+v", backlog.Fases)
	}
	if len(backlog.Microtareas) == 0 {
		t.Fatalf("expected microtareas")
	}
	for index, task := range backlog.Microtareas {
		wantID := fmt.Sprintf("BLG-%03d", index+1)
		if task.ID != wantID {
			t.Fatalf("task id=%q, want %q", task.ID, wantID)
		}
		assertCompleteMicrotaskV0(t, task)
		assertNoForbiddenWriteSetV0(t, task)
	}
	if !containsStringV0(backlog.ContratosRequeridos, "AppSpecV0") {
		t.Fatalf("missing AppSpecV0 contract: %+v", backlog.ContratosRequeridos)
	}
	if backlog.Riesgos == nil {
		t.Fatalf("risks should be a JSON array")
	}
	if backlog.PreguntasAbiertas == nil {
		t.Fatalf("open questions should be a JSON array")
	}
}

func TestGenerarBacklogInicialPropuestoV0RechazaSpecNoValida(t *testing.T) {
	spec := validBacklogSpecV0(t)
	spec.Validation.Estado = "provisional"

	_, issues := GenerarBacklogInicialPropuestoV0(spec)
	if !hasIssueCodeV0(issues, ErrAppSpecInvalida) {
		t.Fatalf("expected %s, got %+v", ErrAppSpecInvalida, issues)
	}
}

func TestGenerarBacklogInicialPropuestoV0IncluyePersistenciaDeployYConectores(t *testing.T) {
	req := validMinimalRequestV0()
	req.Datos = DatosRequestV0{
		DBRequired:         true,
		NecesidadFuncional: "Guardar reservas y cambios de estado.",
	}
	req.Deploy.Target = "contenedor"
	req.Integraciones = []ConnectorRequestV0{{
		Nombre:    "pagos",
		Proposito: "Confirmar cobros externos.",
		Requerido: true,
	}}
	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected spec issues: %+v", issues)
	}
	spec.Connectors.Required[1].Contrato = "PagosGateway v0"

	backlog, issues := GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("unexpected backlog issues: %+v", issues)
	}
	if !containsStringV0(backlog.ContratosRequeridos, "PersistenceRepository v0") {
		t.Fatalf("missing persistence contract: %+v", backlog.ContratosRequeridos)
	}
	if !containsStringV0(backlog.ContratosRequeridos, "DeploymentPlan v0") {
		t.Fatalf("missing deploy contract: %+v", backlog.ContratosRequeridos)
	}
	if !containsStringV0(backlog.ContratosRequeridos, "PagosGateway v0") {
		t.Fatalf("missing connector contract: %+v", backlog.ContratosRequeridos)
	}
	assertHasTaskContractV0(t, backlog.Microtareas, "PersistenceRepository v0")
	assertHasTaskContractV0(t, backlog.Microtareas, "DeploymentPlan v0")
	assertHasTaskContractV0(t, backlog.Microtareas, "PagosGateway v0")
}

func TestGenerarBacklogInicialPropuestoV0PropagaPreguntasAbiertasComoBloqueos(t *testing.T) {
	spec := validBacklogSpecV0(t)
	spec.Scope.PreguntasAbiertas = []string{"Confirmar reglas de aprobacion.", "Confirmar reglas de aprobacion."}

	backlog, issues := GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if got := len(backlog.PreguntasAbiertas); got != 1 {
		t.Fatalf("open questions=%+v, len=%d", backlog.PreguntasAbiertas, got)
	}
	firstTask := backlog.Microtareas[0]
	if !containsStringV0(firstTask.Bloqueos, "Confirmar reglas de aprobacion.") {
		t.Fatalf("first task should block on open question: %+v", firstTask.Bloqueos)
	}
	if !containsStringV0(backlog.Riesgos, "Existen preguntas abiertas que pueden cambiar alcance o contratos.") {
		t.Fatalf("missing open-question risk: %+v", backlog.Riesgos)
	}
}

func validBacklogSpecV0(t *testing.T) AppSpecV0 {
	t.Helper()
	spec, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), time.Date(2026, 5, 4, 12, 30, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("build valid spec: %+v", issues)
	}
	return spec
}

func assertCompleteMicrotaskV0(t *testing.T, task MicrotareaPropuestaV0) {
	t.Helper()
	required := map[string]string{
		"fase":            task.Fase,
		"modulo_sugerido": task.ModuloSugerido,
		"objetivo":        task.Objetivo,
		"contrato":        task.Contrato,
		"validacion":      task.Validacion,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("task %s missing %s: %+v", task.ID, field, task)
		}
	}
	if task.WriteSetPrevisto == nil || len(task.WriteSetPrevisto) == 0 {
		t.Fatalf("task %s missing write-set: %+v", task.ID, task)
	}
	if task.Bloqueos == nil {
		t.Fatalf("task %s should serialize bloqueos as array: %+v", task.ID, task)
	}
}

func assertNoForbiddenWriteSetV0(t *testing.T, task MicrotareaPropuestaV0) {
	t.Helper()
	for _, path := range task.WriteSetPrevisto {
		normalized := strings.ToLower(strings.TrimSpace(path))
		if strings.Contains(normalized, "internal/") {
			t.Fatalf("task %s crosses internal write-set: %+v", task.ID, task.WriteSetPrevisto)
		}
		if strings.Contains(normalized, "postgres") || strings.Contains(normalized, "mysql") || strings.Contains(normalized, "sqlite") {
			t.Fatalf("task %s includes direct DB provider: %+v", task.ID, task.WriteSetPrevisto)
		}
	}
}

func assertHasTaskContractV0(t *testing.T, tasks []MicrotareaPropuestaV0, contract string) {
	t.Helper()
	for _, task := range tasks {
		if task.Contrato == contract {
			assertCompleteMicrotaskV0(t, task)
			return
		}
	}
	t.Fatalf("missing task with contract %q in %+v", contract, tasks)
}

func containsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
