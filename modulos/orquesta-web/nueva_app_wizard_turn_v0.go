package orquestaweb

import (
	"strconv"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const WebNuevaAppWizardTurnSchemaV0 = "web_nueva_app_wizard_turn.v0"

func NewWebNuevaAppWizardTurnResultV0(session WebNuevaAppIntakeSessionV0) WizardTurnResultV0 {
	session = refreshWebNuevaAppIntakeSessionV0(session)
	previewForm := ApplyWebNuevaAppWizardEngineeringDefaultsV0(session.Form)
	specPreview := previewForm.ToAppSpecRequestV0()
	allQuestions, resolved := webNuevaAppWizardAllGapQuestionsWithResolvedV0(previewForm)
	questions := selectWizardTurnQuestionsV0(allQuestions)
	specComplete := len(webNuevaAppWizardHighImportanceQuestionsV0(allQuestions)) == 0 &&
		len(orquestafactory.ValidateAppSpecRequestV0(specPreview)) == 0
	return WizardTurnResultV0{
		SchemaVersion:       WebNuevaAppWizardTurnSchemaV0,
		SessionRef:          firstNuevaAppValueV0(session.SessionRef, session.SessionID),
		Turn:                len(session.Decisions) + 1,
		Questions:           questions,
		ResolvedByExclusion: resolved,
		EngineeringDefaults: WebNuevaAppWizardEngineeringDefaultsV0(),
		GlossaryExpanded:    session.GlossaryExpanded,
		SpecComplete:        specComplete,
		SpecPreview:         &specPreview,
		LaunchReady:         specComplete,
	}
}

func ApplyWebNuevaAppWizardAnswersV0(
	session WebNuevaAppIntakeSessionV0,
	answers []WizardAnswerV0,
) (WebNuevaAppIntakeSessionV0, WizardTurnResultV0) {
	session = refreshWebNuevaAppIntakeSessionV0(session)
	questions, _ := webNuevaAppWizardAllGapQuestionsWithResolvedV0(ApplyWebNuevaAppWizardEngineeringDefaultsV0(session.Form))
	questionsByRef := map[string]WizardQuestionV0{}
	for _, question := range questions {
		questionsByRef[question.QuestionRef] = question
	}
	var decisions []WebNuevaAppIntakeDecisionV0
	var contrasts []WizardContrastV0
	for _, answer := range answers {
		if response, ok := wizardComprehensionResponseV0(questions, answer.ComprehensionQuery); ok {
			result := NewWebNuevaAppWizardTurnResultV0(session)
			result.GlossaryResponse = &response
			return session, result
		}
		question, ok := questionsByRef[trimV0(answer.QuestionRef)]
		if !ok {
			continue
		}
		choice := trimV0(answer.UserChoice)
		if choice == "" {
			continue
		}
		recommended, hasRecommended := wizardQuestionRecommendedOptionV0(question)
		if hasRecommended && recommended.Value != choice {
			contrasts = append(contrasts, WizardContrastV0{
				QuestionRef:   question.QuestionRef,
				UserChoice:    choice,
				Recommended:   recommended.Value,
				RationaleKey:  recommended.RationaleKey,
				Justification: trimV0(answer.Justification),
			})
		}
		next := wizardDecisionsForAnswerV0(question, choice, answer.FreeText)
		for _, decision := range next {
			session = session.ApplyDecisionV0(decision)
		}
		decisions = append(decisions, next...)
	}
	session.Form = ApplyWebNuevaAppWizardEngineeringDefaultsV0(session.Form)
	for {
		_, resolved := webNuevaAppWizardAllGapQuestionsWithResolvedV0(session.Form)
		if len(resolved) == 0 {
			break
		}
		for _, decision := range resolved {
			session = session.ApplyDecisionV0(decision)
			decisions = append(decisions, decision)
		}
	}
	session = refreshWebNuevaAppIntakeSessionV0(session)
	result := NewWebNuevaAppWizardTurnResultV0(session)
	result.Decisions = decisions
	result.Contrasts = contrasts
	return session, result
}

func wizardComprehensionResponseV0(
	questions []WizardQuestionV0,
	query string,
) (WizardGlossaryResponseV0, bool) {
	normalized := normalizeGuidedNeedV0(query)
	normalized = strings.TrimPrefix(normalized, "que es ")
	normalized = strings.TrimPrefix(normalized, "que significa ")
	normalized = strings.Trim(normalized, " ?")
	if normalized == "" {
		return WizardGlossaryResponseV0{}, false
	}
	for _, question := range questions {
		if wizardGlossaryTextMatchesV0(normalized, question.PromptKey, question.HelpKey) {
			return WizardGlossaryResponseV0{Query: trimV0(query), TermKey: question.PromptKey, HelpKey: question.HelpKey}, true
		}
		for _, option := range question.Options {
			if wizardGlossaryTextMatchesV0(normalized, option.LabelKey, option.HelpKey, option.ExampleKey) {
				return WizardGlossaryResponseV0{Query: trimV0(query), TermKey: option.LabelKey, HelpKey: option.HelpKey, ExampleKey: option.ExampleKey}, true
			}
		}
	}
	for _, defaultValue := range WebNuevaAppWizardEngineeringDefaultsV0() {
		if wizardGlossaryTextMatchesV0(normalized, defaultValue.Value, defaultValue.HelpKey, defaultValue.ExampleKey) {
			return WizardGlossaryResponseV0{Query: trimV0(query), TermKey: defaultValue.Value, HelpKey: defaultValue.HelpKey, ExampleKey: defaultValue.ExampleKey}, true
		}
	}
	return WizardGlossaryResponseV0{}, false
}

func wizardGlossaryTextMatchesV0(needle string, values ...string) bool {
	catalog := NewNuevaAppI18nCatalogV0()
	for _, value := range values {
		for _, candidate := range []string{
			value,
			catalog.lookupExact(NuevaAppI18nDefaultLocaleV0, value),
			catalog.lookupExact(NuevaAppI18nEnglishLocaleV0, value),
		} {
			haystack := normalizeGuidedNeedV0(candidate)
			if haystack == "" {
				continue
			}
			if strings.Contains(haystack, needle) || strings.Contains(needle, haystack) {
				return true
			}
		}
	}
	return false
}

func AcceptWebNuevaAppWizardRecommendationsV0(
	session WebNuevaAppIntakeSessionV0,
	maxTurns int,
) (WebNuevaAppIntakeSessionV0, []WizardTurnResultV0) {
	if maxTurns <= 0 {
		maxTurns = 6
	}
	var turns []WizardTurnResultV0
	for turn := 0; turn < maxTurns; turn++ {
		current := NewWebNuevaAppWizardTurnResultV0(session)
		if current.LaunchReady || len(current.Questions) == 0 {
			turns = append(turns, current)
			break
		}
		answers := make([]WizardAnswerV0, 0, len(current.Questions))
		for _, question := range current.Questions {
			if option, ok := wizardQuestionRecommendedOptionV0(question); ok {
				answers = append(answers, WizardAnswerV0{
					QuestionRef: question.QuestionRef,
					UserChoice:  option.Value,
				})
			}
		}
		var applied WizardTurnResultV0
		session, applied = ApplyWebNuevaAppWizardAnswersV0(session, answers)
		turns = append(turns, applied)
		if applied.LaunchReady {
			break
		}
	}
	return session, turns
}

func wizardDecisionsForAnswerV0(
	question WizardQuestionV0,
	choice string,
	freeText bool,
) []WebNuevaAppIntakeDecisionV0 {
	choice = trimV0(choice)
	if freeText {
		return []WebNuevaAppIntakeDecisionV0{guidedAnswerDecisionV0(question.Field, choice)}
	}
	if question.Field == "descripcion" {
		switch choice {
		case "describir_flujo_diario":
			return []WebNuevaAppIntakeDecisionV0{{Field: "descripcion", Value: "Describir que debe poder hacer un usuario un dia normal antes de cerrar el contrato."}}
		case "describir_integraciones":
			return []WebNuevaAppIntakeDecisionV0{{Field: "descripcion", Value: "Describir servicios, datos o herramientas existentes que deben conectarse por adaptadores."}}
		}
	}
	switch question.QuestionRef {
	case "wizard-t1-control-acceso":
		index, _, ok := indexedNuevaAppDecisionFieldV0(question.Field, "integraciones")
		if !ok {
			index = 0
		}
		prefix := "integraciones." + strconv.Itoa(index) + "."
		return []WebNuevaAppIntakeDecisionV0{
			{Field: prefix + "tipo", Value: "auth"},
			{Field: prefix + "nombre", Value: "control de acceso"},
			{Field: prefix + "proposito", Value: wizardAccessPurposeV0(choice)},
			{Field: prefix + "auth", Value: "gestionado por adaptador"},
			{Field: prefix + "criticidad", Value: "alta"},
			{Field: prefix + "requerido", Value: "true"},
		}
	case "wizard-t2-identidad-corporativa":
		index, _, ok := indexedNuevaAppDecisionFieldV0(question.Field, "integraciones")
		if !ok {
			index = 0
		}
		prefix := "integraciones." + strconv.Itoa(index) + "."
		return []WebNuevaAppIntakeDecisionV0{
			{Field: prefix + "tipo", Value: "auth"},
			{Field: prefix + "nombre", Value: "identidad corporativa"},
			{Field: prefix + "proposito", Value: wizardIdentityPurposeV0(choice)},
			{Field: prefix + "auth", Value: choice},
			{Field: prefix + "criticidad", Value: "alta"},
			{Field: prefix + "requerido", Value: "true"},
		}
	case "wizard-t3-observabilidad":
		return []WebNuevaAppIntakeDecisionV0{
			{Field: "calidad.observabilidad", Value: "true"},
			{Field: "agentes.preferencias", Values: []string{"observabilidad: " + choice}},
		}
	case "wizard-t4-persistencia-tecnica":
		return wizardTechnicalPersistenceDecisionsV0(question.Field, choice)
	case "wizard-t5-api-contratos":
		index, _, ok := indexedNuevaAppDecisionFieldV0(question.Field, "integraciones")
		if !ok {
			index = 0
		}
		prefix := "integraciones." + strconv.Itoa(index) + "."
		return []WebNuevaAppIntakeDecisionV0{
			{Field: prefix + "tipo", Value: "api_contract"},
			{Field: prefix + "nombre", Value: "contratos de API"},
			{Field: prefix + "proposito", Value: wizardAPIPurposeV0(choice)},
			{Field: prefix + "direccion", Value: "inbound"},
			{Field: prefix + "auth", Value: "rate-limit e idempotencia por adaptador"},
			{Field: prefix + "criticidad", Value: "alta"},
			{Field: prefix + "requerido", Value: "true"},
		}
	case "wizard-t6-despliegue-avanzado":
		return []WebNuevaAppIntakeDecisionV0{{Field: "deploy.restricciones", Values: []string{choice}}}
	case "wizard-t7-resiliencia-rendimiento":
		return []WebNuevaAppIntakeDecisionV0{{Field: "agentes.preferencias", Values: []string{"resiliencia: " + choice}}}
	case "wizard-t8-cumplimiento-tecnico":
		return []WebNuevaAppIntakeDecisionV0{
			{Field: "calidad.compliance", Values: []string{choice}},
			{Field: "datos.operacion.auditoria", Value: "true"},
			{Field: "agentes.preferencias", Values: []string{"cumplimiento tecnico: " + choice}},
		}
	}
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(question.Field, "integraciones"); ok {
		return wizardIntegrationDecisionsV0(index, suffix, choice)
	}
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(question.Field, "datos.storage"); ok && suffix == "tipo" {
		return []WebNuevaAppIntakeDecisionV0{
			{Field: question.Field, Value: choice},
			{Field: "datos.storage." + strconv.Itoa(index) + ".proposito", Value: wizardStoragePurposeV0(choice)},
			{Field: "datos.storage." + strconv.Itoa(index) + ".requerido", Value: "true"},
		}
	}
	switch question.Field {
	case "plataformas":
		return []WebNuevaAppIntakeDecisionV0{{Field: "plataformas", Values: wizardPlatformValuesV0(choice)}}
	case "usuarios_objetivo":
		return []WebNuevaAppIntakeDecisionV0{{Field: "usuarios_objetivo", Values: wizardAudienceValuesV0(choice)}}
	case "datos.necesidad_funcional":
		return wizardDataNeedDecisionsV0(choice)
	default:
		return []WebNuevaAppIntakeDecisionV0{guidedAnswerDecisionV0(question.Field, choice)}
	}
}

func wizardIntegrationDecisionsV0(index int, suffix string, choice string) []WebNuevaAppIntakeDecisionV0 {
	prefix := "integraciones." + strconv.Itoa(index) + "."
	switch suffix {
	case "tipo":
		switch choice {
		case "calendar_enterprise_neutral", "calendar_google_workspace", "calendar_microsoft_365", "calendar_caldav":
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "tipo", Value: "calendar"},
				{Field: prefix + "nombre", Value: wizardCalendarNameV0(choice)},
				{Field: prefix + "proposito", Value: "Sincronizar eventos, citas o agenda con adaptador autorizado."},
				{Field: prefix + "auth", Value: "oauth gestionado"},
				{Field: prefix + "criticidad", Value: "media"},
				{Field: prefix + "requerido", Value: "true"},
				{Field: prefix + "restricciones", Values: wizardCalendarRestrictionsV0(choice)},
			}
		case "payments_capability":
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "tipo", Value: "payments"},
				{Field: prefix + "nombre", Value: "pagos"},
				{Field: prefix + "proposito", Value: "Procesar pagos mediante adaptador autorizado."},
				{Field: prefix + "auth", Value: "proveedor gestionado"},
				{Field: prefix + "criticidad", Value: "alta"},
			}
		case "ecommerce_capability", "ecommerce_payments_sync":
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "tipo", Value: "ecommerce"},
				{Field: prefix + "nombre", Value: "catalogo, carrito y pagos"},
				{Field: prefix + "proposito", Value: "Gobernar catalogo, carrito, pagos, stock y facturacion mediante adaptadores configurables."},
				{Field: prefix + "auth", Value: "proveedor gestionado"},
				{Field: prefix + "criticidad", Value: "alta"},
				{Field: prefix + "requerido", Value: "true"},
			}
		case "maps_public_sources":
			return guidedMapDecisionsV0(false)
		default:
			if integrationType, ok := wizardDomainIntegrationTypeFromChoiceV0(choice); ok {
				return []WebNuevaAppIntakeDecisionV0{
					{Field: prefix + "tipo", Value: integrationType},
					{Field: prefix + "nombre", Value: wizardDomainIntegrationNameV0(integrationType)},
					{Field: prefix + "proposito", Value: wizardDomainIntegrationPurposeV0(integrationType, choice)},
					{Field: prefix + "auth", Value: "adaptador configurable"},
					{Field: prefix + "criticidad", Value: wizardDomainIntegrationCriticalityV0(integrationType)},
					{Field: prefix + "requerido", Value: "true"},
				}
			}
			return []WebNuevaAppIntakeDecisionV0{{Field: prefix + "tipo", Value: choice}}
		}
	case "auth":
		switch choice {
		case "publica_criticidad_baja":
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "auth", Value: "publica"},
				{Field: prefix + "criticidad", Value: "baja"},
			}
		case "oauth_criticidad_alta":
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "auth", Value: "oauth gestionado"},
				{Field: prefix + "criticidad", Value: "alta"},
			}
		default:
			return []WebNuevaAppIntakeDecisionV0{
				{Field: prefix + "auth", Value: "auth simple"},
				{Field: prefix + "criticidad", Value: "media"},
			}
		}
	default:
		return []WebNuevaAppIntakeDecisionV0{{Field: prefix + suffix, Value: choice}}
	}
}

