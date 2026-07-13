package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestRunSupervisorGoalFirstResidentPreparaReworkPorCheckpointHighConsumptionV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	binder := &goalFirstResidentReworkBinderForTestV0{}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-rework-source-001",
		"checkpoint_only_high_consumption",
	)
	source.Spec.RequiredTests = []orquestagoal.GoalRequiredTestV0{orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
		TestRef:    "test-ref-goal-first-resident-rework-source-001",
		CommandRef: "command-ref-goal-first-resident-rework-source-001",
		Command:    "go test ./modulos/orquesta-app-codex-stack",
	})}
	source.Spec.ClosurePolicy.RequireIndependentRequiredTestAttestation = true
	var err error
	source.Spec, err = (independentSpecBinderForStackTestV0{}).BindGoalRequiredTestSpecV0(ctx, source.Spec)
	if err != nil {
		t.Fatalf("Bind source spec: %v", err)
	}
	source.Spec.WriteSetSHA256 = orquestagoal.GoalWriteSetSHA256V0(source.Spec.WriteSet)
	source.Spec.IntentManifestRef = "intent-manifest-ref-request-ref-goal-first-resident-rework-source-001"
	source.Spec.IntentManifestSHA256 = strings.Repeat("a", 64)
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:                   runStore,
			GoalStateStore:             store,
			GoalReworkLauncher:         launcher,
			GoalRequiredTestSpecBinder: binder,
		},
		Stores: StoresV0{RunStore: runStore, AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 || binder.calls != 1 {
		t.Fatalf("resultado rework inesperado: result=%+v launcher_calls=%d binder_calls=%d", result, launcher.calls, binder.calls)
	}
	spec := launcher.specs[0]
	if spec.RunRef != result.RepairRunRefs[0] ||
		!strings.Contains(spec.Objective, "Rework acotado") ||
		spec.ImplementerAgentRef != "agent-ref-goal-first-resident-rework-binder" ||
		spec.RequestRef != source.Spec.RequestRef ||
		spec.IntentManifestRef != source.Spec.IntentManifestRef || spec.IntentManifestSHA256 != source.Spec.IntentManifestSHA256 ||
		!stringInSetV0(spec.EvidenceRefs, goalFirstResidentReworkPreparedEvidenceRefV0) {
		t.Fatalf("spec rework incompleto: %+v result=%+v", spec, result)
	}
	if _, err := store.LoadGoalWorkStateV0(ctx, result.RepairRunRefs[0]); err != nil {
		t.Fatalf("estado rework no persistido: %v", err)
	}
	if reworkRun, err := runStore.LoadRunV0(ctx, result.RepairRunRefs[0]); err != nil ||
		reworkRun.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		t.Fatalf("run causal de rework no persistida: run=%+v err=%v", reworkRun, err)
	}
	replayed, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil || len(replayed.RepairRunRefs) != 1 ||
		replayed.RepairRunRefs[0] != result.RepairRunRefs[0] ||
		launcher.calls != 1 || binder.calls != 1 {
		t.Fatalf("replay rework no idempotente: replay=%+v err=%v launcher_calls=%d binder_calls=%d", replayed, err, launcher.calls, binder.calls)
	}
	persistedSource, err := store.LoadGoalWorkStateV0(ctx, source.RunRef)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	if goalFirstResidentExistingReworkRunRefV0(persistedSource) != result.RepairRunRefs[0] {
		t.Fatalf("source sin marca de rework: %+v", persistedSource.EvidenceRefs)
	}
}

