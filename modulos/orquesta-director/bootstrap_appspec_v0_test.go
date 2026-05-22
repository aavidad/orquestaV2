package orquestadirector

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBootstrapProyectoDesdeAppSpecV0GeneraRegistroYRunStarted(t *testing.T) {
	cmd := validBootstrapCommandV0(t)

	result, err := BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("BootstrapProyectoDesdeAppSpecV0: %v", err)
	}

	if result.RegistroAceptado.RegistroID == "" {
		t.Fatalf("registro aceptado requerido")
	}
	if result.StartRunCommand.CommandType != orquestacoreworkflow.OrchestrationCommandStartRunV0 {
		t.Fatalf("command_type=%q", result.StartRunCommand.CommandType)
	}
	if len(result.WorkflowResult.Events) != 1 {
		t.Fatalf("events=%d, want 1", len(result.WorkflowResult.Events))
	}
	if result.WorkflowResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("event_type=%q", result.WorkflowResult.Events[0].EventType)
	}
	if len(result.WorkflowResult.Outbox) != 0 {
		t.Fatalf("outbox should stay empty: %+v", result.WorkflowResult.Outbox)
	}
	if result.ProjectRef == "" || result.AppSpecRef == "" {
		t.Fatalf("refs required: %+v", result)
	}
}

func TestBootstrapProyectoDesdeAppSpecV0RechazaIdempotencyKeyVacia(t *testing.T) {
	cmd := validBootstrapCommandV0(t)
	cmd.IdempotencyKey = " "

	_, err := BootstrapProyectoDesdeAppSpecV0(cmd)

	var bootstrapErr BootstrapProyectoDesdeAppSpecErrorV0
	if !errors.As(err, &bootstrapErr) {
		t.Fatalf("error type=%T", err)
	}
	if bootstrapErr.Code != ErrDirectorBootstrapInvalidoV0 {
		t.Fatalf("code=%q", bootstrapErr.Code)
	}
}

func TestBootstrapProyectoDesdeAppSpecV0SalidaCompactaSinDetallesProhibidos(t *testing.T) {
	cmd := validBootstrapCommandV0(t)

	result, err := BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("BootstrapProyectoDesdeAppSpecV0: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	serialized := strings.ToLower(string(data))
	for _, forbidden := range forbiddenBootstrapFragmentsV0() {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized result contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
	for _, leaked := range []string{cmd.AppSpec.App.Nombre, cmd.Backlog.Microtareas[0].Objetivo} {
		if strings.Contains(serialized, strings.ToLower(leaked)) {
			t.Fatalf("serialized result leaks full input detail %q: %s", leaked, serialized)
		}
	}
	if strings.Contains(serialized, strings.ToLower(cmd.AppSpec.SpecID)) {
		t.Fatalf("serialized result leaks raw app spec id %q: %s", cmd.AppSpec.SpecID, serialized)
	}
}

func TestBootstrapProyectoDesdeAppSpecV0RefsDeterministasDesdeCore(t *testing.T) {
	cmd := validBootstrapCommandV0(t)

	first, err := BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.ProjectRef != second.ProjectRef || first.AppSpecRef != second.AppSpecRef {
		t.Fatalf("refs not deterministic: %+v != %+v", first, second)
	}
	if first.ProjectRef != first.RegistroAceptado.ProjectRef {
		t.Fatalf("project_ref mismatch: %q != %q", first.ProjectRef, first.RegistroAceptado.ProjectRef)
	}
	if first.AppSpecRef != first.RegistroAceptado.AppSpecRef {
		t.Fatalf("app_spec_ref mismatch: %q != %q", first.AppSpecRef, first.RegistroAceptado.AppSpecRef)
	}
}

func validBootstrapCommandV0(t *testing.T) BootstrapProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	specID := "appspec-panel-reservas-001"
	return BootstrapProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: "idem-dir-001",
		AppSpec:        neutralRegistrarAppSpecV0(specID, "req-dir-001", "Panel de reservas", "2026-05-04T12:00:00Z"),
		Backlog:        neutralRegistrarBacklogV0(specID, "reservas"),
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T12:10:00Z",
		RequestID:      "req-dir-001",
		CorrelationID:  "corr-dir-001",
	}
}