func wizardDomainIntegrationTypeFromChoiceV0(choice string) (string, bool) {
	for _, suffix := range []string{"_capability", "_sync"} {
		if strings.HasSuffix(choice, suffix) {
			integrationType := strings.TrimSuffix(choice, suffix)
			if trimV0(integrationType) != "" {
				return integrationType, true
			}
		}
	}
	return "", false
}

func wizardDomainIntegrationNameV0(integrationType string) string {
	switch integrationType {
	case "inventory":
		return "inventario y stock"
	case "documents":
		return "documentos y wiki"
	case "project_tasks":
		return "tareas y proyectos"
	case "finance":
		return "finanzas y presupuestos"
	case "crm":
		return "contactos y CRM"
	case "booking":
		return "reservas y turnos"
	case "health":
		return "salud, fitness y habitos"
	case "education":
		return "educacion y cursos"
	case "community":
		return "comunidad y moderacion"
	case "iot":
		return "domotica, IoT y sensores"
	case "media":
		return "galeria y media"
	case "billing":
		return "facturacion y documentos legales"
	default:
		return integrationType
	}
}

func wizardDomainIntegrationPurposeV0(integrationType string, choice string) string {
	if strings.HasSuffix(choice, "_sync") {
		return "Sincronizar datos y eventos del dominio mediante conectores autorizados."
	}
	return "Cubrir las capacidades principales del dominio mediante puertos y adaptadores configurables."
}

