package orquestaweb

import (
	"strconv"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const wizardMaxQuestionsPerTurnV0 = 4

type wizardCrossGapRuleV0 struct {
	RuleRef string
	Build   func(WebNuevaAppFormV0) *WizardQuestionV0
}

var wizardCrossGapRulesV0 = []wizardCrossGapRuleV0{
	{RuleRef: "R1", Build: wizardRuleR1PersonalCompartidoV0},
	{RuleRef: "R2", Build: wizardRuleR2PlataformasV0},
	{RuleRef: "R4", Build: wizardRuleR4StorageV0},
	{RuleRef: "R5", Build: wizardRuleR5IntegracionGobiernoV0},
	{RuleRef: "R6", Build: wizardRuleR6MovilPlataformasV0},
	{RuleRef: "R7", Build: wizardRuleR7DeployCompatibleV0},
	{RuleRef: "R8", Build: wizardRuleR8UsuariosCompartidosV0},
}

func WebNuevaAppWizardGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	return selectWizardTurnQuestionsV0(webNuevaAppWizardAllGapQuestionsV0(form))
}

func webNuevaAppWizardAllGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	form = normalizeWizardFormForQuestionsV0(form)
	out := make([]WizardQuestionV0, 0, 16)
	out = append(out, wizardRequiredGapQuestionsV0(form)...)
	for _, rule := range wizardCrossGapRulesV0 {
		if question := rule.Build(form); question != nil {
			out = append(out, *question)
		}
	}
	out = append(out, wizardRuleR3IntegracionesDominioV0(form)...)
	return dedupeWizardQuestionsV0(out)
}

func wizardRequiredGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	var out []WizardQuestionV0
	if trimV0(form.Locale) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-locale",
			"locale",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("es-ES", "nueva_app.wizard.option.locale.es", true, "nueva_app.wizard.rationale.locale.es"),
				wizardOptionV0("en-US", "nueva_app.wizard.option.locale.en", false, ""),
			},
		))
	}
	if trimV0(form.Nombre) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-nombre",
			"nombre",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0(wizardNameFromObjectiveV0(form.Objetivo), "nueva_app.wizard.option.nombre.inferido", true, "nueva_app.wizard.rationale.nombre.inferido"),
				wizardOptionV0("Nueva app", "nueva_app.wizard.option.nombre.generico", false, ""),
			},
		))
	}
	if trimV0(form.Objetivo) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-objetivo",
			"objetivo",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("Describir objetivo verificable", "nueva_app.wizard.option.objetivo.describir", true, "nueva_app.wizard.rationale.objetivo.describir"),
			},
		))
	}
	if trimV0(form.TipoApp) == "" {
		recommended := wizardRecommendedTipoAppV0(form)
		out = append(out, wizardQuestionV0(
			"wizard-q-tipo-app",
			"tipo_app",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("web", "nueva_app.wizard.option.tipo_app.web", recommended == "web", "nueva_app.wizard.rationale.tipo_app.web"),
				wizardOptionV0("api", "nueva_app.wizard.option.tipo_app.api", recommended == "api", "nueva_app.wizard.rationale.tipo_app.api"),
				wizardOptionV0("mobile", "nueva_app.wizard.option.tipo_app.mobile", recommended == "mobile", "nueva_app.wizard.rationale.tipo_app.mobile"),
				wizardOptionV0("desktop", "nueva_app.wizard.option.tipo_app.desktop", recommended == "desktop", "nueva_app.wizard.rationale.tipo_app.desktop"),
			},
		))
	}
	if !webNuevaAppIntakeFieldCapturedV0(form, "datos") {
		out = append(out, wizardQuestionV0(
			"wizard-q-datos",
			"datos.necesidad_funcional",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("gestion_con_persistencia", "nueva_app.wizard.option.datos.gestion", true, "nueva_app.wizard.rationale.datos.gestion"),
				wizardOptionV0("solo_consulta", "nueva_app.wizard.option.datos.consulta", false, ""),
				wizardOptionV0("sin_persistencia", "nueva_app.wizard.option.datos.sin_persistencia", false, ""),
			},
		))
	}
	if trimV0(form.Deploy.Target) == "" {
		recommended := wizardRecommendedDeployTargetV0(form.TipoApp)
		out = append(out, wizardQuestionV0(
			"wizard-q-deploy",
			"deploy.target",
			WizardTopicEntregaV0,
			WizardImportanceAltaV0,
			wizardDeployOptionsV0(form.TipoApp, recommended),
		))
	}
	return out
}