func neutralRegistrarAppSpecV0(specID string, requestID string, name string, createdAt string) orquestacore.RegistrarAppSpecV0 {
	return orquestacore.RegistrarAppSpecV0{
		SchemaVersion: orquestacore.RegistrarAppSpecSchemaV0,
		SpecID:        specID,
		RequestID:     requestID,
		CreatedAt:     createdAt,
		App: orquestacore.RegistrarAppInfoV0{
			Nombre: name,
		},
		Validation: orquestacore.RegistrarValidationSummaryV0{
			Estado: "valida",
		},
	}
}

func neutralRegistrarBacklogV0(specID string, prefix string) orquestacore.RegistrarBacklogInicialV0 {
	return orquestacore.RegistrarBacklogInicialV0{
		SchemaVersion: orquestacore.RegistrarBacklogInicialSchemaV0,
		SpecID:        specID,
		Fases: []orquestacore.RegistrarFaseInicialV0{
			{ID: "descubrimiento", Nombre: "Descubrimiento", Objetivo: "Alcance inicial claro.", Orden: 1},
			{ID: "diseno", Nombre: "Diseno", Objetivo: "Flujos y datos definidos.", Orden: 2},
			{ID: "planificacion", Nombre: "Planificacion", Objetivo: "Microtareas preparadas.", Orden: 3},
			{ID: "programacion", Nombre: "Programacion", Objetivo: "Implementacion verificable.", Orden: 4},
			{ID: "cierre", Nombre: "Cierre", Objetivo: "Validacion y entrega cerradas.", Orden: 5},
		},
		Microtareas: []orquestacore.RegistrarMicrotareaV0{
			{
				ID:               prefix + "-task-001",
				Fase:             "descubrimiento",
				ModuloSugerido:   "docs",
				Objetivo:         "Definir alcance funcional inicial.",
				WriteSetPrevisto: []string{"docs/alcance.md"},
				Contrato:         "FunctionContractV0",
				Validacion:       "Alcance revisado.",
			},
			{
				ID:               prefix + "-task-002",
				Fase:             "diseno",
				ModuloSugerido:   "app",
				Objetivo:         "Disenar flujo principal.",
				WriteSetPrevisto: []string{"app/flujo.md"},
				Contrato:         "FunctionContractV0",
				Validacion:       "Flujo principal revisado.",
			},
			{
				ID:               prefix + "-task-003",
				Fase:             "planificacion",
				ModuloSugerido:   "app",
				Objetivo:         "Preparar microtareas ejecutables.",
				WriteSetPrevisto: []string{"app/tasks.md"},
				Contrato:         "FunctionContractV0",
				Validacion:       "Microtareas verificables.",
			},
			{
				ID:               prefix + "-task-004",
				Fase:             "programacion",
				ModuloSugerido:   "app",
				Objetivo:         "Implementar comportamiento principal.",
				WriteSetPrevisto: []string{"app/main.go"},
				Contrato:         "FunctionContractV0",
				Validacion:       "Pruebas locales verdes.",
			},
			{
				ID:               prefix + "-task-005",
				Fase:             "cierre",
				ModuloSugerido:   "docs",
				Objetivo:         "Cerrar evidencia de entrega.",
				WriteSetPrevisto: []string{"docs/cierre.md"},
				Contrato:         "FunctionContractV0",
				Validacion:       "Evidencia registrada.",
			},
		},
		ContratosRequeridos: []string{
			"descubrimiento.contract.v0",
			"diseno.contract.v0",
			"planificacion.contract.v0",
			"programacion.contract.v0",
			"cierre.contract.v0",
		},
		Riesgos:           []string{"Dependencias externas pendientes de adaptar."},
		PreguntasAbiertas: []string{"Confirmar prioridad de entrega."},
	}
}

func forbiddenBootstrapFragmentsV0() []string {
	return []string{
		"db",
		"sql",
		"runtime",
		"provider",
		"proveedor",
		"oauth",
		"home",
		"secret",
		"token",
		"password",
	}
}