func wizardDomainIntegrationCriticalityV0(integrationType string) string {
	switch integrationType {
	case "finance", "billing", "health", "ecommerce":
		return "alta"
	default:
		return "media"
	}
}

func wizardDataNeedDecisionsV0(choice string) []WebNuevaAppIntakeDecisionV0 {
	switch choice {
	case "gestion_con_persistencia":
		return []WebNuevaAppIntakeDecisionV0{
			{Field: "datos.db_required", Value: "true"},
			{Field: "datos.necesidad_funcional", Value: "Gestionar datos propios con persistencia y trazabilidad."},
		}
	case "solo_consulta":
		return []WebNuevaAppIntakeDecisionV0{
			{Field: "datos.db_required", Value: "false"},
			{Field: "datos.necesidad_funcional", Value: "Consultar datos existentes desde fuentes autorizadas."},
		}
	default:
		return []WebNuevaAppIntakeDecisionV0{
			{Field: "datos.db_required", Value: "false"},
			{Field: "datos.necesidad_funcional", Value: "No requiere persistencia propia inicialmente."},
		}
	}
}

func wizardTechnicalPersistenceDecisionsV0(field string, choice string) []WebNuevaAppIntakeDecisionV0 {
	index, _, ok := indexedNuevaAppDecisionFieldV0(field, "datos.storage")
	if !ok {
		index = 0
	}
	prefix := "datos.storage." + strconv.Itoa(index) + "."
	storageType := "relacional"
	switch choice {
	case "kv_embebido_backups_restore":
		storageType = "clave_valor"
	case "redis_cache_jobs_cron":
		storageType = "clave_valor_cache"
	}
	return []WebNuevaAppIntakeDecisionV0{
		{Field: prefix + "tipo", Value: storageType},
		{Field: prefix + "proposito", Value: wizardTechnicalPersistencePurposeV0(choice)},
		{Field: prefix + "requerido", Value: "true"},
		{Field: prefix + "restricciones", Values: wizardTechnicalPersistenceRestrictionsV0(choice)},
	}
}

