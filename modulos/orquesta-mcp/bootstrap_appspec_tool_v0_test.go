package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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
		AppSpec:        spec,
		Backlog:        backlog,
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T12:10:00Z",
		RequestID:      req.RequestID,
		CorrelationID:  "corr-bootstrap",
	}
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
