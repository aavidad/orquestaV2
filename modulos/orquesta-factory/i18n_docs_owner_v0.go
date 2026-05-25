package orquestafactory

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

func BuildI18nDocsPlanFromAppSpecV0(spec AppSpecV0) (i18ndocs.AppI18nDocsPlanV0, []i18ndocs.I18nDocsValidationIssueV0) {
	seed := i18nDocsPlanSeedFromAppSpecV0(spec)
	plan := i18ndocs.BuildAppI18nDocsPlanV0(seed)
	return plan, i18ndocs.ValidateAppI18nDocsPlanV0(plan)
}

func I18nDocsActiveOwnerForAppSpecV0(spec AppSpecV0) (i18ndocs.ActiveI18nDocsCompositionOwnerV0, []i18ndocs.I18nDocsValidationIssueV0) {
	return i18ndocs.BuildActiveI18nDocsCompositionOwnerV0(i18nDocsPlanSeedFromAppSpecV0(spec))
}

func i18nDocsPlanSeedFromAppSpecV0(spec AppSpecV0) i18ndocs.PlanSeedV0 {
	return i18ndocs.PlanSeedV0{
		AppIDHint:          spec.App.Slug,
		AppTitle:           spec.App.Nombre,
		PrimaryActionLabel: "Crear app",
		UILocales:          spec.I18N.Locales,
		DocsLocales:        spec.Docs.Locales,
		UIDefaultLocale:    spec.I18N.DefaultLocale,
		DocsDefaultLocale:  spec.I18N.DefaultLocale,
	}
}