func wizardTechnicalPersistencePurposeV0(choice string) string {
	switch choice {
	case "postgresql_migraciones_backups_restore":
		return "Persistir datos compartidos con PostgreSQL, migraciones versionadas y restore probado."
	case "mysql_migraciones_backups_restore":
		return "Persistir datos compartidos con MySQL, migraciones versionadas y restore probado."
	case "kv_embebido_backups_restore":
		return "Persistir estado clave-valor embebido con backups y restore probado."
	case "redis_cache_jobs_cron":
		return "Cubrir cache, tareas en segundo plano y cron con adaptador gestionado."
	default:
		return "Persistir datos locales con SQLite, migraciones, backups diarios y restore probado."
	}
}

func wizardTechnicalPersistenceRestrictionsV0(choice string) []string {
	switch choice {
	case "postgresql_migraciones_backups_restore":
		return []string{"motor preferido: PostgreSQL", "migraciones versionadas con rollback", "backup automatico y restore probado"}
	case "mysql_migraciones_backups_restore":
		return []string{"motor preferido: MySQL", "migraciones versionadas con rollback", "backup automatico y restore probado"}
	case "kv_embebido_backups_restore":
		return []string{"motor preferido: KV embebido", "backup automatico y restore probado"}
	case "redis_cache_jobs_cron":
		return []string{"cache Redis o equivalente por adaptador", "jobs y cron gobernados", "backup/restore si hay estado durable"}
	default:
		return []string{"motor preferido: SQLite", "migraciones versionadas con rollback", "backup diario y restore semanal probado"}
	}
}

