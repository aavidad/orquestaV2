package orquestaappdirectorservice

import (
	"context"
	"strings"
	"time"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	appDirectorGoalClosureReasonAcceptedV0 = "goal_first_accepted"
	appDirectorGoalClosureReasonBlockedV0  = "goal_first_closure_blocked"
)

func startAppDirectorGoalFirstV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	ports StartAppDirectorPortsV0,
) (StartAppDirectorResultV0, bool, error) {
	if !appDirectorGoalPortsReadyV0(ports) {
		return StartAppDirectorResultV0{}, false, nil
	}
	goalSpec := buildStartAppDirectorGoalWorkSpecV0(request, spec, prepared)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(goalSpec); len(issues) > 0 {
		return StartAppDirectorResultV0{}, false, AppDirectorServiceIssueV0{Field: "goal_spec"}
	}
	goalStarted, err := orquestagoal.StartGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkStartRequestV0{
			RunRef:       prepared.Run.RunID,
			Spec:         goalSpec,
			EvidenceRefs: compactStartAppDirectorStringsV0(append([]string{"evidence-ref-app-director-goal-state-v0"}, prepared.EvidenceRefs...)),
		},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:   ports.GoalLauncher,
			StateStore: ports.GoalStateStore,
		},
	)
	if err != nil {
		return StartAppDirectorResultV0{}, false, err
	}
	if ports.GoalFirstRunMarkerStore != nil {
		if err := ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(
			ctx,
			appDirectorGoalFirstRunMarkerFromLaunchV0(
				prepared.Run.RunID,
				goalStarted.Receipt,
				prepared.EvidenceRefs,
			),
		); err != nil {
			return StartAppDirectorResultV0{}, false, err
		}
	}
	return startAppDirectorGoalFirstResultV0(request, spec, prepared, goalStarted.Receipt), true, nil
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
	run := orquestacoreworkflow.OrchestrationRunV0{}
	goalObserved, err := orquestagoal.ObserveGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkObserveRequestV0{RunRef: request.RunRef},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Observer:         ports.GoalObserver,
			ClosureValidator: ports.GoalClosureValidator,
			StateStore:       ports.GoalStateStore,
		},
	)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	state := goalObserved.State
	result := goalObserved.Result
	closure := goalObserved.Closure
	if goalObserved.Terminal {
		run, err = reflectAppDirectorGoalClosureInRunV0(ctx, request, state, result, closure, ports)
		if err != nil {
			return ObserveAppDirectorGoalResultV0{}, err
		}
	}
	return ObserveAppDirectorGoalResultV0{
		SchemaVersion:         ObserveAppDirectorGoalResultSchemaV0,
		Status:                state.Status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		RunRef:                state.RunRef,
		GoalRef:               state.GoalRef,
		ExternalGoalRef:       state.ExternalGoalRef,
		Run:                   run,
		GoalResult:            result,
		Closure:               closure,
		EvidenceRefs:          append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func reflectAppDirectorGoalClosureInRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if ports.RunStore == nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	if ports.EventSink == nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, AppDirectorServiceIssueV0{Field: "ports.event_sink"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, state.RunRef)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return run, nil
	}
	if !closure.Accepted {
		return blockAppDirectorGoalRunV0(ctx, request, state, result, closure, ports)
	}
	return closeAppDirectorGoalRunV0(ctx, request, run, state, result, closure, ports)
}

func blockAppDirectorGoalRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockerRef := appDirectorGoalBlockerRefV0(state)
	command, err := orquestacoreworkflow.NewBlockRunCommandV0(
		appDirectorGoalCommandMetaV0(request, state.RunRef, "block-goal", blockerRef),
		orquestacoreworkflow.BlockRunCommandPayloadV0{
			BlockerID:    blockerRef,
			ReasonCode:   appDirectorGoalClosureReasonBlockedV0,
			Summary:      "Goal-first bloqueado por cierre no aceptado.",
			SourceGroup:  "app-director-goal-first",
			EvidenceRefs: appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return ports.RunStore.LoadRunV0(ctx, state.RunRef)
}

func closeAppDirectorGoalRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	run, err := resolveAppDirectorGoalBlockerIfPresentV0(ctx, request, run, state, result, closure, ports)
	if err != nil {
		return run, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return run, nil
	}
	validationRef := appDirectorGoalValidationRefV0(state)
	closureRef := appDirectorGoalClosureRefV0(state)
	if run, err = openAppDirectorGoalFinalValidationIfNeededV0(ctx, request, run, state, validationRef, ports); err != nil {
		return run, err
	}
	if run, err = registerAppDirectorGoalFinalValidationIfNeededV0(ctx, request, run, state, result, closure, validationRef, ports); err != nil {
		return run, err
	}
	if run, err = openAppDirectorGoalClosureIfNeededV0(ctx, request, run, state, closureRef, ports); err != nil {
		return run, err
	}
	return closeAppDirectorGoalRunIfNeededV0(ctx, request, run, state, result, closure, validationRef, closureRef, ports)
}

func resolveAppDirectorGoalBlockerIfPresentV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockerRef := appDirectorGoalBlockerRefV0(state)
	if !appDirectorGoalRunHasBlockerV0(run, blockerRef) {
		return run, nil
	}
	return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "resolve-goal-blocker", blockerRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
		return orquestacoreworkflow.NewResolveRunBlockerCommandV0(meta, orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blockerRef,
			ReasonCode:   appDirectorGoalClosureReasonAcceptedV0,
			Summary:      "Goal-first aceptado tras observacion de cierre.",
			EvidenceRefs: appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		})
	})
}

func openAppDirectorGoalFinalValidationIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	validationRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0 &&
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "open-final-validation", validationRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewOpenPhaseCommandV0(meta, orquestacoreworkflow.OpenPhaseCommandPayloadV0{
				PhaseID: string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
				Reason:  "Validacion final run-level de goal-first aceptado.",
			})
		})
	}
	return run, nil
}

func registerAppDirectorGoalFinalValidationIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	validationRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if !appDirectorGoalStringInSetV0(run.Validations, validationRef) {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "register-final-validation", validationRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewRegisterFinalValidationCommandV0(meta, orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
				ValidationRef: validationRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
				Summary:       "Validacion final run-level de goal-first aceptado.",
				EvidenceRefs:  appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
			})
		})
	}
	return run, nil
}

func openAppDirectorGoalClosureIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	closureRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "open-closure", closureRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewOpenPhaseCommandV0(meta, orquestacoreworkflow.OpenPhaseCommandPayloadV0{
				PhaseID: string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
				Reason:  "Cierre de run goal-first aceptada.",
			})
		})
	}
	return run, nil
}

func closeAppDirectorGoalRunIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	validationRef string,
	closureRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if !appDirectorGoalStringInSetV0(run.Closures, closureRef) {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "close-run", closureRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewCloseRunCommandV0(meta, orquestacoreworkflow.CloseRunCommandPayloadV0{
				ClosureRef:    closureRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
				ValidationRef: validationRef,
				Summary:       "Run goal-first cerrada con closure aceptada.",
				EvidenceRefs:  appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
			})
		})
	}
	return run, nil
}

type appDirectorGoalWorkflowCommandBuilderV0 func(orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error)

func applyAppDirectorGoalWorkflowCommandV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	runRef string,
	action string,
	ref string,
	ports StartAppDirectorPortsV0,
	build appDirectorGoalWorkflowCommandBuilderV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	command, err := build(appDirectorGoalCommandMetaV0(request, runRef, action, ref))
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return ports.RunStore.LoadRunV0(ctx, runRef)
}

func appDirectorGoalCommandMetaV0(
	request ObserveAppDirectorGoalRequestV0,
	runRef string,
	action string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	token := appDirectorGoalSafeTokenV0(runRef, ref, action)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-director-goal-" + strings.TrimSpace(action) + "-" + token,
		RunID:          strings.TrimSpace(runRef),
		IdempotencyKey: "idem-app-director-goal-" + strings.TrimSpace(action) + "-" + token,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    firstStartAppDirectorGoalValueV0(request.RequestedBy, "orquesta-app-director-service"),
		OccurredAt:     appDirectorGoalOccurredAtV0(request.OccurredAt),
	}
}

func appDirectorGoalOccurredAtV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func appDirectorGoalBlockerRefV0(state AppDirectorGoalStateV0) string {
	return "blocker-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalValidationRefV0(state AppDirectorGoalStateV0) string {
	return "validation-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalClosureRefV0(state AppDirectorGoalStateV0) string {
	return "closure-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalCommandEvidenceRefsV0(
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) []string {
	refs := compactStartAppDirectorStringsV0(append(append(append(
		[]string{"evidence-ref-app-director-goal-observed-v0"},
		state.EvidenceRefs...,
	), result.EvidenceRefs...), closure.EvidenceRefs...))
	if len(refs) > 20 {
		return append([]string(nil), refs[:20]...)
	}
	return refs
}

func appDirectorGoalRunHasBlockerV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	blockerRef string,
) bool {
	return appDirectorGoalStringInSetV0(run.Blockers, blockerRef)
}

func appDirectorGoalStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func NewAppDirectorGoalStateV0(state AppDirectorGoalStateV0) (AppDirectorGoalStateV0, error) {
	return orquestagoal.NewGoalWorkStateV0(state)
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
		RequiredTests:      startAppDirectorGoalRequiredTestsV0(token, writeSetPath, spec),
		AcceptanceCriteria: startAppDirectorGoalAcceptanceCriteriaV0(spec, writeSetPath),
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{
			{ArtifactRef: "artifact-ref-" + token + "-source", ArtifactType: "source_tree", Required: true},
			{ArtifactRef: "artifact-ref-" + token + "-handoff", ArtifactType: "handoff_report", Required: true},
		},
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
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, MaxReworkGoals: 1, PreserveArtifacts: true},
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
		SchemaVersion:         StartAppDirectorResultSchemaVersionV0,
		Status:                status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		CorrelationID:         request.CorrelationID,
		AppSpec:               spec,
		Run:                   prepared.Run,
		GoalRef:               strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(receipt.ExternalGoalRef),
		GoalStatus:            strings.TrimSpace(receipt.Status),
		GoalLaunchReceipt:     &receipt,
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

func startAppDirectorGoalRequiredTestsV0(
	token string,
	writeSetPath string,
	spec orquestafactory.AppSpecV0,
) []orquestagoal.GoalRequiredTestV0 {
	testRef := "test-ref-" + token + "-generated-app"
	return []orquestagoal.GoalRequiredTestV0{{
		TestRef:    testRef,
		CommandRef: "command-ref-" + token + "-verify-generated-app",
		Command: "verificar la app generada bajo " + writeSetPath +
			" con las pruebas propias del stack elegido o una prueba local documentada",
		AcceptanceCriteria: compactStartAppDirectorStringsV0([]string{
			"El arbol fuente requerido existe bajo " + writeSetPath + ".",
			"La app cumple el contrato funcional: " + strings.TrimSpace(spec.App.Objetivo),
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
