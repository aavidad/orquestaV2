package orquestaweb

import (
	"strconv"
	"strings"
)

func normalizeWebNuevaAppIntakeDecisionV0(decision WebNuevaAppIntakeDecisionV0) WebNuevaAppIntakeDecisionV0 {
	return WebNuevaAppIntakeDecisionV0{
		Field:  trimV0(decision.Field),
		Value:  trimV0(decision.Value),
		Values: compactStringsV0(decision.Values),
	}
}

func applyWebNuevaAppIntakeDecisionToFormV0(
	form WebNuevaAppFormV0,
	decision WebNuevaAppIntakeDecisionV0,
) WebNuevaAppFormV0 {
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(decision.Field, "integraciones"); ok {
		form.Integraciones = ensureNuevaAppConnectorSlotV0(form.Integraciones, index)
		switch suffix {
		case "tipo":
			form.Integraciones[index].Tipo = decision.Value
		case "nombre":
			form.Integraciones[index].Nombre = decision.Value
		case "proposito":
			form.Integraciones[index].Proposito = decision.Value
		case "direccion":
			form.Integraciones[index].Direccion = decision.Value
		case "auth":
			form.Integraciones[index].Auth = decision.Value
		case "data_scope":
			form.Integraciones[index].DataScope = decision.Value
		case "criticidad":
			form.Integraciones[index].Criticidad = decision.Value
		case "requerido":
			form.Integraciones[index].Requerido = decision.Value == "true"
		case "restricciones":
			form.Integraciones[index].Restricciones = decision.Values
		}
		return form
	}
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(decision.Field, "datos.tipos_detallados"); ok {
		form.Datos.TiposDetallados = ensureNuevaAppDataTypeSlotV0(form.Datos.TiposDetallados, index)
		switch suffix {
		case "nombre":
			form.Datos.TiposDetallados[index].Nombre = decision.Value
		case "proposito":
			form.Datos.TiposDetallados[index].Proposito = decision.Value
		case "sensibilidad":
			form.Datos.TiposDetallados[index].Sensibilidad = decision.Value
		case "retencion":
			form.Datos.TiposDetallados[index].Retencion = decision.Value
		case "volumen":
			form.Datos.TiposDetallados[index].Volumen = decision.Value
		case "restricciones":
			form.Datos.TiposDetallados[index].Restricciones = decision.Values
		}
		return form
	}
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(decision.Field, "datos.fuentes"); ok {
		form.Datos.Fuentes = ensureNuevaAppDataSourceSlotV0(form.Datos.Fuentes, index)
		switch suffix {
		case "nombre":
			form.Datos.Fuentes[index].Nombre = decision.Value
		case "tipo":
			form.Datos.Fuentes[index].Tipo = decision.Value
		case "proposito":
			form.Datos.Fuentes[index].Proposito = decision.Value
		case "owner":
			form.Datos.Fuentes[index].Owner = decision.Value
		case "frecuencia":
			form.Datos.Fuentes[index].Frecuencia = decision.Value
		case "restricciones":
			form.Datos.Fuentes[index].Restricciones = decision.Values
		}
		return form
	}
	if index, suffix, ok := indexedNuevaAppDecisionFieldV0(decision.Field, "datos.storage"); ok {
		form.Datos.Storage = ensureNuevaAppDataStorageSlotV0(form.Datos.Storage, index)
		switch suffix {
		case "tipo":
			form.Datos.Storage[index].Tipo = decision.Value
		case "proposito":
			form.Datos.Storage[index].Proposito = decision.Value
		case "requerido":
			form.Datos.Storage[index].Requerido = decision.Value == "true"
		case "restricciones":
			form.Datos.Storage[index].Restricciones = decision.Values
		}
		return form
	}
	switch decision.Field {
	case "locale":
		form.Locale = decision.Value
	case "nombre":
		form.Nombre = decision.Value
	case "objetivo":
		form.Objetivo = decision.Value
	case "tipo_app":
		form.TipoApp = decision.Value
	case "request_kind":
		form.RequestKind = decision.Value
	case "execution_mode":
		form.ExecutionMode = decision.Value
	case "descripcion":
		form.Descripcion = decision.Value
	case "usuarios_objetivo":
		form.UsuariosObjetivo = decision.Values
	case "plataformas":
		form.Plataformas = decision.Values
	case "restricciones":
		form.Restricciones = decision.Values
	case "preferencias_tecnicas.arquitectura":
		form.PreferenciasTecnicas.Arquitectura = decision.Value
	case "preferencias_tecnicas.lenguaje":
		form.PreferenciasTecnicas.Lenguaje = decision.Value
	case "preferencias_tecnicas.framework":
		form.PreferenciasTecnicas.Framework = decision.Value
	case "preferencias_tecnicas.restricciones":
		form.PreferenciasTecnicas.Restricciones = nuevaAppDecisionValuesOrScalarV0(decision)
	case "preferencias_tecnicas.preferencias":
		form.PreferenciasTecnicas.Preferencias = nuevaAppDecisionValuesOrScalarV0(decision)
	case "project_source.kind":
		form.ProjectSource.Kind = decision.Value
	case "project_source.git_url":
		form.ProjectSource.GitURL = decision.Value
	case "project_source.branch":
		form.ProjectSource.Branch = decision.Value
	case "project_source.local_path":
		form.ProjectSource.LocalPath = decision.Value
	case "deploy.target":
		form.Deploy.Target = decision.Value
	case "deploy.restricciones":
		form.Deploy.Restricciones = decision.Values
	case "datos.db_required":
		form.Datos.DBRequired = decision.Value == "true"
	case "datos.necesidad_funcional":
		form.Datos.NecesidadFuncional = decision.Value
	case "datos.tipos_datos":
		form.Datos.TiposDatos = decision.Values
	case "datos.sensibilidad":
		form.Datos.Sensibilidad = decision.Value
	case "datos.retencion":
		form.Datos.Retencion = decision.Value
	case "datos.operacion.criticidad":
		form.Datos.Operacion.Criticidad = decision.Value
	case "datos.operacion.disponibilidad":
		form.Datos.Operacion.Disponibilidad = decision.Value
	case "datos.operacion.rpo":
		form.Datos.Operacion.RPO = decision.Value
	case "datos.operacion.rto":
		form.Datos.Operacion.RTO = decision.Value
	case "datos.operacion.auditoria":
		form.Datos.Operacion.Auditoria = decision.Value == "true"
	case "datos.operacion.restricciones":
		form.Datos.Operacion.Restricciones = decision.Values
	case "calidad.pruebas":
		form.Calidad.Pruebas = decision.Value
	case "calidad.accesibilidad":
		form.Calidad.Accesibilidad = decision.Value
	case "calidad.accesibilidad_opciones":
		form.Calidad.AccesibilidadOpciones = decision.Values
	case "calidad.compliance":
		form.Calidad.Compliance = nuevaAppDecisionValuesOrScalarV0(decision)
	case "calidad.observabilidad":
		form.Calidad.Observabilidad = nuevaAppDecisionOptionalBoolV0(decision)
	case "documentacion.usuario":
		form.Documentacion.Usuario = nuevaAppDecisionOptionalBoolV0(decision)
	case "documentacion.desarrollo":
		form.Documentacion.Desarrollo = nuevaAppDecisionOptionalBoolV0(decision)
	case "documentacion.sistemas":
		form.Documentacion.Sistemas = nuevaAppDecisionOptionalBoolV0(decision)
	case "documentacion.profundidad":
		form.Documentacion.Profundidad = decision.Value
	case "documentacion.locales":
		form.Documentacion.Locales = nuevaAppDecisionValuesOrScalarV0(decision)
	case "agentes.revision_humana":
		form.Agentes.RevisionHumana = nuevaAppDecisionOptionalBoolV0(decision)
	case "agentes.autonomia":
		form.Agentes.Autonomia = decision.Value
	case "agentes.preferencias":
		form.Agentes.Preferencias = decision.Values
	case "i18n.enabled":
		form.I18N.Enabled = nuevaAppDecisionOptionalBoolV0(decision)
	case "i18n.default_locale":
		form.I18N.DefaultLocale = decision.Value
	case "i18n.locales":
		form.I18N.Locales = nuevaAppDecisionValuesOrScalarV0(decision)
	case "i18n.justificacion":
		form.I18N.Justificacion = decision.Value
	case "project_source.project_ref":
		form.ProjectSource.ProjectRef = decision.Value
	}
	return form
}