func TestGoalFirstResidentReworkSpecV0PreservaIdentidadManifestYDerivaLegacyV0(t *testing.T) {
	manifestSource := goalFirstResidentReworkSourceStateForTestV0("run-ref-rework-manifest-001", "checkpoint_only_high_consumption")
	manifestSource.Spec.IntentManifestRef = "intent-manifest-ref-" + manifestSource.Spec.RequestRef
	manifestSource.Spec.IntentManifestSHA256 = strings.Repeat("a", 64)
	manifestRework := goalFirstResidentReworkSpecV0(manifestSource, "checkpoint_only_high_consumption", nil)
	if manifestRework.RequestRef != manifestSource.Spec.RequestRef ||
		manifestRework.IntentManifestRef != manifestSource.Spec.IntentManifestRef ||
		manifestRework.IntentManifestSHA256 != manifestSource.Spec.IntentManifestSHA256 {
		t.Fatalf("manifest identity changed: source=%+v rework=%+v", manifestSource.Spec, manifestRework)
	}

	legacySource := goalFirstResidentReworkSourceStateForTestV0("run-ref-rework-legacy-001", "checkpoint_only_high_consumption")
	legacyRework := goalFirstResidentReworkSpecV0(legacySource, "checkpoint_only_high_consumption", nil)
	if legacyRework.RequestRef == legacySource.Spec.RequestRef ||
		!strings.HasPrefix(legacyRework.RequestRef, legacySource.Spec.RequestRef+"-rework-") {
		t.Fatalf("legacy request identity not derived: source=%q rework=%q", legacySource.Spec.RequestRef, legacyRework.RequestRef)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorCheckpointStartedTerminalV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-checkpoint-started-001",
		goalFirstResidentReworkReasonCheckpointStartedV0,
	)
	source.LastResult.Checklist.MissingRefs = []string{"implementation", "required_tests"}
	source.LastResult.EvidenceRefs = []string{"evidence-ref-checkpoint-started"}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!hasGoalFirstResidentReworkContextForTestV0(
			launcher.specs[0].ContextRefs,
			"rework_reason",
			goalFirstResidentReworkReasonCheckpointStartedV0,
		) ||
		!launcher.specs[0].ReworkPolicy.PreserveArtifacts {
		t.Fatalf("autorework checkpoint_started no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
}

func TestGoalFirstResidentCheckpointStartedNoSeInfiereDelSummaryV0(t *testing.T) {
	state := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-summary-only-001",
		"unrelated_terminal_issue",
	)
	state.LastResult.EvidenceRefs = nil
	state.LastResult.Summary = "checkpoint_started es solo texto informativo"

	if reason, _, ok := goalFirstResidentReworkReasonV0(state); ok {
		t.Fatalf("checkpoint_started inferido del summary: reason=%q state=%+v", reason, state)
	}
}

func TestGoalFirstResidentMaterialProgressReplanRespetaPresupuestoV0(t *testing.T) {
	state := orquestagoal.GoalWorkStateV0{
		Status: orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{ReworkPolicy: orquestagoal.GoalReworkPolicyV0{
			MaxReworkGoals: 1,
		}},
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status: orquestagoal.GoalStatusBlockedV0,
			Issues: []orquestagoal.GoalWorkIssueV0{{Code: goalFirstResidentReworkReasonMaterialProgressV0}},
		},
	}
	reason, _, ok := goalFirstResidentReworkReasonV0(state)
	if !ok || reason != goalFirstResidentReworkReasonMaterialProgressV0 ||
		!goalFirstResidentReworkBudgetAvailableV0(state.Spec) {
		t.Fatalf("reason=%q ok=%v spec=%+v", reason, ok, state.Spec)
	}
	state.Spec.ContextRefs = append(state.Spec.ContextRefs, orquestagoal.GoalContextRefV0{
		Kind: "source_goal", Ref: "goal-ref-material-progress-source",
	})
	if goalFirstResidentReworkBudgetAvailableV0(state.Spec) {
		t.Fatal("rework material permitido tras agotar max_rework_goals")
	}
}

func TestGoalFirstResidentMaterialProgressHardStopNuncaRelanzaV0(t *testing.T) {
	state := orquestagoal.GoalWorkStateV0{
		Status: orquestagoal.GoalStatusBlockedV0,
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status: orquestagoal.GoalStatusBlockedV0,
			Issues: []orquestagoal.GoalWorkIssueV0{{Code: goalFirstResidentHardStopMaterialProgressV0}},
		},
	}
	if reason, refs, ok := goalFirstResidentReworkReasonV0(state); ok || reason != "" || len(refs) != 0 {
		t.Fatalf("reason=%q refs=%v ok=%v", reason, refs, ok)
	}
}

