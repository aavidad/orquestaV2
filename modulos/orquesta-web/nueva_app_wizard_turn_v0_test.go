package orquestaweb

import (
	"os"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWizardAgendaDesdeSoloObjetivoV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-wizard-agenda", "", "", "")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{
		Field: "objetivo",
		Value: "quiero una app para una agenda",
	})

	allQuestions := webNuevaAppWizardAllGapQuestionsV0(session.Form)

	if !hasWizardQuestionRefV0(allQuestions, "wizard-r2-plataformas") ||
		!hasWizardQuestionOptionV0(allQuestions, "wizard-r2-plataformas", "web_mobile") ||
		!hasWizardQuestionRefV0(allQuestions, "wizard-r1-uso-personal-compartido") ||
		!hasWizardQuestionOptionV0(allQuestions, "wizard-r1-uso-personal-compartido", "compartir_auth_simple") ||
		!hasWizardQuestionRefV0(allQuestions, "wizard-r3-integracion-agenda") ||
		!hasWizardQuestionOptionV0(allQuestions, "wizard-r3-integracion-agenda", "calendar_google_workspace") ||
		!hasWizardQuestionOptionV0(allQuestions, "wizard-r3-integracion-agenda", "calendar_microsoft_365") ||
		!hasWizardQuestionOptionV0(allQuestions, "wizard-r3-integracion-agenda", "calendar_caldav") {
		t.Fatalf("preguntas agenda incompletas: %+v", allQuestions)
	}

	session, turns := AcceptWebNuevaAppWizardRecommendationsV0(session, 6)

	if len(turns) == 0 || len(turns) > 6 {
		t.Fatalf("turns=%d", len(turns))
	}
	final := NewWebNuevaAppWizardTurnResultV0(session)
	if !final.LaunchReady || !final.SpecComplete || final.SpecPreview == nil {
		t.Fatalf("final no listo: %+v", final)
	}
	if issues := orquestafactory.ValidateAppSpecRequestV0(*final.SpecPreview); len(issues) != 0 {
		t.Fatalf("spec final invalida: %+v\n%+v", issues, *final.SpecPreview)
	}
	if session.Form.TipoApp != "web" ||
		!stringSliceHasV0(session.Form.Plataformas, "web") ||
		!stringSliceHasV0(session.Form.Plataformas, "mobile") ||
		len(session.Form.Integraciones) == 0 ||
		session.Form.Integraciones[0].Tipo != "calendar" ||
		session.Form.Integraciones[0].Auth == "" ||
		session.Form.Integraciones[0].Criticidad == "" {
		t.Fatalf("session agenda incompleta: %+v", session.Form)
	}
	if !stringSliceHasV0(final.SpecPreview.PreferenciasTecnicas.Preferencias, "hexagonal_puertos_adaptadores") ||
		!stringSliceHasV0(final.SpecPreview.PreferenciasTecnicas.Preferencias, "logging_estructurado") ||
		final.SpecPreview.I18N.Enabled == nil ||
		!*final.SpecPreview.I18N.Enabled ||
		final.SpecPreview.Calidad.Pruebas != "alta" ||
		final.SpecPreview.Calidad.Observabilidad == nil ||
		!*final.SpecPreview.Calidad.Observabilidad ||
		final.SpecPreview.Documentacion.Desarrollo == nil ||
		!*final.SpecPreview.Documentacion.Desarrollo {
		t.Fatalf("defaults de ingenieria no aplicados: %+v", *final.SpecPreview)
	}
}

func TestWizardHuecosCruzadosDetectaDatosIntegracionMovilYDeployV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-wizard-cross", "es-ES", "App mixta", "app movil con datos externos")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "mobile"})
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "datos.db_required", Value: "true"})
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "integraciones.0.tipo", Value: "api"})
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "deploy.target", Value: "desktop"})

	allQuestions := webNuevaAppWizardAllGapQuestionsV0(session.Form)

	for _, ref := range []string{
		"wizard-r4-storage-db",
		"wizard-r5-integracion-gobierno",
		"wizard-r6-movil-plataformas",
		"wizard-r7-deploy-compatible",
	} {
		if !hasWizardQuestionRefV0(allQuestions, ref) {
			t.Fatalf("no detecta %s en %+v", ref, allQuestions)
		}
	}
}

