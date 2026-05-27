package orquestaweb

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
	case "deploy.target":
		form.Deploy.Target = decision.Value
	case "datos.necesidad_funcional":
		form.Datos.NecesidadFuncional = decision.Value
	case "datos.tipos_datos":
		form.Datos.TiposDatos = decision.Values
	case "datos.sensibilidad":
		form.Datos.Sensibilidad = decision.Value
	case "calidad.pruebas":
		form.Calidad.Pruebas = decision.Value
	case "calidad.accesibilidad":
		form.Calidad.Accesibilidad = decision.Value
	case "agentes.autonomia":
		form.Agentes.Autonomia = decision.Value
	case "i18n.default_locale":
		form.I18N.DefaultLocale = decision.Value
	case "project_source.project_ref":
		form.ProjectSource.ProjectRef = decision.Value
	}
	return form
}