func TestRunSupervisorGoalFirstResidentReworkEsIdempotenteV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-rework-source-002",
		"goal_active_no_checkpoint_high_consumption",
	)
	existingRunRef := "run-ref-goal-first-resident-rework-existing-002"
	source.EvidenceRefs = append(source.EvidenceRefs, goalFirstResidentReworkExistingEvidencePrefixV0+existingRunRef)
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if launcher.calls != 0 ||
		len(result.RepairRunRefs) != 1 ||
		result.RepairRunRefs[0] != existingRunRef {
		t.Fatalf("rework no idempotente: result=%+v calls=%d", result, launcher.calls)
	}
	if !strings.Contains(result.Diagnostics[len(result.Diagnostics)-1].Code, "already_prepared") {
		t.Fatalf("diagnostico idempotente ausente: %+v", result.Diagnostics)
	}
}

func TestRunSupervisorGoalFirstResidentReconciliaBackendMissingTrasCleanupExternoV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	runControl := &goalFirstResidentRunControlTerminalForTestV0{}
	runRef := "run-ref-goal-first-resident-backend-missing-001"
	goalRef := "goal-ref-goal-first-resident-backend-missing-001"
	externalGoalRef := "thread-ref-goal-first-resident-backend-missing-001"
	source := orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RequestRef:   "request-ref-goal-first-resident-backend-missing-001",
			RunRef:       runRef,
			ProjectRef:   "project-ref-goal-first-resident-backend-missing",
			Objective:    "Reconciliar cleanup externo sin operador.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/goal-first-resident-backend-missing.md",
				Purpose: "evidencia de test",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-resident-backend-missing-source"},
	}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	stats := &goalFirstResidentDirectorStatsForTestV0{
		result: orquestamcp.MCPDirectorStatsToolResultV0{
			Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
			RunRef: runRef,
		},
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
			RunControlTerminal: runControl,
		},
		Stores: StoresV0{AppGoalStateStore: store},
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			DirectorStats: stats,
		},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stats.calls != 1 ||
		runControl.calls != 1 ||
		result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, goalFirstResidentBackendMissingReconciledV0) {
		t.Fatalf("resultado inesperado: result=%+v stats_calls=%d launcher_calls=%d run_control_calls=%d", result, stats.calls, launcher.calls, runControl.calls)
	}
	if runControl.command.RunRef != runRef ||
		runControl.command.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!strings.Contains(runControl.command.Reason, "external cleanup") ||
		strings.Contains(runControl.command.Reason, "forced") ||
		!stringInSetV0(runControl.command.EvidenceRefs, "evidence-ref-run-control-terminal-after-goal-reconcile") ||
		!stringInSetV0(runControl.command.EvidenceRefs, goalFirstResidentBackendMissingEvidenceRefV0) {
		t.Fatalf("run control no completado como cleanup externo: %+v", runControl.command)
	}
	if len(launcher.specs) != 1 ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, goalFirstResidentBackendMissingReconciledV0) ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, goalFirstResidentBackendMissingEvidenceRefV0) {
		t.Fatalf("spec rework sin evidencias de cleanup externo: %+v", launcher.specs)
	}
	persistedSource, err := store.LoadGoalWorkStateV0(ctx, source.RunRef)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	if persistedSource.Status != orquestagoal.GoalStatusBlockedV0 ||
		persistedSource.LastResult == nil ||
		persistedSource.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		persistedSource.LastClosure == nil ||
		!persistedSource.LastClosure.NeedsRework ||
		!goalFirstResidentHasIssueOrEvidenceV0(persistedSource, goalFirstResidentReworkReasonBackendMissingV0) ||
		goalFirstResidentExistingReworkRunRefV0(persistedSource) != result.RepairRunRefs[0] {
		t.Fatalf("source no reconciliado/rework: %+v", persistedSource)
	}
}

