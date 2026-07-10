package orquestaappdirectorservice

import (
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	startAppDirectorGoalNewAppPhasePolicyRefV0       = "new_app_phase_timeout_policy_v0:brainstorming_arquitectura_requires_early_artifact"
	startAppDirectorGoalNewAppPhasePolicyCriterionV0 = "Politica de fase: brainstorming_arquitectura debe producir un artefacto, plan verificable o bloqueo terminal antes de ampliar contexto; no basta un receipt inicial invalido."
)

func buildStartAppDirectorGoalWorkSpecV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) orquestagoal.GoalWorkSpecV0 {
	token := appDirectorGoalSafeTokenV0(prepared.Run.RunID, request.RunRef, spec.RequestID, spec.App.Slug)
	writeSetPath := "generated-apps/" + appDirectorGoalSafeTokenV0(spec.App.Slug, spec.RequestID, token)
	evidenceRef := "evidence-ref-app-director-goal-first-v0"
	contextRefs := []orquestagoal.GoalContextRefV0{
		{Kind: "request", Ref: spec.RequestID, Purpose: "Solicitud publica normalizada", Required: true},
		{Kind: "run", Ref: prepared.Run.RunID, Purpose: "Run persistida por Orquesta", Required: true},
		{Kind: "app_spec", Ref: spec.SpecID, Purpose: "Contrato AppSpecV0 validado", Required: true},
		{Kind: "phase_policy", Ref: startAppDirectorGoalNewAppPhasePolicyRefV0, Purpose: "Politica de progreso temprano para evitar timeout inicial sin artefactos.", Required: true},
	}
	contextRefs = append(contextRefs, startAppDirectorGoalTechnicalContextRefsV0(spec)...)
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:         "goal-ref-app-director-" + token,
		RequestRef:      spec.RequestID,
		RunRef:          prepared.Run.RunID,
		ProjectRef:      firstStartAppDirectorGoalValueV0(request.ProjectRef, spec.ProjectSource.ProjectRef, spec.SpecID),
		WorkKind:        "new_app",
		WorkProfileKind: "implementation",
		Objective:       startAppDirectorGoalObjectiveV0(spec),
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs:     contextRefs,
		RuleRefs:        startAppDirectorGoalRuleRefsV0(spec),
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    writeSetPath,
			Purpose: "Arbol fuente de la app generada desde el contrato AppSpecV0.",
		}},
		RequiredTests:      startAppDirectorGoalRequiredTestsV0(token, writeSetPath, spec),
		AcceptanceCriteria: startAppDirectorGoalAcceptanceCriteriaV0(spec, writeSetPath),
		ArtifactContracts:  startAppDirectorGoalArtifactContractsV0(token, spec),
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{evidenceRef},
			prepared.EvidenceRefs...,
		)),
		Budget: startAppDirectorGoalBudgetV0(spec),
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests: true,
			RequireArtifacts:     true,
			RequiredEvidenceRefs: []string{evidenceRef},
		},
		ReworkPolicy: startAppDirectorGoalReworkPolicyV0(),
	})
}

func startAppDirectorGoalTechnicalContextRefsV0(spec orquestafactory.AppSpecV0) []orquestagoal.GoalContextRefV0 {
	var refs []orquestagoal.GoalContextRefV0
	if language := strings.TrimSpace(spec.Technical.Language); language != "" {
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:     "technical_stack",
			Ref:      "technical-language-" + appDirectorGoalSafeTokenV0(language),
			Purpose:  "Lenguaje solicitado por AppSpecV0: " + language,
			Required: true,
		})
	}
	if framework := strings.TrimSpace(spec.Technical.Framework); framework != "" {
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:     "technical_stack",
			Ref:      "technical-framework-" + appDirectorGoalSafeTokenV0(framework),
			Purpose:  "Framework solicitado por AppSpecV0: " + framework,
			Required: true,
		})
	}
	return refs
}

