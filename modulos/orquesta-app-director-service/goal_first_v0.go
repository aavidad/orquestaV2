package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func startAppDirectorGoalFirstV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	ports StartAppDirectorPortsV0,
) (StartAppDirectorResultV0, bool, error) {
	if ports.GoalLauncher == nil {
		return StartAppDirectorResultV0{}, false, nil
	}
	goalSpec := buildStartAppDirectorGoalWorkSpecV0(request, spec, prepared)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(goalSpec); len(issues) > 0 {
		return StartAppDirectorResultV0{}, false, AppDirectorServiceIssueV0{Field: "goal_spec"}
	}
	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(ctx, goalSpec)
	if err != nil {
		return StartAppDirectorResultV0{}, false, err
	}
	if ports.GoalStateStore != nil {
		if err := ports.GoalStateStore.SaveGoalWorkStateV0(
			ctx,
			newStartAppDirectorGoalStateV0(prepared.Run.RunID, goalSpec, receipt, prepared.EvidenceRefs),
		); err != nil {
			return StartAppDirectorResultV0{}, false, err
		}
	}
	return startAppDirectorGoalFirstResultV0(request, spec, prepared, receipt), true, nil
}

func ObserveAppDirectorGoalV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	ports StartAppDirectorPortsV0,
) (ObserveAppDirectorGoalResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.RunRef = strings.TrimSpace(request.RunRef)
	if request.RunRef == "" {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	if ports.GoalStateStore == nil {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "ports.goal_state_store"}
	}
	if ports.GoalObserver == nil {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "ports.goal_observer"}
	}
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	state, err = NewAppDirectorGoalStateV0(state)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	result, err := ports.GoalObserver.ObserveGoalWorkV0(ctx, orquestagoal.GoalObservationRequestV0{
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
	})
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if issues := orquestagoal.ValidateGoalWorkResultV0(result); len(issues) > 0 {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "goal_result"}
	}
	state.LastResult = &result
	state.Status = result.Status
	state.EvidenceRefs = compactStartAppDirectorStringsV0(append(state.EvidenceRefs, result.EvidenceRefs...))
	closure := orquestagoal.GoalClosureValidationV0{}
	if startAppDirectorGoalResultTerminalV0(result.Status) {
		if ports.GoalClosureValidator == nil {
			return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "ports.goal_closure_validator"}
		}
		closure, err = ports.GoalClosureValidator.ValidateGoalWorkClosureV0(ctx, state.Spec, result)
		if err != nil {
			return ObserveAppDirectorGoalResultV0{}, err
		}
		state.LastClosure = &closure
		state.EvidenceRefs = compactStartAppDirectorStringsV0(append(state.EvidenceRefs, closure.EvidenceRefs...))
	}
	state, err = NewAppDirectorGoalStateV0(state)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	if err := ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	return ObserveAppDirectorGoalResultV0{
		SchemaVersion:   ObserveAppDirectorGoalResultSchemaV0,
		Status:          state.Status,
		RunRef:          state.RunRef,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
		GoalResult:      result,
		Closure:         closure,
		EvidenceRefs:    append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func newStartAppDirectorGoalStateV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	evidenceRefs []string,
) AppDirectorGoalStateV0 {
	goalRef := strings.TrimSpace(receipt.GoalRef)
	if goalRef == "" {
		goalRef = strings.TrimSpace(spec.GoalRef)
	}
	externalGoalRef := strings.TrimSpace(receipt.ExternalGoalRef)
	if externalGoalRef == "" {
		externalGoalRef = strings.TrimSpace(spec.GoalRef)
	}
	return AppDirectorGoalStateV0{
		SchemaVersion:   AppDirectorGoalStateSchemaVersionV0,
		RunRef:          strings.TrimSpace(runRef),
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          strings.TrimSpace(receipt.Status),
		Spec:            orquestagoal.NormalizeGoalWorkSpecV0(spec),
		LaunchReceipt:   receipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			append([]string{"evidence-ref-app-director-goal-state-v0"}, evidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
}

func NewAppDirectorGoalStateV0(state AppDirectorGoalStateV0) (AppDirectorGoalStateV0, error) {
	return orquestagoal.NewGoalWorkStateV0(state)
}

func startAppDirectorGoalResultTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestagoal.GoalStatusCompleteV0, orquestagoal.GoalStatusBlockedV0:
		return true
	default:
		return false
	}
}