func TestRunSupervisorGoalFirstNoResidentNoReconciliaBackendMissingTrasCleanupExternoV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	runRef := "run-ref-goal-first-no-resident-backend-missing-001"
	goalRef := "goal-ref-goal-first-no-resident-backend-missing-001"
	externalGoalRef := "thread-ref-goal-first-no-resident-backend-missing-001"
	source := orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RequestRef:   "request-ref-goal-first-no-resident-backend-missing-001",
			RunRef:       runRef,
			ProjectRef:   "project-ref-goal-first-no-resident-backend-missing",
			Objective:    "No reconciliar cleanup externo fuera de modo residente.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/goal-first-no-resident-backend-missing.md",
				Purpose: "evidencia de test",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-no-resident-backend-missing-source"},
	}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	stats := &goalFirstResidentDirectorStatsForTestV0{
		result: orquestamcp.MCPDirectorStatsToolResultV0{
			Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
			RunRef: runRef,
		},
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			DirectorStats: stats,
		},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{RunRef: source.RunRef})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stats.calls != 0 ||
		launcher.calls != 0 ||
		len(result.RepairRunRefs) != 0 ||
		result.StopReason != "goal_first_observe_required" {
		t.Fatalf("supervision no residente reconcilio cleanup externo: result=%+v stats_calls=%d launcher_calls=%d", result, stats.calls, launcher.calls)
	}
	persistedSource, err := store.LoadGoalWorkStateV0(ctx, source.RunRef)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	if persistedSource.Status != orquestagoal.GoalStatusRunningV0 ||
		persistedSource.LastResult != nil ||
		persistedSource.LastClosure != nil ||
		goalFirstResidentHasIssueOrEvidenceV0(persistedSource, goalFirstResidentReworkReasonBackendMissingV0) ||
		goalFirstResidentHasIssueOrEvidenceV0(persistedSource, goalFirstResidentBackendMissingReconciledV0) {
		t.Fatalf("source no residente mutado por reconciliacion: %+v", persistedSource)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorTimeoutInicialSinArtefactosV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-new-app-timeout-001",
		goalFirstResidentReworkReasonActiveTimeoutV0,
	)
	source.LastResult.Status = orquestagoal.GoalStatusInvalidV0
	source.LastResult.Summary = goalFirstResidentReworkReasonActiveTimeoutV0
	source.LastResult.ArtifactRefs = nil
	source.LastResult.ArtifactPaths = nil
	source.LastResult.DomainReceiptRefs = nil
	source.LastResult.EvidenceRefs = []string{"evidence-ref-codex-app-server-goal-active-timeout"}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{Code: goalFirstResidentReworkReasonActiveTimeoutV0, Field: "goal_backend"}}
	source.LastClosure.EvidenceRefs = []string{"evidence-ref-codex-app-server-goal-active-timeout"}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!strings.Contains(launcher.specs[0].Objective, "Rework acotado") {
		t.Fatalf("resultado rework inesperado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
	if !hasGoalFirstResidentReworkContextForTestV0(
		launcher.specs[0].ContextRefs,
		"rework_reason",
		goalFirstResidentReworkReasonActiveTimeoutV0,
	) {
		t.Fatalf("context refs sin motivo timeout: %+v", launcher.specs[0].ContextRefs)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorBloqueoOperativoRecuperableV0(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name        string
		reason      string
		evidenceRef string
	}{
		{
			name:        "workdir_inexistente",
			reason:      goalFirstResidentReworkReasonWorkdirV0,
			evidenceRef: "evidence-ref-codex-app-server-write-set-prepare-failed",
		},
		{
			name:        "auth_provider",
			reason:      goalFirstResidentReworkReasonAuthV0,
			evidenceRef: "evidence-ref-codex-app-server-provider-unauthorized",
		},
		{
			name:        "auth_missing",
			reason:      goalFirstResidentReworkReasonAuthMissingV0,
			evidenceRef: "evidence-ref-codex-app-server-auth-missing",
		},
		{
			name:        "quota_provider",
			reason:      goalFirstResidentReworkReasonProviderLimitedV0,
			evidenceRef: "evidence-ref-codex-app-server-goal-provider-limited",
		},
		{
			name:        "storage_quota",
			reason:      goalFirstResidentReworkReasonStorageQuotaV0,
			evidenceRef: "evidence-ref-codex-app-server-storage-quota-exceeded",
		},
		{
			name:        "backend_caido",
			reason:      goalFirstResidentReworkReasonBackendUnavailableV0,
			evidenceRef: "evidence-ref-codex-app-server-backend-unavailable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newGoalFirstQueueStateStoreForTestV0()
			launcher := &goalFirstResidentReworkLauncherForTestV0{}
			source := goalFirstResidentReworkSourceStateForTestV0(
				"run-ref-goal-first-resident-operational-blocker-"+tc.name+"-001",
				tc.reason,
			)
			source.LastResult.Summary = tc.reason
			source.LastResult.EvidenceRefs = []string{tc.evidenceRef}
			source.LastClosure.EvidenceRefs = []string{tc.evidenceRef}
			if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
				t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
			}
			executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
				Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
					GoalStateStore:     store,
					GoalReworkLauncher: launcher,
				},
				Stores: StoresV0{AppGoalStateStore: store},
			}}

			result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
				RunRef:       source.RunRef,
				ResidentMode: true,
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if result.StopReason != "goal_first_resident_rework_prepared" ||
				len(result.RepairRunRefs) != 1 ||
				launcher.calls != 1 ||
				!hasGoalFirstResidentReworkContextForTestV0(
					launcher.specs[0].ContextRefs,
					"rework_reason",
					tc.reason,
				) ||
				!stringInSetV0(launcher.specs[0].EvidenceRefs, tc.evidenceRef) {
				t.Fatalf("rework operativo no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
			}
		})
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorArtefactosParcialesTerminalesV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-partial-artifacts-001",
		orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0,
	)
	source.LastResult.EvidenceRefs = []string{
		"evidence-ref-goal-materialized-partial-artifacts-written",
		"evidence-ref-goal-materialized-checkpoint-detected",
	}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{
		Code:  orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0,
		Field: "goal_first.partial_artifacts",
	}}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!hasGoalFirstResidentReworkContextForTestV0(
			launcher.specs[0].ContextRefs,
			"rework_reason",
			orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0,
		) ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") {
		t.Fatalf("rework por artefactos parciales no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorArtifactPathsOmitidosV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-artifact-paths-omitted-001",
		orquestamcp.MCPGoalFirstArtifactPathsOmittedMaterializedV0,
	)
	source.LastResult.EvidenceRefs = []string{
		"evidence-ref-goal-materialized-artifact-paths-omitted",
		"evidence-ref-goal-materialized-artifact-paths-omitted:run:trabajo-extra",
	}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{
		Code:  orquestamcp.MCPGoalFirstArtifactPathsOmittedMaterializedV0,
		Field: "goal_first.artifact_paths",
	}}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!hasGoalFirstResidentReworkContextForTestV0(
			launcher.specs[0].ContextRefs,
			"rework_reason",
			orquestamcp.MCPGoalFirstArtifactPathsOmittedMaterializedV0,
		) ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, "evidence-ref-goal-materialized-artifact-paths-omitted") {
		t.Fatalf("rework por artifact_paths omitidos no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorArtefactosFueraDeWriteSetV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-out-of-scope-001",
		orquestamcp.MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
	)
	source.LastResult.EvidenceRefs = []string{
		"evidence-ref-goal-materialized-out-of-scope-artifacts",
		"evidence-ref-goal-materialized-out-of-scope-artifacts:run:html-index",
	}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{
		Code:  orquestamcp.MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
		Field: "goal_first.write_set",
	}}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!hasGoalFirstResidentReworkContextForTestV0(
			launcher.specs[0].ContextRefs,
			"rework_reason",
			orquestamcp.MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
		) ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, "evidence-ref-goal-materialized-out-of-scope-artifacts") {
		t.Fatalf("rework por artefactos fuera de write-set no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
}

func TestRunSupervisorGoalFirstResidentPreparaReworkPorQAFallidaPublicaV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-qa-failed-public-text-001",
		orquestamcp.MCPGoalFirstQAFailedPublicTextV0,
	)
	source.LastResult.EvidenceRefs = []string{
		"evidence-ref-goal-materialized-qa-failed-public-text",
		"evidence-ref-goal-materialized:qa-report-001",
	}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{
		Code:  orquestamcp.MCPGoalFirstQAFailedPublicTextV0,
		Field: "goal_first.public_text",
	}}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 ||
		!hasGoalFirstResidentReworkContextForTestV0(
			launcher.specs[0].ContextRefs,
			"rework_reason",
			orquestamcp.MCPGoalFirstQAFailedPublicTextV0,
		) ||
		!stringInSetV0(launcher.specs[0].EvidenceRefs, "evidence-ref-goal-materialized-qa-failed-public-text") {
		t.Fatalf("rework por QA fallida no preparado: result=%+v calls=%d spec=%+v", result, launcher.calls, launcher.specs)
	}
}