func startAppDirectorGoalRuleRefsV0(spec orquestafactory.AppSpecV0) []orquestagoal.GoalRuleRefV0 {
	refs := []orquestagoal.GoalRuleRefV0{
		{Kind: "repo", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
		{Kind: "docs", Ref: "docs/orquesta_goal_first_codex_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
		{Kind: "contract", Ref: "modulos/orquesta-factory/docs/contratos.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
		{Kind: "contract", Ref: "modulos/orquesta-web/docs/guia_nueva_app_opciones_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
	}
	if language := strings.TrimSpace(spec.Technical.Language); language != "" {
		refs = append(refs, orquestagoal.GoalRuleRefV0{
			Kind:        "technical_constraint",
			Ref:         "technical_constraint:language=" + appDirectorGoalSafeTokenV0(language),
			Enforcement: orquestagoal.GoalRuleEnforcementHardV0,
		})
	}
	if framework := strings.TrimSpace(spec.Technical.Framework); framework != "" {
		refs = append(refs, orquestagoal.GoalRuleRefV0{
			Kind:        "technical_constraint",
			Ref:         "technical_constraint:framework=" + appDirectorGoalSafeTokenV0(framework),
			Enforcement: orquestagoal.GoalRuleEnforcementHardV0,
		})
	}
	return refs
}

func startAppDirectorGoalArtifactContractsV0(token string, spec orquestafactory.AppSpecV0) []orquestagoal.GoalArtifactContractV0 {
	contracts := []orquestagoal.GoalArtifactContractV0{
		{ArtifactRef: "artifact-ref-" + token + "-source", ArtifactType: "source_tree", Required: true},
		{ArtifactRef: "artifact-ref-" + token + "-handoff", ArtifactType: "handoff_report", Required: true},
	}
	if appDirectorGoalTechnicalStackDeclaredV0(spec) {
		contracts = append(contracts, orquestagoal.GoalArtifactContractV0{
			ArtifactRef:  "artifact-ref-" + token + "-technical-stack",
			ArtifactType: "technical_stack_manifest",
			Required:     true,
		})
	}
	return contracts
}

func startAppDirectorGoalObjectiveV0(spec orquestafactory.AppSpecV0) string {
	return "Construir la app " + strings.TrimSpace(spec.App.Nombre) + " hasta dejar un arbol fuente verificable y documentado. Objetivo funcional: " + strings.TrimSpace(spec.App.Objetivo)
}

func startAppDirectorGoalAcceptanceCriteriaV0(spec orquestafactory.AppSpecV0, writeSetPath string) []string {
	return compactStartAppDirectorStringsV0([]string{
		"Crear la app bajo el write-set autorizado " + writeSetPath + ".",
		"Nombre: " + spec.App.Nombre + ". Tipo: " + spec.App.TipoApp + ". Plataformas: " + startAppDirectorGoalJoinV0(spec.Platforms) + ".",
		"Arquitectura solicitada: " + spec.Architecture.Patron + ". Mantener dominio/aplicacion, puertos, adaptadores y bootstrap separados si el patron no lo contradice.",
		"Stack tecnico solicitado: lenguaje=" + startAppDirectorGoalValueOrDefaultV0(spec.Technical.Language) + "; framework=" + startAppDirectorGoalValueOrDefaultV0(spec.Technical.Framework) + "; restricciones=" + startAppDirectorGoalJoinV0(spec.Technical.Restrictions) + "; preferencias=" + startAppDirectorGoalJoinV0(spec.Technical.Preferences) + ". Si lenguaje o framework estan declarados, son contrato verificable: entregar manifiesto tecnico y artefactos coherentes.",
		"Objetivo funcional: " + spec.App.Objetivo,
		"Datos: persistencia=" + startAppDirectorGoalBoolV0(spec.Data.PersistenceRequired) + "; necesidades=" + startAppDirectorGoalJoinV0(spec.Data.Needs) + "; sensibilidad=" + spec.Data.Sensitivity + "; storage=" + startAppDirectorGoalStorageSummaryV0(spec.Data.Storage) + "; tipos=" + startAppDirectorGoalDataTypesSummaryV0(spec.Data.Types) + ".",
		"Integraciones requeridas: " + startAppDirectorGoalConnectorsSummaryV0(spec.Connectors.Required) + ". Integraciones opcionales: " + startAppDirectorGoalConnectorsSummaryV0(spec.Connectors.Optional) + ".",
		"Calidad: pruebas=" + spec.Quality.Tests + "; accesibilidad=" + spec.Quality.Accessibility + "; opciones_accesibilidad=" + startAppDirectorGoalJoinV0(spec.Quality.AccessibilityOptions) + "; observabilidad=" + startAppDirectorGoalBoolV0(spec.Quality.Observability) + ".",
		"I18n: enabled=" + startAppDirectorGoalBoolV0(spec.I18N.Enabled) + "; default_locale=" + spec.I18N.DefaultLocale + "; locales=" + startAppDirectorGoalJoinV0(spec.I18N.Locales) + ".",
		"Documentacion: usuario=" + startAppDirectorGoalBoolV0(spec.Docs.User) + "; desarrollo=" + startAppDirectorGoalBoolV0(spec.Docs.Development) + "; sistemas=" + startAppDirectorGoalBoolV0(spec.Docs.Systems) + "; profundidad=" + spec.Docs.Depth + ". Si profundidad=profunda, entregar manuales de usuario, desarrollo y sistemas con flujos, comandos, criterios de aceptacion y operacion.",
		"Project source: kind=" + spec.ProjectSource.Kind + "; project_ref=" + spec.ProjectSource.ProjectRef + "; branch=" + spec.ProjectSource.Branch + ". No publicar rutas locales ni credenciales.",
		startAppDirectorGoalNewAppPhasePolicyCriterionV0,
		"Entregar resumen final con artefactos, pruebas ejecutadas o justificadas, decisiones pendientes y bloqueos si existen.",
	})
}

func startAppDirectorGoalRequiredTestsV0(token, writeSetPath string, spec orquestafactory.AppSpecV0) []orquestagoal.GoalRequiredTestV0 {
	testRef := "test-ref-" + token + "-generated-app"
	return []orquestagoal.GoalRequiredTestV0{{
		TestRef:    testRef,
		CommandRef: "command-ref-" + token + "-verify-generated-app",
		Command: "verificar la app generada bajo " + writeSetPath +
			" con las pruebas propias del stack elegido o una prueba local documentada",
		AcceptanceCriteria: compactStartAppDirectorStringsV0([]string{
			"El arbol fuente requerido existe bajo " + writeSetPath + ".",
			"La app cumple el contrato funcional: " + strings.TrimSpace(spec.App.Objetivo),
			"La app materializa el stack tecnico declarado: lenguaje=" + startAppDirectorGoalValueOrDefaultV0(spec.Technical.Language) + "; framework=" + startAppDirectorGoalValueOrDefaultV0(spec.Technical.Framework) + ".",
			"Las pruebas declaradas para calidad=" + strings.TrimSpace(spec.Quality.Tests) + " pasan o quedan justificadas como no aplicables con evidencia local.",
		}),
		EvidenceRefs: []string{"evidence-ref-app-director-goal-required-test-v0"},
	}}
}

func startAppDirectorGoalBudgetV0(spec orquestafactory.AppSpecV0) orquestagoal.GoalBudgetV0 {
	maxSubgoals := 6
	if strings.TrimSpace(spec.AgentPreferences.Autonomy) == "alta" || strings.TrimSpace(spec.Quality.Tests) == "alta" {
		maxSubgoals = 10
	}
	return orquestagoal.GoalBudgetV0{MaxRuntimeSeconds: 600, MaxSubgoals: maxSubgoals, MaxReworkGoals: 2}
}

func startAppDirectorGoalReworkPolicyV0() orquestagoal.GoalReworkPolicyV0 {
	return orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, MaxReworkGoals: 2, PreserveArtifacts: true}
}

func startAppDirectorGoalJoinV0(values []string) string {
	values = compactStartAppDirectorStringsV0(values)
	if len(values) == 0 {
		return "sin_declarar"
	}
	return strings.Join(values, ", ")
}

func startAppDirectorGoalBoolV0(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func startAppDirectorGoalValueOrDefaultV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "sin_declarar"
}

func appDirectorGoalTechnicalStackDeclaredV0(spec orquestafactory.AppSpecV0) bool {
	return strings.TrimSpace(spec.Technical.Language) != "" || strings.TrimSpace(spec.Technical.Framework) != ""
}

func startAppDirectorGoalStorageSummaryV0(values []orquestafactory.DataStorageSpecV0) string {
	if len(values) == 0 {
		return "sin_declarar"
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value.Tipo)
		if value.Proposito != "" {
			item += ":" + strings.TrimSpace(value.Proposito)
		}
		if value.Requerido {
			item += ":requerido"
		}
		out = append(out, item)
	}
	return startAppDirectorGoalJoinV0(out)
}

func startAppDirectorGoalDataTypesSummaryV0(values []orquestafactory.DataTypeSpecV0) string {
	if len(values) == 0 {
		return "sin_declarar"
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value.Nombre)
		if value.Sensibilidad != "" {
			item += ":sensibilidad=" + strings.TrimSpace(value.Sensibilidad)
		}
		if value.Volumen != "" {
			item += ":volumen=" + strings.TrimSpace(value.Volumen)
		}
		out = append(out, item)
	}
	return startAppDirectorGoalJoinV0(out)
}

func startAppDirectorGoalConnectorsSummaryV0(values []orquestafactory.ConnectorSpecV0) string {
	if len(values) == 0 {
		return "sin_declarar"
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value.Nombre)
		if value.Proposito != "" {
			item += ":" + strings.TrimSpace(value.Proposito)
		}
		out = append(out, item)
	}
	return startAppDirectorGoalJoinV0(out)
}