func buildStartAppDirectorGoalWorkSpecV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) orquestagoal.GoalWorkSpecV0 {
	token := appDirectorGoalSafeTokenV0(prepared.Run.RunID, request.RunRef, spec.RequestID, spec.App.Slug)
	writeSetPath := "generated-apps/" + appDirectorGoalSafeTokenV0(spec.App.Slug, spec.RequestID, token)
	evidenceRef := "evidence-ref-app-director-goal-first-v0"
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:         "goal-ref-app-director-" + token,
		RequestRef:      spec.RequestID,
		RunRef:          prepared.Run.RunID,
		ProjectRef:      firstStartAppDirectorGoalValueV0(request.ProjectRef, spec.ProjectSource.ProjectRef, spec.SpecID),
		WorkKind:        "new_app",
		WorkProfileKind: "implementation",
		Objective:       startAppDirectorGoalObjectiveV0(spec),
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs: []orquestagoal.GoalContextRefV0{
			{Kind: "request", Ref: spec.RequestID, Purpose: "Solicitud publica normalizada", Required: true},
			{Kind: "run", Ref: prepared.Run.RunID, Purpose: "Run persistida por Orquesta", Required: true},
			{Kind: "app_spec", Ref: spec.SpecID, Purpose: "Contrato AppSpecV0 validado", Required: true},
		},
		RuleRefs: []orquestagoal.GoalRuleRefV0{
			{Kind: "repo", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
			{Kind: "docs", Ref: "docs/orquesta_goal_first_codex_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
			{Kind: "contract", Ref: "modulos/orquesta-factory/docs/contratos.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
			{Kind: "contract", Ref: "modulos/orquesta-web/docs/guia_nueva_app_opciones_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
		},
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    writeSetPath,
			Purpose: "Arbol fuente de la app generada desde el contrato AppSpecV0.",
		}},
		AcceptanceCriteria: startAppDirectorGoalAcceptanceCriteriaV0(spec, writeSetPath),
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{
			{ArtifactRef: "artifact-ref-" + token + "-source", ArtifactType: "source_tree", Required: true},
			{ArtifactRef: "artifact-ref-" + token + "-handoff", ArtifactType: "handoff_report", Required: true},
		},
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{evidenceRef},
			prepared.EvidenceRefs...,
		)),
		Budget:        startAppDirectorGoalBudgetV0(spec),
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequireArtifacts: true, RequiredEvidenceRefs: []string{evidenceRef}},
		ReworkPolicy:  orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, MaxReworkGoals: 1, PreserveArtifacts: true},
	})
}