func TestRunSupervisorGoalFirstResidentNoReworkPorIssueRecuperableActivoV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-partial-artifacts-active-001",
		orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0,
	)
	source.Status = orquestagoal.GoalStatusRunningV0
	source.LastResult = nil
	source.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status: orquestagoal.GoalStatusRunningV0,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0,
			Field: "goal_first.partial_artifacts",
		}},
	}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if launcher.calls != 0 ||
		len(result.RepairRunRefs) != 0 ||
		result.StopReason != "goal_first_observe_required" {
		t.Fatalf("estado activo no debe lanzar rework: result=%+v calls=%d", result, launcher.calls)
	}
}

func TestRunSupervisorGoalFirstNoResidentNoLanzaReworkV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-rework-source-003",
		"checkpoint_only_high_consumption",
	)
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{RunRef: source.RunRef})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if launcher.calls != 0 ||
		len(result.RepairRunRefs) != 0 ||
		result.StopReason != "goal_first_observe_required" {
		t.Fatalf("supervision no residente lanzo rework: result=%+v calls=%d", result, launcher.calls)
	}
}

func TestRunSupervisorGoalFirstResidentNoRelanzaTimeoutConArtefactosV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-new-app-timeout-artifact-001",
		goalFirstResidentReworkReasonActiveTimeoutV0,
	)
	source.LastResult.Summary = goalFirstResidentReworkReasonActiveTimeoutV0
	source.LastResult.ArtifactRefs = []string{"artifact-ref-generated-app-code-001"}
	source.LastResult.EvidenceRefs = []string{"evidence-ref-codex-app-server-goal-active-timeout"}
	source.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{Code: goalFirstResidentReworkReasonActiveTimeoutV0, Field: "goal_backend"}}
	source.LastClosure.EvidenceRefs = []string{"evidence-ref-codex-app-server-goal-active-timeout"}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       source.RunRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if launcher.calls != 0 ||
		len(result.RepairRunRefs) != 0 ||
		result.StopReason != "goal_first_observe_required" {
		t.Fatalf("timeout con artefactos no debe relanzar: result=%+v calls=%d", result, launcher.calls)
	}
}

