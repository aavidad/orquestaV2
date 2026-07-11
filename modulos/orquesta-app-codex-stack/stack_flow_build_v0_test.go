package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type codexStackRuntimeForTestV0 interface {
	orquestaruntime.ExternalAgentProcessRuntimePortV0
	orquestacionnucleoapp.ProcessRuntimeStopPortV0
	orquestaruntimecodexdelivery.CodexProcessSnapshotSourcePortV0
}

func mustBuildCodexStackForTestV0(
	t *testing.T,
	runtime codexStackRuntimeForTestV0,
) StackV0 {
	t.Helper()
	return mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, nil)
}

func TestBuildStackV0PromotionEnabledRequiresGoalFirstSnapshotStoreV0(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{
		Enabled: true,
		Port:    &fakeAutoprogrammingPromotionPortV0{},
	}
	if _, err := BuildStackV0(config); err == nil || !strings.Contains(err.Error(), "goal_first_snapshot_store requerido") {
		t.Fatalf("err=%v", err)
	}
	config.AutoprogrammingPromotion.GoalFirstSnapshotStore = orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	if _, err := BuildStackV0(config); err != nil {
		t.Fatalf("BuildStackV0 con snapshot store: %v", err)
	}
}

func mustBuildCodexStackWithDomainWorkForTestV0(
	t *testing.T,
	runtime codexStackRuntimeForTestV0,
	domainWork orquestamcp.MCPDomainWorkExecutorPortV0,
) StackV0 {
	t.Helper()
	return mustBuildCodexStackWithDomainWorkAndRequiredTestsForTestV0(t, runtime, domainWork, nil)
}

func mustBuildCodexStackWithDomainWorkAndRequiredTestsForTestV0(
	t *testing.T,
	runtime codexStackRuntimeForTestV0,
	domainWork orquestamcp.MCPDomainWorkExecutorPortV0,
	requiredTests orquestacionnucleoapp.RequiredTestRunnerPortV0,
) StackV0 {
	t.Helper()
	config := codexStackBaseConfigForTestV0(t, runtime, domainWork, requiredTests)
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
}

func mustBuildCodexStackWithGoalBackendForTestV0(
	t *testing.T,
	runtime codexStackRuntimeForTestV0,
	launcher orquestagoal.GoalWorkLauncherPortV0,
	observer orquestagoal.GoalWorkObservationPortV0,
	goalStates orquestagoal.GoalWorkStateStorePortV0,
) StackV0 {
	t.Helper()
	config := codexStackBaseConfigForTestV0(t, runtime, nil, nil)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = observer
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = goalStates
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
}

func codexStackBaseConfigForTestV0(
	t *testing.T,
	runtime codexStackRuntimeForTestV0,
	domainWork orquestamcp.MCPDomainWorkExecutorPortV0,
	requiredTests orquestacionnucleoapp.RequiredTestRunnerPortV0,
) ConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	runMemory := orquestarunmemory.NewRunMemoryStoreV0()
	return ConfigV0{
		Enabled: true,
		Timeout: time.Second,
		DirectorLimits: orquestaweb.WebArrancarDirectorAppLimitsV0{
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     2,
		},
		Stores: StoresV0{
			RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(),
			EventSink:       orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			OutboxLedger:    orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
			TaskStore:       orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
			WaitStateStore:  orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(),
			AppChangeStore:  orquestaappchange.NewInMemoryAppChangeStoreV0(),
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			RunControl:      runMemory,
			RunQueue:        runMemory,
		},
		Codex: CodexRuntimeConfigV0{
			CommandPath:    filepath.Join(projectDir, "codex-bin"),
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
			Model:          "gpt-5.5",
			ModelRouting: CodexModelRoutingConfigV0{
				Policy: orquestacapacity.ModelRoutingPolicyV0{
					PolicyRef:        "policy-ref-test",
					Strict:           true,
					TrivialModelRef:  "luna",
					NormalModelRef:   "terra",
					CriticalModelRef: "sol",
					TrivialEffort:    "low",
					NormalEffort:     "medium",
					ComplexEffort:    "high",
					CriticalEffort:   "high",
				},
				ModelAlias: map[string]string{
					"luna":  "gpt-5.6-luna",
					"terra": "gpt-5.6-terra",
					"sol":   "gpt-5.6-sol",
				},
				TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{},
			},
			ReasoningEffort: string(orquestacoreworkflow.OrchestrationCapacityXHighV0),
			Sandbox:         "workspace-write",
			Runtime:         runtime,
			ProcessStopper:  runtime,
			SnapshotSource:  runtime,
			MaxBatchReady:   4,
			MaxConcurrency:  32,
			WaitInterval:    time.Millisecond,
			ProgressPolicy:  codexStackProgressPolicyForTestV0(),
			ApprovalPolicy:  "never",
			PromptHints:     []string{"Prueba fake: escribe ACK compacto."},
		},
		Capacity: CapacityConfigV0{
			Tier:            orquestacoreworkflow.OrchestrationCapacityXHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityXHighV0,
			OccurredAt:      "2026-05-10T10:00:00Z",
			RequestedBy:     "orquesta-app-stack-test",
			Summary:         "Capacidad fake para prueba de stack.",
			EvidenceRefs:    []string{"evidence-ref-app-stack-test"},
		},
		ReviewGate: ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
		RequiredTests:                 requiredTests,
		DomainWork:                    domainWork,
		AllowLegacyExternalWorkRun:    true,
		AllowLegacyAutoprogrammingRun: true,
		DomainDelivery: DomainWorkDeliveryBridgeConfigV0{
			Enabled: domainWork != nil,
		},
	}
}

