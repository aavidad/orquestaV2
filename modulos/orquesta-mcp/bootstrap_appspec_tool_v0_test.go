package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestMCPBootstrapDescriptorYResourceV0Compactos(t *testing.T) {
	descriptor := MCPBootstrapToolDescriptorV0Value()
	resource := NewMCPBootstrapResourceV0()

	if descriptor.Name != MCPBootstrapToolNameV0 || descriptor.ResourceURI != MCPBootstrapResourceURIV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	if resource.Owner != "orquesta-director" || resource.ToolName != MCPBootstrapToolNameV0 {
		t.Fatalf("resource identidad: %+v", resource)
	}
	if !strings.Contains(descriptor.InputSchema, "BootstrapProyectoDesdeAppSpecCommandV0") {
		t.Fatalf("descriptor no referencia command publico: %+v", descriptor)
	}
	if !strings.Contains(resource.Output.Shape, "workflow compacto") {
		t.Fatalf("resource no describe salida compacta: %+v", resource.Output)
	}

	assertBootstrapJSONSaneadoV0(t, descriptor, 900)
	assertBootstrapJSONSaneadoV0(t, resource, 2200)
}

func TestExecuteMCPBootstrapToolV0InvocaDirectorYDevuelveJSONCompacto(t *testing.T) {
	input := MCPBootstrapToolInputV0{
		RequestID:     "req-envelope",
		CorrelationID: "corr-envelope",
		Command:       validMCPBootstrapCommandV0(t),
	}
	input.Command.RequestID = ""
	input.Command.CorrelationID = ""
	input.Command.RequestedBy = ""

	result := ExecuteMCPBootstrapToolV0(input)

	if result.Estado != MCPBootstrapEstadoOKV0 {
		t.Fatalf("estado=%q errores=%+v", result.Estado, result.Errores)
	}
	if result.RequestID != "req-envelope" || result.CorrelationID != "corr-envelope" {
		t.Fatalf("envelope no normalizado: %+v", result)
	}
	if result.ProjectRef == "" || result.AppSpecRef == "" || result.Registro.RegistroID == "" {
		t.Fatalf("refs requeridas: %+v", result)
	}
	if result.Workflow.CommandType != "StartRun" || result.Workflow.Events != 1 || result.Workflow.Outbox != 0 {
		t.Fatalf("workflow compacto inesperado: %+v", result.Workflow)
	}
	if len(result.Workflow.EventTypes) != 1 || result.Workflow.EventTypes[0] != "RunStarted" {
		t.Fatalf("event types=%+v", result.Workflow.EventTypes)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	serialized := strings.ToLower(string(payload))
	for _, leaked := range []string{
		input.Command.AppSpec.App.Nombre,
		input.Command.AppSpec.SpecID,
		input.Command.Backlog.Microtareas[0].Objetivo,
		"payload\":",
		"app_spec\":",
		"backlog\":",
	} {
		if strings.Contains(serialized, strings.ToLower(leaked)) {
			t.Fatalf("resultado filtra detalle %q: %s", leaked, serialized)
		}
	}
}

func TestExecuteMCPBootstrapToolV0DevuelveErrorPublicoDirector(t *testing.T) {
	input := MCPBootstrapToolInputV0{
		RequestID:     "req-bad",
		CorrelationID: "corr-bad",
		Command:       validMCPBootstrapCommandV0(t),
	}
	input.Command.IdempotencyKey = " "

	result := ExecuteMCPBootstrapToolV0(input)

	if result.Estado != MCPBootstrapEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("error esperado: %+v", result)
	}
	if result.Errores[0].Code != orquestadirector.ErrDirectorBootstrapInvalidoV0 ||
		result.Errores[0].Field != "idempotency_key" ||
		result.Errores[0].CorrelationID != "corr-bad" {
		t.Fatalf("error publico incorrecto: %+v", result.Errores[0])
	}
	if result.ProjectRef != "" || result.Workflow.Events != 0 || result.Registro.RegistroID != "" {
		t.Fatalf("error no debe inventar salida ok: %+v", result)
	}
}

