package orquestaappcodexstack

import (
	"path/filepath"
	"testing"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
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
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime")
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
			AppChangeStore:  orquestaappchange.NewInMemoryAppChangeStoreV0(),
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0(),
			RunControl:      runMemory,
			RunQueue:        runMemory,
		},
		Codex: CodexRuntimeConfigV0{
			CommandPath:    filepath.Join(projectDir, "codex-bin"),
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
			Model:          "gpt-5.5",
			Sandbox:        "workspace-write",
			Runtime:        runtime,
			ProcessStopper: runtime,
			SnapshotSource: runtime,
			MaxBatchReady:  4,
			MaxConcurrency: 4,
			WaitInterval:   time.Millisecond,
			ProgressPolicy: codexStackProgressPolicyForTestV0(),
			ApprovalPolicy: "never",
			PromptHints:    []string{"Prueba fake: escribe ACK compacto."},
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

func TestProgressSourceV0UsaWaitIntervalComoVentanaMinima(t *testing.T) {
	waitInterval := 2 * time.Second
	source := progressSourceV0(ConfigV0{
		Stores: StoresV0{
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0(),
		},
		Codex: CodexRuntimeConfigV0{
			WaitInterval: waitInterval,
		},
	})
	if source.MinUnchangedSampleInterval != waitInterval {
		t.Fatalf("min interval=%s want %s", source.MinUnchangedSampleInterval, waitInterval)
	}
}