func codexStackProgressPolicyForTestV0() orquestaruntime.AgentProgressHeartbeatPolicyV0 {
	return orquestaruntime.AgentProgressHeartbeatPolicyV0{
		StalledAfterNoProgressTicks: 10000,
		LoopAfterRepeatedActions:    10000,
	}
}

func TestBuildStackV0CableaRequiredTestRunnerV0(t *testing.T) {
	runner := fakeCodexStackRequiredTestRunnerV0{}
	stack := mustBuildCodexStackWithDomainWorkAndRequiredTestsForTestV0(t, newFakeCodexStackRuntimeV0(), nil, runner)
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("RequiredTestRunner no cableado")
	}
}

func TestBuildStackV0ExponeBindingsMCPNativosV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.MCPTransportBindings.ArrancarDirector == nil ||
		stack.MCPTransportBindings.NuevaAppWizard == nil ||
		stack.MCPTransportBindings.NuevaAppWizardBot == nil ||
		stack.MCPTransportBindings.DirectorStats == nil ||
		stack.MCPTransportBindings.RunQueuePriority == nil ||
		stack.MCPTransportBindings.RunSupervisor == nil ||
		stack.MCPTransportBindings.AutoprogrammingPrepareRun == nil ||
		stack.MCPTransportBindings.AutoprogrammingObserveActiveGoals == nil ||
		stack.MCPTransportBindings.ServerShutdown == nil {
		t.Fatalf("bindings MCP incompletos: %+v", stack.MCPTransportBindings)
	}
}

func TestBuildStackV0PropagaDiagnosticosAutoprogramacionABindingsV0(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AutoprogrammingStatusDiagnostics = []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "codex_goal_backend_degraded",
		Scope:        "app_goal",
		Message:      "codex goal backend degradado: codex_app_server_auth_missing",
		EvidenceRefs: []string{"evidence-ref-server-codex-goal-backend-degraded-app_goal"},
	}}
	config.AutoprogrammingGoalProgressPolicy = orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0{
		CheckpointOnlyHighConsumptionTokens: 42000,
	}

	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	if len(stack.MCPTransportBindings.AutoprogrammingStatusDiagnostics) != 1 ||
		stack.MCPTransportBindings.AutoprogrammingStatusDiagnostics[0].Code != "codex_goal_backend_degraded" ||
		stack.MCPTransportBindings.AutoprogrammingStatusDiagnostics[0].Scope != "app_goal" {
		t.Fatalf("diagnostics=%+v", stack.MCPTransportBindings.AutoprogrammingStatusDiagnostics)
	}
	runControl, ok := stack.MCPTransportBindings.RunControl.(orquestamcp.MCPRunControlToolExecutorV0)
	if !ok ||
		stack.MCPTransportBindings.AutoprogrammingGoalProgressPolicy.CheckpointOnlyHighConsumptionTokens != 42000 ||
		runControl.GoalProgressPolicy.CheckpointOnlyHighConsumptionTokens != 42000 {
		t.Fatalf("goal progress policy no propagada: %+v run_control=%+v",
			stack.MCPTransportBindings.AutoprogrammingGoalProgressPolicy,
			stack.MCPTransportBindings.RunControl,
		)
	}
}

