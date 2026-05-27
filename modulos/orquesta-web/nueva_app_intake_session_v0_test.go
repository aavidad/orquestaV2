package orquestaweb

import (
	"encoding/json"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppIntakeSessionV0CreaSesionDesdeIdeaYPideCamposCriticos(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0(" session-1 ", " es ", " Agenda ", " Coordinar ensayos ")

	if session.SchemaVersion != WebNuevaAppIntakeSessionSchemaV0 {
		t.Fatalf("schema=%q", session.SchemaVersion)
	}
	if session.SessionID != "session-1" || session.Locale != "es" {
		t.Fatalf("refs normalizadas inesperadas: %+v", session)
	}
	if session.Estado != WebNuevaAppIntakeEstadoRequiereDatos {
		t.Fatalf("estado=%q", session.Estado)
	}
	if session.AppSpecPartial.Nombre != "Agenda" ||
		session.AppSpecPartial.Objetivo != "Coordinar ensayos" ||
		session.AppSpecPartial.TipoApp != "" {
		t.Fatalf("app spec parcial inesperado: %+v", session.AppSpecPartial)
	}
	if len(session.PendingQuestions) != 1 || session.PendingQuestions[0] != "tipo_app" {
		t.Fatalf("preguntas=%+v", session.PendingQuestions)
	}
	if len(session.Questions) != 1 ||
		session.Questions[0].LabelKey != "nueva_app.campo.tipo_app" ||
		!session.Questions[0].Required {
		t.Fatalf("questions i18n=%+v", session.Questions)
	}
	if session.AppSpecPartial.ExecutionMode != orquestafactory.ExecutionModeNormalV0 {
		t.Fatalf("execution_mode=%q", session.AppSpecPartial.ExecutionMode)
	}
}

func TestApplyWebNuevaAppIntakeAnswerV0RegistraDecisionYDejaDraftListo(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-1", "es", "Agenda", "Coordinar ensayos")

	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{
		Field: " tipo_app ",
		Value: " web ",
	})

	if session.Estado != WebNuevaAppIntakeEstadoLista {
		t.Fatalf("estado=%q", session.Estado)
	}
	if session.AppSpecPartial.TipoApp != "web" {
		t.Fatalf("tipo_app=%q", session.AppSpecPartial.TipoApp)
	}
	if len(session.PendingQuestions) != 0 {
		t.Fatalf("preguntas=%+v", session.PendingQuestions)
	}
	if len(session.Decisions) != 1 || session.Decisions[0].Field != "tipo_app" || session.Decisions[0].Value != "web" {
		t.Fatalf("decisiones=%+v", session.Decisions)
	}
	if sectionStatusV0(session.Sections, "identidad") != "complete" {
		t.Fatalf("sections=%+v", session.Sections)
	}
	if !session.Handoff.Ready ||
		session.Handoff.TargetTool != WebNuevaAppDirectorTargetToolV0 ||
		session.Handoff.TargetPath != ArrancarDirectorAppEndpointV0 ||
		session.Handoff.FallbackTool != WebNuevaAppFallbackTargetToolV0 {
		t.Fatalf("handoff=%+v", session.Handoff)
	}
	if session.Handoff.RequestRef != "session-1" ||
		len(session.Handoff.ContextRefs) != 2 ||
		!hasIntakeContextValueV0(session.Handoff.ContextSummary, "tipo_app", "web") {
		t.Fatalf("handoff refs/context=%+v", session.Handoff)
	}
}

func TestApplyWebNuevaAppIntakeAnswerV0NoValidaEnumsDeFactory(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-invalid", "es", "Agenda", "Coordinar ensayos")

	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{
		Field: "tipo_app",
		Value: "mainframe",
	})

	if session.Estado != WebNuevaAppIntakeEstadoLista {
		t.Fatalf("la sesion solo debe quedar lista para validar: %q", session.Estado)
	}
	if session.AppSpecPartial.TipoApp != "mainframe" {
		t.Fatalf("la web no debe corregir reglas de factory: %+v", session.AppSpecPartial)
	}
}

func TestWebNuevaAppIntakeSessionV0SnapshotCompactoSinDumpsInternos(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-1", "es", "Agenda", "Coordinar ensayos")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{
		Field:  "plataformas",
		Values: []string{" web ", "web"},
	})

	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	got := string(raw)
	forbidden := []string{"ValidationIssue", "AppSpecRequestV0", "runtime_provider", "agent_id"}
	for _, value := range forbidden {
		if strings.Contains(got, value) {
			t.Fatalf("snapshot expone detalle interno %q: %s", value, got)
		}
	}
	hasPendingQuestions := strings.Contains(got, `"pending_questions"`) || strings.Contains(got, `"preguntas_pendientes"`)
	if !hasPendingQuestions || !strings.Contains(got, `"field_index"`) {
		t.Fatalf("snapshot no conserva campos de operador: %s", got)
	}
}

func sectionStatusV0(sections []WebNuevaAppIntakeSectionV0, id string) string {
	for _, section := range sections {
		if section.ID == id {
			return section.Status
		}
	}
	return ""
}

func hasIntakeContextValueV0(values []WebNuevaAppIntakeContextV0, key string, value string) bool {
	for _, item := range values {
		if item.Key == key && item.Value == value {
			return true
		}
	}
	return false
}