func TestWizardPackDominioPorKeywordV0(t *testing.T) {
	tests := []struct {
		name        string
		objective   string
		questionRef string
		optionValue string
	}{
		{
			name:        "tienda",
			objective:   "quiero una tienda con catalogo, carrito y pagos",
			questionRef: "wizard-r3-integracion-tienda",
			optionValue: "ecommerce_capability",
		},
		{
			name:        "finanzas",
			objective:   "quiero controlar gastos, presupuestos y extractos OFX",
			questionRef: "wizard-r3-integracion-finanzas",
			optionValue: "finance_capability",
		},
		{
			name:        "reservas",
			objective:   "quiero una app de reservas y turnos con disponibilidad",
			questionRef: "wizard-r3-integracion-reservas",
			optionValue: "booking_capability",
		},
		{
			name:        "iot",
			objective:   "quiero domotica con sensores MQTT y panel de telemetria",
			questionRef: "wizard-r3-integracion-iot",
			optionValue: "iot_capability",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := wizardSessionWithObjectiveV0("session-wizard-domain-"+tt.name, tt.objective)
			questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
			if !hasWizardQuestionRefV0(questions, tt.questionRef) ||
				!hasWizardQuestionOptionV0(questions, tt.questionRef, tt.optionValue) {
				t.Fatalf("pack dominio no activado: ref=%s option=%s questions=%+v", tt.questionRef, tt.optionValue, questions)
			}
		})
	}
}

func TestWizardPacksCombinadosSinDuplicadosV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-domain-combined", "quiero una tienda con reservas, turnos, catalogo y pagos")
	questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
	if !hasWizardQuestionRefV0(questions, "wizard-r3-integracion-tienda") ||
		!hasWizardQuestionRefV0(questions, "wizard-r3-integracion-reservas") {
		t.Fatalf("packs combinados no activados: %+v", questions)
	}
	fields := map[string]bool{}
	for _, question := range questions {
		if !strings.HasPrefix(question.QuestionRef, "wizard-r3-integracion-") {
			continue
		}
		if fields[question.Field] {
			t.Fatalf("pack dominio duplico field=%s en %+v", question.Field, questions)
		}
		fields[question.Field] = true
	}
}

func TestWizardSinDominioPreguntaAbiertaV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-domain-open", "quiero una herramienta para organizar ideas raras")
	questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
	if !hasWizardQuestionRefV0(questions, "wizard-r3-dominio-abierto") ||
		!hasWizardQuestionOptionV0(questions, "wizard-r3-dominio-abierto", "describir_flujo_diario") {
		t.Fatalf("sin dominio no pregunta abierto: %+v", questions)
	}
}

func TestWizardPackDominioAplicaDecisionComoConectorV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-domain-decision", "quiero controlar gastos y presupuestos")
	session, result := ApplyWebNuevaAppWizardAnswersV0(session, []WizardAnswerV0{{
		QuestionRef: "wizard-r3-integracion-finanzas",
		UserChoice:  "finance_capability",
	}})
	if len(result.Decisions) == 0 ||
		len(session.Form.Integraciones) == 0 ||
		session.Form.Integraciones[0].Tipo != "finance" ||
		session.Form.Integraciones[0].Criticidad != "alta" ||
		!session.Form.Integraciones[0].Requerido {
		t.Fatalf("pack dominio no materializa conector gobernado: session=%+v result=%+v", session.Form, result)
	}
}

