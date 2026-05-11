package orquestaappdirectorintake

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func applyAppDirectorIntakeWizardAnswersV0(
	draft orquestafactory.AppSpecRequestV0,
	answers []AppDirectorIntakeWizardAnswerV0,
) (orquestafactory.AppSpecRequestV0, []AppDirectorIntakeWizardIssueV0) {
	var issues []AppDirectorIntakeWizardIssueV0
	for _, answer := range answers {
		var ok bool
		draft, ok = applyAppDirectorIntakeWizardAnswerV0(draft, answer)
		if !ok {
			issues = append(issues, AppDirectorIntakeWizardIssueV0{
				Code:  ErrAppDirectorIntakeWizardFieldUnsupportedV0,
				Field: strings.TrimSpace(answer.Field),
			})
		}
	}
	return normalizeAppDirectorIntakeWizardDraftV0(draft), issues
}

func applyAppDirectorIntakeWizardAnswerV0(
	draft orquestafactory.AppSpecRequestV0,
	answer AppDirectorIntakeWizardAnswerV0,
) (orquestafactory.AppSpecRequestV0, bool) {
	field := strings.TrimSpace(answer.Field)
	value := strings.TrimSpace(answer.Value)
	values := appDirectorIntakeWizardAnswerValuesV0(answer)
	switch field {
	case "request_id":
		draft.RequestID = value
	case "source":
		draft.Source = value
	case "locale":
		draft.Locale = value
	case "request_kind":
		draft.RequestKind = value
	case "execution_mode":
		draft.ExecutionMode = value
	case "nombre":
		draft.Nombre = value
	case "objetivo":
		draft.Objetivo = value
	case "descripcion":
		draft.Descripcion = value
	case "tipo_app":
		draft.TipoApp = value
	case "usuarios_objetivo":
		draft.UsuariosObjetivo = values
	case "plataformas":
		draft.Plataformas = values
	case "preferencias_tecnicas.lenguaje":
		draft.PreferenciasTecnicas.Lenguaje = value
	case "preferencias_tecnicas.arquitectura":
		draft.PreferenciasTecnicas.Arquitectura = value
	case "datos.db_required":
		draft.Datos.DBRequired = appDirectorIntakeWizardBoolValueV0(value)
	case "datos.necesidad_funcional":
		draft.Datos.NecesidadFuncional = value
	case "calidad.pruebas":
		draft.Calidad.Pruebas = value
	case "calidad.accesibilidad":
		draft.Calidad.Accesibilidad = value
	case "agentes.autonomia":
		draft.Agentes.Autonomia = value
	default:
		return draft, false
	}
	return draft, true
}

func appDirectorIntakeWizardAnswerValuesV0(
	answer AppDirectorIntakeWizardAnswerV0,
) []string {
	if len(answer.Values) > 0 {
		return compactDirectorIntakeStringsV0(answer.Values)
	}
	return compactDirectorIntakeStringsV0(strings.Split(answer.Value, ","))
}

func appDirectorIntakeWizardBoolValueV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "si", "yes", "y":
		return true
	default:
		return false
	}
}
