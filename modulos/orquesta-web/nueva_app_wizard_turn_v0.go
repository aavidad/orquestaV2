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
	questions := WebNuevaAppWizardGapQuestionsV0(previewForm)
	specComplete := len(webNuevaAppWizardHighImportanceQuestionsV0(questions)) == 0 &&
		len(orquestafactory.ValidateAppSpecRequestV0(specPreview)) == 0
	return WizardTurnResultV0{
		SchemaVersion:       WebNuevaAppWizardTurnSchemaV0,
		SessionRef:          firstNuevaAppValueV0(session.SessionRef, session.SessionID),
		Turn:                len(session.Decisions) + 1,
		Questions:           questions,
		EngineeringDefaults: WebNuevaAppWizardEngineeringDefaultsV0(),
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
	questions := webNuevaAppWizardAllGapQuestionsV0(ApplyWebNuevaAppWizardEngineeringDefaultsV0(session.Form))
	questionsByRef := map[string]WizardQuestionV0{}
	for _, question := range questions {
		questionsByRef[question.QuestionRef] = question
	}
	var decisions []WebNuevaAppIntakeDecisionV0
	var contrasts []WizardContrastV0
	for _, answer := range answers {
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
				QuestionRef:  question.QuestionRef,
				UserChoice:   choice,
				Recommended:  recommended.Value,
				RationaleKey: recommended.RationaleKey,
			})
		}
		next := wizardDecisionsForAnswerV0(question, choice, answer.FreeText)
		for _, decision := range next {
			session = session.ApplyDecisionV0(decision)
		}
		decisions = append(decisions, next...)
	}
	session.Form = ApplyWebNuevaAppWizardEngineeringDefaultsV0(session.Form)
	session = refreshWebNuevaAppIntakeSessionV0(session)
	result := NewWebNuevaAppWizardTurnResultV0(session)
	result.Decisions = decisions
	result.Contrasts = contrasts
	return session, result
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
		case "maps_public_sources":
			return guidedMapDecisionsV0(false)
		default:
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