func nuevaAppDecisionValuesOrScalarV0(decision WebNuevaAppIntakeDecisionV0) []string {
	if len(decision.Values) > 0 {
		return decision.Values
	}
	if trimV0(decision.Value) == "" {
		return []string{}
	}
	return compactStringsV0(strings.Split(decision.Value, ","))
}

func nuevaAppDecisionOptionalBoolV0(decision WebNuevaAppIntakeDecisionV0) *bool {
	switch strings.ToLower(trimV0(decision.Value)) {
	case "true", "si", "s\u00ed", "yes", "1":
		value := true
		return &value
	case "false", "no", "0":
		value := false
		return &value
	default:
		return nil
	}
}

func indexedNuevaAppDecisionFieldV0(field, prefix string) (int, string, bool) {
	rest := strings.TrimPrefix(field, prefix+".")
	if rest == field {
		return 0, "", false
	}
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 0 {
		return 0, "", false
	}
	return index, parts[1], true
}

func ensureNuevaAppConnectorSlotV0(values []WebNuevaAppConnectorFormV0, index int) []WebNuevaAppConnectorFormV0 {
	for len(values) <= index {
		values = append(values, WebNuevaAppConnectorFormV0{})
	}
	return values
}

func ensureNuevaAppDataTypeSlotV0(values []WebNuevaAppDataTypeFormV0, index int) []WebNuevaAppDataTypeFormV0 {
	for len(values) <= index {
		values = append(values, WebNuevaAppDataTypeFormV0{})
	}
	return values
}

func ensureNuevaAppDataStorageSlotV0(values []WebNuevaAppDataStorageFormV0, index int) []WebNuevaAppDataStorageFormV0 {
	for len(values) <= index {
		values = append(values, WebNuevaAppDataStorageFormV0{})
	}
	return values
}

func ensureNuevaAppDataSourceSlotV0(values []WebNuevaAppDataSourceFormV0, index int) []WebNuevaAppDataSourceFormV0 {
	for len(values) <= index {
		values = append(values, WebNuevaAppDataSourceFormV0{})
	}
	return values
}