func TestWizardConservaRecomendacionVisibleSiUsuarioEligeOtraOpcionV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-wizard-contrast", "es-ES", "", "quiero una app para una agenda")
	questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
	if !hasWizardQuestionOptionV0(questions, "wizard-r2-plataformas", "web_mobile") {
		t.Fatalf("pregunta plataforma sin recomendacion esperada: %+v", questions)
	}

	session, result := ApplyWebNuevaAppWizardAnswersV0(session, []WizardAnswerV0{{
		QuestionRef: "wizard-r2-plataformas",
		UserChoice:  "web",
	}})

	if len(result.Contrasts) != 1 ||
		result.Contrasts[0].QuestionRef != "wizard-r2-plataformas" ||
		result.Contrasts[0].UserChoice != "web" ||
		result.Contrasts[0].Recommended != "web_mobile" ||
		result.Contrasts[0].RationaleKey == "" {
		t.Fatalf("contraste no conserva recomendacion: %+v", result.Contrasts)
	}
	if !stringSliceHasV0(session.Form.Plataformas, "web") ||
		stringSliceHasV0(session.Form.Plataformas, "mobile") {
		t.Fatalf("eleccion del usuario no debe sobreescribirse: %+v", session.Form.Plataformas)
	}
}

func TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0(t *testing.T) {
	response := NewWebNuevaAppIntakeGuidedResponseV0(WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-wizard-endpoint",
		Need:      "quiero una app para una agenda",
	})

	if response.Wizard.SchemaVersion != WebNuevaAppWizardTurnSchemaV0 ||
		len(response.Wizard.Questions) == 0 ||
		len(response.Wizard.EngineeringDefaults) == 0 ||
		response.Wizard.SpecPreview == nil {
		t.Fatalf("wizard no incluido en respuesta: %+v", response.Wizard)
	}
	if !hasWizardQuestionRefV0(response.Wizard.Questions, "wizard-q-locale") &&
		!hasWizardQuestionRefV0(response.Wizard.Questions, "wizard-q-tipo-app") {
		t.Fatalf("primer turno wizard inesperado: %+v", response.Wizard.Questions)
	}
}

func TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0(t *testing.T) {
	response := NewWebNuevaAppIntakeGuidedResponseV0(WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-wizard-api-answer",
		Need:      "quiero una app para una agenda",
		WizardAnswers: []WizardAnswerV0{{
			QuestionRef: "wizard-r2-plataformas",
			UserChoice:  "web",
		}},
	})

	if !stringSliceHasV0(response.Session.Form.Plataformas, "web") ||
		stringSliceHasV0(response.Session.Form.Plataformas, "mobile") {
		t.Fatalf("wizard_answers no aplico plataformas: %+v", response.Session.Form.Plataformas)
	}
	if len(response.Wizard.Decisions) == 0 {
		t.Fatalf("wizard_answers no devolvio decisiones aplicadas: %+v", response.Wizard)
	}
	if len(response.Wizard.Contrasts) != 1 ||
		response.Wizard.Contrasts[0].QuestionRef != "wizard-r2-plataformas" ||
		response.Wizard.Contrasts[0].Recommended != "web_mobile" {
		t.Fatalf("wizard_answers no conserva contraste: %+v", response.Wizard.Contrasts)
	}
}

func TestWizardPreguntasRicasTienenCatalogoI18NV0(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()
	sessions := []WebNuevaAppIntakeSessionV0{
		wizardSessionWithObjectiveV0("session-i18n-agenda", "quiero una app para una agenda"),
		wizardSessionWithObjectiveV0("session-i18n-tienda", "quiero una tienda con pagos y checkout"),
		wizardSessionWithObjectiveV0("session-i18n-mapas", "quiero una app con mapas y ubicacion cercana"),
		wizardSessionWithObjectiveV0("session-i18n-compartida", "quiero una agenda compartida para clientes"),
	}

	cross := NewWebNuevaAppIntakeSessionV0("session-i18n-cross", "es-ES", "App mixta", "app movil con datos externos")
	cross = cross.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "mobile"})
	cross = cross.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "datos.db_required", Value: "true"})
	cross = cross.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "integraciones.0.tipo", Value: "api"})
	cross = cross.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "deploy.target", Value: "desktop"})
	sessions = append(sessions, cross)

	for _, session := range sessions {
		for _, question := range webNuevaAppWizardAllGapQuestionsV0(session.Form) {
			requireWizardI18nKeyV0(t, catalog, question.PromptKey)
			requireWizardI18nKeyV0(t, catalog, question.WhyKey)
			for _, option := range question.Options {
				requireWizardI18nKeyV0(t, catalog, option.LabelKey)
				if option.RationaleKey != "" {
					requireWizardI18nKeyV0(t, catalog, option.RationaleKey)
				}
			}
		}
	}
	for _, defaultValue := range WebNuevaAppWizardEngineeringDefaultsV0() {
		requireWizardI18nKeyV0(t, catalog, defaultValue.WhyKey)
	}
}