func wizardRuleR1PersonalCompartidoV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	if len(compactStringsV0(form.UsuariosObjetivo)) > 0 || !wizardObjectiveHasShareAmbiguityV0(need) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r1-uso-personal-compartido",
		"usuarios_objetivo",
		WizardTopicUsoV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("compartir_auth_simple", "nueva_app.wizard.option.uso.compartir", true, "nueva_app.wizard.rationale.uso.compartir"),
			wizardOptionV0("personal", "nueva_app.wizard.option.uso.personal", false, ""),
		},
	)
	return &question
}

func wizardRuleR2PlataformasV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if len(compactStringsV0(form.Plataformas)) > 0 || wizardTipoAppIsMobileLikeV0(form.TipoApp) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r2-plataformas",
		"plataformas",
		WizardTopicUsoV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("web_mobile", "nueva_app.wizard.option.plataformas.ambas", true, "nueva_app.wizard.rationale.plataformas.ambas"),
			wizardOptionV0("web", "nueva_app.wizard.option.plataformas.pc", false, ""),
			wizardOptionV0("mobile", "nueva_app.wizard.option.plataformas.movil", false, ""),
		},
	)
	return &question
}

func wizardRuleR3IntegracionesDominioV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	nextIntegrationIndex := wizardNextIntegrationIndexV0(form)
	var out []WizardQuestionV0
	for _, pack := range wizardDomainIntegrationPacksV0() {
		if !guidedContainsAnyV0(need, pack.Keywords...) || wizardHasIntegrationTypeV0(form, pack.IntegrationType) {
			continue
		}
		question := wizardQuestionV0(
			pack.QuestionRef,
			"integraciones."+strconv.Itoa(nextIntegrationIndex)+".tipo",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			pack.Options,
		)
		out = append(out, question)
		nextIntegrationIndex++
		if nextIntegrationIndex >= nuevaAppMaxIntegrationRowsV0 {
			break
		}
	}
	if len(out) == 0 &&
		trimV0(form.Objetivo) != "" &&
		!wizardObjectiveMatchesKnownDomainPackV0(need) &&
		!wizardHasOpenDomainDescriptionV0(form) {
		out = append(out, wizardQuestionV0(
			"wizard-r3-dominio-abierto",
			"descripcion",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("describir_flujo_diario", "nueva_app.wizard.option.dominio.flujo_diario", true, "nueva_app.wizard.rationale.dominio.flujo_diario"),
				wizardOptionV0("describir_integraciones", "nueva_app.wizard.option.dominio.integraciones", false, ""),
			},
		))
	}
	if out == nil {
		return []WizardQuestionV0{}
	}
	return out
}

func wizardRuleR4StorageV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if !form.Datos.DBRequired || wizardHasStorageTypeV0(form) {
		return nil
	}
	recommended := "relacional"
	if len(form.Datos.Fuentes) > 0 || len(form.Integraciones) > 0 {
		recommended = "mixta"
	}
	question := wizardQuestionV0(
		"wizard-r4-storage-db",
		"datos.storage.0.tipo",
		WizardTopicDatosV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("relacional", "nueva_app.wizard.option.storage.relacional", recommended == "relacional", "nueva_app.wizard.rationale.storage.relacional"),
			wizardOptionV0("documental", "nueva_app.wizard.option.storage.documental", recommended == "documental", "nueva_app.wizard.rationale.storage.documental"),
			wizardOptionV0("mixta", "nueva_app.wizard.option.storage.mixta", recommended == "mixta", "nueva_app.wizard.rationale.storage.mixta"),
			wizardOptionV0("sin_preferencia", "nueva_app.wizard.option.storage.sin_preferencia", recommended == "sin_preferencia", "nueva_app.wizard.rationale.storage.sin_preferencia"),
		},
	)
	return &question
}