func TestRunSupervisorGoalFirstResidentReworkPreparaSucesorPorAttestorInfraTipadoV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-attestor-infra-001",
		orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0,
	)
	source.LastResult.EvidenceRefs = nil
	source.LastClosure.EvidenceRefs = nil
	source.LastClosure.NeedsRework = false
	source.Spec.ContextRefs = []orquestagoal.GoalContextRefV0{{
		Kind: "source_task", Ref: "task-ref-attestor-rework-code-gate3", Required: true,
	}}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef: source.RunRef, ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" || len(result.RepairRunRefs) != 1 || launcher.calls != 1 {
		t.Fatalf("attestor infra no preparo sucesor causal: result=%+v calls=%d", result, launcher.calls)
	}
	if !hasGoalFirstResidentReworkContextForTestV0(launcher.specs[0].ContextRefs, "source_run", source.RunRef) ||
		!hasGoalFirstResidentReworkContextForTestV0(launcher.specs[0].ContextRefs, "rework_reason", orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0) ||
		!hasGoalFirstResidentReworkContextForTestV0(launcher.specs[0].ContextRefs, "source_task", "task-ref-attestor-rework-code-gate3") {
		t.Fatalf("sucesor sin causalidad de attestor infra: %+v", launcher.specs[0].ContextRefs)
	}
}

func TestRunSupervisorGoalFirstResidentReworkNoPreparaSucesorPorTextoAttestorInfraV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	source := goalFirstResidentReworkSourceStateForTestV0(
		"run-ref-goal-first-resident-attestor-infra-text-001",
		"unrelated_blocked_issue",
	)
	source.LastResult.Issues = nil
	source.LastClosure.Issues = nil
	source.LastResult.Summary = orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0
	source.LastResult.EvidenceRefs = []string{"evidence-ref-" + orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0}
	source.LastClosure.EvidenceRefs = []string{"log-ref-" + orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{AppGoalStateStore: store},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef: source.RunRef, ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if launcher.calls != 0 || len(result.RepairRunRefs) != 0 || result.StopReason != "goal_first_observe_required" {
		t.Fatalf("texto de attestor infra preparo sucesor indebidamente: result=%+v calls=%d", result, launcher.calls)
	}
}