func wizardAPIPurposeV0(choice string) string {
	switch choice {
	case "grpc_contracts":
		return "Exponer contratos gRPC versionados con limites, idempotencia y documentacion tecnica."
	case "graphql_schema":
		return "Exponer schema GraphQL versionado con limites, idempotencia y documentacion tecnica."
	case "webhooks_firmados_reintentos":
		return "Emitir webhooks firmados con reintentos, idempotencia y trazabilidad."
	default:
		return "Exponer REST versionada con OpenAPI, rate limiting e idempotencia en mutaciones."
	}
}

func wizardAccessPurposeV0(choice string) string {
	switch choice {
	case "acl_fina":
		return "Gestionar permisos finos por recurso mediante adaptador de autorizacion."
	case "multi_tenant":
		return "Aislar clientes o grupos dentro de la misma app con permisos separados."
	default:
		return "Gestionar roles simples, sesiones y MFA opcional para usuarios autenticados."
	}
}

func wizardIdentityPurposeV0(choice string) string {
	switch choice {
	case "ldap_bind":
		return "Integrar entrada con directorio LDAP o Active Directory clasico."
	case "saml_sso":
		return "Integrar SSO corporativo SAML manteniendo proveedor como adaptador."
	case "scim_groups":
		return "Sincronizar usuarios y grupos corporativos para roles de la app."
	default:
		return "Integrar SSO OIDC corporativo sin custodiar contrasenas en la app."
	}
}