func TestBuildStackV0CableaVerificadorWorktreeSiHaySnapshotStoreV0(t *testing.T) {
	store := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	config := ConfigV0{
		Stores: StoresV0{
			ReceiptStore: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
		},
		Codex: CodexRuntimeConfigV0{
			ProjectWorkDir: t.TempDir(),
			RuntimeWorkDir: filepath.Join(t.TempDir(), "runtime"),
		},
		ReviewGate: ReviewGateConfigV0{
			LineBudgetSnapshotStore: store,
		},
	}

	resolver := recordingSpecResolverV0(config)
	if resolver.WorktreeBaselineRecorder == nil {
		t.Fatalf("baseline recorder no cableado")
	}
	source, ok := deliverySourceV0(config).(orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0)
	if !ok || source.WorktreeVerifier == nil {
		t.Fatalf("delivery source sin verificador worktree: %T %+v", source, source)
	}
}

func TestAgentBatchDispatcherV0CableaGateDeProcesosVivosCodex(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	config := ConfigV0{
		Stores: StoresV0{
			ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			OutboxLedger:    orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
		},
		Codex: CodexRuntimeConfigV0{
			Runtime:         runtime,
			ProcessStopper:  runtime,
			SnapshotSource:  runtime,
			MaxBatchReady:   3,
			MaxConcurrency:  10,
			ProjectWorkDir:  t.TempDir(),
			RuntimeWorkDir:  filepath.Join(t.TempDir(), "runtime"),
			CommandPath:     filepath.Join(t.TempDir(), "codex-bin"),
			Sandbox:         "workspace-write",
			ApprovalPolicy:  "never",
			ReasoningEffort: string(orquestacoreworkflow.OrchestrationCapacityHighV0),
		},
	}
	dispatcher := agentBatchDispatcherV0(config)
	gate, ok := dispatcher.CapacityGate.(*orquestacionnucleoapp.LiveProcessCapacityGateV0)
	if !ok || gate.Limit != 10 {
		t.Fatalf("capacity gate=%T %+v", dispatcher.CapacityGate, dispatcher.CapacityGate)
	}
}

type fakeCodexStackRequiredTestRunnerV0 struct{}

func (fakeCodexStackRequiredTestRunnerV0) RunRequiredTestsV0(
	context.Context,
	orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, nil
}

func TestOperationalPlanStateStoreV0UsaStoreExplicitoOWriterLegible(t *testing.T) {
	explicitStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	if got := operationalPlanStateStoreV0(ConfigV0{Stores: StoresV0{
		OperationalPlanStateStore: explicitStore,
	}}); got != explicitStore {
		t.Fatalf("store explicito=%T want explicit", got)
	}
	writerStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	if got := operationalPlanStateStoreV0(ConfigV0{Stores: StoresV0{
		OperationalPlanStateWriter: writerStore,
	}}); got != writerStore {
		t.Fatalf("writer legible=%T want writerStore", got)
	}
}

func TestWaitStateStoreV0UsaStoreExplicitoOTaskStoreLegible(t *testing.T) {
	explicitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	if got := waitStateStoreV0(ConfigV0{Stores: StoresV0{
		WaitStateStore: explicitStore,
	}}); got != explicitStore {
		t.Fatalf("store explicito=%T want explicit", got)
	}
	taskStore := codexStackWorkflowTaskAndWaitStateStoreForTestV0{
		InMemoryWorkflowTaskStoreV0:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
		InMemoryWorkflowTaskWaitStateStoreV0: orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(),
	}
	if got := waitStateStoreV0(ConfigV0{Stores: StoresV0{
		TaskStore: taskStore,
	}}); got != taskStore {
		t.Fatalf("task store legible=%T want taskStore", got)
	}
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.Ports.WaitStateStore == nil {
		t.Fatalf("WaitStateStore no cableado")
	}
}

type codexStackWorkflowTaskAndWaitStateStoreForTestV0 struct {
	*orquestacionnucleoapp.InMemoryWorkflowTaskStoreV0
	*orquestacionnucleoapp.InMemoryWorkflowTaskWaitStateStoreV0
}

func TestProgressSourceV0UsaWaitIntervalComoVentanaMinima(t *testing.T) {
	waitInterval := 2 * time.Second
	source := progressSourceV0(ConfigV0{
		Stores: StoresV0{
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
		},
		Codex: CodexRuntimeConfigV0{
			WaitInterval: waitInterval,
		},
	})
	if source.MinUnchangedSampleInterval != waitInterval {
		t.Fatalf("min interval=%s want %s", source.MinUnchangedSampleInterval, waitInterval)
	}
}

func TestStatsProgressSourceV0EmiteProgresoNormal(t *testing.T) {
	source := statsProgressSourceV0(ConfigV0{
		Stores: StoresV0{
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
		},
	})
	if !source.EmitProgressing {
		t.Fatalf("stats progress source debe emitir progreso normal")
	}
	if progressSourceV0(ConfigV0{}).EmitProgressing {
		t.Fatalf("progress source de decision no debe emitir progreso normal")
	}
}
