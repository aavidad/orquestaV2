package orquestaappdirectorintake

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func normalizeAppDirectorIntakeWizardRequestV0(
	request AppDirectorIntakeWizardRequestV0,
) AppDirectorIntakeWizardRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.Source = strings.TrimSpace(request.Source)
	request.Locale = strings.TrimSpace(request.Locale)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Draft = normalizeAppDirectorIntakeWizardDraftV0(request.Draft)
	if request.RequestRef == "" {
		request.RequestRef = request.Draft.RequestID
	}
	if request.Draft.RequestID == "" {
		request.Draft.RequestID = request.RequestRef
	}
	if request.Draft.Source == "" {
		request.Draft.Source = request.Source
	}
	if request.Draft.Locale == "" {
		request.Draft.Locale = request.Locale
	}
	if request.RunRef == "" && request.RequestRef != "" {
		request.RunRef = "run-" + safeDirectorIntakeRefPartV0(request.RequestRef)
	}
	if request.ProjectRef == "" && request.Draft.Nombre != "" {
		request.ProjectRef = "project-" + safeDirectorIntakeRefPartV0(request.Draft.Nombre)
	}
	if request.ProjectRef == "" && request.RequestRef != "" {
		request.ProjectRef = "project-" + safeDirectorIntakeRefPartV0(request.RequestRef)
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestRef
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-intake-wizard"
	}
	return request
}

func normalizeAppDirectorIntakeWizardDraftV0(
	draft orquestafactory.AppSpecRequestV0,
) orquestafactory.AppSpecRequestV0 {
	draft.SchemaVersion = strings.TrimSpace(draft.SchemaVersion)
	if draft.SchemaVersion == "" {
		draft.SchemaVersion = orquestafactory.AppSpecRequestSchemaV0
	}
	draft.RequestID = strings.TrimSpace(draft.RequestID)
	draft.Source = strings.TrimSpace(draft.Source)
	draft.Locale = strings.TrimSpace(draft.Locale)
	draft.RequestKind = strings.TrimSpace(draft.RequestKind)
	draft.ExecutionMode = strings.TrimSpace(draft.ExecutionMode)
	draft.Nombre = strings.TrimSpace(draft.Nombre)
	draft.Objetivo = strings.TrimSpace(draft.Objetivo)
	draft.Descripcion = strings.TrimSpace(draft.Descripcion)
	draft.TipoApp = strings.TrimSpace(draft.TipoApp)
	draft.UsuariosObjetivo = compactDirectorIntakeStringsV0(draft.UsuariosObjetivo)
	draft.Plataformas = compactDirectorIntakeStringsV0(draft.Plataformas)
	draft.Restricciones = compactDirectorIntakeStringsV0(draft.Restricciones)
	draft.PreferenciasTecnicas.Lenguaje = strings.TrimSpace(draft.PreferenciasTecnicas.Lenguaje)
	draft.PreferenciasTecnicas.Framework = strings.TrimSpace(draft.PreferenciasTecnicas.Framework)
	draft.PreferenciasTecnicas.Arquitectura = strings.TrimSpace(draft.PreferenciasTecnicas.Arquitectura)
	draft.Datos.NecesidadFuncional = strings.TrimSpace(draft.Datos.NecesidadFuncional)
	draft.Calidad.Pruebas = strings.TrimSpace(draft.Calidad.Pruebas)
	draft.Calidad.Accesibilidad = strings.TrimSpace(draft.Calidad.Accesibilidad)
	draft.Agentes.Autonomia = strings.TrimSpace(draft.Agentes.Autonomia)
	return draft
}
