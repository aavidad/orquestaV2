package orquestadirector

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
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
	req := orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "req-dir-001",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Panel de reservas",
		Objetivo:      "Gestionar solicitudes de reserva.",
		TipoApp:       "web",
	}
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("build app spec: %+v", issues)
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("build backlog: %+v", issues)
	}
	return BootstrapProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: "idem-dir-001",
		AppSpec:        spec,
		Backlog:        backlog,
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T12:10:00Z",
		RequestID:      req.RequestID,
		CorrelationID:  "corr-dir-001",
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