func TestWizardTodaOpcionTieneAyudaV0(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()
	for _, question := range wizardAllRegisteredQuestionsForHelpTestV0() {
		requireWizardI18nKeyV0(t, catalog, question.PromptKey)
		requireWizardI18nKeyV0(t, catalog, question.WhyKey)
		requireWizardI18nKeyV0(t, catalog, question.HelpKey)
		for _, option := range question.Options {
			requireWizardI18nKeyV0(t, catalog, option.LabelKey)
			requireWizardI18nKeyV0(t, catalog, option.HelpKey)
			if option.ExampleKey != "" {
				requireWizardI18nKeyV0(t, catalog, option.ExampleKey)
			}
			if option.RationaleKey != "" {
				requireWizardI18nKeyV0(t, catalog, option.RationaleKey)
			}
		}
	}
	for _, defaultValue := range WebNuevaAppWizardEngineeringDefaultsV0() {
		requireWizardI18nKeyV0(t, catalog, defaultValue.WhyKey)
		requireWizardI18nKeyV0(t, catalog, defaultValue.HelpKey)
		requireWizardI18nKeyV0(t, catalog, defaultValue.ExampleKey)
	}
}

func TestWizardPreguntaQueEsRespondeYNoAvanzaV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-comprehension", "quiero una app para una agenda")
	before := NewWebNuevaAppWizardTurnResultV0(session)
	if len(before.Questions) == 0 {
		t.Fatalf("sesion sin preguntas para consulta")
	}
	nextSession, result := ApplyWebNuevaAppWizardAnswersV0(session, []WizardAnswerV0{{
		ComprehensionQuery: "que es CalDAV?",
	}})
	if len(nextSession.Decisions) != len(session.Decisions) {
		t.Fatalf("consulta de comprension no debe consumir turno: before=%d after=%d", len(session.Decisions), len(nextSession.Decisions))
	}
	if result.GlossaryResponse == nil ||
		result.GlossaryResponse.HelpKey != "nueva_app.wizard.help.option.integracion.calendar.caldav" ||
		result.GlossaryResponse.ExampleKey == "" {
		t.Fatalf("respuesta de glosario inesperada: %+v", result.GlossaryResponse)
	}
	if !hasWizardQuestionRefV0(result.Questions, before.Questions[0].QuestionRef) {
		t.Fatalf("la consulta debe reemitir la misma pregunta: before=%+v after=%+v", before.Questions, result.Questions)
	}
}

func TestWizardKernelLinuxCExcluyeWebV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-kernel", "modulo para el nucleo de Linux en C con build y pruebas")
	questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
	for _, ref := range []string{"wizard-q-tipo-app", "wizard-r2-plataformas", "wizard-t1-control-acceso", "wizard-t2-identidad-corporativa"} {
		if hasWizardQuestionRefV0(questions, ref) {
			t.Fatalf("kernel C no debe preguntar %s: %+v", ref, questions)
		}
	}
	if !hasWizardQuestionRefV0(questions, "wizard-t3-observabilidad") ||
		!hasWizardQuestionRefV0(questions, "wizard-t6-despliegue-avanzado") {
		t.Fatalf("kernel C debe conservar observabilidad y despliegue: %+v", questions)
	}
}