func goalFirstResidentReworkSourceStateForTestV0(runRef, reason string) orquestagoal.GoalWorkStateV0 {
	goalRef := strings.Replace(runRef, "run-ref-", "goal-ref-", 1)
	return orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RequestRef:   strings.Replace(runRef, "run-ref-", "request-ref-", 1),
			RunRef:       runRef,
			ProjectRef:   "project-ref-goal-first-resident-rework-test",
			Objective:    "Continuar trabajo goal-first con entrega verificable.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/goal-first-resident-rework-test.md",
				Purpose: "evidencia de test",
			}},
			EvidenceRefs: []string{"evidence-ref-goal-first-resident-rework-source"},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusRunningV0,
			GoalRef:       goalRef,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       goalRef,
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-checkpoint-only-high-consumption",
				"evidence-ref-autoprogramming-no-checkpoint-high-consumption",
			},
			Issues: []orquestagoal.GoalWorkIssueV0{{Code: reason, Field: "goal_backend"}},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []orquestagoal.GoalWorkIssueV0{{Code: reason, Field: "goal_backend"}},
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-resident-rework-source-state"},
	}
}

type goalFirstResidentReworkLauncherForTestV0 struct {
	calls int
	specs []orquestagoal.GoalWorkSpecV0
}

type goalFirstResidentReworkBinderForTestV0 struct{ calls int }

func (binder *goalFirstResidentReworkBinderForTestV0) BindGoalRequiredTestSpecV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	binder.calls++
	spec.ImplementerAgentRef = "agent-ref-goal-first-resident-rework-binder"
	spec.ImplementerCredentialRef = "credential-ref-goal-first-resident-rework-binder"
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = "policy-ref-goal-first-resident-rework-binder"
	return orquestagoal.NormalizeGoalWorkSpecV0(spec), nil
}

type goalFirstResidentDirectorStatsForTestV0 struct {
	calls  int
	result orquestamcp.MCPDirectorStatsToolResultV0
}

type goalFirstResidentRunControlTerminalForTestV0 struct {
	calls   int
	command orquestaruncontrol.CompleteRunControlCommandV0
}

func (terminal *goalFirstResidentRunControlTerminalForTestV0) CompleteRunControlV0(
	_ context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	terminal.calls++
	terminal.command = command
	return orquestaruncontrol.RunControlStateV0{
		RunRef:       strings.TrimSpace(command.RunRef),
		Status:       command.TargetStatus,
		EvidenceRefs: append([]string(nil), command.EvidenceRefs...),
	}, nil
}

func (fake *goalFirstResidentDirectorStatsForTestV0) Execute(
	_ context.Context,
	_ orquestamcp.MCPDirectorStatsToolInputV0,
) (orquestamcp.MCPDirectorStatsToolResultV0, error) {
	fake.calls++
	return fake.result, nil
}

func hasGoalFirstResidentReworkContextForTestV0(
	refs []orquestagoal.GoalContextRefV0,
	kind string,
	ref string,
) bool {
	for _, ctxRef := range refs {
		if strings.TrimSpace(ctxRef.Kind) == kind && strings.TrimSpace(ctxRef.Ref) == ref {
			return true
		}
	}
	return false
}

func (launcher *goalFirstResidentReworkLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	launcher.specs = append(launcher.specs, spec)
	receipt := orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-goal-first-resident-rework-test",
		EvidenceRefs:    []string{"evidence-ref-goal-first-resident-rework-launch"},
	}
	if spec.IntentManifestRef != "" || spec.IntentManifestSHA256 != "" {
		receipt = orquestagoal.ApplyGoalExecutionAuthorityToReceiptV0(
			receipt,
			orquestagoal.GoalExecutionAuthorityForProviderV0(spec, "provider-ref-test-resident-rework"),
		)
	}
	return receipt, nil
}