func wizardRuleR5IntegracionGobiernoV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	for index, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "" && trimV0(integration.Nombre) == "" {
			continue
		}
		if trimV0(integration.Auth) != "" && trimV0(integration.Criticidad) != "" {
			continue
		}
		question := wizardQuestionV0(
			"wizard-r5-integracion-gobierno",
			"integraciones."+strconv.Itoa(index)+".auth",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("auth_simple_criticidad_media", "nueva_app.wizard.option.integracion.gobierno.simple", true, "nueva_app.wizard.rationale.integracion.gobierno.simple"),
				wizardOptionV0("publica_criticidad_baja", "nueva_app.wizard.option.integracion.gobierno.publica", false, ""),
				wizardOptionV0("oauth_criticidad_alta", "nueva_app.wizard.option.integracion.gobierno.oauth_alta", false, ""),
			},
		)
		return &question
	}
	return nil
}

func wizardRuleR6MovilPlataformasV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if !wizardTipoAppIsMobileLikeV0(form.TipoApp) || wizardHasConcreteMobilePlatformV0(form) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r6-movil-plataformas",
		"plataformas",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("mobile_ios_android", "nueva_app.wizard.option.mobile.ios_android", true, "nueva_app.wizard.rationale.mobile.ios_android"),
			wizardOptionV0("mobile_ios", "nueva_app.wizard.option.mobile.ios", false, ""),
			wizardOptionV0("mobile_android", "nueva_app.wizard.option.mobile.android", false, ""),
			wizardOptionV0("web_mobile", "nueva_app.wizard.option.mobile.web_responsive", false, ""),
		},
	)
	return &question
}

func wizardRuleR7DeployCompatibleV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	target := trimV0(form.Deploy.Target)
	if target == "" || wizardDeployTargetCompatibleV0(form.TipoApp, target) {
		return nil
	}
	recommended := wizardRecommendedDeployTargetV0(form.TipoApp)
	question := wizardQuestionV0(
		"wizard-r7-deploy-compatible",
		"deploy.target",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		wizardDeployOptionsV0(form.TipoApp, recommended),
	)
	return &question
}

func wizardRuleR8UsuariosCompartidosV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if len(compactStringsV0(form.UsuariosObjetivo)) > 0 || !wizardFormSharedUseV0(form) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r8-usuarios-compartido",
		"usuarios_objetivo",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("equipo_pequeno", "nueva_app.wizard.option.usuarios.equipo", true, "nueva_app.wizard.rationale.usuarios.equipo"),
			wizardOptionV0("profesionales_internos", "nueva_app.wizard.option.usuarios.internos", false, ""),
			wizardOptionV0("clientes_invitados", "nueva_app.wizard.option.usuarios.clientes", false, ""),
		},
	)
	return &question
}

type wizardDomainIntegrationPackV0 struct {
	QuestionRef     string
	IntegrationType string
	Keywords        []string
	Options         []WizardOptionV0
}