func wizardPlatformValuesV0(choice string) []string {
	switch choice {
	case "web_mobile":
		return []string{"web", "mobile"}
	case "mobile_ios_android":
		return []string{"mobile", "ios", "android"}
	case "mobile_ios":
		return []string{"mobile", "ios"}
	case "mobile_android":
		return []string{"mobile", "android"}
	default:
		return compactStringsV0(strings.Split(choice, "_"))
	}
}

func wizardAudienceValuesV0(choice string) []string {
	switch choice {
	case "compartir_auth_simple":
		return []string{"usuarios autenticados", "uso compartido"}
	case "equipo_pequeno":
		return []string{"equipo pequeno"}
	case "profesionales_internos":
		return []string{"profesionales internos"}
	case "clientes_invitados":
		return []string{"clientes invitados"}
	default:
		return []string{choice}
	}
}

func wizardStoragePurposeV0(choice string) string {
	switch choice {
	case "mixta":
		return "Combinar datos propios y fuentes externas con adaptadores autorizados."
	case "documental":
		return "Guardar documentos o registros flexibles."
	case "sin_preferencia":
		return "Dejar la eleccion al generador segun el modelo final."
	default:
		return "Persistir entidades principales y relaciones del dominio."
	}
}

func wizardCalendarNameV0(choice string) string {
	switch choice {
	case "calendar_google_workspace":
		return "Google Calendar / Workspace"
	case "calendar_microsoft_365":
		return "Microsoft 365 Calendar"
	case "calendar_caldav":
		return "CalDAV"
	default:
		return "agenda de empresa"
	}
}

func wizardCalendarRestrictionsV0(choice string) []string {
	switch choice {
	case "calendar_google_workspace":
		return []string{"preferir Google Calendar si el adaptador autorizado lo soporta"}
	case "calendar_microsoft_365":
		return []string{"preferir Microsoft 365 si el adaptador autorizado lo soporta"}
	case "calendar_caldav":
		return []string{"preferir CalDAV si el adaptador autorizado lo soporta"}
	default:
		return []string{"mantener proveedor de calendario como adaptador configurable"}
	}
}

func webNuevaAppWizardHighImportanceQuestionsV0(questions []WizardQuestionV0) []WizardQuestionV0 {
	out := make([]WizardQuestionV0, 0, len(questions))
	for _, question := range questions {
		if question.Importance == WizardImportanceAltaV0 {
			out = append(out, question)
		}
	}
	if out == nil {
		return []WizardQuestionV0{}
	}
	return out
}
