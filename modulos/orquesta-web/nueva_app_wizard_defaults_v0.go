package orquestaweb

func WebNuevaAppWizardEngineeringDefaultsV0() []WizardDefaultV0 {
	return []WizardDefaultV0{
		{Area: "arquitectura", Value: "hexagonal_puertos_adaptadores", WhyKey: "nueva_app.wizard.default.architecture.why"},
		{Area: "i18n", Value: "i18n_es_en_por_catalogo", WhyKey: "nueva_app.wizard.default.i18n.why"},
		{Area: "calidad", Value: "tests_unitarios_y_arquitectura_ratchet", WhyKey: "nueva_app.wizard.default.tests.why"},
		{Area: "calidad", Value: "linters_format_y_scripts_verificacion", WhyKey: "nueva_app.wizard.default.linters.why"},
		{Area: "calidad", Value: "errores_tipados_catalogo_publico", WhyKey: "nueva_app.wizard.default.errors.why"},
		{Area: "observabilidad", Value: "logging_estructurado", WhyKey: "nueva_app.wizard.default.observability.why"},
		{Area: "docs", Value: "handoff_report_source_tree_technical_stack_manifest", WhyKey: "nueva_app.wizard.default.docs.why"},
	}
}

func ApplyWebNuevaAppWizardEngineeringDefaultsV0(form WebNuevaAppFormV0) WebNuevaAppFormV0 {
	if trimV0(form.PreferenciasTecnicas.Arquitectura) == "" {
		form.PreferenciasTecnicas.Arquitectura = "hexagonal"
	}
	form.PreferenciasTecnicas.Preferencias = appendUniqueStringsV0(
		form.PreferenciasTecnicas.Preferencias,
		"hexagonal_puertos_adaptadores",
		"i18n_es_en_por_catalogo",
		"tests_unitarios",
		"test_arquitectura_ratchet",
		"linters_format",
		"errores_tipados_catalogo_publico",
		"logging_estructurado",
		"ci_ready_scripts_verificacion",
		"documentacion_tecnica_handoff_source_tree_stack_manifest",
	)
	if form.I18N.Enabled == nil {
		form.I18N.Enabled = boolPtrV0(true)
	}
	if trimV0(form.I18N.DefaultLocale) == "" {
		form.I18N.DefaultLocale = wizardDefaultLocaleV0(form.Locale)
	}
	form.I18N.Locales = appendUniqueStringsV0(form.I18N.Locales, "es-ES", "en-US")
	if trimV0(form.Calidad.Pruebas) == "" {
		form.Calidad.Pruebas = "alta"
	}
	if form.Calidad.Observabilidad == nil {
		form.Calidad.Observabilidad = boolPtrV0(true)
	}
	if form.Documentacion.Desarrollo == nil {
		form.Documentacion.Desarrollo = boolPtrV0(true)
	}
	if form.Documentacion.Sistemas == nil {
		form.Documentacion.Sistemas = boolPtrV0(true)
	}
	if trimV0(form.Documentacion.Profundidad) == "" {
		form.Documentacion.Profundidad = "normal"
	}
	form.Documentacion.Locales = appendUniqueStringsV0(form.Documentacion.Locales, "es-ES", "en-US")
	return form
}

func wizardDefaultLocaleV0(locale string) string {
	switch normalizeGuidedNeedV0(locale) {
	case "en", "en-us", "en_us":
		return "en-US"
	default:
		return "es-ES"
	}
}

func boolPtrV0(value bool) *bool {
	return &value
}

func appendUniqueStringsV0(values []string, additions ...string) []string {
	return compactStringsV0(append(append([]string{}, values...), additions...))
}
