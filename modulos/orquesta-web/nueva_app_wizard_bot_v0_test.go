package orquestaweb

import (
	"strings"
	"testing"
)

func TestWizardBotGroundingEstrictoV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-bot-grounding", "quiero una app para una agenda")
	session, reply := NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{
		SessionRef: session.SessionRef,
		UserText:   "que es CalDAV?",
		Locale:     "es-ES",
	})

	if session.SessionRef == "" {
		t.Fatalf("session ref vacia")
	}
	if len(reply.GroundingRefs) == 0 || !hasStringPrefixV0(reply.GroundingRefs, "wizard-corpus:") {
		t.Fatalf("grounding ausente: %+v", reply.GroundingRefs)
	}
	if !strings.Contains(reply.Say, "CalDAV") || !strings.Contains(reply.Say, "Siguiente pregunta") {
		t.Fatalf("respuesta no usa corpus y reemite turno: %q", reply.Say)
	}

	_, unknown := NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{
		SessionRef: session.SessionRef,
		UserText:   "que es frobnitz cuantico?",
		Locale:     "es-ES",
	})
	if !strings.Contains(unknown.Say, "No lo se") || len(unknown.TurnResult.Questions) == 0 {
		t.Fatalf("desconocido no responde honestamente con turno: %q", unknown.Say)
	}
}

func TestWizardBotSlotFillingRegistraRespuestasV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-bot-slot", "agenda para empresa con dominio Windows y correo")
	session.Form.Nombre = "Agenda Empresa"
	session.Form.Locale = "es-ES"
	session.Form.TipoApp = "web"
	session.Form.Plataformas = []string{"web"}
	session.Form.Descripcion = "flujo diario de agenda para empresa"
	session.Form.Datos.NecesidadFuncional = "gestion_con_persistencia"
	session.Form.Datos.Sensibilidad = "interna"
	session.Form.Deploy.Target = "contenedor"
	session.Form.UsuariosObjetivo = []string{"equipo"}
	session.Form.Agentes.Autonomia = "media"
	session.Form.Restricciones = []string{"online"}
	session.Form.Documentacion.Profundidad = "normal"
	session.Form.Integraciones = []WebNuevaAppConnectorFormV0{{Tipo: "calendar", Nombre: "agenda"}}
	session = refreshWebNuevaAppIntakeSessionV0(session)
	for step := 0; step < 8; step++ {
		before := NewWebNuevaAppWizardTurnResultV0(session)
		if hasWizardQuestionRefV0(before.Questions, "wizard-t2-identidad-corporativa") {
			break
		}
		session, _ = NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{SessionRef: session.SessionRef, UserText: "1", Locale: "es-ES"})
	}
	before := NewWebNuevaAppWizardTurnResultV0(session)
	if !hasWizardQuestionRefV0(before.Questions, "wizard-t2-identidad-corporativa") {
		t.Fatalf("precondicion sin pregunta identidad: %+v", before.Questions)
	}

	session, reply := NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{
		SessionRef: session.SessionRef,
		UserText:   "conectada al AD",
		Locale:     "es-ES",
	})

	if len(reply.FilledAnswers) != 1 ||
		reply.FilledAnswers[0].QuestionRef != "wizard-t2-identidad-corporativa" ||
		reply.FilledAnswers[0].UserChoice != "ldap_bind" {
		t.Fatalf("slot filling identidad inesperado: %+v", reply.FilledAnswers)
	}
	if !hasWizardIntegrationAuthV0(session.Form.Integraciones, "auth", "ldap_bind") {
		t.Fatalf("respuesta no registrada como WizardAnswer normal: %+v", session.Form.Integraciones)
	}
	if hasWizardQuestionRefV0(reply.TurnResult.Questions, "wizard-t2-identidad-corporativa") {
		t.Fatalf("no respeto exclusion/resolucion tras responder identidad: %+v", reply.TurnResult.Questions)
	}
}

func TestWizardBotSinProveedorFuncionaV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-bot-agenda", "quiero una app para una agenda")
	var reply WizardBotReplyV0

	for step := 0; step < 20; step++ {
		turn := NewWebNuevaAppWizardTurnResultV0(session)
		if turn.LaunchReady {
			reply = WizardBotReplyV0{TurnResult: turn}
			break
		}
		if step == 1 {
			session, reply = NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{SessionRef: session.SessionRef, UserText: "que es agenda?", Locale: "es-ES"})
			if len(reply.FilledAnswers) != 0 || len(reply.TurnResult.Questions) == 0 {
				t.Fatalf("consulta RAG consumio turno: %+v", reply)
			}
		}
		session, reply = NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{SessionRef: session.SessionRef, UserText: "1", Locale: "es-ES"})
	}

	final := NewWebNuevaAppWizardTurnResultV0(session)
	if !final.LaunchReady || !final.SpecComplete || final.SpecPreview == nil {
		t.Fatalf("bot determinista no completa agenda: reply=%+v final=%+v session=%+v", reply, final, session.Form)
	}
}

func TestWizardBotNoInventaOpcionesV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-bot-no-inventa", "quiero una app para una agenda")
	_, reply := NewWebNuevaAppWizardBotReplyV0(session, WizardBotTurnV0{SessionRef: session.SessionRef, UserText: "que opciones hay de logs?", Locale: "es-ES"})

	allowed := map[string]bool{}
	for _, question := range reply.TurnResult.Questions {
		for _, option := range question.Options {
			allowed[option.Value] = true
		}
	}
	for _, token := range strings.Fields(reply.Say) {
		clean := strings.Trim(token, " .,:;()")
		if strings.HasPrefix(clean, "inventada_") || strings.HasPrefix(clean, "fake_") {
			t.Fatalf("opcion inventada en say: %q allowed=%+v", reply.Say, allowed)
		}
	}
	for _, answer := range reply.FilledAnswers {
		if !allowed[answer.UserChoice] {
			t.Fatalf("filled answer fuera de opciones emitidas: %+v allowed=%+v", answer, allowed)
		}
	}
}

func hasStringPrefixV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasWizardIntegrationAuthV0(values []WebNuevaAppConnectorFormV0, tipo string, auth string) bool {
	for _, value := range values {
		if value.Tipo == tipo && value.Auth == auth {
			return true
		}
	}
	return false
}