func TestWizardExclusionPorAudienciaPersonalV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-personal", "agenda personal solo para mi")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "usuarios_objetivo", Values: []string{"personal"}})
	questions, resolved := webNuevaAppWizardAllGapQuestionsWithResolvedV0(session.Form)
	if hasWizardQuestionRefV0(questions, "wizard-t1-control-acceso") ||
		hasWizardQuestionRefV0(questions, "wizard-t2-identidad-corporativa") {
		t.Fatalf("audiencia personal no debe preguntar RBAC/AD/SSO: %+v", questions)
	}
	if resolved == nil {
		t.Fatalf("resolved debe ser slice no nil")
	}
}

func TestWizardOpcionUnicaSeAdoptaSinPreguntarV0(t *testing.T) {
	form := WebNuevaAppFormV0{Objetivo: "agenda personal"}
	question := wizardQuestionV0("wizard-test-opcion-unica", "usuarios_objetivo", WizardTopicUsoV0, WizardImportanceAltaV0, []WizardOptionV0{
		wizardOptionV0("personal", "nueva_app.wizard.option.u1.personal", true, "nueva_app.wizard.rationale.u1.equipo"),
		wizardOptionV0("equipo", "nueva_app.wizard.option.u1.equipo", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "audience", Value: "team"}}),
	})
	questions, resolved := wizardQuestionsAfterFactExclusionsV0(form, []WizardQuestionV0{question})
	if hasWizardQuestionRefV0(questions, "wizard-test-opcion-unica") {
		t.Fatalf("opcion unica no debe preguntarse: %+v", questions)
	}
	if len(resolved) == 0 {
		t.Fatalf("opcion unica debe producir resolucion por exclusion")
	}
}

func TestWizardReevaluaExclusionesAlCambiarRespuestaV0(t *testing.T) {
	personal := wizardSessionWithObjectiveV0("session-wizard-reevalua-personal", "agenda personal")
	personal = personal.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "usuarios_objetivo", Values: []string{"personal"}})
	if hasWizardQuestionRefV0(webNuevaAppWizardAllGapQuestionsV0(personal.Form), "wizard-t1-control-acceso") {
		t.Fatalf("personal no debe activar T1")
	}
	equipo := wizardSessionWithObjectiveV0("session-wizard-reevalua-equipo", "agenda para equipo")
	equipo = equipo.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "usuarios_objetivo", Values: []string{"equipo"}})
	if !hasWizardQuestionRefV0(webNuevaAppWizardAllGapQuestionsV0(equipo.Form), "wizard-t1-control-acceso") {
		t.Fatalf("equipo debe devolver T1 a la cola")
	}
}

func TestWizardDesviacionConJustificacionV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-justificacion", "quiero una app para una agenda")
	_, result := ApplyWebNuevaAppWizardAnswersV0(session, []WizardAnswerV0{{
		QuestionRef:   "wizard-r2-plataformas",
		UserChoice:    "web",
		Justification: "solo se usara en oficina",
	}})
	if len(result.Contrasts) != 1 || result.Contrasts[0].Justification != "solo se usara en oficina" {
		t.Fatalf("contraste no conserva justificacion: %+v", result.Contrasts)
	}
}

func TestWizardActiveDirectoryPorContextoEmpresaV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-wizard-ad", "app para mi empresa con dominio Windows y Active Directory")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "usuarios_objetivo", Values: []string{"equipo"}})
	questions := webNuevaAppWizardAllGapQuestionsV0(session.Form)
	if !hasWizardQuestionRefV0(questions, "wizard-t2-identidad-corporativa") ||
		!hasWizardQuestionOptionV0(questions, "wizard-t2-identidad-corporativa", "ldap_bind") ||
		!hasWizardQuestionOptionV0(questions, "wizard-t2-identidad-corporativa", "oidc_sso") {
		t.Fatalf("contexto empresa no activa AD/LDAP/OIDC: %+v", questions)
	}
}

