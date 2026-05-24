package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	runMemory := orquestarunmemory.NewRunMemoryStoreV0()
	stack, err := BuildStackV0(ConfigV0{
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
			CommandPath:     filepath.Join(projectDir, "codex-bin"),
			ProjectWorkDir:  projectDir,
			RuntimeWorkDir:  runtimeDir,
			Model:           "gpt-5.5",
			ReasoningEffort: string(orquestacoreworkflow.OrchestrationCapacityXHighV0),
			Sandbox:         "workspace-write",
			Runtime:         runtime,
			ProcessStopper:  runtime,
			SnapshotSource:  runtime,
			MaxBatchReady:   4,
			MaxConcurrency:  4,
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
		RequiredTests: requiredTests,
		DomainWork:    domainWork,
		DomainDelivery: DomainWorkDeliveryBridgeConfigV0{
			Enabled: domainWork != nil,
		},
	})
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
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
		stack.MCPTransportBindings.DirectorStats == nil ||
		stack.MCPTransportBindings.RunQueuePriority == nil ||
		stack.MCPTransportBindings.RunSupervisor == nil ||
		stack.MCPTransportBindings.AutoprogrammingPrepareRun == nil ||
		stack.MCPTransportBindings.ServerShutdown == nil {
		t.Fatalf("bindings MCP incompletos: %+v", stack.MCPTransportBindings)
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