func TestNewMCPBootstrapOKResultV0TrimDedupeWorkflow(t *testing.T) {
	directorResult, err := orquestadirector.BootstrapProyectoDesdeAppSpecV0(validMCPBootstrapCommandV0(t))
	if err != nil {
		t.Fatalf("bootstrap director: %v", err)
	}
	directorResult.RegistroAceptado.Warnings = []string{" revisar contrato ", "revisar contrato", ""}
	directorResult.WorkflowResult.Events = append(directorResult.WorkflowResult.Events, directorResult.WorkflowResult.Events[0])

	result := NewMCPBootstrapOKResultV0(directorResult, " req ", " corr ")

	if len(result.Registro.Warnings) != 1 || result.Registro.Warnings[0] != "revisar contrato" {
		t.Fatalf("warnings no compactas: %+v", result.Registro.Warnings)
	}
	if len(result.Workflow.EventTypes) != 1 || result.Workflow.EventTypes[0] != "RunStarted" {
		t.Fatalf("event types no compactos: %+v", result.Workflow.EventTypes)
	}
}

func validMCPBootstrapCommandV0(t *testing.T) orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	req := orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "req-bootstrap",
		Source:        "orquesta-mcp",
		Locale:        "es-ES",
		Nombre:        "Panel de reservas",
		Objetivo:      "Gestionar solicitudes de reserva.",
		TipoApp:       "web",
	}
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("app spec issues: %+v", issues)
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("backlog issues: %+v", issues)
	}
	return orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: "idem-bootstrap",
		AppSpec:        registrarAppSpecFromFactoryForMCPBootstrapTestV0(spec),
		Backlog:        registrarBacklogFromFactoryForMCPBootstrapTestV0(backlog),
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T12:10:00Z",
		RequestID:      req.RequestID,
		CorrelationID:  "corr-bootstrap",
	}
}

func registrarAppSpecFromFactoryForMCPBootstrapTestV0(spec orquestafactory.AppSpecV0) orquestacore.RegistrarAppSpecV0 {
	return orquestacore.RegistrarAppSpecV0{
		SchemaVersion: spec.SchemaVersion,
		SpecID:        spec.SpecID,
		RequestID:     spec.RequestID,
		CreatedAt:     spec.CreatedAt,
		App: orquestacore.RegistrarAppInfoV0{
			Nombre: spec.App.Nombre,
		},
		Validation: orquestacore.RegistrarValidationSummaryV0{
			Estado: spec.Validation.Estado,
		},
	}
}

func registrarBacklogFromFactoryForMCPBootstrapTestV0(backlog orquestafactory.BacklogInicialPropuestoV0) orquestacore.RegistrarBacklogInicialV0 {
	return orquestacore.RegistrarBacklogInicialV0{
		SchemaVersion:       backlog.SchemaVersion,
		SpecID:              backlog.SpecID,
		Fases:               registrarFasesFromFactoryForMCPBootstrapTestV0(backlog.Fases),
		Microtareas:         registrarMicrotareasFromFactoryForMCPBootstrapTestV0(backlog.Microtareas),
		ContratosRequeridos: append([]string(nil), backlog.ContratosRequeridos...),
		Riesgos:             append([]string(nil), backlog.Riesgos...),
		PreguntasAbiertas:   append([]string(nil), backlog.PreguntasAbiertas...),
	}
}

func registrarFasesFromFactoryForMCPBootstrapTestV0(fases []orquestafactory.FaseInicialV0) []orquestacore.RegistrarFaseInicialV0 {
	result := make([]orquestacore.RegistrarFaseInicialV0, 0, len(fases))
	for _, fase := range fases {
		result = append(result, orquestacore.RegistrarFaseInicialV0{
			ID:       fase.ID,
			Nombre:   fase.Nombre,
			Objetivo: fase.Objetivo,
			Orden:    fase.Orden,
		})
	}
	return result
}

func registrarMicrotareasFromFactoryForMCPBootstrapTestV0(tasks []orquestafactory.MicrotareaPropuestaV0) []orquestacore.RegistrarMicrotareaV0 {
	result := make([]orquestacore.RegistrarMicrotareaV0, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, orquestacore.RegistrarMicrotareaV0{
			ID:               task.ID,
			Fase:             task.Fase,
			ModuloSugerido:   task.ModuloSugerido,
			Objetivo:         task.Objetivo,
			WriteSetPrevisto: append([]string(nil), task.WriteSetPrevisto...),
			Contrato:         task.Contrato,
			Validacion:       task.Validacion,
			Bloqueos:         append([]string(nil), task.Bloqueos...),
		})
	}
	return result
}

func assertBootstrapJSONSaneadoV0(t *testing.T, value any, maxBytes int) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := strings.ToLower(string(payload))
	if len(text) > maxBytes {
		t.Fatalf("payload demasiado largo: %d bytes: %s", len(text), text)
	}
	for _, forbidden := range []string{"sql", "dsn", "oauth", "token", "password", "home", "app_spec_completo", "backlog_completo"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("payload contiene fragmento prohibido %q: %s", forbidden, text)
		}
	}
}