func TestWizardGlosarioGeneradoV0(t *testing.T) {
	const glossaryPath = "../../docs/wizard_glosario_generado.md"
	expected := wizardGlossaryMarkdownV0()
	if os.Getenv("ORQUESTA_UPDATE_WIZARD_GLOSSARY") == "1" {
		if err := os.WriteFile(glossaryPath, []byte(expected), 0o644); err != nil {
			t.Fatalf("no se pudo actualizar glosario: %v", err)
		}
	}
	current, err := os.ReadFile(glossaryPath)
	if err != nil {
		t.Fatalf("glosario generado ausente: %v", err)
	}
	if string(current) != expected {
		t.Fatalf("glosario generado desactualizado; ejecuta ORQUESTA_UPDATE_WIZARD_GLOSSARY=1 go test ./modulos/orquesta-web -run TestWizardGlosarioGeneradoV0")
	}
}

func wizardAllRegisteredQuestionsForHelpTestV0() []WizardQuestionV0 {
	var questions []WizardQuestionV0
	questions = append(questions, wizardRequiredGapQuestionsV0(WebNuevaAppFormV0{})...)
	questions = append(questions, wizardUniversalDimensionQuestionsV0(WebNuevaAppFormV0{Objetivo: "app de equipo con datos y servidor"})...)
	questions = append(questions, wizardTechnicalDimensionQuestionsV0(WebNuevaAppFormV0{
		Objetivo:         "app para empresa con dominio Windows",
		UsuariosObjetivo: []string{"equipo"},
		Deploy:           WebNuevaAppDeployFormV0{Target: "contenedor"},
	})...)
	questions = append(questions, wizardRuleR3IntegracionesDominioV0(WebNuevaAppFormV0{Objetivo: "agenda tienda mapa inventario notas tareas finanzas contactos reservas salud educacion comunidad iot galeria facturacion"})...)
	for _, question := range []*WizardQuestionV0{
		wizardRuleR1PersonalCompartidoV0(WebNuevaAppFormV0{Objetivo: "agenda"}),
		wizardRuleR2PlataformasV0(WebNuevaAppFormV0{Objetivo: "agenda"}),
		wizardRuleR4StorageV0(WebNuevaAppFormV0{Datos: WebNuevaAppDatosFormV0{DBRequired: true}}),
		wizardRuleR5IntegracionGobiernoV0(WebNuevaAppFormV0{Integraciones: []WebNuevaAppConnectorFormV0{{Tipo: "api", Nombre: "externa"}}}),
		wizardRuleR6MovilPlataformasV0(WebNuevaAppFormV0{TipoApp: "mobile"}),
		wizardRuleR7DeployCompatibleV0(WebNuevaAppFormV0{TipoApp: "mobile", Deploy: WebNuevaAppDeployFormV0{Target: "desktop"}}),
		wizardRuleR8UsuariosCompartidosV0(WebNuevaAppFormV0{Objetivo: "app compartida para clientes"}),
	} {
		if question != nil {
			questions = append(questions, *question)
		}
	}
	return dedupeWizardQuestionsV0(questions)
}

func hasWizardQuestionRefV0(questions []WizardQuestionV0, ref string) bool {
	for _, question := range questions {
		if question.QuestionRef == ref {
			return true
		}
	}
	return false
}

func wizardSessionWithObjectiveV0(sessionID string, objective string) WebNuevaAppIntakeSessionV0 {
	session := NewWebNuevaAppIntakeSessionV0(sessionID, "", "", "")
	return session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{
		Field: "objetivo",
		Value: objective,
	})
}

func requireWizardI18nKeyV0(t *testing.T, catalog NuevaAppI18nCatalogV0, key string) {
	t.Helper()
	if strings.TrimSpace(key) == "" {
		t.Fatalf("clave i18n vacia")
	}
	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		if text := strings.TrimSpace(catalog.lookupExact(locale, key)); text == "" {
			t.Fatalf("clave wizard sin texto exacto %s/%s", locale, key)
		}
	}
}

func hasWizardQuestionOptionV0(questions []WizardQuestionV0, ref string, value string) bool {
	for _, question := range questions {
		if question.QuestionRef != ref {
			continue
		}
		for _, option := range question.Options {
			if option.Value == value {
				return true
			}
		}
	}
	return false
}