func TestRunSupervisorGoalFirstResidentReconciliaProcesoMuertoConVeredictoCausalV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	runRef := "run-ref-goal-first-resident-process-dead-001"
	goalRef := "goal-ref-goal-first-resident-process-dead-001"
	source := orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RequestRef:   "request-ref-goal-first-resident-process-dead-001",
			RunRef:       runRef,
			ProjectRef:   "project-ref-goal-first-resident-process-dead",
			Objective:    "No esperar infinito un proceso muerto con state stale.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/goal-first-resident-process-dead.md",
				Purpose: "evidencia de test",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusRunningV0,
			GoalRef:       goalRef,
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-resident-process-dead-source"},
	}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-goal-first-resident-process-dead-001",
		ProcessRef:     "process-ref-goal-first-resident-process-dead-001",
		SessionRef:     "session-ref-goal-first-resident-process-dead-001",
		LaunchRef:      "launch-ref-goal-first-resident-process-dead-001",
		ReadinessRef:   "readiness-ref-goal-first-resident-process-dead-001",
		EvidenceRefs:   []string{"evidence-ref-goal-first-resident-process-dead-process"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{
			AppGoalStateStore: store,
			ProcessRegistry:   processRegistry,
		},
		Codex: CodexRuntimeConfigV0{SnapshotSource: evidenciaEstadoSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				"process-ref-goal-first-resident-process-dead-001": {
					SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
					ProcessRef:    "process-ref-goal-first-resident-process-dead-001",
					SessionRef:    "session-ref-goal-first-resident-process-dead-001",
					LaunchRef:     "launch-ref-goal-first-resident-process-dead-001",
					Status:        orquestaruntime.ProcessRuntimeStoppedV0,
				},
			},
		}},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       runRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason != "goal_first_resident_rework_prepared" ||
		len(result.RepairRunRefs) != 1 ||
		launcher.calls != 1 {
		t.Fatalf("proceso muerto sin rework: result=%+v calls=%d", result, launcher.calls)
	}
	hasReconciled := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "goal_first_resident_process_dead_reconciled" {
			hasReconciled = true
		}
	}
	if !hasReconciled {
		t.Fatalf("sin diagnostico de reconciliacion: %+v", result.Diagnostics)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusBlockedV0 ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.NeedsRework {
		t.Fatalf("state no reconciliado como terminal reconciliable: %+v", persisted)
	}
}

func TestRunSupervisorGoalFirstResidentProcesoVivoNoReconciliaV0(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstResidentReworkLauncherForTestV0{}
	runRef := "run-ref-goal-first-resident-process-live-001"
	goalRef := "goal-ref-goal-first-resident-process-live-001"
	source := orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RequestRef:   "request-ref-goal-first-resident-process-live-001",
			RunRef:       runRef,
			ProjectRef:   "project-ref-goal-first-resident-process-live",
			Objective:    "Proceso vivo confirmado no debe reconciliarse.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/goal-first-resident-process-live.md",
				Purpose: "evidencia de test",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusRunningV0,
			GoalRef:       goalRef,
		},
	}
	if err := store.SaveGoalWorkStateV0(ctx, source); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 source: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-goal-first-resident-process-live-001",
		ProcessRef:     "process-ref-goal-first-resident-process-live-001",
		SessionRef:     "session-ref-goal-first-resident-process-live-001",
		LaunchRef:      "launch-ref-goal-first-resident-process-live-001",
		ReadinessRef:   "readiness-ref-goal-first-resident-process-live-001",
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	executor := CodexStackRunSupervisorExecutorV0{Stack: &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:     store,
			GoalReworkLauncher: launcher,
		},
		Stores: StoresV0{
			AppGoalStateStore: store,
			ProcessRegistry:   processRegistry,
		},
		Codex: CodexRuntimeConfigV0{SnapshotSource: evidenciaEstadoSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				"process-ref-goal-first-resident-process-live-001": {
					SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
					ProcessRef:    "process-ref-goal-first-resident-process-live-001",
					SessionRef:    "session-ref-goal-first-resident-process-live-001",
					LaunchRef:     "launch-ref-goal-first-resident-process-live-001",
					Status:        orquestaruntime.ProcessRuntimeRunningV0,
				},
			},
		}},
	}}

	result, err := executor.Execute(ctx, orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:       runRef,
		ResidentMode: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.StopReason == "goal_first_resident_rework_prepared" || launcher.calls != 0 {
		t.Fatalf("proceso vivo reconciliado indebidamente: result=%+v calls=%d", result, launcher.calls)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("state vivo alterado: %+v", persisted)
	}
}