func wizardDomainIntegrationPacksV0() []wizardDomainIntegrationPackV0 {
	return []wizardDomainIntegrationPackV0{
		{
			QuestionRef:     "wizard-r3-integracion-agenda",
			IntegrationType: "calendar",
			Keywords:        []string{"agenda", "calendario", "calendar", "cita", "citas"},
			Options: []WizardOptionV0{
				wizardOptionV0("calendar_enterprise_neutral", "nueva_app.wizard.option.integracion.calendar.enterprise", true, "nueva_app.wizard.rationale.integracion.calendar.enterprise"),
				wizardOptionV0("calendar_google_workspace", "nueva_app.wizard.option.integracion.calendar.google", false, ""),
				wizardOptionV0("calendar_microsoft_365", "nueva_app.wizard.option.integracion.calendar.microsoft", false, ""),
				wizardOptionV0("calendar_caldav", "nueva_app.wizard.option.integracion.calendar.caldav", false, ""),
			},
		},
		{
			QuestionRef:     "wizard-r3-integracion-tienda",
			IntegrationType: "ecommerce",
			Keywords:        []string{"tienda", "venta", "ventas", "ecommerce", "catalogo", "carrito", "pago", "pagos", "cobro", "checkout"},
			Options: []WizardOptionV0{
				wizardOptionV0("ecommerce_capability", "nueva_app.wizard.option.integracion.ecommerce.capability", true, "nueva_app.wizard.rationale.integracion.ecommerce.capability"),
				wizardOptionV0("ecommerce_payments_sync", "nueva_app.wizard.option.integracion.ecommerce.payments_sync", false, ""),
				wizardOptionV0("payments_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
			},
		},
		{
			QuestionRef:     "wizard-r3-integracion-mapa",
			IntegrationType: "maps",
			Keywords:        []string{"mapa", "mapas", "ubicacion", "geo", "cerca"},
			Options: []WizardOptionV0{
				wizardOptionV0("maps_public_sources", "nueva_app.wizard.option.integracion.maps.public_sources", true, "nueva_app.wizard.rationale.integracion.maps.public_sources"),
				wizardOptionV0("maps_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
			},
		},
		wizardDomainPackV0(
			"wizard-r3-integracion-inventario",
			"inventory",
			[]string{"inventario", "almacen", "almacenaje", "stock", "qr", "codigo de barras", "proveedores"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-notas",
			"documents",
			[]string{"notas", "documentos", "wiki", "markdown", "plantillas", "adjuntos"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-tareas",
			"project_tasks",
			[]string{"tareas", "proyectos", "kanban", "subtareas", "dependencias", "prioridades"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-finanzas",
			"finance",
			[]string{"finanzas", "gastos", "presupuesto", "presupuestos", "extractos", "ofx", "ahorro"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-crm",
			"crm",
			[]string{"contactos", "crm", "clientes", "pipeline", "seguimientos", "vcard"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-reservas",
			"booking",
			[]string{"reservas", "turnos", "citas online", "disponibilidad", "lista de espera"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-salud",
			"health",
			[]string{"salud", "fitness", "habitos", "metricas", "wearables", "racha"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-educacion",
			"education",
			[]string{"educacion", "cursos", "lecciones", "alumnos", "ejercicios", "certificados"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-comunidad",
			"community",
			[]string{"comunidad", "foro", "hilos", "votos", "moderacion", "reputacion"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-iot",
			"iot",
			[]string{"domotica", "iot", "sensores", "mqtt", "dispositivos", "telemetria"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-media",
			"media",
			[]string{"galeria", "media", "fotos", "imagenes", "albumes", "miniaturas", "transcodificacion"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-facturacion",
			"billing",
			[]string{"facturacion", "facturas", "legal", "impuestos", "pdf", "numeracion"},
		),
	}
}

func wizardDomainPackV0(questionRef string, integrationType string, keywords []string) wizardDomainIntegrationPackV0 {
	return wizardDomainIntegrationPackV0{
		QuestionRef:     questionRef,
		IntegrationType: integrationType,
		Keywords:        keywords,
		Options: []WizardOptionV0{
			wizardOptionV0(integrationType+"_capability", "nueva_app.wizard.option.integracion.domain.capability", true, "nueva_app.wizard.rationale.integracion.domain.capability"),
			wizardOptionV0(integrationType+"_sync", "nueva_app.wizard.option.integracion.domain.sync", false, ""),
			wizardOptionV0(integrationType+"_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
		},
	}
}

func wizardQuestionV0(ref, field, topic, importance string, options []WizardOptionV0) WizardQuestionV0 {
	return WizardQuestionV0{
		QuestionRef: ref,
		Field:       field,
		TopicGroup:  topic,
		Importance:  importance,
		PromptKey:   "nueva_app.wizard.question." + ref + ".prompt",
		WhyKey:      "nueva_app.wizard.question." + ref + ".why",
		HelpKey:     "nueva_app.wizard.question." + ref + ".help",
		Options:     normalizeWizardOptionsV0(options),
	}
}

func wizardOptionV0(value, labelKey string, recommended bool, rationaleKey string) WizardOptionV0 {
	if recommended && trimV0(rationaleKey) == "" {
		rationaleKey = "nueva_app.wizard.rationale.default"
	}
	return WizardOptionV0{
		Value:        trimV0(value),
		LabelKey:     trimV0(labelKey),
		HelpKey:      wizardOptionHelpKeyV0(labelKey),
		ExampleKey:   wizardOptionExampleKeyV0(labelKey),
		Recommended:  recommended,
		RationaleKey: trimV0(rationaleKey),
	}
}

func wizardOptionHelpKeyV0(labelKey string) string {
	return strings.Replace(trimV0(labelKey), "nueva_app.wizard.option.", "nueva_app.wizard.help.option.", 1)
}

func wizardOptionExampleKeyV0(labelKey string) string {
	key := strings.Replace(trimV0(labelKey), "nueva_app.wizard.option.", "nueva_app.wizard.example.option.", 1)
	if _, ok := nuevaAppWizardOptionExampleKeysV0()[key]; !ok {
		return ""
	}
	return key
}

func normalizeWizardOptionsV0(options []WizardOptionV0) []WizardOptionV0 {
	out := make([]WizardOptionV0, 0, len(options))
	recommended := false
	for _, option := range options {
		if trimV0(option.Value) == "" {
			continue
		}
		if option.Recommended {
			if recommended {
				option.Recommended = false
				option.RationaleKey = ""
			}
			recommended = true
		}
		out = append(out, option)
	}
	if !recommended && len(out) > 0 {
		out[0].Recommended = true
		if out[0].RationaleKey == "" {
			out[0].RationaleKey = "nueva_app.wizard.rationale.default"
		}
	}
	if out == nil {
		return []WizardOptionV0{}
	}
	return out
}

func selectWizardTurnQuestionsV0(questions []WizardQuestionV0) []WizardQuestionV0 {
	questions = dedupeWizardQuestionsV0(questions)
	for _, topic := range []string{WizardTopicUsoV0, WizardTopicDatosV0, WizardTopicEntregaV0} {
		var selected []WizardQuestionV0
		for _, question := range questions {
			if question.TopicGroup != topic {
				continue
			}
			selected = append(selected, question)
			if len(selected) == wizardMaxQuestionsPerTurnV0 {
				break
			}
		}
		if selected != nil {
			return selected
		}
	}
	return []WizardQuestionV0{}
}

func dedupeWizardQuestionsV0(questions []WizardQuestionV0) []WizardQuestionV0 {
	seenRef := map[string]bool{}
	out := make([]WizardQuestionV0, 0, len(questions))
	for _, question := range questions {
		if trimV0(question.QuestionRef) == "" || seenRef[question.QuestionRef] {
			continue
		}
		seenRef[question.QuestionRef] = true
		out = append(out, question)
	}
	if out == nil {
		return []WizardQuestionV0{}
	}
	return out
}

func normalizeWizardFormForQuestionsV0(form WebNuevaAppFormV0) WebNuevaAppFormV0 {
	form.Locale = trimV0(form.Locale)
	form.Nombre = trimV0(form.Nombre)
	form.Objetivo = trimV0(form.Objetivo)
	form.TipoApp = trimV0(form.TipoApp)
	return form
}

func wizardRecommendedTipoAppV0(form WebNuevaAppFormV0) string {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	switch {
	case guidedContainsAnyV0(need, "api", "servicio", "backend"):
		return "api"
	case guidedContainsAnyV0(need, "movil", "mobile", "android", "ios", "nativo", "native"):
		return "mobile"
	case guidedContainsAnyV0(need, "desktop", "escritorio", "pc offline"):
		return "desktop"
	default:
		return "web"
	}
}

func wizardRecommendedDeployTargetV0(tipoApp string) string {
	switch trimV0(tipoApp) {
	case "mobile":
		return "mobile_store"
	case "desktop":
		return "desktop"
	case "api", "web", "mixed", "":
		return "contenedor"
	default:
		return "local"
	}
}

func wizardDeployOptionsV0(tipoApp, recommended string) []WizardOptionV0 {
	values := []string{"local", "contenedor", "paas"}
	switch trimV0(tipoApp) {
	case "mobile":
		values = []string{"mobile_store", "local", "contenedor"}
	case "desktop":
		values = []string{"desktop", "local", "contenedor"}
	case "api":
		values = []string{"contenedor", "paas", "serverless", "kubernetes"}
	}
	out := make([]WizardOptionV0, 0, len(values))
	for _, value := range values {
		out = append(out, wizardOptionV0(
			value,
			"nueva_app.wizard.option.deploy."+value,
			value == recommended,
			"nueva_app.wizard.rationale.deploy."+value,
		))
	}
	return out
}

func wizardDeployTargetCompatibleV0(tipoApp, target string) bool {
	switch trimV0(tipoApp) {
	case "mobile":
		return containsStringV0(target, "mobile_store", "local", "contenedor")
	case "desktop":
		return containsStringV0(target, "desktop", "local", "contenedor")
	case "web", "api", "mixed", "":
		return !containsStringV0(target, "mobile_store", "desktop")
	default:
		return true
	}
}

func wizardObjectiveHasShareAmbiguityV0(need string) bool {
	if guidedContainsAnyV0(need, "personal", "privado", "solo para mi", "individual") {
		return false
	}
	if guidedContainsAnyV0(need, "compartir", "compartida", "colaborativo", "equipo", "empresa") {
		return false
	}
	return guidedContainsAnyV0(need, "agenda", "calendario", "calendar", "tareas", "notas", "proyecto")
}

func wizardFormSharedUseV0(form WebNuevaAppFormV0) bool {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion + " " + strings.Join(form.Restricciones, " "))
	if guidedContainsAnyV0(need, "compartir", "compartida", "colaborativo", "equipo", "empresa", "clientes", "usuarios") {
		return true
	}
	for _, integration := range form.Integraciones {
		if trimV0(integration.Auth) != "" {
			return true
		}
	}
	return false
}

func wizardTipoAppIsMobileLikeV0(tipoApp string) bool {
	return containsStringV0(trimV0(tipoApp), "mobile", "mixed")
}

func wizardHasConcreteMobilePlatformV0(form WebNuevaAppFormV0) bool {
	joined := normalizeGuidedNeedV0(strings.Join(append(form.Plataformas, form.PreferenciasTecnicas.Preferencias...), " "))
	return guidedContainsAnyV0(joined, "ios", "android", "web responsive", "web_responsive", "mobile_ios", "mobile_android")
}

func wizardHasIntegrationTypeV0(form WebNuevaAppFormV0, integrationType string) bool {
	for _, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == integrationType {
			return true
		}
	}
	return false
}

func wizardHasStorageTypeV0(form WebNuevaAppFormV0) bool {
	for _, storage := range form.Datos.Storage {
		if orquestafactory.DataStorageTypeSupportedV0(trimV0(storage.Tipo)) {
			return true
		}
	}
	return false
}

func wizardHasOpenDomainDescriptionV0(form WebNuevaAppFormV0) bool {
	description := normalizeGuidedNeedV0(form.Descripcion)
	return guidedContainsAnyV0(description, "usuario un dia normal", "flujo diario", "funcion principal", "capacidad principal")
}

func wizardObjectiveMatchesKnownDomainPackV0(need string) bool {
	for _, pack := range wizardDomainIntegrationPacksV0() {
		if guidedContainsAnyV0(need, pack.Keywords...) {
			return true
		}
	}
	return false
}

func wizardNextIntegrationIndexV0(form WebNuevaAppFormV0) int {
	for index, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "" && trimV0(integration.Nombre) == "" {
			return index
		}
	}
	if len(form.Integraciones) >= nuevaAppMaxIntegrationRowsV0 {
		return nuevaAppMaxIntegrationRowsV0 - 1
	}
	return len(form.Integraciones)
}

func wizardNameFromObjectiveV0(objective string) string {
	name := guidedTitleFromNeedV0(objective)
	if trimV0(name) == "" {
		return "Nueva app"
	}
	return name
}

func wizardQuestionRecommendedOptionV0(question WizardQuestionV0) (WizardOptionV0, bool) {
	for _, option := range question.Options {
		if option.Recommended {
			return option, true
		}
	}
	return WizardOptionV0{}, false
}

func containsStringV0(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