func startAppDirectorGoalFirstResultV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
) StartAppDirectorResultV0 {
	status := StartAppDirectorStatusPendingV0
	switch strings.TrimSpace(receipt.Status) {
	case orquestagoal.GoalStatusAcceptedV0, orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusCompleteV0:
		status = StartAppDirectorStatusStartedV0
	}
	return StartAppDirectorResultV0{
		SchemaVersion:     StartAppDirectorResultSchemaVersionV0,
		Status:            status,
		CorrelationID:     request.CorrelationID,
		AppSpec:           spec,
		Run:               prepared.Run,
		GoalRef:           strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef:   strings.TrimSpace(receipt.ExternalGoalRef),
		GoalStatus:        strings.TrimSpace(receipt.Status),
		GoalLaunchReceipt: &receipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			append([]string{"evidence-ref-app-director-goal-first-launched-v0"}, prepared.EvidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
}

func startAppDirectorGoalObjectiveV0(spec orquestafactory.AppSpecV0) string {
	return "Construir la app " + strings.TrimSpace(spec.App.Nombre) + " hasta dejar un arbol fuente verificable y documentado. Objetivo funcional: " + strings.TrimSpace(spec.App.Objetivo)
}

func startAppDirectorGoalAcceptanceCriteriaV0(
	spec orquestafactory.AppSpecV0,
	writeSetPath string,
) []string {
	return compactStartAppDirectorStringsV0([]string{
		"Crear la app bajo el write-set autorizado " + writeSetPath + ".",
		"Nombre: " + spec.App.Nombre + ". Tipo: " + spec.App.TipoApp + ". Plataformas: " + startAppDirectorGoalJoinV0(spec.Platforms) + ".",
		"Arquitectura solicitada: " + spec.Architecture.Patron + ". Mantener dominio/aplicacion, puertos, adaptadores y bootstrap separados si el patron no lo contradice.",
		"Objetivo funcional: " + spec.App.Objetivo,
		"Datos: persistencia=" + startAppDirectorGoalBoolV0(spec.Data.PersistenceRequired) + "; necesidades=" + startAppDirectorGoalJoinV0(spec.Data.Needs) + "; sensibilidad=" + spec.Data.Sensitivity + "; storage=" + startAppDirectorGoalStorageSummaryV0(spec.Data.Storage) + "; tipos=" + startAppDirectorGoalDataTypesSummaryV0(spec.Data.Types) + ".",
		"Integraciones requeridas: " + startAppDirectorGoalConnectorsSummaryV0(spec.Connectors.Required) + ". Integraciones opcionales: " + startAppDirectorGoalConnectorsSummaryV0(spec.Connectors.Optional) + ".",
		"Calidad: pruebas=" + spec.Quality.Tests + "; accesibilidad=" + spec.Quality.Accessibility + "; opciones_accesibilidad=" + startAppDirectorGoalJoinV0(spec.Quality.AccessibilityOptions) + "; observabilidad=" + startAppDirectorGoalBoolV0(spec.Quality.Observability) + ".",
		"I18n: enabled=" + startAppDirectorGoalBoolV0(spec.I18N.Enabled) + "; default_locale=" + spec.I18N.DefaultLocale + "; locales=" + startAppDirectorGoalJoinV0(spec.I18N.Locales) + ".",
		"Documentacion: usuario=" + startAppDirectorGoalBoolV0(spec.Docs.User) + "; desarrollo=" + startAppDirectorGoalBoolV0(spec.Docs.Development) + "; sistemas=" + startAppDirectorGoalBoolV0(spec.Docs.Systems) + ".",
		"Project source: kind=" + spec.ProjectSource.Kind + "; project_ref=" + spec.ProjectSource.ProjectRef + "; branch=" + spec.ProjectSource.Branch + ". No publicar rutas locales ni credenciales.",
		"Entregar resumen final con artefactos, pruebas ejecutadas o justificadas, decisiones pendientes y bloqueos si existen.",
	})
}

func startAppDirectorGoalBudgetV0(spec orquestafactory.AppSpecV0) orquestagoal.GoalBudgetV0 {
	maxSubgoals := 6
	if strings.TrimSpace(spec.AgentPreferences.Autonomy) == "alta" || strings.TrimSpace(spec.Quality.Tests) == "alta" {
		maxSubgoals = 10
	}
	return orquestagoal.GoalBudgetV0{MaxSubgoals: maxSubgoals, MaxReworkGoals: 1}
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

func firstStartAppDirectorGoalValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func appDirectorGoalSafeTokenV0(values ...string) string {
	source := firstStartAppDirectorGoalValueV0(values...)
	source = strings.ToLower(strings.TrimSpace(source))
	var builder strings.Builder
	lastDash := false
	for _, r := range source {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "app"
	}
	return token
}
