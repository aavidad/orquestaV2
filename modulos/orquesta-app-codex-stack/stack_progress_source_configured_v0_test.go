package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestStatsProgressSourceV0ExponeFuenteConfiguradaSinObservaciones(t *testing.T) {
	source := statsProgressSourceV0(ConfigV0{
		Stores: StoresV0{
			ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
		},
	})
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-stats-progress-source-configured-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-stats-progress-source-configured-001"},
		Agents:        []string{"agent-stats-progress-source-configured-001"},
		StartedAgents: []string{"agent-stats-progress-source-configured-001"},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		context.Background(),
		run,
		nil,
		source,
		nil,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{
			CorrelationID: "corr-stats-progress-source-configured-001",
		},
	)

	if stats.Progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceLoadedV0 {
		t.Fatalf("progress source status=%s issues=%+v", stats.Progress.SourceStatus, stats.Progress.Issues)
	}
}
